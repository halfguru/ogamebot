// Package farmer implements the auto-farm worker that scans galaxies,
// spies inactives, and attacks profitable targets.
package farmer

import (
	"context"

	"github.com/user/ogame-bot/internal/model"
)

// FarmerStateReader provides read access to cached game state needed by the farmer.
// state.Manager implicitly satisfies this interface.
type FarmerStateReader interface {
	GetPlanets(ctx context.Context) ([]model.Planet, error)
	GetResearch(ctx context.Context) (model.Research, error)
}

// FarmTarget represents an evaluated farm target ready for attack.
type FarmTarget struct {
	Coordinate    model.Coordinate
	PlayerName    string
	MetalLoot     int64 // estimated metal loot
	CrystalLoot   int64
	DeuteriumLoot int64
	TotalValue    int64 // metal-equivalent total loot value
	NetProfit     int64 // totalValue - estimated fuel cost
	HasDefense    bool
	Distance      int
}
