// Package defender implements the fleet-save escape route calculator.
// It evaluates all possible fleet-save destinations and ranks them by safety.
package defender

import (
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
// SolarSatellite is excluded as it is not a mobile ship.
var shipDB = map[int]shipStats{
	// TODO: populate with ship stats
}

// CalcDistance computes the distance between two coordinates.
func CalcDistance(from, to model.Coordinate) int {
	// TODO: implement
	return 0
}

// effectiveSpeed calculates the effective speed of a ship after applying
// drive technology bonuses.
func effectiveSpeed(baseSpeed int, driveType string, driveLevel int) int64 {
	// TODO: implement
	return 0
}

// fuelConsumption calculates total fuel consumption for a fleet at given speed.
func fuelConsumption(distance int, speed int, ships model.Ships, research model.Research) int64 {
	// TODO: implement
	return 0
}

// flightDuration calculates how long a fleet takes to travel the given distance.
func flightDuration(distance int, speed int, from, to model.Coordinate, ships model.Ships, research model.Research) time.Duration {
	// TODO: implement
	return 0
}
