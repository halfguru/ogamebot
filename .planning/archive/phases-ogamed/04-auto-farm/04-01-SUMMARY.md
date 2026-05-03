---
phase: 04-auto-farm
plan: 01
subsystem: ogamed-client
tags: [galaxy-scan, espionage, auto-farm, config, migration]
dependency_graph:
  requires: [phase-03]
  provides: [GetGalaxyInfos, GetEspionageReportMessages, GetEspionageReport, DeleteAllEspionageReports, AutoFarmConfig, farm_targets, farm_attacks]
  affects: [internal/ogamed, internal/config, internal/model, internal/farmer, internal/state, internal/builder, internal/defender]
tech_stack:
  added: []
  patterns: [getTyped generic deserialization, type alias for shared types, inline FeatureConfig embedding]
key_files:
  created:
    - internal/farmer/types.go
    - internal/state/migrations/004_farm.sql
  modified:
    - internal/model/types.go
    - internal/ogamed/client.go
    - internal/config/config.go
    - internal/builder/builder_test.go
    - internal/defender/defender_test.go
    - internal/state/manager_test.go
decisions:
  - GalaxyRange type alias (= model.GalaxyRange) in config package to avoid circular imports
  - AutoFarmConfig embeds FeatureConfig inline for YAML flattening
  - AutoFarm poll interval minimum 60s (10x slower than auto-build at 10s)
metrics:
  duration: 7m
  completed: 2026-04-26
---

# Phase 4 Plan 01: Auto-Farm Foundation Summary

Extended ogamed client with galaxy scanning and espionage report methods, added auto-farm domain types, config with galaxy ranges, and SQLite migration for farm tables.

## Commits

| Commit   | Message                                                    |
|----------|------------------------------------------------------------|
| 59e4b9e  | test(04-01): add failing tests for galaxy scanning and espionage types (TDD RED) |
| 97e2879  | feat(04-01): add galaxy scanning and espionage report domain types (TDD GREEN) |
| 90f4d97  | feat(04-01): add client methods, AutoFarmConfig, and farm DB migration |

## Tasks Completed

### Task 1: Add domain types for galaxy scanning and espionage reports (TDD)

**Files:** `internal/model/types.go`, `internal/farmer/types.go`

- Added `SystemInfos`, `PlanetPosition` types for galaxy scan results with inactive/vacation/banned player flags
- Added `EspionageReportSummary`, `EspionageReport` types for espionage report data with defense fields and `HasDefensesInformation` guard
- Added `GalaxyRange` type with YAML tags for config galaxy/system ranges
- Created `internal/farmer/types.go` with `FarmerStateReader` interface and `FarmTarget` scoring type
- TDD: RED commit (failing tests) → GREEN commit (implementation)

### Task 2: Add client methods, config, and DB migration

**Files:** `internal/ogamed/client.go`, `internal/config/config.go`, `internal/state/migrations/004_farm.sql`

- Extended `ClientInterface` with 4 new methods: `GetGalaxyInfos`, `GetEspionageReportMessages`, `GetEspionageReport`, `DeleteAllEspionageReports`
- All methods use existing `getTyped`/`postTyped` generic pattern
- Replaced `AutoFarm FeatureConfig` with `AutoFarmConfig` containing `GalaxyRanges`, `MinProfitThreshold`, `MaxProbesPerTarget`, `MaxAttacksPerCycle`, `SkipDefended`
- Added config validation: autoFarm requires galaxy ranges when enabled, poll interval ≥ 60s
- Created `004_farm.sql` migration with `farm_targets` and `farm_attacks` tables
- Updated all mock clients across builder, defender, and state test packages

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated mock client in state/manager_test.go**
- **Found during:** Task 2 build
- **Issue:** Plan only mentioned updating builder_test.go and defender_test.go mock clients, but state/manager_test.go also implements ClientInterface and failed to compile
- **Fix:** Added the 4 new interface methods to state package's mockClient
- **Files modified:** `internal/state/manager_test.go`
- **Commit:** 90f4d97

## Verification Results

```
go build ./...              # PASS — all packages compile
go test ./... -count=1      # PASS — all tests pass across 8 packages
ClientInterface methods     # 21 (17 existing + 4 new)
AutoFarmConfig type         # confirmed
004_farm.sql migration      # confirmed (farm_targets + farm_attacks tables)
```

## TDD Gate Compliance

- RED gate commit: 59e4b9e ✅
- GREEN gate commit: 97e2879 ✅
- REFACTOR gate: Not needed (clean implementation)

## Self-Check: PASSED

| Item | Status |
|------|--------|
| internal/model/types.go | FOUND |
| internal/farmer/types.go | FOUND |
| internal/ogamed/client.go | FOUND |
| internal/config/config.go | FOUND |
| internal/state/migrations/004_farm.sql | FOUND |
| 59e4b9e (RED commit) | FOUND |
| 97e2879 (GREEN commit) | FOUND |
| 90f4d97 (Task 2 commit) | FOUND |
