package ogamex

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/user/ogame-bot/internal/model"
)

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
}

var _ ClientInterface = (*Client)(nil)

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
