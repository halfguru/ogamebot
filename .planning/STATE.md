# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-25)

**Core value:** The bot must reliably protect your fleet and grow your empire while you're away — if fleet-save fails, everything else is pointless.
**Current focus:** Phase 5 — Web Dashboard (final)

## Current Position

Phase: 5 of 5 (Web Dashboard)
Plan: 1 of ? plans
Status: Plan 05-01 complete (Dashboard API layer)
Last activity: 2026-04-26 — Plan 05-01 complete

Progress: [█████████░] 85%

## Performance Metrics

**Velocity:**
- Total plans completed: 13
- Average duration: 8.5 min
- Total execution time: 110 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Core Infrastructure | 5 | 33 min | 6.6 min |
| 2. Fleet Safety | 3 | 36 min | 12.0 min |
| 3. Auto-Build | 2 | 21 min | 10.5 min |
| 4. Auto-Farm | 2 | 14 min | 7.0 min |
| 5. Web Dashboard | 1 | 6 min | 6.0 min |

**Recent Trend:**
- Last 5 plans: 03-02 (10 min), 04-01 (7 min), 04-02 (7 min), 05-01 (6 min)
- Trend: Stable

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: 5 phases derived from 17 v1 requirements; safety-first ordering (fleet-save before growth features)
- Phase 1 context: pnpm monorepo, YAML config, SQLite + modernc.org/sqlite, Docker Compose
- 01-01 (Go): Missing env vars return immediate error with variable name; all 11 domain structs in single model package; constants use untyped int
- 01-02 (Go): rateLimiterInterface for testability, HTTP errors mapped to OgamedError for retry, two-pass generic unmarshal for getTyped[T]
- 01-03 (Go): Replaced golang-migrate with custom migration runner to avoid CGo dep and m.Close() bug; fleet full-replace per cycle; Dockerfile.ogamed for source builds
- 02-01 (Go): RecallEnabled uses *bool pointer for YAML default handling; ships encoded as repeated params; MissionHold=5 replaces incorrect MissionACSTransport
- 02-02 (Go): Planet↔moon at distance=0 is valid escape route (10s min flight, 0 fuel); safety scoring uses weighted sum (+1000 attacked, +500 planet, -100 moon, +distance/50, +fuel/10k)
- 02-03 (Go): Active fleet-save check in savePlanet for defense-in-depth; reaction delay = minDelay + rand(0, timeUntilAttack - safetyMargin - minDelay); test fastDefenderConfig() pattern for timing-dependent tests
- 03-01 (Go): ROI uses metal-equivalent scoring (metal=1, crystal=1.5, deuterium=2.0); energy-producing buildings valued at 0.5 per unit; AutoBuildConfig defaults {MetalMine:30, CrystalMine:28, DeutSynth:26, SolarPlant:26, FusionReactor:20}; server speed cached in state manager
- 03-02 (Go): Builder poll loop evaluates ROI across all planets each tick; anti-detection via configurable antiDetectPct (7% default, 0 in tests); per-planet max-level overrides take precedence over global defaults; builder skips planet on GetConstructions error (conservative)
- 04-01 (Go): GalaxyRange type alias in config to avoid circular import; AutoFarmConfig embeds FeatureConfig inline; AutoFarm poll minimum 60s (galaxy scans are expensive); mock clients updated in all 3 test packages when ClientInterface expanded
- 04-02 (Go): Inlined estimateFuelCost in farmer instead of exporting from defender; farmer reserves 2 fleet slots for defender; max 10 probes per cycle to avoid API spam; simplified fuel formula (baseFuel only, no drive tech variation)
- 05-01 (Go): StateReader interface decouples dashboard from state.Manager; gorilla/chat hub pattern with max 10 WS clients; Go 1.22+ method routing for REST endpoints; CORS middleware supports wildcard or whitelist

### Pending Todos

None yet.

### Blockers/Concerns

- OGame formula constants (building costs, production rates) — RESOLVED in Phase 3 research, implemented in 03-01
- Residential proxy required for deployment — operational concern, not blocking development

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-04-26
Stopped at: Completed 05-01 (Dashboard API layer). Phase 5 in progress.
Resume file: .planning/phases/05-web-dashboard/05-01-SUMMARY.md
