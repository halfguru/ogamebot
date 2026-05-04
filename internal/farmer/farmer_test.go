package farmer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamex"

	_ "github.com/user/ogame-bot/internal/state" // registers sqlite driver
)

// --- Mock implementations ---

// mockFarmerClient satisfies ogamex.ClientInterface for farmer tests.
type mockFarmerClient struct {
	slots           model.Slots
	slotsErr        error
	galaxyInfos     map[string]model.SystemInfos // "galaxy:system" → result
	galaxyErr       error
	spyMessages     []model.EspionageReportSummary
	spyMessagesErr  error
	spyReports      map[int64]model.EspionageReport // msgID → report
	spyReportErr    error
	deleteReportsErr error
	shipsOnPlanet   map[int]model.Ships // planetID → ships
	shipsErr        error
	sendFleetID     int64
	sendFleetErr    error
	sendFleetCalled int
}

func (m *mockFarmerClient) Login(_ context.Context) error                       { return nil }
func (m *mockFarmerClient) Logout(_ context.Context) error                      { return nil }
func (m *mockFarmerClient) GetServerTime(_ context.Context) (string, error)     { return "", nil }
func (m *mockFarmerClient) IsUnderAttack(_ context.Context) (bool, error)       { return false, nil }
func (m *mockFarmerClient) GetPlanets(_ context.Context) ([]model.Planet, error) { return nil, nil }
func (m *mockFarmerClient) GetResources(_ context.Context, _ int) (model.Resources, model.PlanetDetails, error) {
	return model.Resources{}, model.PlanetDetails{}, nil
}
func (m *mockFarmerClient) GetResourceBuildings(_ context.Context, _ int) (model.ResourceBuildings, error) {
	return model.ResourceBuildings{}, nil
}
func (m *mockFarmerClient) GetFacilities(_ context.Context, _ int) (model.Facilities, error) {
	return model.Facilities{}, nil
}
func (m *mockFarmerClient) GetShips(_ context.Context, planetID int) (model.Ships, error) {
	if m.shipsErr != nil {
		return model.Ships{}, m.shipsErr
	}
	if m.shipsOnPlanet != nil {
		if s, ok := m.shipsOnPlanet[planetID]; ok {
			return s, nil
		}
	}
	return model.Ships{}, nil
}
func (m *mockFarmerClient) GetDefence(_ context.Context, _ int) (model.Defence, error) {
	return model.Defence{}, nil
}
func (m *mockFarmerClient) GetFleets(_ context.Context) ([]model.Fleet, error) {
	return nil, nil
}
func (m *mockFarmerClient) GetResearch(_ context.Context) (model.Research, error) {
	return model.Research{}, nil
}
func (m *mockFarmerClient) GetServerSpeed(_ context.Context) (int, error)     { return 1, nil }
func (m *mockFarmerClient) GetServerVersion(_ context.Context) (string, error) { return "", nil }
func (m *mockFarmerClient) GetAttacks(_ context.Context) ([]model.AttackEvent, error) {
	return nil, nil
}
func (m *mockFarmerClient) GetSlots(_ context.Context) (model.Slots, error) {
	return m.slots, m.slotsErr
}
func (m *mockFarmerClient) SendFleet(_ context.Context, _ model.SendFleetRequest) (int64, error) {
	m.sendFleetCalled++
	return m.sendFleetID, m.sendFleetErr
}
func (m *mockFarmerClient) CancelFleet(_ context.Context, _ int64) error { return nil }
func (m *mockFarmerClient) GetConstructions(_ context.Context, _ int) (model.Constructions, error) {
	return model.Constructions{}, nil
}
func (m *mockFarmerClient) BuildBuilding(_ context.Context, _, _ int) error { return nil }
func (m *mockFarmerClient) BuildResearch(_ context.Context, _, _ int) error { return nil }
func (m *mockFarmerClient) GetGalaxyInfos(_ context.Context, galaxy, system int) (model.SystemInfos, error) {
	if m.galaxyErr != nil {
		return model.SystemInfos{}, m.galaxyErr
	}
	key := fmt.Sprintf("%d:%d", galaxy, system)
	if m.galaxyInfos != nil {
		if info, ok := m.galaxyInfos[key]; ok {
			return info, nil
		}
	}
	return model.SystemInfos{}, nil
}
func (m *mockFarmerClient) GetEspionageReportMessages(_ context.Context) ([]model.EspionageReportSummary, error) {
	return m.spyMessages, m.spyMessagesErr
}
func (m *mockFarmerClient) GetEspionageReport(_ context.Context, messageID int64) (model.EspionageReport, error) {
	if m.spyReportErr != nil {
		return model.EspionageReport{}, m.spyReportErr
	}
	if m.spyReports != nil {
		if r, ok := m.spyReports[messageID]; ok {
			return r, nil
		}
	}
	return model.EspionageReport{}, nil
}
func (m *mockFarmerClient) DeleteAllEspionageReports(_ context.Context) error {
	return m.deleteReportsErr
}

var _ ogamex.ClientInterface = (*mockFarmerClient)(nil)

// mockFarmerStateReader satisfies FarmerStateReader for farmer tests.
type mockFarmerStateReader struct {
	planets    []model.Planet
	planetsErr error
	research   model.Research
	researchErr error
}

func (m *mockFarmerStateReader) GetPlanets(_ context.Context) ([]model.Planet, error) {
	return m.planets, m.planetsErr
}
func (m *mockFarmerStateReader) GetResearch(_ context.Context) (model.Research, error) {
	return m.research, m.researchErr
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

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS farm_targets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		galaxy INTEGER NOT NULL, system INTEGER NOT NULL, position INTEGER NOT NULL,
		player_id INTEGER NOT NULL DEFAULT 0, player_name TEXT NOT NULL DEFAULT '',
		is_inactive BOOLEAN NOT NULL DEFAULT FALSE, is_long_inactive BOOLEAN NOT NULL DEFAULT FALSE,
		last_scanned_at DATETIME, last_espionage_at DATETIME, last_attack_at DATETIME,
		metal_loot INTEGER NOT NULL DEFAULT 0, crystal_loot INTEGER NOT NULL DEFAULT 0,
		deuterium_loot INTEGER NOT NULL DEFAULT 0, has_defense BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		UNIQUE(galaxy, system, position))`)
	if err != nil {
		t.Fatalf("Failed to create farm_targets table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS farm_attacks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		fleet_id INTEGER NOT NULL, planet_id INTEGER NOT NULL,
		target_galaxy INTEGER NOT NULL, target_system INTEGER NOT NULL, target_position INTEGER NOT NULL,
		ships_sent INTEGER NOT NULL DEFAULT 0,
		metal_looted INTEGER NOT NULL DEFAULT 0, crystal_looted INTEGER NOT NULL DEFAULT 0,
		deuterium_looted INTEGER NOT NULL DEFAULT 0,
		sent_at DATETIME NOT NULL DEFAULT (datetime('now')),
		returned_at DATETIME,
		UNIQUE(fleet_id))`)
	if err != nil {
		t.Fatalf("Failed to create farm_attacks table: %v", err)
	}

	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func defaultAutoFarmConfig() config.AutoFarmConfig {
	return config.AutoFarmConfig{
		FeatureConfig: config.FeatureConfig{
			Enabled:        true,
			PollIntervalMs: 60000,
		},
		GalaxyRanges: []model.GalaxyRange{
			{Galaxy: 1, SystemStart: 100, SystemEnd: 102},
		},
		MinProfitThreshold: 10000,
		MaxProbesPerTarget:  5,
		MaxAttacksPerCycle:  3,
		SkipDefended:        true,
	}
}

func testFarmer(mc *mockFarmerClient, ms *mockFarmerStateReader, db *sql.DB, cfg config.AutoFarmConfig) *Farmer {
	return NewFarmer(mc, ms, db, cfg, testLogger())
}

// --- Constructor Test ---

func TestNewFarmer(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockFarmerClient{}
	ms := &mockFarmerStateReader{}
	cfg := defaultAutoFarmConfig()

	f := NewFarmer(mc, ms, db, cfg, testLogger())

	if f == nil {
		t.Fatal("NewFarmer returned nil")
	}
}

// --- isInactiveTarget Tests ---

func TestIsInactiveTarget_ValidInactive(t *testing.T) {
	pos := model.PlanetPosition{
		Name:     "Planet X",
		Inactive: true,
		Vacation: false,
		Banned:   false,
	}
	if !isInactiveTarget(pos) {
		t.Error("expected true for inactive, non-vacation, non-banned planet with name")
	}
}

func TestIsInactiveTarget_VacationPlayer(t *testing.T) {
	pos := model.PlanetPosition{
		Name:     "Planet X",
		Inactive: true,
		Vacation: true,
		Banned:   false,
	}
	if isInactiveTarget(pos) {
		t.Error("expected false for vacation player")
	}
}

func TestIsInactiveTarget_EmptySlot(t *testing.T) {
	pos := model.PlanetPosition{
		Name:     "",
		Inactive: true,
		Vacation: false,
		Banned:   false,
	}
	if isInactiveTarget(pos) {
		t.Error("expected false for empty slot (no name)")
	}
}

func TestIsInactiveTarget_BannedPlayer(t *testing.T) {
	pos := model.PlanetPosition{
		Name:     "Planet X",
		Inactive: true,
		Vacation: false,
		Banned:   true,
	}
	if isInactiveTarget(pos) {
		t.Error("expected false for banned player")
	}
}

func TestIsInactiveTarget_ActivePlayer(t *testing.T) {
	pos := model.PlanetPosition{
		Name:     "Planet X",
		Inactive: false,
		Vacation: false,
		Banned:   false,
	}
	if isInactiveTarget(pos) {
		t.Error("expected false for active player")
	}
}

// --- calcLootValue Tests ---

func TestCalcLootValue(t *testing.T) {
	tests := []struct {
		name                           string
		metal, crystal, deuterium      int64
		expected                       int64
	}{
		{"zero resources", 0, 0, 0, 0},
		{"metal only", 1000, 0, 0, 1000},
		{"crystal only", 0, 1000, 0, 1500},           // 1000 * 1.5
		{"deuterium only", 0, 0, 1000, 2000},          // 1000 * 2.0
		{"mixed", 10000, 4000, 2000, 20000},           // 10000 + 4000*1.5 + 2000*2.0 = 10000+6000+4000
		{"typical farm", 50000, 30000, 10000, 115000}, // 50000 + 30000*1.5 + 10000*2.0 = 50000+45000+20000
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcLootValue(tt.metal, tt.crystal, tt.deuterium)
			if got != tt.expected {
				t.Errorf("calcLootValue(%d, %d, %d) = %d, want %d",
					tt.metal, tt.crystal, tt.deuterium, got, tt.expected)
			}
		})
	}
}

// --- hasDefense Tests ---

func TestHasDefense_WithDefense(t *testing.T) {
	tests := []struct {
		name   string
		report model.EspionageReport
	}{
		{"rocket launcher", model.EspionageReport{RocketLauncher: 5}},
		{"light laser", model.EspionageReport{LightLaser: 1}},
		{"heavy laser", model.EspionageReport{HeavyLaser: 10}},
		{"gauss cannon", model.EspionageReport{GaussCannon: 2}},
		{"ion cannon", model.EspionageReport{IonCannon: 3}},
		{"plasma turret", model.EspionageReport{PlasmaTurret: 1}},
		{"small shield", model.EspionageReport{SmallShieldDome: 1}},
		{"large shield", model.EspionageReport{LargeShieldDome: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !hasDefense(tt.report) {
				t.Errorf("expected true for report with %s", tt.name)
			}
		})
	}
}

func TestHasDefense_NoDefense(t *testing.T) {
	report := model.EspionageReport{} // all zeros
	if hasDefense(report) {
		t.Error("expected false for report with no defenses")
	}
}

// --- pickClosestPlanet Tests ---

func TestPickClosestPlanet(t *testing.T) {
	planets := []model.Planet{
		{ID: 1, Name: "Far", Coordinate: model.Coordinate{Galaxy: 1, System: 100, Position: 1}},
		{ID: 2, Name: "Close", Coordinate: model.Coordinate{Galaxy: 1, System: 150, Position: 5}},
		{ID: 3, Name: "Mid", Coordinate: model.Coordinate{Galaxy: 1, System: 200, Position: 3}},
	}
	target := model.Coordinate{Galaxy: 1, System: 155, Position: 10}

	closest := pickClosestPlanet(planets, target)
	if closest.ID != 2 {
		t.Errorf("expected planet 2 (Close), got planet %d (%s)", closest.ID, closest.Name)
	}
}

func TestPickClosestPlanet_EmptyList(t *testing.T) {
	planets := []model.Planet{}
	target := model.Coordinate{Galaxy: 1, System: 100, Position: 1}

	result := pickClosestPlanet(planets, target)
	if result.ID != 0 {
		t.Errorf("expected zero-value planet for empty list, got ID %d", result.ID)
	}
}

func TestPickClosestPlanet_SinglePlanet(t *testing.T) {
	planets := []model.Planet{
		{ID: 1, Name: "Only", Coordinate: model.Coordinate{Galaxy: 1, System: 100, Position: 1}},
	}
	target := model.Coordinate{Galaxy: 2, System: 200, Position: 10}

	result := pickClosestPlanet(planets, target)
	if result.ID != 1 {
		t.Errorf("expected planet 1, got planet %d", result.ID)
	}
}

// --- cargoNeeded Tests ---

func TestCargoNeeded_SmallCargo(t *testing.T) {
	tests := []struct {
		name      string
		totalLoot int64
		expected  int64
	}{
		{"zero", 0, 0},
		{"less than capacity", 3000, 1},
		{"exactly capacity", 5000, 1},
		{"slightly over", 5001, 2},
		{"large loot", 25000, 5},
		{"uneven division", 12000, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cargoNeeded(tt.totalLoot, false)
			if got != tt.expected {
				t.Errorf("cargoNeeded(%d, false) = %d, want %d", tt.totalLoot, got, tt.expected)
			}
		})
	}
}

func TestCargoNeeded_LargeCargo(t *testing.T) {
	tests := []struct {
		name      string
		totalLoot int64
		expected  int64
	}{
		{"zero", 0, 0},
		{"less than capacity", 10000, 1},
		{"exactly capacity", 25000, 1},
		{"large loot", 100000, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cargoNeeded(tt.totalLoot, true)
			if got != tt.expected {
				t.Errorf("cargoNeeded(%d, true) = %d, want %d", tt.totalLoot, got, tt.expected)
			}
		})
	}
}

// --- estimateFuelCost Tests ---

func TestEstimateFuelCost(t *testing.T) {
	tests := []struct {
		name        string
		distance    int
		speed       int
		cargoCount  int64
		research    model.Research
		expected    int64
	}{
		{
			name:       "zero distance",
			distance:   0,
			speed:      10,
			cargoCount: 10,
			research:   model.Research{},
			expected:   0,
		},
		{
			name:       "short distance with combustion",
			distance:   100,
			speed:      10,
			cargoCount: 10,
			research:   model.Research{CombustionDrive: 10},
			// baseFuel(10) * 10 * (100/35000) * ((10/10+1)/2)^2 = 100*0.002857*1.0 = 0.2857 → rounds to 0
			expected: 0,
		},
		{
			name:       "long distance with many ships",
			distance:   20000,
			speed:      10,
			cargoCount: 100,
			research:   model.Research{CombustionDrive: 10},
			// 10 * 100 * (20000/35000) * ((10/10+1)/2)^2 = 1000 * 0.5714 * 1.0 = ~571
			expected: 571,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateFuelCost(tt.distance, tt.speed, tt.cargoCount, tt.research)
			// Allow some tolerance for floating point rounding
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > 2 {
				t.Errorf("estimateFuelCost(%d, %d, %d, ...) = %d, want approximately %d (diff=%d)",
					tt.distance, tt.speed, tt.cargoCount, got, tt.expected, diff)
			}
		})
	}
}

func TestEstimateFuelCost_ImpulseDrive(t *testing.T) {
	// When impulse drive >= 5, small cargo switches to impulse drive
	research := model.Research{ImpulseDrive: 8, CombustionDrive: 10}
	got := estimateFuelCost(10000, 10, 50, research)
	// Should use impulse drive level for calculation but baseFuel is still 10
	// The drive level doesn't affect fuel cost directly in our simplified formula
	// since we use baseFuel=10 regardless of drive type for small cargo
	if got <= 0 {
		t.Errorf("expected positive fuel cost, got %d", got)
	}
}

// --- evaluateReport Tests ---

func TestEvaluateReport_ViableTarget(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockFarmerClient{}
	ms := &mockFarmerStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 150, Position: 5}},
		},
		research: model.Research{CombustionDrive: 10},
	}
	cfg := defaultAutoFarmConfig()
	cfg.MinProfitThreshold = 1000

	f := testFarmer(mc, ms, db, cfg)

	report := model.EspionageReport{
		ID:         100,
		Metal:      100000,
		Crystal:    50000,
		Deuterium:  20000,
		Coordinate: model.Coordinate{Galaxy: 1, System: 155, Position: 3},
		// No defense
	}

	ownPlanets := ms.planets
	research := ms.research

	target, viable := f.evaluateReport(report, ownPlanets, research)
	if !viable {
		t.Fatal("expected report to be viable")
	}
	if target.Coordinate.Galaxy != 1 || target.Coordinate.System != 155 {
		t.Errorf("unexpected coordinate: %v", target.Coordinate)
	}
	// Metal loot should be 50% of reported
	if target.MetalLoot != 50000 {
		t.Errorf("expected MetalLoot=50000, got %d", target.MetalLoot)
	}
	if target.CrystalLoot != 25000 {
		t.Errorf("expected CrystalLoot=25000, got %d", target.CrystalLoot)
	}
	if target.DeuteriumLoot != 10000 {
		t.Errorf("expected DeuteriumLoot=10000, got %d", target.DeuteriumLoot)
	}
	if target.TotalValue <= 0 {
		t.Errorf("expected positive TotalValue, got %d", target.TotalValue)
	}
	if target.NetProfit <= 0 {
		t.Errorf("expected positive NetProfit, got %d", target.NetProfit)
	}
	if target.HasDefense {
		t.Error("expected HasDefense=false")
	}
}

func TestEvaluateReport_SkipDefended(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockFarmerClient{}
	ms := &mockFarmerStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 150, Position: 5}},
		},
		research: model.Research{CombustionDrive: 10},
	}
	cfg := defaultAutoFarmConfig()
	cfg.SkipDefended = true

	f := testFarmer(mc, ms, db, cfg)

	report := model.EspionageReport{
		ID:              100,
		Metal:           100000,
		Crystal:         50000,
		Deuterium:       20000,
		Coordinate:      model.Coordinate{Galaxy: 1, System: 155, Position: 3},
		RocketLauncher:  10, // has defense
	}

	_, viable := f.evaluateReport(report, ms.planets, ms.research)
	if viable {
		t.Error("expected defended target to be non-viable when SkipDefended=true")
	}
}

func TestEvaluateReport_DefendedButNotSkipping(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockFarmerClient{}
	ms := &mockFarmerStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 150, Position: 5}},
		},
		research: model.Research{CombustionDrive: 10},
	}
	cfg := defaultAutoFarmConfig()
	cfg.SkipDefended = false // don't skip defended
	cfg.MinProfitThreshold = 1000

	f := testFarmer(mc, ms, db, cfg)

	report := model.EspionageReport{
		ID:             100,
		Metal:          500000,
		Crystal:        200000,
		Deuterium:      100000,
		Coordinate:     model.Coordinate{Galaxy: 1, System: 155, Position: 3},
		RocketLauncher: 10, // has defense but we're not skipping
	}

	target, viable := f.evaluateReport(report, ms.planets, ms.research)
	if !viable {
		t.Error("expected defended target to be viable when SkipDefended=false")
	}
	if !target.HasDefense {
		t.Error("expected HasDefense=true on the target")
	}
}

func TestEvaluateReport_BelowThreshold(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockFarmerClient{}
	ms := &mockFarmerStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 150, Position: 5}},
		},
		research: model.Research{CombustionDrive: 10},
	}
	cfg := defaultAutoFarmConfig()
	cfg.MinProfitThreshold = 1000000 // very high threshold

	f := testFarmer(mc, ms, db, cfg)

	report := model.EspionageReport{
		ID:         100,
		Metal:      1000,
		Crystal:    500,
		Deuterium:  200,
		Coordinate: model.Coordinate{Galaxy: 1, System: 155, Position: 3},
	}

	_, viable := f.evaluateReport(report, ms.planets, ms.research)
	if viable {
		t.Error("expected target below threshold to be non-viable")
	}
}

// --- Galaxy Scanning Integration Tests ---

func TestScanGalaxies(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockFarmerClient{
		galaxyInfos: map[string]model.SystemInfos{
			"1:100": {
				Galaxy: 1, System: 100,
				Planets: []model.PlanetPosition{
					{Name: "Inactive1", Inactive: true, Coordinate: model.Coordinate{Galaxy: 1, System: 100, Position: 3}},
					{Name: "Active1", Inactive: false, Coordinate: model.Coordinate{Galaxy: 1, System: 100, Position: 5}},
				},
			},
			"1:101": {
				Galaxy: 1, System: 101,
				Planets: []model.PlanetPosition{
					{Name: "Inactive2", Inactive: true, Coordinate: model.Coordinate{Galaxy: 1, System: 101, Position: 7}},
				},
			},
		},
	}
	ms := &mockFarmerStateReader{}
	cfg := defaultAutoFarmConfig()

	f := testFarmer(mc, ms, db, cfg)

	inactives, err := f.scanGalaxies(context.Background())
	if err != nil {
		t.Fatalf("scanGalaxies error: %v", err)
	}
	if len(inactives) != 2 {
		t.Fatalf("expected 2 inactives, got %d", len(inactives))
	}
}

func TestScanGalaxies_EmptyRanges(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockFarmerClient{}
	ms := &mockFarmerStateReader{}
	cfg := defaultAutoFarmConfig()
	cfg.GalaxyRanges = nil

	f := testFarmer(mc, ms, db, cfg)

	inactives, err := f.scanGalaxies(context.Background())
	if err != nil {
		t.Fatalf("scanGalaxies error: %v", err)
	}
	if len(inactives) != 0 {
		t.Errorf("expected 0 inactives with no ranges, got %d", len(inactives))
	}
}
