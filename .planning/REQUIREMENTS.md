# Requirements: OGameX Bot

**Defined:** 2026-05-03
**Core Value:** The bot must reliably protect your fleet and grow your empire while you're away — if fleet-save fails, everything else is pointless.

## v1 Requirements

### Infrastructure

- [ ] **INFRA-01**: Bot authenticates with OGameX via email/password, maintains session cookies and CSRF tokens across requests
- [ ] **INFRA-02**: Bot automatically refreshes expired CSRF tokens (from `newAjaxToken` in responses) and re-authenticates on session expiry (401/redirect)
- [ ] **INFRA-03**: Bot loads all configuration from YAML file including OGameX URL, credentials, feature toggles, and per-feature parameters
- [ ] **INFRA-04**: Bot runs as a single Go binary on Windows with no external dependencies

### Game State

- [ ] **STATE-01**: Bot retrieves and caches current planet list (coordinates, names, planet vs moon) from OGameX
- [ ] **STATE-02**: Bot retrieves current resources (metal, crystal, deuterium, energy) per planet via AJAX endpoints
- [ ] **STATE-03**: Bot retrieves building levels per planet (mines, facilities, moon buildings)
- [ ] **STATE-04**: Bot retrieves current research levels for the player
- [ ] **STATE-05**: Bot retrieves fleet movements (own fleets and hostile incoming) via fleet event endpoints
- [ ] **STATE-06**: Bot retrieves ship counts per planet
- [ ] **STATE-07**: Bot caches all game state in SQLite with periodic refresh, providing a single source of truth

### Fleet Safety

- [ ] **SAFE-01**: Bot detects incoming hostile fleets within a configurable polling interval via fleet event endpoints
- [ ] **SAFE-02**: Bot automatically saves fleet + resources before attack lands using deploy-with-recall mission (phalanx-safe)
- [ ] **SAFE-03**: Bot handles moon-based fleets with appropriate escape destinations
- [ ] **SAFE-04**: Bot recalls saved fleet after attack passes

### Auto-Build

- [ ] **BUILD-01**: Bot calculates ROI (production increase / build cost) for every upgradeable building across all planets
- [ ] **BUILD-02**: Bot automatically queues the highest-ROI building upgrade when a build slot is free
- [ ] **BUILD-03**: Bot respects configured max-level caps per building type
- [ ] **BUILD-04**: Bot can start research when research lab is idle and no building needs the slot

### Auto-Farm

- [ ] **FARM-01**: Bot scans configured galaxy/system ranges and identifies inactive players
- [ ] **FARM-02**: Bot sends espionage probes to inactives and parses spy reports for resources and defense
- [ ] **FARM-03**: Bot dispatches attacks when estimated loot exceeds configurable profit threshold

### Dashboard

- [ ] **DASH-01**: Dashboard displays real-time empire overview with planets, resources, and fleet movements
- [ ] **DASH-02**: Dashboard shows build queues, recent bot actions, and event logs
- [ ] **DASH-03**: Dashboard updates in real-time via WebSocket without manual page refresh

## v2 Requirements

### Expeditions

- **EXPD-01**: Bot manages expedition slots and auto-sends expeditions with optimal fleet composition
- **EXPD-02**: Bot auto-resends expeditions when they return

### Notifications

- **NOTF-01**: Bot sends Telegram notifications for attack alerts and build completions
- **NOTF-02**: Bot sends daily status summary via Telegram

### Advanced

- **ADVN-01**: Multi-account support (multiple OGameX users)
- **ADVN-02**: Auto-shipyard and auto-defense building
- **ADVN-03**: Auto-colonize new planets
- **ADVN-04**: Config hot-reload without bot restart
- **ADVN-05**: Sleep mode (reduce activity during specific hours)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Official OGame support | Gameforge anti-bot is insurmountable |
| Browser-based UI proxy | Complexity without core value |
| Combat simulator | Use existing tools |
| Marketplace automation | Niche feature, OGameX may not have it |
| Lifeform-specific logic | OGameX targets pre-Lifeform OGame |
| Mobile app | Web dashboard is mobile-responsive |
| Adding REST API to OGameX | Bot works with existing AJAX endpoints |
| Alliance management | Not core bot functionality |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFRA-01 | Phase 1 | Pending |
| INFRA-02 | Phase 1 | Pending |
| INFRA-03 | Phase 1 | Pending |
| INFRA-04 | Phase 1 | Pending |
| STATE-01 | Phase 2 | Pending |
| STATE-02 | Phase 2 | Pending |
| STATE-03 | Phase 2 | Pending |
| STATE-04 | Phase 2 | Pending |
| STATE-05 | Phase 2 | Pending |
| STATE-06 | Phase 2 | Pending |
| STATE-07 | Phase 2 | Pending |
| SAFE-01 | Phase 3 | Pending |
| SAFE-02 | Phase 3 | Pending |
| SAFE-03 | Phase 3 | Pending |
| SAFE-04 | Phase 3 | Pending |
| BUILD-01 | Phase 4 | Pending |
| BUILD-02 | Phase 4 | Pending |
| BUILD-03 | Phase 4 | Pending |
| BUILD-04 | Phase 4 | Pending |
| FARM-01 | Phase 5 | Pending |
| FARM-02 | Phase 5 | Pending |
| FARM-03 | Phase 5 | Pending |
| DASH-01 | Phase 6 | Pending |
| DASH-02 | Phase 6 | Pending |
| DASH-03 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 25 total
- Mapped to phases: 25
- Unmapped: 0 ✓

---
*Requirements defined: 2026-05-03*
*Last updated: 2026-05-03 after initial definition*
