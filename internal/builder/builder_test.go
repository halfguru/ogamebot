package builder

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"testing"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamed"

	_ "github.com/user/ogame-bot/internal/state" // registers sqlite driver
)

// --- Mock implementations ---

// mockBuilderClient satisfies ogamed.ClientInterface for builder tests.
type mockBuilderClient struct {
	constructions map[int]model.Constructions // planetID → constructions
	constructErr  error
	buildErr      error
	buildCalled   bool
	lastPlanetID  int
	lastBuildingID int
}

func (m *mockBuilderClient) Login(_ context.Context) error                       { return nil }
func (m *mockBuilderClient) Logout(_ context.Context) error                      { return nil }
func (m *mockBuilderClient) GetServerTime(_ context.Context) (string, error)     { return "", nil }
func (m *mockBuilderClient) IsUnderAttack(_ context.Context) (bool, error)       { return false, nil }
func (m *mockBuilderClient) GetPlanets(_ context.Context) ([]model.Planet, error) { return nil, nil }
func (m *mockBuilderClient) GetResources(_ context.Context, _ int) (model.Resources, error) {
	return model.Resources{}, nil
}
func (m *mockBuilderClient) GetResourceBuildings(_ context.Context, _ int) (model.ResourceBuildings, error) {
	return model.ResourceBuildings{}, nil
}
func (m *mockBuilderClient) GetFacilities(_ context.Context, _ int) (model.Facilities, error) {
	return model.Facilities{}, nil
}
func (m *mockBuilderClient) GetShips(_ context.Context, _ int) (model.Ships, error) {
	return model.Ships{}, nil
}
func (m *mockBuilderClient) GetDefence(_ context.Context, _ int) (model.Defence, error) {
	return model.Defence{}, nil
}
func (m *mockBuilderClient) GetFleets(_ context.Context) ([]model.Fleet, error) {
	return nil, nil
}
func (m *mockBuilderClient) GetResearch(_ context.Context) (model.Research, error) {
	return model.Research{}, nil
}
func (m *mockBuilderClient) GetServerSpeed(_ context.Context) (int, error)     { return 1, nil }
func (m *mockBuilderClient) GetServerVersion(_ context.Context) (string, error) { return "", nil }
func (m *mockBuilderClient) GetAttacks(_ context.Context) ([]model.AttackEvent, error) {
	return nil, nil
}
func (m *mockBuilderClient) GetSlots(_ context.Context) (model.Slots, error) {
	return model.Slots{}, nil
}
func (m *mockBuilderClient) SendFleet(_ context.Context, _ model.SendFleetRequest) (int64, error) {
	return 0, nil
}
func (m *mockBuilderClient) CancelFleet(_ context.Context, _ int64) error { return nil }
func (m *mockBuilderClient) GetConstructions(_ context.Context, planetID int) (model.Constructions, error) {
	if m.constructErr != nil {
		return model.Constructions{}, m.constructErr
	}
	if m.constructions != nil {
		if c, ok := m.constructions[planetID]; ok {
			return c, nil
		}
	}
	return model.Constructions{}, nil
}
func (m *mockBuilderClient) BuildBuilding(_ context.Context, planetID, buildingID int) error {
	m.buildCalled = true
	m.lastPlanetID = planetID
	m.lastBuildingID = buildingID
	return m.buildErr
}
func (m *mockBuilderClient) GetGalaxyInfos(_ context.Context, _, _ int) (model.SystemInfos, error) {
	return model.SystemInfos{}, nil
}
func (m *mockBuilderClient) GetEspionageReportMessages(_ context.Context) ([]model.EspionageReportSummary, error) {
	return nil, nil
}
func (m *mockBuilderClient) GetEspionageReport(_ context.Context, _ int64) (model.EspionageReport, error) {
	return model.EspionageReport{}, nil
}
func (m *mockBuilderClient) DeleteAllEspionageReports(_ context.Context) error {
	return nil
}
func (m *mockBuilderClient) GetCaptchaChallenge(_ context.Context) (ogamed.CaptchaChallenge, error) {
	return ogamed.CaptchaChallenge{}, nil
}
func (m *mockBuilderClient) SolveCaptchaChallenge(_ context.Context, _ string, _ int) error {
	return nil
}

// mockBuilderStateReader satisfies BuilderStateReader for builder tests.
type mockBuilderStateReader struct {
	planets     []model.Planet
	planetsErr  error
	resources   map[int]model.Resources // planetID → resources
	resErr      error
	research    model.Research
	researchErr error
	buildings   map[int]model.ResourceBuildings // planetID → buildings
	buildErr    error
	facilities  map[int]model.Facilities // planetID → facilities
	facilErr    error
	speed       int
	speedErr    error
}

func (m *mockBuilderStateReader) GetPlanets(_ context.Context) ([]model.Planet, error) {
	return m.planets, m.planetsErr
}
func (m *mockBuilderStateReader) GetResources(_ context.Context, planetID int) (model.Resources, error) {
	if m.resErr != nil {
		return model.Resources{}, m.resErr
	}
	if m.resources != nil {
		if r, ok := m.resources[planetID]; ok {
			return r, nil
		}
	}
	return model.Resources{}, nil
}
func (m *mockBuilderStateReader) GetResearch(_ context.Context) (model.Research, error) {
	return m.research, m.researchErr
}
func (m *mockBuilderStateReader) GetBuildings(_ context.Context, planetID int) (model.ResourceBuildings, error) {
	if m.buildErr != nil {
		return model.ResourceBuildings{}, m.buildErr
	}
	if m.buildings != nil {
		if b, ok := m.buildings[planetID]; ok {
			return b, nil
		}
	}
	return model.ResourceBuildings{}, nil
}
func (m *mockBuilderStateReader) GetFacilities(_ context.Context, planetID int) (model.Facilities, error) {
	if m.facilErr != nil {
		return model.Facilities{}, m.facilErr
	}
	if m.facilities != nil {
		if f, ok := m.facilities[planetID]; ok {
			return f, nil
		}
	}
	return model.Facilities{}, nil
}
func (m *mockBuilderStateReader) GetServerSpeed(_ context.Context) (int, error) {
	return m.speed, m.speedErr
}

// --- Test helpers ---

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS planets (
		id INTEGER PRIMARY KEY, name TEXT NOT NULL, galaxy INTEGER NOT NULL,
		system INTEGER NOT NULL, position INTEGER NOT NULL, is_moon BOOLEAN NOT NULL DEFAULT FALSE,
		diameter INTEGER NOT NULL DEFAULT 0, fields_used INTEGER NOT NULL DEFAULT 0,
		fields_total INTEGER NOT NULL DEFAULT 0, temperature_min INTEGER NOT NULL DEFAULT 0,
		temperature_max INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL DEFAULT (datetime('now')))`)
	if err != nil {
		t.Fatalf("Failed to create planets table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS build_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		planet_id INTEGER NOT NULL REFERENCES planets(id),
		building_id INTEGER NOT NULL,
		building_name TEXT NOT NULL,
		from_level INTEGER NOT NULL,
		to_level INTEGER NOT NULL,
		cost_metal INTEGER NOT NULL DEFAULT 0,
		cost_crystal INTEGER NOT NULL DEFAULT 0,
		cost_deut INTEGER NOT NULL DEFAULT 0,
		roi_score REAL NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')))`)
	if err != nil {
		t.Fatalf("Failed to create build_events table: %v", err)
	}

	return db
}

func insertTestPlanet(t *testing.T, db *sql.DB, id int, name string, fieldsUsed, fieldsTotal int) {
	t.Helper()
	_, err := db.Exec(`INSERT OR REPLACE INTO planets (id, name, galaxy, system, position, fields_used, fields_total)
		VALUES (?, ?, 1, 1, ?, ?, ?)`, id, name, id, fieldsUsed, fieldsTotal)
	if err != nil {
		t.Fatalf("Failed to insert test planet: %v", err)
	}
}

func testBuilder(mc *mockBuilderClient, ms *mockBuilderStateReader, db *sql.DB, cfg config.AutoBuildConfig) *Builder {
	b := NewBuilder(mc, ms, db, cfg, testLogger())
	b.antiDetectPct = 0 // deterministic in tests
	return b
}

func defaultAutoBuildConfig() config.AutoBuildConfig {
	return config.AutoBuildConfig{
		FeatureConfig: config.FeatureConfig{
			Enabled:        true,
			PollIntervalMs: 30000,
		},
		MaxLevels: map[string]int{
			"MetalMine":            30,
			"CrystalMine":          28,
			"DeuteriumSynthesizer": 26,
			"SolarPlant":           26,
			"FusionReactor":        20,
		},
		PlanetOverrides: map[string]map[string]int{},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// --- Constructor Test ---

func TestBuilderConstructor(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{}
	log := testLogger()
	cfg := defaultAutoBuildConfig()

	b := NewBuilder(mc, ms, db, cfg, log)

	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}
}

// --- resolveMaxLevel Tests ---

func TestResolveMaxLevelGlobalDefault(t *testing.T) {
	cfg := defaultAutoBuildConfig()
	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{}
	db := newTestDB(t)
	defer db.Close()
	b := testBuilder(mc, ms, db, cfg)

	maxLvl := b.resolveMaxLevel("MetalMine", "Homeworld")
	if maxLvl != 30 {
		t.Errorf("expected global default 30 for MetalMine, got %d", maxLvl)
	}
}

func TestResolveMaxLevelPlanetOverride(t *testing.T) {
	cfg := defaultAutoBuildConfig()
	cfg.PlanetOverrides = map[string]map[string]int{
		"Homeworld": {"MetalMine": 35},
	}
	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{}
	db := newTestDB(t)
	defer db.Close()
	b := testBuilder(mc, ms, db, cfg)

	maxLvl := b.resolveMaxLevel("MetalMine", "Homeworld")
	if maxLvl != 35 {
		t.Errorf("expected planet override 35 for MetalMine on Homeworld, got %d", maxLvl)
	}
}

func TestResolveMaxLevelNoConfig(t *testing.T) {
	cfg := config.AutoBuildConfig{
		FeatureConfig: config.FeatureConfig{Enabled: true, PollIntervalMs: 30000},
		MaxLevels:     map[string]int{}, // empty
	}
	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{}
	db := newTestDB(t)
	defer db.Close()
	b := testBuilder(mc, ms, db, cfg)

	maxLvl := b.resolveMaxLevel("MetalMine", "Homeworld")
	if maxLvl != 0 {
		t.Errorf("expected 0 for missing max level config, got %d", maxLvl)
	}
}

// --- buildingLevel Tests ---

func TestBuildingLevel(t *testing.T) {
	buildings := model.ResourceBuildings{
		MetalMine:            15,
		CrystalMine:          12,
		DeuteriumSynthesizer: 10,
		SolarPlant:           18,
		FusionReactor:        5,
	}

	tests := []struct {
		buildingID int
		expected   int
	}{
		{constants.BuildingMetalMine, 15},
		{constants.BuildingCrystalMine, 12},
		{constants.BuildingDeuteriumSynthesizer, 10},
		{constants.BuildingSolarPlant, 18},
		{constants.BuildingFusionReactor, 5},
		{999, 0}, // unknown building
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("building_%d", tt.buildingID), func(t *testing.T) {
			got := buildingLevel(buildings, tt.buildingID)
			if got != tt.expected {
				t.Errorf("buildingLevel(%d) = %d, want %d", tt.buildingID, got, tt.expected)
			}
		})
	}
}

// --- Poll Tests ---

func TestPollPicksHighestROI(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)
	insertTestPlanet(t, db, 2, "Colony", 30, 200)

	// Planet 1: Metal Mine 15, Crystal Mine 12 — expensive upgrades
	// Planet 2: Metal Mine 1, Crystal Mine 1 — cheap upgrades, very high ROI
	// Expect: Planet 2's buildings dominate; Metal Mine 1→2 should win
	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
			{ID: 2, Name: "Colony", TemperatureMin: -20, TemperatureMax: 60, FieldsUsed: 30, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 500000, Crystal: 200000, Deuterium: 100000, Energy: 500},
			2: {Metal: 500000, Crystal: 200000, Deuterium: 100000, Energy: 500},
		},
		research: model.Research{PlasmaTechnology: 10, EnergyTechnology: 5},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 15, CrystalMine: 12, DeuteriumSynthesizer: 10, SolarPlant: 18, FusionReactor: 5},
			2: {MetalMine: 1, CrystalMine: 1, DeuteriumSynthesizer: 0, SolarPlant: 5, FusionReactor: 0},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 5, NaniteFactory: 0},
			2: {RoboticsFactory: 2, NaniteFactory: 0},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()
	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx)

	if !mc.buildCalled {
		t.Fatal("BuildBuilding should have been called")
	}
	// Planet 2 should win (lower level buildings have better ROI)
	if mc.lastPlanetID != 2 {
		t.Errorf("expected build on planet 2, got planet %d", mc.lastPlanetID)
	}
	// Metal Mine 1→2 should win (ROI ~0.35) over Crystal Mine 1→2 (ROI ~0.19)
	if mc.lastBuildingID != constants.BuildingMetalMine {
		t.Errorf("expected Metal Mine (1), got building %d", mc.lastBuildingID)
	}
}

func TestPollSkipsActiveConstruction(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{
		constructions: map[int]model.Constructions{
			1: {Building: model.Construction{ID: 1, Level: 16}}, // active construction!
		},
	}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 500000, Crystal: 200000, Deuterium: 100000, Energy: 500},
		},
		research: model.Research{},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 15, CrystalMine: 12},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 5},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()
	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx)

	if mc.buildCalled {
		t.Error("BuildBuilding should NOT have been called — planet is full")
	}
}

func TestPollRespectsMaxLevelCap(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 5000000, Crystal: 2000000, Deuterium: 1000000, Energy: 5000},
		},
		research: model.Research{PlasmaTechnology: 10, EnergyTechnology: 5},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 15, CrystalMine: 12, SolarPlant: 20}, // Crystal Mine is available, not at max
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 10, NaniteFactory: 2},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()
	cfg.MaxLevels["MetalMine"] = 15 // Metal Mine already at max!

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx)

	if !mc.buildCalled {
		t.Fatal("BuildBuilding should have been called — other buildings available")
	}
	if mc.lastBuildingID == constants.BuildingMetalMine {
		t.Error("Should not have upgraded Metal Mine — it's at max level")
	}
}

func TestPollRespectsPerPlanetOverride(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 5000000, Crystal: 2000000, Deuterium: 1000000, Energy: 5000},
		},
		research: model.Research{PlasmaTechnology: 10, EnergyTechnology: 5},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 15, CrystalMine: 12, DeuteriumSynthesizer: 10, SolarPlant: 20},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 10, NaniteFactory: 2},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()
	// Override: MetalMine capped at 15 on Homeworld
	cfg.PlanetOverrides = map[string]map[string]int{
		"Homeworld": {"MetalMine": 15},
	}

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx)

	if !mc.buildCalled {
		t.Fatal("BuildBuilding should have been called — other buildings available")
	}
	if mc.lastBuildingID == constants.BuildingMetalMine {
		t.Error("Should not have upgraded Metal Mine — planet override caps it at 15")
	}
}

func TestPollDisabledNoBuild(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 500000, Crystal: 200000, Energy: 500},
		},
		research: model.Research{},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 15, CrystalMine: 12},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 5},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()
	cfg.Enabled = false

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx)

	if mc.buildCalled {
		t.Error("BuildBuilding should NOT have been called — builder is disabled")
	}
}

func TestPollRecordsBuildEvent(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 500000, Crystal: 200000, Deuterium: 100000, Energy: 500},
		},
		research: model.Research{PlasmaTechnology: 5, EnergyTechnology: 3},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 1, CrystalMine: 10, SolarPlant: 15},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 2},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx)

	if !mc.buildCalled {
		t.Fatal("BuildBuilding should have been called")
	}

	// Verify build event was recorded in database
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM build_events WHERE planet_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query build_events: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 build event, got %d", count)
	}

	// Verify fields — Metal Mine 1→2 should have the best ROI
	var buildingName string
	var fromLevel, toLevel int
	var roiScore float64
	err = db.QueryRow("SELECT building_name, from_level, to_level, roi_score FROM build_events WHERE planet_id = 1").Scan(
		&buildingName, &fromLevel, &toLevel, &roiScore)
	if err != nil {
		t.Fatalf("Failed to query build event details: %v", err)
	}
	if buildingName != "Metal Mine" {
		t.Errorf("expected Metal Mine, got %s", buildingName)
	}
	if fromLevel != 1 || toLevel != 2 {
		t.Errorf("expected from_level=1, to_level=2, got %d→%d", fromLevel, toLevel)
	}
	if roiScore <= 0 {
		t.Errorf("expected positive ROI score, got %f", roiScore)
	}
}

func TestPollHandlesConstructionsAPIError(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{
		constructErr: fmt.Errorf("API error"),
	}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 500000, Crystal: 200000, Energy: 500},
		},
		research: model.Research{},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 10},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 5},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	// Should not panic, should log and continue
	b.poll(ctx)

	if mc.buildCalled {
		t.Error("BuildBuilding should NOT have been called — GetConstructions errored")
	}
}

func TestPollHandlesBuildBuildingAPIError(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{
		buildErr: fmt.Errorf("build API error"),
	}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 500000, Crystal: 200000, Deuterium: 100000, Energy: 500},
		},
		research: model.Research{},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 5, CrystalMine: 3},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 2},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	// Should not panic — should log the error and continue
	b.poll(ctx)

	if !mc.buildCalled {
		t.Error("BuildBuilding should have been called (attempt made)")
	}
}

func TestPollEmptyPlanetList(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockBuilderClient{}
	ms := &mockBuilderStateReader{
		planets: []model.Planet{},
		speed:   1,
	}
	cfg := defaultAutoBuildConfig()

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx) // Should not panic

	if mc.buildCalled {
		t.Error("BuildBuilding should NOT have been called with no planets")
	}
}

// --- Sorting verification test ---

func TestPollCandidatesSortedByROI(t *testing.T) {
	// This test verifies that candidates are sorted by ROI score descending
	// by checking the building that was actually built
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld", 50, 200)

	mc := &mockBuilderClient{}
	// Metal Mine level 1 → 2 (extremely cheap, high ROI)
	// Crystal Mine level 10 → 11 (more expensive, lower ROI)
	ms := &mockBuilderStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", TemperatureMin: 0, TemperatureMax: 100, FieldsUsed: 50, FieldsTotal: 200},
		},
		resources: map[int]model.Resources{
			1: {Metal: 500000, Crystal: 200000, Deuterium: 100000, Energy: 1000},
		},
		research: model.Research{PlasmaTechnology: 10, EnergyTechnology: 5},
		buildings: map[int]model.ResourceBuildings{
			1: {MetalMine: 1, CrystalMine: 10, SolarPlant: 15},
		},
		facilities: map[int]model.Facilities{
			1: {RoboticsFactory: 5},
		},
		speed: 1,
	}
	cfg := defaultAutoBuildConfig()

	b := testBuilder(mc, ms, db, cfg)

	ctx := context.Background()
	b.poll(ctx)

	if !mc.buildCalled {
		t.Fatal("BuildBuilding should have been called")
	}
	// Metal Mine 1→2 should win (cheaper, higher ROI)
	if mc.lastBuildingID != constants.BuildingMetalMine {
		t.Errorf("expected Metal Mine (highest ROI), got building %d", mc.lastBuildingID)
	}
}

// Helper to verify sorted candidates (used internally)
func sortCandidates(candidates []ROIResult) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ROIScore > candidates[j].ROIScore
	})
}
