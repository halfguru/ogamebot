package defender

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/state"
)

// --- Mock implementations ---

// mockClient satisfies ogamed.ClientInterface for defender tests.
type mockClient struct {
	attacks     []model.AttackEvent
	attacksErr  error
	slots       model.Slots
	slotsErr    error
	sendResult  int64
	sendErr     error
	cancelErr   error
	fleets      []model.Fleet
	fleetsErr   error
	ships       model.Ships
	shipsErr    error
	serverTime  string
	serverErr   error
	// Track calls
	sendCalled bool
	lastReq    model.SendFleetRequest
	cancelled  []int64
}

func (m *mockClient) Login(_ context.Context) error                          { return nil }
func (m *mockClient) Logout(_ context.Context) error                         { return nil }
func (m *mockClient) GetServerTime(_ context.Context) (string, error)        { return m.serverTime, m.serverErr }
func (m *mockClient) IsUnderAttack(_ context.Context) (bool, error)          { return false, nil }
func (m *mockClient) GetPlanets(_ context.Context) ([]model.Planet, error)   { return nil, nil }
func (m *mockClient) GetResources(_ context.Context, _ int) (model.Resources, error) {
	return model.Resources{}, nil
}
func (m *mockClient) GetResourceBuildings(_ context.Context, _ int) (model.ResourceBuildings, error) {
	return model.ResourceBuildings{}, nil
}
func (m *mockClient) GetFacilities(_ context.Context, _ int) (model.Facilities, error) {
	return model.Facilities{}, nil
}
func (m *mockClient) GetShips(_ context.Context, _ int) (model.Ships, error) {
	return m.ships, m.shipsErr
}
func (m *mockClient) GetDefence(_ context.Context, _ int) (model.Defence, error) {
	return model.Defence{}, nil
}
func (m *mockClient) GetFleets(_ context.Context) ([]model.Fleet, error) {
	return m.fleets, m.fleetsErr
}
func (m *mockClient) GetResearch(_ context.Context) (model.Research, error) {
	return model.Research{}, nil
}
func (m *mockClient) GetServerSpeed(_ context.Context) (int, error)     { return 1, nil }
func (m *mockClient) GetServerVersion(_ context.Context) (string, error) { return "", nil }
func (m *mockClient) GetAttacks(_ context.Context) ([]model.AttackEvent, error) {
	return m.attacks, m.attacksErr
}
func (m *mockClient) GetSlots(_ context.Context) (model.Slots, error) {
	return m.slots, m.slotsErr
}
func (m *mockClient) SendFleet(_ context.Context, req model.SendFleetRequest) (int64, error) {
	m.sendCalled = true
	m.lastReq = req
	return m.sendResult, m.sendErr
}
func (m *mockClient) CancelFleet(_ context.Context, fleetID int64) error {
	m.cancelled = append(m.cancelled, fleetID)
	return m.cancelErr
}

// mockStateReader satisfies StateReader for defender tests.
type mockStateReader struct {
	planets   []model.Planet
	planetsErr error
	resources model.Resources
	resErr     error
	research  model.Research
	resErr2    error
	fleets    []model.Fleet
	fleetsErr error
}

func (m *mockStateReader) GetPlanets(_ context.Context) ([]model.Planet, error) {
	return m.planets, m.planetsErr
}
func (m *mockStateReader) GetResources(_ context.Context, _ int) (model.Resources, error) {
	return m.resources, m.resErr
}
func (m *mockStateReader) GetFleets(_ context.Context) ([]model.Fleet, error) {
	return m.fleets, m.fleetsErr
}
func (m *mockStateReader) GetResearch(_ context.Context) (model.Research, error) {
	return m.research, m.resErr2
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

	// Create planets table (referenced by fleet_save_events)
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

	// Create fleet_save_events table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS fleet_save_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		planet_id INTEGER NOT NULL REFERENCES planets(id),
		fleet_id INTEGER NOT NULL,
		dest_planet_id INTEGER NOT NULL,
		attack_id INTEGER NOT NULL DEFAULT 0,
		sent_at DATETIME NOT NULL DEFAULT (datetime('now')),
		recall_at DATETIME,
		completed BOOLEAN NOT NULL DEFAULT FALSE,
		recalled BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')))`)
	if err != nil {
		t.Fatalf("Failed to create fleet_save_events table: %v", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_fleet_save_planet_active ON fleet_save_events(planet_id, completed) WHERE completed = FALSE`)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	return db
}

func insertTestPlanet(t *testing.T, db *sql.DB, id int, name string) {
	t.Helper()
	_, err := db.Exec(`INSERT OR REPLACE INTO planets (id, name, galaxy, system, position) VALUES (?, ?, 1, 1, ?)`,
		id, name, id)
	if err != nil {
		t.Fatalf("Failed to insert test planet: %v", err)
	}
}

func defaultDefenderConfig() config.DefenderConfig {
	return config.DefenderConfig{
		FeatureConfig: config.FeatureConfig{
			Enabled:        true,
			PollIntervalMs: 5000,
		},
		SafetyMarginMs:    120000,
		RecallEnabled:     boolPtr(true),
		MaxReturnFlightS:  600,
		MinReactionDelayS: 30,
		MaxReactionDelayS: 120,
	}
}

func boolPtr(b bool) *bool { return &b }

// --- Fleet-Save Tracking Tests ---

func TestDefenderConstructor(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)

	if d == nil {
		t.Fatal("NewDefender returned nil")
	}
}

func TestActiveFleetSaveReturnsNilWhenNone(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)

	ctx := context.Background()
	event, err := d.activeFleetSave(ctx, 1)
	if err != nil {
		t.Fatalf("activeFleetSave returned error: %v", err)
	}
	if event != nil {
		t.Fatalf("expected nil, got event: %+v", event)
	}
}

func TestRecordFleetSave(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	err := d.recordFleetSave(ctx, 1, 100, 2, 50, time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("recordFleetSave returned error: %v", err)
	}
}

func TestActiveFleetSaveReturnsSaveWhenExists(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	recallAt := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second)
	err := d.recordFleetSave(ctx, 1, 100, 2, 50, recallAt)
	if err != nil {
		t.Fatalf("recordFleetSave returned error: %v", err)
	}

	event, err := d.activeFleetSave(ctx, 1)
	if err != nil {
		t.Fatalf("activeFleetSave returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.PlanetID != 1 {
		t.Errorf("expected PlanetID=1, got %d", event.PlanetID)
	}
	if event.FleetID != 100 {
		t.Errorf("expected FleetID=100, got %d", event.FleetID)
	}
	if event.DestPlanetID != 2 {
		t.Errorf("expected DestPlanetID=2, got %d", event.DestPlanetID)
	}
	if event.AttackID != 50 {
		t.Errorf("expected AttackID=50, got %d", event.AttackID)
	}
	if event.Completed {
		t.Error("expected Completed=false")
	}
}

func TestCompleteFleetSave(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	err := d.recordFleetSave(ctx, 1, 100, 2, 50, time.Time{})
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	err = d.completeFleetSave(ctx, 100)
	if err != nil {
		t.Fatalf("completeFleetSave error: %v", err)
	}

	// After completing, activeFleetSave should return nil
	event, err := d.activeFleetSave(ctx, 1)
	if err != nil {
		t.Fatalf("activeFleetSave error: %v", err)
	}
	if event != nil {
		t.Fatalf("expected nil after complete, got: %+v", event)
	}
}

func TestPendingRecalls(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	// Record a fleet-save with recall_at in the past (should be pending)
	pastTime := time.Now().Add(-1 * time.Minute).UTC().Truncate(time.Second)
	err := d.recordFleetSave(ctx, 1, 100, 2, 50, pastTime)
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	// Record another with recall_at in the future (should NOT be pending)
	futureTime := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second)
	err = d.recordFleetSave(ctx, 2, 101, 1, 51, futureTime)
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	// Record one with no recall (zero time) — should NOT be pending
	err = d.recordFleetSave(ctx, 1, 102, 2, 52, time.Time{})
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	pending, err := d.pendingRecalls(ctx)
	if err != nil {
		t.Fatalf("pendingRecalls error: %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending recall, got %d", len(pending))
	}
	if pending[0].FleetID != 100 {
		t.Errorf("expected FleetID=100, got %d", pending[0].FleetID)
	}
}

func TestMarkRecalled(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	pastTime := time.Now().Add(-1 * time.Minute).UTC().Truncate(time.Second)
	err := d.recordFleetSave(ctx, 1, 100, 2, 50, pastTime)
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	err = d.markRecalled(ctx, 100)
	if err != nil {
		t.Fatalf("markRecalled error: %v", err)
	}

	// After marking recalled, it should no longer appear in pendingRecalls
	pending, err := d.pendingRecalls(ctx)
	if err != nil {
		t.Fatalf("pendingRecalls error: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending recalls after marking recalled, got %d", len(pending))
	}
}

func TestMigration002CreatesTable(t *testing.T) {
	// Test that the migration file is valid SQL by running state.OpenDB
	// which applies all embedded migrations including 002_fleet_save.sql
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	db, err := state.OpenDB(dbPath, log)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	// Verify fleet_save_events table exists
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='fleet_save_events'").Scan(&name)
	if err != nil {
		t.Fatalf("fleet_save_events table not found: %v", err)
	}
	if name != "fleet_save_events" {
		t.Errorf("expected 'fleet_save_events', got %q", name)
	}

	// Verify index exists
	var idxName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_fleet_save_planet_active'").Scan(&idxName)
	if err != nil {
		t.Fatalf("index not found: %v", err)
	}
}

// --- Attack Handling Tests ---

func TestHandleAttacksIdentifiesEndangeredPlanets(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockClient{
		serverTime: time.Now().UTC().Format("2006-01-02 15:04:05"),
	}
	ms := &mockStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}},
			{ID: 2, Name: "Colony", Coordinate: model.Coordinate{Galaxy: 1, System: 2, Position: 3, Type: "planet"}},
		},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)

	// Insert planets into DB
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	attacks := []model.AttackEvent{
		{
			ID:          1,
			MissionType: 1, // Attack
			Destination: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"},
			ArrivalTime: time.Now().Add(30 * time.Minute).UTC(),
			ArriveIn:    1800,
		},
	}

	endangered := d.identifyEndangered(attacks, time.Now().UTC())
	if len(endangered) != 1 {
		t.Fatalf("expected 1 endangered planet, got %d", len(endangered))
	}
	if endangered[0].planet.ID != 1 {
		t.Errorf("expected planet ID=1, got %d", endangered[0].planet.ID)
	}
}

func TestHandleAttacksSkipsEspionage(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockClient{}
	ms := &mockStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}},
		},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)

	// Espionage only — should be skipped
	attacks := []model.AttackEvent{
		{
			ID:          1,
			MissionType: 6, // Espionage
			Destination: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"},
			ArrivalTime: time.Now().Add(10 * time.Minute).UTC(),
			ArriveIn:    600,
		},
	}

	endangered := d.identifyEndangered(attacks, time.Now().UTC())
	if len(endangered) != 0 {
		t.Fatalf("expected 0 endangered (espionage should be skipped), got %d", len(endangered))
	}
}

func TestHandleAttacksSkipsTooClose(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	mc := &mockClient{}
	ms := &mockStateReader{
		planets: []model.Planet{
			{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}},
		},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()
	cfg.SafetyMarginMs = 120000   // 2 minutes
	cfg.MinReactionDelayS = 30    // 30 seconds

	d := NewDefender(mc, ms, db, cfg, log)

	// Attack arriving in 60s — too close (need safetyMargin + minReactionDelay = 150s)
	attacks := []model.AttackEvent{
		{
			ID:          1,
			MissionType: 1, // Attack
			Destination: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"},
			ArrivalTime: time.Now().Add(60 * time.Second).UTC(),
			ArriveIn:    60,
		},
	}

	endangered := d.identifyEndangered(attacks, time.Now().UTC())
	if len(endangered) != 0 {
		t.Fatalf("expected 0 endangered (attack too close), got %d", len(endangered))
	}
}

func fastDefenderConfig() config.DefenderConfig {
	return config.DefenderConfig{
		FeatureConfig: config.FeatureConfig{
			Enabled:        true,
			PollIntervalMs: 100,
		},
		SafetyMarginMs:    2000, // 2 seconds — fast for tests
		RecallEnabled:     boolPtr(true),
		MaxReturnFlightS:  600,
		MinReactionDelayS: 1,
		MaxReactionDelayS: 2,
	}
}

func TestSavePlanetNoViableRoutes(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		slots:      model.Slots{InUse: 0, Total: 10},
		ships:      model.Ships{LightFighter: 10},
		sendResult: 999,
	}
	ms := &mockStateReader{
		planets:   []model.Planet{}, // No destinations — no viable routes
		resources: model.Resources{Metal: 1000, Crystal: 500, Deuterium: 200},
		research:  model.Research{},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := fastDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	planet := model.Planet{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}}
	attacks := []model.AttackEvent{
		{ID: 1, MissionType: 1, ArrivalTime: time.Now().Add(30 * time.Minute).UTC()},
	}

	// Should not panic, should not send fleet
	d.savePlanet(ctx, planet, attacks, 10*time.Second)

	if mc.sendCalled {
		t.Error("SendFleet should NOT have been called with no viable routes")
	}
}

func TestSavePlanetFleetSlotsFull(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		slots: model.Slots{InUse: 10, Total: 10}, // Full!
	}
	ms := &mockStateReader{
		planets: []model.Planet{
			{ID: 2, Name: "Colony", Coordinate: model.Coordinate{Galaxy: 1, System: 2, Position: 3, Type: "planet"}},
		},
		resources: model.Resources{Metal: 1000, Crystal: 500, Deuterium: 200},
		research:  model.Research{},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := fastDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	planet := model.Planet{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}}
	attacks := []model.AttackEvent{
		{ID: 1, MissionType: 1, ArrivalTime: time.Now().Add(30 * time.Minute).UTC()},
	}

	d.savePlanet(ctx, planet, attacks, 10*time.Second)

	if mc.sendCalled {
		t.Error("SendFleet should NOT have been called when slots are full")
	}
}

func TestSavePlanetDoesNotResave(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		slots:      model.Slots{InUse: 0, Total: 10},
		ships:      model.Ships{LightFighter: 10},
		sendResult: 100,
	}
	ms := &mockStateReader{
		planets: []model.Planet{
			{ID: 2, Name: "Colony", Coordinate: model.Coordinate{Galaxy: 1, System: 2, Position: 3, Type: "planet"}},
		},
		resources: model.Resources{Metal: 1000, Crystal: 500, Deuterium: 200},
		research:  model.Research{CombustionDrive: 10},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := fastDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	// Record an active fleet-save for planet 1
	err := d.recordFleetSave(ctx, 1, 50, 2, 1, time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	planet := model.Planet{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}}
	attacks := []model.AttackEvent{
		{ID: 2, MissionType: 1, ArrivalTime: time.Now().Add(30 * time.Minute).UTC()},
	}

	// Should skip because active fleet-save exists
	d.savePlanet(ctx, planet, attacks, 10*time.Second)

	if mc.sendCalled {
		t.Error("SendFleet should NOT have been called — planet already has active save")
	}
}

func TestSavePlanetNoShips(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		slots: model.Slots{InUse: 0, Total: 10},
		ships: model.Ships{}, // No ships!
	}
	ms := &mockStateReader{
		planets: []model.Planet{
			{ID: 2, Name: "Colony", Coordinate: model.Coordinate{Galaxy: 1, System: 2, Position: 3, Type: "planet"}},
		},
		resources: model.Resources{Metal: 1000, Crystal: 500, Deuterium: 200},
		research:  model.Research{},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := fastDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	planet := model.Planet{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}}
	attacks := []model.AttackEvent{
		{ID: 1, MissionType: 1, ArrivalTime: time.Now().Add(30 * time.Minute).UTC()},
	}

	d.savePlanet(ctx, planet, attacks, 10*time.Second)

	if mc.sendCalled {
		t.Error("SendFleet should NOT have been called with no ships")
	}
}

func TestSavePlanetSuccessful(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		slots:      model.Slots{InUse: 0, Total: 10},
		ships:      model.Ships{LightFighter: 10, SmallCargo: 5},
		sendResult: 200,
	}
	ms := &mockStateReader{
		planets: []model.Planet{
			{ID: 2, Name: "Colony", Coordinate: model.Coordinate{Galaxy: 1, System: 2, Position: 3, Type: "planet"}},
		},
		resources: model.Resources{Metal: 1000, Crystal: 500, Deuterium: 50000},
		research:  model.Research{CombustionDrive: 10},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := fastDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	planet := model.Planet{ID: 1, Name: "Homeworld", Coordinate: model.Coordinate{Galaxy: 1, System: 1, Position: 1, Type: "planet"}}
	attacks := []model.AttackEvent{
		{ID: 10, MissionType: 1, ArrivalTime: time.Now().Add(30 * time.Minute).UTC()},
	}

	d.savePlanet(ctx, planet, attacks, 10*time.Second)

	if !mc.sendCalled {
		t.Fatal("SendFleet should have been called")
	}

	// Verify the fleet-save was recorded
	event, err := d.activeFleetSave(ctx, 1)
	if err != nil {
		t.Fatalf("activeFleetSave error: %v", err)
	}
	if event == nil {
		t.Fatal("expected fleet-save event to be recorded")
	}
	if event.FleetID != 200 {
		t.Errorf("expected FleetID=200, got %d", event.FleetID)
	}

	// Verify SendFleetRequest fields
	if mc.lastReq.PlanetID != 1 {
		t.Errorf("expected PlanetID=1, got %d", mc.lastReq.PlanetID)
	}
	if mc.lastReq.Mission != 3 { // MissionDeploy
		t.Errorf("expected Mission=3 (Deploy), got %d", mc.lastReq.Mission)
	}
	if mc.lastReq.Galaxy != 1 || mc.lastReq.System != 2 || mc.lastReq.Position != 3 {
		t.Errorf("unexpected destination: %+v", mc.lastReq)
	}
}

// --- Recall Tests ---

func TestProcessRecalls(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		fleets: []model.Fleet{
			{ID: 100, ReturnFlight: false},
		},
	}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	// Record a fleet-save with recall_at in the past
	pastTime := time.Now().Add(-1 * time.Minute).UTC()
	err := d.recordFleetSave(ctx, 1, 100, 2, 50, pastTime)
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	d.processRecalls(ctx)

	if len(mc.cancelled) != 1 {
		t.Fatalf("expected 1 CancelFleet call, got %d", len(mc.cancelled))
	}
	if mc.cancelled[0] != 100 {
		t.Errorf("expected CancelFleet(100), got CancelFleet(%d)", mc.cancelled[0])
	}
}

func TestProcessRecallsSkipsAlreadyReturning(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		fleets: []model.Fleet{
			{ID: 100, ReturnFlight: true}, // Already returning
		},
	}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	pastTime := time.Now().Add(-1 * time.Minute).UTC()
	err := d.recordFleetSave(ctx, 1, 100, 2, 50, pastTime)
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	d.processRecalls(ctx)

	if len(mc.cancelled) != 0 {
		t.Fatalf("expected 0 CancelFleet calls (already returning), got %d", len(mc.cancelled))
	}
}

func TestProcessRecallsHandlesMissingFleet(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{
		fleets: []model.Fleet{}, // Fleet gone
	}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	pastTime := time.Now().Add(-1 * time.Minute).UTC()
	err := d.recordFleetSave(ctx, 1, 100, 2, 50, pastTime)
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	d.processRecalls(ctx)

	// Missing fleet should be completed (not cancelled)
	if len(mc.cancelled) != 0 {
		t.Fatalf("expected 0 CancelFleet calls, got %d", len(mc.cancelled))
	}

	// Fleet-save should now be completed
	event, err := d.activeFleetSave(ctx, 1)
	if err != nil {
		t.Fatalf("activeFleetSave error: %v", err)
	}
	if event != nil {
		t.Error("expected nil — fleet-save should be completed for missing fleet")
	}
}

func TestProcessRecallsDisabled(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	insertTestPlanet(t, db, 1, "Homeworld")
	insertTestPlanet(t, db, 2, "Colony")

	mc := &mockClient{}
	ms := &mockStateReader{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := defaultDefenderConfig()
	cfg.RecallEnabled = boolPtr(false)

	d := NewDefender(mc, ms, db, cfg, log)
	ctx := context.Background()

	pastTime := time.Now().Add(-1 * time.Minute).UTC()
	err := d.recordFleetSave(ctx, 1, 100, 2, 50, pastTime)
	if err != nil {
		t.Fatalf("recordFleetSave error: %v", err)
	}

	d.processRecalls(ctx)

	if len(mc.cancelled) != 0 {
		t.Fatalf("expected 0 CancelFleet calls (recall disabled), got %d", len(mc.cancelled))
	}
}

// --- Reaction Delay Tests ---

func TestCalcReactionDelayCappedBySafetyMargin(t *testing.T) {
	cfg := defaultDefenderConfig()
	cfg.MinReactionDelayS = 30
	cfg.MaxReactionDelayS = 120
	cfg.SafetyMarginMs = 120000 // 2 minutes

	// timeUntilAttack = 3 minutes, safety margin = 2 minutes
	// maxAllowedDelay = 3min - 2min = 1min
	// So delay should be in [30s, 60s]
	delay := calcReactionDelay(3*time.Minute, cfg)

	if delay < 30*time.Second {
		t.Errorf("delay %v is less than minReactionDelay 30s", delay)
	}
	if delay > 60*time.Second {
		t.Errorf("delay %v exceeds maxAllowedDelay 60s (3min - 2min safety)", delay)
	}
}

func TestCalcReactionDelayTooClose(t *testing.T) {
	cfg := defaultDefenderConfig()
	cfg.MinReactionDelayS = 30
	cfg.MaxReactionDelayS = 120
	cfg.SafetyMarginMs = 120000

	// timeUntilAttack = 60s, safety margin = 120s
	// maxAllowedDelay = 60s - 120s = negative → too late
	delay := calcReactionDelay(60*time.Second, cfg)

	if delay != 0 {
		t.Errorf("expected 0 delay (too close), got %v", delay)
	}
}
