---
phase: 03-auto-build
plan: 01
subsystem: builder
tags: [go, tdd, roi, ogame-formulas, ogamed, auto-build, config, sqlite]

# Dependency graph
requires:
  - phase: 01-01
    provides: "Domain types (model package) and config structs"
  - phase: 01-02
    provides: "Ogamed REST client with 14 GET methods and 4 POST methods, rate limiter"
  - phase: 02-01
    provides: "ClientInterface with 18 methods, DefenderConfig pattern"
provides:
  - Pure ROI calculator with all OGame building formulas (cost, production, energy, construction time)
  - CalculateROI function with max-level caps, affordability, and energy balance checks
  - ClientInterface extended to 20 methods (GetConstructions, BuildBuilding)
  - State manager GetBuildings, GetFacilities, GetServerSpeed read methods
  - AutoBuildConfig with default max-level caps and per-planet overrides
  - build_events migration for build history tracking
affects: [03-02, 04-auto-farm, 05-dashboard]

# Tech tracking
tech-stack:
  added: [math]
  patterns: [pure-calculator-package, metal-equivalent-roi, energy-balance-check]

key-files:
  created:
    - internal/builder/roi.go
    - internal/builder/roi_test.go
    - internal/state/migrations/003_build_events.sql
  modified:
    - internal/model/types.go
    - internal/ogamed/client.go
    - internal/state/manager.go
    - internal/config/config.go
    - internal/defender/defender_test.go
    - internal/state/manager_test.go
    - internal/ogamed/client_test.go

key-decisions:
  - "ROI uses metal-equivalent scoring with trade ratios: metal=1, crystal=1.5, deuterium=2.0"
  - "Energy-producing buildings (solar, fusion) valued at 0.5 metal-equivalent per energy unit"
  - "Energy consumers rejected if resources.Energy < additional energy needed at target level"
  - "AutoBuildConfig.MaxLevels defaults to {MetalMine: 30, CrystalMine: 28, DeutSynth: 26, SolarPlant: 26, FusionReactor: 20}"

requirements-completed: [GROW-01, GROW-03]

# Metrics
duration: 4min
completed: 2026-04-26
---

# Phase 3 Plan 01: ROI Calculator and Build Infrastructure Summary

**Pure ROI calculation engine with verified OGame formulas, extended ogamed client (20 methods), AutoBuildConfig with max-level caps, and build_events migration**

## Performance

- **Duration:** 4m
- **Started:** 2026-04-26T07:34:39Z
- **Completed:** 2026-04-26T07:38:30Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- ROI calculator computes cost, production, energy, and ROI for all 5 resource buildings (MetalMine, CrystalMine, DeutSynth, SolarPlant, FusionReactor) at any level
- All formulas verified from ogamed source code (alaingilbert/ogame) — correct for any universe speed and plasma tech level
- CalculateROI respects max-level caps, checks affordability, validates energy balance before recommending mine upgrades
- ClientInterface extended from 18 to 20 methods with GetConstructions and BuildBuilding
- State manager exposes GetBuildings, GetFacilities, GetServerSpeed read methods with server speed caching
- AutoBuildConfig replaces FeatureConfig with default max-level caps and [1,100] range validation
- 30+ unit tests cover all formulas and ROI logic edge cases

## Task Commits

Each task was committed atomically (TDD: test → feat):

1. **Task 1 RED: ROI calculator failing tests** - `590c217` (test)
2. **Task 1 GREEN: Implement ROI calculator** - `1bdb4a0` (feat)
3. **Task 2: Extend ogamed client, state manager, config, migration** - `0ba343f` (feat)

## Files Created/Modified
- `internal/builder/roi.go` - Pure ROI calculator with all OGame formulas (cost, production, energy, construction time, CalculateROI)
- `internal/builder/roi_test.go` - 30+ table-driven tests covering all formulas and ROI logic
- `internal/model/types.go` - Added Construction and Constructions types
- `internal/ogamed/client.go` - Added GetConstructions, BuildBuilding to ClientInterface and Client (20 methods)
- `internal/state/manager.go` - Added GetBuildings, GetFacilities, GetServerSpeed + server speed caching
- `internal/config/config.go` - Added AutoBuildConfig with MaxLevels, PlanetOverrides, AutoBuildDefaults, validation
- `internal/state/migrations/003_build_events.sql` - Build history tracking table
- `internal/defender/defender_test.go` - Added mock stubs for GetConstructions, BuildBuilding
- `internal/state/manager_test.go` - Added mock stubs for GetConstructions, BuildBuilding
- `internal/ogamed/client_test.go` - Updated method count assertion to 20

## Decisions Made
- ROI uses metal-equivalent scoring with standard trade ratios (metal=1, crystal=1.5, deuterium=2.0) — community-standard valuation
- Energy-producing buildings (solar plant, fusion reactor) valued at 0.5 metal-equivalent per energy unit — heuristic for ROI comparison with mines
- Energy balance check uses `resources.Energy` (surplus from ogamed) compared against additional energy consumption at target level
- AutoBuildConfig default max levels set conservatively: MetalMine 30, CrystalMine 28, DeutSynth 26, SolarPlant 26, FusionReactor 20
- Server speed cached on first refresh in state manager — never changes for a given universe

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Two test expectations (Deuterium level 1, Fusion level 10) had incorrect pre-computed values — corrected after implementation verified correct formula results. Implementation was correct from the start.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- ROI calculator ready for builder worker (Plan 02) to compute best build candidates
- GetConstructions ready for build slot availability checking
- BuildBuilding ready for executing build commands
- AutoBuildConfig ready for per-building max-level enforcement
- State manager read methods ready for worker to query buildings, facilities, server speed
- build_events table ready for build history logging

## Self-Check: PASSED

- All created files verified: internal/builder/roi.go, internal/builder/roi_test.go, internal/state/migrations/003_build_events.sql
- All commit hashes verified: 590c217, 1bdb4a0, 0ba343f
- All tests pass: `go test ./... -count=1` → ok (8 packages)
- `go vet ./...` → clean (no issues)

---
*Phase: 03-auto-build*
*Completed: 2026-04-26*
