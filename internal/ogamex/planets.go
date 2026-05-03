package ogamex

import (
	"context"
	"fmt"

	"github.com/user/ogame-bot/internal/model"
)

func (c *Client) GetPlanets(ctx context.Context) ([]model.Planet, error) {
	body, err := c.doGet(ctx, "/overview")
	if err != nil {
		return nil, fmt.Errorf("fetching overview: %w", err)
	}
	return parsePlanetList(toReader(body))
}

func (c *Client) GetResources(ctx context.Context, planetID int) (model.Resources, error) {
	path := fmt.Sprintf("/overview?cp=%d", planetID)
	body, err := c.doGet(ctx, path)
	if err != nil {
		return model.Resources{}, fmt.Errorf("fetching resources: %w", err)
	}
	return parseResources(toReader(body))
}

func (c *Client) GetResourceBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error) {
	path := fmt.Sprintf("/resources?cp=%d", planetID)
	body, err := c.doGet(ctx, path)
	if err != nil {
		return model.ResourceBuildings{}, fmt.Errorf("fetching resource buildings: %w", err)
	}
	return parseResourceBuildings(toReader(body))
}

func (c *Client) GetFacilities(ctx context.Context, planetID int) (model.Facilities, error) {
	path := fmt.Sprintf("/facilities?cp=%d", planetID)
	body, err := c.doGet(ctx, path)
	if err != nil {
		return model.Facilities{}, fmt.Errorf("fetching facilities: %w", err)
	}
	return parseFacilities(toReader(body))
}

func (c *Client) GetShips(ctx context.Context, planetID int) (model.Ships, error) {
	path := fmt.Sprintf("/shipyard?cp=%d", planetID)
	body, err := c.doGet(ctx, path)
	if err != nil {
		return model.Ships{}, fmt.Errorf("fetching ships: %w", err)
	}
	return parseShips(toReader(body))
}

func (c *Client) GetDefence(ctx context.Context, planetID int) (model.Defence, error) {
	path := fmt.Sprintf("/defense?cp=%d", planetID)
	body, err := c.doGet(ctx, path)
	if err != nil {
		return model.Defence{}, fmt.Errorf("fetching defence: %w", err)
	}
	return parseDefence(toReader(body))
}
