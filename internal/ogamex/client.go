package ogamex

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/user/ogame-bot/internal/model"
	"github.com/user/ogame-bot/internal/ogamed"
)

var _ ogamed.ClientInterface = (*Client)(nil)

type Client struct {
	baseURL    string
	httpClient *http.Client
	csrfToken  string
	email      string
	password   string
	log        *slog.Logger
	mu         sync.Mutex
}

func NewClient(baseURL, email, password string, log *slog.Logger) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		email:    email,
		password: password,
		log:      log.With("component", "ogamex-client"),
	}
}

func (c *Client) getCSRFToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.csrfToken
}

func (c *Client) setCSRFToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.csrfToken = token
}

func (c *Client) GetServerTime(ctx context.Context) (string, error) {
	return "", fmt.Errorf("ogamex: GetServerTime not implemented")
}

func (c *Client) IsUnderAttack(ctx context.Context) (bool, error) {
	return false, fmt.Errorf("ogamex: IsUnderAttack not implemented")
}

func (c *Client) GetPlanets(ctx context.Context) ([]model.Planet, error) {
	return nil, fmt.Errorf("ogamex: GetPlanets not implemented")
}

func (c *Client) GetResources(ctx context.Context, planetID int) (model.Resources, error) {
	return model.Resources{}, fmt.Errorf("ogamex: GetResources not implemented")
}

func (c *Client) GetResourceBuildings(ctx context.Context, planetID int) (model.ResourceBuildings, error) {
	return model.ResourceBuildings{}, fmt.Errorf("ogamex: GetResourceBuildings not implemented")
}

func (c *Client) GetFacilities(ctx context.Context, planetID int) (model.Facilities, error) {
	return model.Facilities{}, fmt.Errorf("ogamex: GetFacilities not implemented")
}

func (c *Client) GetShips(ctx context.Context, planetID int) (model.Ships, error) {
	return model.Ships{}, fmt.Errorf("ogamex: GetShips not implemented")
}

func (c *Client) GetDefence(ctx context.Context, planetID int) (model.Defence, error) {
	return model.Defence{}, fmt.Errorf("ogamex: GetDefence not implemented")
}

func (c *Client) GetFleets(ctx context.Context) ([]model.Fleet, error) {
	return nil, fmt.Errorf("ogamex: GetFleets not implemented")
}

func (c *Client) GetResearch(ctx context.Context) (model.Research, error) {
	return model.Research{}, fmt.Errorf("ogamex: GetResearch not implemented")
}

func (c *Client) GetServerSpeed(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("ogamex: GetServerSpeed not implemented")
}

func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	return "", fmt.Errorf("ogamex: GetServerVersion not implemented")
}

func (c *Client) GetAttacks(ctx context.Context) ([]model.AttackEvent, error) {
	return nil, fmt.Errorf("ogamex: GetAttacks not implemented")
}

func (c *Client) GetSlots(ctx context.Context) (model.Slots, error) {
	return model.Slots{}, fmt.Errorf("ogamex: GetSlots not implemented")
}

func (c *Client) SendFleet(ctx context.Context, req model.SendFleetRequest) (int64, error) {
	return 0, fmt.Errorf("ogamex: SendFleet not implemented")
}

func (c *Client) CancelFleet(ctx context.Context, fleetID int64) error {
	return fmt.Errorf("ogamex: CancelFleet not implemented")
}

func (c *Client) GetConstructions(ctx context.Context, planetID int) (model.Constructions, error) {
	return model.Constructions{}, fmt.Errorf("ogamex: GetConstructions not implemented")
}

func (c *Client) BuildBuilding(ctx context.Context, planetID, buildingID int) error {
	return fmt.Errorf("ogamex: BuildBuilding not implemented")
}

func (c *Client) GetGalaxyInfos(ctx context.Context, galaxy, system int) (model.SystemInfos, error) {
	return model.SystemInfos{}, fmt.Errorf("ogamex: GetGalaxyInfos not implemented")
}

func (c *Client) GetEspionageReportMessages(ctx context.Context) ([]model.EspionageReportSummary, error) {
	return nil, fmt.Errorf("ogamex: GetEspionageReportMessages not implemented")
}

func (c *Client) GetEspionageReport(ctx context.Context, messageID int64) (model.EspionageReport, error) {
	return model.EspionageReport{}, fmt.Errorf("ogamex: GetEspionageReport not implemented")
}

func (c *Client) DeleteAllEspionageReports(ctx context.Context) error {
	return fmt.Errorf("ogamex: DeleteAllEspionageReports not implemented")
}

func (c *Client) GetCaptchaChallenge(ctx context.Context) (ogamed.CaptchaChallenge, error) {
	return ogamed.CaptchaChallenge{}, fmt.Errorf("ogamex: GetCaptchaChallenge not implemented")
}

func (c *Client) SolveCaptchaChallenge(ctx context.Context, challengeID string, answer int) error {
	return fmt.Errorf("ogamex: SolveCaptchaChallenge not implemented")
}
