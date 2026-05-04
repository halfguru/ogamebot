package defender

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamex"
)

// StateReader provides read access to cached game state.
// state.Manager implicitly satisfies this interface.
type StateReader interface {
	GetPlanets(ctx context.Context) ([]model.Planet, error)
	GetResources(ctx context.Context, planetID int) (model.Resources, model.PlanetDetails, error)
	GetFleets(ctx context.Context) ([]model.Fleet, error)
	GetResearch(ctx context.Context) (model.Research, error)
	RefreshNow(ctx context.Context) error
}

// fleetSaveEvent represents a fleet-save tracking record in the database.
type fleetSaveEvent struct {
	ID           int64
	PlanetID     int64
	FleetID      int64
	DestPlanetID int64
	AttackID     int64
	SentAt       time.Time
	RecallAt     *time.Time
	Completed    bool
	Recalled     bool
	CreatedAt    time.Time
}

// endangeredPlanet holds a planet that needs fleet-saving and the attacks targeting it.
type endangeredPlanet struct {
	planet          model.Planet
	attacks         []model.AttackEvent
	timeUntilAttack time.Duration
}

// Defender orchestrates fleet-save operations: polls for attacks, saves endangered fleets, recalls after danger passes.
type Defender struct {
	client      ogamex.ClientInterface
	stateMgr    StateReader
	db          *sql.DB
	cfg         config.DefenderConfig
	log         *slog.Logger
	broadcaster Broadcaster
}

type Broadcaster interface {
	Broadcast(msgType string, data interface{})
}

// NewDefender creates a new Defender with all required dependencies.
func NewDefender(client ogamex.ClientInterface, stateMgr StateReader, db *sql.DB, cfg config.DefenderConfig, log *slog.Logger) *Defender {
	return &Defender{
		client:   client,
		stateMgr: stateMgr,
		db:       db,
		cfg:      cfg,
		log:      log.With("component", "defender"),
	}
}

func (d *Defender) SetBroadcaster(b Broadcaster) {
	d.broadcaster = b
}

func (d *Defender) broadcast(msgType string, data interface{}) {
	if d.broadcaster != nil {
		d.broadcaster.Broadcast(msgType, data)
	}
}

// --- Fleet-Save Tracking Methods ---

// activeFleetSave returns the active (not completed) fleet-save event for a planet, or nil if none.
func (d *Defender) activeFleetSave(ctx context.Context, planetID int64) (*fleetSaveEvent, error) {
	var event fleetSaveEvent
	var completed int
	var recalled int
	var recallAt sql.NullTime

	err := d.db.QueryRowContext(ctx,
		`SELECT id, planet_id, fleet_id, dest_planet_id, attack_id, sent_at, recall_at, completed, recalled, created_at
		 FROM fleet_save_events WHERE planet_id = ? AND completed = FALSE ORDER BY id DESC LIMIT 1`,
		planetID,
	).Scan(&event.ID, &event.PlanetID, &event.FleetID, &event.DestPlanetID, &event.AttackID,
		&event.SentAt, &recallAt, &completed, &recalled, &event.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying active fleet-save for planet %d: %w", planetID, err)
	}

	event.Completed = completed == 1
	event.Recalled = recalled == 1
	if recallAt.Valid {
		event.RecallAt = &recallAt.Time
	}

	return &event, nil
}

// recordFleetSave inserts a new fleet-save tracking event.
func (d *Defender) recordFleetSave(ctx context.Context, planetID, fleetID, destPlanetID, attackID int64, recallAt time.Time) error {
	var recallAtVal interface{}
	if !recallAt.IsZero() {
		recallAtVal = recallAt.UTC().Format("2006-01-02 15:04:05")
	}

	_, err := d.db.ExecContext(ctx,
		`INSERT INTO fleet_save_events (planet_id, fleet_id, dest_planet_id, attack_id, recall_at)
		 VALUES (?, ?, ?, ?, ?)`,
		planetID, fleetID, destPlanetID, attackID, recallAtVal)
	if err != nil {
		return fmt.Errorf("recording fleet-save event: %w", err)
	}
	return nil
}

// completeFleetSave marks a fleet-save event as completed.
func (d *Defender) completeFleetSave(ctx context.Context, fleetID int64) error {
	_, err := d.db.ExecContext(ctx,
		`UPDATE fleet_save_events SET completed = TRUE WHERE fleet_id = ?`, fleetID)
	if err != nil {
		return fmt.Errorf("completing fleet-save for fleet %d: %w", fleetID, err)
	}
	return nil
}

// markRecalled marks a fleet-save event as recalled.
func (d *Defender) markRecalled(ctx context.Context, fleetID int64) error {
	_, err := d.db.ExecContext(ctx,
		`UPDATE fleet_save_events SET recalled = TRUE WHERE fleet_id = ?`, fleetID)
	if err != nil {
		return fmt.Errorf("marking fleet-save recalled for fleet %d: %w", fleetID, err)
	}
	return nil
}

// pendingRecalls returns fleet-save events that are due for recall (recall_at <= now, not yet recalled or completed).
func (d *Defender) pendingRecalls(ctx context.Context) ([]fleetSaveEvent, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT id, planet_id, fleet_id, dest_planet_id, attack_id, sent_at, recall_at, completed, recalled, created_at
		 FROM fleet_save_events
		 WHERE completed = FALSE AND recalled = FALSE AND recall_at IS NOT NULL AND recall_at <= datetime('now')
		 ORDER BY recall_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("querying pending recalls: %w", err)
	}
	defer rows.Close()

	var events []fleetSaveEvent
	for rows.Next() {
		var event fleetSaveEvent
		var completed int
		var recalled int
		var recallAt sql.NullTime

		err := rows.Scan(&event.ID, &event.PlanetID, &event.FleetID, &event.DestPlanetID,
			&event.AttackID, &event.SentAt, &recallAt, &completed, &recalled, &event.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning pending recall: %w", err)
		}

		event.Completed = completed == 1
		event.Recalled = recalled == 1
		if recallAt.Valid {
			event.RecallAt = &recallAt.Time
		}

		events = append(events, event)
	}

	return events, nil
}

// --- Attack Identification ---

// identifyEndangered determines which planets need fleet-saving from the given attacks.
// It filters out espionage-only attacks, groups by destination, and checks timing.
func (d *Defender) identifyEndangered(attacks []model.AttackEvent, serverNow time.Time) []endangeredPlanet {
	// Dangerous mission types: Attack(1), ACS Attack(2), Moon Destruction(9), Missile Attack(10)
	dangerMissions := map[int]bool{1: true, 2: true, 9: true, 10: true}

	// Group dangerous attacks by destination coordinates
	type coordKey struct{ galaxy, system, position int; typeStr string }
	attackGroups := make(map[coordKey][]model.AttackEvent)

	for _, atk := range attacks {
		if !dangerMissions[atk.MissionType] {
			continue // Skip espionage and non-dangerous missions
		}
		key := coordKey{atk.Destination.Galaxy, atk.Destination.System, atk.Destination.Position, atk.Destination.Type}
		attackGroups[key] = append(attackGroups[key], atk)
	}

	// Get all planets to resolve destinations
	planets, err := d.stateMgr.GetPlanets(context.Background())
	if err != nil {
		d.log.Error("Failed to get planets for attack identification", "error", err)
		return nil
	}

	// Build coordinate → planet lookup
	planetLookup := make(map[coordKey]model.Planet)
	for _, p := range planets {
		key := coordKey{p.Coordinate.Galaxy, p.Coordinate.System, p.Coordinate.Position, p.Coordinate.Type}
		planetLookup[key] = p
	}

	safetyMargin := time.Duration(d.cfg.SafetyMarginMs) * time.Millisecond
	minReaction := time.Duration(d.cfg.MinReactionDelayS) * time.Second
	minTimeRequired := safetyMargin + minReaction

	var endangered []endangeredPlanet
	for key, attacks := range attackGroups {
		planet, ok := planetLookup[key]
		if !ok {
			continue // Attack destination doesn't match our planet
		}

		// Find earliest arrival
		earliest := findEarliestArrival(attacks)
		timeUntilAttack := earliest.Sub(serverNow)

		if timeUntilAttack < minTimeRequired {
			d.log.Warn("Attack too close to react safely",
				"planet", planet.Name,
				"timeUntilAttack", timeUntilAttack,
				"minRequired", minTimeRequired)
			continue
		}

		endangered = append(endangered, endangeredPlanet{
			planet:          planet,
			attacks:         attacks,
			timeUntilAttack: timeUntilAttack,
		})
	}

	return endangered
}

// --- Poll and Run ---

// Run starts the defender poll loop. Blocks until context is cancelled.
func (d *Defender) Run(ctx context.Context) {
	interval := time.Duration(d.cfg.PollIntervalMs) * time.Millisecond
	d.log.Info("Starting defender", "interval", interval, "safetyMargin", time.Duration(d.cfg.SafetyMarginMs)*time.Millisecond)

	for {
		// Randomized jitter: [0, interval/2)
		jitter := time.Duration(rand.Intn(int(interval.Milliseconds()/2)+1)) * time.Millisecond
		waitTime := interval + jitter

		select {
		case <-ctx.Done():
			d.log.Info("Defender stopped")
			return
		case <-time.After(waitTime):
			d.poll(ctx)
		}
	}
}

// poll checks for attacks and processes any pending recalls.
func (d *Defender) poll(ctx context.Context) {
	if err := d.stateMgr.RefreshNow(ctx); err != nil {
		d.log.Error("Failed to force-refresh state", "error", err)
	}

	attacks, err := d.client.GetAttacks(ctx)
	if err != nil {
		d.log.Error("Failed to check attacks", "error", err)
		return
	}
	if len(attacks) == 0 {
		return
	}

	d.handleAttacks(ctx, attacks)
	d.processRecalls(ctx)
}

// handleAttacks processes detected attack events and triggers fleet-saves.
func (d *Defender) handleAttacks(ctx context.Context, attacks []model.AttackEvent) {
	// Get server time for accurate timing
	serverTimeStr, err := d.client.GetServerTime(ctx)
	if err != nil {
		d.log.Error("Failed to get server time", "error", err)
		return
	}
	serverNow, err := time.Parse("2006-01-02 15:04:05", serverTimeStr)
	if err != nil {
		d.log.Error("Failed to parse server time", "raw", serverTimeStr, "error", err)
		return
	}

	endangered := d.identifyEndangered(attacks, serverNow)
	for _, ep := range endangered {
		// Check if planet already has active fleet-save
		active, err := d.activeFleetSave(ctx, int64(ep.planet.ID))
		if err != nil {
			d.log.Error("Failed to check active fleet-save", "planet", ep.planet.Name, "error", err)
			continue
		}
		if active != nil {
			d.log.Info("Planet already has active fleet-save, skipping",
				"planet", ep.planet.Name, "fleetID", active.FleetID)
			continue
		}

		// Launch save in goroutine (handles reaction delay internally)
		go d.savePlanet(ctx, ep.planet, ep.attacks, ep.timeUntilAttack)
	}
}

// --- Fleet-Save Execution ---

// savePlanet executes a fleet-save for an endangered planet.
func (d *Defender) savePlanet(ctx context.Context, planet model.Planet, attacks []model.AttackEvent, timeUntilAttack time.Duration) {
	// 0. Check for existing active fleet-save (prevent duplicates)
	active, err := d.activeFleetSave(ctx, int64(planet.ID))
	if err != nil {
		d.log.Error("Failed to check active fleet-save", "planet", planet.Name, "error", err)
		return
	}
	if active != nil {
		d.log.Info("Planet already has active fleet-save, skipping",
			"planet", planet.Name, "fleetID", active.FleetID)
		return
	}

	// 1. Randomized reaction delay (anti-detection)
	reactionDelay := calcReactionDelay(timeUntilAttack, d.cfg)
	if reactionDelay == 0 {
		d.log.Warn("Attack too close to react safely",
			"planet", planet.Name, "timeUntilAttack", timeUntilAttack)
		return
	}

	d.log.Info("Scheduling fleet-save",
		"planet", planet.Name, "delay", reactionDelay, "timeUntilAttack", timeUntilAttack)

	select {
	case <-ctx.Done():
		return
	case <-time.After(reactionDelay):
		// proceed
	}

	// 2. Check fleet slots
	slots, err := d.client.GetSlots(ctx)
	if err != nil {
		d.log.Error("Failed to check fleet slots", "error", err)
		return
	}
	if slots.InUse >= slots.Total {
		d.log.Error("No fleet slots available — cannot save fleet!", "planet", planet.Name)
		return
	}

	// 3. Get current state
	resources, _, err := d.stateMgr.GetResources(ctx, planet.ID)
	if err != nil {
		d.log.Error("Failed to get resources", "error", err)
		return
	}

	ships, err := d.client.GetShips(ctx, planet.ID) // Fresh from ogamed
	if err != nil {
		d.log.Error("Failed to get ships", "error", err)
		return
	}

	research, err := d.stateMgr.GetResearch(ctx)
	if err != nil {
		d.log.Error("Failed to get research", "error", err)
		return
	}

	allPlanets, err := d.stateMgr.GetPlanets(ctx)
	if err != nil {
		d.log.Error("Failed to get planets", "error", err)
		return
	}

	// 4. Check if any ships to save
	if !hasShips(ships) {
		d.log.Info("No ships on planet — nothing to save", "planet", planet.Name)
		return
	}

	// 5. Calculate escape routes
	routes := CalcEscapeRoutes(planet, ships, resources, allPlanets, attacks, research)
	if len(routes) == 0 {
		d.log.Error("NO SAFE DESTINATION FOUND for fleet-save!",
			"planet", planet.Name, "resources", resources)
		return
	}

	route := routes[0] // safest route
	d.log.Info("Executing fleet-save",
		"planet", planet.Name, "destination", route.Dest,
		"speed", route.Speed, "fuel", route.FuelCost,
		"duration", route.Duration, "safetyScore", route.SafetyScore)

	// 6. Build SendFleetRequest
	shipList := shipsToList(ships)
	coordType := 1 // planet
	if route.Dest.Type == "moon" {
		coordType = 3
	}

	req := model.SendFleetRequest{
		PlanetID:  planet.ID,
		Ships:     shipList,
		Speed:     route.Speed,
		Galaxy:    route.Dest.Galaxy,
		System:    route.Dest.System,
		Position:  route.Dest.Position,
		Type:      coordType,
		Mission:   constants.MissionDeploy,
		Metal:     route.MetalLoad,
		Crystal:   route.CrystalLoad,
		Deuterium: route.DeutLoad,
	}

	// 7. Send the fleet
	fleetID, err := d.client.SendFleet(ctx, req)
	if err != nil {
		d.log.Error("FLEET-SAVE FAILED!", "planet", planet.Name, "error", err)
		return
	}

	// 8. Record fleet-save event
	recallAt := time.Time{} // zero = no recall scheduled
	if d.cfg.IsRecallEnabled() {
		// Only schedule recall if return flight time is within max
		if route.Duration.Seconds() <= float64(d.cfg.MaxReturnFlightS) {
			attackArrival := findEarliestArrival(attacks)
			recallAt = attackArrival.Add(30 * time.Second)
		}
	}

	attackID := int64(0)
	if len(attacks) > 0 {
		attackID = attacks[0].ID
	}

	if err := d.recordFleetSave(ctx, int64(planet.ID), fleetID, int64(route.DestPlanetID), attackID, recallAt); err != nil {
		d.log.Error("Failed to record fleet-save event", "error", err)
	}

	d.log.Info("Fleet-save executed successfully",
		"planet", planet.Name, "fleetID", fleetID, "destination", route.Dest)
	d.broadcast("fleet_save", map[string]interface{}{
		"planetId":   planet.ID,
		"planetName": planet.Name,
		"fleetId":    fleetID,
		"destination": route.Dest,
	})
}

// --- Recall Processing ---

// processRecalls checks for fleet-save events that are due for recall and executes them.
func (d *Defender) processRecalls(ctx context.Context) {
	if !d.cfg.IsRecallEnabled() {
		return
	}

	pending, err := d.pendingRecalls(ctx)
	if err != nil {
		d.log.Error("Failed to query pending recalls", "error", err)
		return
	}

	for _, event := range pending {
		// Check if fleet is still outgoing (not already returning)
		fleets, err := d.client.GetFleets(ctx)
		if err != nil {
			d.log.Error("Failed to get fleets for recall check", "error", err)
			continue
		}

		fleet := findFleetByID(fleets, event.FleetID)
		if fleet == nil {
			// Fleet no longer exists (landed, was destroyed, etc.)
			d.completeFleetSave(ctx, event.FleetID)
			d.log.Warn("Fleet no longer found, marking save complete", "fleetID", event.FleetID)
			continue
		}

		if fleet.ReturnFlight {
			// Already returning — just mark as recalled
			d.markRecalled(ctx, event.FleetID)
			continue
		}

		// Cancel the fleet (recall it)
		if err := d.client.CancelFleet(ctx, event.FleetID); err != nil {
			d.log.Error("Failed to recall fleet", "fleetID", event.FleetID, "error", err)
			continue
		}

		d.markRecalled(ctx, event.FleetID)
		d.log.Info("Fleet recalled successfully", "fleetID", event.FleetID, "planetID", event.PlanetID)
	}
}

// --- Helper Functions ---

// calcReactionDelay returns a randomized delay for anti-detection, capped to preserve safety margin.
// Returns 0 if there's not enough time to react safely.
func calcReactionDelay(timeUntilAttack time.Duration, cfg config.DefenderConfig) time.Duration {
	minDelay := time.Duration(cfg.MinReactionDelayS) * time.Second
	minSafety := time.Duration(cfg.SafetyMarginMs) * time.Millisecond

	maxAllowedDelay := timeUntilAttack - minSafety
	if maxAllowedDelay < minDelay {
		return 0 // Too late to react safely
	}

	// Random delay in [minDelay, maxAllowedDelay)
	delayRange := maxAllowedDelay - minDelay
	if delayRange <= 0 {
		return minDelay
	}

	return minDelay + time.Duration(rand.Int63n(int64(delayRange)))
}

// shipsToList converts Ships struct to []ShipCount, skipping zero counts and SolarSatellite.
func shipsToList(ships model.Ships) []model.ShipCount {
	var list []model.ShipCount
	allShips := shipsToSlice(ships)
	for _, s := range allShips {
		if s.Count > 0 {
			list = append(list, model.ShipCount{ID: s.ID, Count: s.Count})
		}
	}
	return list
}

// findEarliestArrival returns the earliest ArrivalTime from a list of attacks.
func findEarliestArrival(attacks []model.AttackEvent) time.Time {
	if len(attacks) == 0 {
		return time.Time{}
	}
	earliest := attacks[0].ArrivalTime
	for _, atk := range attacks[1:] {
		if atk.ArrivalTime.Before(earliest) {
			earliest = atk.ArrivalTime
		}
	}
	return earliest
}

// findFleetByID finds a fleet by its ID in a list of fleets.
func findFleetByID(fleets []model.Fleet, id int64) *model.Fleet {
	for i := range fleets {
		if int64(fleets[i].ID) == id {
			return &fleets[i]
		}
	}
	return nil
}
