package ogamed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/user/ogame-bot/internal/model"
)

const defaultHTTPTimeout = 30 * time.Second

type CaptchaChallenge struct {
	ID       string `json:"ID"`
	Icons    string `json:"Icons"`
	Question string `json:"Question"`
}

// ClientInterface defines all ogamed API methods.
// State manager depends on this interface, not the concrete Client.
type ClientInterface interface {
	Login(ctx context.Context) error
	Logout(ctx context.Context) error
	GetServerTime(ctx context.Context) (string, error)
	IsUnderAttack(ctx context.Context) (bool, error)
	GetPlanets(ctx context.Context) ([]model.Planet, error)
	GetResources(ctx context.Context, planetID int) (model.Resources, error)
	GetResourceBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error)
	GetFacilities(ctx context.Context, planetID int) (model.Facilities, error)
	GetShips(ctx context.Context, planetID int) (model.Ships, error)
	GetDefence(ctx context.Context, planetID int) (model.Defence, error)
	GetFleets(ctx context.Context) ([]model.Fleet, error)
	GetResearch(ctx context.Context) (model.Research, error)
	GetServerSpeed(ctx context.Context) (int, error)
	GetServerVersion(ctx context.Context) (string, error)
	GetAttacks(ctx context.Context) ([]model.AttackEvent, error)
	GetSlots(ctx context.Context) (model.Slots, error)
	SendFleet(ctx context.Context, req model.SendFleetRequest) (int64, error)
	CancelFleet(ctx context.Context, fleetID int64) error
	GetConstructions(ctx context.Context, planetID int) (model.Constructions, error)
	BuildBuilding(ctx context.Context, planetID, buildingID int) error
	BuildResearch(ctx context.Context, planetID, researchID int) error
	GetGalaxyInfos(ctx context.Context, galaxy, system int) (model.SystemInfos, error)
	GetEspionageReportMessages(ctx context.Context) ([]model.EspionageReportSummary, error)
	GetEspionageReport(ctx context.Context, messageID int64) (model.EspionageReport, error)
	DeleteAllEspionageReports(ctx context.Context) error
	GetCaptchaChallenge(ctx context.Context) (CaptchaChallenge, error)
	SolveCaptchaChallenge(ctx context.Context, challengeID string, answer int) error
}

// Client implements ClientInterface with rate limiting and retry.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	rateLimiter rateLimiterInterface
	log         *slog.Logger
	retryCfg    RetryConfig
}

// rateLimiterInterface abstracts the rate limiter for testing.
type rateLimiterInterface interface {
	Wait(ctx context.Context, endpoint string) error
}

// NewClient creates a new ogamed REST client.
func NewClient(baseURL string, limiter *RateLimiter, log *slog.Logger) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		rateLimiter: limiter,
		log:        log.With("component", "ogamed-client"),
		retryCfg:   DefaultRetryConfig,
	}
}

// get performs a GET request with rate limiting and retry. Returns the response body bytes.
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx, path); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	var body []byte
	err := retryWithBackoff(ctx, func() error {
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("executing request: %w", err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}

		c.log.Debug("ogamed request completed",
			"path", path,
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		// Map HTTP-level errors to OgamedError for retry decisions
		if resp.StatusCode >= 500 {
			return &OgamedError{Code: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
		}
		if resp.StatusCode >= 400 {
			return &OgamedError{Code: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
		}

		return nil
	}, c.retryCfg, c.log)

	if err != nil {
		return nil, err
	}
	return body, nil
}

// post performs a POST request with rate limiting and retry. Returns the response body bytes.
func (c *Client) post(ctx context.Context, path string, data url.Values) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx, path); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	var body []byte
	err := retryWithBackoff(ctx, func() error {
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(data.Encode()))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("executing request: %w", err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}

		c.log.Debug("ogamed POST request completed",
			"path", path,
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		// Map HTTP-level errors to OgamedError for retry decisions
		if resp.StatusCode >= 500 {
			return &OgamedError{Code: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
		}
		if resp.StatusCode >= 400 {
			return &OgamedError{Code: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
		}

		return nil
	}, c.retryCfg, c.log)

	if err != nil {
		return nil, err
	}
	return body, nil
}

// getTyped performs a GET request and deserializes the ogamed response envelope into type T.
func getTyped[T any](c *Client, ctx context.Context, path string) (T, error) {
	var zero T

	body, err := c.get(ctx, path)
	if err != nil {
		return zero, err
	}

	// First unmarshal into envelope with raw message
	var envelope OgamedResponse[json.RawMessage]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return zero, fmt.Errorf("unmarshaling ogamed response: %w", err)
	}

	if envelope.Status != "ok" {
		return zero, &OgamedError{Code: envelope.Code, Message: envelope.Message}
	}

	// Unmarshal the Result field into the target type
	var result T
	if envelope.Result != nil {
		if err := json.Unmarshal(envelope.Result, &result); err != nil {
			return zero, fmt.Errorf("unmarshaling result: %w", err)
		}
	}

	return result, nil
}

// postTyped performs a POST request and deserializes the ogamed response envelope into type T.
func postTyped[T any](c *Client, ctx context.Context, path string, data url.Values) (T, error) {
	var zero T

	body, err := c.post(ctx, path, data)
	if err != nil {
		return zero, err
	}

	// First unmarshal into envelope with raw message
	var envelope OgamedResponse[json.RawMessage]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return zero, fmt.Errorf("unmarshaling ogamed response: %w", err)
	}

	if envelope.Status != "ok" {
		return zero, &OgamedError{Code: envelope.Code, Message: envelope.Message}
	}

	// Unmarshal the Result field into the target type
	var result T
	if envelope.Result != nil {
		if err := json.Unmarshal(envelope.Result, &result); err != nil {
			return zero, fmt.Errorf("unmarshaling result: %w", err)
		}
	}

	return result, nil
}

// Login authenticates with the OGame server via ogamed.
func (c *Client) Login(ctx context.Context) error {
	_, err := getTyped[any](c, ctx, "/bot/login")
	return err
}

// Logout ends the ogamed session.
func (c *Client) Logout(ctx context.Context) error {
	_, err := getTyped[any](c, ctx, "/bot/logout")
	return err
}

// GetServerTime returns the current OGame server time.
func (c *Client) GetServerTime(ctx context.Context) (string, error) {
	return getTyped[string](c, ctx, "/bot/server/time")
}

// IsUnderAttack checks if the player is currently under attack.
func (c *Client) IsUnderAttack(ctx context.Context) (bool, error) {
	return getTyped[bool](c, ctx, "/bot/is-under-attack")
}

// GetPlanets returns all planets owned by the player.
func (c *Client) GetPlanets(ctx context.Context) ([]model.Planet, error) {
	return getTyped[[]model.Planet](c, ctx, "/bot/planets")
}

// GetResources returns the current resources on a planet.
func (c *Client) GetResources(ctx context.Context, planetID int) (model.Resources, error) {
	path := fmt.Sprintf("/bot/planets/%d/resources", planetID)
	return getTyped[model.Resources](c, ctx, path)
}

// GetResourceBuildings returns the resource building levels on a planet.
func (c *Client) GetResourceBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error) {
	path := fmt.Sprintf("/bot/planets/%d/resources-buildings", planetID)
	return getTyped[model.ResourceBuildings](c, ctx, path)
}

// GetFacilities returns the facility levels on a planet.
func (c *Client) GetFacilities(ctx context.Context, planetID int) (model.Facilities, error) {
	path := fmt.Sprintf("/bot/planets/%d/facilities", planetID)
	return getTyped[model.Facilities](c, ctx, path)
}

// GetShips returns the ship quantities on a planet.
func (c *Client) GetShips(ctx context.Context, planetID int) (model.Ships, error) {
	path := fmt.Sprintf("/bot/planets/%d/ships", planetID)
	return getTyped[model.Ships](c, ctx, path)
}

// GetDefence returns the defence quantities on a planet.
func (c *Client) GetDefence(ctx context.Context, planetID int) (model.Defence, error) {
	path := fmt.Sprintf("/bot/planets/%d/defence", planetID)
	return getTyped[model.Defence](c, ctx, path)
}

// GetFleets returns all active fleets.
func (c *Client) GetFleets(ctx context.Context) ([]model.Fleet, error) {
	return getTyped[[]model.Fleet](c, ctx, "/bot/fleets")
}

// GetResearch returns the player's research levels.
func (c *Client) GetResearch(ctx context.Context) (model.Research, error) {
	return getTyped[model.Research](c, ctx, "/bot/get-research")
}

// GetServerSpeed returns the server speed multiplier.
func (c *Client) GetServerSpeed(ctx context.Context) (int, error) {
	return getTyped[int](c, ctx, "/bot/server/speed")
}

// GetServerVersion returns the OGame server version.
func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	return getTyped[string](c, ctx, "/bot/server/version")
}

// GetAttacks returns all incoming attacks.
func (c *Client) GetAttacks(ctx context.Context) ([]model.AttackEvent, error) {
	return getTyped[[]model.AttackEvent](c, ctx, "/bot/attacks")
}

// GetSlots returns the current fleet and expedition slot usage.
func (c *Client) GetSlots(ctx context.Context) (model.Slots, error) {
	return getTyped[model.Slots](c, ctx, "/bot/slots")
}

// SendFleet dispatches a fleet and returns the fleet ID.
func (c *Client) SendFleet(ctx context.Context, req model.SendFleetRequest) (int64, error) {
	path := fmt.Sprintf("/bot/planets/%d/send-fleet", req.PlanetID)

	data := url.Values{}
	for _, ship := range req.Ships {
		data.Add("ships", fmt.Sprintf("%d,%d", ship.ID, ship.Count))
	}
	data.Set("speed", strconv.Itoa(req.Speed))
	data.Set("galaxy", strconv.Itoa(req.Galaxy))
	data.Set("system", strconv.Itoa(req.System))
	data.Set("position", strconv.Itoa(req.Position))
	data.Set("type", strconv.Itoa(req.Type))
	data.Set("mission", strconv.Itoa(req.Mission))
	data.Set("metal", strconv.FormatInt(req.Metal, 10))
	data.Set("crystal", strconv.FormatInt(req.Crystal, 10))
	data.Set("deuterium", strconv.FormatInt(req.Deuterium, 10))

	return postTyped[int64](c, ctx, path, data)
}

// CancelFleet cancels a fleet by its ID.
func (c *Client) CancelFleet(ctx context.Context, fleetID int64) error {
	path := fmt.Sprintf("/bot/fleets/%d/cancel", fleetID)
	_, err := postTyped[any](c, ctx, path, url.Values{})
	return err
}

// GetConstructions returns the current construction status for a planet.
func (c *Client) GetConstructions(ctx context.Context, planetID int) (model.Constructions, error) {
	path := fmt.Sprintf("/bot/planets/%d/constructions", planetID)
	return getTyped[model.Constructions](c, ctx, path)
}

// BuildBuilding starts constructing the given building on the specified planet.
func (c *Client) BuildBuilding(ctx context.Context, planetID, buildingID int) error {
	path := fmt.Sprintf("/bot/planets/%d/build/building/%d", planetID, buildingID)
	_, err := postTyped[any](c, ctx, path, url.Values{})
	return err
}

func (c *Client) BuildResearch(ctx context.Context, planetID, researchID int) error {
	path := fmt.Sprintf("/bot/planets/%d/build/research/%d", planetID, researchID)
	_, err := postTyped[any](c, ctx, path, url.Values{})
	return err
}

// GetGalaxyInfos scans a solar system and returns player/planet information.
func (c *Client) GetGalaxyInfos(ctx context.Context, galaxy, system int) (model.SystemInfos, error) {
	path := fmt.Sprintf("/bot/galaxy-infos/%d/%d", galaxy, system)
	return getTyped[model.SystemInfos](c, ctx, path)
}

// GetEspionageReportMessages returns all espionage report message summaries.
func (c *Client) GetEspionageReportMessages(ctx context.Context) ([]model.EspionageReportSummary, error) {
	return getTyped[[]model.EspionageReportSummary](c, ctx, "/bot/get-espionage-report-messages")
}

// GetEspionageReport returns the full espionage report for a given message ID.
func (c *Client) GetEspionageReport(ctx context.Context, messageID int64) (model.EspionageReport, error) {
	path := fmt.Sprintf("/bot/get-espionage-report/%d", messageID)
	return getTyped[model.EspionageReport](c, ctx, path)
}

// DeleteAllEspionageReports deletes all espionage report messages.
func (c *Client) DeleteAllEspionageReports(ctx context.Context) error {
	_, err := postTyped[any](c, ctx, "/bot/delete-all-espionage-reports", url.Values{})
	return err
}

func (c *Client) GetCaptchaChallenge(ctx context.Context) (CaptchaChallenge, error) {
	return getTyped[CaptchaChallenge](c, ctx, "/bot/captcha/challenge")
}

func (c *Client) SolveCaptchaChallenge(ctx context.Context, challengeID string, answer int) error {
	data := url.Values{}
	data.Set("id", challengeID)
	data.Set("answer", strconv.Itoa(answer))
	_, err := postTyped[any](c, ctx, "/bot/captcha/solve", data)
	return err
}
