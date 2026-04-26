package defender

import (
	"math"
	"testing"
	"time"

	"github.com/user/ogame-bot/internal/model"
)

// ---------- CalcDistance ----------

func TestCalcDistance(t *testing.T) {
	tests := []struct {
		name string
		from model.Coordinate
		to   model.Coordinate
		want int
	}{
		{
			name: "same system different positions",
			from: model.Coordinate{Galaxy: 1, System: 100, Position: 3},
			to:   model.Coordinate{Galaxy: 1, System: 100, Position: 8},
			want: 5,
		},
		{
			name: "same system same position",
			from: model.Coordinate{Galaxy: 1, System: 50, Position: 5},
			to:   model.Coordinate{Galaxy: 1, System: 50, Position: 5},
			want: 0,
		},
		{
			name: "same galaxy different system",
			from: model.Coordinate{Galaxy: 1, System: 100, Position: 3},
			to:   model.Coordinate{Galaxy: 1, System: 110, Position: 8},
			want: 55, // 5*abs(100-110) + abs(3-8) = 50+5
		},
		{
			name: "different galaxy",
			from: model.Coordinate{Galaxy: 1, System: 100, Position: 3},
			to:   model.Coordinate{Galaxy: 3, System: 200, Position: 8},
			want: 40000, // 20000*abs(1-3)
		},
		{
			name: "adjacent systems same position",
			from: model.Coordinate{Galaxy: 2, System: 50, Position: 10},
			to:   model.Coordinate{Galaxy: 2, System: 51, Position: 10},
			want: 5, // 5*1 + 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcDistance(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CalcDistance(%v, %v) = %d, want %d", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// ---------- Ship stats database ----------

func TestShipStatsDB(t *testing.T) {
	// Verify all 13 mobile ship types exist (SolarSatellite excluded)
	expectedShips := map[int]struct {
		baseSpeed   int
		baseFuel    int
		driveType   string
		cargoCap    int
	}{
		202: {5000, 10, "combustion", 5000},       // SmallCargo
		203: {7500, 50, "combustion", 25000},       // LargeCargo
		204: {12500, 20, "combustion", 50},         // LightFighter
		205: {10000, 75, "impulse", 100},           // HeavyFighter
		206: {15000, 300, "impulse", 800},          // Cruiser
		207: {10000, 500, "hyperspace", 1500},      // Battleship
		208: {2500, 1000, "impulse", 7500},         // ColonyShip
		209: {2000, 300, "combustion", 20000},      // Recycler
		210: {100000000, 1, "combustion", 0},       // EspionageProbe
		211: {4000, 1000, "impulse", 500},          // Bomber
		213: {5000, 1000, "hyperspace", 2000},      // Destroyer
		214: {100, 1, "hyperspace", 1000000},       // Deathstar
		215: {10000, 250, "hyperspace", 750},       // Battlecruiser
	}

	for shipID, expected := range expectedShips {
		stats, ok := shipDB[shipID]
		if !ok {
			t.Errorf("shipDB missing ship ID %d", shipID)
			continue
		}
		if stats.BaseSpeed != expected.baseSpeed {
			t.Errorf("shipDB[%d].BaseSpeed = %d, want %d", shipID, stats.BaseSpeed, expected.baseSpeed)
		}
		if stats.BaseFuel != expected.baseFuel {
			t.Errorf("shipDB[%d].BaseFuel = %d, want %d", shipID, stats.BaseFuel, expected.baseFuel)
		}
		if stats.DriveType != expected.driveType {
			t.Errorf("shipDB[%d].DriveType = %q, want %q", shipID, stats.DriveType, expected.driveType)
		}
		if stats.CargoCapacity != expected.cargoCap {
			t.Errorf("shipDB[%d].CargoCapacity = %d, want %d", shipID, stats.CargoCapacity, expected.cargoCap)
		}
	}

	// SolarSatellite should NOT be in the database
	if _, ok := shipDB[212]; ok {
		t.Error("shipDB should not contain SolarSatellite (212)")
	}
}

// ---------- effectiveSpeed ----------

func TestEffectiveSpeed(t *testing.T) {
	tests := []struct {
		name       string
		baseSpeed  int
		driveType  string
		driveLevel int
		want       int64
	}{
		{
			name:       "combustion drive level 10",
			baseSpeed:  5000,
			driveType:  "combustion",
			driveLevel: 10,
			want:       10000, // 5000 * (1 + 0.1*10) = 5000 * 2 = 10000
		},
		{
			name:       "impulse drive level 5",
			baseSpeed:  10000,
			driveType:  "impulse",
			driveLevel: 5,
			want:       20000, // 10000 * (1 + 0.2*5) = 10000 * 2 = 20000
		},
		{
			name:       "hyperspace drive level 8",
			baseSpeed:  10000,
			driveType:  "hyperspace",
			driveLevel: 8,
			want:       34000, // 10000 * (1 + 0.3*8) = 10000 * 3.4 = 34000
		},
		{
			name:       "zero drive level returns base speed",
			baseSpeed:  5000,
			driveType:  "combustion",
			driveLevel: 0,
			want:       5000, // 5000 * (1 + 0) = 5000
		},
		{
			name:       "small cargo impulse upgrade at impulse 5",
			baseSpeed:  10000, // impulse base speed for small cargo
			driveType:  "impulse",
			driveLevel: 5,
			want:       20000, // 10000 * (1 + 0.2*5) = 20000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveSpeed(tt.baseSpeed, tt.driveType, tt.driveLevel)
			if got != tt.want {
				t.Errorf("effectiveSpeed(%d, %q, %d) = %d, want %d",
					tt.baseSpeed, tt.driveType, tt.driveLevel, got, tt.want)
			}
		})
	}
}

// ---------- fuelConsumption ----------

func TestFuelConsumption(t *testing.T) {
	tests := []struct {
		name     string
		distance int
		speed    int // 1-10
		ships    model.Ships
		research model.Research
		wantGt   int64 // fuel should be > 0 and reasonable
		wantZero bool  // if true, expect 0
	}{
		{
			name:     "single small cargo short trip",
			distance: 1000,
			speed:    10,
			ships:    model.Ships{SmallCargo: 10},
			research: model.Research{CombustionDrive: 10},
			wantGt:   1,
		},
		{
			name:     "no ships returns zero fuel",
			distance: 100,
			speed:    10,
			ships:    model.Ships{},
			research: model.Research{},
			wantZero: true,
		},
		{
			name:     "fleet with multiple ship types",
			distance: 50,
			speed:    10,
			ships:    model.Ships{SmallCargo: 100, LargeCargo: 50},
			research: model.Research{CombustionDrive: 10},
			wantGt:   100,
		},
		{
			name:     "low speed uses less fuel than high speed",
			distance: 100,
			speed:    5,
			ships:    model.Ships{SmallCargo: 100},
			research: model.Research{CombustionDrive: 10},
			wantGt:   1, // just check it's positive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuelConsumption(tt.distance, tt.speed, tt.ships, tt.research)
			if tt.wantZero {
				if got != 0 {
					t.Errorf("fuelConsumption() = %d, want 0", got)
				}
				return
			}
			if got <= 0 {
				t.Errorf("fuelConsumption() = %d, want > 0", got)
			}
		})
	}
}

func TestFuelConsumption_SpeedScaling(t *testing.T) {
	// Higher speed should cost more fuel for same distance
	ships := model.Ships{SmallCargo: 100}
	research := model.Research{CombustionDrive: 10}
	distance := 5000

	fuelSpeed5 := fuelConsumption(distance, 5, ships, research)
	fuelSpeed10 := fuelConsumption(distance, 10, ships, research)

	if fuelSpeed10 <= fuelSpeed5 {
		t.Errorf("fuel at speed 10 (%d) should be > fuel at speed 5 (%d)", fuelSpeed10, fuelSpeed5)
	}
}

// ---------- flightDuration ----------

func TestFlightDuration(t *testing.T) {
	tests := []struct {
		name        string
		distance    int
		speed       int
		ships       model.Ships
		research    model.Research
		wantGt      time.Duration // minimum expected duration
		wantLess    time.Duration // maximum expected duration
		wantZero    bool
	}{
		{
			name:     "short flight with fast ship",
			distance: 10,
			speed:    10,
			ships:    model.Ships{LightFighter: 1},
			research: model.Research{CombustionDrive: 10},
			wantGt:   0,              // must be positive
			wantLess: 10 * time.Hour, // should be under 10h for short distance
		},
		{
			name:     "no ships returns zero duration",
			distance: 100,
			speed:    10,
			ships:    model.Ships{},
			research: model.Research{},
			wantZero: true,
		},
		{
			name:     "slow ship takes longer than fast ship",
			distance: 1000,
			speed:    10,
			ships:    model.Ships{Deathstar: 1},
			research: model.Research{HyperspaceDrive: 0},
			wantGt:   10 * time.Minute, // deathstar is very slow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := model.Coordinate{Galaxy: 1, System: 100, Position: 1}
			to := model.Coordinate{Galaxy: 1, System: 100, Position: 1} // coords don't matter when distance is given directly

			got := flightDuration(tt.distance, tt.speed, from, to, tt.ships, tt.research)
			if tt.wantZero {
				if got != 0 {
					t.Errorf("flightDuration() = %v, want 0", got)
				}
				return
			}
			if got <= tt.wantGt {
				t.Errorf("flightDuration() = %v, want > %v", got, tt.wantGt)
			}
			if tt.wantLess > 0 && got > tt.wantLess {
				t.Errorf("flightDuration() = %v, want < %v", got, tt.wantLess)
			}
		})
	}
}

func TestFlightDuration_SlowestDeterminesDuration(t *testing.T) {
	// A fleet with a deathstar + light fighters should fly at deathstar speed
	fleetSlow := model.Ships{Deathstar: 1, LightFighter: 100}
	fleetFast := model.Ships{LightFighter: 100}
	research := model.Research{CombustionDrive: 10, HyperspaceDrive: 5}

	from := model.Coordinate{Galaxy: 1, System: 100, Position: 1}
	to := model.Coordinate{Galaxy: 1, System: 100, Position: 1}

	durationSlow := flightDuration(100, 10, from, to, fleetSlow, research)
	durationFast := flightDuration(100, 10, from, to, fleetFast, research)

	if durationSlow <= durationFast {
		t.Errorf("fleet with deathstar (%v) should be slower than light fighters only (%v)",
			durationSlow, durationFast)
	}
}

func TestFlightDuration_LowerSpeedTakesLonger(t *testing.T) {
	ships := model.Ships{SmallCargo: 10}
	research := model.Research{CombustionDrive: 10}
	distance := 50

	from := model.Coordinate{Galaxy: 1, System: 1, Position: 1}
	to := model.Coordinate{Galaxy: 1, System: 1, Position: 1}

	durSpeed5 := flightDuration(distance, 5, from, to, ships, research)
	durSpeed10 := flightDuration(distance, 10, from, to, ships, research)

	// Lower speed should take longer
	if durSpeed5 <= durSpeed10 {
		t.Errorf("speed 5 duration (%v) should be > speed 10 duration (%v)", durSpeed5, durSpeed10)
	}

	// At half speed, duration should be roughly double
	ratio := float64(durSpeed5) / float64(durSpeed10)
	if math.Abs(ratio-2.0) > 0.5 {
		t.Errorf("speed 5 / speed 10 ratio = %.2f, expected ~2.0", ratio)
	}
}

// ---------- CalcEscapeRoutes ----------

func TestCalcEscapeRoutes(t *testing.T) {
	origin := model.Planet{
		ID:   1,
		Name: "Homeworld",
		Coordinate: model.Coordinate{
			Galaxy:   1,
			System:   100,
			Position: 3,
			Type:     "planet",
		},
		IsMoon: false,
	}

	destPlanet := model.Planet{
		ID:   2,
		Name: "Colony",
		Coordinate: model.Coordinate{
			Galaxy:   1,
			System:   150,
			Position: 8,
			Type:     "planet",
		},
		IsMoon: false,
	}

	destMoon := model.Planet{
		ID:   3,
		Name: "Homeworld Moon",
		Coordinate: model.Coordinate{
			Galaxy:   1,
			System:   100,
			Position: 3,
			Type:     "moon",
		},
		IsMoon: true,
	}

	ships := model.Ships{SmallCargo: 100, LargeCargo: 20}
	resources := model.Resources{Metal: 500000, Crystal: 200000, Deuterium: 100000}
	research := model.Research{CombustionDrive: 12}

	t.Run("generates routes to all own planets at multiple speeds", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet, destMoon}, nil, research)
		if len(routes) == 0 {
			t.Fatal("expected routes, got empty slice")
		}
		// Should have routes for different speed settings to multiple destinations
		for _, r := range routes {
			if r.Speed < 1 || r.Speed > 10 {
				t.Errorf("route speed %d out of range [1,10]", r.Speed)
			}
			if r.FuelCost < 0 {
				t.Errorf("route fuel cost %d should be >= 0", r.FuelCost)
			}
			if r.Duration <= 0 {
				t.Errorf("route duration %v should be > 0", r.Duration)
			}
		}
	})

	t.Run("routes filtered by fuel sufficiency", func(t *testing.T) {
		// Very little deuterium — only the cheapest routes should survive
		lowDeut := model.Resources{Metal: 500000, Crystal: 200000, Deuterium: 5}
		routes := CalcEscapeRoutes(origin, ships, lowDeut, []model.Planet{destPlanet}, nil, research)
		for _, r := range routes {
			if r.FuelCost > 5 {
				t.Errorf("route fuel cost %d exceeds available deuterium 5", r.FuelCost)
			}
		}
	})

	t.Run("destination under attack is penalized", func(t *testing.T) {
		attacks := []model.AttackEvent{
			{
				ID:          1,
				MissionType: 1,
				Destination: destPlanet.Coordinate,
				ArriveIn:    600,
			},
		}

		routesNoAttack := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet, destMoon}, nil, research)
		routesWithAttack := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet, destMoon}, attacks, research)

		if len(routesNoAttack) == 0 || len(routesWithAttack) == 0 {
			t.Fatal("expected routes in both cases")
		}

		// Find routes to destPlanet and compare safety scores
		var scoreNoAttack, scoreWithAttack int
		for _, r := range routesNoAttack {
			if r.DestPlanetID == destPlanet.ID {
				scoreNoAttack = r.SafetyScore
				break
			}
		}
		for _, r := range routesWithAttack {
			if r.DestPlanetID == destPlanet.ID {
				scoreWithAttack = r.SafetyScore
				break
			}
		}

		if scoreWithAttack <= scoreNoAttack {
			t.Errorf("attacked destination score %d should be > safe score %d", scoreWithAttack, scoreNoAttack)
		}
	})

	t.Run("empty fleet returns empty routes", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, model.Ships{}, resources, []model.Planet{destPlanet}, nil, research)
		if len(routes) != 0 {
			t.Errorf("expected 0 routes for empty fleet, got %d", len(routes))
		}
	})

	t.Run("no destinations returns empty slice without error", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, ships, resources, nil, nil, research)
		if routes == nil {
			t.Error("expected empty slice, not nil")
		}
		if len(routes) != 0 {
			t.Errorf("expected 0 routes with no destinations, got %d", len(routes))
		}
	})

	t.Run("routes sorted by safety score ascending", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet, destMoon}, nil, research)
		if len(routes) < 2 {
			t.Skip("need at least 2 routes to verify sorting")
		}
		for i := 1; i < len(routes); i++ {
			if routes[i].SafetyScore < routes[i-1].SafetyScore {
				t.Errorf("routes not sorted: route[%d].SafetyScore=%d < route[%d].SafetyScore=%d",
					i, routes[i].SafetyScore, i-1, routes[i-1].SafetyScore)
			}
		}
	})

	t.Run("fuel deducted from available deuterium", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet}, nil, research)
		if len(routes) == 0 {
			t.Fatal("expected at least one route")
		}
		for _, r := range routes {
			expectedDeut := int64(resources.Deuterium) - r.FuelCost
			if expectedDeut < 0 {
				expectedDeut = 0
			}
			if r.DeutLoad != expectedDeut {
				t.Errorf("DeutLoad = %d, want %d (deuterium %d - fuel %d)",
					r.DeutLoad, expectedDeut, resources.Deuterium, r.FuelCost)
			}
		}
	})

	t.Run("deploy mission used for all routes", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet}, nil, research)
		for _, r := range routes {
			if r.Mission != 3 { // MissionDeploy
				t.Errorf("route mission = %d, want 3 (deploy)", r.Mission)
			}
		}
	})

	t.Run("speed 10 preferred when fuel allows", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet}, nil, research)
		if len(routes) == 0 {
			t.Fatal("expected routes")
		}
		// The best route (first) should be at speed 10 if fuel allows
		// This tests that higher speeds are preferred in generation
		hasSpeed10 := false
		for _, r := range routes {
			if r.Speed == 10 {
				hasSpeed10 = true
				break
			}
		}
		if !hasSpeed10 {
			t.Error("expected at least one route at speed 10 when fuel is sufficient")
		}
	})

	t.Run("moon origin planet produces routes with correct destination types", func(t *testing.T) {
		moonOrigin := model.Planet{
			ID:   10,
			Name: "My Moon",
			Coordinate: model.Coordinate{
				Galaxy:   1,
				System:   100,
				Position: 3,
				Type:     "moon",
			},
			IsMoon: true,
		}
		routes := CalcEscapeRoutes(moonOrigin, ships, resources, []model.Planet{destPlanet}, nil, research)
		if len(routes) == 0 {
			t.Fatal("expected routes from moon origin")
		}
		// Destination should use its actual coordinate type
		for _, r := range routes {
			if r.DestPlanetID == destPlanet.ID {
				if r.Dest.Type != "planet" {
					t.Errorf("moon-to-planet route dest type = %q, want %q", r.Dest.Type, "planet")
				}
			}
		}
	})

	t.Run("origin skipped as destination", func(t *testing.T) {
		// Include origin in ownPlanets — should be skipped
		routes := CalcEscapeRoutes(origin, ships, resources, []model.Planet{origin, destPlanet}, nil, research)
		for _, r := range routes {
			if r.DestPlanetID == origin.ID {
				t.Error("routes should not include origin as destination")
			}
		}
	})

	t.Run("moon destination has lower safety score than planet", func(t *testing.T) {
		routes := CalcEscapeRoutes(origin, ships, resources, []model.Planet{destPlanet, destMoon}, nil, research)
		if len(routes) == 0 {
			t.Fatal("expected routes")
		}

		var planetScore, moonScore int
		var foundPlanet, foundMoon bool
		for _, r := range routes {
			if r.DestPlanetID == destPlanet.ID && r.Speed == 10 {
				planetScore = r.SafetyScore
				foundPlanet = true
			}
			if r.DestPlanetID == destMoon.ID && r.Speed == 10 {
				moonScore = r.SafetyScore
				foundMoon = true
			}
		}

		if !foundPlanet || !foundMoon {
			t.Skip("could not find both planet and moon routes at speed 10")
		}

		if moonScore >= planetScore {
			t.Errorf("moon destination score %d should be < planet destination score %d", moonScore, planetScore)
		}
	})
}
