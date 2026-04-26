---
phase: 02-fleet-safety
plan: 01
subsystem: ogamed-client
tags: [go, tdd, ogamed, post-methods, fleet-save, defender-config, mission-constants]

# Dependency graph
requires:
  - phase: 01-01
    provides: "Domain types (model package) and config structs (RateLimitConfig)"
  - phase: 01-02
    provides: "Ogamed REST client with 14 GET methods, rate limiter, retry, envelope validation"
provides:
  - ClientInterface with 18 methods (4 new: SendFleet, CancelFleet, GetAttacks, GetSlots)
  - post() and postTyped[T]() methods mirroring get/getTyped pattern
  - AttackEvent, SendFleetRequest, Slots domain types
  - DefenderConfig with safety margins, reaction delays, and defaults
  - Fixed mission constants (MissionHold=5, MissionMissileAttack=10, no collisions)
affects: [02-02, 02-03, 03-auto-build, 04-auto-farm]

# Tech tracking
tech-stack:
  added: [net/url, strconv]
  patterns: [post-mirrors-get-pattern, pointer-bool-for-yaml-defaults, url-encoded-form-body]

key-files:
  created: []
  modified:
    - internal/ogamed/client.go
    - internal/ogamed/client_test.go
    - internal/model/types.go
    - internal/model/types_test.go
    - internal/constants/missions.go
    - internal/constants/constants_test.go
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/state/manager_test.go
    - config.example.yaml

key-decisions:
  - "RecallEnabled uses *bool pointer to distinguish YAML not-set from explicitly false, with DefenderDefaults() applying true as safe default"
  - "Ships encoded as repeated params (ships=204,100&ships=203,50) matching ogamed's expected format"
  - "MissionACSTransport removed — replaced with MissionHold=5 (Hold position, correct OGame mission)"

requirements-completed: [SAFE-01, SAFE-02]

# Metrics
duration: 5min
completed: 2026-04-26
---

# Phase 2 Plan 01: Fleet-Safe Client Extensions Summary

**Extended ogamed client from 14 to 18 methods with POST operations (SendFleet, CancelFleet, GetAttacks, GetSlots), new domain types (AttackEvent, SendFleetRequest, Slots), DefenderConfig with safety margins, and fixed mission constants**

## Performance

- **Duration:** 5m
- **Started:** 2026-04-26T06:27:18Z
- **Completed:** 2026-04-26T06:32:00Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- ClientInterface extended from 14 to 18 methods with 4 new fleet-safety endpoints
- post() and postTyped[T]() methods mirror get()/getTyped[T] pattern: rate limit → retry → envelope validation
- SendFleet sends url-encoded POST body with ships as repeated params matching ogamed format
- AttackEvent type with nullable Ships field for incoming attack data from ogamed
- SendFleetRequest type for constructing fleet dispatch calls
- Slots type with fleet and expedition slot tracking (InUse/Total/ExpInUse/ExpTotal)
- Mission constants fixed: MissionHold=5 replaces incorrect MissionACSTransport, MissionMissileAttack=10 added
- DefenderConfig replaces FeatureConfig for defender with safety margins, reaction delays, recall toggle
- DefenderDefaults() applies safe defaults: SafetyMarginMs=120000, RecallEnabled=true, MaxReturnFlightS=600
- Config validation enforces SafetyMarginMs >= 10000, MinReactionDelayS >= 5

## Task Commits

Each task was committed atomically (TDD: test → feat):

1. **Task 1 RED: Domain types and mission constants tests** - `60bcade` (test)
2. **Task 1 GREEN: Implement types and fix constants** - `bf18f2b` (feat)
3. **Task 2 RED: POST methods and DefenderConfig tests** - `641ea7d` (test)
4. **Task 2 GREEN: Implement POST methods and DefenderConfig** - `be824d6` (feat)

## Files Created/Modified
- `internal/ogamed/client.go` - Added post(), postTyped[T](), SendFleet, CancelFleet, GetAttacks, GetSlots (18 interface methods)
- `internal/ogamed/client_test.go` - Added 7 new tests: SendFleet body encoding, CancelFleet, GetAttacks, GetSlots, POST rate limiter, interface compliance (18 methods)
- `internal/model/types.go` - Added AttackEvent (nullable Ships), SendFleetRequest, Slots types
- `internal/model/types_test.go` - Added 4 new tests: AttackEvent JSON, AttackEvent null ships, Slots JSON, SendFleetRequest fields
- `internal/constants/missions.go` - Replaced MissionACSTransport with MissionHold=5, added MissionMissileAttack=10
- `internal/constants/constants_test.go` - Updated test table, added uniqueness test, new constants verification
- `internal/config/config.go` - Added DefenderConfig with *bool RecallEnabled, DefenderDefaults(), validation rules
- `internal/config/config_test.go` - Added 3 new tests: load with fields, defaults applied, validation rejects invalid
- `internal/state/manager_test.go` - Added 4 mock methods for new interface methods
- `config.example.yaml` - Updated with full defender safety config fields and comments

## Decisions Made
- RecallEnabled uses `*bool` pointer to distinguish "not set in YAML" (nil → default true) from "explicitly set to false" — standard Go pattern for YAML default handling
- Ships encoded as repeated `ships` params (e.g., `ships=204,100&ships=203,50`) matching ogamed's expected POST format
- MissionACSTransport was incorrect — replaced with MissionHold=5 (Hold position is the real OGame mission type 5)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated state manager mock client for new interface methods**
- **Found during:** Task 2 GREEN phase
- **Issue:** ClientInterface grew from 14 to 18 methods, breaking state/manager_test.go mockClient compilation
- **Fix:** Added GetAttacks, GetSlots, SendFleet, CancelFleet stub methods to mockClient struct
- **Files modified:** internal/state/manager_test.go
- **Verification:** Full test suite passes `go test ./... -count=1`
- **Committed in:** be824d6 (Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 blocking issue)
**Impact on plan:** Minimal — downstream mock client updated to match new interface

## TDD Gate Compliance

| Gate | Commit | Description |
|------|--------|-------------|
| RED (Task 1) | `60bcade` | Tests for AttackEvent, Slots, SendFleetRequest, MissionHold, MissionMissileAttack — fail to compile |
| GREEN (Task 1) | `bf18f2b` | Types and constants implemented — all tests pass |
| RED (Task 2) | `641ea7d` | Tests for SendFleet, CancelFleet, GetAttacks, GetSlots, DefenderConfig — fail to compile |
| GREEN (Task 2) | `be824d6` | POST methods and DefenderConfig implemented — all tests pass |

All TDD gates satisfied: 2 RED commits, 2 GREEN commits. No REFACTOR commits needed.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- SendFleet and CancelFleet ready for defender worker (Plan 02-03) to perform fleet-save
- GetAttacks ready for attack detection polling
- GetSlots ready for fleet slot availability checking
- DefenderConfig ready for safety margin calculations

## Self-Check: PASSED

- All modified files verified in git log (4 commits)
- All commit hashes verified: 60bcade, bf18f2b, 641ea7d, be824d6
- All tests pass: `go test ./... -count=1` → ok
- ClientInterface method count verified: 18

---
*Phase: 02-fleet-safety*
*Completed: 2026-04-26*
