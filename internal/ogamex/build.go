package ogamex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

var resourceBuildingIDs = map[int]bool{
	1:  true,
	2:  true,
	3:  true,
	4:  true,
	12: true,
	22: true,
	23: true,
	24: true,
}

type buildResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Success *bool  `json:"success"`
}

func buildingEndpoint(buildingID int) (string, error) {
	if resourceBuildingIDs[buildingID] {
		return "/resources/add-buildrequest", nil
	}
	if buildingID >= 14 && buildingID <= 44 {
		return "/facilities/add-buildrequest", nil
	}
	return "", fmt.Errorf("ogamex: unknown building ID %d", buildingID)
}

func (c *Client) BuildBuilding(ctx context.Context, planetID, buildingID int) error {
	endpoint, err := buildingEndpoint(buildingID)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?cp=%d", endpoint, planetID)

	data := url.Values{}
	data.Set("technologyId", fmt.Sprintf("%d", buildingID))

	body, err := c.doPost(ctx, path, data)
	if err != nil {
		return fmt.Errorf("ogamex: BuildBuilding POST failed: %w", err)
	}

	var resp buildResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("ogamex: BuildBuilding response parse error: %w", err)
	}

	if resp.Status != "success" && resp.Status != "ok" {
		msg := resp.Message
		if msg == "" {
			msg = string(body)
		}
		return fmt.Errorf("ogamex: BuildBuilding failed: %s", msg)
	}

	if resp.Success != nil && !*resp.Success {
		msg := resp.Message
		if msg == "" {
			msg = string(body)
		}
		return fmt.Errorf("ogamex: BuildBuilding failed: %s", msg)
	}

	return nil
}

var researchIDs = map[int]bool{
	106: true, 108: true, 109: true, 110: true, 111: true,
	113: true, 114: true, 115: true, 117: true, 118: true,
	120: true, 121: true, 122: true, 123: true, 124: true, 199: true,
}

func (c *Client) BuildResearch(ctx context.Context, planetID, researchID int) error {
	path := fmt.Sprintf("/research/add-buildrequest?cp=%d", planetID)

	data := url.Values{}
	data.Set("technologyId", fmt.Sprintf("%d", researchID))

	body, err := c.doPost(ctx, path, data)
	if err != nil {
		return fmt.Errorf("ogamex: BuildResearch POST failed: %w", err)
	}

	var resp buildResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("ogamex: BuildResearch response parse error: %w", err)
	}

	if resp.Status != "success" && resp.Status != "ok" {
		msg := resp.Message
		if msg == "" {
			msg = string(body)
		}
		return fmt.Errorf("ogamex: BuildResearch failed: %s", msg)
	}

	if resp.Success != nil && !*resp.Success {
		msg := resp.Message
		if msg == "" {
			msg = string(body)
		}
		return fmt.Errorf("ogamex: BuildResearch failed: %s", msg)
	}

	return nil
}
