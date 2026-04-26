# Phase 1: Core Infrastructure - Research

**Researched:** 2026-04-26
**Domain:** Go bot engine infrastructure (ogamed REST client, SQLite state cache, YAML config, rate limiting, Docker)
**Confidence:** HIGH

## Summary

Phase 1 builds the Go bot engine from scratch — connecting to the OGame REST API via ogamed, caching game state in SQLite, loading YAML config with env-var interpolation, throttling requests with random delays, and running the whole stack via Docker Compose. The project pivoted from TypeScript to Go, so existing TypeScript code in `packages/bot/` and `packages/shared/` serves as a **reference for data structures and API patterns** but will be replaced by Go implementations.

Go is the natural choice here: ogamed is itself a Go binary, the developer knows Go best, goroutines simplify concurrent polling, and single-binary deployment eliminates runtime dependency management. The standard Go library provides everything needed for logging (`log/slog`), HTTP client (`net/http`), and testing (`testing`). Only three external dependencies are needed: `modernc.org/sqlite` (pure Go SQLite driver), `gopkg.in/yaml.v3` (YAML parsing), and `github.com/golang-migrate/migrate/v4` (embedded SQL migrations).

**Primary recommendation:** Use the standard Go project layout (`cmd/bot/main.go` + `internal/` packages), keep external dependencies minimal, and port the proven patterns from the existing TypeScript implementation (rate limiter chokepoint, envelope validation, env-var interpolation) directly into idiomatic Go.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Go module for bot engine (`cmd/bot/` entrypoint, `internal/` packages). Separate pnpm workspace for dashboard (`packages/dashboard`, `packages/shared`).
- **D-02:** Go for bot engine with standard Go project layout. TypeScript for dashboard only (Phase 5).
- **D-03:** Shared types exist in Go packages (`internal/ogamed/types/`). Dashboard types will be generated from bot's REST API (OpenAPI/codegen in Phase 5).
- **D-04:** YAML config file (`config.yaml`) for user-facing bot settings. Use `gopkg.in/yaml.v3`.
- **D-05:** Config validated with Go struct tags + manual validation at startup. Invalid config = clear error + exit.
- **D-06:** Config structure: account credentials, ogamed connection settings, per-feature toggles and parameters, logging level.
- **D-07:** Secrets loaded from environment variables, referenced in config via `${ENV_VAR}` interpolation.
- **D-08:** SQLite via `modernc.org/sqlite` (pure Go, no CGo required) for all persistent state. Single-file database.
- **D-09:** Game state cached in SQLite tables (planets, resources, buildings, fleets, research). Updated on each poll cycle.
- **D-10:** Schema migrations via `golang-migrate/migrate` or embedded SQL migration files.
- **D-11:** Type-safe ogamed REST client in `internal/ogamed/`. Go structs for all request/response types.
- **D-12:** Validate ogamed responses match expected structure. Handle unknown/missing fields gracefully.
- **D-13:** Automatic retries with exponential backoff for transient failures (network errors, 5xx responses).
- **D-14:** Rate limiter wraps all ogamed calls. Minimum 1-3 second random delay between requests. Configurable per-endpoint.
- **D-15:** Docker Compose with two services: `ogamed` (official ogamed image) and `bot` (Go binary). Shared Docker network, bot calls ogamed via `http://ogamed:8080`.
- **D-16:** Environment-based configuration. `.env` file for secrets, `config.yaml` mounted as volume.
- **D-17:** `data/` directory mounted as persistent volume for SQLite database.
- **D-18:** Go `log/slog` structured logging (stdlib, no external dependency). JSON in production, text in dev.
- **D-19:** All ogamed API calls logged at debug level with request/response timing.

### Agent's Discretion
- Exact package structure within `internal/`
- Test framework selection (stdlib `testing` recommended)
- Specific SQLite schema design (table structure, indexes)
- Build setup for production Docker image (multi-stage Go build)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | Bot connects to ogamed REST API and maintains session across restarts | ogamed client with envelope validation, retry logic, login endpoint; ogamed handles session persistence via cookies internally |
| INFRA-02 | Bot retrieves and caches game state (planets, resources, fleets, buildings, research) | Go structs mapping ogamed JSON responses; SQLite tables for persistence; game state manager with poll-based refresh |
| INFRA-03 | Bot loads configuration from YAML file with feature toggles and per-feature parameters | `gopkg.in/yaml.v3` for parsing; env-var interpolation with regex; struct tag validation |
| INFRA-04 | Bot implements request throttling with random intervals between actions | Shared rate limiter with configurable min/max delay per endpoint; `time.Sleep` with random jitter |
| INFRA-05 | Bot runs as a Docker Compose stack (ogamed + bot) with environment-based config | Multi-stage Go Dockerfile; Docker Compose v2; env_file for secrets; volume mounts for data |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| OGame API communication | API / Backend (ogamed) | — | ogamed owns session management, anti-detection, device fingerprinting. Bot never talks to OGame directly. |
| Bot logic / scheduling | API / Backend (Go bot) | — | Go goroutines for concurrent polling, game state management, feature workers |
| Persistent state storage | Database / Storage (SQLite) | — | Single-file DB, zero-ops, co-located with bot process |
| Configuration | Filesystem (YAML) | Environment (secrets) | YAML for structured config, env vars for secrets only |
| Container orchestration | Docker Compose | — | Two-service stack: ogamed + bot on shared network |
| Logging | API / Backend (Go bot) | — | `log/slog` structured logging, no external dependency |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`net/http`) | 1.26 | HTTP client for ogamed REST API | Zero dependencies, production-proven, context support for cancellation/timeouts |
| Go stdlib (`log/slog`) | 1.26 | Structured logging | Built-in JSON handler, level filtering, no external dependency needed |
| Go stdlib (`testing`) | 1.26 | Test framework | No external test framework needed. `testify` adds assertions but stdlib is sufficient for a Go-expert developer |
| `modernc.org/sqlite` | v1.50.0 | Pure Go SQLite driver | CGo-free means simple cross-compilation, no C toolchain needed in Docker. `database/sql` compatible. [VERIFIED: go list -m] |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML config parsing | De-facto standard Go YAML library. Struct tag support. [VERIFIED: go list -m] |
| `github.com/golang-migrate/migrate/v4` | v4.19.1 | Embedded SQL migrations | Run migrations from Go code using `embed.FS`. No CLI dependency. SQLite driver built-in. [VERIFIED: go list -m] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | latest | Test assertions (optional) | If developer prefers fluent assertions over `if got != want` pattern. Consider for readability but not required. |
| Go stdlib (`embed`) | 1.26 | Embed migration SQL files in binary | Single-binary deployment — migrations travel with the executable |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `modernc.org/sqlite` | `mattn/go-sqlite3` | mattn requires CGo — adds build complexity, C toolchain in Docker. modernc is pure Go. [VERIFIED: go list -m] |
| `golang-migrate/migrate` | Hand-rolled migration runner | golang-migrate handles version tracking, up/down, checksums. Not worth hand-rolling. |
| `log/slog` | `zerolog` / `zap` | External loggers are faster in microbenchmarks but slog is stdlib, sufficient for a single-user bot, and has zero dependency overhead. |
| Go stdlib `net/http` | `resty` / `req` | Third-party HTTP clients add sugar but `net/http` is fully capable and the developer knows it best. |

**Installation:**
```bash
# Initialize Go module (from project root)
go mod init github.com/user/ogame-bot

# Core dependencies
go get modernc.org/sqlite@v1.50.0
go get gopkg.in/yaml.v3@v3.0.1
go get github.com/golang-migrate/migrate/v4@v4.19.1

# Migration source drivers
go get github.com/golang-migrate/migrate/v4/source/iofs
go get github.com/golang-migrate/migrate/v4/database/sqlite3

# Optional test assertions
go get github.com/stretchr/testify
```

**Version verification (2026-04-26):**
```
modernc.org/sqlite      v1.50.0   [VERIFIED: go list -m]
gopkg.in/yaml.v3        v3.0.1    [VERIFIED: go list -m]
golang-migrate/migrate  v4.19.1   [VERIFIED: go list -m]
Go runtime              1.26.2    [VERIFIED: go version on host]
```

## Architecture Patterns

### System Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                    Docker Compose Network                      │
│                                                                │
│  ┌──────────────────────┐    ┌──────────────────────────────┐ │
│  │     ogamed           │    │       Go Bot Engine          │ │
│  │  (Go binary)         │    │                              │ │
│  │                      │REST│  cmd/bot/main.go             │ │
│  │  Handles:            │◄───│         │                    │ │
│  │  • Login/sessions    │    │  internal/                  │ │
│  │  • Anti-detection    │    │    ├─ config/               │ │
│  │  • Device fingerprint│    │    │   └─ config.go         │ │
│  │  • Captcha           │    │    ├─ ogamed/               │ │
│  │  • Cookie mgmt       │    │    │   ├─ client.go         │ │
│  │                      │    │    │   ├─ types.go           │ │
│  │  Port 8080           │    │    │   ├─ rate_limiter.go   │ │
│  │                      │    │    │   └─ retry.go           │ │
│  └──────────────────────┘    │    ├─ state/                │ │
│                               │    │   ├─ manager.go        │ │
│                               │    │   └─ db.go             │ │
│                               │    ├─ model/                │ │
│                               │    │   └─ types.go          │ │
│                               │    └─ migrations/           │ │
│                               │        └─ 001_init.sql      │ │
│                               │                              │ │
│                               │  ┌─────────┐  ┌───────────┐ │ │
│                               │  │ SQLite  │  │config.yaml│ │ │
│                               │  │  DB     │  │  (.env)   │ │ │
│                               │  └─────────┘  └───────────┘ │ │
│                               └──────────────────────────────┘ │
│                                                                │
│  Volumes:                                                      │
│    ./data/     → /app/data/    (SQLite database)               │
│    ./config.yaml → /app/config.yaml  (bot configuration)       │
│    ./.env      → env_file      (secrets)                       │
└──────────────────────────────────────────────────────────────┘
         │
         │  (ogamed communicates directly with OGame servers)
         ▼
   ┌───────────────┐
   │  OGame Servers │
   └───────────────┘
```

### Recommended Project Structure
```
ogame/
├── cmd/
│   └── bot/
│       └── main.go              # Entrypoint: load config, init DB, start client, run main loop
├── internal/
│   ├── config/
│   │   └── config.go            # YAML loading, env-var interpolation, validation
│   ├── ogamed/
│   │   ├── client.go            # HTTP client wrapping ogamed REST API
│   │   ├── types.go             # Go structs for ogamed request/response types
│   │   ├── rate_limiter.go      # Shared rate limiter with random delay
│   │   └── retry.go             # Exponential backoff with jitter
│   ├── state/
│   │   ├── manager.go           # Game state manager: poll, cache, expose state
│   │   └── db.go                # SQLite connection, migrations, query helpers
│   └── model/
│       └── types.go             # Domain types: Coordinate, Resources, Planet, Fleet, etc.
├── migrations/
│   └── 001_init.sql             # Initial schema: planets, resources, fleets, research tables
├── Dockerfile                   # Multi-stage Go build
├── docker-compose.yml           # ogamed + bot services
├── go.mod
├── go.sum
├── config.example.yaml          # Example config (existing, keep)
├── .env.example                 # Example env vars (existing, keep)
│
│   # Existing TS workspace (kept for Phase 5 dashboard)
├── packages/
│   ├── dashboard/               # SolidJS dashboard (Phase 5)
│   └── shared/                  # TS types (reference only, will be replaced by Go types + OpenAPI in Phase 5)
├── package.json                 # pnpm workspace root (kept for dashboard)
├── pnpm-workspace.yaml
└── tsconfig.base.json
```

### Pattern 1: Ogamed REST Client with Envelope Validation
**What:** A Go struct with methods mapping 1:1 to ogamed REST endpoints. Every call goes through envelope validation, rate limiting, and retry.
**When to use:** All OGame API interaction.
**Example:**
```go
// Source: Based on existing TS ogamed-client.ts, ported to Go patterns
// + ogamed wiki: https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation

package ogamed

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    "log/slog"
)

// OgamedResponse is the standard envelope for ALL ogamed REST responses.
// Every endpoint returns: {"Status":"ok"|"error","Code":200,"Message":"","Result":...}
type OgamedResponse[T any] struct {
    Status  string `json:"Status"`
    Code    int    `json:"Code"`
    Message string `json:"Message"`
    Result  T      `json:"Result"`
}

// Client wraps all ogamed REST API calls with rate limiting and retry.
type Client struct {
    baseURL     string
    httpClient  *http.Client
    rateLimiter *RateLimiter
    log         *slog.Logger
}

func NewClient(baseURL string, limiter *RateLimiter, log *slog.Logger) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        rateLimiter: limiter,
        log:         log.With("component", "ogamed-client"),
    }
}

// get performs a GET request with rate limiting, retry, and envelope validation.
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
    if err := c.rateLimiter.Wait(ctx, path); err != nil {
        return nil, fmt.Errorf("rate limiter: %w", err)
    }

    var body []byte
    err := retryWithBackoff(ctx, func() error {
        start := time.Now()
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
        if err != nil {
            return err
        }
        resp, err := c.httpClient.Do(req)
        if err != nil {
            return fmt.Errorf("HTTP request failed: %w", err)
        }
        defer resp.Body.Close()
        body, err = io.ReadAll(resp.Body)
        if err != nil {
            return fmt.Errorf("reading response body: %w", err)
        }
        c.log.Debug("API call completed",
            "path", path,
            "duration_ms", time.Since(start).Milliseconds(),
            "status", resp.StatusCode,
        )
        return nil
    })
    return body, err
}

// Login authenticates with ogamed. ogamed handles session persistence internally.
func (c *Client) Login(ctx context.Context) error {
    _, err := c.get(ctx, "/bot/login")
    return err
}

// IsUnderAttack checks if the account is currently under attack.
func (c *Client) IsUnderAttack(ctx context.Context) (bool, error) {
    return c.getBool(ctx, "/bot/is-under-attack")
}

// GetPlanets retrieves all planets belonging to the account.
func (c *Client) GetPlanets(ctx context.Context) ([]Planet, error) {
    var result []Planet
    err := c.getTyped(ctx, "/bot/planets", &result)
    return result, err
}
```

### Pattern 2: YAML Config with Env-Var Interpolation
**What:** Load YAML config, interpolate `${ENV_VAR}` references from environment, validate with Go struct tags.
**When to use:** Startup configuration loading.
**Example:**
```go
// Source: Based on existing TS config-loader.ts, ported to Go
// + gopkg.in/yaml.v3 patterns [VERIFIED: Context7]

package config

import (
    "fmt"
    "os"
    "regexp"
    "log/slog"

    "gopkg.in/yaml.v3"
)

// Config is the top-level bot configuration.
// Validated at startup — invalid config = clear error + exit.
type Config struct {
    Account  AccountConfig  `yaml:"account"`
    Ogamed   OgamedConfig   `yaml:"ogamed"`
    Features FeaturesConfig `yaml:"features"`
    RateLimit RateLimitConfig `yaml:"rateLimit"`
    LogLevel string         `yaml:"logLevel"`
}

type AccountConfig struct {
    Universe string `yaml:"universe"`
    Username string `yaml:"username"`
    Password string `yaml:"password"` // Will contain ${OGAME_PASSWORD}, interpolated at load
}

type OgamedConfig struct {
    URL string `yaml:"url"`
}

type FeaturesConfig struct {
    Defender  FeatureConfig `yaml:"defender"`
    AutoBuild FeatureConfig `yaml:"autoBuild"`
    AutoFarm  FeatureConfig `yaml:"autoFarm"`
}

type FeatureConfig struct {
    Enabled        bool `yaml:"enabled"`
    PollIntervalMs int  `yaml:"pollIntervalMs"`
}

type RateLimitConfig struct {
    DefaultMinDelayMs int                              `yaml:"defaultMinDelayMs"`
    DefaultMaxDelayMs int                              `yaml:"defaultMaxDelayMs"`
    EndpointOverrides map[string]EndpointDelayConfig   `yaml:"endpointOverrides"`
}

type EndpointDelayConfig struct {
    MinMs int `yaml:"minMs"`
    MaxMs int `yaml:"maxMs"`
}

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

func Load(path string, log *slog.Logger) (*Config, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config file: %w", err)
    }

    // Interpolate ${ENV_VAR} references from environment
    interpolated := envVarPattern.ReplaceAllStringFunc(string(raw), func(match string) string {
        varName := envVarPattern.FindStringSubmatch(match)[1]
        value, ok := os.LookupEnv(varName)
        if !ok {
            log.Error("Environment variable referenced in config but not set", "var", varName)
            os.Exit(1)
        }
        return value
    })

    var cfg Config
    if err := yaml.Unmarshal([]byte(interpolated), &cfg); err != nil {
        return nil, fmt.Errorf("parsing YAML config: %w", err)
    }

    // Manual validation (struct tags don't cover cross-field validation)
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &cfg, nil
}

func (c *Config) Validate() error {
    if c.Account.Universe == "" {
        return fmt.Errorf("account.universe is required")
    }
    if c.Account.Username == "" {
        return fmt.Errorf("account.username is required")
    }
    if c.Account.Password == "" {
        return fmt.Errorf("account.password is required")
    }
    if c.Ogamed.URL == "" {
        return fmt.Errorf("ogamed.url is required")
    }
    if c.RateLimit.DefaultMinDelayMs < 500 {
        return fmt.Errorf("rateLimit.defaultMinDelayMs must be >= 500ms")
    }
    return nil
}
```

### Pattern 3: Shared Rate Limiter with Random Delay
**What:** A goroutine-safe rate limiter that enforces random delay intervals between ogamed API calls.
**When to use:** All ogamed API calls — this is the single chokepoint.
**Example:**
```go
// Source: Based on existing TS rate-limiter.ts, ported to Go

package ogamed

import (
    "context"
    "math/rand"
    "sync"
    "time"
)

type RateLimiter struct {
    mu         sync.Mutex
    lastCall   time.Time
    config     RateLimitConfig
}

func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
    return &RateLimiter{config: cfg}
}

// Wait blocks until the minimum delay since the last API call has elapsed.
// The delay includes random jitter for anti-detection.
func (r *RateLimiter) Wait(ctx context.Context, endpoint string) error {
    r.mu.Lock()

    override, hasOverride := r.config.EndpointOverrides[endpoint]
    minDelay := r.config.DefaultMinDelayMs
    maxDelay := r.config.DefaultMaxDelayMs
    if hasOverride {
        minDelay = override.MinMs
        maxDelay = override.MaxMs
    }

    // Random delay within [minDelay, maxDelay]
    jitteredDelay := time.Duration(minDelay+rand.Intn(maxDelay-minDelay)) * time.Millisecond
    elapsed := time.Since(r.lastCall)
    waitTime := jitteredDelay - elapsed

    r.mu.Unlock()

    if waitTime > 0 {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(waitTime):
        }
    }

    r.mu.Lock()
    r.lastCall = time.Now()
    r.mu.Unlock()
    return nil
}
```

### Pattern 4: SQLite State with Embedded Migrations
**What:** Open SQLite database, run embedded migrations on startup, provide query helpers for game state.
**When to use:** All persistent state access.
**Example:**
```go
// Source: modernc.org/sqlite patterns [VERIFIED: Context7]
// + golang-migrate/migrate patterns [VERIFIED: Context7]

package state

import (
    "database/sql"
    "embed"
    "fmt"
    "log/slog"

    _ "modernc.org/sqlite"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/source/iofs"
    "github.com/golang-migrate/migrate/v4/database/sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func OpenDB(dbPath string, log *slog.Logger) (*sql.DB, error) {
    // Open with WAL mode for better concurrent read performance
    dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, fmt.Errorf("opening database: %w", err)
    }

    // Connection pool settings for single-user SQLite
    db.SetMaxOpenConns(1) // SQLite only allows one writer at a time
    db.SetMaxIdleConns(1)

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("pinging database: %w", err)
    }

    // Run migrations
    if err := runMigrations(db, log); err != nil {
        return nil, fmt.Errorf("running migrations: %w", err)
    }

    return db, nil
}

func runMigrations(db *sql.DB, log *slog.Logger) error {
    sourceDriver, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return fmt.Errorf("creating migration source: %w", err)
    }

    dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
    if err != nil {
        return fmt.Errorf("creating migration db driver: %w", err)
    }

    m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
    if err != nil {
        return fmt.Errorf("creating migrator: %w", err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("applying migrations: %w", err)
    }

    log.Info("Database migrations applied")
    return nil
}
```

### Anti-Patterns to Avoid
- **Multiple HTTP clients to ogamed:** All API calls MUST go through a single `Client` instance with a shared `RateLimiter`. Multiple clients = uncoordinated requests = rate limit violations. [CITED: PITFALLS.md Pitfall 6]
- **Polling ogamed directly from feature workers:** Workers read from cached state, not from ogamed. Only the `StateManager` refreshes from ogamed. [CITED: ARCHITECTURE.md Anti-Pattern 1]
- **CGo dependencies:** Use `modernc.org/sqlite` only. Any CGo dependency (mattn/go-sqlite3, sqlite3.h) breaks cross-compilation and inflates the Docker image. [VERIFIED: Context7 modernc docs]
- **Ignoring context cancellation:** Every HTTP call and long-running operation must accept `context.Context` for graceful shutdown. [ASSUMED] Go best practice.
- **Global mutable state:** Use dependency injection (pass structs to constructors) rather than package-level variables. Makes testing straightforward.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SQLite driver | C bindings / custom driver | `modernc.org/sqlite` via `database/sql` | Pure Go, no CGo, drop-in for `database/sql` interface |
| Database migrations | Custom version tracker | `golang-migrate/migrate` with `embed.FS` | Handles versioning, checksums, up/down. Embedded in binary. |
| YAML parsing | Custom YAML parser | `gopkg.in/yaml.v3` | De-facto standard, struct tag support, well-tested |
| Structured logging | Custom log formatter | `log/slog` (stdlib) | JSON handler built-in, level filtering, child loggers |
| HTTP client | Custom HTTP wrapper around raw TCP | `net/http` (stdlib) | Production-grade, context support, connection pooling |
| Retry logic | Custom loop without backoff | Port existing TS `retryWithBackoff` pattern | Exponential backoff + jitter is deceptively complex to get right |
| Rate limiting | `time.Sleep(fixed)` in each caller | Shared `RateLimiter` struct | Centralized delay tracking, per-endpoint config, thread-safe |

**Key insight:** Go's standard library is extremely capable. The bot needs only 3 external packages (sqlite, yaml, migrate). Every other problem is solvable with stdlib.

## Common Pitfalls

### Pitfall 1: SQLite Concurrent Access Pattern
**What goes wrong:** Opening multiple write connections or not setting `MaxOpenConns(1)` causes "database is locked" errors.
**Why it happens:** SQLite only supports one writer at a time. Go's `database/sql` pool opens multiple connections by default.
**How to avoid:** Set `db.SetMaxOpenConns(1)` and use WAL mode (`_pragma=journal_mode(WAL)`). This is standard for single-user Go+SQLite apps. [VERIFIED: Context7 modernc docs]
**Warning signs:** "database is locked" errors under load; writes timing out.

### Pitfall 2: ogamed JSON Field Name Mismatch
**What goes wrong:** ogamed returns PascalCase JSON fields (`"MetalMine"`, `"PlayerID"`, `"ReturnFlight"`), but Go convention is to use snake_case struct tags.
**Why it happens:** ogamed is a Go project that serializes its Go structs directly (PascalCase). Your Go structs must explicitly tag `json:"MetalMine"` not `json:"metal_mine"`.
**How to avoid:** Always match ogamed's exact JSON field names. Reference the wiki examples: `{"Status":"ok","Code":200,"Message":"","Result":{...}}`. [VERIFIED: ogamed wiki full documentation]
**Warning signs:** All struct fields deserializing as zero values; no errors but wrong data.

### Pitfall 3: Forgetting ogamed Login Before API Calls
**What goes wrong:** All API calls return errors because ogamed hasn't logged in yet.
**Why it happens:** ogamed requires `GET /bot/login` before any game API calls. It auto-logins if `OGAMED_AUTO_LOGIN=true` (set in Docker env), but the bot should still call login explicitly to verify connectivity.
**How to avoid:** Always call `client.Login(ctx)` as the first operation after startup. Verify login succeeded. If ogamed restarts, the bot must re-login. [VERIFIED: ogamed wiki + Dockerfile OGAMED_AUTO_LOGIN env var]
**Warning signs:** 401/403 responses from ogamed; `Status: "error"` in responses.

### Pitfall 4: Not Using Context for Graceful Shutdown
**What goes wrong:** Bot process hangs on shutdown because a goroutine is blocked on an HTTP call with no context cancellation.
**Why it happens:** Using `context.Background()` instead of a cancellable context, or passing `nil` context to HTTP requests.
**How to avoid:** Create a root context with `context.WithCancel(context.Background())`. Cancel it on SIGINT/SIGTERM. Pass this context to ALL HTTP calls and long-running operations. [ASSUMED] Go best practice.
**Warning signs:** `docker compose down` takes 30+ seconds (default timeout); hanging goroutines on restart.

### Pitfall 5: ogamed Response Validation Blind Spots
**What goes wrong:** OGame game updates change API response structure, ogamed breaks silently, bot stores garbage data in SQLite.
**Why it happens:** ogamed scrapes HTML — when Gameforge changes page structure, extractors break (ogamed issues #148, #150). Bot has no way to know the data is wrong.
**How to avoid:** Validate critical fields in responses. If `Resources.Metal` is negative, `Planet.ID` is 0, or timestamps parse to year 0001 — treat as API broken, log error, don't update state. [CITED: PITFALLS.md Pitfall 2]
**Warning signs:** Zero values in cached state; timestamps with year 0001; sudden planet count changes.

### Pitfall 6: Docker Networking Misconfiguration
**What goes wrong:** Bot container can't reach ogamed at `http://ogamed:8080`.
**Why it happens:** Docker Compose services must be on the same network. Using `localhost` instead of the service name. ogamed binding to `127.0.0.1` instead of `0.0.0.0`.
**How to avoid:** Use Docker Compose default network. ogamed Dockerfile already sets `OGAMED_HOST=0.0.0.0`. Bot config should reference `http://ogamed:8080` (Docker service name). [VERIFIED: ogamed Dockerfile]
**Warning signs:** Connection refused errors; bot logs show "dial tcp 127.0.0.1:8080: connect: connection refused".

## Code Examples

### Main Entrypoint
```go
// cmd/bot/main.go
// Source: Standard Go CLI pattern + project-specific initialization

package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"

    "github.com/user/ogame-bot/internal/config"
    "github.com/user/ogame-bot/internal/ogamed"
    "github.com/user/ogame-bot/internal/state"
)

func main() {
    // 1. Load config
    log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
    cfg, err := config.Load("config.yaml", log)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
        os.Exit(1)
    }

    // 2. Setup structured logging based on config
    level := parseLogLevel(cfg.LogLevel)
    log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

    // 3. Open SQLite database
    dbPath := filepath.Join("data", "bot.db")
    if err := os.MkdirAll("data", 0755); err != nil {
        log.Error("Failed to create data directory", "error", err)
        os.Exit(1)
    }
    db, err := state.OpenDB(dbPath, log)
    if err != nil {
        log.Error("Failed to open database", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    // 4. Create ogamed client with rate limiter
    rateLimiter := ogamed.NewRateLimiter(cfg.RateLimit)
    client := ogamed.NewClient(cfg.Ogamed.URL, rateLimiter, log)

    // 5. Login to ogamed
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    if err := client.Login(ctx); err != nil {
        log.Error("Failed to login to ogamed", "error", err)
        os.Exit(1)
    }

    // 6. Start game state manager
    stateMgr := state.NewManager(db, client, log)
    go stateMgr.Run(ctx)

    // 7. Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Info("Shutting down gracefully...")
    cancel()
}
```

### Multi-Stage Dockerfile for Go Bot
```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bot ./cmd/bot

# Runtime stage
FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /bot /app/bot
COPY config.example.yaml /app/config.example.yaml

WORKDIR /app
EXPOSE 0
CMD ["./bot"]
```

### Docker Compose
```yaml
# docker-compose.yml
services:
  ogamed:
    build: https://github.com/alaingilbert/ogame.git
    environment:
      OGAMED_UNIVERSE: ${OGAMED_UNIVERSE}
      OGAMED_USERNAME: ${OGAMED_USERNAME}
      OGAMED_PASSWORD: ${OGAMED_PASSWORD}
      OGAMED_LANGUAGE: ${OGAMED_LANGUAGE:-en}
      OGAMED_HOST: "0.0.0.0"
      OGAMED_PORT: "8080"
      OGAMED_AUTO_LOGIN: "true"
      OGAMED_PROXY: ${OGAMED_PROXY:-}
      OGAMED_PROXY_TYPE: ${OGAMED_PROXY_TYPE:-socks5}
    ports:
      - "8080:8080"  # Expose for debugging; remove in production

  bot:
    build: .
    depends_on:
      - ogamed
    env_file: .env
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/app/data
    environment:
      OGAME_PASSWORD: ${OGAME_PASSWORD}
```

### Initial Migration
```sql
-- migrations/001_init.sql
-- Source: Based on ogamed API response structures [VERIFIED: ogamed wiki]

CREATE TABLE IF NOT EXISTS planets (
    id              INTEGER PRIMARY KEY,
    name            TEXT NOT NULL,
    galaxy          INTEGER NOT NULL,
    system          INTEGER NOT NULL,
    position        INTEGER NOT NULL,
    is_moon         BOOLEAN NOT NULL DEFAULT FALSE,
    diameter        INTEGER NOT NULL DEFAULT 0,
    fields_used     INTEGER NOT NULL DEFAULT 0,
    fields_total    INTEGER NOT NULL DEFAULT 0,
    temperature_min INTEGER NOT NULL DEFAULT 0,
    temperature_max INTEGER NOT NULL DEFAULT 0,
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS resources (
    planet_id   INTEGER PRIMARY KEY REFERENCES planets(id),
    metal       INTEGER NOT NULL DEFAULT 0,
    crystal     INTEGER NOT NULL DEFAULT 0,
    deuterium   INTEGER NOT NULL DEFAULT 0,
    energy      INTEGER NOT NULL DEFAULT 0,
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS buildings (
    planet_id               INTEGER PRIMARY KEY REFERENCES planets(id),
    metal_mine              INTEGER NOT NULL DEFAULT 0,
    crystal_mine            INTEGER NOT NULL DEFAULT 0,
    deuterium_synthesizer   INTEGER NOT NULL DEFAULT 0,
    solar_plant             INTEGER NOT NULL DEFAULT 0,
    fusion_reactor          INTEGER NOT NULL DEFAULT 0,
    metal_storage           INTEGER NOT NULL DEFAULT 0,
    crystal_storage         INTEGER NOT NULL DEFAULT 0,
    deuterium_tank          INTEGER NOT NULL DEFAULT 0,
    updated_at              DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS facilities (
    planet_id           INTEGER PRIMARY KEY REFERENCES planets(id),
    robotics_factory    INTEGER NOT NULL DEFAULT 0,
    shipyard            INTEGER NOT NULL DEFAULT 0,
    research_lab        INTEGER NOT NULL DEFAULT 0,
    nanite_factory      INTEGER NOT NULL DEFAULT 0,
    terraformer         INTEGER NOT NULL DEFAULT 0,
    updated_at          DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS research (
    id                          INTEGER PRIMARY KEY CHECK (id = 1),  -- Singleton row
    energy_technology           INTEGER NOT NULL DEFAULT 0,
    laser_technology            INTEGER NOT NULL DEFAULT 0,
    ion_technology              INTEGER NOT NULL DEFAULT 0,
    plasma_technology           INTEGER NOT NULL DEFAULT 0,
    combustion_drive            INTEGER NOT NULL DEFAULT 0,
    impulse_drive               INTEGER NOT NULL DEFAULT 0,
    hyperspace_drive            INTEGER NOT NULL DEFAULT 0,
    espionage_technology        INTEGER NOT NULL DEFAULT 0,
    computer_technology         INTEGER NOT NULL DEFAULT 0,
    astrophysics                INTEGER NOT NULL DEFAULT 0,
    intergalactic_research_net  INTEGER NOT NULL DEFAULT 0,
    weapons_technology          INTEGER NOT NULL DEFAULT 0,
    shielding_technology        INTEGER NOT NULL DEFAULT 0,
    armour_technology           INTEGER NOT NULL DEFAULT 0,
    updated_at                  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS fleets (
    id              INTEGER PRIMARY KEY,
    mission         INTEGER NOT NULL,
    return_flight   BOOLEAN NOT NULL DEFAULT FALSE,
    origin_galaxy   INTEGER NOT NULL,
    origin_system   INTEGER NOT NULL,
    origin_position INTEGER NOT NULL,
    dest_galaxy     INTEGER NOT NULL,
    dest_system     INTEGER NOT NULL,
    dest_position   INTEGER NOT NULL,
    metal           INTEGER NOT NULL DEFAULT 0,
    crystal         INTEGER NOT NULL DEFAULT 0,
    deuterium       INTEGER NOT NULL DEFAULT 0,
    arrival_time    INTEGER NOT NULL DEFAULT 0,
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CGo SQLite (`mattn/go-sqlite3`) | Pure Go SQLite (`modernc.org/sqlite`) | ~2022+ mature | No C toolchain needed, simpler Docker builds, cross-compilation works |
| Third-party loggers required | `log/slog` in stdlib | Go 1.21 (2023) | No external dependency for structured logging |
| Manual migration SQL execution | `golang-migrate/migrate` with `embed.FS` | Ongoing | Single-binary deployment with embedded migrations |
| TypeScript bot engine | Go bot engine | 2026-04-26 pivot | Shares language with ogamed, goroutines for concurrency, single binary |

**Deprecated/outdated:**
- `mattn/go-sqlite3`: Still maintained but requires CGo. Use `modernc.org/sqlite` for pure Go. [VERIFIED: Context7]
- Go `log` package: Replaced by `log/slog` for structured logging in Go 1.21+. [VERIFIED: Go stdlib]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `docker compose` (v2, as Go plugin) is available and works — verified: Docker Compose v5.1.3 on host | Environment Availability | LOW — verified on host |
| A2 | ogamed Docker image can be built from GitHub repo directly in Docker Compose `build` directive | Docker | MEDIUM — may need to use a pre-built image or local Dockerfile |
| A3 | `modernc.org/sqlite` supports WAL mode via pragma in DSN | Standard Stack | LOW — verified via Context7 docs showing pragma support |
| A4 | `golang-migrate/migrate` has a SQLite3 database driver compatible with `modernc.org/sqlite` | Standard Stack | MEDIUM — needs verification at implementation time; the `sqlite3` driver in migrate may expect the CGo driver |
| A5 | Go 1.26 is compatible with all listed dependencies | Standard Stack | LOW — all libraries support Go 1.22+ per their go.mod |

**Key assumption requiring validation:** A4 — `golang-migrate/migrate`'s `sqlite3` database driver uses `github.com/mattn/go-sqlite3` under the hood. Since we use `modernc.org/sqlite` (pure Go), we may need to use the migrate library differently — either: (a) pass the `*sql.DB` instance directly via `sqlite3.WithInstance()` which should work with any `database/sql` driver, or (b) use an alternative migration approach if the driver type is checked. The `WithInstance` approach should work since both register as `database/sql` drivers, but this needs testing. [ASSUMED]

## Open Questions

1. **ogamed Docker image source**
   - What we know: ogamed has a Dockerfile in its repo. No pre-built image on Docker Hub (404 on hub.docker.com/r/alaingilbert/ogamed).
   - What's unclear: Best way to reference it in docker-compose.yml — `build: https://github.com/alaingilbert/ogame.git` may not work directly since the Dockerfile is in the repo root but the go.mod context matters.
   - Recommendation: Clone/copy the ogamed Dockerfile locally, or build the ogamed image separately and reference it. Consider adding a `Dockerfile.ogamed` or using a git submodule.

2. **golang-migrate SQLite driver compatibility with modernc**
   - What we know: golang-migrate has a `sqlite3` driver. modernc.org/sqlite registers as `"sqlite"` driver name (not `"sqlite3"`).
   - What's unclear: Whether `migrate`'s `sqlite3` driver can open a `*sql.DB` that was opened with `modernc.org/sqlite`.
   - Recommendation: Use `migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)` where `dbDriver` is created via `sqlite3.WithInstance(db)`. If this doesn't work, fall back to running migrations with raw `db.Exec()` calls (simplest alternative for a small project).

3. **Existing TypeScript code fate**
   - What we know: `packages/bot/` has 6 TypeScript files implementing client, config, rate limiter, retry, logger. `packages/shared/` has types and schemas.
   - What's unclear: Should the TS code be deleted, or kept for reference until Go replacement is complete?
   - Recommendation: Keep TS code during transition. Delete `packages/bot/` once Go replacement is verified working. Keep `packages/shared/` for dashboard (Phase 5) — types will be replaced by OpenAPI codegen later.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go runtime | Bot engine | ✓ | 1.26.2 | — |
| Docker | Containerization | ✓ | 29.4.1 | — |
| Docker Compose | Stack orchestration | ✓ | 5.1.3 | — |
| Node.js | Dashboard (Phase 5) | ✓ | 25.9.0 | — |
| pnpm | Dashboard workspace | ✓ | 10.33.2 | — |
| SQLite (modernc) | State storage | ✓ (Go dep) | v1.50.0 | — |
| gcc/CGo | Not needed | N/A | N/A | Pure Go — no CGo required |

**Missing dependencies with no fallback:**
- None — all required tools are available on the host.

**Missing dependencies with fallback:**
- None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + optional `testify` assertions |
| Config file | None — Go tests follow `*_test.go` convention |
| Quick run command | `go test ./internal/... -v -count=1` |
| Full suite command | `go test ./... -v -race -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-01 | ogamed client connects, login succeeds, retry on failure | unit | `go test ./internal/ogamed/... -run TestClient -v` | ❌ Wave 0 |
| INFRA-01 | Session survives bot restart | integration | `go test ./internal/ogamed/... -run TestSession -v` | ❌ Wave 0 |
| INFRA-02 | Game state cached in SQLite, manager polls and updates | unit | `go test ./internal/state/... -run TestManager -v` | ❌ Wave 0 |
| INFRA-02 | Planet/fleet/resource structs deserialize correctly from ogamed JSON | unit | `go test ./internal/ogamed/... -run TestUnmarshal -v` | ❌ Wave 0 |
| INFRA-03 | YAML config loads, env vars interpolated, invalid config rejected | unit | `go test ./internal/config/... -run TestLoad -v` | ❌ Wave 0 |
| INFRA-04 | Rate limiter enforces min delay, random jitter, per-endpoint overrides | unit | `go test ./internal/ogamed/... -run TestRateLimiter -v` | ❌ Wave 0 |
| INFRA-05 | Docker Compose starts both containers, bot reaches ogamed | manual | `docker compose up --build && docker compose logs bot` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/... -v -count=1`
- **Per wave merge:** `go test ./... -v -race -count=1`
- **Phase gate:** Full suite green + `docker compose up --build` smoke test passing

### Wave 0 Gaps
- [ ] `internal/config/config_test.go` — covers INFRA-03
- [ ] `internal/ogamed/client_test.go` — covers INFRA-01
- [ ] `internal/ogamed/types_test.go` — covers INFRA-02 (deserialization)
- [ ] `internal/ogamed/rate_limiter_test.go` — covers INFRA-04
- [ ] `internal/ogamed/retry_test.go` — covers retry behavior
- [ ] `internal/state/manager_test.go` — covers INFRA-02 (state caching)
- [ ] `internal/state/db_test.go` — covers SQLite operations
- [ ] Framework install: `go mod init` — needs to be run (no go.mod exists yet)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | ogamed handles OGame authentication. Bot authenticates to ogamed via Docker network (localhost only). |
| V3 Session Management | yes | ogamed manages sessions. Bot must re-login on ogamed restart. |
| V4 Access Control | no | Single-user bot, no multi-user access control needed. |
| V5 Input Validation | yes | Go struct tags + manual validation for config. Response validation for ogamed JSON. |
| V6 Cryptography | no | No custom crypto needed. HTTPS to OGame handled by ogamed. |

### Known Threat Patterns for Go Bot + Docker

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Credential leak in git | Information Disclosure | `.gitignore` for `config.yaml`, `.env`; env-var interpolation for secrets |
| ogamed port exposed publicly | Tampering | Docker internal networking; don't expose 8080 in production |
| SQLite database corruption | Denial of Service | WAL mode, single writer (`MaxOpenConns(1)`), backups via volume mount |
| Container escape | Elevation of Privilege | Run as non-root user in Dockerfile; read-only config mount |

## Sources

### Primary (HIGH confidence)
- ogamed REST API documentation — https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation (all endpoints verified)
- ogamed Dockerfile — https://github.com/alaingilbert/ogame/blob/master/Dockerfile (env vars verified)
- Context7 `/gitlab_cznic/sqlite` — modernc.org/sqlite usage patterns, database/sql driver setup, pragma support
- Context7 `/golang-migrate/migrate` — embedded migration patterns, iofs source, WithInstance API
- Existing TypeScript implementation — `packages/bot/src/` (client, config, rate-limiter, retry patterns)
- Go stdlib documentation — `log/slog`, `net/http`, `testing`, `embed`, `database/sql`

### Secondary (MEDIUM confidence)
- Go toolchain verification on host — `go version` → 1.26.2
- Docker/Docker Compose availability on host — Docker 29.4.1, Compose 5.1.3
- Package version verification — `go list -m` for sqlite, yaml, migrate

### Tertiary (LOW confidence)
- ogamed Docker Hub image availability — 404 response, no pre-built image found
- golang-migrate sqlite3 driver compatibility with modernc.org/sqlite — not tested, flagged in assumptions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified via `go list -m`, Context7 docs confirm patterns
- Architecture: HIGH — patterns ported from working TypeScript implementation, Go idioms well-established
- Pitfalls: HIGH — cross-referenced with existing PITFALLS.md and ogamed issue history
- Docker: MEDIUM — ogamed Docker image sourcing needs resolution (open question)

**Research date:** 2026-04-26
**Valid until:** 2026-05-26 (stable — Go ecosystem and libraries change slowly)
