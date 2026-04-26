package state

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamed"
)

const defaultRefreshInterval = 60 * time.Second

// Manager polls ogamed periodically and caches game state in SQLite.
// Per D-09 (game state cached): the manager is the ONLY component that
// refreshes from ogamed — workers read from cached SQLite state.
type Manager struct {
	db          *sql.DB
	client      ogamed.ClientInterface
	log         *slog.Logger
	interval    time.Duration
	serverSpeed int // cached from ogamed, never changes for a given universe
}

// NewManager creates a game state manager with the default refresh interval.
func NewManager(db *sql.DB, client ogamed.ClientInterface, log *slog.Logger) *Manager {
	return &Manager{
		db:       db,
		client:   client,
		log:      log.With("component", "state-manager"),
		interval: defaultRefreshInterval,
	}
}

// SetInterval changes the refresh interval. Used for testing.
func (m *Manager) SetInterval(d time.Duration) {
	m.interval = d
}

// Run starts the periodic state refresh loop. Blocks until context is cancelled.
func (m *Manager) Run(ctx context.Context) {
	m.log.Info("Starting game state manager", "interval", m.interval)

	// Initial refresh
	if err := m.refresh(ctx); err != nil {
		m.log.Error("Initial state refresh failed", "error", err)
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("State manager stopped")
			return
		case <-ticker.C:
			if err := m.refresh(ctx); err != nil {
				m.log.Error("State refresh failed", "error", err)
			}
		}
	}
}

// refresh fetches all game state from ogamed and upserts into SQLite.
// Per CONTEXT.md: "stateless-safe — can be restarted at any time without causing problems"
// Errors are logged but don't crash — we keep cached data on failure.
func (m *Manager) refresh(ctx context.Context) error {
	m.log.Debug("Refreshing game state from ogamed")

	// Fetch global state
	planets, err := m.client.GetPlanets(ctx)
	if err != nil {
		return fmt.Errorf("fetching planets: %w", err)
	}

	// Cache server speed on first refresh (never changes for a given universe)
	if m.serverSpeed == 0 {
		speed, err := m.client.GetServerSpeed(ctx)
		if err != nil {
			m.log.Warn("Failed to fetch server speed", "error", err)
		} else {
			m.serverSpeed = speed
			m.log.Info("Cached server speed", "speed", speed)
		}
	}

	fleets, err := m.client.GetFleets(ctx)
	if err != nil {
		return fmt.Errorf("fetching fleets: %w", err)
	}

	research, err := m.client.GetResearch(ctx)
	if err != nil {
		return fmt.Errorf("fetching research: %w", err)
	}

	// Upsert planets and per-planet data
	for _, planet := range planets {
		if err := m.upsertPlanet(ctx, planet); err != nil {
			m.log.Error("Failed to upsert planet", "planetID", planet.ID, "error", err)
			continue // Per TS plan: catch per-planet errors without aborting whole refresh
		}

		// Fetch and persist per-planet resources
		resources, err := m.client.GetResources(ctx, planet.ID)
		if err != nil {
			m.log.Warn("Failed to fetch resources", "planetID", planet.ID, "error", err)
		} else {
			m.upsertResources(ctx, planet.ID, resources)
		}

		// Fetch and persist per-planet buildings
		buildings, err := m.client.GetResourceBuildings(ctx, planet.ID)
		if err != nil {
			m.log.Warn("Failed to fetch buildings", "planetID", planet.ID, "error", err)
		} else {
			m.upsertBuildings(ctx, planet.ID, buildings)
		}

		// Fetch and persist per-planet facilities
		facilities, err := m.client.GetFacilities(ctx, planet.ID)
		if err != nil {
			m.log.Warn("Failed to fetch facilities", "planetID", planet.ID, "error", err)
		} else {
			m.upsertFacilities(ctx, planet.ID, facilities)
		}
	}

	// Upsert fleets (full replace — delete all then insert)
	m.replaceFleets(ctx, fleets)

	// Upsert research (singleton row)
	m.upsertResearch(ctx, research)

	m.log.Info("State refresh complete", "planets", len(planets), "fleets", len(fleets))
	return nil
}

// --- Upsert helpers ---

func (m *Manager) upsertPlanet(ctx context.Context, p model.Planet) error {
	isMoon := 0
	if p.IsMoon {
		isMoon = 1
	}
	_, err := m.db.ExecContext(ctx, `INSERT OR REPLACE INTO planets
		(id, name, galaxy, system, position, is_moon, diameter, fields_used, fields_total, temperature_min, temperature_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Coordinate.Galaxy, p.Coordinate.System, p.Coordinate.Position,
		isMoon, p.Diameter, p.FieldsUsed, p.FieldsTotal, p.TemperatureMin, p.TemperatureMax)
	return err
}

func (m *Manager) upsertResources(ctx context.Context, planetID int, r model.Resources) error {
	_, err := m.db.ExecContext(ctx, `INSERT OR REPLACE INTO resources
		(planet_id, metal, crystal, deuterium, energy)
		VALUES (?, ?, ?, ?, ?)`,
		planetID, r.Metal, r.Crystal, r.Deuterium, r.Energy)
	return err
}

func (m *Manager) upsertBuildings(ctx context.Context, planetID int, b model.ResourceBuildings) error {
	_, err := m.db.ExecContext(ctx, `INSERT OR REPLACE INTO buildings
		(planet_id, metal_mine, crystal_mine, deuterium_synthesizer, solar_plant, fusion_reactor,
		 metal_storage, crystal_storage, deuterium_tank)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		planetID, b.MetalMine, b.CrystalMine, b.DeuteriumSynthesizer, b.SolarPlant,
		b.FusionReactor, b.MetalStorage, b.CrystalStorage, b.DeuteriumStorage)
	return err
}

func (m *Manager) upsertFacilities(ctx context.Context, planetID int, f model.Facilities) error {
	_, err := m.db.ExecContext(ctx, `INSERT OR REPLACE INTO facilities
		(planet_id, robotics_factory, shipyard, research_lab, alliance_depot,
		 missile_silo, nanite_factory, terraformer, space_dock)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		planetID, f.RoboticsFactory, f.Shipyard, f.ResearchLab, f.AllianceDepot,
		f.MissileSilo, f.NaniteFactory, f.Terraformer, f.SpaceDock)
	return err
}

func (m *Manager) replaceFleets(ctx context.Context, fleets []model.Fleet) {
	// Delete all existing fleets (fleets are ephemeral — full replace each cycle)
	m.db.ExecContext(ctx, "DELETE FROM fleets")

	for _, f := range fleets {
		retFlight := 0
		if f.ReturnFlight {
			retFlight = 1
		}
		_, err := m.db.ExecContext(ctx, `INSERT INTO fleets
			(id, mission, return_flight, origin_galaxy, origin_system, origin_position,
			 dest_galaxy, dest_system, dest_position, metal, crystal, deuterium, arrival_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.ID, f.Mission, retFlight,
			f.Origin.Galaxy, f.Origin.System, f.Origin.Position,
			f.Destination.Galaxy, f.Destination.System, f.Destination.Position,
			f.Metal, f.Crystal, f.Deuterium, f.ArrivalTime)
		if err != nil {
			m.log.Warn("Failed to insert fleet", "fleetID", f.ID, "error", err)
		}
	}
}

func (m *Manager) upsertResearch(ctx context.Context, r model.Research) error {
	_, err := m.db.ExecContext(ctx, `INSERT OR REPLACE INTO research
		(id, energy_technology, laser_technology, ion_technology, hyperspace_technology,
		 plasma_technology, combustion_drive, impulse_drive, hyperspace_drive,
		 espionage_technology, computer_technology, astrophysics,
		 intergalactic_research_network, graviton_technology,
		 weapon_technology, shielding_technology, armour_technology)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.EnergyTechnology, r.LaserTechnology, r.IonTechnology, r.HyperspaceTechnology,
		r.PlasmaTechnology, r.CombustionDrive, r.ImpulseDrive, r.HyperspaceDrive,
		r.EspionageTechnology, r.ComputerTechnology, r.Astrophysics,
		r.IntergalacticResearchNetwork, r.GravitonTechnology,
		r.WeaponTechnology, r.ShieldingTechnology, r.ArmourTechnology)
	return err
}

// --- Read methods ---

// GetPlanets returns all cached planets from SQLite.
func (m *Manager) GetPlanets(ctx context.Context) ([]model.Planet, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT id, name, galaxy, system, position, is_moon,
		diameter, fields_used, fields_total, temperature_min, temperature_max FROM planets`)
	if err != nil {
		return nil, fmt.Errorf("querying planets: %w", err)
	}
	defer rows.Close()

	var planets []model.Planet
	for rows.Next() {
		var p model.Planet
		var isMoon int
		err := rows.Scan(&p.ID, &p.Name, &p.Coordinate.Galaxy, &p.Coordinate.System,
			&p.Coordinate.Position, &isMoon, &p.Diameter, &p.FieldsUsed, &p.FieldsTotal,
			&p.TemperatureMin, &p.TemperatureMax)
		if err != nil {
			return nil, fmt.Errorf("scanning planet: %w", err)
		}
		p.IsMoon = isMoon == 1
		planets = append(planets, p)
	}
	return planets, nil
}

// GetResources returns the cached resources for a planet.
func (m *Manager) GetResources(ctx context.Context, planetID int) (model.Resources, error) {
	var r model.Resources
	err := m.db.QueryRowContext(ctx,
		"SELECT metal, crystal, deuterium, energy FROM resources WHERE planet_id = ?", planetID).
		Scan(&r.Metal, &r.Crystal, &r.Deuterium, &r.Energy)
	if err != nil {
		return model.Resources{}, fmt.Errorf("querying resources for planet %d: %w", planetID, err)
	}
	return r, nil
}

// GetFleets returns all cached fleets from SQLite.
func (m *Manager) GetFleets(ctx context.Context) ([]model.Fleet, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT id, mission, return_flight,
		origin_galaxy, origin_system, origin_position,
		dest_galaxy, dest_system, dest_position,
		metal, crystal, deuterium, arrival_time FROM fleets`)
	if err != nil {
		return nil, fmt.Errorf("querying fleets: %w", err)
	}
	defer rows.Close()

	var fleets []model.Fleet
	for rows.Next() {
		var f model.Fleet
		var retFlight int
		err := rows.Scan(&f.ID, &f.Mission, &retFlight,
			&f.Origin.Galaxy, &f.Origin.System, &f.Origin.Position,
			&f.Destination.Galaxy, &f.Destination.System, &f.Destination.Position,
			&f.Metal, &f.Crystal, &f.Deuterium, &f.ArrivalTime)
		if err != nil {
			return nil, fmt.Errorf("scanning fleet: %w", err)
		}
		f.ReturnFlight = retFlight == 1
		fleets = append(fleets, f)
	}
	return fleets, nil
}

// GetResearch returns the cached research from SQLite.
func (m *Manager) GetResearch(ctx context.Context) (model.Research, error) {
	var r model.Research
	err := m.db.QueryRowContext(ctx, `SELECT energy_technology, laser_technology, ion_technology,
		hyperspace_technology, plasma_technology, combustion_drive, impulse_drive, hyperspace_drive,
		espionage_technology, computer_technology, astrophysics, intergalactic_research_network,
		graviton_technology, weapon_technology, shielding_technology, armour_technology
		FROM research WHERE id = 1`).Scan(
		&r.EnergyTechnology, &r.LaserTechnology, &r.IonTechnology,
		&r.HyperspaceTechnology, &r.PlasmaTechnology, &r.CombustionDrive,
		&r.ImpulseDrive, &r.HyperspaceDrive, &r.EspionageTechnology,
		&r.ComputerTechnology, &r.Astrophysics, &r.IntergalacticResearchNetwork,
		&r.GravitonTechnology, &r.WeaponTechnology, &r.ShieldingTechnology, &r.ArmourTechnology)
	if err != nil {
		return model.Research{}, fmt.Errorf("querying research: %w", err)
	}
	return r, nil
}

// GetBuildings returns the cached resource buildings for a planet.
func (m *Manager) GetBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error) {
	var b model.ResourceBuildings
	err := m.db.QueryRowContext(ctx,
		`SELECT metal_mine, crystal_mine, deuterium_synthesizer, solar_plant, fusion_reactor,
			metal_storage, crystal_storage, deuterium_tank FROM buildings WHERE planet_id = ?`, planetID).
		Scan(&b.MetalMine, &b.CrystalMine, &b.DeuteriumSynthesizer, &b.SolarPlant, &b.FusionReactor,
			&b.MetalStorage, &b.CrystalStorage, &b.DeuteriumStorage)
	if err != nil {
		return model.ResourceBuildings{}, fmt.Errorf("querying buildings for planet %d: %w", planetID, err)
	}
	return b, nil
}

// GetFacilities returns the cached facilities for a planet.
func (m *Manager) GetFacilities(ctx context.Context, planetID int) (model.Facilities, error) {
	var f model.Facilities
	err := m.db.QueryRowContext(ctx,
		`SELECT robotics_factory, shipyard, research_lab, alliance_depot,
			missile_silo, nanite_factory, terraformer, space_dock FROM facilities WHERE planet_id = ?`, planetID).
		Scan(&f.RoboticsFactory, &f.Shipyard, &f.ResearchLab, &f.AllianceDepot,
			&f.MissileSilo, &f.NaniteFactory, &f.Terraformer, &f.SpaceDock)
	if err != nil {
		return model.Facilities{}, fmt.Errorf("querying facilities for planet %d: %w", planetID, err)
	}
	return f, nil
}

// GetServerSpeed returns the cached server speed multiplier.
// Returns error if the value hasn't been cached yet (refresh hasn't run).
func (m *Manager) GetServerSpeed(_ context.Context) (int, error) {
	if m.serverSpeed == 0 {
		return 0, fmt.Errorf("server speed not yet cached — wait for first refresh")
	}
	return m.serverSpeed, nil
}
