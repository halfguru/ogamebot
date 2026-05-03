# Architecture Research: OGameX Bot

## Current Architecture (what's kept)

The existing codebase has a clean layered architecture with an interface-based client abstraction. The pivot replaces only the concrete HTTP client — everything above the interface is preserved.

```
┌─────────────────────────────────────────────────────┐
│                    cmd/bot/main.go                   │
│              (entry point, wiring, startup)          │
└──────────┬──────────┬──────────┬──────────┬─────────┘
           │          │          │          │
     ┌─────▼────┐ ┌──▼────┐ ┌──▼────┐ ┌──▼──────┐
     │ defender │ │builder│ │farmer │ │dashboard│
     │ (worker) │ │(work.)│ │(work.)│ │ (web UI)│
     └────┬─────┘ └──┬────┘ └──┬────┘ └────┬────┘
          │          │          │            │
          ▼          ▼          ▼            │
     ┌────────────────────────────────┐      │
     │       state.Manager           │      │
     │  (SQLite cache, periodic poll)│      │
     └────────────┬──────────────────┘      │
                  │                          │
     ┌────────────▼──────────────────┐      │
     │     ogamed.ClientInterface    │◄─────┘
     │  (26 methods, abstracted)     │
     └────────────┬──────────────────┘
                  │                         ← SWAP BOUNDARY
     ┌────────────▼──────────────────┐
     │   ogamed.Client (concrete)    │  ← TO BE REPLACED
     │   REST calls to ogamed daemon │
     └───────────────────────────────┘
```

### Preserved components (zero or minimal changes)

| Component | Package | Changes Needed |
|-----------|---------|----------------|
| Domain types | `internal/model` | Add OGameX-specific JSON tags (both PascalCase and snake_case) |
| Game constants | `internal/constants` | None (ship/building IDs are the same game) |
| ROI calculator | `internal/builder/roi.go` | None (pure math) |
| Escape routes | `internal/defender/escape.go` | None (pure math) |
| Workers | `internal/defender, builder, farmer` | Import path change only (`ogamed` → `ogamex`) |
| State manager | `internal/state` | Import path change + adjust for any new methods |
| Dashboard | `internal/dashboard` | None (reads from SQLite) |
| Config | `internal/config` | Replace `OgamedConfig` with `OGameXConfig` |
| Migrations | `internal/state/migrations` | None (SQLite schema is client-agnostic) |

### Interface methods to implement

The `ClientInterface` in `internal/ogamed/client.go:28` defines 26 methods. The new OGameX client must implement all of them. Some methods won't have direct OGameX equivalents and will need creative mapping (see New Client Architecture).

## New Client Architecture

### Package structure

```
internal/ogamex/
├── client.go          # Client struct, constructor, ClientInterface impl
├── session.go         # Login, session management, cookie jar, CSRF token
├── transport.go       # HTTP transport with auto-CSRF injection, planet context
├── parser.go          # HTML parsers for page-scraped data
├── parser_test.go     # Parser unit tests with fixture HTML files
├── fixtures/          # Saved HTML pages for parser tests
├── ajax.go            # AJAX endpoint helpers (request/response types)
├── fleet.go           # Fleet dispatch and recall implementation
├── build.go           # Build request implementation
└── galaxy.go          # Galaxy scan, espionage report implementation
```

### Why a new package (`internal/ogamex`)

- Clean import path swap: `"github.com/user/ogame-bot/internal/ogamex"` replaces `"github.com/user/ogame-bot/internal/ogamed"`
- No risk of accidentally using ogamed-specific code
- Tests can use fixture HTML files without touching real servers
- The old `internal/ogamed/` package can be deleted once the new client is validated

### Client struct design

```go
package ogamex

type Client struct {
    baseURL    string           // e.g. "https://main.ogamex.dev"
    httpClient *http.Client     // cookie jar manages session cookies
    csrfToken  string           // current CSRF token, updated from every response
    log        *slog.Logger
    mu         sync.Mutex       // protects csrfToken for concurrent access
}

func NewClient(baseURL string, log *slog.Logger) *Client { ... }
```

Key design decisions:
- **`*http.Client` with cookie jar**: Go's `net/http/cookiejar` automatically stores and sends session cookies. No manual cookie management.
- **Single `csrfToken` field**: Updated atomically from every response's `newAjaxToken` field. Protected by mutex for concurrent goroutine access (state manager + workers all call client).
- **No rate limiter needed**: OGameX has no rate limiting on game endpoints (only login at 20/min). The existing `RateLimiter` type can be dropped.
- **No retry logic for 419 (CSRF expiry)**: Instead, the client automatically re-authenticates and retries once on CSRF failure.

### OGameX endpoint mapping to ClientInterface methods

| ClientInterface method | OGameX mechanism | Type |
|------------------------|------------------|------|
| `Login` | POST `/login` | Form POST |
| `Logout` | POST `/logout` | Form POST |
| `GetServerTime` | Parse from any page HTML (`<meta>` or JS var) or `/ajax/fleet/eventbox/fetch` response `serverTime` | JSON response |
| `IsUnderAttack` | GET `/ajax/fleet/eventbox/fetch` → check `hostile > 0` | AJAX JSON |
| `GetPlanets` | Parse planet list from any page HTML (top nav planet menu) | HTML scrape |
| `GetResources` | Parse resource bars from any page HTML | HTML scrape |
| `GetResourceBuildings` | GET `/ajax/resources?technology=<id>` per building, OR parse `/resources` page HTML | AJAX JSON / HTML |
| `GetFacilities` | Parse `/facilities` page HTML | HTML scrape |
| `GetShips` | Parse `/fleet` or `/shipyard` page HTML | HTML scrape |
| `GetDefence` | Parse `/defense` page HTML | HTML scrape |
| `GetFleets` | GET `/ajax/fleet/eventbox/fetch` for counts + GET `/ajax/fleet/eventlist/fetch` for details (HTML) | AJAX |
| `GetResearch` | Parse `/research` page HTML | HTML scrape |
| `GetServerSpeed` | Parse server settings or game constants from page | HTML scrape / hardcoded |
| `GetServerVersion` | Parse footer or page source | HTML scrape |
| `GetAttacks` | GET `/ajax/fleet/eventbox/fetch` → hostile missions, then cross-reference with eventlist | AJAX |
| `GetSlots` | Parse from `/fleet` page HTML | HTML scrape |
| `SendFleet` | POST `/ajax/fleet/dispatch/send-fleet` | AJAX POST |
| `CancelFleet` | POST `/ajax/fleet/dispatch/recall-fleet` | AJAX POST |
| `GetConstructions` | Parse build queue from `/resources` or `/facilities` page HTML | HTML scrape |
| `BuildBuilding` | POST `/resources/add-buildrequest` with `_token`, `technologyId` | Form POST |
| `GetGalaxyInfos` | POST `/ajax/galaxy` with galaxy/system coords | AJAX POST |
| `GetEspionageReportMessages` | Parse `/messages` page HTML (espionage tab) | HTML scrape |
| `GetEspionageReport` | GET `/ajax/messages/{id}` | AJAX JSON |
| `DeleteAllEspionageReports` | POST `/messages` with delete all action | Form POST |
| `GetCaptchaChallenge` | N/A — OGameX has no captchas | No-op (return empty) |
| `SolveCaptchaChallenge` | N/A | No-op |

### The dual-strategy problem: HTML scraping vs AJAX

OGameX is a server-rendered Laravel app. Game data is embedded in HTML pages. AJAX endpoints (`/ajax/resources`, `/ajax/facilities`, etc.) return JSON with **rendered HTML fragments inside**, not structured data. The exceptions are:

**Structured JSON endpoints (use directly):**
- `/ajax/fleet/eventbox/fetch` — fleet mission counts, next event timing
- `/ajax/fleet/dispatch/check-target` — target info, ship data, possible missions
- `/ajax/fleet/dispatch/send-fleet` — send fleet action
- `/ajax/fleet/dispatch/recall-fleet` — recall fleet action
- `/ajax/messages/{id}` — espionage report data
- `/ajax/galaxy` — galaxy scan results

**HTML-only pages (parse with Go HTML parser):**
- `/overview` — planet list, current resources, build queue
- `/resources` — building levels, build queue
- `/facilities` — facility levels, build queue
- `/research` — research levels, research queue
- `/shipyard` — ship counts, shipyard queue
- `/defense` — defense counts
- `/fleet` — ship counts, fleet slots, expedition slots
- `/fleet/movement` — active fleet missions with details

**Strategy: prefer structured JSON where available, fall back to HTML parsing.**

For HTML parsing, use Go's standard `golang.org/x/net/html` tokenizer. Each parser function takes an `io.Reader` (HTTP response body) and returns typed domain objects. This makes parsers testable with saved fixture HTML files.

## CSRF Token Management

### How OGameX CSRF works

OGameX uses Laravel's CSRF protection. The flow is:

1. **Login** returns HTML page with `<meta name="csrf-token" content="...">` + sets `XSRF-TOKEN` cookie
2. **Every AJAX POST** requires `_token` parameter matching the current CSRF token
3. **Every AJAX response** includes `newAjaxToken` field with a fresh CSRF token
4. Laravel rotates the CSRF token periodically (or on every POST in some configs)

### Token lifecycle in the client

```
Login → extract token from HTML meta tag or XSRF-TOKEN cookie
  │
  ▼
POST request → include _token=<current> in form body
  │
  ▼
Response → parse newAjaxToken from JSON body → update stored token
  │
  ▼
Next request → use updated token
```

### Implementation

```go
// session.go

func (c *Client) login(ctx context.Context, email, password string) error {
    // 1. GET /login → extract CSRF token from HTML meta tag
    // 2. POST /login with email, password, _token
    // 3. On success: session cookie set by cookie jar
    // 4. Extract CSRF token from response (meta tag or XSRF-TOKEN cookie)
}

func (c *Client) csrfToken() string {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.csrfToken
}

func (c *Client) setCSRFToken(token string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.csrfToken = token
}
```

### Token refresh from responses

The `doAJAX` and `doPOST` helpers automatically extract `newAjaxToken` from every JSON response:

```go
func (c *Client) doAJAX(ctx context.Context, method, path string, data url.Values) ([]byte, error) {
    // ... make request ...
    // Parse response JSON
    var resp struct {
        NewAjaxToken string `json:"newAjaxToken"`
        // ... other fields
    }
    json.Unmarshal(body, &resp)
    if resp.NewAjaxToken != "" {
        c.setCSRFToken(resp.NewAjaxToken)
    }
    return body, nil
}
```

### CSRF failure recovery

If a POST gets HTTP 419 (CSRF token mismatch) or the response indicates invalid token:
1. Re-login (GET `/login` → extract new token → POST `/login`)
2. Retry the original request once with the new token
3. If still fails, return error to caller

This is wrapped in a `withRetry` helper:

```go
func (c *Client) doPOST(ctx context.Context, path string, data url.Values) ([]byte, error) {
    data.Set("_token", c.csrfToken())
    resp, err := c.httpClient.PostForm(c.baseURL+path, data)
    if resp.StatusCode == 419 {
        c.login(ctx, c.email, c.password) // re-authenticate
        data.Set("_token", c.csrfToken())
        resp, err = c.httpClient.PostForm(c.baseURL+path, data)
    }
    // ... handle response ...
}
```

## Session Persistence

### Session lifecycle

Laravel sessions are cookie-based (`laravel_session` cookie). The Go `net/http/cookiejar` handles this automatically:

1. POST `/login` → response includes `Set-Cookie: laravel_session=...`
2. Cookie jar stores it
3. All subsequent requests automatically include the cookie
4. Session expires after Laravel's configured lifetime (typically 2 hours of inactivity on demo)

### Auto-reconnection

The client detects expired sessions via:
- HTTP 401 responses
- Redirects to `/login` page (Laravel middleware redirects unauthenticated requests)
- CSRF validation failures (419)

On detection, the client automatically re-authenticates:

```go
func (c *Client) ensureAuthenticated(ctx context.Context) error {
    // Try a lightweight request (e.g., GET /ajax/fleet/eventbox/fetch)
    // If redirected to /login or gets 401, re-login
    // Return error if re-login fails
}
```

### Config changes

```yaml
# Old config
ogamed:
  url: "http://ogamed:8080"

# New config
ogamex:
  url: "https://main.ogamex.dev"
  email: "${OGAMEX_EMAIL}"
  password: "${OGAMEX_PASSWORD}"
  universe: ""  # OGameX uses single-universe, may not be needed
```

The `AccountConfig` struct already has `Username` and `Password` fields — these map directly to OGameX's email/password login. The `OgamedConfig` struct gets replaced with `OGameXConfig`.

## Component Boundaries

```
┌───────────────────────────────────────────────────────────────┐
│                         cmd/bot/main.go                       │
│  Config → Logger → DB → OGameX Client → Login → State Mgr    │
│  → Workers (defender, builder, farmer) → Dashboard            │
└───────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────┐
│                     internal/ogamex/                          │
│  ┌─────────────────┐  ┌──────────────┐  ┌───────────────┐   │
│  │    client.go     │  │  session.go  │  │  transport.go │   │
│  │ ClientInterface  │  │ Login/Logout │  │ CSRF inject   │   │
│  │ implementation   │  │ Session mgmt │  │ Retry logic   │   │
│  └────────┬────────┘  └──────┬───────┘  └───────┬───────┘   │
│           │                  │                   │            │
│  ┌────────▼──────────────────▼───────────────────▼────────┐  │
│  │                    ajax.go                              │  │
│  │  doGET, doPOST, doAJAX — HTTP helpers with CSRF         │  │
│  └───────────────────────┬────────────────────────────────┘  │
│                          │                                    │
│  ┌───────────┐  ┌───────▼───────┐  ┌────────────────────┐   │
│  │ parser.go │  │   fleet.go    │  │    build.go        │   │
│  │ HTML →    │  │ SendFleet,    │  │ BuildBuilding,     │   │
│  │ domain    │  │ CancelFleet,  │  │ GetConstructions   │   │
│  │ types     │  │ GetFleets     │  │                    │   │
│  └───────────┘  └───────────────┘  └────────────────────┘   │
└───────────────────────────────────────────────────────────────┘

         ↕ ClientInterface ↕

┌───────────────────────────────────────────────────────────────┐
│              internal/state/manager.go                        │
│  Polls ClientInterface → caches to SQLite                     │
│  Workers read from SQLite, never call client directly*        │
└───────────────────────────────────────────────────────────────┘

         ↕ SQLite ↕

┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  defender    │  │   builder    │  │    farmer    │
│  (fleet-save)│  │ (auto-build) │  │ (auto-farm)  │
│              │  │              │  │              │
│ Uses client  │  │ Uses client  │  │ Uses client  │
│ for:         │  │ for:         │  │ for:         │
│ - GetAttacks │  │ - BuildBldg  │  │ - SendFleet  │
│ - SendFleet  │  │ - GetConstr  │  │ - GalaxyScan │
│ - CancelFlt  │  │              │  │ - Espionage  │
│ - GetShips   │  │              │  │              │
└──────────────┘  └──────────────┘  └──────────────┘

         ↕ WebSocket / REST ↕

┌───────────────────────────────────────────────────────────────┐
│              internal/dashboard/                              │
│  Go HTTP server + WebSocket + SolidJS frontend               │
│  Reads from state.Manager + SQLite (never calls client)      │
└───────────────────────────────────────────────────────────────┘
```

### Key boundary rules

1. **Workers call `ClientInterface` directly for actions** (SendFleet, BuildBuilding) but **read state from `StateReader`** (SQLite cache)
2. **State manager is the only component that polls for data** (GetPlanets, GetResources, etc.) on a 60-second cycle
3. **Dashboard never touches the client** — it reads from SQLite and state manager
4. **Client owns session management** — callers never see CSRF tokens or cookies

### Exception: Defender direct client calls

The defender bypasses the state manager for some calls because it needs **fresh real-time data**:
- `GetAttacks` — attacks are time-sensitive, cached data is stale
- `GetShips` — needs exact current ship count before fleet-save
- `GetSlots` — needs real-time slot availability

This pattern continues with the new client. These calls go through the same `ClientInterface` but hit OGameX directly instead of ogamed.

## Data Flow

### Login flow

```
main.go
  │
  ├─ ogamex.NewClient(url, log)
  ├─ client.Login(ctx)
  │   ├─ GET /login → extract CSRF token from <meta name="csrf-token">
  │   ├─ POST /login {email, password, _token}
  │   ├─ Cookie jar stores laravel_session cookie
  │   └─ Extract and store CSRF token from response
  │
  └─ Continue with state manager + workers
```

### State refresh flow (every 60 seconds)

```
state.Manager.refresh(ctx)
  │
  ├─ client.GetPlanets(ctx)
  │   └─ GET /overview?cp=<current_planet> → parse HTML for planet menu
  │      └─ Returns []model.Planet
  │
  ├─ For each planet:
  │   ├─ client.GetResources(ctx, planetID)
  │   │   └─ GET /overview?cp=<planetID> → parse resource bars in HTML
  │   │
  │   ├─ client.GetResourceBuildings(ctx, planetID)
  │   │   └─ GET /resources?cp=<planetID> → parse building levels from HTML
  │   │
  │   └─ client.GetFacilities(ctx, planetID)
  │       └─ GET /facilities?cp=<planetID> → parse facility levels from HTML
  │
  ├─ client.GetFleets(ctx)
  │   └─ GET /ajax/fleet/eventbox/fetch → JSON with fleet counts
  │   └─ GET /ajax/fleet/eventlist/fetch → parse HTML for fleet details
  │
  ├─ client.GetResearch(ctx)
  │   └─ GET /research → parse research levels from HTML
  │
  └─ Upsert all data into SQLite
```

### Fleet-send flow (defender)

```
defender.poll(ctx)
  │
  ├─ client.GetAttacks(ctx)
  │   └─ GET /ajax/fleet/eventbox/fetch → check hostile count
  │   └─ If hostile > 0: GET /ajax/fleet/eventlist/fetch → parse hostile fleet details
  │
  ├─ identifyEndangered(attacks, serverTime)
  │
  └─ savePlanet(ctx, planet, attacks, timeUntilAttack)
      ├─ (reaction delay)
      ├─ client.GetSlots(ctx)  → GET /fleet?cp=<planet> → parse slots from HTML
      ├─ client.GetShips(ctx, planetID) → GET /fleet?cp=<planet> → parse ship counts
      │
      ├─ CalcEscapeRoutes(...) → pure math, no HTTP
      │
      └─ client.SendFleet(ctx, req)
          ├─ POST /ajax/fleet/dispatch/check-target (validate target)
          ├─ POST /ajax/fleet/dispatch/send-fleet {ships, galaxy, system, position, type, mission, speed, metal, crystal, deuterium, _token}
          └─ Return fleet ID from response
```

### Build flow (builder)

```
builder.poll(ctx)
  │
  ├─ Read state from SQLite (planets, resources, buildings, research)
  │
  ├─ For each planet:
  │   └─ client.GetConstructions(ctx, planetID)
  │       └─ GET /resources?cp=<planet> → parse build queue from HTML
  │
  ├─ CalculateROI(...) → pure math
  │
  └─ client.BuildBuilding(ctx, planetID, buildingID)
      └─ POST /resources/add-buildrequest {technologyId: <id>, _token}
      └─ Parse response JSON for success/failure
```

## Suggested Build Order

The build order respects dependency chains: each phase produces a testable increment.

### Phase 1: Client shell + login

**Goal**: Authenticate with OGameX and maintain a session.

**Files to create/modify**:
- `internal/ogamex/client.go` — Client struct, constructor
- `internal/ogamex/session.go` — Login, CSRF token extraction
- `internal/ogamex/transport.go` — HTTP helpers with CSRF injection

**Test**: Login to OGameX demo server, verify session cookie is set, verify CSRF token is stored.

**Dependencies**: None (this is the foundation).

### Phase 2: HTML parser framework + planet list

**Goal**: Parse the overview page to extract the planet list.

**Files to create**:
- `internal/ogamex/parser.go` — Generic HTML parsing helpers
- `internal/ogamex/fixtures/overview.html` — Saved HTML for tests

**Modify**:
- `internal/ogamex/client.go` — Implement `GetPlanets()`

**Test**: Fetch `/overview`, parse planet list, verify correct number of planets with correct coordinates.

**Dependencies**: Phase 1 (need authenticated session).

### Phase 3: Resource and building parsers

**Goal**: Parse resources, buildings, facilities from page HTML.

**Files to create**:
- Fixture HTML files for resources, facilities, research pages

**Modify**:
- `internal/ogamex/client.go` — Implement `GetResources()`, `GetResourceBuildings()`, `GetFacilities()`, `GetResearch()`

**Test**: Fetch pages, parse data, verify against manual inspection.

**Dependencies**: Phase 2 (need planet list to switch planets with `?cp=`).

### Phase 4: Fleet operations

**Goal**: Read fleet state and send/recall fleets.

**Files to create**:
- `internal/ogamex/fleet.go` — Fleet dispatch, recall
- Fixture HTML for fleet movement page, eventlist

**Modify**:
- `internal/ogamex/client.go` — Implement `GetFleets()`, `SendFleet()`, `CancelFleet()`, `GetSlots()`, `GetAttacks()`, `IsUnderAttack()`

**Test**: Fetch eventbox JSON, parse fleet counts. Send a test transport between own planets. Recall it.

**Dependencies**: Phase 3 (need to be logged in, need planet IDs).

### Phase 5: Build operations

**Goal**: Queue building construction.

**Files to create**:
- `internal/ogamex/build.go` — Build request helpers

**Modify**:
- `internal/ogamex/client.go` — Implement `BuildBuilding()`, `GetConstructions()`

**Test**: Queue a build on test account, verify it appears in construction queue.

**Dependencies**: Phase 3 (need building level data to know what to build).

### Phase 6: Galaxy + espionage

**Goal**: Scan galaxy, read espionage reports.

**Files to create**:
- `internal/ogamex/galaxy.go` — Galaxy scan, espionage reports
- Fixture HTML for galaxy results, messages page

**Modify**:
- `internal/ogamex/client.go` — Implement `GetGalaxyInfos()`, `GetEspionageReportMessages()`, `GetEspionageReport()`, `DeleteAllEspionageReports()`

**Test**: Scan a galaxy range, read espionage messages.

**Dependencies**: Phase 4 (fleet operations needed for espionage probes).

### Phase 7: Integration + wiring swap

**Goal**: Replace ogamed client with ogamex client in main.go, verify all workers function.

**Files to modify**:
- `cmd/bot/main.go` — Swap `ogamed.NewClient(...)` → `ogamex.NewClient(...)`
- `internal/config/config.go` — Replace `OgamedConfig` with `OGameXConfig`
- `config.yaml` — Update config structure
- Remove captcha handling from main.go (OGameX has no captchas)
- Remove rate limiter creation from main.go
- Remove Docker dependency from deployment

**Test**: Full end-to-end test — login, state refresh, detect attacks, fleet-save, build, farm.

**Dependencies**: All previous phases.

### Phase 8: Cleanup

**Goal**: Remove dead code, update documentation.

- Delete `internal/ogamed/` package
- Remove ogamed-related config fields
- Remove Docker Compose files for ogamed
- Update AGENTS.md, PROJECT.md, README
- Remove rate limiter code (not needed for OGameX)

---

### Build order dependency graph

```
Phase 1 (login)
    │
    ▼
Phase 2 (planets)
    │
    ▼
Phase 3 (resources/buildings)
    │
    ├──► Phase 4 (fleet ops) ──► Phase 6 (galaxy/espionage)
    │                                    │
    ├──► Phase 5 (build ops)             │
    │                                    │
    └────────────────┬───────────────────┘
                     │
                     ▼
              Phase 7 (integration swap)
                     │
                     ▼
              Phase 8 (cleanup)
```

Phases 4, 5, and 6 can be parallelized once Phase 3 is complete, though they should be built sequentially for a single developer to maintain focus.
