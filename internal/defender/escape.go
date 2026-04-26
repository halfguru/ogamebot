// Package defender implements the fleet-save escape route calculator.
// It evaluates all possible fleet-save destinations and ranks them by safety.
package defender

import (
	"math"
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
	if distance == 0 {
		return 0
	}

	slowest := slowestShipSpeed(ships, research)
	if slowest == 0 {
		return 0
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

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
