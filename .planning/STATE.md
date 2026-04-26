# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-25)

**Core value:** The bot must reliably protect your fleet and grow your empire while you're away — if fleet-save fails, everything else is pointless.
**Current focus:** Phase 4 — Auto-Farm (in progress)

## Current Position

Phase: 5 of 5 (Web Dashboard)
Plan: 0 of ? plans
Status: Phase 4 complete, advancing to Phase 5 (final)
Last activity: 2026-04-26 — Phase 4 verified (auto-farm, 9/9 passed)

Progress: [████████░░] 80%

## Performance Metrics

**Velocity:**
- Total plans completed: 12
- Average duration: 8.7 min
- Total execution time: 104 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Core Infrastructure | 5 | 33 min | 6.6 min |
| 2. Fleet Safety | 3 | 36 min | 12.0 min |
| 3. Auto-Build | 2 | 21 min | 10.5 min |
| 4. Auto-Farm | 2 | 14 min | 7.0 min |

**Recent Trend:**
- Last 5 plans: 03-01 (11 min), 03-02 (10 min), 04-01 (7 min), 04-02 (7 min)
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
Stopped at: Completed 04-02 (Farmer worker). Phase 4 Auto-Farm complete.
Resume file: .planning/phases/04-auto-farm/04-02-SUMMARY.md
