---
phase: 03-auto-build
plan: 02
subsystem: builder
tags: [go, tdd, builder-worker, poll-loop, roi-evaluation, anti-detection, build-events]

# Dependency graph
requires:
  - phase: 03-01
    provides: "ROI calculator (CalculateROI), ClientInterface (GetConstructions, BuildBuilding), state read methods (GetBuildings, GetFacilities, GetServerSpeed), AutoBuildConfig with max-level caps"
provides:
  - Builder worker with poll loop that evaluates ROI across all planets and executes highest-ROI build
  - BuilderStateReader interface for DI (GetPlanets, GetResources, GetResearch, GetBuildings, GetFacilities, GetServerSpeed)
  - resolveMaxLevel with per-planet override precedence over global defaults
  - buildingLevel helper mapping building IDs to ResourceBuildings struct fields
  - recordBuildEvent for audit trail in build_events table
  - Anti-detection: configurable probability of picking 2nd-best ROI candidate
affects: [03-auto-build-complete, 04-auto-farm, 05-dashboard]

# Tech tracking
tech-stack:
  added: []
  patterns: [poll-loop-worker, roi-based-decision, anti-detection-random, per-planet-config-override]

key-files:
  created:
    - internal/builder/builder.go
    - internal/builder/builder_test.go
  modified:
    - cmd/bot/main.go

key-decisions:
  - "Anti-detection probability configurable via antiDetectPct field (default 7%, set to 0 in tests for determinism)"
  - "Builder skips planets on GetConstructions API error (conservative — avoids building when state unknown)"
  - "Builder records build event AFTER successful BuildBuilding call (no event recorded on API failure)"
  - "Per-planet max-level overrides take precedence over global defaults; missing config returns 0 (effectively disabled)"

requirements-completed: [GROW-01, GROW-02, GROW-03]

# Metrics
duration: 10min
completed: 2026-04-26
---

# Phase 3 Plan 02: Builder Worker Summary

**Builder worker with ROI-based poll loop that evaluates all planets, picks highest-ROI candidate, respects max-level caps, and wires into main.go alongside defender**

## Performance

- **Duration:** 10m
- **Started:** 2026-04-26T07:48:24Z
- **Completed:** 2026-04-26T07:58:32Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Builder worker evaluates ROI for 5 resource buildings (MetalMine, CrystalMine, DeutSynth, SolarPlant, FusionReactor) across all planets each poll tick
- Picks highest-ROI candidate when a build slot is free (GROW-02)
- Respects max-level caps from config and per-planet overrides (GROW-03)
- Skips planets with no free fields (FieldsUsed >= FieldsTotal)
- Skips planets with active construction (Building.ID != 0)
- Energy surplus checked via CalculateROI before recommending energy-consuming mines (GROW-01)
- Anti-detection: 7% chance to pick 2nd-best ROI candidate
- Every build recorded to build_events table for audit trail (repudiation mitigation)
- Builder wired in main.go — starts in goroutine when autoBuild.enabled is true
- 15 builder-specific tests + 30+ existing ROI tests all pass

## Task Commits

TDD cycle: RED → GREEN → wire:

1. **Task 1 RED: Failing tests for builder worker** - `b9cbca0` (test)
2. **Task 1 GREEN: Implement builder worker** - `086fbd2` (feat)
3. **Task 2: Wire builder into main.go + fix test determinism** - `e6d8464` (feat)

## Files Created/Modified
- `internal/builder/builder.go` - Builder worker: BuilderStateReader interface, Builder struct with poll loop, resolveMaxLevel, buildingLevel, recordBuildEvent, anti-detection
- `internal/builder/builder_test.go` - 15 tests: constructor, resolveMaxLevel (global/override/missing), buildingLevel (all 5 buildings + unknown), poll scenarios (highest ROI, skip construction, skip full planet, max-level cap, per-planet override, disabled, build event recording, API errors, empty planets, ROI sorting)
- `cmd/bot/main.go` - Builder wiring alongside defender (import builder package, start in goroutine when enabled)

## Decisions Made
- Anti-detection probability made configurable via antiDetectPct field — avoids test flakiness from random selection while keeping 7% default in production
- Builder skips entire planet on GetConstructions error (conservative approach — better to skip than build when state is uncertain)
- Build events recorded only after successful BuildBuilding call — no phantom records on API failure
- Missing max-level config returns 0, effectively disabling that building type — explicit config required

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test expectations had wrong ROI assumptions**
- **Found during:** Task 1 GREEN phase
- **Issue:** TestPollPicksHighestROI expected Metal Mine 5→6 to beat Crystal Mine 3→4, but Crystal Mine at low levels has better ROI score (cheaper cost, good return)
- **Fix:** Adjusted test building levels to create clear ROI winner (Metal Mine 1→2 on Colony beats everything)
- **Files modified:** internal/builder/builder_test.go
- **Commit:** 086fbd2

**2. [Rule 1 - Bug] Anti-detection randomness caused flaky tests**
- **Found during:** Task 2 verification
- **Issue:** 7% random chance to pick 2nd-best ROI candidate triggered in CI, causing TestPollPicksHighestROI to fail non-deterministically
- **Fix:** Made anti-detection probability a configurable field (antiDetectPct), set to 0 in tests via testBuilder helper
- **Files modified:** internal/builder/builder.go, internal/builder/builder_test.go
- **Commit:** e6d8464

## User Setup Required
None — builder starts automatically when `features.autoBuild.enabled: true` in config.yaml.

## Next Phase Readiness
- Builder worker fully operational — will auto-build highest-ROI buildings when enabled
- Ready for Phase 3 completion and Phase 4 (Auto-Farm) or Phase 5 (Dashboard) development
- Build events table populated for dashboard consumption

## Self-Check: PASSED

- Created files verified: internal/builder/builder.go, internal/builder/builder_test.go
- Commit hashes verified: b9cbca0, 086fbd2, e6d8464
- All tests pass: `go test ./... -count=1` → ok (8 packages)
- `go vet ./...` → clean (no issues)

---
*Phase: 03-auto-build*
*Completed: 2026-04-26*
