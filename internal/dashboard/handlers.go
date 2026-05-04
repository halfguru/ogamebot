package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/user/ogame-bot/internal/builder"
	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/model"
)

// StateReader is the interface that state.Manager satisfies.
// Dashboard handlers read game state through this interface.
type StateReader interface {
	GetPlanets(ctx context.Context) ([]model.Planet, error)
	GetResources(ctx context.Context, planetID int) (model.Resources, model.PlanetDetails, error)
	GetFleets(ctx context.Context) ([]model.Fleet, error)
	GetResearch(ctx context.Context) (model.Research, error)
	GetBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error)
	GetFacilities(ctx context.Context, planetID int) (model.Facilities, error)
	GetLastRefreshTime() time.Time
	GetPlanetCount() int
}

// Handlers holds dependencies for REST API endpoint handlers.
type PlanReader interface {
	GetPlan() builder.BuildPlan
}

type Handlers struct {
	stateMgr  StateReader
	planMgr   PlanReader
	db        *sql.DB
	hub       *Hub
	log       *slog.Logger
	features  config.FeaturesConfig
	startTime time.Time
}

func NewHandlers(stateMgr StateReader, planMgr PlanReader, db *sql.DB, hub *Hub, log *slog.Logger, features config.FeaturesConfig, startTime time.Time) *Handlers {
	return &Handlers{
		stateMgr:  stateMgr,
		planMgr:   planMgr,
		db:        db,
		hub:       hub,
		log:       log.With("component", "dashboard-handlers"),
		features:  features,
		startTime: startTime,
	}
}

// handlePlanets returns all planets with resources and buildings.
// GET /api/planets
func (h *Handlers) handlePlanets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	planets, err := h.stateMgr.GetPlanets(ctx)
	if err != nil {
		h.writeError(w, "failed to fetch planets", http.StatusInternalServerError)
		h.log.Error("handlePlanets: GetPlanets failed", "error", err)
		return
	}

	apiPlanets := make([]APIPlanet, 0, len(planets))
	for _, p := range planets {
		apiPlanet := APIPlanet{
			ID:             p.ID,
			Name:           p.Name,
			Galaxy:         p.Coordinate.Galaxy,
			System:         p.Coordinate.System,
			Position:       p.Coordinate.Position,
			IsMoon:         p.IsMoon,
			Diameter:       p.Diameter,
			FieldsUsed:     p.FieldsUsed,
			FieldsTotal:    p.FieldsTotal,
			TemperatureMin: p.TemperatureMin,
			TemperatureMax: p.TemperatureMax,
			ImageType:      p.ImageType,
		}

		res, _, err := h.stateMgr.GetResources(ctx, p.ID)
		if err != nil {
			h.log.Warn("handlePlanets: GetResources failed", "planetID", p.ID, "error", err)
		} else {
			apiPlanet.Resources = APIResources{
				Metal:               res.Metal,
				Crystal:             res.Crystal,
				Deuterium:           res.Deuterium,
				Energy:              res.Energy,
				MetalStorage:        res.MetalStorage,
				CrystalStorage:      res.CrystalStorage,
				DeuteriumStorage:    res.DeuteriumStorage,
				MetalProduction:     res.MetalProduction,
				CrystalProduction:   res.CrystalProduction,
				DeuteriumProduction: res.DeuteriumProduction,
			}
		}

		bld, err := h.stateMgr.GetBuildings(ctx, p.ID)
		if err != nil {
			h.log.Warn("handlePlanets: GetBuildings failed", "planetID", p.ID, "error", err)
		} else {
			apiPlanet.Buildings = APIBuildings{
				MetalMine:            bld.MetalMine,
				CrystalMine:          bld.CrystalMine,
				DeuteriumSynthesizer: bld.DeuteriumSynthesizer,
				SolarPlant:           bld.SolarPlant,
				FusionReactor:        bld.FusionReactor,
				MetalStorage:         bld.MetalStorage,
				CrystalStorage:       bld.CrystalStorage,
				DeuteriumStorage:     bld.DeuteriumStorage,
			}
		}

		fac, err := h.stateMgr.GetFacilities(ctx, p.ID)
		if err != nil {
			h.log.Warn("handlePlanets: GetFacilities failed", "planetID", p.ID, "error", err)
		} else {
			apiPlanet.Facilities = APIFacilities{
				RoboticsFactory: fac.RoboticsFactory,
				Shipyard:        fac.Shipyard,
				ResearchLab:     fac.ResearchLab,
				NaniteFactory:   fac.NaniteFactory,
			}
		}

		apiPlanets = append(apiPlanets, apiPlanet)
	}

	h.writeJSON(w, apiPlanets)
}

// handleFleets returns all active fleets.
// GET /api/fleets
func (h *Handlers) handleFleets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fleets, err := h.stateMgr.GetFleets(ctx)
	if err != nil {
		h.writeError(w, "failed to fetch fleets", http.StatusInternalServerError)
		h.log.Error("handleFleets: GetFleets failed", "error", err)
		return
	}

	apiFleets := make([]APIFleet, 0, len(fleets))
	for _, f := range fleets {
		apiFleets = append(apiFleets, APIFleet{
			ID:            f.ID,
			Mission:       f.Mission,
			ReturnFlight:  f.ReturnFlight,
			OriginGalaxy:   f.Origin.Galaxy,
			OriginSystem:   f.Origin.System,
			OriginPosition: f.Origin.Position,
			DestGalaxy:     f.Destination.Galaxy,
			DestSystem:     f.Destination.System,
			DestPosition:   f.Destination.Position,
			ArrivalTime:    f.ArrivalTime,
			Metal:         f.Metal,
			Crystal:       f.Crystal,
			Deuterium:     f.Deuterium,
		})
	}

	h.writeJSON(w, apiFleets)
}

// handleResearch returns current research levels.
// GET /api/research
func (h *Handlers) handleResearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	research, err := h.stateMgr.GetResearch(ctx)
	if err != nil {
		h.writeError(w, "failed to fetch research", http.StatusInternalServerError)
		h.log.Error("handleResearch: GetResearch failed", "error", err)
		return
	}

	apiResearch := APIResearch{
		EnergyTechnology:             research.EnergyTechnology,
		LaserTechnology:              research.LaserTechnology,
		IonTechnology:                research.IonTechnology,
		HyperspaceTechnology:         research.HyperspaceTechnology,
		PlasmaTechnology:             research.PlasmaTechnology,
		CombustionDrive:              research.CombustionDrive,
		ImpulseDrive:                 research.ImpulseDrive,
		HyperspaceDrive:              research.HyperspaceDrive,
		EspionageTechnology:          research.EspionageTechnology,
		ComputerTechnology:           research.ComputerTechnology,
		Astrophysics:                 research.Astrophysics,
		IntergalacticResearchNetwork: research.IntergalacticResearchNetwork,
		GravitonTechnology:           research.GravitonTechnology,
		WeaponTechnology:             research.WeaponTechnology,
		ShieldingTechnology:          research.ShieldingTechnology,
		ArmourTechnology:             research.ArmourTechnology,
	}

	h.writeJSON(w, apiResearch)
}

// handleBuildEvents returns recent build events from auto-builder.
// GET /api/events/builds
func (h *Handlers) handleBuildEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, planet_id, building_id, building_name, from_level, to_level,
		        cost_metal, cost_crystal, cost_deut, roi_score, build_time_seconds, created_at
		 FROM build_events ORDER BY id DESC LIMIT 50`)
	if err != nil {
		h.writeError(w, "failed to fetch build events", http.StatusInternalServerError)
		h.log.Error("handleBuildEvents: query failed", "error", err)
		return
	}
	defer rows.Close()

	events := make([]APIBuildEvent, 0)
	for rows.Next() {
		var e APIBuildEvent
		if err := rows.Scan(&e.ID, &e.PlanetID, &e.BuildingID, &e.BuildingName, &e.FromLevel, &e.ToLevel,
			&e.CostMetal, &e.CostCrystal, &e.CostDeut, &e.ROIScore, &e.BuildTimeSeconds, &e.CreatedAt); err != nil {
			h.writeError(w, "failed to scan build event", http.StatusInternalServerError)
			h.log.Error("handleBuildEvents: scan failed", "error", err)
			return
		}
		events = append(events, e)
	}

	h.writeJSON(w, events)
}

// handleFleetSaveEvents returns recent fleet-save events from defender.
// GET /api/events/fleet-saves
func (h *Handlers) handleFleetSaveEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, planet_id, fleet_id, dest_planet_id, attack_id,
		        sent_at, recall_at, completed, recalled
		 FROM fleet_save_events ORDER BY id DESC LIMIT 20`)
	if err != nil {
		h.writeError(w, "failed to fetch fleet-save events", http.StatusInternalServerError)
		h.log.Error("handleFleetSaveEvents: query failed", "error", err)
		return
	}
	defer rows.Close()

	events := make([]APIFleetSaveEvent, 0)
	for rows.Next() {
		var e APIFleetSaveEvent
		if err := rows.Scan(&e.ID, &e.PlanetID, &e.FleetID, &e.DestPlanetID, &e.AttackID,
			&e.SentAt, &e.RecallAt, &e.Completed, &e.Recalled); err != nil {
			h.writeError(w, "failed to scan fleet-save event", http.StatusInternalServerError)
			h.log.Error("handleFleetSaveEvents: scan failed", "error", err)
			return
		}
		events = append(events, e)
	}

	h.writeJSON(w, events)
}

// handleFarmAttacks returns recent farm attack events from auto-farmer.
// GET /api/events/farm-attacks
func (h *Handlers) handleFarmAttacks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, fleet_id, planet_id, target_galaxy, target_system, target_position,
		        ships_sent, metal_looted, crystal_looted, deuterium_looted, sent_at
		 FROM farm_attacks ORDER BY id DESC LIMIT 50`)
	if err != nil {
		h.writeError(w, "failed to fetch farm attacks", http.StatusInternalServerError)
		h.log.Error("handleFarmAttacks: query failed", "error", err)
		return
	}
	defer rows.Close()

	attacks := make([]APIFarmAttack, 0)
	for rows.Next() {
		var a APIFarmAttack
		var targetGalaxy, targetSystem, targetPosition int
		if err := rows.Scan(&a.ID, &a.FleetID, &a.PlanetID,
			&targetGalaxy, &targetSystem, &targetPosition,
			&a.ShipsSent, &a.MetalLooted, &a.CrystalLooted, &a.DeutLooted, &a.SentAt); err != nil {
			h.writeError(w, "failed to scan farm attack", http.StatusInternalServerError)
			h.log.Error("handleFarmAttacks: scan failed", "error", err)
			return
		}
		a.TargetCoord = strconv.Itoa(targetGalaxy) + ":" + strconv.Itoa(targetSystem) + ":" + strconv.Itoa(targetPosition)
		attacks = append(attacks, a)
	}

	h.writeJSON(w, attacks)
}

func (h *Handlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	lastRefresh := h.stateMgr.GetLastRefreshTime()
	var lastRefreshStr string
	if !lastRefresh.IsZero() {
		lastRefreshStr = lastRefresh.UTC().Format(time.RFC3339)
	}

	response := map[string]interface{}{
		"status":           "running",
		"uptime":           time.Since(h.startTime).String(),
		"lastStateRefresh": lastRefreshStr,
		"planetsCount":     h.stateMgr.GetPlanetCount(),
		"features": map[string]bool{
			"defender":  h.features.Defender.Enabled,
			"autoBuild": h.features.AutoBuild.Enabled,
			"autoFarm":  h.features.AutoFarm.Enabled,
		},
	}
	h.writeJSON(w, response)
}

func (h *Handlers) handleBuilderPlan(w http.ResponseWriter, r *http.Request) {
	if h.planMgr == nil {
		h.writeJSON(w, builder.BuildPlan{})
		return
	}
	h.writeJSON(w, h.planMgr.GetPlan())
}

// writeJSON sets the content type and writes a JSON response.
func (h *Handlers) writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes a JSON error response.
func (h *Handlers) writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
