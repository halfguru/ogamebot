---
phase: 01-core-infrastructure
plan: 03
subsystem: infra
tags: [go, sqlite, state-manager, docker, docker-compose, migrations, modernc-sqlite]

# Dependency graph
requires:
  - phase: 01-01
    provides: "Domain types (model package), config structs, YAML loader"
  - phase: 01-02
    provides: "ogamed ClientInterface (14 methods), RateLimiter, retry logic"
provides:
  - SQLite database with embedded migrations (6 game state tables + schema_migrations tracking)
  - OpenDB with WAL mode, foreign keys, MaxOpenConns(1) for single-writer safety
  - Game state manager that polls ogamed periodically and upserts data into SQLite
  - Read methods for cached state (GetPlanets, GetResources, GetFleets, GetResearch)
  - Bot main entrypoint wiring config → DB → client → login → state manager → graceful shutdown
  - Multi-stage Dockerfile with CGO_ENABLED=0
  - Docker Compose stack: ogamed + bot with shared network
  - Dockerfile.ogamed fallback for building ogamed from source
affects: [02-fleet-save, 03-auto-build, 04-auto-farm, 05-dashboard]

# Tech tracking
tech-stack:
  added: [modernc.org/sqlite, docker, docker-compose]
  patterns: [embedded-migrations, manual-migration-runner, insert-or-replace-upsert, fleet-full-replace, graceful-shutdown-signals]

key-files:
  created:
    - internal/state/migrations/001_init.sql
    - internal/state/db.go
    - internal/state/db_test.go
    - internal/state/manager.go
    - internal/state/manager_test.go
    - cmd/bot/main.go
    - Dockerfile
    - docker-compose.yml
    - docker/Dockerfile.ogamed
    - .dockerignore
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Replaced golang-migrate with custom migration runner to avoid mattn/go-sqlite3 CGo dep and m.Close() closing *sql.DB"
  - "Manual migration runner uses schema_migrations table for idempotency tracking"
  - "Fleets use full replace (DELETE + INSERT) per cycle since fleet data is ephemeral"
  - "ogamed Docker image built from source via Dockerfile.ogamed instead of unreliable remote context"
  - "ogamed port 8080 bound to 127.0.0.1 only for security"

patterns-established:
  - "Embedded migrations: embed.FS with .sql files, version tracking via schema_migrations table"
  - "INSERT OR REPLACE upsert: idempotent state caching for planets/resources/buildings/facilities/research"
  - "State manager polling pattern: initial refresh + ticker loop with context cancellation"
  - "Mock client pattern: function fields on struct implementing ClientInterface for per-test customization"
  - "Docker Compose security: internal networking, read-only config mount, 127.0.0.1 port binding"

requirements-completed: [INFRA-02, INFRA-05]

# Metrics
duration: 11min
completed: 2026-04-26
---

# Phase 1 Plan 03: State Manager, Entrypoint, and Docker Summary

**SQLite database with embedded migrations, game state manager polling ogamed with per-planet error resilience, bot entrypoint with graceful shutdown, and Docker Compose stack for ogamed + bot**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-26T05:52:44Z
- **Completed:** 2026-04-26T06:03:52Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments
- SQLite database with 6 game state tables, WAL mode, foreign keys, and custom embedded migration runner
- Game state manager polls ogamed, caches all state in SQLite with INSERT OR REPLACE upserts and per-planet error resilience
- Bot entrypoint wires all components with graceful SIGINT/SIGTERM shutdown
- Docker Compose stack: ogamed (built from source) + bot with shared networking and persistent SQLite volume

## Task Commits

Each task was committed atomically:

1. **Task 1: SQLite database with embedded migrations** - `2c2249c` (feat)
2. **Task 2: Game state manager with ogamed polling and SQLite caching** - `bc06488` (feat)
3. **Task 3: Main entrypoint, Dockerfile, and docker-compose.yml** - `0336acd` (feat)

## Files Created/Modified
- `internal/state/migrations/001_init.sql` - Initial schema: planets, resources, buildings, facilities, research, fleets (6 tables)
- `internal/state/db.go` - OpenDB with WAL mode, foreign keys, MaxOpenConns(1), custom migration runner with schema_migrations tracking
- `internal/state/db_test.go` - 6 tests: table creation, idempotency, WAL mode, CRUD, MaxOpenConns, foreign keys
- `internal/state/manager.go` - Game state manager: Run loop, refresh, upsert helpers, read methods (GetPlanets, GetResources, GetFleets, GetResearch)
- `internal/state/manager_test.go` - 7 tests with mock client: full refresh, GetPlanets, error resilience, periodic refresh, context cancel, GetResearch, GetFleets
- `cmd/bot/main.go` - Bot entrypoint: config → logger → DB → rate limiter → client → login → state manager → signal handler
- `Dockerfile` - Multi-stage Go build with CGO_ENABLED=0 (pure Go, no CGo)
- `docker-compose.yml` - Two-service stack: ogamed + bot, 127.0.0.1:8080 binding, bot-data volume
- `docker/Dockerfile.ogamed` - Builds ogamed from source via `go install`
- `.dockerignore` - Excludes node_modules, .planning, data from Docker build context
- `go.mod` - Added modernc.org/sqlite dependency
- `go.sum` - Updated checksums

## Decisions Made
- Replaced golang-migrate with custom migration runner: golang-migrate's `m.Close()` closes the underlying `*sql.DB`, making it unusable after migration. Also avoids pulling in mattn/go-sqlite3 as a transitive CGo dependency
- Manual migration runner with `schema_migrations` table for version tracking — simpler, no external deps, idempotent
- Fleet data uses full replace (DELETE all + INSERT) per cycle since fleets are ephemeral
- ogamed built from source via `Dockerfile.ogamed` instead of remote Docker context (more reliable per research A2)
- ogamed port 8080 bound to 127.0.0.1 only for security (T-03-01)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Replaced golang-migrate with custom migration runner**
- **Found during:** Task 1 (Database setup)
- **Issue:** golang-migrate's `m.Close()` closes the underlying `*sql.DB`, making the database connection unusable after migrations. Also, golang-migrate's sqlite3 driver transitively depends on mattn/go-sqlite3 (CGo), conflicting with our pure-Go modernc.org/sqlite driver.
- **Fix:** Replaced with a simple custom migration runner that reads embedded `.sql` files, executes them via `db.Exec()`, and tracks applied migrations in a `schema_migrations` table. Renamed migration file from `001_init.up.sql` to `001_init.sql`.
- **Files modified:** internal/state/db.go
- **Commit:** 2c2249c

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Positive — eliminated CGo dependency, simplified migration system, avoided runtime bug

## Issues Encountered
None

## User Setup Required
None - no external service configuration required beyond what's already in .env.example and config.example.yaml.

## Next Phase Readiness
- All 5 INFRA requirements satisfied (INFRA-01 through INFRA-05)
- Bot ready for Docker deployment via `docker compose up`
- State manager ready for downstream consumers (fleet-save, auto-build, auto-farm workers)
- Read methods (GetPlanets, GetResources, GetFleets, GetResearch) available for feature workers

## Self-Check: PASSED

- All 10 created files verified present
- All 3 commit hashes verified in git log
- All tests pass: `go test ./... -count=1` → ok

---
*Phase: 01-core-infrastructure*
*Completed: 2026-04-26*
