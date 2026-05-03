---
phase: 02-fleet-safety
plan: 03
subsystem: defender-engine
tags: [go, tdd, fleet-save, defender, poll-loop, attack-detection, recall, anti-detection]

# Dependency graph
requires:
  - phase: 02-01
    provides: "ClientInterface with SendFleet/CancelFleet/GetAttacks/GetSlots, DefenderConfig, mission constants"
  - phase: 02-02
    provides: "CalcEscapeRoutes, EscapeRoute struct, flight/fuel formulas, ship stats DB"
provides:
  - Defender worker with poll loop, attack handling, fleet-save orchestration, recall scheduling
  - StateReader interface for defender DI (state.Manager satisfies implicitly)
  - Fleet-save tracking table and CRUD methods (activeFleetSave, recordFleetSave, completeFleetSave, markRecalled, pendingRecalls)
  - identifyEndangered: attack grouping by destination, espionage filtering, timing safety check
  - savePlanet: reaction delay → slot check → escape routes → SendFleet → record event
  - processRecalls: cancel deployed fleets after danger passes
  - calcReactionDelay: randomized anti-detection delay capped by safety margin
  - Defender wired into cmd/bot/main.go when enabled
affects: [03-auto-build, 04-auto-farm]

# Tech tracking
tech-stack:
  added: []
  patterns: [state-reader-interface-for-DI, fleet-save-tracking-table, endangered-planet-grouping]

key-files:
  created:
    - internal/defender/defender.go
    - internal/defender/defender_test.go
    - internal/state/migrations/002_fleet_save.sql
  modified:
    - cmd/bot/main.go

key-decisions:
  - "Active fleet-save check added to savePlanet (not just handleAttacks) for defense-in-depth against duplicate saves"
  - "Test config uses fastDefenderConfig() with SafetyMarginMs=2000 and short reaction delays to keep tests under 30s total"
  - "Reaction delay randomized in [minDelay, maxAllowedDelay) where maxAllowedDelay = timeUntilAttack - safetyMargin"

requirements-completed: [SAFE-01, SAFE-02, SAFE-03]

# Metrics
duration: 15min
completed: 2026-04-26
---

# Phase 2 Plan 03: Defender Worker Summary

**Defender worker with randomized poll loop, attack detection with espionage filtering, fleet-save orchestration via escape routes, SQLite tracking table, and recall scheduling wired into bot startup**

## Performance

- **Duration:** 15m
- **Started:** 2026-04-26T06:55:14Z
- **Completed:** 2026-04-26T07:09:49Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Defender struct with StateReader interface for clean dependency injection (state.Manager satisfies implicitly)
- Fleet-save tracking via SQLite: record, query active, complete, mark recalled, query pending recalls
- Poll loop with randomized interval + jitter for anti-detection (T-02-10 mitigation)
- identifyEndangered groups attacks by destination, filters espionage (mission 6), checks timing safety margin
- savePlanet: randomized reaction delay → slot check → fresh state from ogamed → CalcEscapeRoutes → SendFleet → record event
- processRecalls: cancel deployed fleets after danger passes, handle already-returning and missing fleets gracefully
- All failure modes handled: no ships, no routes (critical alert logged), no slots, already saved, too close to react
- cmd/bot/main.go starts defender goroutine when `features.defender.enabled: true`

## Task Commits

Each task was committed atomically (TDD: test → feat):

1. **Task 1 RED: Fleet-save tracking table and Defender struct tests** - `a00888b` (test)
2. **Task 1+2 GREEN: Implement Defender worker with all methods** - `bd01281` (feat)

## Files Created/Modified
- `internal/defender/defender.go` - Defender worker: struct, tracking methods, Run loop, identifyEndangered, savePlanet, processRecalls, calcReactionDelay, helpers (554 lines)
- `internal/defender/defender_test.go` - 20 new tests covering tracking, attack handling, save, recall, delay (928 lines)
- `internal/state/migrations/002_fleet_save.sql` - fleet_save_events table with active index (13 lines)
- `cmd/bot/main.go` - Wires defender goroutine when enabled, added `time` and `defender` imports

## Decisions Made
- Active fleet-save check placed inside savePlanet (not just handleAttacks) for defense-in-depth — prevents duplicate saves even if savePlanet is called directly
- Test config uses `fastDefenderConfig()` with SafetyMarginMs=2000, MinReactionDelayS=1, MaxReactionDelayS=2 to keep tests fast while testing real timing logic
- Reaction delay formula: `minDelay + rand(0, timeUntilAttack - safetyMargin - minDelay)` — ensures fleet-save always completes before attack arrival while adding anti-detection randomness

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed test reaction delay causing timeouts**
- **Found during:** Task 2 GREEN phase
- **Issue:** Tests using defaultDefenderConfig() with MinReactionDelayS=30 and SafetyMarginMs=120000 caused savePlanet tests to sleep 30+ seconds and timeout
- **Fix:** Created fastDefenderConfig() with SafetyMarginMs=2000, MinReactionDelayS=1, MaxReactionDelayS=2; used timeUntilAttack=10s for all savePlanet tests
- **Files modified:** internal/defender/defender_test.go
- **Verification:** All 20 defender tests pass in under 20 seconds total
- **Committed in:** bd01281 (Task 2 GREEN commit)

**2. [Rule 2 - Missing Critical] Added active fleet-save check inside savePlanet**
- **Found during:** Task 2 GREEN phase
- **Issue:** Plan only checked active fleet-save in handleAttacks, but TestSavePlanetDoesNotResave called savePlanet directly — duplicate save was not prevented
- **Fix:** Added activeFleetSave check at the start of savePlanet for defense-in-depth
- **Files modified:** internal/defender/defender.go
- **Verification:** TestSavePlanetDoesNotResave passes, no duplicate SendFleet calls
- **Committed in:** bd01281 (Task 2 GREEN commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both fixes improve correctness — tests run fast, duplicate saves prevented at the execution layer

## TDD Gate Compliance

| Gate | Commit | Description |
|------|--------|-------------|
| RED (Task 1) | `a00888b` | Tests for fleet-save tracking, attack handling, savePlanet, processRecalls, reaction delay — fail to compile (NewDefender undefined) |
| GREEN (Tasks 1+2) | `bd01281` | Full Defender implementation — all 20 new tests pass + 26 existing escape tests |

All TDD gates satisfied: 1 RED commit, 1 GREEN commit. No REFACTOR commits needed.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Defender worker complete — fleet-safety loop fully operational when enabled in config
- Ready for Phase 3 (auto-build) — can run alongside state manager without conflicts
- Threat model mitigations implemented: T-02-06 (malformed attacks filtered), T-02-07 (duplicate saves prevented), T-02-08 (slot check), T-02-10 (randomized delays)

## Self-Check: PASSED

- All modified files verified in git log (2 commits)
- All commit hashes verified: a00888b, bd01281
- All tests pass: `go test ./... -count=1 -timeout=60s` → ok
- Test file line count: 928 (minimum 100)
- Build verification: `go build ./cmd/bot/` → success

---
*Phase: 02-fleet-safety*
*Completed: 2026-04-26*
