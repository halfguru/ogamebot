// Package dashboard provides the HTTP/WebSocket API layer for the web dashboard.
// REST endpoints serve game state (planets, resources, fleets) and bot event history
// (builds, fleet-saves, farm attacks). WebSocket endpoint pushes real-time updates.
package dashboard

import "time"

// APIPlanet is a planet with nested resources and buildings for the dashboard API.
type APIPlanet struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	Galaxy         int        `json:"galaxy"`
	System         int        `json:"system"`
	Position       int        `json:"position"`
	IsMoon         bool       `json:"isMoon"`
	Diameter       int        `json:"diameter"`
	FieldsUsed     int        `json:"fieldsUsed"`
	FieldsTotal    int        `json:"fieldsTotal"`
	TemperatureMin int        `json:"temperatureMin"`
	TemperatureMax int        `json:"temperatureMax"`
	Resources      APIResources `json:"resources"`
	Buildings      APIBuildings `json:"buildings"`
}

// APIResources represents resource amounts for the dashboard API.
type APIResources struct {
	Metal     int `json:"metal"`
	Crystal   int `json:"crystal"`
	Deuterium int `json:"deuterium"`
	Energy    int `json:"energy"`
}

// APIBuildings represents resource building levels for the dashboard API.
type APIBuildings struct {
	MetalMine            int `json:"metalMine"`
	CrystalMine          int `json:"crystalMine"`
	DeuteriumSynthesizer int `json:"deuteriumSynthesizer"`
	SolarPlant           int `json:"solarPlant"`
	FusionReactor        int `json:"fusionReactor"`
	MetalStorage         int `json:"metalStorage"`
	CrystalStorage       int `json:"crystalStorage"`
	DeuteriumStorage     int `json:"deuteriumStorage"`
}

// APIFleet represents a fleet movement for the dashboard API.
type APIFleet struct {
	ID           int  `json:"id"`
	Mission      int  `json:"mission"`
	ReturnFlight bool `json:"returnFlight"`
	OriginGalaxy   int `json:"originGalaxy"`
	OriginSystem   int `json:"originSystem"`
	OriginPosition int `json:"originPosition"`
	DestGalaxy     int `json:"destGalaxy"`
	DestSystem     int `json:"destSystem"`
	DestPosition   int `json:"destPosition"`
	ArrivalTime    int64 `json:"arrivalTime"`
	Metal         int  `json:"metal"`
	Crystal       int  `json:"crystal"`
	Deuterium     int  `json:"deuterium"`
}

// APIResearch represents research levels for the dashboard API.
type APIResearch struct {
	EnergyTechnology             int `json:"energyTechnology"`
	LaserTechnology              int `json:"laserTechnology"`
	IonTechnology                int `json:"ionTechnology"`
	HyperspaceTechnology         int `json:"hyperspaceTechnology"`
	PlasmaTechnology             int `json:"plasmaTechnology"`
	CombustionDrive              int `json:"combustionDrive"`
	ImpulseDrive                 int `json:"impulseDrive"`
	HyperspaceDrive              int `json:"hyperspaceDrive"`
	EspionageTechnology          int `json:"espionageTechnology"`
	ComputerTechnology           int `json:"computerTechnology"`
	Astrophysics                 int `json:"astrophysics"`
	IntergalacticResearchNetwork int `json:"intergalacticResearchNetwork"`
	GravitonTechnology           int `json:"gravitonTechnology"`
	WeaponTechnology             int `json:"weaponTechnology"`
	ShieldingTechnology          int `json:"shieldingTechnology"`
	ArmourTechnology             int `json:"armourTechnology"`
}

// APIBuildEvent represents a build event from the auto-builder for the dashboard API.
type APIBuildEvent struct {
	ID           int64   `json:"id"`
	PlanetID     int     `json:"planetId"`
	BuildingName string  `json:"buildingName"`
	FromLevel    int     `json:"fromLevel"`
	ToLevel      int     `json:"toLevel"`
	CostMetal    int     `json:"costMetal"`
	CostCrystal  int     `json:"costCrystal"`
	CostDeut     int     `json:"costDeut"`
	ROIScore     float64 `json:"roiScore"`
	CreatedAt    string  `json:"createdAt"`
}

// APIFleetSaveEvent represents a fleet-save event from the defender for the dashboard API.
type APIFleetSaveEvent struct {
	ID           int64      `json:"id"`
	PlanetID     int64      `json:"planetId"`
	FleetID      int64      `json:"fleetId"`
	DestPlanetID int64      `json:"destPlanetId"`
	AttackID     int64      `json:"attackId"`
	SentAt       time.Time  `json:"sentAt"`
	RecallAt     *time.Time `json:"recallAt"`
	Completed    bool       `json:"completed"`
	Recalled     bool       `json:"recalled"`
}

// APIFarmAttack represents a farm attack from the auto-farmer for the dashboard API.
type APIFarmAttack struct {
	ID            int64     `json:"id"`
	FleetID       int64     `json:"fleetId"`
	PlanetID      int64     `json:"planetId"`
	TargetCoord   string    `json:"targetCoord"` // "galaxy:system:position"
	ShipsSent     int       `json:"shipsSent"`
	MetalLooted   int64     `json:"metalLooted"`
	CrystalLooted int64     `json:"crystalLooted"`
	DeutLooted    int64     `json:"deuteriumLooted"`
	SentAt        time.Time `json:"sentAt"`
}

// WSMessage is a WebSocket broadcast message.
type WSMessage struct {
	Type string      `json:"type"` // "state_update", "build_event", "fleet_save_event", "farm_attack"
	Data interface{} `json:"data"`
}
