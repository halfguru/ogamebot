package ogamex

import (
	"context"
	"fmt"

	"github.com/user/ogame-bot/internal/model"
)

func (c *Client) GetResearch(ctx context.Context) (model.Research, error) {
	body, err := c.doGet(ctx, "/research")
	if err != nil {
		return model.Research{}, fmt.Errorf("fetching research: %w", err)
	}
	return parseResearch(toReader(body))
}

func (c *Client) GetConstructions(ctx context.Context, planetID int) (model.Constructions, error) {
	path := fmt.Sprintf("/resources?cp=%d", planetID)
	body, err := c.doGet(ctx, path)
	if err != nil {
		return model.Constructions{}, fmt.Errorf("fetching constructions: %w", err)
	}
	return parseConstructions(toReader(body))
}

func (c *Client) GetServerSpeed(ctx context.Context) (int, error) {
	body, err := c.doGet(ctx, "/overview")
	if err != nil {
		return 0, fmt.Errorf("fetching overview for speed: %w", err)
	}
	speed, _ := parseServerInfo(body)
	return speed, nil
}

func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	body, err := c.doGet(ctx, "/overview")
	if err != nil {
		return "", fmt.Errorf("fetching overview for version: %w", err)
	}
	_, version := parseServerInfo(body)
	return version, nil
}
