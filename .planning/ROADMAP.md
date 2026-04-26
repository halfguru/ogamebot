# Roadmap: OGame Bot

## Overview

Build an OGame automation bot that protects your fleet and grows your empire. Start with infrastructure and fleet safety (the core value), then layer on empire growth, combat automation, and finally a web dashboard for monitoring. Safety before growth, simple before complex, each phase delivers a verifiable capability.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Core Infrastructure** - Connect to OGame via ogamed, maintain game state, throttling, config, Docker
- [ ] **Phase 2: Fleet Safety** - Detect incoming attacks and auto-save fleet with phalanx-safe deploy+recall
- [ ] **Phase 3: Auto-Build** - ROI-based automatic building upgrades across all planets
- [ ] **Phase 4: Auto-Farm** - Scan galaxy, spy inactives, attack profitable targets
- [ ] **Phase 5: Web Dashboard** - Real-time empire overview, build queues, fleet movements, activity logs

## Phase Details

### Phase 1: Core Infrastructure
**Goal**: Bot connects to OGame via ogamed and maintains reliable, throttled game state access
**Depends on**: Nothing (first phase)
**Requirements**: INFRA-01, INFRA-02, INFRA-03, INFRA-04, INFRA-05
**Success Criteria** (what must be TRUE):
  1. Bot authenticates with ogamed and survives restarts without manual re-login
  2. Bot caches and exposes current game state (planets, resources, fleets, buildings, research) from a single source of truth
  3. Bot loads all configuration from a YAML/JSON file, including feature toggles and per-feature parameters
  4. Bot spaces all API calls with randomized intervals — no two requests fire within the same second
  5. Bot runs via `docker compose up` with both ogamed and bot containers connected and communicating
**Plans**: 3 plans in 3 waves

Plans:
- [x] 01-01-PLAN.md — Monorepo scaffolding + shared types, schemas, and constants ✓ 2026-04-26
- [ ] 01-02-PLAN.md — Config loader, logger, ogamed client with rate limiting and retry
- [ ] 01-03-PLAN.md — Database schema, game state manager, main entry point, Docker Compose

### Phase 2: Fleet Safety
**Goal**: Bot reliably detects incoming attacks and auto-saves fleet and resources before impact
**Depends on**: Phase 1
**Requirements**: SAFE-01, SAFE-02, SAFE-03
**Success Criteria** (what must be TRUE):
  1. Bot detects incoming hostile fleets within a configurable polling interval
  2. Bot automatically deploys fleet + resources on a phalanx-safe mission (deploy with recall) before the attack lands
  3. Bot handles moon-based fleets with appropriate escape destinations and mission types
**Plans**: TBD

Plans:
- [ ] 02-01: TBD
- [ ] 02-02: TBD

### Phase 3: Auto-Build
**Goal**: Bot automatically grows the empire by upgrading the most profitable buildings across all planets
**Depends on**: Phase 2
**Requirements**: GROW-01, GROW-02, GROW-03
**Success Criteria** (what must be TRUE):
  1. Bot calculates ROI (production increase / build cost) for every upgradeable building across all planets
  2. Bot automatically queues the highest-ROI building upgrade when a build slot is free
  3. Bot never upgrades a building beyond its configured max-level cap
**Plans**: TBD

Plans:
- [ ] 03-01: TBD
- [ ] 03-02: TBD

### Phase 4: Auto-Farm
**Goal**: Bot automatically discovers and raids inactive players for resources
**Depends on**: Phase 3
**Requirements**: COMB-01, COMB-02, COMB-03
**Success Criteria** (what must be TRUE):
  1. Bot scans configured galaxy/system ranges and identifies inactive players
  2. Bot sends espionage probes to inactives and parses spy reports for resources and defense counts
  3. Bot dispatches attacks only when estimated loot exceeds the configurable profit threshold
**Plans**: TBD

Plans:
- [ ] 04-01: TBD
- [ ] 04-02: TBD

### Phase 5: Web Dashboard
**Goal**: Users can monitor their empire and bot activity from any device through a web interface
**Depends on**: Phase 4
**Requirements**: MON-01, MON-02, MON-03
**Success Criteria** (what must be TRUE):
  1. Dashboard displays real-time empire overview with planets, resources, and fleet movements
  2. Dashboard shows build queues, recent bot actions, and event logs
  3. Dashboard updates in real-time via WebSocket without manual page refresh
**Plans**: TBD
**UI hint**: yes

Plans:
- [ ] 05-01: TBD
- [ ] 05-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core Infrastructure | 1/3 | In progress | 2026-04-26 |
| 2. Fleet Safety | 0/? | Not started | - |
| 3. Auto-Build | 0/? | Not started | - |
| 4. Auto-Farm | 0/? | Not started | - |
| 5. Web Dashboard | 0/? | Not started | - |
