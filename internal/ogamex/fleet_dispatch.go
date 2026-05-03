package ogamex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/user/ogame-bot/internal/model"
)

type checkTargetResponse struct {
	Status           string         `json:"status"`
	Orders           map[string]bool `json:"orders"`
	TargetInhabited  bool           `json:"targetInhabited"`
	TargetPlayerID   int            `json:"targetPlayerId"`
	NewAjaxToken     string         `json:"newAjaxToken"`
}

type sendFleetResponse struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message"`
	RedirectURL  string            `json:"redirectUrl"`
	Errors       []sendFleetError  `json:"errors"`
	NewAjaxToken string            `json:"newAjaxToken"`
}

type sendFleetError struct {
	Message string `json:"message"`
	Error   int    `json:"error"`
}

type recallFleetResponse struct {
	Success      bool   `json:"success"`
	NewAjaxToken string `json:"newAjaxToken"`
}

func (c *Client) SendFleet(ctx context.Context, req model.SendFleetRequest) (int64, error) {
	_, err := c.doGet(ctx, "/overview?cp="+strconv.Itoa(req.PlanetID))
	if err != nil {
		return 0, fmt.Errorf("switching to planet %d: %w", req.PlanetID, err)
	}

	checkData := url.Values{
		"galaxy":  {strconv.Itoa(req.Galaxy)},
		"system":  {strconv.Itoa(req.System)},
		"position": {strconv.Itoa(req.Position)},
		"type":    {strconv.Itoa(req.Type)},
		"token":   {c.getCSRFToken()},
	}

	checkBody, err := c.doAJAXPost(ctx, "/ajax/fleet/dispatch/check-target", checkData)
	if err != nil {
		return 0, fmt.Errorf("check-target request: %w", err)
	}

	var checkResp checkTargetResponse
	if err := json.Unmarshal(checkBody, &checkResp); err != nil {
		return 0, fmt.Errorf("parsing check-target response: %w", err)
	}
	if checkResp.Status != "success" {
		return 0, fmt.Errorf("check-target failed: status=%s", checkResp.Status)
	}

	missionKey := strconv.Itoa(req.Mission)
	if !checkResp.Orders[missionKey] {
		return 0, fmt.Errorf("mission %d not available for target (%d:%d:%d type=%d)",
			req.Mission, req.Galaxy, req.System, req.Position, req.Type)
	}

	sendData := url.Values{
		"galaxy":   {strconv.Itoa(req.Galaxy)},
		"system":   {strconv.Itoa(req.System)},
		"position": {strconv.Itoa(req.Position)},
		"type":     {strconv.Itoa(req.Type)},
		"mission":  {strconv.Itoa(req.Mission)},
		"speed":    {strconv.Itoa(req.Speed)},
		"metal":    {strconv.FormatInt(req.Metal, 10)},
		"crystal":  {strconv.FormatInt(req.Crystal, 10)},
		"deuterium": {strconv.FormatInt(req.Deuterium, 10)},
		"token":    {c.getCSRFToken()},
	}

	for _, ship := range req.Ships {
		sendData.Set("am"+strconv.Itoa(ship.ID), strconv.Itoa(ship.Count))
	}

	sendBody, err := c.doAJAXPost(ctx, "/ajax/fleet/dispatch/send-fleet", sendData)
	if err != nil {
		return 0, fmt.Errorf("send-fleet request: %w", err)
	}

	var sendResp sendFleetResponse
	if err := json.Unmarshal(sendBody, &sendResp); err != nil {
		return 0, fmt.Errorf("parsing send-fleet response: %w", err)
	}

	if !sendResp.Success {
		if len(sendResp.Errors) > 0 {
			return 0, fmt.Errorf("send-fleet failed: %s (code %d)", sendResp.Errors[0].Message, sendResp.Errors[0].Error)
		}
		return 0, fmt.Errorf("send-fleet failed: %s", sendResp.Message)
	}

	return 0, nil
}

func (c *Client) CancelFleet(ctx context.Context, fleetID int64) error {
	data := url.Values{
		"fleet_mission_id": {strconv.FormatInt(fleetID, 10)},
	}

	body, err := c.doAJAXPost(ctx, "/ajax/fleet/dispatch/recall-fleet", data)
	if err != nil {
		return fmt.Errorf("recall-fleet request: %w", err)
	}

	var resp recallFleetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing recall-fleet response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("recall-fleet failed for fleet %d", fleetID)
	}

	return nil
}
