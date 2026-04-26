---
phase: 01-core-infrastructure
plan: 02
subsystem: infra
tags: [yaml, zod, pino, rate-limiter, retry, ogamed-client, ofetch]

# Dependency graph
requires: [01-01]
provides:
  - YAML config loading with env interpolation and Zod validation
  - pino structured logger factory with dev/prod modes
  - Centralized rate limiter with random jitter and per-endpoint overrides
  - Exponential backoff retry helper
  - Typed ogamed REST client wrapping 14 endpoints with validation
affects: [03-game-state]

# Tech tracking
tech-stack:
  added: [js-yaml@4.1.1, dotenv@17.4.2, pino@10.3.1, pino-pretty@13.1.3, ofetch@1.5.1]
  patterns: [YAML + env interpolation for config, Zod 4 factory defaults for nested objects, rate limiter with jitter, exponential backoff retry, typed REST client with generic get<T>()]

key-files:
  created:
    - packages/bot/src/config/config-schema.ts
    - packages/bot/src/config/config-loader.ts
    - packages/bot/src/utils/logger.ts
    - packages/bot/src/client/rate-limiter.ts
    - packages/bot/src/client/retry.ts
    - packages/bot/src/client/ogamed-client.ts
    - config.example.yaml
  modified:
    - packages/bot/package.json

key-decisions:
  - "Zod 4 requires factory functions for .default() on objects — .default(() => ({...})) instead of .default({})"
  - "Zod 4 z.record() requires explicit key schema — z.record(z.string(), valueSchema) instead of z.record(valueSchema)"
  - "zod added as direct bot dependency (needed for config-schema.ts) in addition to transitive via @ogame-bot/shared"
  - "Retry helper does NOT retry ZodError or 4xx — those indicate real problems, not transient failures"
  - "Rate limiter uses shared lastRequestTime across all endpoints — single chokepoint prevents multi-worker spikes"

patterns-established:
  - "Config: YAML with ${ENV_VAR} interpolation → dotenv loads .env → Zod validates → exit(1) on invalid"
  - "API client: rateLimiter.acquire() → retryWithBackoff(fn) → ofetch → ogamedResponseSchema.parse() → return typed Result"
  - "Logger: createLogger(name, level) with pino-pretty auto-enabled for debug/trace levels"

requirements-completed: [INFRA-01, INFRA-03, INFRA-04]

# Metrics
duration: 3min
completed: 2026-04-26
---

# Phase 1 Plan 02: Config, Logger, and Ogamed Client Summary

**YAML config with Zod validation + env interpolation, pino structured logger, typed ogamed REST client with rate limiting, retry, and Zod response validation across 14 endpoints**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-26T04:50:04Z
- **Completed:** 2026-04-26T04:53:48Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Config system loads YAML, interpolates ${ENV_VAR} references, validates with Zod schema, exits with clear errors on invalid config
- pino logger factory with JSON in production, pretty-print in debug/trace mode
- Rate limiter enforces configurable min/max random delay with per-endpoint overrides
- Retry helper with exponential backoff + jitter, skips retries on ZodError and 4xx client errors
- OgamedClient wraps 14 typed endpoints (login, logout, server time, under attack, planets, resources, buildings, facilities, ships, defence, fleets, research, server speed, server version)
- Every API call validated with ogamedResponseSchema before use

## Task Commits

Each task was committed atomically:

1. **Task 1: Config loader with YAML + env interpolation + Zod validation, and pino logger** - `a04ef9c` (feat)
2. **Task 2: Ogamed client with rate limiter, retry, and Zod response validation** - `13e2dc3` (feat)

## Files Created/Modified
- `packages/bot/src/config/config-schema.ts` - Zod schema for bot config (account, ogamed, features, rateLimit, logLevel)
- `packages/bot/src/config/config-loader.ts` - YAML loading with env interpolation, dotenv, Zod validation
- `packages/bot/src/utils/logger.ts` - pino logger factory with dev/prod modes
- `packages/bot/src/client/rate-limiter.ts` - Centralized throttle with random jitter and per-endpoint overrides
- `packages/bot/src/client/retry.ts` - Exponential backoff with jitter, no retry on ZodError/4xx
- `packages/bot/src/client/ogamed-client.ts` - Typed REST client wrapping 14 ogamed endpoints
- `config.example.yaml` - Documented example config (31 lines)
- `packages/bot/package.json` - Added zod and @types/js-yaml dependencies

## Decisions Made
- Zod 4 requires factory functions `.default(() => ({...}))` for object defaults — not `.default({})`
- Zod 4 `z.record()` needs explicit key schema: `z.record(z.string(), valueSchema)`
- zod added as direct bot dependency for config schema (in addition to transitive via shared)
- Retry helper skips ZodError and 4xx errors — those indicate real problems, not transient failures
- Rate limiter uses shared `lastRequestTime` — single chokepoint prevents multi-worker request spikes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Zod 4 API differences from plan code**
- **Found during:** Task 1
- **Issue:** Plan's config-schema.ts used `.default({})` and `z.record(z.object(...))` which don't compile in Zod 4
- **Fix:** Changed to factory functions `.default(() => ({...}))` and explicit key schema `z.record(z.string(), ...)`
- **Files modified:** packages/bot/src/config/config-schema.ts
- **Commit:** a04ef9c

**2. [Rule 3 - Blocking] Missing zod and @types/js-yaml dependencies**
- **Found during:** Task 1
- **Issue:** bot package didn't have zod as direct dependency (needed for config-schema.ts) and js-yaml lacked type declarations
- **Fix:** Added zod and @types/js-yaml as bot dependencies
- **Files modified:** packages/bot/package.json, pnpm-lock.yaml
- **Commit:** a04ef9c

## Issues Encountered
- None beyond the Zod 4 API adjustments documented above

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Config system ready for main.ts to call loadConfig('config.yaml')
- Logger ready for createLogger('bot', config.logLevel)
- OgamedClient ready for instantiation with config.ogamed.url and RateLimiter
- All components ready for Plan 03 (game state manager) to consume

---
*Phase: 01-core-infrastructure*
*Completed: 2026-04-26*
