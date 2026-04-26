package ogamed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"log/slog"

	"github.com/user/ogame-bot/internal/config"
	"github.com/user/ogame-bot/internal/model"
)

// newTestClient creates a Client pointing at the given httptest server,
// with a rate limiter that has minimal delays for fast tests.
func newTestClient(ts *httptest.Server) *Client {
	cfg := config.RateLimitConfig{
		DefaultMinDelayMs: 1,
		DefaultMaxDelayMs: 1,
	}
	limiter := NewRateLimiter(cfg)
	return NewClient(ts.URL, limiter, slog.Default())
}

func TestClient_Login_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot/login" {
			t.Errorf("expected path /bot/login, got %s", r.URL.Path)
		}
		resp := OgamedResponse[any]{Status: "ok", Code: 200}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.Login(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestClient_Login_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OgamedResponse[any]{Status: "error", Code: 401, Message: "unauthorized"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.Login(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ogamedErr, ok := err.(*OgamedError)
	if !ok {
		t.Fatalf("expected OgamedError, got %T: %v", err, err)
	}
	if ogamedErr.Code != 401 {
		t.Errorf("expected code 401, got %d", ogamedErr.Code)
	}
	if ogamedErr.Message != "unauthorized" {
		t.Errorf("expected message 'unauthorized', got %q", ogamedErr.Message)
	}
}

func TestClient_GetPlanets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot/planets" {
			t.Errorf("expected path /bot/planets, got %s", r.URL.Path)
		}
		planets := []model.Planet{
			{
				ID:   1,
				Name: "Home",
				Coordinate: model.Coordinate{
					Galaxy:   1,
					System:   2,
					Position: 3,
					Type:     "planet",
				},
				Diameter:       12000,
				FieldsUsed:     50,
				FieldsTotal:    200,
				TemperatureMin: -20,
				TemperatureMax: 60,
				IsMoon:         false,
			},
		}
		resp := OgamedResponse[[]model.Planet]{
			Status: "ok",
			Code:   200,
			Result: planets,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	planets, err := client.GetPlanets(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(planets) != 1 {
		t.Fatalf("expected 1 planet, got %d", len(planets))
	}
	p := planets[0]
	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
	if p.Name != "Home" {
		t.Errorf("expected Name 'Home', got %q", p.Name)
	}
	if p.Coordinate.Galaxy != 1 || p.Coordinate.System != 2 || p.Coordinate.Position != 3 {
		t.Errorf("unexpected coordinate: %+v", p.Coordinate)
	}
	if p.Diameter != 12000 {
		t.Errorf("expected Diameter 12000, got %d", p.Diameter)
	}
}

func TestClient_GetResources(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/bot/planets/1/resources"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		resources := model.Resources{
			Metal:     10000,
			Crystal:   5000,
			Deuterium: 2000,
			Energy:    800,
		}
		resp := OgamedResponse[model.Resources]{
			Status: "ok",
			Code:   200,
			Result: resources,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	resources, err := client.GetResources(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resources.Metal != 10000 {
		t.Errorf("expected Metal 10000, got %d", resources.Metal)
	}
	if resources.Crystal != 5000 {
		t.Errorf("expected Crystal 5000, got %d", resources.Crystal)
	}
	if resources.Deuterium != 2000 {
		t.Errorf("expected Deuterium 2000, got %d", resources.Deuterium)
	}
}

func TestClient_RateLimiterCalled(t *testing.T) {
	var waitCalls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OgamedResponse[[]model.Planet]{Status: "ok", Code: 200, Result: []model.Planet{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Create client with a tracking wrapper around the rate limiter
	cfg := config.RateLimitConfig{
		DefaultMinDelayMs: 1,
		DefaultMaxDelayMs: 1,
	}
	baseLimiter := NewRateLimiter(cfg)
	client := &Client{
		baseURL: ts.URL,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		rateLimiter: &trackingLimiter{inner: baseLimiter, calls: &waitCalls},
		log:         slog.Default().With("component", "ogamed-client"),
		retryCfg:    DefaultRetryConfig,
	}

	_, err := client.GetPlanets(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if waitCalls.Load() != 1 {
		t.Errorf("expected rate limiter Wait to be called once, got %d", waitCalls.Load())
	}
}

func TestClient_RetryOnFailure(t *testing.T) {
	var requestCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count == 1 {
			// First request: return server error
			w.WriteHeader(http.StatusInternalServerError)
			resp := OgamedResponse[any]{Status: "error", Code: 500, Message: "internal error"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		// Second request: success
		resp := OgamedResponse[string]{Status: "ok", Code: 200, Result: "2024-01-01T00:00:00"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetServerTime(context.Background())
	if err != nil {
		t.Fatalf("expected nil error after retry, got %v", err)
	}
	if result != "2024-01-01T00:00:00" {
		t.Errorf("expected server time '2024-01-01T00:00:00', got %q", result)
	}
	if requestCount.Load() != 2 {
		t.Errorf("expected 2 requests (1 fail + 1 success), got %d", requestCount.Load())
	}
}

// TestClient_AllEndpointsExist is a compile-time check that Client implements ClientInterface.
func TestClient_AllEndpointsExist(t *testing.T) {
	var _ ClientInterface = (*Client)(nil)
	t.Log("Client implements ClientInterface with all 14 methods")
}

// TestClient_GetAllEndpoints tests a few more endpoints for coverage
func TestClient_GetServerTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot/server/time" {
			t.Errorf("expected path /bot/server/time, got %s", r.URL.Path)
		}
		resp := OgamedResponse[string]{Status: "ok", Code: 200, Result: "2024-01-01T12:00:00Z"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetServerTime(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result != "2024-01-01T12:00:00Z" {
		t.Errorf("expected '2024-01-01T12:00:00Z', got %q", result)
	}
}

func TestClient_IsUnderAttack(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OgamedResponse[bool]{Status: "ok", Code: 200, Result: true}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	underAttack, err := client.IsUnderAttack(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !underAttack {
		t.Error("expected true, got false")
	}
}

func TestClient_GetResearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot/get-research" {
			t.Errorf("expected path /bot/get-research, got %s", r.URL.Path)
		}
		research := model.Research{
			EnergyTechnology:    5,
			ComputerTechnology:  8,
			CombustionDrive:     6,
		}
		resp := OgamedResponse[model.Research]{Status: "ok", Code: 200, Result: research}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	research, err := client.GetResearch(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if research.EnergyTechnology != 5 {
		t.Errorf("expected EnergyTechnology 5, got %d", research.EnergyTechnology)
	}
	if research.ComputerTechnology != 8 {
		t.Errorf("expected ComputerTechnology 8, got %d", research.ComputerTechnology)
	}
}

func TestClient_GetFleets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot/fleets" {
			t.Errorf("expected path /bot/fleets, got %s", r.URL.Path)
		}
		fleets := []model.Fleet{
			{
				ID:           100,
				Mission:      1,
				ReturnFlight: false,
				Origin:       model.Coordinate{Galaxy: 1, System: 2, Position: 3, Type: "planet"},
				Destination:  model.Coordinate{Galaxy: 1, System: 5, Position: 7, Type: "planet"},
			},
		}
		resp := OgamedResponse[[]model.Fleet]{Status: "ok", Code: 200, Result: fleets}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	fleets, err := client.GetFleets(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(fleets) != 1 {
		t.Fatalf("expected 1 fleet, got %d", len(fleets))
	}
	if fleets[0].ID != 100 {
		t.Errorf("expected fleet ID 100, got %d", fleets[0].ID)
	}
	if fleets[0].Mission != 1 {
		t.Errorf("expected Mission 1, got %d", fleets[0].Mission)
	}
}

func TestClient_GetServerSpeed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OgamedResponse[int]{Status: "ok", Code: 200, Result: 7}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	speed, err := client.GetServerSpeed(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if speed != 7 {
		t.Errorf("expected speed 7, got %d", speed)
	}
}

func TestClient_EnvelopeError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OgamedResponse[any]{Status: "error", Code: 500, Message: "fail"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetPlanets(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ogamedErr, ok := err.(*OgamedError)
	if !ok {
		t.Fatalf("expected OgamedError, got %T", err)
	}
	if ogamedErr.Code != 500 {
		t.Errorf("expected code 500, got %d", ogamedErr.Code)
	}
	if ogamedErr.Message != "fail" {
		t.Errorf("expected message 'fail', got %q", ogamedErr.Message)
	}
}

// trackingLimiter wraps a RateLimiter to track Wait calls.
type trackingLimiter struct {
	inner *RateLimiter
	calls *atomic.Int32
}

func (t *trackingLimiter) Wait(ctx context.Context, endpoint string) error {
	t.calls.Add(1)
	return t.inner.Wait(ctx, endpoint)
}

// Test client path formatting for planet-specific endpoints
func TestClient_GetResourceBuildings(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/bot/planets/42/resources-buildings"
		if r.URL.Path != expected {
			t.Errorf("expected path %s, got %s", expected, r.URL.Path)
		}
		buildings := model.ResourceBuildings{
			MetalMine:    20,
			CrystalMine:  18,
			SolarPlant:   22,
			MetalStorage: 10,
		}
		resp := OgamedResponse[model.ResourceBuildings]{Status: "ok", Code: 200, Result: buildings}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	buildings, err := client.GetResourceBuildings(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if buildings.MetalMine != 20 {
		t.Errorf("expected MetalMine 20, got %d", buildings.MetalMine)
	}
}

func TestClient_GetFacilities(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/bot/planets/1/facilities"
		if r.URL.Path != expected {
			t.Errorf("expected path %s, got %s", expected, r.URL.Path)
		}
		facilities := model.Facilities{RoboticsFactory: 10, Shipyard: 8}
		resp := OgamedResponse[model.Facilities]{Status: "ok", Code: 200, Result: facilities}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	facilities, err := client.GetFacilities(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if facilities.RoboticsFactory != 10 {
		t.Errorf("expected RoboticsFactory 10, got %d", facilities.RoboticsFactory)
	}
}

func TestClient_GetShips(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/bot/planets/1/ships"
		if r.URL.Path != expected {
			t.Errorf("expected path %s, got %s", expected, r.URL.Path)
		}
		ships := model.Ships{LightFighter: 100, SmallCargo: 50}
		resp := OgamedResponse[model.Ships]{Status: "ok", Code: 200, Result: ships}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	ships, err := client.GetShips(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ships.LightFighter != 100 {
		t.Errorf("expected LightFighter 100, got %d", ships.LightFighter)
	}
}

func TestClient_GetDefence(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/bot/planets/1/defence"
		if r.URL.Path != expected {
			t.Errorf("expected path %s, got %s", expected, r.URL.Path)
		}
		defence := model.Defence{RocketLauncher: 500, LightLaser: 200}
		resp := OgamedResponse[model.Defence]{Status: "ok", Code: 200, Result: defence}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	defence, err := client.GetDefence(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if defence.RocketLauncher != 500 {
		t.Errorf("expected RocketLauncher 500, got %d", defence.RocketLauncher)
	}
}

func TestClient_GetServerVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OgamedResponse[string]{Status: "ok", Code: 200, Result: "9.1.0"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	version, err := client.GetServerVersion(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if version != "9.1.0" {
		t.Errorf("expected version '9.1.0', got %q", version)
	}
}

func TestClient_Logout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot/logout" {
			t.Errorf("expected path /bot/logout, got %s", r.URL.Path)
		}
		resp := OgamedResponse[any]{Status: "ok", Code: 200}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.Logout(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestClient_Logout_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OgamedResponse[any]{Status: "error", Code: 403, Message: "forbidden"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.Logout(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ogamedErr, ok := err.(*OgamedError)
	if !ok {
		t.Fatalf("expected OgamedError, got %T", err)
	}
	if ogamedErr.Code != 403 {
		t.Errorf("expected code 403, got %d", ogamedErr.Code)
	}
}

// Suppress unused import warning for fmt
var _ = fmt.Sprintf
