# Phase 1: Core Infrastructure - Context

**Gathered:** 2026-04-26
**Updated:** 2026-04-26 (Go pivot)
**Status:** Ready for replanning

<domain>
## Phase Boundary

Connect to OGame via ogamed REST API, maintain game state, config, throttling, and Docker setup. This is the foundation every other phase builds on — no bot features work without this layer.

**PIVOT:** Bot engine changed from TypeScript to Go. Developer knows Go best. ogamed is already Go. Dashboard stays TypeScript/SolidJS (Phase 5).

</domain>

<decisions>
## Implementation Decisions

### Project Structure
- **D-01:** Go module for bot engine (`cmd/bot/` entrypoint, `internal/` packages). Separate pnpm workspace for dashboard (`packages/dashboard`, `packages/shared`).
- **D-02:** Go for bot engine with standard Go project layout. TypeScript for dashboard only (Phase 5).
- **D-03:** Shared types exist in Go packages (`internal/ogamed/types/`). Dashboard types will be generated from bot's REST API (OpenAPI/codegen in Phase 5).

### Configuration
- **D-04:** YAML config file (`config.yaml`) for user-facing bot settings. Use `gopkg.in/yaml.v3`.
- **D-05:** Config validated with Go struct tags + manual validation at startup. Invalid config = clear error + exit.
- **D-06:** Config structure: account credentials, ogamed connection settings, per-feature toggles and parameters, logging level.
- **D-07:** Secrets loaded from environment variables, referenced in config via `${ENV_VAR}` interpolation.

### State Storage
- **D-08:** SQLite via `modernc.org/sqlite` (pure Go, no CGo required) for all persistent state. Single-file database.
- **D-09:** Game state cached in SQLite tables (planets, resources, buildings, fleets, research). Updated on each poll cycle.
- **D-10:** Schema migrations via `golang-migrate/migrate` or embedded SQL migration files.

### ogamed Client
- **D-11:** Type-safe ogamed REST client in `internal/ogamed/`. Go structs for all request/response types.
- **D-12:** Validate ogamed responses match expected structure. Handle unknown/missing fields gracefully (ogamed game-update breakage is a known pitfall).
- **D-13:** Automatic retries with exponential backoff for transient failures (network errors, 5xx responses).
- **D-14:** Rate limiter wraps all ogamed calls. Minimum 1-3 second random delay between requests. Configurable per-endpoint.

### Docker
- **D-15:** Docker Compose with two services: `ogamed` (official ogamed image) and `bot` (Go binary). Shared Docker network, bot calls ogamed via `http://ogamed:8080`.
- **D-16:** Environment-based configuration. `.env` file for secrets, `config.yaml` mounted as volume.
- **D-17:** `data/` directory mounted as persistent volume for SQLite database.

### Logging
- **D-18:** Go `log/slog` structured logging (stdlib, no external dependency). JSON in production, text in dev.
- **D-19:** All ogamed API calls logged at debug level with request/response timing.

### Agent's Discretion
- Exact package structure within `internal/`
- Test framework selection (stdlib `testing` recommended)
- Specific SQLite schema design (table structure, indexes)
- Build setup for production Docker image (multi-stage Go build)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Project vision, core value, constraints, key decisions
- `.planning/REQUIREMENTS.md` — INFRA-01 through INFRA-05 requirements and acceptance criteria
- `.planning/ROADMAP.md` — Phase 1 detail section with success criteria

### Research
- `.planning/research/STACK.md` — Technology stack recommendations with rationale
- `.planning/research/ARCHITECTURE.md` — Component boundaries, data flow, project structure
- `.planning/research/PITFALLS.md` — Known pitfalls including ogamed breakage, rate limiting, captcha

### External References
- `https://github.com/alaingilbert/ogame` — ogamed REST API documentation and endpoints list
- `https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation` — Full ogamed REST API docs

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `packages/shared/` — TypeScript types and schemas already built (Plans 01-01, 01-02). Useful as reference for Go struct definitions but dashboard integration deferred to Phase 5.
- Root `package.json`, `pnpm-workspace.yaml`, `tsconfig.base.json` — still needed for dashboard workspace.

### Established Patterns
- None for Go — this phase establishes patterns for all subsequent phases

### Integration Points
- ogamed REST API at `http://ogamed:8080` — all game interaction flows through this
- SQLite database — all state reads/writes
- config.yaml — all feature configuration

</code_context>

<specifics>
## Specific Ideas

- Follow TBot's pattern of organizing bot logic as independent workers that run on their own tick intervals
- Follow Cruiser's philosophy of being stateless-safe: "can be restarted at any time without causing problems"
- Rate limiter should be a shared infrastructure component that all workers use, preventing multi-worker request spikes
- Use Go interfaces for ogamed client so workers can be tested with mocks
- Keep `internal/` flat initially — split into sub-packages as needed

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-core-infrastructure*
*Context gathered: 2026-04-26*
*Updated for Go pivot: 2026-04-26*
