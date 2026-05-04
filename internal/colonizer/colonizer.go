package colonizer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"time"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamex"
)

type ColonizerStateReader interface {
	GetPlanets(ctx context.Context) ([]model.Planet, error)
	GetResearch(ctx context.Context) (model.Research, error)
}

type Broadcaster interface {
	Broadcast(msgType string, data interface{})
}

type Colonizer struct {
	client      ogamex.ClientInterface
	stateMgr    ColonizerStateReader
	db          *sql.DB
	cfg         config.ColonizerConfig
	log         *slog.Logger
	broadcaster Broadcaster
}

func NewColonizer(client ogamex.ClientInterface, stateMgr ColonizerStateReader, db *sql.DB, cfg config.ColonizerConfig, log *slog.Logger) *Colonizer {
	return &Colonizer{
		client:   client,
		stateMgr: stateMgr,
		db:       db,
		cfg:      cfg,
		log:      log.With("component", "colonizer"),
	}
}

func (c *Colonizer) SetBroadcaster(b Broadcaster) {
	c.broadcaster = b
}

func (c *Colonizer) broadcast(msgType string, data interface{}) {
	if c.broadcaster != nil {
		c.broadcaster.Broadcast(msgType, data)
	}
}

func (c *Colonizer) Run(ctx context.Context) {
	interval := time.Duration(c.cfg.PollIntervalMs) * time.Millisecond
	c.log.Info("Starting colonizer", "interval", interval)

	for {
		jitter := time.Duration(rand.Intn(int(interval.Milliseconds()/2)+1)) * time.Millisecond
		waitTime := interval + jitter

		select {
		case <-ctx.Done():
			c.log.Info("Colonizer stopped")
			return
		case <-time.After(waitTime):
			c.poll(ctx)
		}
	}
}

func (c *Colonizer) poll(ctx context.Context) {
	if !c.cfg.Enabled {
		return
	}

	planets, err := c.stateMgr.GetPlanets(ctx)
	if err != nil {
		c.log.Error("Failed to get planets", "error", err)
		return
	}
	if len(planets) == 0 {
		return
	}

	if len(planets) >= c.cfg.TargetPlanetCount {
		c.log.Debug("Target planet count reached", "current", len(planets), "target", c.cfg.TargetPlanetCount)
		return
	}

	research, err := c.stateMgr.GetResearch(ctx)
	if err != nil {
		c.log.Error("Failed to get research", "error", err)
		return
	}

	maxColonies := 1 + research.Astrophysics
	if len(planets) >= maxColonies {
		c.log.Debug("Astrophysics limits reached", "maxColonies", maxColonies, "current", len(planets))
		return
	}

	slots, err := c.client.GetSlots(ctx)
	if err != nil {
		c.log.Error("Failed to get slots", "error", err)
		return
	}
	if slots.InUse >= slots.Total {
		c.log.Debug("No fleet slots available")
		return
	}

	var originPlanet *model.Planet
	var originPlanetID int
	for i := range planets {
		ships, err := c.client.GetShips(ctx, planets[i].ID)
		if err != nil {
			c.log.Warn("Failed to get ships", "planet", planets[i].Name, "error", err)
			continue
		}
		if ships.ColonyShip > 0 {
			originPlanet = &planets[i]
			originPlanetID = planets[i].ID
			break
		}
	}
	if originPlanet == nil {
		c.log.Info("No colony ship available on any planet, skipping")
		return
	}

	homeGalaxy := originPlanet.Coordinate.Galaxy
	homeSystem := originPlanet.Coordinate.System

	scoreMap := make(map[int]int)
	for i, pos := range c.cfg.PreferPositions {
		scoreMap[pos] = len(c.cfg.PreferPositions) - i
	}

	type scoredPosition struct {
		galaxy   int
		system   int
		position int
		score    int
		distance int
	}

	var candidates []scoredPosition

	attempts := 0
	for sysOffset := 0; sysOffset <= c.cfg.ScanRadius && attempts < c.cfg.MaxAttempts*10; sysOffset++ {
		for _, dir := range []int{0, 1, -1} {
			if sysOffset == 0 && dir != 0 {
				continue
			}
			if sysOffset != 0 && dir == 0 {
				continue
			}
			sys := homeSystem + sysOffset*dir
			if sys < 1 {
				continue
			}

			sysInfo, err := c.client.GetGalaxyInfos(ctx, homeGalaxy, sys)
			if err != nil {
				c.log.Warn("Failed to scan system", "galaxy", homeGalaxy, "system", sys, "error", err)
				continue
			}

			occupied := make(map[int]bool)
			for _, p := range sysInfo.Planets {
				occupied[p.Position] = true
			}

			for pos := 1; pos <= 15; pos++ {
				if occupied[pos] {
					continue
				}
				score := scoreMap[pos]
				dist := sys - homeSystem
				if dist < 0 {
					dist = -dist
				}
				candidates = append(candidates, scoredPosition{
					galaxy:   homeGalaxy,
					system:   sys,
					position: pos,
					score:    score,
					distance: dist,
				})
			}

			attempts++
			if attempts >= c.cfg.MaxAttempts*10 {
				break
			}
		}
	}

	if len(candidates) == 0 {
		c.log.Debug("No empty positions found in scan range")
		return
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].distance < candidates[j].distance
	})

	sent := 0
	for _, cand := range candidates {
		if sent >= c.cfg.MaxAttempts {
			break
		}

		req := model.SendFleetRequest{
			PlanetID: originPlanetID,
			Ships:    []model.ShipCount{{ID: constants.ShipColonyShip, Count: 1}},
			Speed:    10,
			Galaxy:   cand.galaxy,
			System:   cand.system,
			Position: cand.position,
			Type:     1,
			Mission:  constants.MissionColonize,
		}

		fleetID, err := c.client.SendFleet(ctx, req)
		if err != nil {
			c.log.Warn("Failed to send colony ship",
				"target", fmt.Sprintf("%d:%d:%d", cand.galaxy, cand.system, cand.position),
				"error", err)
			continue
		}

		c.log.Info("Colony ship dispatched",
			"fleetID", fleetID,
			"origin", originPlanet.Name,
			"target", fmt.Sprintf("%d:%d:%d", cand.galaxy, cand.system, cand.position),
			"score", cand.score,
		)

		c.recordColonizeEvent(ctx, originPlanetID, cand.galaxy, cand.system, cand.position)

		c.broadcast("colonize", map[string]interface{}{
			"fleetId":       fleetID,
			"originPlanet":  originPlanet.Name,
			"target":        fmt.Sprintf("%d:%d:%d", cand.galaxy, cand.system, cand.position),
			"score":         cand.score,
		})

		sent++
	}
}

func (c *Colonizer) recordColonizeEvent(ctx context.Context, originPlanetID, galaxy, system, position int) {
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO colonize_events (origin_planet_id, target_galaxy, target_system, target_position, status)
		 VALUES (?, ?, ?, ?, 'sent')`,
		originPlanetID, galaxy, system, position)
	if err != nil {
		c.log.Warn("Failed to record colonize event", "error", err)
	}
}
