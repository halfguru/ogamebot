package farmer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/defender"
	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamed"
)

// Farmer orchestrates auto-farming: scans galaxies, spies inactives, attacks profitable targets.
type Farmer struct {
	client      ogamed.ClientInterface
	stateMgr    FarmerStateReader
	db          *sql.DB
	cfg         config.AutoFarmConfig
	log         *slog.Logger
	broadcaster Broadcaster
}

type Broadcaster interface {
	Broadcast(msgType string, data interface{})
}

// NewFarmer creates a new Farmer with all required dependencies.
func NewFarmer(client ogamed.ClientInterface, stateMgr FarmerStateReader, db *sql.DB, cfg config.AutoFarmConfig, log *slog.Logger) *Farmer {
	return &Farmer{
		client:   client,
		stateMgr: stateMgr,
		db:       db,
		cfg:      cfg,
		log:      log.With("component", "farmer"),
	}
}

func (f *Farmer) SetBroadcaster(b Broadcaster) {
	f.broadcaster = b
}

func (f *Farmer) broadcast(msgType string, data interface{}) {
	if f.broadcaster != nil {
		f.broadcaster.Broadcast(msgType, data)
	}
}

// Run starts the farmer poll loop. Blocks until context is cancelled.
func (f *Farmer) Run(ctx context.Context) {
	interval := time.Duration(f.cfg.PollIntervalMs) * time.Millisecond
	f.log.Info("Starting farmer", "interval", interval)

	for {
		jitter := time.Duration(rand.Intn(int(interval.Milliseconds()/2)+1)) * time.Millisecond
		waitTime := interval + jitter

		select {
		case <-ctx.Done():
			f.log.Info("Farmer stopped")
			return
		case <-time.After(waitTime):
			f.poll(ctx)
		}
	}
}

// poll executes one farm cycle: scan → spy → evaluate → attack.
func (f *Farmer) poll(ctx context.Context) {
	if !f.cfg.Enabled {
		return
	}

	// 1. Check fleet slots
	slots, err := f.client.GetSlots(ctx)
	if err != nil {
		f.log.Error("Failed to check slots", "error", err)
		return
	}
	if slots.InUse >= slots.Total {
		f.log.Debug("No fleet slots available, skipping farm cycle")
		return
	}

	// 2. Scan galaxy ranges for inactives
	inactives, err := f.scanGalaxies(ctx)
	if err != nil {
		f.log.Error("Failed to scan galaxies", "error", err)
		return
	}
	if len(inactives) == 0 {
		f.log.Debug("No inactive targets found")
		return
	}
	f.log.Info("Found inactive targets", "count", len(inactives))

	// 3. Send espionage probes to un-scanned targets
	f.spyTargets(ctx, inactives)

	// 4. Read espionage reports and evaluate targets
	targets, err := f.evaluateReports(ctx)
	if err != nil {
		f.log.Error("Failed to evaluate reports", "error", err)
		return
	}
	if len(targets) == 0 {
		f.log.Debug("No profitable targets found")
		return
	}

	// 5. Attack top targets
	f.attackTargets(ctx, targets, slots)
}

// --- Galaxy Scanning ---

// scanGalaxies scans configured galaxy ranges and returns inactive player positions.
func (f *Farmer) scanGalaxies(ctx context.Context) ([]model.PlanetPosition, error) {
	var inactives []model.PlanetPosition

	for _, rng := range f.cfg.GalaxyRanges {
		for system := rng.SystemStart; system <= rng.SystemEnd; system++ {
			sysInfo, err := f.client.GetGalaxyInfos(ctx, rng.Galaxy, system)
			if err != nil {
				f.log.Warn("Failed to scan system", "galaxy", rng.Galaxy, "system", system, "error", err)
				continue
			}

			for _, pos := range sysInfo.Planets {
				if isInactiveTarget(pos) {
					inactives = append(inactives, pos)
				}
			}
		}
	}

	return inactives, nil
}

// isInactiveTarget returns true if a planet position qualifies as a farm target.
func isInactiveTarget(pos model.PlanetPosition) bool {
	return pos.Name != "" && pos.Inactive && !pos.Vacation && !pos.Banned
}

// --- Espionage ---

// spyTargets sends espionage probes to inactive targets.
func (f *Farmer) spyTargets(ctx context.Context, inactives []model.PlanetPosition) {
	ownPlanets, err := f.stateMgr.GetPlanets(ctx)
	if err != nil {
		f.log.Error("Failed to get own planets for spy origin", "error", err)
		return
	}
	if len(ownPlanets) == 0 {
		return
	}

	probed := 0
	for _, target := range inactives {
		if probed >= 10 { // Limit probes per cycle to avoid spamming
			break
		}

		// Pick closest own planet as spy origin
		origin := pickClosestPlanet(ownPlanets, target.Coordinate)

		// Send probes
		probeCount := f.cfg.MaxProbesPerTarget
		req := model.SendFleetRequest{
			PlanetID: origin.ID,
			Ships:    []model.ShipCount{{ID: constants.ShipEspionageProbe, Count: probeCount}},
			Speed:    10,
			Galaxy:   target.Coordinate.Galaxy,
			System:   target.Coordinate.System,
			Position: target.Coordinate.Position,
			Type:     1, // planet
			Mission:  constants.MissionEspionage,
		}

		_, err := f.client.SendFleet(ctx, req)
		if err != nil {
			f.log.Warn("Failed to send espionage probe",
				"target", fmt.Sprintf("%d:%d:%d", target.Coordinate.Galaxy, target.Coordinate.System, target.Coordinate.Position),
				"error", err)
			continue
		}

		f.log.Info("Sent espionage probe",
			"target", fmt.Sprintf("%d:%d:%d", target.Coordinate.Galaxy, target.Coordinate.System, target.Coordinate.Position),
			"player", target.PlayerName,
			"probes", probeCount)

		probed++

		// Record target in DB
		f.upsertTarget(ctx, target)
	}
}

// --- Report Evaluation ---

// evaluateReports reads espionage reports and returns viable attack targets sorted by net profit.
func (f *Farmer) evaluateReports(ctx context.Context) ([]FarmTarget, error) {
	reports, err := f.client.GetEspionageReportMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching espionage report messages: %w", err)
	}
	if len(reports) == 0 {
		return nil, nil
	}

	ownPlanets, err := f.stateMgr.GetPlanets(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting own planets: %w", err)
	}

	research, err := f.stateMgr.GetResearch(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting research: %w", err)
	}

	var targets []FarmTarget

	for _, summary := range reports {
		report, err := f.client.GetEspionageReport(ctx, summary.ID)
		if err != nil {
			f.log.Warn("Failed to get espionage report", "msgID", summary.ID, "error", err)
			continue
		}

		target, viable := f.evaluateReport(report, ownPlanets, research)
		if viable {
			targets = append(targets, target)
		}
	}

	// Sort by net profit descending
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].NetProfit > targets[j].NetProfit
	})

	// Clean up processed reports
	if err := f.client.DeleteAllEspionageReports(ctx); err != nil {
		f.log.Warn("Failed to delete espionage reports", "error", err)
	}

	return targets, nil
}

// evaluateReport evaluates a single espionage report and returns a FarmTarget if viable.
func (f *Farmer) evaluateReport(report model.EspionageReport, ownPlanets []model.Planet, research model.Research) (FarmTarget, bool) {
	// Check defense
	if f.cfg.SkipDefended && hasDefense(report) {
		return FarmTarget{}, false
	}

	// Calculate loot (50% plunder ratio for standard attack)
	plunderRatio := 0.5
	metalLoot := int64(float64(report.Metal) * plunderRatio)
	crystalLoot := int64(float64(report.Crystal) * plunderRatio)
	deuteriumLoot := int64(float64(report.Deuterium) * plunderRatio)

	// Metal-equivalent value
	totalValue := calcLootValue(metalLoot, crystalLoot, deuteriumLoot)

	// Find closest own planet and calculate distance
	origin := pickClosestPlanet(ownPlanets, report.Coordinate)
	distance := defender.CalcDistance(origin.Coordinate, report.Coordinate)

	// Estimate fuel cost using small cargos
	cargoCount := cargoNeeded(totalValue, false)
	fuel := estimateFuelCost(distance, 10, cargoCount, research)

	netProfit := totalValue - fuel

	if netProfit < f.cfg.MinProfitThreshold {
		return FarmTarget{}, false
	}

	return FarmTarget{
		Coordinate:    report.Coordinate,
		MetalLoot:     metalLoot,
		CrystalLoot:   crystalLoot,
		DeuteriumLoot: deuteriumLoot,
		TotalValue:    totalValue,
		NetProfit:     netProfit,
		HasDefense:    hasDefense(report),
		Distance:      distance,
	}, true
}

// --- Attack Execution ---

// attackTargets dispatches attacks to the top N farm targets.
func (f *Farmer) attackTargets(ctx context.Context, targets []FarmTarget, slots model.Slots) {
	ownPlanets, err := f.stateMgr.GetPlanets(ctx)
	if err != nil {
		f.log.Error("Failed to get own planets for attack", "error", err)
		return
	}

	availableSlots := slots.Total - slots.InUse
	maxAttacks := min(f.cfg.MaxAttacksPerCycle, availableSlots-2) // Reserve 2 slots for defender

	if maxAttacks <= 0 {
		f.log.Debug("Not enough fleet slots for farming", "available", availableSlots)
		return
	}

	attacked := 0
	for _, target := range targets {
		if attacked >= maxAttacks {
			break
		}

		origin := pickClosestPlanet(ownPlanets, target.Coordinate)

		// Determine cargo ships needed
		totalLoot := target.MetalLoot + target.CrystalLoot + target.DeuteriumLoot
		cargoCount := cargoNeeded(totalLoot, false) // false = small cargo

		// Verify we have enough ships on origin planet
		ships, err := f.client.GetShips(ctx, origin.ID)
		if err != nil {
			f.log.Warn("Failed to get ships", "planet", origin.Name, "error", err)
			continue
		}

		if int64(ships.SmallCargo) < cargoCount {
			f.log.Debug("Not enough small cargo on planet",
				"planet", origin.Name, "needed", cargoCount, "available", ships.SmallCargo)
			continue
		}

		req := model.SendFleetRequest{
			PlanetID:  origin.ID,
			Ships:     []model.ShipCount{{ID: constants.ShipSmallCargo, Count: int(cargoCount)}},
			Speed:     10,
			Galaxy:    target.Coordinate.Galaxy,
			System:    target.Coordinate.System,
			Position:  target.Coordinate.Position,
			Type:      1, // planet
			Mission:   constants.MissionAttack,
			Metal:     0,
			Crystal:   0,
			Deuterium: 0,
		}

		fleetID, err := f.client.SendFleet(ctx, req)
		if err != nil {
			f.log.Warn("Failed to send attack fleet",
				"target", fmt.Sprintf("%d:%d:%d", target.Coordinate.Galaxy, target.Coordinate.System, target.Coordinate.Position),
				"error", err)
			continue
		}

		f.log.Info("Attack dispatched",
			"fleetID", fleetID,
			"target", fmt.Sprintf("%d:%d:%d", target.Coordinate.Galaxy, target.Coordinate.System, target.Coordinate.Position),
			"origin", origin.Name,
			"ships", cargoCount,
			"netProfit", target.NetProfit)

		// Record attack in DB
		f.recordAttack(ctx, fleetID, int64(origin.ID), target, int(cargoCount))

		f.broadcast("farm_attack", map[string]interface{}{
			"fleetId":  fleetID,
			"target":   fmt.Sprintf("%d:%d:%d", target.Coordinate.Galaxy, target.Coordinate.System, target.Coordinate.Position),
			"origin":   origin.Name,
			"ships":    cargoCount,
			"profit":   target.NetProfit,
		})

		attacked++
	}
}

// --- Pure Helper Functions ---

// calcLootValue converts resource amounts to metal-equivalent value.
// Trade ratios: metal=1, crystal=1.5, deuterium=2.0
func calcLootValue(metal, crystal, deuterium int64) int64 {
	return metal + int64(float64(crystal)*1.5) + int64(float64(deuterium)*2.0)
}

// hasDefense returns true if the espionage report shows any defensive structures.
func hasDefense(report model.EspionageReport) bool {
	return report.RocketLauncher > 0 || report.LightLaser > 0 ||
		report.HeavyLaser > 0 || report.GaussCannon > 0 ||
		report.IonCannon > 0 || report.PlasmaTurret > 0 ||
		report.SmallShieldDome > 0 || report.LargeShieldDome > 0
}

// pickClosestPlanet returns the own planet closest to the target coordinate.
func pickClosestPlanet(ownPlanets []model.Planet, target model.Coordinate) model.Planet {
	if len(ownPlanets) == 0 {
		return model.Planet{}
	}
	closest := ownPlanets[0]
	minDist := defender.CalcDistance(closest.Coordinate, target)
	for _, p := range ownPlanets[1:] {
		d := defender.CalcDistance(p.Coordinate, target)
		if d < minDist {
			closest = p
			minDist = d
		}
	}
	return closest
}

// cargoNeeded returns the number of cargo ships needed to carry the given total resources.
func cargoNeeded(totalLoot int64, largeCargo bool) int64 {
	capacity := int64(5000)
	if largeCargo {
		capacity = 25000
	}
	if totalLoot <= 0 {
		return 0
	}
	return int64(math.Ceil(float64(totalLoot) / float64(capacity)))
}

// estimateFuelCost estimates deuterium cost for small cargo fleet at given distance.
// Uses simplified OGame fuel formula for small cargos only.
func estimateFuelCost(distance int, speed int, cargoCount int64, research model.Research) int64 {
	if distance == 0 {
		return 0
	}
	// SmallCargo base fuel = 10
	baseFuel := 10
	speedPct := float64(speed)/10.0 + 1.0
	consumption := float64(baseFuel) * float64(cargoCount) * (float64(distance) / 35000.0) * math.Pow(speedPct/2.0, 2)
	return int64(math.Round(consumption))
}

// --- Database Helpers ---

// upsertTarget inserts or updates a farm target in the database.
func (f *Farmer) upsertTarget(ctx context.Context, pos model.PlanetPosition) {
	_, err := f.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO farm_targets (galaxy, system, position, player_id, player_name, is_inactive, is_long_inactive, last_scanned_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		pos.Coordinate.Galaxy, pos.Coordinate.System, pos.Coordinate.Position,
		pos.PlayerID, pos.PlayerName, pos.Inactive, pos.LongInactive)
	if err != nil {
		f.log.Warn("Failed to upsert farm target", "error", err)
	}
}

// recordAttack records a farm attack event in the database.
func (f *Farmer) recordAttack(ctx context.Context, fleetID, planetID int64, target FarmTarget, shipsSent int) {
	_, err := f.db.ExecContext(ctx,
		`INSERT INTO farm_attacks (fleet_id, planet_id, target_galaxy, target_system, target_position, ships_sent, metal_looted, crystal_looted, deuterium_looted)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fleetID, planetID, target.Coordinate.Galaxy, target.Coordinate.System, target.Coordinate.Position,
		shipsSent, target.MetalLoot, target.CrystalLoot, target.DeuteriumLoot)
	if err != nil {
		f.log.Warn("Failed to record farm attack", "error", err)
	}
}
