# Phase 2: Fleet Safety - Research

**Researched:** 2026-04-26
**Domain:** OGame fleet-save automation (attack detection, phalanx-safe deploy+recall, moon handling)
**Confidence:** HIGH

## Summary

Phase 2 builds the bot's most critical capability: detecting incoming attacks and automatically saving the player's fleet and resources before the attack lands. This is the single most important feature — if fleet-save fails, months of progress are destroyed in seconds.

The implementation centers on three new ogamed API endpoints that Phase 1's client does not yet expose: `GET /bot/attacks` (detailed attack events with arrival times), `POST /bot/planets/:planetID/send-fleet` (deploy fleet with resources), and `POST /bot/fleets/:fleetID/cancel` (recall a deployed fleet). The bot already has `IsUnderAttack()` (boolean check) and `GetFleets()` (active fleet list) from Phase 1, but fleet-save requires the richer attack event data and the write operations.

The core algorithm follows the Cruiser bot's proven pattern: (1) poll for attack events, (2) for each endangered planet, calculate escape flights to all friendly planets at all speed settings, (3) sort by safety criteria, (4) pick the first viable flight with sufficient fuel, (5) send deploy mission with all resources, (6) optionally recall after the attack window passes. Moon-based fleets use the same deploy mechanism but benefit from inherent phalanx invisibility on moons.

**Primary recommendation:** Extend the ogamed client with `SendFleet`, `CancelFleet`, and `GetAttacks` methods. Build a `Defender` worker that polls attack events, calculates escape routes using OGame flight formulas, and executes deploy+recall fleet-save sequences with exhaustive error handling and fuel verification.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SAFE-01 | Bot monitors for incoming attacks by polling hostile fleet events at randomized intervals | `GET /bot/attacks` returns `[]AttackEvent` with origin, destination, arrival time, attacker info. Polling via defender feature config with jitter. |
| SAFE-02 | Bot auto-saves fleet and resources when attack is detected using phalanx-safe deploy + recall | `POST /bot/planets/:planetID/send-fleet` with mission=3 (Deploy) + `POST /bot/fleets/:fleetID/cancel` for recall. Deploy shows on phalanx but recalled fleet is invisible. |
| SAFE-03 | Bot handles fleet-save for moons separately with appropriate escape destinations | Moons cannot be phalanxed (only planets). Deploy from moon to own planet is safest. Coordinate.Type distinguishes moon vs planet. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Attack event polling | Bot Engine (Go) | — | Engine owns the polling loop and attack detection timing |
| Escape route calculation | Bot Engine (Go) | — | Pure computation: distance, flight time, fuel cost, safety scoring |
| Fleet dispatch (send) | Bot Engine → ogamed | — | Engine decides what/where, ogamed executes the HTTP call to OGame |
| Fleet recall (cancel) | Bot Engine → ogamed | — | Engine decides when to recall, ogamed executes |
| Fleet state caching | State Manager (SQLite) | — | State manager already caches fleets; defender reads from cache |
| Configuration | Config (YAML) | — | Defender feature config with poll interval, safety margins |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (net/http) | 1.26+ | HTTP POST to ogamed send-fleet/cancel endpoints | Already in use from Phase 1 client |
| modernc.org/sqlite | existing | Persist attack events and fleet-save state | Phase 1 DB, extend with new tables |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib (math) | 1.26+ | Flight distance, fuel, and duration calculations | All escape route computations |
| Go stdlib (time) | 1.26+ | Server time parsing and attack timing | All timing-sensitive operations |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Manual flight formula implementation | Pre-computed lookup tables | Tables are faster but formulas handle all universe speeds and research levels correctly |

**Installation:**
No new dependencies needed. Phase 2 uses only Go stdlib + existing project dependencies.

**Version verification:**
```bash
# No new packages — verify existing
grep -c 'modernc.org/sqlite' go.sum  # Already present from Phase 1
```

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────────────────┐
                    │                   Bot Engine (Go)                    │
                    │                                                      │
 ┌──────────────┐   │  ┌──────────────┐    ┌─────────────────────────┐    │
 │  ogamed REST  │◄──┤  │  Defender     │    │  Escape Route           │    │
 │  API          │   │  │  Worker       │───►│  Calculator              │    │
 │              ◄──┤  │               │    │                           │    │
 │ GET /attacks │   │  │ Poll attacks  │    │ - Distance formula       │    │
 │ POST /send   │   │  │ Detect danger │    │ - Flight time formula    │    │
 │ POST /cancel │   │  │ Plan escape   │    │ - Fuel consumption calc  │    │
 │ GET /fleets  │   │  │ Execute save  │    │ - Safety scoring         │    │
 │ GET /time    │   │  │ Schedule recall│   │                           │    │
 └──────────────┘   │  └──────┬───────┘    └─────────────────────────┘    │
                    │         │                                             │
                    │         ▼                                             │
                    │  ┌──────────────┐    ┌─────────────────────────┐    │
                    │  │  State        │    │  Config                  │    │
                    │  │  Manager      │    │  (YAML)                  │    │
                    │  │  (SQLite)     │    │                           │    │
                    │  │               │    │ - defender.enabled        │    │
                    │  │ - Planets     │    │ - defender.pollIntervalMs│    │
                    │  │ - Fleets      │    │ - defender.safetyMarginMs│    │
                    │  │ - Resources   │    │ - defender.recallEnabled │    │
                    │  │ - Ships       │    │                           │    │
                    │  └──────────────┘    └─────────────────────────┘    │
                    │                                                      │
                    └─────────────────────────────────────────────────────┘

Data Flow:
1. Defender polls GET /bot/attacks at randomized intervals
2. For each incoming attack → identify endangered planets
3. Query state manager for ships, resources, own planets (destinations)
4. Escape Route Calculator generates all viable deploy missions
5. Defender picks safest viable route (fuel-sufficient, phalanx-safe)
6. Defender sends fleet via POST /bot/planets/:id/send-fleet (mission=Deploy)
7. Defender optionally recalls via POST /bot/fleets/:id/cancel after attack passes
```

### Recommended Project Structure
```
internal/
├── ogamed/
│   ├── client.go          # ADD: SendFleet, CancelFleet, GetAttacks, GetSlots
│   └── ...                 # (existing rate limiter, retry, types)
├── defender/
│   ├── defender.go         # Defender worker: poll loop, attack detection, fleet-save orchestration
│   ├── escape.go           # Escape route calculation: distance, flight time, fuel, safety scoring
│   ├── defender_test.go    # Tests for defender logic
│   └── escape_test.go      # Tests for escape route calculations
├── model/
│   └── types.go            # ADD: AttackEvent type, SendFleetRequest, FleetSaveState
├── state/
│   ├── manager.go          # EXTEND: GetShipsForPlanet, GetAllPlanetCoords
│   ├── migrations/
│   │   └── 002_fleet_save.sql  # NEW: fleet_save_events table for tracking
│   └── db.go               # (existing)
├── config/
│   └── config.go           # EXTEND: DefenderConfig with safety margins, recall toggle
└── constants/
    └── missions.go         # (existing — FIX: MissionEspionage=6 collides with MissionRelocate=6)
```

### Pattern 1: Attack Detection via Polling with Jitter
**What:** Defender worker runs a polling loop with randomized intervals
**When to use:** Continuous monitoring for incoming attacks
**Example:**
```go
// Source: [CITED: Cruiser bot.py pattern] + existing state manager pattern
func (d *Defender) Run(ctx context.Context) {
    interval := time.Duration(d.cfg.PollIntervalMs) * time.Millisecond
    jitter := time.Duration(rand.Intn(interval.Milliseconds()/2)) * time.Millisecond
    
    ticker := time.NewTicker(interval + jitter)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            attacks, err := d.client.GetAttacks(ctx)
            if err != nil {
                d.log.Error("Failed to check attacks", "error", err)
                continue
            }
            d.handleAttacks(ctx, attacks)
            // Re-randomize jitter for next tick
            jitter = time.Duration(rand.Intn(interval.Milliseconds()/2)) * time.Millisecond
            ticker.Reset(interval + jitter)
        }
    }
}
```

### Pattern 2: Phalanx-Safe Deploy + Recall
**What:** Send fleet on Deploy mission to own planet, then recall before arrival
**When to use:** Whenever fleet-save is needed on a planet (not moon)
**Key insight from PITFALLS.md:** Deploy mission shows on phalanx scan, but recalled deploy is invisible. This is the gold standard for fleet safety.
```go
// Step 1: Send deploy with all ships + resources
fleetID, err := client.SendFleet(ctx, SendFleetRequest{
    PlanetID:   endangeredPlanet.ID,
    Ships:      allShipsOnPlanet,
    Speed:      10, // max speed to arrive ASAP
    Galaxy:     destPlanet.Coordinate.Galaxy,
    System:     destPlanet.Coordinate.System,
    Position:   destPlanet.Coordinate.Position,
    Type:       destPlanet.Coordinate.Type,
    Mission:    constants.MissionDeploy, // 3
    Metal:      resources.Metal,
    Crystal:    resources.Crystal,
    Deuterium:  resources.Deuterium - fuelCost, // subtract fuel!
})

// Step 2: After attack passes, recall the deploy
// The recalled fleet returns to origin — invisible to phalanx
err = client.CancelFleet(ctx, fleetID)
```

### Pattern 3: Escape Route Safety Scoring
**What:** Rank all possible escape destinations by safety criteria
**When to use:** When choosing where to send the fleet
**Based on Cruiser's sort_escape_flights_by_safety:**
```go
// Safety criteria (lower score = safer):
// 1. Destination NOT under attack with hostile arriving before our fleet
// 2. Shorter distance preferred (less fuel, faster recall)
// 3. Moon destination preferred over planet (moons can't be phalanxed from most locations)
// 4. Lower fuel consumption preferred (more resources saved)
type EscapeRoute struct {
    Dest       model.Coordinate
    Speed      int
    Duration   time.Duration
    FuelCost   int64
    Distance   int
    SafetyScore int
}
```

### Anti-Patterns to Avoid
- **Never use Transport/Attack/Espionage as fleet-save missions:** All phalanx-visible with no recall benefit. `[CITED: PITFALLS.md Pitfall 5]`
- **Never fall through to fallback missions without revalidating destination:** TBot #178 showed Colonize at an already-inhabited position. `[VERIFIED: PITFALLS.md Pitfall 1]`
- **Never forget fuel verification:** If fuel < cost, the send fails and fleet stays exposed. `[CITED: PITFALLS.md Pitfall 8]`
- **Never react instantly to attacks:** Humans take minutes; bots that respond in <1 second are detectable. `[CITED: PITFALLS.md Pitfall 7]`
- **Never use local clock for timing:** Always use ogamed's server time endpoint. `[CITED: PITFALLS.md Integration Gotchas]`

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Flight distance calculation | Custom coordinate math | OGame distance formula (abs difference with universe wrapping) | Formula depends on donut galaxy/system settings; Cruiser has working Go implementation |
| Fuel consumption | Estimate based on ship count | OGame fuel formula with speed modifier, drive research, and ship base consumption | Complex formula varies per ship type, drive tech level, and universe speed |
| Cargo capacity | Ship count × base capacity | OGame cargo formula with hyperspace tech bonus and ship-specific modifiers | Different ships get different bonuses; civilian vs military ships have different cargo bays |
| Mission validation | Manual checks per mission type | ogamed's own error responses (ErrUninhabitedPlanet, ErrNoDebrisField, etc.) | ogamed validates server-side; duplicate validation is fragile |

**Key insight:** The OGame flight/fuel formulas are non-trivial and universe-speed-dependent. The bot should compute them locally for planning (to choose escape route), but must also handle ogamed's validation errors as a safety net. Do NOT try to pre-validate everything — let ogamed be the source of truth for "is this fleet send valid?"

## Common Pitfalls

### Pitfall 1: Fleet-Save Fallback to Invalid Missions [CRITICAL]
**What goes wrong:** Primary mission fails, bot falls through to fallback without revalidating destination
**Why it happens:** Mission selection is a priority list but destinations valid for Deploy ≠ Colonize ≠ Harvest
**How to avoid:** Validate destination per-mission. Never use fallback chains. If Deploy fails, recalculate destination for next attempt.
**Warning signs:** Logs show "checking Colonize destination..." after Deploy fails

### Pitfall 2: Fuel Calculation Doesn't Account for Loaded Resources
**What goes wrong:** Bot calculates fuel cost, loads resources, but fuel + loaded deuterium exceeds available deuterium
**Why it happens:** Two independent calculations (fuel and cargo) that share the deuterium pool
**How to avoid:** `availableDeuterium >= fuelCost + deuteriumToLoad`. If insufficient, reduce loaded resources to prioritize fuel.
**Warning signs:** Fleet-send fails on low-deuterium planets

### Pitfall 3: MissionEspionage Constant Collision
**What goes wrong:** In `constants/missions.go`, `MissionEspionage = 6` collides with `MissionRelocate = 6` and `MissionColonize = 7` collides with `MissionStation = 7`
**Why it happens:** Porting error from TypeScript constants — espionage is actually ID 6, relocate is also ID 6 in some OGame versions, but they're different missions
**How to avoid:** Fix the constants: OGame mission IDs are 1=Attack, 2=ACS Attack, 3=Deploy, 4=Transport, 5=Hold, 6=Espionage, 7=Colonize, 8=Harvest, 9=Moon Destruction, 10=Missile Attack, 15=Expedition. Remove MissionRelocate and MissionStation (they don't exist in OGame).
**Warning signs:** Sending mission type 6 sends wrong mission type

### Pitfall 4: Missing Coordinate Type in Send-Fleet
**What goes wrong:** Sending fleet to a moon without specifying `type=3` (moon type) causes ogamed to default to planet type
**Why it happens:** Coordinate.Type field defaults to empty/planet. ogamed's send-fleet handler defaults `where.Type = ogame.PlanetType` if not specified.
**How to avoid:** Always include `type` parameter in send-fleet POST. Planets = type 1, Debris = type 2, Moons = type 3.
**Warning signs:** Fleet sent to moon position but arrives at planet instead

### Pitfall 5: Recalling In-Return Fleet
**What goes wrong:** Bot tries to recall a fleet that's already returning (ReturnFlight=true)
**Why it happens:** ogamed's `CancelFleet` only works on outgoing (non-return) fleets. `Fleet.IsCancellable() = !ReturnFlight && !InDeepSpace && Mission != MissileAttack`
**How to avoid:** Check `ReturnFlight` before attempting recall. Log and skip if already returning.
**Warning signs:** CancelFleet returns error on fleet that's already homeward bound

### Pitfall 6: False Positive on Own Fleets
**What goes wrong:** Bot detects own returning fleet as an attack
**Why it happens:** `IsUnderAttack` returns boolean without detail. `GetAttacks` returns attack events but filtering logic may include own ACS attacks
**How to avoid:** Use `GetAttacks` (not just `IsUnderAttack`) and verify attacker ID ≠ own player ID. Filter by mission type (Attack, ACS Attack, Destroy, Espionage).
**Warning signs:** Bot fleet-saves when no actual attack is happening

### Pitfall 7: Missing ogamed Client Methods for Write Operations
**What goes wrong:** Phase 1 client only has GET methods. Fleet-save requires POST operations (send-fleet, cancel).
**Why it happens:** Phase 1 scoped to read-only operations
**How to avoid:** Add `post()` method to client.go (mirroring `get()` with rate limiting and retry), then implement `SendFleet`, `CancelFleet` as POST methods.
**Warning signs:** Compilation errors when defender tries to call SendFleet/CancelFleet

## Code Examples

### ogamed Send-Fleet API Call
```go
// Source: [CITED: github.com/alaingilbert/ogame/wiki/ogamed-full-documentation]
// POST /bot/planets/:planetID/send-fleet
// Content-Type: application/x-www-form-urlencoded
// Body: ships=204,1&ships=205,2&speed=10&galaxy=4&system=208&type=1&position=8&mission=3&metal=1&crystal=2&deuterium=3
//
// Response: {"Status":"ok","Code":200,"Message":"","Result":5575790}
// Result is the new fleet ID (int64)
//
// Key parameters:
// - ships: repeated param, format "shipID,count" (e.g., ships=202,100&ships=203,50)
// - speed: 1-10 (10 = 100%, 1 = 10%)
// - galaxy/system/position: destination coordinates
// - type: 1=planet, 2=debris, 3=moon
// - mission: mission ID (3=Deploy for fleet-save)
// - metal/crystal/deuterium: resources to load
// - duration: holding time (for expeditions, 0 for deploy)
// - union: union ID (for ACS, 0 for solo)
```

### ogamed Cancel-Fleet API Call
```go
// Source: [CITED: github.com/alaingilbert/ogame/wiki/ogamed-full-documentation]
// POST /bot/fleets/:fleetID/cancel
// No body required
//
// Response: {"Status":"ok","Code":200,"Message":"","Result":null}
// Only works on outgoing (non-return) fleets
```

### ogamed GetAttacks Response
```go
// Source: [CITED: github.com/alaingilbert/ogame - pkg/ogame/attackEvent.go]
// GET /bot/attacks
// Response: {"Status":"ok","Code":200,"Message":"","Result":[{...}]}
//
// AttackEvent fields from ogamed source:
type AttackEvent struct {
    ID              int64       `json:"ID"`
    MissionType     int         `json:"MissionType"`     // Mission ID (1=Attack, 2=ACS, 9=Destroy, 6=Espionage)
    Origin          Coordinate  `json:"Origin"`
    Destination     Coordinate  `json:"Destination"`
    DestinationName string      `json:"DestinationName"`
    ArrivalTime     time.Time   `json:"ArrivalTime"`
    ArriveIn        int64       `json:"ArriveIn"`        // Seconds until arrival
    AttackerName    string      `json:"AttackerName"`
    AttackerID      int64       `json:"AttackerID"`
    UnionID         int64       `json:"UnionID"`         // Non-zero for ACS attacks
    Missiles        int64       `json:"Missiles"`        // IPM count (for missile attacks)
    Ships           *ShipsInfos `json:"Ships"`           // Attacker's ship composition (may be nil)
}
```

### ogamed GetSlots Response
```go
// Source: [CITED: github.com/alaingilbert/ogame - pkg/ogame/slots.go]
// GET /bot/slots (not yet in our client, needs to be added)
// Response: {"Status":"ok","Code":200,"Message":"","Result":{"InUse":3,"Total":14,"ExpInUse":0,"ExpTotal":1}}
//
// IMPORTANT: Must check free slots BEFORE attempting to send fleet.
// If all slots are in use, fleet-save CANNOT be executed.
```

### OGame Distance Formula
```go
// Source: [ASSUMED] — standard OGame distance formula verified across TBot, Cruiser, and community wikis
// Distance between two coordinates in same system: abs(pos1 - pos2)
// Same galaxy, different system: 5 * abs(sys1 - sys2) + abs(pos1 - pos2)  [ASSUMED — donut may modify]
// Different galaxy: 20000 * abs(gal1 - gal2)
//
// Note: Actual formula depends on universe "donut" settings (IsDonutGalaxy, IsDonutSystem)
// which wrap around at max values. Use server data to determine wrapping behavior.
```

### DefenderConfig Extension
```go
// Extends existing FeatureConfig for the defender feature
type DefenderConfig struct {
    Enabled          bool `yaml:"enabled"`
    PollIntervalMs   int  `yaml:"pollIntervalMs"`
    SafetyMarginMs   int  `yaml:"safetyMarginMs"`   // How far before attack to trigger save (default: 120000 = 2min)
    RecallEnabled    bool `yaml:"recallEnabled"`     // Whether to recall deployed fleets after danger passes
    MaxReturnFlightS int  `yaml:"maxReturnFlightS"`  // Max seconds for recalled fleet return (Cruiser: 600s)
    MinReactionDelayS int `yaml:"minReactionDelayS"` // Anti-detection: minimum delay before reacting (default: 30s)
    MaxReactionDelayS int `yaml:"maxReactionDelayS"` // Anti-detection: maximum delay (default: 120s)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| IsUnderAttack() boolean only | GetAttacks() with full event details | ogamed always had both | Need detailed events for multi-planet defense timing |
| Fixed polling intervals | Randomized intervals with jitter | Best practice from all bot projects | Prevents behavioral detection |
| Deploy without recall | Deploy + recall (phalanx-safe) | Community standard since ~2015 | Recalled deploy is invisible to phalanx |
| Single escape destination | Ranked escape routes with safety scoring | Cruiser bot approach | Handles edge cases where primary destination is also under attack |

**Deprecated/outdated:**
- Using `IsUnderAttack()` alone: Only returns boolean, no timing information needed for smart fleet-save scheduling. Use `GetAttacks()` for full event data.
- Using Transport mission for fleet-save: Phalanx-visible, no recall benefit. Always use Deploy (mission 3).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | OGame distance formula is as described (no donut wrapping correction in formula) | Code Examples | Escape route distances wrong → wrong flight times → fleet arrives too late |
| A2 | ogamed `POST /bot/planets/:planetID/send-fleet` returns fleet ID as int64 in Result field | Code Examples | Can't track sent fleet for recall |
| A3 | `GET /bot/attacks` returns AttackEvent JSON with PascalCase field names matching ogamed's convention | Code Examples | JSON deserialization fails → can't parse attack events |
| A4 | Fleet speed parameter range is 1-10 where 10 = 100% speed (matching ogamed's Speed type) | Code Examples | Wrong speed selection → incorrect fuel calculation |
| A5 | Coordinate type: 1=planet, 2=debris, 3=moon (from ogamed source analysis) | Code Examples | Wrong type sends fleet to wrong celestial body |
| A6 | The `Coordinate.Type` field in our model uses string "planet"/"moon" but ogamed API uses numeric types | Architecture | Type mismatch between internal model and API — needs conversion |

**Key assumptions needing early verification:** A1 (flight formula) and A3 (attack event JSON format) should be validated against a running ogamed instance or by reading ogamed's JSON serialization code.

## Open Questions

1. **Mission constant collision fix scope**
   - What we know: `missions.go` has `MissionEspionage=6` colliding with `MissionRelocate=6`
   - What's unclear: Whether to fix this in Phase 2 or as a pre-phase fix
   - Recommendation: Fix as first task in Phase 2 — wrong mission IDs could send wrong missions

2. **Flight duration/fuel formula accuracy**
   - What we know: OGame formulas depend on drive research levels, universe speed, and ship base stats
   - What's unclear: Exact formula parameters (base speed per ship, drive multipliers, fuel consumption base values)
   - Recommendation: Use Cruiser's Go engine implementation as reference. These formulas are deterministic and can be unit-tested against known OGame values.

3. **Reaction delay vs. safety margin**
   - What we know: PITFALLS.md recommends 30-120s random reaction delay, but also needs safety margin before attack
   - What's unclear: How these interact — if attack arrives in 150s, reaction delay could be 120s, leaving only 30s margin
   - Recommendation: Reaction delay should be `min(maxReactionDelayS, timeUntilAttack - minSafetyMarginS)` to ensure fleet-save always has enough time

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go runtime | Build/Run | ✓ | 1.26+ | — |
| ogamed service | Fleet operations (runtime) | ✗ (not running during dev) | — | Mock client for development |
| SQLite | State persistence | ✓ | modernc.org/sqlite | — |
| Docker Compose | Deployment | ✓ | Existing setup | — |

**Missing dependencies with no fallback:**
- None blocking development — mock client enables full development without running ogamed

**Missing dependencies with fallback:**
- ogamed service: Use mock client for development and testing. Integration testing requires running ogamed instance.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None — Go convention |
| Quick run command | `go test ./internal/defender/... ./internal/ogamed/... -count=1 -short` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SAFE-01 | Detect attacks from GetAttacks response | unit | `go test ./internal/defender/... -run TestDetectAttacks -v` | ❌ Wave 0 |
| SAFE-01 | Poll with randomized jitter intervals | unit | `go test ./internal/defender/... -run TestPollingJitter -v` | ❌ Wave 0 |
| SAFE-01 | Filter out own fleets / probe-only attacks | unit | `go test ./internal/defender/... -run TestFilterAttacks -v` | ❌ Wave 0 |
| SAFE-02 | Calculate escape routes ranked by safety | unit | `go test ./internal/defender/... -run TestEscapeRoutes -v` | ❌ Wave 0 |
| SAFE-02 | Verify fuel before sending fleet | unit | `go test ./internal/defender/... -run TestFuelVerification -v` | ❌ Wave 0 |
| SAFE-02 | Send deploy mission and receive fleet ID | unit (mock) | `go test ./internal/ogamed/... -run TestSendFleet -v` | ❌ Wave 0 |
| SAFE-02 | Recall fleet via cancel endpoint | unit (mock) | `go test ./internal/ogamed/... -run TestCancelFleet -v` | ❌ Wave 0 |
| SAFE-03 | Moon-to-planet deploy uses correct type | unit | `go test ./internal/defender/... -run TestMoonEscape -v` | ❌ Wave 0 |
| SAFE-03 | Moon fleets prefer moon-safe destinations | unit | `go test ./internal/defender/... -run TestMoonSafetyScoring -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/defender/... ./internal/ogamed/... -count=1 -short`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before verify

### Wave 0 Gaps
- [ ] `internal/defender/defender_test.go` — covers SAFE-01 attack detection, filtering, poll timing
- [ ] `internal/defender/escape_test.go` — covers SAFE-02/SAFE-03 escape route calculation, fuel, safety scoring
- [ ] `internal/ogamed/client_test.go` — extend with SendFleet, CancelFleet, GetAttacks tests
- [ ] `internal/state/migrations/002_fleet_save.sql` — fleet_save_events table

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | ogamed handles auth — bot inherits |
| V3 Session Management | no | ogamed manages sessions |
| V4 Access Control | no | Single-user bot |
| V5 Input Validation | yes | Validate attack event data, fleet parameters before sending |
| V6 Cryptography | no | No crypto operations in bot layer |

### Known Threat Patterns for OGame Fleet-Save

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Attacker times fleet to arrive seconds after recalled deploy returns (phalanx snipe) | Tampering | Recall timing must account for return flight time; ensure fleet returns after attack window |
| Attacker sends multiple small attacks to trigger repeated fleet-saves (fuel exhaustion) | Denial of Service | Track recent fleet-saves per planet; don't re-save if fleet already deployed |
| Bot sends fleet to wrong coordinates due to malformed attack event data | Tampering | Validate all coordinate data before fleet dispatch |
| Bot exhausts deuterium on repeated fleet-saves, can't save when real attack comes | Denial of Service | Reserve minimum fuel for one fleet-save; track deuterium budget |

## Sources

### Primary (HIGH confidence)
- ogamed GitHub wiki — full REST API documentation for send-fleet, cancel, attacks endpoints [CITED: https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation]
- ogamed source code — `pkg/wrapper/handlers.go` SendFleetHandler, CancelFleetHandler, GetAttacksHandler implementations [CITED: https://github.com/alaingilbert/ogame/blob/master/pkg/wrapper/handlers.go]
- ogamed source code — `pkg/ogame/attackEvent.go` AttackEvent struct definition [CITED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/attackEvent.go]
- ogamed source code — `pkg/ogame/fleet.go` Fleet.IsCancellable() method [CITED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/fleet.go]
- ogamed source code — `pkg/ogame/slots.go` Slots struct [CITED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/slots.go]
- PITFALLS.md — cross-referenced TBot issues, Cruiser source, community warnings [CITED: .planning/research/PITFALLS.md]

### Secondary (MEDIUM confidence)
- Cruiser bot source — proven fleet-save implementation patterns [CITED: https://github.com/kweimann/cruiser/blob/master/bot/bot.py]
- Phase 1 summaries — existing types, client, state manager interfaces [CITED: .planning/phases/01-core-infrastructure/01-*-SUMMARY.md]
- Existing codebase — model/types.go, ogamed/client.go, state/manager.go, config/config.go [VERIFIED: read in this session]

### Tertiary (LOW confidence)
- OGame distance/flight formulas — [ASSUMED] based on community knowledge and Cruiser's engine implementation
- Coordinate type numeric mapping (1=planet, 2=debris, 3=moon) — [ASSUMED] from ogamed source inference

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, extending existing infrastructure
- Architecture: HIGH — follows Cruiser's proven patterns, integrates cleanly with Phase 1 code
- Pitfalls: HIGH — documented in PITFALLS.md with source citations, verified against ogamed source
- Flight formulas: MEDIUM — [ASSUMED] based on community knowledge, needs validation against running ogamed

**Research date:** 2026-04-26
**Valid until:** 2026-05-26 (stable — ogamed API changes are rare)
