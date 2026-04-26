// Package model defines domain types for all OGame entities.
// JSON tags use PascalCase to match ogamed's exact response field names.
package model

import "time"

// Coordinate represents a position in the OGame universe.
type Coordinate struct {
	Galaxy   int    `json:"Galaxy"`
	System   int    `json:"System"`
	Position int    `json:"Position"`
	Type     string `json:"Type"` // "planet" or "moon"
}

// Resources represents the resource amounts on a planet.
type Resources struct {
	Metal      int `json:"Metal"`
	Crystal    int `json:"Crystal"`
	Deuterium  int `json:"Deuterium"`
	Energy     int `json:"Energy"`
	DarkMatter int `json:"Darkmatter"`
}

// Planet represents a player's planet or moon.
type Planet struct {
	ID             int        `json:"ID"`
	Name           string     `json:"Name"`
	Coordinate     Coordinate `json:"Coordinate"`
	Diameter       int        `json:"Diameter"`
	FieldsUsed     int        `json:"FieldsUsed"`
	FieldsTotal    int        `json:"FieldsTotal"`
	TemperatureMin int        `json:"TemperatureMin"`
	TemperatureMax int        `json:"TemperatureMax"`
	IsMoon         bool       `json:"IsMoon"`
}

// ShipCount represents a ship type and its quantity in a fleet.
type ShipCount struct {
	ID    int `json:"ID"`
	Count int `json:"Count"`
}

// Fleet represents a moving fleet in the game.
type Fleet struct {
	ID           int         `json:"ID"`
	Mission      int         `json:"Mission"`
	ReturnFlight bool        `json:"ReturnFlight"`
	Origin       Coordinate  `json:"Origin"`
	Destination  Coordinate  `json:"Destination"`
	ArrivalTime  int64       `json:"ArrivalTime"`
	Ships        []ShipCount `json:"Ships"`
	Metal        int         `json:"Metal"`
	Crystal      int         `json:"Crystal"`
	Deuterium    int         `json:"Deuterium"`
}

// FleetSlots represents the total and in-use fleet slot counts.
type FleetSlots struct {
	Total  int `json:"Total"`
	InUse  int `json:"InUse"`
}

// ResourceBuildings represents the levels of resource-producing buildings.
type ResourceBuildings struct {
	MetalMine            int `json:"MetalMine"`
	CrystalMine          int `json:"CrystalMine"`
	DeuteriumSynthesizer int `json:"DeuteriumSynthesizer"`
	SolarPlant           int `json:"SolarPlant"`
	FusionReactor        int `json:"FusionReactor"`
	MetalStorage         int `json:"MetalStorage"`
	CrystalStorage       int `json:"CrystalStorage"`
	DeuteriumStorage     int `json:"DeuteriumStorage"`
}

// Facilities represents the levels of facility buildings on a planet.
type Facilities struct {
	RoboticsFactory int `json:"RoboticsFactory"`
	Shipyard        int `json:"Shipyard"`
	ResearchLab     int `json:"ResearchLab"`
	AllianceDepot   int `json:"AllianceDepot"`
	MissileSilo     int `json:"MissileSilo"`
	NaniteFactory   int `json:"NaniteFactory"`
	Terraformer     int `json:"Terraformer"`
	SpaceDock       int `json:"SpaceDock"`
}

// Defence represents the quantities of defensive structures on a planet.
type Defence struct {
	RocketLauncher       int `json:"RocketLauncher"`
	LightLaser           int `json:"LightLaser"`
	HeavyLaser           int `json:"HeavyLaser"`
	GaussCannon          int `json:"GaussCannon"`
	IonCannon            int `json:"IonCannon"`
	PlasmaTurret         int `json:"PlasmaTurret"`
	SmallShield          int `json:"SmallShield"`
	LargeShield          int `json:"LargeShield"`
	AntiBallisticMissile int `json:"AntiBallisticMissile"`
	InterplanetaryMissile int `json:"InterplanetaryMissile"`
}

// Ships represents the quantities of ships on a planet.
type Ships struct {
	LightFighter   int `json:"LightFighter"`
	HeavyFighter   int `json:"HeavyFighter"`
	Cruiser        int `json:"Cruiser"`
	Battleship     int `json:"Battleship"`
	Battlecruiser  int `json:"Battlecruiser"`
	Bomber         int `json:"Bomber"`
	Destroyer      int `json:"Destroyer"`
	Deathstar      int `json:"Deathstar"`
	SmallCargo     int `json:"SmallCargo"`
	LargeCargo     int `json:"LargeCargo"`
	ColonyShip     int `json:"ColonyShip"`
	Recycler       int `json:"Recycler"`
	EspionageProbe int `json:"EspionageProbe"`
	SolarSatellite int `json:"SolarSatellite"`
}

// Research represents the levels of all research technologies.
type Research struct {
	EnergyTechnology             int `json:"EnergyTechnology"`
	LaserTechnology              int `json:"LaserTechnology"`
	IonTechnology                int `json:"IonTechnology"`
	HyperspaceTechnology         int `json:"HyperspaceTechnology"`
	PlasmaTechnology             int `json:"PlasmaTechnology"`
	CombustionDrive              int `json:"CombustionDrive"`
	ImpulseDrive                 int `json:"ImpulseDrive"`
	HyperspaceDrive              int `json:"HyperspaceDrive"`
	EspionageTechnology          int `json:"EspionageTechnology"`
	ComputerTechnology           int `json:"ComputerTechnology"`
	Astrophysics                 int `json:"Astrophysics"`
	IntergalacticResearchNetwork int `json:"IntergalacticResearchNetwork"`
	GravitonTechnology           int `json:"GravitonTechnology"`
	WeaponTechnology             int `json:"WeaponTechnology"`
	ShieldingTechnology          int `json:"ShieldingTechnology"`
	ArmourTechnology             int `json:"ArmourTechnology"`
}

// AttackEvent represents an incoming attack detected by ogamed.
type AttackEvent struct {
	ID              int64       `json:"ID"`
	MissionType     int         `json:"MissionType"`
	Origin          Coordinate  `json:"Origin"`
	Destination     Coordinate  `json:"Destination"`
	DestinationName string      `json:"DestinationName"`
	ArrivalTime     time.Time   `json:"ArrivalTime"`
	ArriveIn        int64       `json:"ArriveIn"`
	AttackerName    string      `json:"AttackerName"`
	AttackerID      int64       `json:"AttackerID"`
	UnionID         int64       `json:"UnionID"`
	Missiles        int64       `json:"Missiles"`
	Ships           *[]ShipCount `json:"Ships"`
}

// SendFleetRequest contains all parameters needed to dispatch a fleet.
type SendFleetRequest struct {
	PlanetID  int
	Ships     []ShipCount
	Speed     int // 1-10
	Galaxy    int
	System    int
	Position  int
	Type      int // 1=planet, 3=moon
	Mission   int
	Metal     int64
	Crystal   int64
	Deuterium int64
}

// Construction represents an active construction on a planet.
type Construction struct {
	ID        int   `json:"ID"`
	Level     int   `json:"Level"`
	Countdown int64 `json:"Countdown"`
}

// Constructions represents all active constructions on a planet.
type Constructions struct {
	Building  Construction `json:"Building"`
	Research  Construction `json:"Research"`
	Shipyard  Construction `json:"Shipyard"`
}

// Slots represents fleet and expedition slot usage.
type Slots struct {
	InUse    int `json:"InUse"`
	Total    int `json:"Total"`
	ExpInUse int `json:"ExpInUse"`
	ExpTotal int `json:"ExpTotal"`
}
