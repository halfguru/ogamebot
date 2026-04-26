# Phase 1 — Validation Gate

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.5 |
| Config | `vitest.config.ts` (root) |
| Quick run | `pnpm -r exec vitest run` |
| Full suite | `pnpm -r exec vitest run --coverage` |

## Requirements → Test Map

| Req ID | Behavior | Test Type | Command | Created In |
|--------|----------|-----------|---------|------------|
| INFRA-01 | Ogamed client connects and authenticates | unit (mocked) + integration | `vitest run packages/bot/src/client/` | Wave 2 (Plan 01-02) |
| INFRA-01 | Client retries on connection failure | unit | `vitest run packages/bot/src/client/` | Wave 2 (Plan 01-02) |
| INFRA-02 | Game state refreshes from ogamed and persists to SQLite | unit (mocked) + integration | `vitest run packages/bot/src/state/` | Wave 3 (Plan 01-03) |
| INFRA-02 | Zod schemas validate real ogamed response shapes | unit | `vitest run packages/shared/src/schemas/` | Wave 1 (Plan 01-01) |
| INFRA-03 | Config loads from YAML with env interpolation | unit | `vitest run packages/bot/src/config/` | Wave 2 (Plan 01-02) |
| INFRA-03 | Invalid config produces clear error and exits | unit | `vitest run packages/bot/src/config/` | Wave 2 (Plan 01-02) |
| INFRA-04 | Rate limiter enforces minimum delays between calls | unit | `vitest run packages/bot/src/client/` | Wave 2 (Plan 01-02) |
| INFRA-04 | Rate limiter adds random jitter | unit | `vitest run packages/bot/src/client/` | Wave 2 (Plan 01-02) |
| INFRA-05 | Docker Compose starts both containers | integration (manual) | `docker compose up --abort-on-container-exit` | Wave 3 (Plan 01-03) |
| INFRA-05 | Bot reaches ogamed via internal network | integration (manual) | `docker compose exec bot wget -qO- http://ogamed:8080/bot/server/time` | Wave 3 (Plan 01-03) |

## Sampling Rate

- **Per task commit:** `pnpm --filter bot exec vitest run`
- **Per wave merge:** `pnpm -r exec vitest run`
- **Phase gate:** Full suite green + `docker compose up` smoke test passes

## Pre-Flight Checklist (Wave 0 gaps)

These must be created during Plan 01-01 (Wave 1):

- [ ] `vitest.config.ts` — root config for test framework
- [ ] `packages/bot/tests/` — test directory with shared fixtures
- [ ] `packages/shared/tests/` — test directory for schema validation
- [ ] Framework install: `pnpm add -Dw vitest` — included in scaffolding
- [ ] Docker integration test script: `tests/docker-smoke.sh`

## Phase Gate Criteria

Phase 1 is complete when ALL of the following pass:

1. `pnpm ls --recursive` — all 3 workspace packages resolve
2. `pnpm -r exec tsc --noEmit` — zero type errors
3. `pnpm -r exec vitest run` — all tests green
4. `docker compose up` — both containers start, bot reaches ogamed at `http://ogamed:8080`
5. Bot logs "Connected to ogamed" with pino structured output
6. `config.yaml` loads with `${ENV_VAR}` interpolation working
