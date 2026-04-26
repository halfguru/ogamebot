package model

import (
	"encoding/json"
	"testing"
)

func TestCoordinateJSON(t *testing.T) {
	raw := `{"Galaxy":1,"System":2,"Position":3,"Type":"planet"}`

	var coord Coordinate
	if err := json.Unmarshal([]byte(raw), &coord); err != nil {
		t.Fatalf("failed to unmarshal Coordinate: %v", err)
	}

	if coord.Galaxy != 1 {
		t.Errorf("Galaxy = %d, want 1", coord.Galaxy)
	}
	if coord.System != 2 {
		t.Errorf("System = %d, want 2", coord.System)
	}
	if coord.Position != 3 {
		t.Errorf("Position = %d, want 3", coord.Position)
	}
	if coord.Type != "planet" {
		t.Errorf("Type = %q, want %q", coord.Type, "planet")
	}

	// Round-trip
	data, err := json.Marshal(coord)
	if err != nil {
		t.Fatalf("failed to marshal Coordinate: %v", err)
	}
	var coord2 Coordinate
	if err := json.Unmarshal(data, &coord2); err != nil {
		t.Fatalf("failed to unmarshal round-trip Coordinate: %v", err)
	}
	if coord2 != coord {
		t.Errorf("round-trip mismatch: got %+v, want %+v", coord2, coord)
	}
}

func TestPlanetJSON(t *testing.T) {
	raw := `{"ID":336,"Name":"Homeworld","Coordinate":{"Galaxy":1,"System":2,"Position":3,"Type":"planet"},"Diameter":12800,"FieldsUsed":10,"FieldsTotal":163,"TemperatureMin":15,"TemperatureMax":65,"IsMoon":false}`

	var planet Planet
	if err := json.Unmarshal([]byte(raw), &planet); err != nil {
		t.Fatalf("failed to unmarshal Planet: %v", err)
	}

	if planet.ID != 336 {
		t.Errorf("ID = %d, want 336", planet.ID)
	}
	if planet.Name != "Homeworld" {
		t.Errorf("Name = %q, want %q", planet.Name, "Homeworld")
	}
	if planet.Coordinate.Galaxy != 1 {
		t.Errorf("Coordinate.Galaxy = %d, want 1", planet.Coordinate.Galaxy)
	}
	if planet.Diameter != 12800 {
		t.Errorf("Diameter = %d, want 12800", planet.Diameter)
	}
	if planet.FieldsUsed != 10 {
		t.Errorf("FieldsUsed = %d, want 10", planet.FieldsUsed)
	}
	if planet.FieldsTotal != 163 {
		t.Errorf("FieldsTotal = %d, want 163", planet.FieldsTotal)
	}
	if planet.TemperatureMin != 15 {
		t.Errorf("TemperatureMin = %d, want 15", planet.TemperatureMin)
	}
	if planet.TemperatureMax != 65 {
		t.Errorf("TemperatureMax = %d, want 65", planet.TemperatureMax)
	}
	if planet.IsMoon != false {
		t.Errorf("IsMoon = %v, want false", planet.IsMoon)
	}
}

func TestResourcesJSON(t *testing.T) {
	raw := `{"Metal":1000,"Crystal":500,"Deuterium":200,"Energy":3000,"Darkmatter":0}`

	var res Resources
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		t.Fatalf("failed to unmarshal Resources: %v", err)
	}

	if res.Metal != 1000 {
		t.Errorf("Metal = %d, want 1000", res.Metal)
	}
	if res.Crystal != 500 {
		t.Errorf("Crystal = %d, want 500", res.Crystal)
	}
	if res.Deuterium != 200 {
		t.Errorf("Deuterium = %d, want 200", res.Deuterium)
	}
	if res.Energy != 3000 {
		t.Errorf("Energy = %d, want 3000", res.Energy)
	}
	if res.DarkMatter != 0 {
		t.Errorf("DarkMatter = %d, want 0", res.DarkMatter)
	}
}

func TestFleetJSON(t *testing.T) {
	raw := `{
		"ID": 42,
		"Mission": 1,
		"ReturnFlight": false,
		"Origin": {"Galaxy":1,"System":2,"Position":3,"Type":"planet"},
		"Destination": {"Galaxy":2,"System":3,"Position":4,"Type":"planet"},
		"ArrivalTime": 1700000000,
		"Ships": [{"ID":204,"Count":100}],
		"Metal": 5000,
		"Crystal": 3000,
		"Deuterium": 1000
	}`

	var fleet Fleet
	if err := json.Unmarshal([]byte(raw), &fleet); err != nil {
		t.Fatalf("failed to unmarshal Fleet: %v", err)
	}

	if fleet.ID != 42 {
		t.Errorf("ID = %d, want 42", fleet.ID)
	}
	if fleet.Mission != 1 {
		t.Errorf("Mission = %d, want 1", fleet.Mission)
	}
	if fleet.ReturnFlight != false {
		t.Errorf("ReturnFlight = %v, want false", fleet.ReturnFlight)
	}
	if fleet.Origin.Galaxy != 1 {
		t.Errorf("Origin.Galaxy = %d, want 1", fleet.Origin.Galaxy)
	}
	if fleet.Destination.Galaxy != 2 {
		t.Errorf("Destination.Galaxy = %d, want 2", fleet.Destination.Galaxy)
	}
	if fleet.ArrivalTime != 1700000000 {
		t.Errorf("ArrivalTime = %d, want 1700000000", fleet.ArrivalTime)
	}
	if len(fleet.Ships) != 1 {
		t.Fatalf("Ships length = %d, want 1", len(fleet.Ships))
	}
	if fleet.Ships[0].ID != 204 {
		t.Errorf("Ships[0].ID = %d, want 204", fleet.Ships[0].ID)
	}
	if fleet.Ships[0].Count != 100 {
		t.Errorf("Ships[0].Count = %d, want 100", fleet.Ships[0].Count)
	}
	if fleet.Metal != 5000 {
		t.Errorf("Metal = %d, want 5000", fleet.Metal)
	}
}

func TestFleetSlotsJSON(t *testing.T) {
	raw := `{"Total":14,"InUse":3}`

	var slots FleetSlots
	if err := json.Unmarshal([]byte(raw), &slots); err != nil {
		t.Fatalf("failed to unmarshal FleetSlots: %v", err)
	}

	if slots.Total != 14 {
		t.Errorf("Total = %d, want 14", slots.Total)
	}
	if slots.InUse != 3 {
		t.Errorf("InUse = %d, want 3", slots.InUse)
	}
}

func TestResourceBuildingsJSON(t *testing.T) {
	raw := `{"MetalMine":25,"CrystalMine":20,"DeuteriumSynthesizer":18,"SolarPlant":22,"FusionReactor":0,"MetalStorage":10,"CrystalStorage":8,"DeuteriumStorage":7}`

	var b ResourceBuildings
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		t.Fatalf("failed to unmarshal ResourceBuildings: %v", err)
	}

	if b.MetalMine != 25 {
		t.Errorf("MetalMine = %d, want 25", b.MetalMine)
	}
	if b.CrystalMine != 20 {
		t.Errorf("CrystalMine = %d, want 20", b.CrystalMine)
	}
}

func TestFacilitiesJSON(t *testing.T) {
	raw := `{"RoboticsFactory":10,"Shipyard":12,"ResearchLab":8,"AllianceDepot":0,"MissileSilo":0,"NaniteFactory":5,"Terraformer":0,"SpaceDock":0}`

	var f Facilities
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatalf("failed to unmarshal Facilities: %v", err)
	}

	if f.RoboticsFactory != 10 {
		t.Errorf("RoboticsFactory = %d, want 10", f.RoboticsFactory)
	}
	if f.Shipyard != 12 {
		t.Errorf("Shipyard = %d, want 12", f.Shipyard)
	}
	if f.NaniteFactory != 5 {
		t.Errorf("NaniteFactory = %d, want 5", f.NaniteFactory)
	}
}

func TestDefenceJSON(t *testing.T) {
	raw := `{"RocketLauncher":100,"LightLaser":50,"HeavyLaser":10,"GaussCannon":5,"IonCannon":0,"PlasmaTurret":2,"SmallShield":1,"LargeShield":0,"AntiBallisticMissile":20,"InterplanetaryMissile":0}`

	var d Defence
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("failed to unmarshal Defence: %v", err)
	}

	if d.RocketLauncher != 100 {
		t.Errorf("RocketLauncher = %d, want 100", d.RocketLauncher)
	}
	if d.PlasmaTurret != 2 {
		t.Errorf("PlasmaTurret = %d, want 2", d.PlasmaTurret)
	}
}

func TestShipsJSON(t *testing.T) {
	raw := `{"LightFighter":200,"HeavyFighter":50,"Cruiser":30,"Battleship":10,"Battlecruiser":5,"Bomber":3,"Destroyer":2,"Deathstar":1,"SmallCargo":100,"LargeCargo":40,"ColonyShip":1,"Recycler":10,"EspionageProbe":50,"SolarSatellite":200}`

	var s Ships
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("failed to unmarshal Ships: %v", err)
	}

	if s.LightFighter != 200 {
		t.Errorf("LightFighter = %d, want 200", s.LightFighter)
	}
	if s.SmallCargo != 100 {
		t.Errorf("SmallCargo = %d, want 100", s.SmallCargo)
	}
	if s.Deathstar != 1 {
		t.Errorf("Deathstar = %d, want 1", s.Deathstar)
	}
}

func TestResearchJSON(t *testing.T) {
	raw := `{"EnergyTechnology":12,"LaserTechnology":5,"IonTechnology":4,"HyperspaceTechnology":8,"PlasmaTechnology":7,"CombustionDrive":10,"ImpulseDrive":6,"HyperspaceDrive":5,"EspionageTechnology":12,"ComputerTechnology":10,"Astrophysics":18,"IntergalacticResearchNetwork":8,"GravitonTechnology":1,"WeaponTechnology":12,"ShieldingTechnology":10,"ArmourTechnology":10}`

	var r Research
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("failed to unmarshal Research: %v", err)
	}

	if r.EnergyTechnology != 12 {
		t.Errorf("EnergyTechnology = %d, want 12", r.EnergyTechnology)
	}
	if r.Astrophysics != 18 {
		t.Errorf("Astrophysics = %d, want 18", r.Astrophysics)
	}
	if r.GravitonTechnology != 1 {
		t.Errorf("GravitonTechnology = %d, want 1", r.GravitonTechnology)
	}
}

func TestAttackEventJSON(t *testing.T) {
	raw := `{
		"ID": 12345,
		"MissionType": 1,
		"Origin": {"Galaxy":1,"System":50,"Position":8,"Type":"planet"},
		"Destination": {"Galaxy":2,"System":100,"Position":4,"Type":"planet"},
		"DestinationName": "Homeworld",
		"ArrivalTime": "2026-04-26T12:00:00Z",
		"ArriveIn": 3600,
		"AttackerName": "Enemy",
		"AttackerID": 99999,
		"UnionID": 0,
		"Missiles": 0,
		"Ships": [{"ID":204,"Count":500},{"ID":205,"Count":100}]
	}`

	var event AttackEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("failed to unmarshal AttackEvent: %v", err)
	}

	if event.ID != 12345 {
		t.Errorf("ID = %d, want 12345", event.ID)
	}
	if event.MissionType != 1 {
		t.Errorf("MissionType = %d, want 1", event.MissionType)
	}
	if event.Origin.Galaxy != 1 || event.Origin.System != 50 {
		t.Errorf("Origin = %+v, want Galaxy=1 System=50", event.Origin)
	}
	if event.DestinationName != "Homeworld" {
		t.Errorf("DestinationName = %q, want %q", event.DestinationName, "Homeworld")
	}
	if event.ArriveIn != 3600 {
		t.Errorf("ArriveIn = %d, want 3600", event.ArriveIn)
	}
	if event.AttackerName != "Enemy" {
		t.Errorf("AttackerName = %q, want %q", event.AttackerName, "Enemy")
	}
	if event.AttackerID != 99999 {
		t.Errorf("AttackerID = %d, want 99999", event.AttackerID)
	}
	if len(*event.Ships) != 2 {
		t.Fatalf("Ships length = %d, want 2", len(*event.Ships))
	}
	if (*event.Ships)[0].ID != 204 || (*event.Ships)[0].Count != 500 {
		t.Errorf("Ships[0] = %+v, want ID=204 Count=500", (*event.Ships)[0])
	}
}

func TestAttackEvent_NullShips(t *testing.T) {
	raw := `{
		"ID": 99,
		"MissionType": 10,
		"Origin": {"Galaxy":1,"System":1,"Position":1,"Type":"planet"},
		"Destination": {"Galaxy":1,"System":1,"Position":1,"Type":"planet"},
		"DestinationName": "Target",
		"ArrivalTime": "2026-04-26T12:00:00Z",
		"ArriveIn": 600,
		"AttackerName": "Attacker",
		"AttackerID": 100,
		"UnionID": 0,
		"Missiles": 50
	}`

	var event AttackEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("failed to unmarshal AttackEvent with null Ships: %v", err)
	}

	if event.Ships != nil {
		t.Errorf("Ships should be nil when not provided, got %+v", event.Ships)
	}
	if event.Missiles != 50 {
		t.Errorf("Missiles = %d, want 50", event.Missiles)
	}
}

func TestSlotsJSON(t *testing.T) {
	raw := `{"InUse":3,"Total":14,"ExpInUse":1,"ExpTotal":2}`

	var slots Slots
	if err := json.Unmarshal([]byte(raw), &slots); err != nil {
		t.Fatalf("failed to unmarshal Slots: %v", err)
	}

	if slots.InUse != 3 {
		t.Errorf("InUse = %d, want 3", slots.InUse)
	}
	if slots.Total != 14 {
		t.Errorf("Total = %d, want 14", slots.Total)
	}
	if slots.ExpInUse != 1 {
		t.Errorf("ExpInUse = %d, want 1", slots.ExpInUse)
	}
	if slots.ExpTotal != 2 {
		t.Errorf("ExpTotal = %d, want 2", slots.ExpTotal)
	}
}

func TestSendFleetRequest_Fields(t *testing.T) {
	req := SendFleetRequest{
		PlanetID: 336,
		Ships: []ShipCount{
			{ID: 204, Count: 100},
			{ID: 203, Count: 50},
		},
		Speed:     10,
		Galaxy:    2,
		System:    100,
		Position:  8,
		Type:      1,
		Mission:   4,
		Metal:     5000,
		Crystal:   3000,
		Deuterium: 1000,
	}

	if req.PlanetID != 336 {
		t.Errorf("PlanetID = %d, want 336", req.PlanetID)
	}
	if len(req.Ships) != 2 {
		t.Fatalf("Ships length = %d, want 2", len(req.Ships))
	}
	if req.Ships[0].ID != 204 || req.Ships[0].Count != 100 {
		t.Errorf("Ships[0] = %+v, want ID=204 Count=100", req.Ships[0])
	}
	if req.Speed != 10 {
		t.Errorf("Speed = %d, want 10", req.Speed)
	}
	if req.Mission != 4 {
		t.Errorf("Mission = %d, want 4", req.Mission)
	}
	if req.Metal != 5000 {
		t.Errorf("Metal = %d, want 5000", req.Metal)
	}
}
