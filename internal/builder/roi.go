// Package builder contains the ROI calculation engine for auto-building.
// All functions are pure math with no I/O dependencies.
package builder

import (
	"math"
	"time"

	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/model"
)

// BuildingDef defines the base properties of an OGame building.
type BuildingDef struct {
	Name           string
	BaseMetal      int
	BaseCrystal    int
	BaseDeut       int
	Factor         float64
	EnergyConsumer bool // true = consumes energy (mines), false = produces energy (solar/fusion)
}

// BuildingDefs maps building IDs to their definitions.
// Source: verified from ogamed source (alaingilbert/ogame)
var BuildingDefs = map[int]BuildingDef{
	constants.BuildingMetalMine:            {"Metal Mine", 60, 15, 0, 1.5, true},
	constants.BuildingCrystalMine:          {"Crystal Mine", 48, 24, 0, 1.6, true},
	constants.BuildingDeuteriumSynthesizer: {"Deuterium Synthesizer", 225, 75, 0, 1.5, true},
	constants.BuildingSolarPlant:           {"Solar Plant", 75, 30, 0, 1.5, false},
	constants.BuildingFusionReactor:        {"Fusion Reactor", 900, 360, 180, 1.8, false},
}

// ROIResult represents the ROI analysis for a single building upgrade candidate.
type ROIResult struct {
	PlanetID           int
	BuildingID         int
	BuildingName       string
	CurrentLevel       int
	TargetLevel        int
	CostMetal          int
	CostCrystal        int
	CostDeuterium      int
	ProductionIncrease float64 // hourly production increase in metal-equivalent
	ROIScore           float64 // productionIncrease / totalCostValue
	BuildTime          time.Duration
	EnergyDelta        int // additional energy consumed (negative) or produced (positive)
}

// BuildingCost returns the resources needed to upgrade to the given level.
// Formula: baseCost * factor^(level-1)
// Source: verified from ogamed source (baseLevelable.go)
func BuildingCost(baseCost model.Resources, factor float64, level int) model.Resources {
	return model.Resources{
		Metal:     int(float64(baseCost.Metal) * math.Pow(factor, float64(level-1))),
		Crystal:   int(float64(baseCost.Crystal) * math.Pow(factor, float64(level-1))),
		Deuterium: int(float64(baseCost.Deuterium) * math.Pow(factor, float64(level-1))),
	}
}

// MetalProduction returns hourly metal production at the given level.
// Formula: 30 * (1 + plasmaTech/100) * speed * level * 1.1^level + 30*speed (basic income)
// Source: verified from ogamed source (metalMine.go)
func MetalProduction(level, plasmaTech, universeSpeed int) int {
	if level == 0 {
		return 30 * universeSpeed
	}
	basicIncome := 30.0 * float64(universeSpeed)
	levelProduction := 30.0 * (1.0 + float64(plasmaTech)/100.0) * float64(universeSpeed) *
		float64(level) * math.Pow(1.1, float64(level))
	return int(levelProduction + basicIncome)
}

// CrystalProduction returns hourly crystal production at the given level.
// Note: plasma bonus is 0.66% per level (not 1% like metal).
// Formula: 20 * speed * (1 + plasmaTech*0.0066) * level * 1.1^level + 15*speed (basic income)
// Source: verified from ogamed source (crystalMine.go)
func CrystalProduction(level, plasmaTech, universeSpeed int) int {
	if level == 0 {
		return 15 * universeSpeed
	}
	basicIncome := 15.0 * float64(universeSpeed)
	levelProduction := 20.0 * float64(universeSpeed) *
		(1.0 + float64(plasmaTech)*0.0066) *
		float64(level) * math.Pow(1.1, float64(level))
	return int(levelProduction + basicIncome)
}

// DeuteriumProduction returns hourly deuterium production at the given level.
// Note: depends on average planet temperature. Plasma bonus is 0.33% per level.
// Formula: 10 * (1 + plasmaTech*0.0033) * level * 1.1^level * (-0.004*avgTemp + 1.36) * speed
// Source: verified from ogamed source (deuteriumSynthesizer.go)
func DeuteriumProduction(level, plasmaTech, avgTemperature, universeSpeed int) int {
	if level == 0 {
		return 0
	}
	production := 10.0 * (1.0 + float64(plasmaTech)*0.0033) *
		float64(level) * math.Pow(1.1, float64(level)) *
		(-0.004*float64(avgTemperature) + 1.36) * float64(universeSpeed)
	return int(math.Round(production))
}

// MineEnergyConsumption returns energy consumed by metal/crystal mines at the given level.
// Formula: ceil(10 * level * 1.1^level)
// Source: verified from ogamed source (metalMine.go, crystalMine.go)
func MineEnergyConsumption(level int) int {
	return int(math.Ceil(10.0 * float64(level) * math.Pow(1.1, float64(level))))
}

// DeutMineEnergyConsumption returns energy consumed by deuterium synthesizer at the given level.
// Formula: ceil(20 * level * 1.1^level)
// Source: verified from ogamed source (deuteriumSynthesizer.go)
func DeutMineEnergyConsumption(level int) int {
	return int(math.Ceil(20.0 * float64(level) * math.Pow(1.1, float64(level))))
}

// SolarProduction returns energy produced by solar plant at the given level.
// Formula: floor(20 * level * 1.1^level)
// Source: verified from ogamed source (solarPlant.go)
func SolarProduction(level int) int {
	if level == 0 {
		return 0
	}
	return int(math.Floor(20.0 * float64(level) * math.Pow(1.1, float64(level))))
}

// FusionProduction returns energy produced by fusion reactor at the given level.
// Formula: round(30 * level * (1.05 + energyTech*0.01)^level)
// Source: verified from ogamed source (fusionReactor.go)
func FusionProduction(level, energyTech int) int {
	if level == 0 {
		return 0
	}
	return int(math.Round(30.0 * float64(level) * math.Pow(1.05+float64(energyTech)*0.01, float64(level))))
}

// ConstructionTime returns how long a building takes to construct.
// Formula: (metalCost + crystalCost) / (2500 * (1 + roboticsFactory) * speed * 2^naniteFactory) hours
// Source: verified from ogamed source (baseBuilding.go)
func ConstructionTime(metalCost, crystalCost, roboticsFactory, naniteFactory, universeSpeed int) time.Duration {
	hours := float64(metalCost+crystalCost) /
		(2500.0 * (1.0 + float64(roboticsFactory)) * float64(universeSpeed) * math.Pow(2, float64(naniteFactory)))
	seconds := hours * 3600
	if seconds < 1 {
		seconds = 1
	}
	return time.Duration(int(math.Floor(seconds))) * time.Second
}

// CalculateROI computes ROI for upgrading buildingID on a specific planet.
// Pure function — no side effects, no external dependencies.
func CalculateROI(
	buildingID int,
	currentLevel int,
	planet model.Planet,
	buildings model.ResourceBuildings,
	facilities model.Facilities,
	research model.Research,
	resources model.Resources,
	universeSpeed int,
	maxLevel int,
) (ROIResult, bool) {
	// Check max level cap
	if currentLevel >= maxLevel {
		return ROIResult{}, false
	}

	targetLevel := currentLevel + 1
	def, ok := BuildingDefs[buildingID]
	if !ok {
		return ROIResult{}, false
	}

	// Calculate build cost
	cost := BuildingCost(model.Resources{
		Metal: def.BaseMetal, Crystal: def.BaseCrystal, Deuterium: def.BaseDeut,
	}, def.Factor, targetLevel)

	// Check affordability
	if resources.Metal < cost.Metal || resources.Crystal < cost.Crystal || resources.Deuterium < cost.Deuterium {
		return ROIResult{}, false
	}

	// Calculate production increase in metal-equivalent
	prodIncrease := productionIncrease(buildingID, currentLevel, targetLevel, research, avgTemp(planet), universeSpeed)

	// Calculate energy delta
	energyDelta := energyDelta(buildingID, currentLevel, targetLevel, research)

	// For energy consumers: check if planet has enough energy surplus
	if def.EnergyConsumer {
		// additionalEnergyNeeded is negative (consumes more energy)
		additionalEnergyNeeded := -energyDelta // make positive for comparison
		// resources.Energy represents current surplus (positive = surplus, negative = deficit)
		if resources.Energy < additionalEnergyNeeded {
			return ROIResult{}, false
		}
	}

	// Calculate ROI score (production increase / total cost valued in metal-equivalent)
	// Standard trade ratios: metal=1, crystal=1.5, deuterium=2.0
	totalCostValue := float64(cost.Metal) + float64(cost.Crystal)*1.5 + float64(cost.Deuterium)*2.0
	roiScore := prodIncrease / totalCostValue

	// Calculate build time
	buildTime := ConstructionTime(cost.Metal, cost.Crystal, facilities.RoboticsFactory, facilities.NaniteFactory, universeSpeed)

	return ROIResult{
		PlanetID:           planet.ID,
		BuildingID:         buildingID,
		BuildingName:       def.Name,
		CurrentLevel:       currentLevel,
		TargetLevel:        targetLevel,
		CostMetal:          cost.Metal,
		CostCrystal:        cost.Crystal,
		CostDeuterium:      cost.Deuterium,
		ProductionIncrease: prodIncrease,
		ROIScore:           roiScore,
		BuildTime:          buildTime,
		EnergyDelta:        energyDelta,
	}, true
}

// productionIncrease computes the metal-equivalent production increase from upgrading
// a building from currentLevel to targetLevel.
// Trade ratios: metal=1, crystal=1.5, deuterium=2.0
func productionIncrease(buildingID int, currentLevel, targetLevel int, research model.Research, avgTemp, universeSpeed int) float64 {
	switch buildingID {
	case constants.BuildingMetalMine:
		before := MetalProduction(currentLevel, research.PlasmaTechnology, universeSpeed)
		after := MetalProduction(targetLevel, research.PlasmaTechnology, universeSpeed)
		return float64(after - before) * 1.0 // metal ratio = 1.0
	case constants.BuildingCrystalMine:
		before := CrystalProduction(currentLevel, research.PlasmaTechnology, universeSpeed)
		after := CrystalProduction(targetLevel, research.PlasmaTechnology, universeSpeed)
		return float64(after-before) * 1.5 // crystal ratio = 1.5
	case constants.BuildingDeuteriumSynthesizer:
		before := DeuteriumProduction(currentLevel, research.PlasmaTechnology, avgTemp, universeSpeed)
		after := DeuteriumProduction(targetLevel, research.PlasmaTechnology, avgTemp, universeSpeed)
		return float64(after-before) * 2.0 // deuterium ratio = 2.0
	case constants.BuildingSolarPlant:
		// Solar plant doesn't produce resources — its value is enabling mines.
		// For ROI purposes, assign a small fixed value based on energy produced.
		before := SolarProduction(currentLevel)
		after := SolarProduction(targetLevel)
		// Value energy at ~0.5 metal-equivalent per unit (rough heuristic)
		return float64(after-before) * 0.5
	case constants.BuildingFusionReactor:
		before := FusionProduction(currentLevel, research.EnergyTechnology)
		after := FusionProduction(targetLevel, research.EnergyTechnology)
		return float64(after-before) * 0.5
	default:
		return 0
	}
}

// energyDelta computes the change in energy consumption/production when upgrading.
// Positive = produces more energy, Negative = consumes more energy.
func energyDelta(buildingID int, currentLevel, targetLevel int, research model.Research) int {
	switch buildingID {
	case constants.BuildingMetalMine:
		return -(MineEnergyConsumption(targetLevel) - MineEnergyConsumption(currentLevel))
	case constants.BuildingCrystalMine:
		return -(MineEnergyConsumption(targetLevel) - MineEnergyConsumption(currentLevel))
	case constants.BuildingDeuteriumSynthesizer:
		return -(DeutMineEnergyConsumption(targetLevel) - DeutMineEnergyConsumption(currentLevel))
	case constants.BuildingSolarPlant:
		return SolarProduction(targetLevel) - SolarProduction(currentLevel)
	case constants.BuildingFusionReactor:
		return FusionProduction(targetLevel, research.EnergyTechnology) - FusionProduction(currentLevel, research.EnergyTechnology)
	default:
		return 0
	}
}

// avgTemp returns the average temperature of a planet.
func avgTemp(p model.Planet) int {
	return (p.TemperatureMin + p.TemperatureMax) / 2
}
