package ogamex

import (
	"os"
	"strings"
	"testing"

	"github.com/user/ogame-bot/internal/constants"
)

func TestParseAmount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{"12345", 12345},
		{"423.70000001224", 424},
		{"483.93666667891", 484},
		{"12,345", 12345},
		{"500", 500},
		{"0", 0},
		{"", 0},
		{"  ", 0},
	}
	for _, tt := range tests {
		got := parseAmount(tt.input)
		if got != tt.want {
			t.Errorf("parseAmount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParsePlanetList(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/overview.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	planets, err := parsePlanetList(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parsePlanetList error: %v", err)
	}
	if len(planets) != 2 {
		t.Fatalf("expected 2 planets, got %d", len(planets))
	}
	p := planets[0]
	if p.ID != 12345 {
		t.Errorf("planet[0] ID = %d, want 12345", p.ID)
	}
	if p.Name != "Homeworld" {
		t.Errorf("planet[0] Name = %q, want %q", p.Name, "Homeworld")
	}
	if p.IsMoon {
		t.Errorf("planet[0] should not be moon")
	}
	if p.Coordinate.Galaxy != 1 || p.Coordinate.System != 2 || p.Coordinate.Position != 3 {
		t.Errorf("planet[0] coords = %v, want [1:2:3]", p.Coordinate)
	}
	m := planets[1]
	if m.ID != 67890 {
		t.Errorf("planet[1] ID = %d, want 67890", m.ID)
	}
	if !m.IsMoon {
		t.Errorf("planet[1] should be moon")
	}
}

func TestParseResources(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/overview.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	res, err := parseResources(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parseResources error: %v", err)
	}
	if res.Metal != 12345 {
		t.Errorf("Metal = %d, want 12345", res.Metal)
	}
	if res.Crystal != 6789 {
		t.Errorf("Crystal = %d, want 6789", res.Crystal)
	}
	if res.Deuterium != 2345 {
		t.Errorf("Deuterium = %d, want 2345", res.Deuterium)
	}
	if res.Energy != 500 {
		t.Errorf("Energy = %d, want 500", res.Energy)
	}
}

func TestParseResourceBuildings(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/resources.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	b, err := parseResourceBuildings(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parseResourceBuildings error: %v", err)
	}
	if b.MetalMine != 25 {
		t.Errorf("MetalMine = %d, want 25", b.MetalMine)
	}
	if b.CrystalMine != 22 {
		t.Errorf("CrystalMine = %d, want 22", b.CrystalMine)
	}
	if b.DeuteriumSynthesizer != 18 {
		t.Errorf("DeuteriumSynthesizer = %d, want 18", b.DeuteriumSynthesizer)
	}
	if b.SolarPlant != 20 {
		t.Errorf("SolarPlant = %d, want 20", b.SolarPlant)
	}
	if b.FusionReactor != 5 {
		t.Errorf("FusionReactor = %d, want 5", b.FusionReactor)
	}
	if b.MetalStorage != 10 {
		t.Errorf("MetalStorage = %d, want 10", b.MetalStorage)
	}
	if b.CrystalStorage != 8 {
		t.Errorf("CrystalStorage = %d, want 8", b.CrystalStorage)
	}
	if b.DeuteriumStorage != 7 {
		t.Errorf("DeuteriumStorage = %d, want 7", b.DeuteriumStorage)
	}
}

func TestParseFacilities(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/facilities.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	f, err := parseFacilities(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parseFacilities error: %v", err)
	}
	if f.RoboticsFactory != 10 {
		t.Errorf("RoboticsFactory = %d, want 10", f.RoboticsFactory)
	}
	if f.Shipyard != 8 {
		t.Errorf("Shipyard = %d, want 8", f.Shipyard)
	}
	if f.ResearchLab != 7 {
		t.Errorf("ResearchLab = %d, want 7", f.ResearchLab)
	}
	if f.AllianceDepot != 0 {
		t.Errorf("AllianceDepot = %d, want 0", f.AllianceDepot)
	}
	if f.MissileSilo != 3 {
		t.Errorf("MissileSilo = %d, want 3", f.MissileSilo)
	}
	if f.NaniteFactory != 1 {
		t.Errorf("NaniteFactory = %d, want 1", f.NaniteFactory)
	}
	if f.Terraformer != 0 {
		t.Errorf("Terraformer = %d, want 0", f.Terraformer)
	}
	if f.SpaceDock != 0 {
		t.Errorf("SpaceDock = %d, want 0", f.SpaceDock)
	}
}

func TestParseShips(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/shipyard.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	s, err := parseShips(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parseShips error: %v", err)
	}
	if s.SmallCargo != 100 {
		t.Errorf("SmallCargo = %d, want 100", s.SmallCargo)
	}
	if s.LargeCargo != 50 {
		t.Errorf("LargeCargo = %d, want 50", s.LargeCargo)
	}
	if s.LightFighter != 200 {
		t.Errorf("LightFighter = %d, want 200", s.LightFighter)
	}
	if s.HeavyFighter != 30 {
		t.Errorf("HeavyFighter = %d, want 30", s.HeavyFighter)
	}
	if s.Cruiser != 40 {
		t.Errorf("Cruiser = %d, want 40", s.Cruiser)
	}
	if s.Battleship != 15 {
		t.Errorf("Battleship = %d, want 15", s.Battleship)
	}
	if s.ColonyShip != 1 {
		t.Errorf("ColonyShip = %d, want 1", s.ColonyShip)
	}
	if s.Recycler != 10 {
		t.Errorf("Recycler = %d, want 10", s.Recycler)
	}
	if s.EspionageProbe != 50 {
		t.Errorf("EspionageProbe = %d, want 50", s.EspionageProbe)
	}
	if s.Bomber != 5 {
		t.Errorf("Bomber = %d, want 5", s.Bomber)
	}
	if s.SolarSatellite != 20 {
		t.Errorf("SolarSatellite = %d, want 20", s.SolarSatellite)
	}
	if s.Destroyer != 3 {
		t.Errorf("Destroyer = %d, want 3", s.Destroyer)
	}
	if s.Deathstar != 0 {
		t.Errorf("Deathstar = %d, want 0", s.Deathstar)
	}
	if s.Battlecruiser != 8 {
		t.Errorf("Battlecruiser = %d, want 8", s.Battlecruiser)
	}
}

func TestParseDefence(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/defense.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	d, err := parseDefence(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parseDefence error: %v", err)
	}
	if d.RocketLauncher != 500 {
		t.Errorf("RocketLauncher = %d, want 500", d.RocketLauncher)
	}
	if d.LightLaser != 200 {
		t.Errorf("LightLaser = %d, want 200", d.LightLaser)
	}
	if d.HeavyLaser != 50 {
		t.Errorf("HeavyLaser = %d, want 50", d.HeavyLaser)
	}
	if d.GaussCannon != 20 {
		t.Errorf("GaussCannon = %d, want 20", d.GaussCannon)
	}
	if d.IonCannon != 10 {
		t.Errorf("IonCannon = %d, want 10", d.IonCannon)
	}
	if d.PlasmaTurret != 5 {
		t.Errorf("PlasmaTurret = %d, want 5", d.PlasmaTurret)
	}
	if d.SmallShield != 1 {
		t.Errorf("SmallShield = %d, want 1", d.SmallShield)
	}
	if d.LargeShield != 1 {
		t.Errorf("LargeShield = %d, want 1", d.LargeShield)
	}
	if d.AntiBallisticMissile != 30 {
		t.Errorf("AntiBallisticMissile = %d, want 30", d.AntiBallisticMissile)
	}
	if d.InterplanetaryMissile != 10 {
		t.Errorf("InterplanetaryMissile = %d, want 10", d.InterplanetaryMissile)
	}
}

func TestParseResearch(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/research.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	r, err := parseResearch(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parseResearch error: %v", err)
	}
	if r.EnergyTechnology != 12 {
		t.Errorf("EnergyTechnology = %d, want 12", r.EnergyTechnology)
	}
	if r.LaserTechnology != 5 {
		t.Errorf("LaserTechnology = %d, want 5", r.LaserTechnology)
	}
	if r.IonTechnology != 3 {
		t.Errorf("IonTechnology = %d, want 3", r.IonTechnology)
	}
	if r.HyperspaceTechnology != 7 {
		t.Errorf("HyperspaceTechnology = %d, want 7", r.HyperspaceTechnology)
	}
	if r.PlasmaTechnology != 8 {
		t.Errorf("PlasmaTechnology = %d, want 8", r.PlasmaTechnology)
	}
	if r.CombustionDrive != 14 {
		t.Errorf("CombustionDrive = %d, want 14", r.CombustionDrive)
	}
	if r.ImpulseDrive != 10 {
		t.Errorf("ImpulseDrive = %d, want 10", r.ImpulseDrive)
	}
	if r.HyperspaceDrive != 6 {
		t.Errorf("HyperspaceDrive = %d, want 6", r.HyperspaceDrive)
	}
	if r.EspionageTechnology != 8 {
		t.Errorf("EspionageTechnology = %d, want 8", r.EspionageTechnology)
	}
	if r.ComputerTechnology != 10 {
		t.Errorf("ComputerTechnology = %d, want 10", r.ComputerTechnology)
	}
	if r.Astrophysics != 9 {
		t.Errorf("Astrophysics = %d, want 9", r.Astrophysics)
	}
	if r.IntergalacticResearchNetwork != 1 {
		t.Errorf("IntergalacticResearchNetwork = %d, want 1", r.IntergalacticResearchNetwork)
	}
	if r.GravitonTechnology != 0 {
		t.Errorf("GravitonTechnology = %d, want 0", r.GravitonTechnology)
	}
	if r.WeaponTechnology != 11 {
		t.Errorf("WeaponTechnology = %d, want 11", r.WeaponTechnology)
	}
	if r.ShieldingTechnology != 9 {
		t.Errorf("ShieldingTechnology = %d, want 9", r.ShieldingTechnology)
	}
	if r.ArmourTechnology != 10 {
		t.Errorf("ArmourTechnology = %d, want 10", r.ArmourTechnology)
	}
}

func TestParseEventbox(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/eventbox.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	box, err := parseEventbox(data)
	if err != nil {
		t.Fatalf("parseEventbox error: %v", err)
	}
	if box.Hostile != 3 {
		t.Errorf("Hostile = %d, want 3", box.Hostile)
	}
	if box.Friendly != 5 {
		t.Errorf("Friendly = %d, want 5", box.Friendly)
	}
	if box.ServerTime != "2026-05-03T12:00:00+0000" {
		t.Errorf("ServerTime = %q, want %q", box.ServerTime, "2026-05-03T12:00:00+0000")
	}
}

func TestParseFleetEvents(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/eventlist.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	fleets, err := parseFleetEvents(data)
	if err != nil {
		t.Fatalf("parseFleetEvents error: %v", err)
	}
	if len(fleets) != 3 {
		t.Fatalf("expected 3 fleets, got %d", len(fleets))
	}
	f := fleets[0]
	if f.ID != 1001 {
		t.Errorf("fleet[0] ID = %d, want 1001", f.ID)
	}
	if f.Mission != constants.MissionAttack {
		t.Errorf("fleet[0] Mission = %d, want %d", f.Mission, constants.MissionAttack)
	}
	if f.ReturnFlight {
		t.Errorf("fleet[0] should not be return flight")
	}
	if f.Origin.Galaxy != 2 || f.Origin.System != 3 || f.Origin.Position != 4 {
		t.Errorf("fleet[0] Origin = %v, want [2:3:4]", f.Origin)
	}
	if f.Destination.Galaxy != 1 || f.Destination.System != 2 || f.Destination.Position != 3 {
		t.Errorf("fleet[0] Destination = %v, want [1:2:3]", f.Destination)
	}
	if f.ArrivalTime != 1700000000 {
		t.Errorf("fleet[0] ArrivalTime = %d, want 1700000000", f.ArrivalTime)
	}

	f2 := fleets[1]
	if f2.Mission != constants.MissionTransport {
		t.Errorf("fleet[1] Mission = %d, want %d", f2.Mission, constants.MissionTransport)
	}

	f3 := fleets[2]
	if f3.Mission != constants.MissionHarvest {
		t.Errorf("fleet[2] Mission = %d, want %d", f3.Mission, constants.MissionHarvest)
	}
	if !f3.ReturnFlight {
		t.Errorf("fleet[2] should be return flight")
	}
}

func TestParseServerInfo(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/overview.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	speed, version := parseServerInfo(data)
	if speed != 7 {
		t.Errorf("speed = %d, want 7", speed)
	}
	if version == "unknown" {
		t.Errorf("version should not be 'unknown', got %q", version)
	}
}

func TestParseCoordinate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		g     int
		s     int
		p     int
	}{
		{"1:2:3", 1, 2, 3},
		{"10:200:5", 10, 200, 5},
		{"", 0, 0, 0},
	}
	for _, tt := range tests {
		c := parseCoordinate(tt.input)
		if c.Galaxy != tt.g || c.System != tt.s || c.Position != tt.p {
			t.Errorf("parseCoordinate(%q) = [%d:%d:%d], want [%d:%d:%d]",
				tt.input, c.Galaxy, c.System, c.Position, tt.g, tt.s, tt.p)
		}
	}
}

func TestParseConstructions(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("fixtures/resources.html")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	c, err := parseConstructions(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parseConstructions error: %v", err)
	}
	if c.Building.Countdown != 12345 {
		t.Errorf("Building.Countdown = %d, want 12345", c.Building.Countdown)
	}
}
