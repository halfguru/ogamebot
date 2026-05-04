# Features Research: OGameX Bot

**Date:** 2026-05-03
**Context:** OGameX bot (open-source OGame clone). Go bot with defender, builder, farmer, and dashboard workers behind a `ClientInterface` abstraction.

---

## Table Stakes Features

Features every OGame bot must have. Without these, users will leave for TBot or manual play.

### 1. Session Management (Complexity: Low)
- Login to OGameX via Laravel Fortify (email/password → session cookie + CSRF token)
- Session keepalive (periodic requests to prevent expiry)
- CSRF token extraction and rotation
- Session recovery after network errors or server restarts
- **Depends on:** Nothing (foundation)
- **Existing:** OGameX client handles session auth directly via HTTP

### 2. Game State Retrieval (Complexity: Medium)
- Fetch planets, resources, fleets, buildings, facilities, ships, defense, research
- Fetch fleet movements (own + hostile via event list)
- Fetch galaxy/system data
- Fetch messages (espionage reports)
- Cache in SQLite for offline access and dashboard queries
- **Depends on:** Session management
- **Existing:** Full state manager + SQLite cache. Client methods need OGameX endpoint mapping

### 3. Fleet-Save / Defender (Complexity: High)
- Poll for incoming attacks at randomized intervals
- Detect hostile fleets targeting own planets/moons
- Calculate escape route (deploy mission to safe destination)
- Dispatch fleet + resources before attack lands
- Recall fleet after danger passes (phalanx-safe deploy+recall)
- Handle moon-based fleets separately
- **Depends on:** Game state, fleet dispatch
- **Existing:** Complete defender worker with escape route calculator. Port to OGameX endpoints only

### 4. Auto-Build / Builder (Complexity: Medium)
- ROI calculation for every upgradeable building across all planets
- Queue highest-ROI upgrade when build slot is free
- Respect configurable max-level caps per building per planet
- Handle insufficient resources (skip, check again later)
- **Depends on:** Game state, building construction API
- **Existing:** Complete builder worker with ROI calculator. Port to OGameX endpoints only

### 5. Auto-Farm / Farmer (Complexity: Medium-High)
- Scan configurable galaxy/system ranges for targets
- Identify inactive players via galaxy data
- Send espionage probes and parse reports
- Calculate loot estimate vs fleet cost
- Attack when profit exceeds configurable threshold
- Track previous targets to avoid re-spying too frequently
- **Depends on:** Game state, fleet dispatch, espionage API
- **Existing:** Complete farmer worker. Port to OGameX endpoints only

### 6. Configuration (Complexity: Low)
- YAML config with feature toggles and per-feature parameters
- Environment variable interpolation for secrets
- Validation with sensible defaults
- **Depends on:** Nothing (loaded at startup)
- **Existing:** Complete config system with validation

### 7. Basic Logging & Status (Complexity: Low)
- Structured logging (slog) for all bot actions
- Fleet-save events, build actions, farm results, errors
- Status endpoint for health checks
- **Depends on:** Nothing
- **Existing:** Logging throughout all workers

---

## Differentiating Features

Features that set this bot apart from TBot (the dominant competitor). Ordered by impact.

### 8. Web Dashboard with Real-Time Updates (Complexity: High)
- SolidJS SPA with real-time empire overview
- Planet resources, fleet movements, build queues
- Activity feed with bot action logs
- WebSocket push (no polling from browser)
- Mobile-responsive (works on phone without native app)
- **Depends on:** Game state, logging
- **Existing:** Complete dashboard with Go REST+WebSocket server and SolidJS frontend
- **Why differentiating:** TBot has a WebUI but it's server-rendered Blazor; ours is purpose-built with real-time updates

### 9. Auto-Expeditions (Complexity: Medium-High)
- Manage expedition fleet slots (max slots = research level)
- Optimize fleet composition for max find chance
- Auto-resend when expedition returns
- Track expedition results (finds, losses, delays)
- **Depends on:** Fleet dispatch, fleet recall, game state
- **Existing:** Not yet built
- **Why differentiating:** Critical for mid/late-game growth; TBot has this but many bots don't

### 10. Auto-Research (Complexity: Medium)
- Automate technology research with configurable target levels per tech
- Research is account-wide (one lab at a time) — queue management
- Priority ordering (e.g., combustion before impulse)
- **Depends on:** Game state, research API (`POST /research/add-buildrequest`)
- **Existing:** Not yet built
- **Why differentiating:** Natural extension of auto-build; ROI logic can be shared

### 11. Resource Consolidation / Auto-Repatriate (Complexity: Medium)
- Periodically move all resources to a hub planet
- Build transport ships automatically when cargo capacity insufficient
- Leave configurable deuterium reserve on source planets
- **Depends on:** Fleet dispatch, shipyard API, game state
- **Existing:** Not yet built
- **Why differentiating:** Enables concentrated building/research; pairs with auto-build and auto-research

### 12. Telegram Notifications + Remote Control (Complexity: Medium)
- Push alerts on: attack detection, fleet-save execution, errors
- Commands: /status, /ghost, /deploy, /build, /sleep, /stop, /start
- Configurable per-feature notification preferences
- **Depends on:** Fleet-save, auto-build, auto-farm (for events to notify about)
- **Existing:** Not yet built
- **Why differentiating:** TBot has this; table stakes for power users

### 13. Multi-Account Support (Complexity: Medium)
- Run multiple OGameX accounts from one bot instance
- Isolated state (SQLite per account) and config
- Shared rate limiter to avoid overwhelming server
- **Depends on:** Configuration, session management (multiple sessions)
- **Existing:** Config structure supports it; state manager is per-account
- **Why differentiating:** TBot has this; essential for multi-universe players

### 14. Sleep Mode with Fleet-Save (Complexity: Medium)
- Bot goes inactive during configurable hours
- Pre-sleep fleet-save (deploy+recall all fleets)
- Wake up and resume normal operations
- **Depends on:** Fleet-save, configuration
- **Existing:** Not yet built
- **Why differentiating:** Mimics human play patterns; reduces server load during off-hours

### 15. Auto-Shipyard / Defense Building (Complexity: Low-Medium)
- Auto-build ships (transports, combat ships) based on configurable targets
- Auto-build defense (plasma turrets, shields) based on defense ratios
- Queue management within shipyard constraints
- **Depends on:** Game state, shipyard API (`POST /shipyard/add-buildrequest`, `POST /defense/add-buildrequest`)
- **Existing:** Not yet built
- **Why differentiating:** Completes the "auto-grow" story alongside buildings and research

### 16. Config Hot-Reload (Complexity: Low)
- Watch config file for changes; apply without restart
- Enable/disable features dynamically
- Update parameters (galaxy ranges, thresholds) on the fly
- **Depends on:** Configuration
- **Existing:** Not yet built
- **Why differentiating:** TBot has this; important for remote management

### 17. Auto-Colonize (Complexity: Low-Medium)
- Send colony ships to configured coordinates
- Manage colony ship production
- Abandon colonies below configurable size threshold, retry
- **Depends on:** Fleet dispatch, shipyard API
- **Existing:** Not yet built
- **Why differentiating:** TBot has this; useful for early-game expansion

---

## OGameX-Specific Opportunities

Things possible with OGameX that are impossible or impractical with official OGame. This is the most interesting category — it's where the bot can do things no official-OGame bot can.

### 18. Zero Anti-Bot Evasion (Complexity: Eliminated)
- No captcha, no fingerprinting, no behavioral analysis
- No residential proxy requirement
- No request-pattern obfuscation needed
- Rate limiting is basic (login throttle only)
- **Impact:** Massive simplification. No anti-bot subsystem needed. Faster development, fewer failure modes, more reliable operation.

### 19. Direct API Access (No Middleware) (Complexity: Medium rewrite)
- OGameX exposes JSON AJAX endpoints directly
- Bot talks HTTP to OGameX directly (session + CSRF)
- Single Go binary deployment
- **Impact:** Simpler deployment, fewer moving parts, full control over API layer
- **Existing:** OGameX client communicates directly via HTTP

### 20. Self-Hosted Server Control (Complexity: Low, if self-hosting)
- If running own OGameX instance: access to admin panel, database
- Can query MySQL directly for game state (bypass HTTP entirely)
- Can modify server settings (speed, debris ratios, fleet speed)
- Can use Laravel artisan commands for diagnostics
- **Impact:** Game-changing for self-hosted OGameX servers. Bot becomes admin tool.
- **Note:** Not applicable to main.ogamex.dev (shared demo server)

### 21. Open-Source Collaboration (Complexity: Varies)
- Can contribute new API endpoints to OGameX upstream
- Request bot-friendly endpoints (e.g., structured game state API)
- Submit PRs for missing features needed by bot
- **Impact:** Bot and server co-evolve. Can shape the API surface to be bot-friendly.

### 22. No Lifeforms Complexity (Complexity: Eliminated)
- OGameX explicitly targets pre-Lifeforms OGame (pre-2022)
- No Lifeform buildings, research, or population mechanics
- Simpler building/research trees
- **Impact:** Significant scope reduction. TBot has 4 Lifeform-related features we can ignore entirely.

### 23. Predictable Game Mechanics (Complexity: Reduced)
- OGameX is open-source — all formulas are auditable in PHP/Rust code
- No secret Gameforge changes or A/B tests
- Combat engine is Rust-based and deterministic
- Can pre-calculate exact outcomes (no need for external combat sim)
- **Impact:** Can build precise ROI calculators, combat predictors, and fleet optimizers with confidence.

---

## Anti-Features (Deliberately NOT Building)

| Feature | Reason |
|---------|--------|
| Browser proxy (play through bot) | Bot unaware of manual actions → state conflicts; massive sync complexity; not core value |
| Combat simulator | External OGameX combat sim exists (ogamex-combat-simulator on GitHub); separate project |
| Marketplace automation | OGameX doesn't have marketplace; niche feature even in official OGame |
| Lifeform automation | OGameX targets pre-Lifeforms OGame; not applicable |
| Mobile app | Web dashboard is mobile-responsive; native app is a separate project |
| SMS notifications | Telegram is free, richer (commands, rich messages), industry standard |
| Message attacker | Alerts attackers; no anti-bot concern on OGameX but still antisocial; draws admin attention |
| Auction automation | OGameX doesn't have auctions; niche feature |
| Alliance management | OGameX alliances not yet implemented; defer until available upstream |

---

## Feature Dependencies

```
Session Management ──────────────────────┐
  │                                       │
  ▼                                       │
Game State Retrieval ────────────────────┤
  │                                       │
  ├──► Fleet-Save / Defender ────────────┤
  │      │                                │
  │      └──► Sleep Mode                  │
  │                                       │
  ├──► Auto-Build / Builder               │
  │      │                                │
  │      └──► Auto-Research               │
  │                                       │
  ├──► Auto-Farm / Farmer                 │
  │                                       │
  ├──► Auto-Expeditions ─────────────────┤
  │                                       │
  ├──► Resource Consolidation             │
  │      │                                │
  │      └──► Auto-Shipyard               │
  │                                       │
  ├──► Auto-Colonize                      │
  │                                       │
  └──► Web Dashboard (reads all state)    │
                                         │
Configuration ───────────────────────────┤
  └──► Config Hot-Reload                 │
                                         │
Telegram Notifications (needs events ────┘
  from: Defender, Builder, Farmer, Expos)
```

### Critical Path (v1 for OGameX pivot)
1. **OGameX Client** — direct HTTP to OGameX (session auth + CSRF)
2. **Session Management** — login, CSRF, keepalive
3. **Game State** — map OGameX AJAX endpoints to domain types
4. **Fleet-Save** — highest priority (core value: protect the fleet)
5. **Auto-Build** — second priority (grow the empire)
6. **Auto-Farm** — third priority (resource income)

### Natural Extensions (v2)
7. Auto-Expeditions
8. Auto-Research
9. Resource Consolidation
10. Telegram Notifications
11. Auto-Shipyard/Defense

### Nice-to-Have (v3)
12. Multi-Account
13. Sleep Mode
14. Auto-Colonize
15. Config Hot-Reload

### OGameX-Specific (when applicable)
16. Self-hosted server integration (if running own OGameX)
17. Upstream API contributions (ongoing)

---

## Competitive Comparison

| Feature | This Bot | TBot | Cruiser |
|---------|----------|------|---------|
| Target | OGameX | Official OGame | Official OGame |
| Language | Go | C#/.NET | Node.js |
| Fleet-Save | Smart (deploy+recall) | Smart (deploy+recall) | Smart (phalanx-safe) |
| Auto-Build | ROI-based | ROI-based | Basic |
| Auto-Farm | Yes | Yes | No |
| Auto-Expeditions | Planned | Yes | No |
| Dashboard | SolidJS + WebSocket | Blazor WebUI | None |
| Telegram | Planned | Yes | No |
| Anti-Bot Evasion | None needed | Captcha solver, proxy | Captcha solver |
| Deployment | Single binary | .NET runtime | Node runtime |
| Self-Host Server Control | Possible | No | No |
| Open Source | Yes | Yes | Yes |

**Key advantage over TBot:** No anti-bot cat-and-mouse game. Simpler deployment (single binary vs .NET runtime). Can evolve with OGameX upstream.

---

*Research completed: 2026-05-03*
