package ogamex

import (
	"context"
	"fmt"

	"github.com/user/ogame-bot/internal/constants"
	"github.com/user/ogame-bot/internal/model"
)

func (c *Client) GetFleets(ctx context.Context) ([]model.Fleet, error) {
	body, err := c.doAJAXGet(ctx, "/ajax/fleet/eventlist/fetch")
	if err != nil {
		return nil, fmt.Errorf("fetching fleet events: %w", err)
	}
	return parseFleetEvents(body)
}

func (c *Client) GetAttacks(ctx context.Context) ([]model.AttackEvent, error) {
	fleets, err := c.GetFleets(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching fleets for attacks: %w", err)
	}

	hostileMissions := map[int]bool{
		constants.MissionAttack:          true,
		constants.MissionACSAttack:       true,
		constants.MissionMoonDestruction: true,
	}

	var attacks []model.AttackEvent
	for _, f := range fleets {
		if hostileMissions[f.Mission] {
			attacks = append(attacks, model.AttackEvent{
				ID:          int64(f.ID),
				MissionType: f.Mission,
				Origin:      f.Origin,
				Destination: f.Destination,
			})
		}
	}

	return attacks, nil
}

func (c *Client) IsUnderAttack(ctx context.Context) (bool, error) {
	body, err := c.doAJAXGet(ctx, "/ajax/fleet/eventbox/fetch")
	if err != nil {
		return false, fmt.Errorf("fetching eventbox: %w", err)
	}
	box, err := parseEventbox(body)
	if err != nil {
		return false, fmt.Errorf("parsing eventbox: %w", err)
	}
	return box.Hostile > 0, nil
}

func (c *Client) GetSlots(ctx context.Context) (model.Slots, error) {
	body, err := c.doGet(ctx, "/fleet")
	if err != nil {
		return model.Slots{}, fmt.Errorf("fetching fleet page: %w", err)
	}
	return parseSlots(toReader(body))
}

func (c *Client) GetServerTime(ctx context.Context) (string, error) {
	body, err := c.doAJAXGet(ctx, "/ajax/fleet/eventbox/fetch")
	if err != nil {
		return "", fmt.Errorf("fetching eventbox for server time: %w", err)
	}
	box, err := parseEventbox(body)
	if err != nil {
		return "", fmt.Errorf("parsing eventbox: %w", err)
	}
	return box.ServerTime, nil
}
