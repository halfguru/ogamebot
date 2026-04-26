# Phase 1: Core Infrastructure - Context

**Gathered:** 2026-04-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Connect to OGame via ogamed REST API, maintain game state, config, throttling, and Docker setup. This is the foundation every other phase builds on — no bot features work without this layer.

</domain>

<decisions>
## Implementation Decisions

### Project Structure
- **D-01:** pnpm monorepo with three packages: `packages/bot` (bot engine + Fastify server), `packages/dashboard` (SolidJS web app), `packages/shared` (shared types, constants, Zod schemas)
- **D-02:** TypeScript throughout with strict mode. ESM modules.
- **D-03:** Shared package exports: OGame type definitions (planets, resources, fleets, buildings, etc.), Zod schemas for ogamed response validation, constants (building IDs, mission types, etc.)

### Configuration
- **D-04:** YAML config file (`config.yaml`) for user-facing bot settings. YAML is more human-readable for this use case (multi-line values, comments supported). Cruiser uses this pattern.
- **D-05:** Config schema validated with Zod at startup. Invalid config = clear error message + exit.
- **D-06:** Config structure: account credentials, ogamed connection settings, per-feature toggles and parameters, logging level.
- **D-07:** Secrets (password) loaded from environment variables, not stored in config file. Config references them via `${ENV_VAR}` interpolation.

### State Storage
- **D-08:** SQLite via Drizzle ORM for all persistent state. Single-file database in data directory. Zero ops, no separate DB server.
- **D-09:** Game state cached in SQLite tables (planets, resources, buildings, fleets, research). Updated on each poll cycle.
- **D-10:** Drizzle migrations for schema evolution. Schema defined in code, migrations auto-generated.

### ogamed Client
- **D-11:** Type-safe ogamed REST client wrapper in `packages/bot`. All endpoints covered with typed request/response.
- **D-12:** Zod schemas validate every ogamed response before use. Invalid responses logged and handled gracefully (ogamed game-update breakage is a known pitfall).
- **D-13:** Automatic retries with exponential backoff for transient failures (network errors, 5xx responses).
- **D-14:** Rate limiter wraps all ogamed calls. Minimum 1-3 second random delay between requests. Configurable per-endpoint (galaxy scanning needs longer delays than planet info).

### Docker
- **D-15:** Docker Compose with two services: `ogamed` (official ogamed image) and `bot` (Node.js). Shared Docker network, bot calls ogamed via `http://ogamed:8080`.
- **D-16:** Environment-based configuration. `.env` file for secrets, `config.yaml` mounted as volume.
- **D-17:** `data/` directory mounted as persistent volume for SQLite database.

### Logging
- **D-18:** pino structured logging (JSON in production, pretty-print in dev). Log levels: trace, debug, info, warn, error, fatal.
- **D-19:** All ogamed API calls logged at debug level with request/response timing.

### Agent's Discretion
- Exact file naming conventions within packages
- Test framework selection (vitest recommended for monorepo compatibility)
- Specific Drizzle schema design (table structure, indexes)
- Build/bundling setup for production Docker image

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
- None — greenfield project

### Established Patterns
- None — this phase establishes patterns for all subsequent phases

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

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-core-infrastructure*
*Context gathered: 2026-04-26*
