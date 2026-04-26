// Package defender implements the fleet-save escape route calculator.
// It evaluates all possible fleet-save destinations and ranks them by safety.
package defender

import (
	"math"
	"sort"
	"time"

	"github.com/user/ogame-bot/internal/model"
)

// shipStats holds the base statistics for a ship type.
type shipStats struct {
	BaseSpeed     int
	BaseFuel      int
	DriveType     string // "combustion", "impulse", or "hyperspace"
	CargoCapacity int
}

// shipDB contains base statistics for all mobile ship types.
// SolarSatellite (212) is excluded as it is not a mobile ship.
var shipDB = map[int]shipStats{
	202: {BaseSpeed: 5000, BaseFuel: 10, DriveType: "combustion", CargoCapacity: 5000},    // SmallCargo
	203: {BaseSpeed: 7500, BaseFuel: 50, DriveType: "combustion", CargoCapacity: 25000},   // LargeCargo
	204: {BaseSpeed: 12500, BaseFuel: 20, DriveType: "combustion", CargoCapacity: 50},     // LightFighter
	205: {BaseSpeed: 10000, BaseFuel: 75, DriveType: "impulse", CargoCapacity: 100},       // HeavyFighter
	206: {BaseSpeed: 15000, BaseFuel: 300, DriveType: "impulse", CargoCapacity: 800},      // Cruiser
	207: {BaseSpeed: 10000, BaseFuel: 500, DriveType: "hyperspace", CargoCapacity: 1500},  // Battleship
	208: {BaseSpeed: 2500, BaseFuel: 1000, DriveType: "impulse", CargoCapacity: 7500},     // ColonyShip
	209: {BaseSpeed: 2000, BaseFuel: 300, DriveType: "combustion", CargoCapacity: 20000},  // Recycler
	210: {BaseSpeed: 100000000, BaseFuel: 1, DriveType: "combustion", CargoCapacity: 0},   // EspionageProbe
	211: {BaseSpeed: 4000, BaseFuel: 1000, DriveType: "impulse", CargoCapacity: 500},      // Bomber
	213: {BaseSpeed: 5000, BaseFuel: 1000, DriveType: "hyperspace", CargoCapacity: 2000},  // Destroyer
	214: {BaseSpeed: 100, BaseFuel: 1, DriveType: "hyperspace", CargoCapacity: 1000000},   // Deathstar
	215: {BaseSpeed: 10000, BaseFuel: 250, DriveType: "hyperspace", CargoCapacity: 750},   // Battlecruiser
}

// universeSpeed is the speed multiplier of the universe (default 1x).
// Will be configurable later.
const universeSpeed = 1

// CalcDistance computes the distance between two coordinates using OGame rules:
//   - Same system: abs(pos1 - pos2)
//   - Same galaxy, different system: 5*abs(sys1-sys2) + abs(pos1-pos2)
//   - Different galaxy: 20000*abs(gal1-gal2)
func CalcDistance(from, to model.Coordinate) int {
	if from.Galaxy != to.Galaxy {
		return 20000 * abs(from.Galaxy-to.Galaxy)
	}
	if from.System != to.System {
		return 5*abs(from.System-to.System) + abs(from.Position-to.Position)
	}
	return abs(from.Position - to.Position)
}

// effectiveSpeed calculates the effective speed of a ship after applying
// drive technology bonuses:
//   - Combustion: baseSpeed * (1 + 0.1*level)
//   - Impulse: baseSpeed * (1 + 0.2*level)
//   - Hyperspace: baseSpeed * (1 + 0.3*level)
func effectiveSpeed(baseSpeed int, driveType string, driveLevel int) int64 {
	var multiplier float64
	switch driveType {
	case "combustion":
		multiplier = 1.0 + 0.1*float64(driveLevel)
	case "impulse":
		multiplier = 1.0 + 0.2*float64(driveLevel)
	case "hyperspace":
		multiplier = 1.0 + 0.3*float64(driveLevel)
	default:
		return int64(baseSpeed)
	}
	return int64(float64(baseSpeed) * multiplier)
}

// getDriveLevel returns the appropriate drive tech level for a given drive type.
func getDriveLevel(driveType string, research model.Research) int {
	switch driveType {
	case "combustion":
		return research.CombustionDrive
	case "impulse":
		return research.ImpulseDrive
	case "hyperspace":
		return research.HyperspaceDrive
	default:
		return 0
	}
}

// getShipEffectiveStats returns the effective speed and fuel for a ship type,
// handling special drive upgrades (Small Cargo at impulse 5, Recycler at impulse 17).
func getShipEffectiveStats(shipID int, research model.Research) (int64, int) {
	stats, ok := shipDB[shipID]
	if !ok {
		return 0, 0
	}

	baseSpeed := stats.BaseSpeed
	driveType := stats.DriveType
	baseFuel := stats.BaseFuel

	// SmallCargo: switches to impulse drive at impulse level 5 (base speed becomes 10000)
	if shipID == 202 && research.ImpulseDrive >= 5 {
		baseSpeed = 10000
		driveType = "impulse"
	}

	// Recycler: switches to impulse drive at impulse level 17 (base speed becomes 4000)
	if shipID == 209 && research.ImpulseDrive >= 17 {
		baseSpeed = 4000
		driveType = "impulse"
	}

	// Bomber: switches to hyperspace drive at hyperspace level 8 (base speed becomes 6000)
	if shipID == 211 && research.HyperspaceDrive >= 8 {
		baseSpeed = 6000
		driveType = "hyperspace"
	}

	level := getDriveLevel(driveType, research)
	speed := effectiveSpeed(baseSpeed, driveType, level)

	return speed, baseFuel
}

// slowestShipSpeed returns the effective speed of the slowest ship in the fleet.
func slowestShipSpeed(ships model.Ships, research model.Research) int64 {
	var slowest int64 = math.MaxInt64
	anyShip := false

	shipCounts := shipsToSlice(ships)
	for _, sc := range shipCounts {
		if sc.Count <= 0 {
			continue
		}
		speed, _ := getShipEffectiveStats(sc.ID, research)
		if speed > 0 && speed < slowest {
			slowest = speed
			anyShip = true
		}
	}

	if !anyShip {
		return 0
	}
	return slowest
}

// flightDuration calculates how long a fleet takes to travel the given distance.
// OGame formula: flightTime = round((3500 / speed * sqrt(distance * 10 / slowestSpeed) + 10) / universeSpeed) seconds
// where speed is the speed setting (1-10) and slowestSpeed is the effective speed of the slowest ship.
func flightDuration(distance int, speed int, from, to model.Coordinate, ships model.Ships, research model.Research) time.Duration {
	slowest := slowestShipSpeed(ships, research)
	if slowest == 0 {
		return 0
	}

	if distance == 0 {
		// Planet↔moon at same position: minimum 10 seconds per OGame formula
		return 10 * time.Second
	}

	// OGame flight time formula in seconds
	flightSeconds := math.Round(
		(3500.0/float64(speed))*math.Sqrt(float64(distance)*10.0/float64(slowest)) + 10.0,
	) / float64(universeSpeed)

	return time.Duration(flightSeconds) * time.Second
}

// fuelConsumption calculates total fuel consumption for a fleet traveling at given speed.
// Per-ship OGame formula: consumption = baseConsumption * count * (distance / 35000) * ((speed/10 + 1) / 2)^2
func fuelConsumption(distance int, speed int, ships model.Ships, research model.Research) int64 {
	if distance == 0 {
		return 0
	}

	var total int64
	shipCounts := shipsToSlice(ships)

	for _, sc := range shipCounts {
		if sc.Count <= 0 {
			continue
		}
		speedEff, baseFuel := getShipEffectiveStats(sc.ID, research)
		if speedEff == 0 || baseFuel == 0 {
			continue
		}

		// fuel = baseFuel * count * (distance / 35000) * ((speed/10 + 1) / 2)^2
		speedPct := float64(speed)/10.0 + 1.0
		consumption := float64(baseFuel) * float64(sc.Count) * (float64(distance) / 35000.0) * math.Pow(speedPct/2.0, 2)

		total += int64(math.Round(consumption))
	}

	return total
}

// shipsToSlice converts a Ships struct to a slice of {ID, Count} pairs
// for iterating over all ship types.
func shipsToSlice(ships model.Ships) []struct{ ID, Count int } {
	return []struct{ ID, Count int }{
		{202, ships.SmallCargo},
		{203, ships.LargeCargo},
		{204, ships.LightFighter},
		{205, ships.HeavyFighter},
		{206, ships.Cruiser},
		{207, ships.Battleship},
		{208, ships.ColonyShip},
		{209, ships.Recycler},
		{210, ships.EspionageProbe},
		{211, ships.Bomber},
		{213, ships.Destroyer},
		{214, ships.Deathstar},
		{215, ships.Battlecruiser},
		// SolarSatellite (212) intentionally excluded — not mobile
	}
}

// EscapeRoute represents a viable fleet-save destination with computed metrics.
type EscapeRoute struct {
	Dest         model.Coordinate
	DestPlanetID int           // planet ID for send-fleet call
	Speed        int           // 1-10
	Duration     time.Duration // flight time at this speed
	FuelCost     int64         // deuterium consumed
	SafetyScore  int           // lower = safer
	MetalLoad    int64         // resources that can be loaded
	CrystalLoad  int64
	DeutLoad     int64 // remaining deuterium after fuel
	Mission      int   // mission type (always deploy for fleet-save)
}

// CalcEscapeRoutes generates all viable escape routes for an endangered planet,
// ranked by safety score (lower = safer). Returns an empty slice if no ships
// or no viable destinations exist — never returns nil.
func CalcEscapeRoutes(
	origin model.Planet,
	ships model.Ships,
	resources model.Resources,
	ownPlanets []model.Planet,
	attacks []model.AttackEvent,
	research model.Research,
) []EscapeRoute {
	// No ships — nothing to save
	if !hasShips(ships) {
		return []EscapeRoute{}
	}

	// No destinations — nowhere to go
	if len(ownPlanets) == 0 {
		return []EscapeRoute{}
	}

	var routes []EscapeRoute

	for _, dest := range ownPlanets {
		// Skip if destination is the same as origin
		if dest.ID == origin.ID {
			continue
		}

		distance := CalcDistance(origin.Coordinate, dest.Coordinate)
		// Distance 0 is valid for planet↔moon at same position (shortest deploy)
		// but skip if it's the exact same body (same ID, same type)
		if distance == 0 && origin.Coordinate.Type == dest.Coordinate.Type {
			continue
		}

		// Try speed settings from 10 (fastest) down to 1 (slowest)
		for speed := 10; speed >= 1; speed-- {
			fuel := fuelConsumption(distance, speed, ships, research)
			availableDeut := int64(resources.Deuterium)

			// Fuel insufficient — skip this speed
			if fuel > availableDeut {
				continue
			}

			duration := flightDuration(distance, speed, origin.Coordinate, dest.Coordinate, ships, research)

			route := EscapeRoute{
				Dest:         dest.Coordinate,
				DestPlanetID: dest.ID,
				Speed:        speed,
				Duration:     duration,
				FuelCost:     fuel,
				MetalLoad:    int64(resources.Metal),
				CrystalLoad:  int64(resources.Crystal),
				DeutLoad:     availableDeut - fuel,
				Mission:      3, // MissionDeploy
			}
			route.SafetyScore = calcSafetyScore(route, distance, attacks)

			routes = append(routes, route)
		}
	}

	// Sort by safety score ascending (lower = safer)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].SafetyScore < routes[j].SafetyScore
	})

	if routes == nil {
		routes = []EscapeRoute{}
	}

	return routes
}

// calcSafetyScore computes a safety score for a route (lower = safer).
//
// Scoring criteria:
//   - Base: 0
//   - +1000 if destination has incoming hostile attack
//   - +500 if destination is a planet (moons are safer — no phalanx)
//   - -100 if destination is a moon (bonus for moon destination)
//   - +distance/50 (farther = riskier)
//   - +fuelCost/10000 (more fuel = riskier)
func calcSafetyScore(route EscapeRoute, distance int, attacks []model.AttackEvent) int {
	score := 0

	// Check if destination is under attack
	for _, atk := range attacks {
		if coordsEqual(atk.Destination, route.Dest) {
			score += 1000
			break
		}
	}

	// Moon vs planet bonus
	if route.Dest.Type == "moon" {
		score -= 100 // moon is safer (can't be phalanxed)
	} else {
		score += 500 // planet is phalanx-visible
	}

	// Distance penalty (farther = riskier)
	score += distance / 50

	// Fuel cost penalty
	score += int(route.FuelCost / 10000)

	return score
}

// hasShips checks if any mobile ships exist in the fleet.
func hasShips(ships model.Ships) bool {
	return ships.SmallCargo > 0 || ships.LargeCargo > 0 ||
		ships.LightFighter > 0 || ships.HeavyFighter > 0 ||
		ships.Cruiser > 0 || ships.Battleship > 0 ||
		ships.Battlecruiser > 0 || ships.Bomber > 0 ||
		ships.Destroyer > 0 || ships.Deathstar > 0 ||
		ships.ColonyShip > 0 || ships.Recycler > 0 ||
		ships.EspionageProbe > 0
}

// coordsEqual checks if two coordinates are the same position and type.
func coordsEqual(a, b model.Coordinate) bool {
	return a.Galaxy == b.Galaxy && a.System == b.System && a.Position == b.Position && a.Type == b.Type
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
