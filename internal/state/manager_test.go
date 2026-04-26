package state

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamed"
)

// mockClient implements ogamed.ClientInterface for testing.
// Each function field can be set per-test; unset fields return zero values.
type mockClient struct {
	loginFunc              func(ctx context.Context) error
	logoutFunc             func(ctx context.Context) error
	getServerTimeFunc      func(ctx context.Context) (string, error)
	isUnderAttackFunc      func(ctx context.Context) (bool, error)
	getPlanetsFunc         func(ctx context.Context) ([]model.Planet, error)
	getResourcesFunc       func(ctx context.Context, planetID int) (model.Resources, error)
	getResourceBuildingsFunc func(ctx context.Context, planetID int) (model.ResourceBuildings, error)
	getFacilitiesFunc      func(ctx context.Context, planetID int) (model.Facilities, error)
	getShipsFunc           func(ctx context.Context, planetID int) (model.Ships, error)
	getDefenceFunc         func(ctx context.Context, planetID int) (model.Defence, error)
	getFleetsFunc          func(ctx context.Context) ([]model.Fleet, error)
	getResearchFunc        func(ctx context.Context) (model.Research, error)
	getServerSpeedFunc     func(ctx context.Context) (int, error)
	getServerVersionFunc   func(ctx context.Context) (string, error)
	getAttacksFunc         func(ctx context.Context) ([]model.AttackEvent, error)
	getSlotsFunc           func(ctx context.Context) (model.Slots, error)
	sendFleetFunc          func(ctx context.Context, req model.SendFleetRequest) (int64, error)
	cancelFleetFunc        func(ctx context.Context, fleetID int64) error
}

func (m *mockClient) Login(ctx context.Context) error {
	if m.loginFunc != nil {
		return m.loginFunc(ctx)
	}
	return nil
}
func (m *mockClient) Logout(ctx context.Context) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx)
	}
	return nil
}
func (m *mockClient) GetServerTime(ctx context.Context) (string, error) {
	if m.getServerTimeFunc != nil {
		return m.getServerTimeFunc(ctx)
	}
	return "", nil
}
func (m *mockClient) IsUnderAttack(ctx context.Context) (bool, error) {
	if m.isUnderAttackFunc != nil {
		return m.isUnderAttackFunc(ctx)
	}
	return false, nil
}
func (m *mockClient) GetPlanets(ctx context.Context) ([]model.Planet, error) {
	if m.getPlanetsFunc != nil {
		return m.getPlanetsFunc(ctx)
	}
	return nil, nil
}
func (m *mockClient) GetResources(ctx context.Context, planetID int) (model.Resources, error) {
	if m.getResourcesFunc != nil {
		return m.getResourcesFunc(ctx, planetID)
	}
	return model.Resources{}, nil
}
func (m *mockClient) GetResourceBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error) {
	if m.getResourceBuildingsFunc != nil {
		return m.getResourceBuildingsFunc(ctx, planetID)
	}
	return model.ResourceBuildings{}, nil
}
func (m *mockClient) GetFacilities(ctx context.Context, planetID int) (model.Facilities, error) {
	if m.getFacilitiesFunc != nil {
		return m.getFacilitiesFunc(ctx, planetID)
	}
	return model.Facilities{}, nil
}
func (m *mockClient) GetShips(ctx context.Context, planetID int) (model.Ships, error) {
	if m.getShipsFunc != nil {
		return m.getShipsFunc(ctx, planetID)
	}
	return model.Ships{}, nil
}
func (m *mockClient) GetDefence(ctx context.Context, planetID int) (model.Defence, error) {
	if m.getDefenceFunc != nil {
		return m.getDefenceFunc(ctx, planetID)
	}
	return model.Defence{}, nil
}
func (m *mockClient) GetFleets(ctx context.Context) ([]model.Fleet, error) {
	if m.getFleetsFunc != nil {
		return m.getFleetsFunc(ctx)
	}
	return nil, nil
}
func (m *mockClient) GetResearch(ctx context.Context) (model.Research, error) {
	if m.getResearchFunc != nil {
		return m.getResearchFunc(ctx)
	}
	return model.Research{}, nil
}
func (m *mockClient) GetServerSpeed(ctx context.Context) (int, error) {
	if m.getServerSpeedFunc != nil {
		return m.getServerSpeedFunc(ctx)
	}
	return 0, nil
}
func (m *mockClient) GetServerVersion(ctx context.Context) (string, error) {
	if m.getServerVersionFunc != nil {
		return m.getServerVersionFunc(ctx)
	}
	return "", nil
}
func (m *mockClient) GetAttacks(ctx context.Context) ([]model.AttackEvent, error) {
	if m.getAttacksFunc != nil {
		return m.getAttacksFunc(ctx)
	}
	return nil, nil
}
func (m *mockClient) GetSlots(ctx context.Context) (model.Slots, error) {
	if m.getSlotsFunc != nil {
		return m.getSlotsFunc(ctx)
	}
	return model.Slots{}, nil
}
func (m *mockClient) SendFleet(ctx context.Context, req model.SendFleetRequest) (int64, error) {
	if m.sendFleetFunc != nil {
		return m.sendFleetFunc(ctx, req)
	}
	return 0, nil
}
func (m *mockClient) CancelFleet(ctx context.Context, fleetID int64) error {
	if m.cancelFleetFunc != nil {
		return m.cancelFleetFunc(ctx, fleetID)
	}
	return nil
}

// setupTestDB creates a fresh SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	db, err := OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("setupTestDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestManager_RefreshAll(t *testing.T) {
	db := setupTestDB(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mc := &mockClient{
		getPlanetsFunc: func(ctx context.Context) ([]model.Planet, error) {
			return []model.Planet{
				{
					ID:             100,
					Name:           "Homeworld",
					Coordinate:     model.Coordinate{Galaxy: 1, System: 2, Position: 3, Type: "planet"},
					Diameter:       12800,
					FieldsUsed:     80,
					FieldsTotal:    163,
					TemperatureMin: 15,
					TemperatureMax: 65,
					IsMoon:         false,
				},
				{
					ID:             200,
					Name:           "Colony",
					Coordinate:     model.Coordinate{Galaxy: 1, System: 50, Position: 8, Type: "planet"},
					Diameter:       6400,
					FieldsUsed:     42,
					FieldsTotal:    100,
					TemperatureMin: -10,
					TemperatureMax: 30,
					IsMoon:         false,
				},
			}, nil
		},
		getResourcesFunc: func(ctx context.Context, planetID int) (model.Resources, error) {
			switch planetID {
			case 100:
				return model.Resources{Metal: 10000, Crystal: 5000, Deuterium: 2000, Energy: 150}, nil
			case 200:
				return model.Resources{Metal: 3000, Crystal: 1500, Deuterium: 800, Energy: 50}, nil
			default:
				return model.Resources{}, fmt.Errorf("unknown planet %d", planetID)
			}
		},
		getResourceBuildingsFunc: func(ctx context.Context, planetID int) (model.ResourceBuildings, error) {
			switch planetID {
			case 100:
				return model.ResourceBuildings{MetalMine: 20, CrystalMine: 18, DeuteriumSynthesizer: 15, SolarPlant: 22}, nil
			case 200:
				return model.ResourceBuildings{MetalMine: 10, CrystalMine: 8, DeuteriumSynthesizer: 6, SolarPlant: 12}, nil
			default:
				return model.ResourceBuildings{}, nil
			}
		},
		getFacilitiesFunc: func(ctx context.Context, planetID int) (model.Facilities, error) {
			switch planetID {
			case 100:
				return model.Facilities{RoboticsFactory: 8, Shipyard: 10, ResearchLab: 6}, nil
			case 200:
				return model.Facilities{RoboticsFactory: 3, Shipyard: 5, ResearchLab: 2}, nil
			default:
				return model.Facilities{}, nil
			}
		},
		getFleetsFunc: func(ctx context.Context) ([]model.Fleet, error) {
			return []model.Fleet{
				{
					ID:           500,
					Mission:      3,
					ReturnFlight: false,
					Origin:       model.Coordinate{Galaxy: 1, System: 2, Position: 3},
					Destination:  model.Coordinate{Galaxy: 1, System: 100, Position: 5},
					ArrivalTime:  1700000000,
					Metal:        100,
					Crystal:      50,
				},
			}, nil
		},
		getResearchFunc: func(ctx context.Context) (model.Research, error) {
			return model.Research{
				EnergyTechnology:   5,
				CombustionDrive:    8,
				ComputerTechnology: 10,
			}, nil
		},
	}

	mgr := NewManager(db, mc, log)
	err := mgr.refresh(context.Background())
	if err != nil {
		t.Fatalf("refresh() error = %v", err)
	}

	// Verify planets
	var planetCount int
	db.QueryRow("SELECT COUNT(*) FROM planets").Scan(&planetCount)
	if planetCount != 2 {
		t.Errorf("planets count: expected 2, got %d", planetCount)
	}

	// Verify planet 100
	var name string
	db.QueryRow("SELECT name FROM planets WHERE id = 100").Scan(&name)
	if name != "Homeworld" {
		t.Errorf("planet 100 name: expected 'Homeworld', got %q", name)
	}

	// Verify resources
	var metal int
	db.QueryRow("SELECT metal FROM resources WHERE planet_id = 100").Scan(&metal)
	if metal != 10000 {
		t.Errorf("planet 100 metal: expected 10000, got %d", metal)
	}

	// Verify buildings
	var metalMine int
	db.QueryRow("SELECT metal_mine FROM buildings WHERE planet_id = 100").Scan(&metalMine)
	if metalMine != 20 {
		t.Errorf("planet 100 metal_mine: expected 20, got %d", metalMine)
	}

	// Verify facilities
	var robotics int
	db.QueryRow("SELECT robotics_factory FROM facilities WHERE planet_id = 100").Scan(&robotics)
	if robotics != 8 {
		t.Errorf("planet 100 robotics_factory: expected 8, got %d", robotics)
	}

	// Verify fleets
	var fleetCount int
	db.QueryRow("SELECT COUNT(*) FROM fleets").Scan(&fleetCount)
	if fleetCount != 1 {
		t.Errorf("fleets count: expected 1, got %d", fleetCount)
	}

	var mission int
	db.QueryRow("SELECT mission FROM fleets WHERE id = 500").Scan(&mission)
	if mission != 3 {
		t.Errorf("fleet 500 mission: expected 3, got %d", mission)
	}

	// Verify research
	var energyTech int
	db.QueryRow("SELECT energy_technology FROM research WHERE id = 1").Scan(&energyTech)
	if energyTech != 5 {
		t.Errorf("research energy_technology: expected 5, got %d", energyTech)
	}
}

func TestManager_GetPlanets(t *testing.T) {
	db := setupTestDB(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Insert test planets directly
	db.Exec(`INSERT INTO planets (id, name, galaxy, system, position, is_moon, diameter, fields_used, fields_total, temperature_min, temperature_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		100, "Homeworld", 1, 2, 3, false, 12800, 80, 163, 15, 65)
	db.Exec(`INSERT INTO planets (id, name, galaxy, system, position, is_moon, diameter, fields_used, fields_total, temperature_min, temperature_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		101, "Moon", 1, 2, 3, true, 5000, 10, 30, -20, 10)

	mc := &mockClient{}
	mgr := NewManager(db, mc, log)

	planets, err := mgr.GetPlanets(context.Background())
	if err != nil {
		t.Fatalf("GetPlanets() error = %v", err)
	}

	if len(planets) != 2 {
		t.Fatalf("expected 2 planets, got %d", len(planets))
	}

	// Find Homeworld
	var hw *model.Planet
	for i := range planets {
		if planets[i].ID == 100 {
			hw = &planets[i]
			break
		}
	}
	if hw == nil {
		t.Fatal("planet 100 not found")
	}

	if hw.Name != "Homeworld" {
		t.Errorf("name: expected 'Homeworld', got %q", hw.Name)
	}
	if hw.Coordinate.Galaxy != 1 || hw.Coordinate.System != 2 || hw.Coordinate.Position != 3 {
		t.Errorf("coordinate: expected (1,2,3), got (%d,%d,%d)", hw.Coordinate.Galaxy, hw.Coordinate.System, hw.Coordinate.Position)
	}
	if hw.IsMoon {
		t.Errorf("is_moon: expected false, got true")
	}
	if hw.Diameter != 12800 {
		t.Errorf("diameter: expected 12800, got %d", hw.Diameter)
	}

	// Find Moon
	var moon *model.Planet
	for i := range planets {
		if planets[i].ID == 101 {
			moon = &planets[i]
			break
		}
	}
	if moon == nil {
		t.Fatal("planet 101 not found")
	}
	if !moon.IsMoon {
		t.Errorf("moon is_moon: expected true, got false")
	}
}

func TestManager_Refresh_ContinuesOnPlanetError(t *testing.T) {
	db := setupTestDB(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mc := &mockClient{
		getPlanetsFunc: func(ctx context.Context) ([]model.Planet, error) {
			return []model.Planet{
				{ID: 100, Name: "Good", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1}},
				{ID: 200, Name: "Bad", Coordinate: model.Coordinate{Galaxy: 2, System: 2, Position: 2}},
			}, nil
		},
		getResourcesFunc: func(ctx context.Context, planetID int) (model.Resources, error) {
			if planetID == 200 {
				return model.Resources{}, fmt.Errorf("ogamed timeout")
			}
			return model.Resources{Metal: 9999, Crystal: 5555, Deuterium: 3333, Energy: 100}, nil
		},
		getResourceBuildingsFunc: func(ctx context.Context, planetID int) (model.ResourceBuildings, error) {
			if planetID == 200 {
				return model.ResourceBuildings{}, fmt.Errorf("ogamed timeout")
			}
			return model.ResourceBuildings{MetalMine: 15}, nil
		},
		getFacilitiesFunc: func(ctx context.Context, planetID int) (model.Facilities, error) {
			return model.Facilities{}, nil
		},
		getFleetsFunc: func(ctx context.Context) ([]model.Fleet, error) {
			return nil, nil
		},
		getResearchFunc: func(ctx context.Context) (model.Research, error) {
			return model.Research{}, nil
		},
	}

	mgr := NewManager(db, mc, log)
	err := mgr.refresh(context.Background())
	if err != nil {
		t.Fatalf("refresh() should not return error when per-planet fetch fails: %v", err)
	}

	// Planet 100 resources should be cached
	var metal int
	db.QueryRow("SELECT metal FROM resources WHERE planet_id = 100").Scan(&metal)
	if metal != 9999 {
		t.Errorf("planet 100 metal: expected 9999, got %d", metal)
	}

	// Planet 200 resources should NOT be in DB (fetch failed)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM resources WHERE planet_id = 200").Scan(&count)
	if count != 0 {
		t.Errorf("planet 200 resources: expected 0 rows (fetch failed), got %d", count)
	}
}

func TestManager_Run_PeriodicRefresh(t *testing.T) {
	db := setupTestDB(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var refreshCount atomic.Int32

	mc := &mockClient{
		getPlanetsFunc: func(ctx context.Context) ([]model.Planet, error) {
			refreshCount.Add(1)
			return nil, nil
		},
		getFleetsFunc: func(ctx context.Context) ([]model.Fleet, error) {
			return nil, nil
		},
		getResearchFunc: func(ctx context.Context) (model.Research, error) {
			return model.Research{}, nil
		},
	}

	mgr := NewManager(db, mc, log)
	mgr.SetInterval(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.Run(ctx)

	// Wait for at least initial + 2 ticks
	time.Sleep(350 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	count := refreshCount.Load()
	if count < 2 {
		t.Errorf("expected at least 2 refreshes in 350ms with 100ms interval, got %d", count)
	}
}

func TestManager_Run_StopsOnCancel(t *testing.T) {
	db := setupTestDB(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mc := &mockClient{
		getPlanetsFunc: func(ctx context.Context) ([]model.Planet, error) {
			return nil, nil
		},
		getFleetsFunc: func(ctx context.Context) ([]model.Fleet, error) {
			return nil, nil
		},
		getResearchFunc: func(ctx context.Context) (model.Research, error) {
			return model.Research{}, nil
		},
	}

	mgr := NewManager(db, mc, log)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	done := make(chan struct{})
	go func() {
		mgr.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Good — manager stopped
	case <-time.After(1 * time.Second):
		t.Fatal("manager did not stop after context cancellation")
	}
}

func TestManager_GetResearch(t *testing.T) {
	db := setupTestDB(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Insert research directly
	db.Exec(`INSERT INTO research (id, energy_technology, combustion_drive, computer_technology)
		VALUES (1, 5, 8, 10)`)

	mc := &mockClient{}
	mgr := NewManager(db, mc, log)

	research, err := mgr.GetResearch(context.Background())
	if err != nil {
		t.Fatalf("GetResearch() error = %v", err)
	}

	if research.EnergyTechnology != 5 {
		t.Errorf("energy_technology: expected 5, got %d", research.EnergyTechnology)
	}
	if research.CombustionDrive != 8 {
		t.Errorf("combustion_drive: expected 8, got %d", research.CombustionDrive)
	}
	if research.ComputerTechnology != 10 {
		t.Errorf("computer_technology: expected 10, got %d", research.ComputerTechnology)
	}
}

func TestManager_GetFleets(t *testing.T) {
	db := setupTestDB(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Insert a fleet directly
	db.Exec(`INSERT INTO fleets (id, mission, return_flight, origin_galaxy, origin_system, origin_position,
		dest_galaxy, dest_system, dest_position, metal, crystal, deuterium, arrival_time)
		VALUES (500, 3, 0, 1, 2, 3, 1, 100, 5, 100, 50, 0, 1700000000)`)

	mc := &mockClient{}
	mgr := NewManager(db, mc, log)

	fleets, err := mgr.GetFleets(context.Background())
	if err != nil {
		t.Fatalf("GetFleets() error = %v", err)
	}

	if len(fleets) != 1 {
		t.Fatalf("expected 1 fleet, got %d", len(fleets))
	}

	f := fleets[0]
	if f.ID != 500 {
		t.Errorf("fleet ID: expected 500, got %d", f.ID)
	}
	if f.Mission != 3 {
		t.Errorf("fleet mission: expected 3, got %d", f.Mission)
	}
	if f.ReturnFlight {
		t.Errorf("return_flight: expected false, got true")
	}
	if f.Origin.Galaxy != 1 || f.Origin.System != 2 || f.Origin.Position != 3 {
		t.Errorf("origin: expected (1,2,3), got (%d,%d,%d)", f.Origin.Galaxy, f.Origin.System, f.Origin.Position)
	}
	if f.Destination.Galaxy != 1 || f.Destination.System != 100 || f.Destination.Position != 5 {
		t.Errorf("destination: expected (1,100,5), got (%d,%d,%d)", f.Destination.Galaxy, f.Destination.System, f.Destination.Position)
	}
	if f.Metal != 100 || f.Crystal != 50 {
		t.Errorf("resources: expected metal=100 crystal=50, got metal=%d crystal=%d", f.Metal, f.Crystal)
	}
}

// Verify interface compliance
var _ ogamed.ClientInterface = (*mockClient)(nil)
