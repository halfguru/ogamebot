# Requirements: OGame Bot

**Defined:** 2026-04-25
**Core Value:** The bot must reliably protect your fleet and grow your empire while you're away — if fleet-save fails, everything else is pointless.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Infrastructure

- [ ] **INFRA-01**: Bot connects to ogamed REST API and maintains session across restarts — *ogamed client built in 01-02*
- [ ] **INFRA-02**: Bot retrieves and caches game state (planets, resources, fleets, buildings, research) — *types/schemas/constants scaffolded in 01-01*
- [x] **INFRA-03**: Bot loads configuration from YAML/JSON file with feature toggles and per-feature parameters ✓ 01-02
- [x] **INFRA-04**: Bot implements request throttling with random intervals between actions ✓ 01-02
- [ ] **INFRA-05**: Bot runs as a Docker Compose stack (ogamed + bot) with environment-based config

### Safety

- [ ] **SAFE-01**: Bot monitors for incoming attacks by polling hostile fleet events at randomized intervals
- [ ] **SAFE-02**: Bot auto-saves fleet and resources when attack is detected using phalanx-safe deploy + recall
- [ ] **SAFE-03**: Bot handles fleet-save for moons separately with appropriate escape destinations

### Growth

- [ ] **GROW-01**: Bot calculates ROI (production increase / build cost) for every upgradeable building across all planets
- [ ] **GROW-02**: Bot automatically queues the most profitable building upgrade based on ROI calculation
- [ ] **GROW-03**: Bot respects configurable max-level caps per building type per planet

### Combat

- [ ] **COMB-01**: Bot scans configurable galaxy/system ranges for inactive players
- [ ] **COMB-02**: Bot sends espionage probes to inactive players and parses spy reports for resources and defense
- [ ] **COMB-03**: Bot attacks targets when estimated loot exceeds configurable profit threshold

### Monitoring

- [ ] **MON-01**: Web dashboard shows real-time empire overview (planets, resources, fleet movements)
- [ ] **MON-02**: Web dashboard shows build queues, recent bot actions, and event logs
- [ ] **MON-03**: Web dashboard updates in real-time via WebSocket connection

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Notifications

- **NOTF-01**: Bot sends Telegram notifications on attack detection
- **NOTF-02**: Bot sends Telegram notifications on fleet-save execution
- **NOTF-03**: Bot sends Telegram notifications on errors and warnings
- **NOTF-04**: Bot supports Telegram commands for remote control (/ghost, /deploy, /build, /sleep, /getinfo)

### Growth Enhancements

- **GROW-04**: Bot manages expedition slots, optimizes fleet composition, and auto-resends expeditions
- **GROW-05**: Bot automates technology research with configurable target levels per tech
- **GROW-06**: Bot consolidates resources to a configurable hub planet (auto-repatriate)
- **GROW-07**: Bot automatically builds transport ships when cargo capacity is insufficient

### Operational

- **OPER-01**: Bot supports multiple accounts with isolated state and config
- **OPER-02**: Bot applies config changes without restart (hot-reload)
- **OPER-03**: Bot enters sleep mode during configurable hours with pre-sleep fleet-save

## Out of Scope

| Feature | Reason |
|---------|--------|
| Browser proxy (play through bot) | Bot unaware of manual actions → conflicts; massive game-state sync complexity |
| Marketplace automation | Niche, rules change frequently, high ban risk from monitoring |
| Combat simulator | Excellent external tools exist (speedsim, trashsim); building one is a separate project |
| Auction automation | Extremely niche, unpredictable timing, bid wars create suspicious patterns |
| Message attacker | Alerts attackers that you're a bot user; draws attention |
| Lifeform automation | New feature with frequent balance changes, massive config surface |
| SMS notifications | Telegram is free, more capable (rich messages, commands), industry standard |
| Mobile app | Web dashboard is mobile-responsive; native app is a separate project |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFRA-01 | Phase 1 | Partial (client built) |
| INFRA-02 | Phase 1 | Partial (types/schemas ready) |
| INFRA-03 | Phase 1 | ✓ 01-02 |
| INFRA-04 | Phase 1 | ✓ 01-02 |
| INFRA-05 | Phase 1 | Pending |
| SAFE-01 | Phase 2 | Pending |
| SAFE-02 | Phase 2 | Pending |
| SAFE-03 | Phase 2 | Pending |
| GROW-01 | Phase 3 | Pending |
| GROW-02 | Phase 3 | Pending |
| GROW-03 | Phase 3 | Pending |
| COMB-01 | Phase 4 | Pending |
| COMB-02 | Phase 4 | Pending |
| COMB-03 | Phase 4 | Pending |
| MON-01 | Phase 5 | Pending |
| MON-02 | Phase 5 | Pending |
| MON-03 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 17 total
- Mapped to phases: 17
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-25*
*Last updated: 2026-04-26 after 01-02 completion*
