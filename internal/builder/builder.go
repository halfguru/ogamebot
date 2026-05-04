package builder

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamex"
)

type BuilderStateReader interface {
	GetPlanets(ctx context.Context) ([]model.Planet, error)
	GetResources(ctx context.Context, planetID int) (model.Resources, error)
	GetResearch(ctx context.Context) (model.Research, error)
	GetBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error)
	GetFacilities(ctx context.Context, planetID int) (model.Facilities, error)
	GetServerSpeed(ctx context.Context) (int, error)
	RefreshNow(ctx context.Context) error
}

var buildingNameToMaxLevelKey = map[int]string{
	constants.BuildingMetalMine:            "MetalMine",
	constants.BuildingCrystalMine:          "CrystalMine",
	constants.BuildingDeuteriumSynthesizer: "DeuteriumSynthesizer",
	constants.BuildingSolarPlant:           "SolarPlant",
	constants.BuildingFusionReactor:        "FusionReactor",
	constants.BuildingMetalStorage:         "MetalStorage",
	constants.BuildingCrystalStorage:       "CrystalStorage",
	constants.BuildingDeuteriumStorage:     "DeuteriumStorage",
	constants.BuildingRoboticsFactory:      "RoboticsFactory",
	constants.BuildingShipyard:             "Shipyard",
	constants.BuildingResearchLab:          "ResearchLab",
	constants.BuildingNaniteFactory:        "NaniteFactory",
}

type prerequisite struct {
	researchID int
	minLevel   int
}

var buildingPrerequisites = map[int][]prerequisite{
	constants.BuildingFusionReactor:   {{113, 3}},
	constants.BuildingShipyard:        {{14, 2}},
	constants.BuildingResearchLab:     {},
	constants.BuildingMissileSilo:     {{21, 1}},
	constants.BuildingNaniteFactory:   {{108, 10}, {14, 10}},
	constants.BuildingTerraformer:     {{15, 1}, {113, 12}},
	constants.BuildingSpaceDock:       {{21, 2}},
}

func meetsPrerequisites(buildingID int, research model.Research, facilities model.Facilities) bool {
	prereqs, ok := buildingPrerequisites[buildingID]
	if !ok {
		return true
	}
	for _, p := range prereqs {
		level := objectLevel(p.researchID, research, facilities)
		if level < p.minLevel {
			return false
		}
	}
	return true
}

func objectLevel(id int, r model.Research, f model.Facilities) int {
	switch id {
	case 14:
		return f.RoboticsFactory
	case 15:
		return f.NaniteFactory
	case 21:
		return f.Shipyard
	case 31:
		return f.ResearchLab
	case 106:
		return r.EspionageTechnology
	case 108:
		return r.ComputerTechnology
	case 109:
		return r.WeaponTechnology
	case 110:
		return r.ShieldingTechnology
	case 111:
		return r.ArmourTechnology
	case 113:
		return r.EnergyTechnology
	case 114:
		return r.HyperspaceTechnology
	case 115:
		return r.CombustionDrive
	case 117:
		return r.ImpulseDrive
	case 118:
		return r.HyperspaceDrive
	case 120:
		return r.LaserTechnology
	case 121:
		return r.IonTechnology
	case 122:
		return r.PlasmaTechnology
	case 123:
		return r.IntergalacticResearchNetwork
	case 124:
		return r.Astrophysics
	case 199:
		return r.GravitonTechnology
	default:
		return 0
	}
}

type buildKey struct {
	planetID   int
	buildingID int
}

type planetState struct {
	planet       model.Planet
	resources    model.Resources
	buildings    model.ResourceBuildings
	facilities   model.Facilities
	constructions model.Constructions
}

type PlanetBuildPlan struct {
	PlanetID      int     `json:"planetId"`
	PlanetName    string  `json:"planetName"`
	BuildingID    int     `json:"buildingId"`
	BuildingName  string  `json:"buildingName"`
	CurrentLevel  int     `json:"currentLevel"`
	TargetLevel   int     `json:"targetLevel"`
	CostMetal     int     `json:"costMetal"`
	CostCrystal   int     `json:"costCrystal"`
	CostDeuterium int     `json:"costDeuterium"`
	ROIScore      float64 `json:"roiScore"`
	Tier          string  `json:"tier"`
	Affordable    bool    `json:"affordable"`
}

type ResearchPlan struct {
	ResearchID    int    `json:"researchId"`
	ResearchName  string `json:"researchName"`
	CurrentLevel  int    `json:"currentLevel"`
	TargetLevel   int    `json:"targetLevel"`
	CostMetal     int    `json:"costMetal"`
	CostCrystal   int    `json:"costCrystal"`
	CostDeuterium int    `json:"costDeuterium"`
}

type BuildPlan struct {
	Planets  []PlanetBuildPlan `json:"planets"`
	Research *ResearchPlan     `json:"research"`
}

type Builder struct {
	client        ogamex.ClientInterface
	stateMgr      BuilderStateReader
	db            *sql.DB
	cfg           config.AutoBuildConfig
	log           *slog.Logger
	antiDetectPct float64
	broadcaster   Broadcaster
	cooldown      map[buildKey]time.Time
	plan          BuildPlan
	planMu        sync.RWMutex
}

type Broadcaster interface {
	Broadcast(msgType string, data interface{})
}

func NewBuilder(client ogamex.ClientInterface, stateMgr BuilderStateReader, db *sql.DB, cfg config.AutoBuildConfig, log *slog.Logger) *Builder {
	return &Builder{
		client:        client,
		stateMgr:      stateMgr,
		db:            db,
		cfg:           cfg,
		log:           log.With("component", "builder"),
		antiDetectPct: 0.07,
		cooldown:      make(map[buildKey]time.Time),
	}
}

func (b *Builder) GetPlan() BuildPlan {
	b.planMu.RLock()
	defer b.planMu.RUnlock()
	return b.plan
}

func (b *Builder) setPlan(plan BuildPlan) {
	b.planMu.Lock()
	defer b.planMu.Unlock()
	b.plan = plan
}

func (b *Builder) SetBroadcaster(br Broadcaster) {
	b.broadcaster = br
}

func (b *Builder) broadcast(msgType string, data interface{}) {
	if b.broadcaster != nil {
		b.broadcaster.Broadcast(msgType, data)
	}
}

func (b *Builder) Run(ctx context.Context) {
	interval := time.Duration(b.cfg.PollIntervalMs) * time.Millisecond
	b.log.Info("Starting builder", "interval", interval)

	for {
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

func (b *Builder) poll(ctx context.Context) {
	if !b.cfg.Enabled {
		return
	}

	if err := b.stateMgr.RefreshNow(ctx); err != nil {
		b.log.Error("Failed to force-refresh state", "error", err)
	}

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

	b.computeAndSetPlan(ctx, planets, speed, research)

	spent := make(map[int]model.Resources)

	if b.tryEnergy(ctx, planets, speed, research, spent) {
		b.tryResearch(ctx, planets, research, spent)
		return
	}
	if b.tryMines(ctx, planets, speed, research, spent) {
		b.tryResearch(ctx, planets, research, spent)
		return
	}
	if b.tryInfrastructure(ctx, planets, research, spent) {
		b.tryResearch(ctx, planets, research, spent)
		return
	}
	if b.tryResearch(ctx, planets, research, spent) {
		return
	}
	if b.tryStorage(ctx, planets, research, spent) {
		b.tryResearch(ctx, planets, research, spent)
		return
	}

	b.log.Debug("No viable build candidates")
}

func (b *Builder) computeAndSetPlan(ctx context.Context, planets []model.Planet, speed int, research model.Research) {
	states := b.gatherPlanetStates(ctx, planets, false)
	planetPlans := make(map[int]*PlanetBuildPlan)

	for _, ps := range states {
		if ps.resources.Energy < 0 {
			for _, buildingID := range []int{constants.BuildingSolarPlant, constants.BuildingFusionReactor} {
				if !meetsPrerequisites(buildingID, research, ps.facilities) {
					continue
				}
				currentLevel := buildingLevel(ps.buildings, ps.facilities, buildingID)
				buildingName := buildingNameToMaxLevelKey[buildingID]
				maxLevel := b.resolveMaxLevel(buildingName, ps.planet.Name)
				result, viable := CalculateROI(buildingID, currentLevel, ps.planet, ps.buildings, ps.facilities, research, ps.resources, speed, maxLevel)
				if viable {
					pp := &PlanetBuildPlan{
						PlanetID:      ps.planet.ID,
						PlanetName:    ps.planet.Name,
						BuildingID:    result.BuildingID,
						BuildingName:  result.BuildingName,
						CurrentLevel:  result.CurrentLevel,
						TargetLevel:   result.TargetLevel,
						CostMetal:     result.CostMetal,
						CostCrystal:   result.CostCrystal,
						CostDeuterium: result.CostDeuterium,
						ROIScore:      result.ROIScore,
						Tier:          "energy",
						Affordable:    ps.resources.Metal >= result.CostMetal && ps.resources.Crystal >= result.CostCrystal && ps.resources.Deuterium >= result.CostDeuterium,
					}
					existing, ok := planetPlans[ps.planet.ID]
					if !ok || pp.ROIScore > existing.ROIScore {
						planetPlans[ps.planet.ID] = pp
					}
				}
			}
		}

		mineIDs := []int{constants.BuildingMetalMine, constants.BuildingCrystalMine, constants.BuildingDeuteriumSynthesizer}
		for _, mineID := range mineIDs {
			currentLevel := buildingLevel(ps.buildings, ps.facilities, mineID)
			buildingName := buildingNameToMaxLevelKey[mineID]
			maxLevel := b.resolveMaxLevel(buildingName, ps.planet.Name)
			result, viable := CalculateROI(mineID, currentLevel, ps.planet, ps.buildings, ps.facilities, research, ps.resources, speed, maxLevel)
			if !viable {
				continue
			}
			pp := &PlanetBuildPlan{
				PlanetID:      ps.planet.ID,
				PlanetName:    ps.planet.Name,
				BuildingID:    result.BuildingID,
				BuildingName:  result.BuildingName,
				CurrentLevel:  result.CurrentLevel,
				TargetLevel:   result.TargetLevel,
				CostMetal:     result.CostMetal,
				CostCrystal:   result.CostCrystal,
				CostDeuterium: result.CostDeuterium,
				ROIScore:      result.ROIScore,
				Tier:          "mines",
				Affordable:    ps.resources.Metal >= result.CostMetal && ps.resources.Crystal >= result.CostCrystal && ps.resources.Deuterium >= result.CostDeuterium,
			}
			existing, ok := planetPlans[ps.planet.ID]
			if !ok || pp.ROIScore > existing.ROIScore {
				planetPlans[ps.planet.ID] = pp
			}
		}

		infraOrder := []int{constants.BuildingResearchLab, constants.BuildingRoboticsFactory, constants.BuildingShipyard, constants.BuildingNaniteFactory}
		for _, buildingID := range infraOrder {
			if !meetsPrerequisites(buildingID, research, ps.facilities) {
				continue
			}
			currentLevel := buildingLevel(ps.buildings, ps.facilities, buildingID)
			buildingName := buildingNameToMaxLevelKey[buildingID]
			maxLevel := b.resolveMaxLevel(buildingName, ps.planet.Name)
			if maxLevel > 0 && currentLevel >= maxLevel {
				continue
			}
			def, ok := BuildingDefs[buildingID]
			if !ok {
				continue
			}
			cost := BuildingCost(model.Resources{Metal: def.BaseMetal, Crystal: def.BaseCrystal, Deuterium: def.BaseDeut}, def.Factor, currentLevel+1)
			pp := &PlanetBuildPlan{
				PlanetID:      ps.planet.ID,
				PlanetName:    ps.planet.Name,
				BuildingID:    buildingID,
				BuildingName:  def.Name,
				CurrentLevel:  currentLevel,
				TargetLevel:   currentLevel + 1,
				CostMetal:     cost.Metal,
				CostCrystal:   cost.Crystal,
				CostDeuterium: cost.Deuterium,
				Tier:          "infrastructure",
				Affordable:    ps.resources.Metal >= cost.Metal && ps.resources.Crystal >= cost.Crystal && ps.resources.Deuterium >= cost.Deuterium,
			}
			if _, ok := planetPlans[ps.planet.ID]; !ok {
				planetPlans[ps.planet.ID] = pp
			}
			break
		}
	}

	var plan BuildPlan
	for _, pp := range planetPlans {
		plan.Planets = append(plan.Planets, *pp)
	}
	sort.Slice(plan.Planets, func(i, j int) bool {
		return plan.Planets[i].ROIScore > plan.Planets[j].ROIScore
	})

	if len(b.cfg.ResearchOrder) > 0 {
		for _, techName := range b.cfg.ResearchOrder {
			researchID, ok := ResearchNameToID[techName]
			if !ok {
				continue
			}
			maxLevel := b.resolveMaxLevel(techName, "")
			currentLevel := objectLevel(researchID, research, model.Facilities{})
			if maxLevel > 0 && currentLevel >= maxLevel {
				continue
			}
			if !MeetsResearchPrerequisites(researchID, research, model.Facilities{}) {
				continue
			}
			cost := ResearchCost(researchID, currentLevel)
			def := ResearchDefs[researchID]
			plan.Research = &ResearchPlan{
				ResearchID:    researchID,
				ResearchName:  def.Name,
				CurrentLevel:  currentLevel,
				TargetLevel:   currentLevel + 1,
				CostMetal:     cost.Metal,
				CostCrystal:   cost.Crystal,
				CostDeuterium: cost.Deuterium,
			}
			break
		}
	}

	b.setPlan(plan)
}

func (b *Builder) addSpent(spent map[int]model.Resources, planetID int, cost model.Resources) {
	s := spent[planetID]
	s.Metal += cost.Metal
	s.Crystal += cost.Crystal
	s.Deuterium += cost.Deuterium
	spent[planetID] = s
}

func deductSpent(res model.Resources, spent map[int]model.Resources, planetID int) model.Resources {
	if s, ok := spent[planetID]; ok {
		res.Metal -= s.Metal
		res.Crystal -= s.Crystal
		res.Deuterium -= s.Deuterium
	}
	return res
}

func (b *Builder) gatherPlanetStates(ctx context.Context, planets []model.Planet, requireFreeBuildSlot bool) []planetState {
	var states []planetState
	for _, planet := range planets {
		if planet.FieldsTotal > 0 && planet.FieldsUsed >= planet.FieldsTotal {
			b.log.Warn("Planet is full, skipping", "planet", planet.Name, "fields", fmt.Sprintf("%d/%d", planet.FieldsUsed, planet.FieldsTotal))
			continue
		}

		constructions, err := b.client.GetConstructions(ctx, planet.ID)
		if err != nil {
			b.log.Error("Failed to get constructions, skipping planet", "planet", planet.Name, "error", err)
			continue
		}
		if requireFreeBuildSlot && constructions.Building.ID != 0 {
			b.log.Debug("Planet has active construction, skipping", "planet", planet.Name, "buildingID", constructions.Building.ID)
			continue
		}

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

		states = append(states, planetState{
			planet:        planet,
			resources:     resources,
			buildings:     buildings,
			facilities:    facilities,
			constructions: constructions,
		})
	}
	return states
}

func (b *Builder) tryEnergy(ctx context.Context, planets []model.Planet, speed int, research model.Research, spent map[int]model.Resources) bool {
	states := b.gatherPlanetStates(ctx, planets, true)
	if len(states) == 0 {
		return false
	}

	for _, ps := range states {
		if ps.resources.Energy >= 0 {
			continue
		}

		effectiveResources := deductSpent(ps.resources, spent, ps.planet.ID)
		energyBuildingIDs := []int{constants.BuildingSolarPlant, constants.BuildingFusionReactor}
		var candidates []ROIResult
		for _, buildingID := range energyBuildingIDs {
			if !meetsPrerequisites(buildingID, research, ps.facilities) {
				continue
			}
			currentLevel := buildingLevel(ps.buildings, ps.facilities, buildingID)
			buildingName := buildingNameToMaxLevelKey[buildingID]
			maxLevel := b.resolveMaxLevel(buildingName, ps.planet.Name)
			result, viable := CalculateROI(buildingID, currentLevel, ps.planet, ps.buildings, ps.facilities, research, effectiveResources, speed, maxLevel)
			if viable {
				candidates = append(candidates, result)
			}
		}

		if len(candidates) == 0 {
			continue
		}

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].ROIScore > candidates[j].ROIScore
		})

		return b.executeBuild(ctx, candidates, spent)
	}

	return false
}

func (b *Builder) tryMines(ctx context.Context, planets []model.Planet, speed int, research model.Research, spent map[int]model.Resources) bool {
	states := b.gatherPlanetStates(ctx, planets, true)
	if len(states) == 0 {
		return false
	}

	mineIDs := []int{
		constants.BuildingMetalMine,
		constants.BuildingCrystalMine,
		constants.BuildingDeuteriumSynthesizer,
	}

	var candidates []ROIResult
	for _, ps := range states {
		effectiveResources := deductSpent(ps.resources, spent, ps.planet.ID)
		for _, mineID := range mineIDs {
			currentLevel := buildingLevel(ps.buildings, ps.facilities, mineID)
			buildingName := buildingNameToMaxLevelKey[mineID]
			maxLevel := b.resolveMaxLevel(buildingName, ps.planet.Name)
			result, viable := CalculateROI(mineID, currentLevel, ps.planet, ps.buildings, ps.facilities, research, effectiveResources, speed, maxLevel)
			if viable {
				candidates = append(candidates, result)
			}
		}
	}

	if len(candidates) == 0 {
		return false
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ROIScore > candidates[j].ROIScore
	})

	return b.executeBuild(ctx, candidates, spent)
}

func (b *Builder) tryInfrastructure(ctx context.Context, planets []model.Planet, research model.Research, spent map[int]model.Resources) bool {
	states := b.gatherPlanetStates(ctx, planets, true)
	if len(states) == 0 {
		return false
	}

	infraOrder := []int{
		constants.BuildingResearchLab,
		constants.BuildingRoboticsFactory,
		constants.BuildingShipyard,
		constants.BuildingNaniteFactory,
	}

	var candidates []ROIResult
	for _, buildingID := range infraOrder {
		if !meetsPrerequisites(buildingID, research, model.Facilities{}) {
			continue
		}
		buildingName := buildingNameToMaxLevelKey[buildingID]
		for _, ps := range states {
			currentLevel := buildingLevel(ps.buildings, ps.facilities, buildingID)
			maxLevel := b.resolveMaxLevel(buildingName, ps.planet.Name)
			if maxLevel == 0 || currentLevel >= maxLevel {
				continue
			}
			result, viable := CalculateROI(buildingID, currentLevel, ps.planet, ps.buildings, ps.facilities, research, ps.resources, 1, maxLevel)
			if viable {
				candidates = append(candidates, result)
				break
			}
		}
		if len(candidates) > 0 {
			break
		}
	}

	if len(candidates) == 0 {
		return false
	}

	return b.executeBuild(ctx, candidates, spent)
}

func (b *Builder) tryResearch(ctx context.Context, planets []model.Planet, research model.Research, spent map[int]model.Resources) bool {
	if len(b.cfg.ResearchOrder) == 0 {
		return false
	}

	states := b.gatherPlanetStates(ctx, planets, false)

	var bestLabPlanet *planetState
	for i := range states {
		ps := &states[i]
		if ps.facilities.ResearchLab <= 0 {
			continue
		}
		if ps.constructions.Research.ID != 0 {
			continue
		}
		if bestLabPlanet == nil || ps.facilities.ResearchLab > bestLabPlanet.facilities.ResearchLab {
			bestLabPlanet = ps
		}
	}
	if bestLabPlanet == nil {
		return false
	}

	for _, techName := range b.cfg.ResearchOrder {
		researchID, ok := ResearchNameToID[techName]
		if !ok {
			continue
		}

		maxLevel := b.resolveMaxLevel(techName, "")
		currentLevel := objectLevel(researchID, research, bestLabPlanet.facilities)
		if maxLevel > 0 && currentLevel >= maxLevel {
			continue
		}

		if !MeetsResearchPrerequisites(researchID, research, bestLabPlanet.facilities) {
			continue
		}

		cost := ResearchCost(researchID, currentLevel)
		effectiveResources := deductSpent(bestLabPlanet.resources, spent, bestLabPlanet.planet.ID)
		if effectiveResources.Metal < cost.Metal ||
			effectiveResources.Crystal < cost.Crystal ||
			effectiveResources.Deuterium < cost.Deuterium {
			continue
		}

		liveResources, err := b.client.GetResources(ctx, bestLabPlanet.planet.ID)
		if err != nil {
			b.log.Warn("Failed to fetch live resources for research, using cached", "planet", bestLabPlanet.planet.ID, "error", err)
			liveResources = bestLabPlanet.resources
		}
		effectiveLiveResources := deductSpent(liveResources, spent, bestLabPlanet.planet.ID)
		if effectiveLiveResources.Metal < cost.Metal ||
			effectiveLiveResources.Crystal < cost.Crystal ||
			effectiveLiveResources.Deuterium < cost.Deuterium {
			b.log.Debug("Skipping research — insufficient live resources",
				"planet", bestLabPlanet.planet.ID,
				"research", ResearchDefs[researchID].Name,
				"need", fmt.Sprintf("metal=%d crystal=%d deut=%d", cost.Metal, cost.Crystal, cost.Deuterium),
				"have", fmt.Sprintf("metal=%d crystal=%d deut=%d", effectiveLiveResources.Metal, effectiveLiveResources.Crystal, effectiveLiveResources.Deuterium),
			)
			continue
		}

		def := ResearchDefs[researchID]

		if err := b.client.BuildResearch(ctx, bestLabPlanet.planet.ID, researchID); err != nil {
			b.log.Error("BuildResearch failed",
				"planet", bestLabPlanet.planet.ID,
				"research", def.Name,
				"error", err)
			return true
		}

		b.log.Info("Started research",
			"planet", bestLabPlanet.planet.ID,
			"research", def.Name,
			"fromLevel", currentLevel,
			"toLevel", currentLevel+1,
		)

		b.addSpent(spent, bestLabPlanet.planet.ID, model.Resources{Metal: cost.Metal, Crystal: cost.Crystal, Deuterium: cost.Deuterium})

		b.broadcast("research", map[string]interface{}{
			"planetId":     bestLabPlanet.planet.ID,
			"researchId":   researchID,
			"researchName": def.Name,
			"fromLevel":    currentLevel,
			"toLevel":      currentLevel + 1,
		})
		return true
	}

	return false
}

func (b *Builder) tryStorage(ctx context.Context, planets []model.Planet, research model.Research, spent map[int]model.Resources) bool {
	states := b.gatherPlanetStates(ctx, planets, true)
	if len(states) == 0 {
		return false
	}

	storageBuildings := []int{
		constants.BuildingMetalStorage,
		constants.BuildingCrystalStorage,
		constants.BuildingDeuteriumStorage,
	}

	var candidates []ROIResult
	for _, ps := range states {
		for _, buildingID := range storageBuildings {
			currentLevel := buildingLevel(ps.buildings, ps.facilities, buildingID)
			def, ok := BuildingDefs[buildingID]
			if !ok {
				continue
			}

			var resourceAmount int
			var capacity float64
			switch buildingID {
			case constants.BuildingMetalStorage:
				resourceAmount = ps.resources.Metal
				capacity = storageCapacity(def.BaseMetal, currentLevel)
			case constants.BuildingCrystalStorage:
				resourceAmount = ps.resources.Crystal
				capacity = storageCapacity(def.BaseCrystal, currentLevel)
			case constants.BuildingDeuteriumStorage:
				resourceAmount = ps.resources.Deuterium
				capacity = storageCapacity(0, currentLevel)
			}

			if capacity <= 0 || float64(resourceAmount) < capacity*0.8 {
				continue
			}

			buildingName := buildingNameToMaxLevelKey[buildingID]
			maxLevel := b.resolveMaxLevel(buildingName, ps.planet.Name)
			result, viable := CalculateROI(buildingID, currentLevel, ps.planet, ps.buildings, ps.facilities, research, ps.resources, 1, maxLevel)
			if viable {
				candidates = append(candidates, result)
			}
		}
	}

	if len(candidates) == 0 {
		return false
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ROIScore > candidates[j].ROIScore
	})

	return b.executeBuild(ctx, candidates, spent)
}

func storageCapacity(baseCost int, level int) float64 {
	if level <= 0 {
		return 10000
	}
	return 5000 * float64(int(1)<<uint(level))
}

func (b *Builder) executeBuild(ctx context.Context, candidates []ROIResult, spent map[int]model.Resources) bool {
	var filtered []ROIResult
	for _, c := range candidates {
		key := buildKey{c.PlanetID, c.BuildingID}
		if until, ok := b.cooldown[key]; ok && time.Now().Before(until) {
			continue
		}
		filtered = append(filtered, c)
	}
	if len(filtered) == 0 {
		b.log.Debug("All candidates on cooldown")
		return false
	}
	candidates = filtered

	pickIdx := 0
	if len(candidates) > 1 && b.antiDetectPct > 0 && rand.Float64() < b.antiDetectPct {
		pickIdx = 1
		b.log.Info("Anti-detection: picking 2nd-best ROI candidate")
	}
	best := candidates[pickIdx]

	liveResources, err := b.client.GetResources(ctx, best.PlanetID)
	if err != nil {
		b.log.Warn("Failed to fetch live resources, using cached", "planet", best.PlanetID, "error", err)
	} else {
		effectiveLive := deductSpent(liveResources, spent, best.PlanetID)
		if effectiveLive.Metal < best.CostMetal || effectiveLive.Crystal < best.CostCrystal || effectiveLive.Deuterium < best.CostDeuterium {
			b.log.Debug("Skipping build — insufficient live resources",
				"planet", best.PlanetID,
				"building", best.BuildingName,
				"need", fmt.Sprintf("metal=%d crystal=%d deut=%d", best.CostMetal, best.CostCrystal, best.CostDeuterium),
				"have", fmt.Sprintf("metal=%d crystal=%d deut=%d", effectiveLive.Metal, effectiveLive.Crystal, effectiveLive.Deuterium),
			)
			return false
		}
	}

	if err := b.client.BuildBuilding(ctx, best.PlanetID, best.BuildingID); err != nil {
		b.log.Error("BuildBuilding failed",
			"planet", best.PlanetID,
			"building", best.BuildingName,
			"error", err)
		b.cooldown[buildKey{best.PlanetID, best.BuildingID}] = time.Now().Add(10 * time.Minute)
		return true
	}

	b.log.Info("Built upgrade",
		"planet", best.PlanetID,
		"building", best.BuildingName,
		"fromLevel", best.CurrentLevel,
		"toLevel", best.TargetLevel,
		"roiScore", fmt.Sprintf("%.6f", best.ROIScore),
		"cost", fmt.Sprintf("metal=%d crystal=%d deut=%d", best.CostMetal, best.CostCrystal, best.CostDeuterium),
		"energyDelta", best.EnergyDelta,
	)

	b.cooldown[buildKey{best.PlanetID, best.BuildingID}] = time.Now().Add(2 * time.Minute)

	b.addSpent(spent, best.PlanetID, model.Resources{Metal: best.CostMetal, Crystal: best.CostCrystal, Deuterium: best.CostDeuterium})

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
	return true
}

func (b *Builder) resolveMaxLevel(buildingName, planetName string) int {
	if b.cfg.PlanetOverrides != nil {
		if planetOverrides, ok := b.cfg.PlanetOverrides[planetName]; ok {
			if maxLvl, ok := planetOverrides[buildingName]; ok {
				return maxLvl
			}
		}
	}
	if b.cfg.MaxLevels != nil {
		if maxLvl, ok := b.cfg.MaxLevels[buildingName]; ok {
			return maxLvl
		}
	}
	return 0
}

func buildingLevel(buildings model.ResourceBuildings, facilities model.Facilities, buildingID int) int {
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
	case constants.BuildingMetalStorage:
		return buildings.MetalStorage
	case constants.BuildingCrystalStorage:
		return buildings.CrystalStorage
	case constants.BuildingDeuteriumStorage:
		return buildings.DeuteriumStorage
	case constants.BuildingRoboticsFactory:
		return facilities.RoboticsFactory
	case constants.BuildingShipyard:
		return facilities.Shipyard
	case constants.BuildingResearchLab:
		return facilities.ResearchLab
	case constants.BuildingNaniteFactory:
		return facilities.NaniteFactory
	default:
		return 0
	}
}

func (b *Builder) recordBuildEvent(ctx context.Context, result ROIResult) error {
	_, err := b.db.ExecContext(ctx,
		`INSERT INTO build_events (planet_id, building_id, building_name, from_level, to_level, cost_metal, cost_crystal, cost_deut, roi_score, build_time_seconds)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.PlanetID, result.BuildingID, result.BuildingName,
		result.CurrentLevel, result.TargetLevel,
		result.CostMetal, result.CostCrystal, result.CostDeuterium,
		result.ROIScore, int(result.BuildTime.Seconds()))
	if err != nil {
		return fmt.Errorf("recording build event for planet %d building %s: %w", result.PlanetID, result.BuildingName, err)
	}
	return nil
}
