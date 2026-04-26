package builder

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
	"github.com/user/ogame-bot/internal/ogamed"
)

// BuilderStateReader provides read access to cached game state needed by the builder.
// state.Manager implicitly satisfies this interface.
type BuilderStateReader interface {
	GetPlanets(ctx context.Context) ([]model.Planet, error)
	GetResources(ctx context.Context, planetID int) (model.Resources, error)
	GetResearch(ctx context.Context) (model.Research, error)
	GetBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error)
	GetFacilities(ctx context.Context, planetID int) (model.Facilities, error)
	GetServerSpeed(ctx context.Context) (int, error)
}

// buildingIDs is the set of resource building IDs the builder evaluates for ROI.
var buildingIDs = []int{
	constants.BuildingMetalMine,
	constants.BuildingCrystalMine,
	constants.BuildingDeuteriumSynthesizer,
	constants.BuildingSolarPlant,
	constants.BuildingFusionReactor,
}

// buildingNameToMaxLevelKey maps building IDs to the config key used in MaxLevels and PlanetOverrides.
var buildingNameToMaxLevelKey = map[int]string{
	constants.BuildingMetalMine:            "MetalMine",
	constants.BuildingCrystalMine:          "CrystalMine",
	constants.BuildingDeuteriumSynthesizer: "DeuteriumSynthesizer",
	constants.BuildingSolarPlant:           "SolarPlant",
	constants.BuildingFusionReactor:        "FusionReactor",
}

// Builder orchestrates auto-building: polls all planets, computes ROI,
// picks the best candidate, and executes the build.
type Builder struct {
	client        ogamed.ClientInterface
	stateMgr      BuilderStateReader
	db            *sql.DB
	cfg           config.AutoBuildConfig
	log           *slog.Logger
	antiDetectPct float64
	broadcaster   Broadcaster
}

type Broadcaster interface {
	Broadcast(msgType string, data interface{})
}

// NewBuilder creates a new Builder with all required dependencies.
func NewBuilder(client ogamed.ClientInterface, stateMgr BuilderStateReader, db *sql.DB, cfg config.AutoBuildConfig, log *slog.Logger) *Builder {
	return &Builder{
		client:        client,
		stateMgr:      stateMgr,
		db:            db,
		cfg:           cfg,
		log:           log.With("component", "builder"),
		antiDetectPct: 0.07,
	}
}

func (b *Builder) SetBroadcaster(br Broadcaster) {
	b.broadcaster = br
}

func (b *Builder) broadcast(msgType string, data interface{}) {
	if b.broadcaster != nil {
		b.broadcaster.Broadcast(msgType, data)
	}
}

// Run starts the builder poll loop. Blocks until context is cancelled.
func (b *Builder) Run(ctx context.Context) {
	interval := time.Duration(b.cfg.PollIntervalMs) * time.Millisecond
	b.log.Info("Starting builder", "interval", interval)

	for {
		// Randomized jitter: [0, interval/2)
		jitter := time.Duration(rand.Intn(int(interval.Milliseconds()/2)+1)) * time.Millisecond
		waitTime := interval + jitter

		select {
		case <-ctx.Done():
			b.log.Info("Builder stopped")
			return
		case <-time.After(waitTime):
			b.poll(ctx)
		}
	}
}

// poll evaluates ROI for all buildings on all planets and builds the best candidate.
func (b *Builder) poll(ctx context.Context) {
	if !b.cfg.Enabled {
		return
	}

	// 1. Get global state
	planets, err := b.stateMgr.GetPlanets(ctx)
	if err != nil {
		b.log.Error("Failed to get planets", "error", err)
		return
	}
	if len(planets) == 0 {
		return
	}

	speed, err := b.stateMgr.GetServerSpeed(ctx)
	if err != nil {
		b.log.Error("Failed to get server speed", "error", err)
		return
	}

	research, err := b.stateMgr.GetResearch(ctx)
	if err != nil {
		b.log.Error("Failed to get research", "error", err)
		return
	}

	// 2. Evaluate each planet
	var candidates []ROIResult

	for _, planet := range planets {
		// Skip full planets
		if planet.FieldsUsed >= planet.FieldsTotal {
			b.log.Warn("Planet is full, skipping", "planet", planet.Name, "fields", fmt.Sprintf("%d/%d", planet.FieldsUsed, planet.FieldsTotal))
			continue
		}

		// Check construction slot
		constructions, err := b.client.GetConstructions(ctx, planet.ID)
		if err != nil {
			b.log.Error("Failed to get constructions, skipping planet", "planet", planet.Name, "error", err)
			continue
		}
		if constructions.Building.ID != 0 {
			b.log.Debug("Planet has active construction, skipping", "planet", planet.Name, "buildingID", constructions.Building.ID)
			continue
		}

		// Get planet state
		resources, err := b.stateMgr.GetResources(ctx, planet.ID)
		if err != nil {
			b.log.Error("Failed to get resources", "planet", planet.Name, "error", err)
			continue
		}

		buildings, err := b.stateMgr.GetBuildings(ctx, planet.ID)
		if err != nil {
			b.log.Error("Failed to get buildings", "planet", planet.Name, "error", err)
			continue
		}

		facilities, err := b.stateMgr.GetFacilities(ctx, planet.ID)
		if err != nil {
			b.log.Error("Failed to get facilities", "planet", planet.Name, "error", err)
			continue
		}

		// 3. Evaluate ROI for each building
		for _, buildingID := range buildingIDs {
			currentLevel := buildingLevel(buildings, buildingID)
			buildingName := buildingNameToMaxLevelKey[buildingID]
			maxLevel := b.resolveMaxLevel(buildingName, planet.Name)

			result, viable := CalculateROI(
				buildingID, currentLevel, planet, buildings, facilities,
				research, resources, speed, maxLevel,
			)
			if viable {
				candidates = append(candidates, result)
			}
		}
	}

	if len(candidates) == 0 {
		b.log.Debug("No viable build candidates found")
		return
	}

	// 4. Sort by ROI score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ROIScore > candidates[j].ROIScore
	})

	// 5. Pick best candidate (anti-detection: configurable chance to pick 2nd best)
	pickIdx := 0
	if len(candidates) > 1 && b.antiDetectPct > 0 && rand.Float64() < b.antiDetectPct {
		pickIdx = 1
		b.log.Info("Anti-detection: picking 2nd-best ROI candidate")
	}
	best := candidates[pickIdx]

	// 6. Execute build
	b.log.Info("Building upgrade",
		"planet", best.PlanetID,
		"building", best.BuildingName,
		"fromLevel", best.CurrentLevel,
		"toLevel", best.TargetLevel,
		"roiScore", fmt.Sprintf("%.6f", best.ROIScore),
		"cost", fmt.Sprintf("metal=%d crystal=%d deut=%d", best.CostMetal, best.CostCrystal, best.CostDeuterium),
		"energyDelta", best.EnergyDelta,
	)

	if err := b.client.BuildBuilding(ctx, best.PlanetID, best.BuildingID); err != nil {
		b.log.Error("BuildBuilding failed",
			"planet", best.PlanetID,
			"building", best.BuildingName,
			"error", err)
		return
	}

	// 7. Record build event
	if err := b.recordBuildEvent(ctx, best); err != nil {
		b.log.Error("Failed to record build event", "error", err)
	}

	b.broadcast("build", map[string]interface{}{
		"planetId":     best.PlanetID,
		"buildingId":   best.BuildingID,
		"buildingName": best.BuildingName,
		"fromLevel":    best.CurrentLevel,
		"toLevel":      best.TargetLevel,
		"roiScore":     best.ROIScore,
	})
}

// resolveMaxLevel returns the max level cap for a building on a specific planet.
// Planet overrides take precedence over global defaults.
// Returns 0 if no config found (effectively disabled).
func (b *Builder) resolveMaxLevel(buildingName, planetName string) int {
	// Check per-planet override first
	if b.cfg.PlanetOverrides != nil {
		if planetOverrides, ok := b.cfg.PlanetOverrides[planetName]; ok {
			if maxLvl, ok := planetOverrides[buildingName]; ok {
				return maxLvl
			}
		}
	}
	// Fall back to global default
	if b.cfg.MaxLevels != nil {
		if maxLvl, ok := b.cfg.MaxLevels[buildingName]; ok {
			return maxLvl
		}
	}
	return 0
}

// buildingLevel returns the current level of a building from the ResourceBuildings struct.
func buildingLevel(buildings model.ResourceBuildings, buildingID int) int {
	switch buildingID {
	case constants.BuildingMetalMine:
		return buildings.MetalMine
	case constants.BuildingCrystalMine:
		return buildings.CrystalMine
	case constants.BuildingDeuteriumSynthesizer:
		return buildings.DeuteriumSynthesizer
	case constants.BuildingSolarPlant:
		return buildings.SolarPlant
	case constants.BuildingFusionReactor:
		return buildings.FusionReactor
	default:
		return 0
	}
}

// recordBuildEvent inserts a build event into the database for audit/repudiation purposes.
func (b *Builder) recordBuildEvent(ctx context.Context, result ROIResult) error {
	_, err := b.db.ExecContext(ctx,
		`INSERT INTO build_events (planet_id, building_id, building_name, from_level, to_level, cost_metal, cost_crystal, cost_deut, roi_score)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.PlanetID, result.BuildingID, result.BuildingName,
		result.CurrentLevel, result.TargetLevel,
		result.CostMetal, result.CostCrystal, result.CostDeuterium,
		result.ROIScore)
	if err != nil {
		return fmt.Errorf("recording build event for planet %d building %s: %w", result.PlanetID, result.BuildingName, err)
	}
	return nil
}
