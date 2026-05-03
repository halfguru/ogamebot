# Roadmap: OGameX Bot

**Created:** 2026-05-03
**Granularity:** coarse (6 phases, 18 plans)
**Strategy:** Safety-first — fleet-save works before growth features ship

## Phase Overview

```
Phase 1: Core Infrastructure  ████████░░░░░░░░░░░░  INFRA-01..04
Phase 2: Game State           ████████████░░░░░░░░  STATE-01..07
Phase 3: Fleet Safety         ████████████████░░░░  SAFE-01..04   ← MILESTONE: bot protects fleet
Phase 4: Auto-Build           ██████████████████░░  BUILD-01..04
Phase 5: Auto-Farm            ████████████████████  FARM-01..03
Phase 6: Web Dashboard        ████████████████████  DASH-01..03
```

---

## Phase 1: Core Infrastructure — OGameX Client

**Goal:** Replace `internal/ogamed/` with `internal/ogamex/` that authenticates with OGameX via Laravel session login, maintains CSRF tokens, and loads configuration for the new target. Bot compiles and login succeeds.

**Dependencies:** None (first phase)
**Requirements:** INFRA-01, INFRA-02, INFRA-03, INFRA-04

**Plans:** 2 plans

Plans:
- [ ] 01-01-PLAN.md — Config + OGameX client skeleton (INFRA-03, INFRA-04)
- [ ] 01-02-PLAN.md — Login + session + CSRF management (INFRA-01, INFRA-02)

**Success Criteria:**
1. `go build ./...` succeeds with new `internal/ogamex/` package alongside `internal/ogamed/`
2. Bot logs into `main.ogamex.dev` with email/password and receives session cookie
3. CSRF token is extracted from login response HTML and sent with subsequent AJAX requests
4. Bot auto-refreshes CSRF token from `newAjaxToken` in JSON responses (thread-safe)
5. Bot re-authenticates on 401/session expiry without crashing

---

## Phase 2: Game State — Planet List, Resources, Buildings, Fleets

**Goal:** Implement all read-only `ClientInterface` methods so `state.Manager` can hydrate SQLite with live OGameX data. Cached state matches what the HTML/JSON pages show.

**Dependencies:** Phase 1 (authenticated client)
**Requirements:** STATE-01, STATE-02, STATE-03, STATE-04, STATE-05, STATE-06, STATE-07

**Plans:** 2 plans

Plans:
- [ ] 02-01-PLAN.md — Parser infrastructure + per-planet state methods (STATE-01, STATE-02, STATE-03, STATE-06)
- [ ] 02-02-PLAN.md — Fleet events + research + constructions + server info (STATE-04, STATE-05, STATE-07)

**Success Criteria:**
1. `GetPlanets()` returns correct planet list with coordinates, names, and moon status
2. `GetResources()` returns live metal/crystal/deuterium/energy per planet
3. `GetFleets()` returns all fleet movements (own + visible hostile) with correct mission types and arrival times
4. `GetResearch()` returns all technology levels
5. State manager runs one full refresh cycle without errors and SQLite contains valid data

---

## Phase 3: Fleet Safety — Defender Worker on New Client

**Goal:** Wire the existing `internal/defender/` worker to the new OGameX client. Attack detection, fleet-save execution, and recall all work end-to-end. This is the critical milestone — the bot protects the fleet.

**Dependencies:** Phase 2 (game state retrieval)
**Requirements:** SAFE-01, SAFE-02, SAFE-03, SAFE-04

**Success Criteria:**
1. Defender detects incoming hostile attack (mission type 1) within configured poll interval
2. Defender saves fleet + resources to a safe destination before attack lands
3. Deploy-with-recall (phalanx-safe) works: fleet deploys, then recalls after attack passes
4. Moon-based fleets escape to appropriate destinations
5. Recall executes after attack passes and fleet returns safely

### Plan 3.1 — Fleet dispatch + cancel on OGameX (SAFE-02, SAFE-03, SAFE-04)

**Scope:**
- Implement `SendFleet()`: OGameX fleet dispatch is two-step:
  1. POST `/ajax/fleet/dispatch/check-target` with fleet composition + destination → get mission options
  2. POST `/ajax/fleet/dispatch/send-fleet` with mission + speed + resources → dispatch
- Implement `CancelFleet()`: POST fleet cancel endpoint with fleet ID
- Map OGameX fleet dispatch request/response to existing `model.SendFleetRequest`
- Handle planet type parameter (planet=1, moon=3) for destinations
- Handle CSRF token in both check-target and send-fleet requests
- Test: manually trigger a deploy mission and cancel it

### Plan 3.2 — Wire defender + attack detection + recall (SAFE-01, SAFE-02, SAFE-04)

**Scope:**
- Verify `GetAttacks()` returns correctly parsed hostile fleet events with arrival times
- Wire `defender.NewDefender(ogamexClient, stateMgr, db, cfg, log)` — only import path changes
- Verify escape route calculator works with OGameX coordinate format
- Test fleet-save end-to-end: simulate incoming attack → defender saves fleet → recall after danger passes
- Verify fleet-save events tracked in SQLite (`fleet_save_events` table)
- Verify dashboard broadcasts fire on fleet-save and recall events
- **Milestone checkpoint:** bot can protect fleet autonomously

---

## Phase 4: Auto-Build — Builder Worker on New Client

**Goal:** Wire the existing `internal/builder/` worker to the new OGameX client. ROI-based auto-building and research work across all planets.

**Dependencies:** Phase 3 (defender working — safety before growth)
**Requirements:** BUILD-01, BUILD-02, BUILD-03, BUILD-04

**Success Criteria:**
1. ROI calculator produces correct scores for all mine types across all planets
2. Builder queues the highest-ROI upgrade when a build slot is free
3. Max-level caps from config are respected (global + per-planet overrides)
4. Research starts when research lab is idle and no building needs the slot

### Plan 4.1 — Build + research actions on OGameX (BUILD-02, BUILD-04)

**Scope:**
- Implement `BuildBuilding()`: POST building upgrade endpoint for a planet (extract from resources page DOM)
- Verify building IDs match OGameX's internal IDs (may differ from ogamed)
- Implement research start: POST research upgrade endpoint
- Handle OGameX response for "not enough resources" / "build slot full" gracefully
- Map OGameX building IDs to `internal/constants/buildings.go` constants if they differ

### Plan 4.2 — Wire builder + ROI validation (BUILD-01, BUILD-03)

**Scope:**
- Wire `builder.NewBuilder(ogamexClient, stateMgr, db, cfg, log)` — import path change only
- Verify `GetConstructions()` correctly reports active build queue (empty slot = ID 0)
- Verify ROI scores match manual calculation for sample planets
- Verify max-level caps enforced: no upgrades beyond configured limits
- Verify build events recorded in SQLite (`build_events` table)
- Test: let builder run for 2 cycles, verify upgrades executed correctly

---

## Phase 5: Auto-Farm — Farmer Worker on New Client

**Goal:** Wire the existing `internal/farmer/` worker to the new OGameX client. Galaxy scanning, espionage, and profitable attacks work.

**Dependencies:** Phase 4 (builder working — growth before farming)
**Requirements:** FARM-01, FARM-02, FARM-03

**Success Criteria:**
1. Galaxy scan returns inactive players in configured ranges
2. Espionage probes sent to inactives and reports parsed for resources + defense
3. Attacks dispatched only when estimated loot exceeds profit threshold

### Plan 5.1 — Galaxy scan + espionage + farming (FARM-01, FARM-02, FARM-03)

**Scope:**
- Implement `GetGalaxyInfos()`: GET galaxy AJAX endpoint for a system, parse planet positions, player names, inactive status
- Implement `GetEspionageReportMessages()`: fetch messages page, parse espionage report list
- Implement `GetEspionageReport()`: fetch individual espionage report, parse resources + defense
- Implement `DeleteAllEspionageReports()`: delete processed messages
- Wire `farmer.NewFarmer(ogamexClient, stateMgr, db, cfg, log)` — import path change only
- Verify farm target selection: inactive → spy → evaluate profit → attack if above threshold
- Verify farm attacks recorded in SQLite (`farm_attacks` table)
- Test: let farmer run one full cycle against live OGameX

---

## Phase 6: Web Dashboard — Real-Time Monitoring

**Goal:** Wire the existing SolidJS dashboard to the new bot engine. Dashboard shows live empire state, fleet movements, build queues, and bot action logs.

**Dependencies:** Phase 5 (all bot features working)
**Requirements:** DASH-01, DASH-02, DASH-03

**Success Criteria:**
1. Dashboard displays empire overview with live planet data from SQLite cache
2. Fleet movements and fleet-save events shown in real-time
3. Build events and farm attacks stream via WebSocket without page refresh

### Plan 6.1 — Dashboard API + WebSocket (DASH-01, DASH-02, DASH-03)

**Scope:**
- Verify `internal/dashboard/` handlers work with new state manager (they read from SQLite — should be transparent)
- Update any hardcoded ogamed references in dashboard TypeScript/API
- Verify WebSocket hub broadcasts fleet-save, build, and farm events
- Test dashboard loads and shows live data from OGameX-connected bot
- Verify CORS and static file serving work for development

### Plan 6.2 — Cleanup + final validation (cross-cutting)

**Scope:**
- Remove all remaining `internal/ogamed/` code and imports
- Remove `docker-compose.yml` ogamed service, `Dockerfile` for ogamed
- Update `README.md` with OGameX setup instructions
- Update `config.example.yaml` with final OGameX config structure
- Remove `cookies.json` from repo (gitignore)
- Run full integration test: login → state refresh → defender → builder → farmer → dashboard
- Verify single Go binary builds with `go build ./cmd/bot/`

---

## Phase Sequencing Rationale

| Ordering | Why |
|----------|-----|
| Infrastructure first | Everything depends on authenticated client |
| Game state before safety | Defender needs planets, resources, fleets from cache |
| Safety before growth | If fleet-save fails, everything else is pointless |
| Build before farm | Farming needs fleet slots; builder uses fewer slots |
| Dashboard last | Dashboard reads from SQLite — transparent to client swap |

## Plan Summary

| Phase | Plans | Requirements | Key Deliverable |
|-------|-------|-------------|-----------------|
| 1. Infrastructure | 2 | INFRA-01..04 | Bot logs into OGameX |
| 2. Game State | 2 | STATE-01..07 | SQLite populated with live data |
| 3. Fleet Safety | 2 | SAFE-01..04 | Bot protects fleet autonomously |
| 4. Auto-Build | 2 | BUILD-01..04 | Bot upgrades buildings by ROI |
| 5. Auto-Farm | 1 | FARM-01..03 | Bot farms inactives profitably |
| 6. Dashboard | 2 | DASH-01..03 | Live web dashboard working |
| **Total** | **11** | **25** | |

---
*Roadmap created: 2026-05-03*
*Last updated: 2026-05-03*
