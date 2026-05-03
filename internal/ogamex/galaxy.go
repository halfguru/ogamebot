package ogamex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/user/ogame-bot/internal/model"
)

type galaxyResponse struct {
	Success      bool   `json:"success"`
	NewAjaxToken string `json:"newAjaxToken"`
	System       struct {
		GalaxyContent []galaxyPosition `json:"galaxyContent"`
	} `json:"system"`
}

type galaxyPosition struct {
	Position int            `json:"position"`
	Planets  []galaxyPlanet `json:"planets"`
	Player   galaxyPlayer   `json:"player"`
}

type galaxyPlanet struct {
	PlanetID   int    `json:"planetId"`
	PlanetName string `json:"planetName"`
	PlanetType int    `json:"planetType"`
	PlayerID   int    `json:"playerId"`
}

type galaxyPlayer struct {
	PlayerID       int           `json:"playerId"`
	PlayerName     string        `json:"playerName"`
	IsInactive     bool          `json:"isInactive"`
	IsLongInactive bool          `json:"isLongInactive"`
	IsOnVacation   bool          `json:"isOnVacation"`
	IsBanned       bool          `json:"isBanned"`
	Actions        galaxyActions `json:"actions"`
}

type galaxyActions struct {
	Highscore struct {
		Available bool `json:"available"`
		Rank      int  `json:"rank"`
	} `json:"highscore"`
}

func (c *Client) GetGalaxyInfos(ctx context.Context, galaxy, system int) (model.SystemInfos, error) {
	data := url.Values{}
	data.Set("galaxy", strconv.Itoa(galaxy))
	data.Set("system", strconv.Itoa(system))

	body, err := c.doAJAXPost(ctx, "/ajax/galaxy", data)
	if err != nil {
		return model.SystemInfos{}, fmt.Errorf("scanning galaxy %d:%d: %w", galaxy, system, err)
	}

	var resp galaxyResponse
	if err := jsonUnmarshal(body, &resp); err != nil {
		return model.SystemInfos{}, fmt.Errorf("parsing galaxy response: %w", err)
	}

	if !resp.Success {
		return model.SystemInfos{}, fmt.Errorf("galaxy scan failed for %d:%d", galaxy, system)
	}

	return mapGalaxyResponse(galaxy, system, resp), nil
}

func mapGalaxyResponse(galaxy, system int, resp galaxyResponse) model.SystemInfos {
	info := model.SystemInfos{
		Galaxy:  galaxy,
		System:  system,
		Planets: []model.PlanetPosition{},
	}

	for _, pos := range resp.System.GalaxyContent {
		if pos.Player.PlayerName == "Deep space" || pos.Player.PlayerID >= 99999 {
			continue
		}

		for _, pl := range pos.Planets {
			isMoon := pl.PlanetType == 3
			pp := model.PlanetPosition{
				Position: pos.Position,
				Name:     pl.PlanetName,
				PlayerID: int64(pos.Player.PlayerID),
				PlayerName: pos.Player.PlayerName,
				Inactive:     pos.Player.IsInactive,
				LongInactive: pos.Player.IsLongInactive,
				Vacation:     pos.Player.IsOnVacation,
				Banned:       pos.Player.IsBanned,
				Rank:         pos.Player.Actions.Highscore.Rank,
				Coordinate: model.Coordinate{
					Galaxy:   galaxy,
					System:   system,
					Position: pos.Position,
				},
				Moon: isMoon,
			}
			if isMoon {
				pp.Coordinate.Type = "moon"
			} else {
				pp.Coordinate.Type = "planet"
			}
			info.Planets = append(info.Planets, pp)
		}
	}

	return info
}
