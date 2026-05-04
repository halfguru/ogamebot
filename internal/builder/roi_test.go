package builder

import (
	"math"
	"testing"
	"time"

	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/model"
)

// --- BuildingCost tests ---

func TestBuildingCost(t *testing.T) {
	tests := []struct {
		name       string
		baseMetal  int
		baseCrys   int
		baseDeut   int
		factor     float64
		level      int
		wantMetal  int
		wantCrys   int
		wantDeut   int
	}{
		{
			name:      "MetalMine level 1",
			baseMetal: 60, baseCrys: 15, baseDeut: 0, factor: 1.5, level: 1,
			wantMetal: 60, wantCrys: 15, wantDeut: 0,
		},
		{
			name:      "MetalMine level 2",
			baseMetal: 60, baseCrys: 15, baseDeut: 0, factor: 1.5, level: 2,
			wantMetal: 90, wantCrys: 22, wantDeut: 0, // 60*1.5=90, 15*1.5=22.5→22
		},
		{
			name:      "CrystalMine level 10",
			baseMetal: 48, baseCrys: 24, baseDeut: 0, factor: 1.6, level: 10,
			wantMetal: int(48 * math.Pow(1.6, 9)), wantCrys: int(24 * math.Pow(1.6, 9)), wantDeut: 0,
		},
		{
			name:      "FusionReactor level 5",
			baseMetal: 900, baseCrys: 360, baseDeut: 180, factor: 1.8, level: 5,
			wantMetal: int(900 * math.Pow(1.8, 4)), wantCrys: int(360 * math.Pow(1.8, 4)), wantDeut: int(180 * math.Pow(1.8, 4)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := BuildingCost(model.Resources{Metal: tt.baseMetal, Crystal: tt.baseCrys, Deuterium: tt.baseDeut}, tt.factor, tt.level)
			if cost.Metal != tt.wantMetal {
				t.Errorf("Metal = %d, want %d", cost.Metal, tt.wantMetal)
			}
			if cost.Crystal != tt.wantCrys {
				t.Errorf("Crystal = %d, want %d", cost.Crystal, tt.wantCrys)
			}
			if cost.Deuterium != tt.wantDeut {
				t.Errorf("Deuterium = %d, want %d", cost.Deuterium, tt.wantDeut)
			}
		})
	}
}

// --- MetalProduction tests ---

func TestMetalProduction(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		plasma   int
		speed    int
		wantMin  int // minimum expected (formula uses int truncation)
		wantExact int // exact expected value when known
	}{
		{
			name: "level 0 returns basic income only",
			level: 0, plasma: 0, speed: 1,
			wantExact: 30, // 30*1
		},
		{
			name: "level 0 speed 7",
			level: 0, plasma: 0, speed: 7,
			wantExact: 210, // 30*7
		},
		{
			name: "level 1 no plasma speed 1",
			level: 1, plasma: 0, speed: 1,
			wantExact: 63, // 30*1*1*1*1.1 + 30*1 = 33 + 30 = 63
		},
		{
			name: "level 1 no plasma speed 7",
			level: 1, plasma: 0, speed: 7,
			wantExact: 441, // 30*7*1*1*1.1 + 30*7 = 231 + 210 = 441
		},
		{
			name: "level 10 plasma 0 speed 1",
			level: 10, plasma: 0, speed: 1,
			wantMin: 750, // rough minimum
		},
		{
			name: "level 10 plasma 10 speed 1",
			level: 10, plasma: 10, speed: 1,
			wantMin: 800, // higher with plasma
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MetalProduction(tt.level, tt.plasma, tt.speed)
			if tt.wantExact != 0 && got != tt.wantExact {
				t.Errorf("MetalProduction(%d, %d, %d) = %d, want exactly %d", tt.level, tt.plasma, tt.speed, got, tt.wantExact)
			}
			if tt.wantMin != 0 && got < tt.wantMin {
				t.Errorf("MetalProduction(%d, %d, %d) = %d, want at least %d", tt.level, tt.plasma, tt.speed, got, tt.wantMin)
			}
		})
	}
}

// --- CrystalProduction tests ---

func TestCrystalProduction(t *testing.T) {
	tests := []struct {
		name      string
		level     int
		plasma    int
		speed     int
		wantExact int
		wantMin   int
	}{
		{
			name: "level 0 returns basic income",
			level: 0, plasma: 0, speed: 1,
			wantExact: 15, // 15*1
		},
		{
			name: "level 0 speed 5",
			level: 0, plasma: 0, speed: 5,
			wantExact: 75, // 15*5
		},
		{
			name: "level 1 no plasma speed 1",
			level: 1, plasma: 0, speed: 1,
			wantExact: 37, // 20*1*(1+0)*1*1.1 + 15*1 = 22 + 15 = 37
		},
		{
			name: "level 10 plasma 0 speed 1",
			level: 10, plasma: 0, speed: 1,
			wantMin: 500,
		},
		{
			name: "level 10 plasma 10 speed 1",
			level: 10, plasma: 10, speed: 1,
			wantMin: 520, // higher with plasma
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CrystalProduction(tt.level, tt.plasma, tt.speed)
			if tt.wantExact != 0 && got != tt.wantExact {
				t.Errorf("CrystalProduction(%d, %d, %d) = %d, want exactly %d", tt.level, tt.plasma, tt.speed, got, tt.wantExact)
			}
			if tt.wantMin != 0 && got < tt.wantMin {
				t.Errorf("CrystalProduction(%d, %d, %d) = %d, want at least %d", tt.level, tt.plasma, tt.speed, got, tt.wantMin)
			}
		})
	}
}

// --- DeuteriumProduction tests ---

func TestDeuteriumProduction(t *testing.T) {
	tests := []struct {
		name      string
		level     int
		plasma    int
		avgTemp   int
		speed     int
		wantExact int
		wantMin   int
	}{
		{
			name: "level 0 returns 0",
			level: 0, plasma: 0, avgTemp: 40, speed: 1,
			wantExact: 0,
		},
		{
			name: "level 1 no plasma avgTemp 40 speed 1",
			level: 1, plasma: 0, avgTemp: 40, speed: 1,
			wantExact: 13, // round(10 * 1 * 1 * 1.1 * (-0.004*40 + 1.36) * 1) = round(13.2) = 13
		},
		{
			name: "level 10 no plasma avgTemp 40 speed 1",
			level: 10, plasma: 0, avgTemp: 40, speed: 1,
			wantMin: 200,
		},
		{
			name: "colder planet produces more deuterium",
			level: 10, plasma: 0, avgTemp: -20, speed: 1,
			wantMin: 350, // colder = more deuterium
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeuteriumProduction(tt.level, tt.plasma, tt.avgTemp, tt.speed)
			if tt.wantExact != 0 && got != tt.wantExact {
				t.Errorf("DeuteriumProduction(%d, %d, %d, %d) = %d, want exactly %d", tt.level, tt.plasma, tt.avgTemp, tt.speed, got, tt.wantExact)
			}
			if tt.wantMin != 0 && got < tt.wantMin {
				t.Errorf("DeuteriumProduction(%d, %d, %d, %d) = %d, want at least %d", tt.level, tt.plasma, tt.avgTemp, tt.speed, got, tt.wantMin)
			}
		})
	}

	// Verify cold produces more than hot
	t.Run("colder produces more", func(t *testing.T) {
		cold := DeuteriumProduction(10, 0, -40, 1)
		hot := DeuteriumProduction(10, 0, 100, 1)
		if cold <= hot {
			t.Errorf("cold(%d) should be > hot(%d)", cold, hot)
		}
	})
}

// --- Energy consumption tests ---

func TestMineEnergyConsumption(t *testing.T) {
	tests := []struct {
		name  string
		level int
		want  int
	}{
		{name: "level 1", level: 1, want: 11}, // ceil(10*1*1.1) = ceil(11) = 11
		{name: "level 10", level: 10, want: 260}, // ceil(10*10*1.1^10) = ceil(259.37) = 260
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MineEnergyConsumption(tt.level)
			if got != tt.want {
				t.Errorf("MineEnergyConsumption(%d) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestDeutMineEnergyConsumption(t *testing.T) {
	tests := []struct {
		name  string
		level int
		want  int
	}{
		{name: "level 1", level: 1, want: 22}, // ceil(20*1*1.1) = ceil(22) = 22
		{name: "level 10", level: 10, want: 519}, // ceil(20*10*1.1^10) = ceil(518.75) = 519
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeutMineEnergyConsumption(tt.level)
			if got != tt.want {
				t.Errorf("DeutMineEnergyConsumption(%d) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

// --- SolarProduction tests ---

func TestSolarProduction(t *testing.T) {
	tests := []struct {
		name  string
		level int
		want  int
	}{
		{name: "level 0", level: 0, want: 0},
		{name: "level 1", level: 1, want: 22}, // floor(20*1*1.1) = floor(22) = 22
		{name: "level 10", level: 10, want: 518}, // floor(20*10*1.1^10) = floor(518.74) = 518
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SolarProduction(tt.level)
			if got != tt.want {
				t.Errorf("SolarProduction(%d) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

// --- FusionProduction tests ---

func TestFusionProduction(t *testing.T) {
	tests := []struct {
		name       string
		level      int
		energyTech int
		want       int
	}{
		{name: "level 0 energyTech 0", level: 0, energyTech: 0, want: 0},
		{name: "level 1 energyTech 0", level: 1, energyTech: 0, want: 32}, // round(30*1*(1.05+0)^1) = round(31.5) = 32
		{name: "level 1 energyTech 5", level: 1, energyTech: 5, want: 33}, // round(30*1*(1.05+0.05)^1) = round(33) = 33
		{name: "level 10 energyTech 0", level: 10, energyTech: 0, want: 489}, // round(30*10*1.05^10) ≈ round(488.65) = 489
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FusionProduction(tt.level, tt.energyTech)
			if got != tt.want {
				t.Errorf("FusionProduction(%d, %d) = %d, want %d", tt.level, tt.energyTech, got, tt.want)
			}
		})
	}
}

// --- ConstructionTime tests ---

func TestConstructionTime(t *testing.T) {
	tests := []struct {
		name             string
		metalCost        int
		crystalCost      int
		roboticsFactory  int
		naniteFactory    int
		universeSpeed    int
		wantMinSeconds   int
		wantMaxSeconds   int
	}{
		{
			name: "basic building no robotics no nanite speed 1",
			metalCost: 60, crystalCost: 15, roboticsFactory: 0, naniteFactory: 0, universeSpeed: 1,
			wantMinSeconds: 100, wantMaxSeconds: 120, // (75/2500)*3600 = 108 seconds
		},
		{
			name: "with robotics 5 speed 1",
			metalCost: 60, crystalCost: 15, roboticsFactory: 5, naniteFactory: 0, universeSpeed: 1,
			wantMinSeconds: 16, wantMaxSeconds: 20, // 108/(1+5) = 18 seconds
		},
		{
			name: "with nanite 1 speed 1",
			metalCost: 60, crystalCost: 15, roboticsFactory: 0, naniteFactory: 1, universeSpeed: 1,
			wantMinSeconds: 50, wantMaxSeconds: 60, // 108/(2) = 54 seconds
		},
		{
			name: "speed 7x",
			metalCost: 60, crystalCost: 15, roboticsFactory: 0, naniteFactory: 0, universeSpeed: 7,
			wantMinSeconds: 14, wantMaxSeconds: 18, // 108/7 ≈ 15.4 seconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConstructionTime(tt.metalCost, tt.crystalCost, tt.roboticsFactory, tt.naniteFactory, tt.universeSpeed)
			gotSeconds := int(got.Seconds())
			if gotSeconds < tt.wantMinSeconds || gotSeconds > tt.wantMaxSeconds {
				t.Errorf("ConstructionTime() = %v (%ds), want between %d-%d seconds", got, gotSeconds, tt.wantMinSeconds, tt.wantMaxSeconds)
			}
		})
	}
}

// --- BuildingDefs tests ---

func TestBuildingDefs(t *testing.T) {
	// Verify all 5 resource building definitions exist
	expectedIDs := []int{
		constants.BuildingMetalMine,
		constants.BuildingCrystalMine,
		constants.BuildingDeuteriumSynthesizer,
		constants.BuildingSolarPlant,
		constants.BuildingFusionReactor,
	}

	for _, id := range expectedIDs {
		def, ok := BuildingDefs[id]
		if !ok {
			t.Errorf("BuildingDefs missing entry for building ID %d", id)
			continue
		}
		if def.Name == "" {
			t.Errorf("BuildingDefs[%d].Name is empty", id)
		}
		if def.Factor <= 0 {
			t.Errorf("BuildingDefs[%d].Factor = %f, want > 0", id, def.Factor)
		}
	}

	// Verify specific definitions
	mm := BuildingDefs[constants.BuildingMetalMine]
	if mm.BaseMetal != 60 || mm.BaseCrystal != 15 || mm.BaseDeut != 0 || mm.Factor != 1.5 || !mm.EnergyConsumer {
		t.Errorf("MetalMine definition wrong: %+v", mm)
	}

	cm := BuildingDefs[constants.BuildingCrystalMine]
	if cm.BaseMetal != 48 || cm.BaseCrystal != 24 || cm.Factor != 1.6 || !cm.EnergyConsumer {
		t.Errorf("CrystalMine definition wrong: %+v", cm)
	}

	ds := BuildingDefs[constants.BuildingDeuteriumSynthesizer]
	if ds.BaseMetal != 225 || ds.BaseCrystal != 75 || ds.Factor != 1.5 || !ds.EnergyConsumer {
		t.Errorf("DeutSynth definition wrong: %+v", ds)
	}

	sp := BuildingDefs[constants.BuildingSolarPlant]
	if sp.BaseMetal != 75 || sp.BaseCrystal != 30 || sp.Factor != 1.5 || sp.EnergyConsumer {
		t.Errorf("SolarPlant definition wrong: %+v", sp)
	}

	fr := BuildingDefs[constants.BuildingFusionReactor]
	if fr.BaseMetal != 900 || fr.BaseCrystal != 360 || fr.BaseDeut != 180 || fr.Factor != 1.8 || fr.EnergyConsumer {
		t.Errorf("FusionReactor definition wrong: %+v", fr)
	}
}

// --- CalculateROI tests ---

func TestCalculateROI_MaxLevel(t *testing.T) {
	// Building at max level should return false
	result, ok := CalculateROI(
		constants.BuildingMetalMine,
		30, // currentLevel
		model.Planet{ID: 1},
		model.ResourceBuildings{MetalMine: 30},
		model.Facilities{},
		model.Research{},
		model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: 9999},
		1,   // universeSpeed
		30,  // maxLevel
	)
	if ok {
		t.Error("CalculateROI should return false for building at max level")
	}
	_ = result
}

func TestCalculateROI_InsufficientResources(t *testing.T) {
	result, ok := CalculateROI(
		constants.BuildingMetalMine,
		10,
		model.Planet{ID: 1, TemperatureMin: 15, TemperatureMax: 65},
		model.ResourceBuildings{MetalMine: 10},
		model.Facilities{},
		model.Research{},
		model.Resources{Metal: 1, Crystal: 1, Deuterium: 0, Energy: 5000},
		1,
		30,
	)
	if !ok {
		t.Error("CalculateROI should return true regardless of resources — affordability checked at build time")
	}
	if result.CostMetal <= 0 {
		t.Error("Expected non-zero cost")
	}
	_ = result
}

func TestCalculateROI_EnergyDeficit(t *testing.T) {
	// MetalMine level 10 → 11 needs more energy. With Energy=5 (deficit), should fail.
	result, ok := CalculateROI(
		constants.BuildingMetalMine,
		10,
		model.Planet{ID: 1, TemperatureMin: 15, TemperatureMax: 65},
		model.ResourceBuildings{MetalMine: 10, SolarPlant: 5},
		model.Facilities{},
		model.Research{},
		model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: -100},
		1,
		30,
	)
	if ok {
		t.Error("CalculateROI should return false for energy deficit on mine upgrade")
	}
	_ = result
}

func TestCalculateROI_SolarPlantAlwaysAllowed(t *testing.T) {
	// Solar plant produces energy, should be allowed even with Energy=0
	result, ok := CalculateROI(
		constants.BuildingSolarPlant,
		10,
		model.Planet{ID: 1, TemperatureMin: 15, TemperatureMax: 65},
		model.ResourceBuildings{SolarPlant: 10},
		model.Facilities{},
		model.Research{},
		model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: -500},
		1,
		30,
	)
	if !ok {
		t.Error("CalculateROI should allow Solar Plant upgrade even with negative energy")
	}
	if result.BuildingID != constants.BuildingSolarPlant {
		t.Errorf("BuildingID = %d, want %d", result.BuildingID, constants.BuildingSolarPlant)
	}
	if result.EnergyDelta <= 0 {
		t.Errorf("Solar Plant upgrade should produce positive EnergyDelta, got %d", result.EnergyDelta)
	}
}

func TestCalculateROI_AffordableMine(t *testing.T) {
	// MetalMine level 10 → 11 with plenty of resources and energy
	result, ok := CalculateROI(
		constants.BuildingMetalMine,
		10,
		model.Planet{ID: 1, TemperatureMin: 15, TemperatureMax: 65},
		model.ResourceBuildings{MetalMine: 10},
		model.Facilities{RoboticsFactory: 5},
		model.Research{PlasmaTechnology: 5},
		model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: 5000},
		1,
		30,
	)
	if !ok {
		t.Fatal("CalculateROI should return true for affordable mine upgrade with energy surplus")
	}

	// Verify result fields
	if result.PlanetID != 1 {
		t.Errorf("PlanetID = %d, want 1", result.PlanetID)
	}
	if result.BuildingID != constants.BuildingMetalMine {
		t.Errorf("BuildingID = %d, want %d", result.BuildingID, constants.BuildingMetalMine)
	}
	if result.CurrentLevel != 10 {
		t.Errorf("CurrentLevel = %d, want 10", result.CurrentLevel)
	}
	if result.TargetLevel != 11 {
		t.Errorf("TargetLevel = %d, want 11", result.TargetLevel)
	}
	if result.CostMetal <= 0 {
		t.Errorf("CostMetal should be positive, got %d", result.CostMetal)
	}
	if result.CostCrystal <= 0 {
		t.Errorf("CostCrystal should be positive, got %d", result.CostCrystal)
	}
	if result.ProductionIncrease <= 0 {
		t.Errorf("ProductionIncrease should be positive, got %f", result.ProductionIncrease)
	}
	if result.ROIScore <= 0 {
		t.Errorf("ROIScore should be positive, got %f", result.ROIScore)
	}
	if result.BuildTime <= 0 {
		t.Errorf("BuildTime should be positive, got %v", result.BuildTime)
	}
	if result.EnergyDelta > 0 {
		t.Errorf("Mine upgrade should consume energy (negative EnergyDelta), got %d", result.EnergyDelta)
	}
}

func TestCalculateROI_FusionReactor(t *testing.T) {
	// FusionReactor produces energy and consumes deuterium for its cost
	result, ok := CalculateROI(
		constants.BuildingFusionReactor,
		5,
		model.Planet{ID: 1, TemperatureMin: 15, TemperatureMax: 65},
		model.ResourceBuildings{FusionReactor: 5},
		model.Facilities{},
		model.Research{EnergyTechnology: 5},
		model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: 5000},
		1,
		20,
	)
	if !ok {
		t.Fatal("CalculateROI should return true for affordable FusionReactor upgrade")
	}
	// Fusion reactor produces energy
	if result.EnergyDelta <= 0 {
		t.Errorf("FusionReactor upgrade should produce energy, got EnergyDelta=%d", result.EnergyDelta)
	}
}

func TestCalculateROI_UnknownBuilding(t *testing.T) {
	// Unknown building ID should return false
	result, ok := CalculateROI(
		999, // non-existent building
		5,
		model.Planet{ID: 1},
		model.ResourceBuildings{},
		model.Facilities{},
		model.Research{},
		model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: 5000},
		1,
		30,
	)
	if ok {
		t.Error("CalculateROI should return false for unknown building ID")
	}
	_ = result
}

func TestCalculateROI_BuildTimeDecreasesWithRobotics(t *testing.T) {
	planet := model.Planet{ID: 1, TemperatureMin: 15, TemperatureMax: 65}
	resources := model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: 5000}
	research := model.Research{PlasmaTechnology: 5}

	result1, _ := CalculateROI(constants.BuildingMetalMine, 10, planet,
		model.ResourceBuildings{MetalMine: 10}, model.Facilities{RoboticsFactory: 0}, research, resources, 1, 30)
	result2, _ := CalculateROI(constants.BuildingMetalMine, 10, planet,
		model.ResourceBuildings{MetalMine: 10}, model.Facilities{RoboticsFactory: 10}, research, resources, 1, 30)

	if result2.BuildTime >= result1.BuildTime {
		t.Errorf("Higher robotics should reduce build time: robotics=0 %v >= robotics=10 %v", result1.BuildTime, result2.BuildTime)
	}
}

// --- Production increase helper test ---

func TestProductionIncrease_PositiveForAllMines(t *testing.T) {
	planet := model.Planet{ID: 1, TemperatureMin: 15, TemperatureMax: 65}
	research := model.Research{PlasmaTechnology: 5, EnergyTechnology: 5}
	facilities := model.Facilities{RoboticsFactory: 5}
	resources := model.Resources{Metal: 999999, Crystal: 999999, Deuterium: 999999, Energy: 5000}

	buildingIDs := []int{
		constants.BuildingMetalMine,
		constants.BuildingCrystalMine,
		constants.BuildingDeuteriumSynthesizer,
		constants.BuildingSolarPlant,
		constants.BuildingFusionReactor,
	}

	for _, buildingID := range buildingIDs {
		result, ok := CalculateROI(buildingID, 10, planet,
			model.ResourceBuildings{}, facilities, research, resources, 1, 30)
		if !ok {
			t.Errorf("CalculateROI for building %d should succeed", buildingID)
			continue
		}
		if result.ProductionIncrease <= 0 {
			t.Errorf("Building %d: ProductionIncrease = %f, want positive", buildingID, result.ProductionIncrease)
		}
		if result.ROIScore <= 0 {
			t.Errorf("Building %d: ROIScore = %f, want positive", buildingID, result.ROIScore)
		}
		if result.BuildTime <= 0 {
			t.Errorf("Building %d: BuildTime = %v, want positive", buildingID, result.BuildTime)
		}
	}
}

// Verify BuildTime is a proper time.Duration
func TestConstructionTime_ReturnsDuration(t *testing.T) {
	dur := ConstructionTime(60, 15, 0, 0, 1)
	if _, ok := interface{}(dur).(time.Duration); !ok {
		t.Errorf("ConstructionTime should return time.Duration, got %T", dur)
	}
}
