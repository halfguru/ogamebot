# Project Research Summary

**Project:** OGame Bot
**Domain:** Game automation bot (OGame) with REST backend + TypeScript bot logic + web dashboard
**Researched:** 2026-04-25
**Confidence:** HIGH

## Executive Summary

This is a single-user game automation bot for OGame — a browser-based space strategy MMO. The standard architecture across all reference projects (TBot, Cruiser, r4fek) is: a Go-based REST backend (ogamed) handles anti-detection and game protocol, while a separate bot logic layer handles automation decisions. The bot runs 24/7 on a VPS, accessed remotely via Telegram notifications and a web dashboard. The critical value proposition is fleet protection — if fleet-save fails, months of player progress can be destroyed in seconds, making reliability the #1 concern above all other features.

The recommended approach is a TypeScript/Node.js monorepo (pnpm workspaces) with four packages: `shared` (types, schemas, constants), `bot` (core engine with worker pattern), `api` (Fastify web server), and `dashboard` (SolidJS frontend). SQLite via Drizzle ORM provides persistent state with zero ops overhead. The architecture follows a strict layering: ogamed REST client → game state manager (cached) → scheduler/event bus → independent feature workers → presentation layer (dashboard + Telegram). Each worker (defender, auto-build, auto-farm, expeditions) operates on scheduled ticks and competes for fleet slots through a priority-based coordinator.

The dominant risks are fleet-save failure (8 distinct failure modes identified from TBot issues), anti-detection (OGame bans detected bots, datacenter IPs are blocked), and ogamed API breakage when OGame updates. Mitigation requires: phalanx-safe deploy-with-recall as the only fleet-save pattern, aggressive request throttling with random jitter from day one, automated captcha solving, residential proxy support, and a health-check system that pauses the bot when ogamed responses look corrupted. Fleet-save must be built first and tested exhaustively before any other feature.

## Key Findings

### Recommended Stack

The stack is locked to ogamed (v53) as the only maintained OGame API wrapper — this is non-negotiable. TypeScript provides a shared language across bot logic and web dashboard, with excellent library support for HTTP clients, schedulers, and Telegram bots. SQLite is the right database for a single-user bot — synchronous API via better-sqlite3 simplifies game logic, zero ops, single-file backup.

**Core technologies:**
- **ogamed v53** — Go REST backend for all OGame operations (fleet, buildings, espionage, galaxy scanning). Only maintained wrapper. Handles device fingerprinting, captcha, anti-detection.
- **TypeScript 5.7+ / Node.js 22 LTS** — Bot logic + web server. Type safety critical for game calculations (ship IDs, coordinates, mission types). Strict mode catches ROI math bugs.
- **SQLite + Drizzle ORM** — Persistent state. Zero ops, synchronous API, single-file DB. Drizzle provides type-safe queries without Prisma's engine binary overhead.
- **Fastify 5.x** — Dashboard API server. Fastest Node.js framework, TypeScript-first, plugin system for clean separation.
- **SolidJS 1.9+** — Dashboard frontend. ~7KB bundle, fine-grained reactivity for real-time game state updates, familiar TSX syntax.
- **Telegraf 4.x** — Telegram bot. Industry standard for Node.js, supports commands and notifications.
- **node-cron** — Scheduled task execution. No Redis needed (unlike BullMQ) for single-instance bot.
- **pnpm workspaces** — Monorepo with `shared`, `bot`, `api`, `dashboard` packages.

### Expected Features

Research analyzed 3 reference projects (TBot at 94 stars, Cruiser at 30 stars, r4fek archived). Feature expectations are well-established in the OGame bot community.

**Must have (table stakes — v1):**
- Attack detection + fleet-save (phalanx-safe deploy with recall) — THE core value; failure = months of progress lost
- ROI-based auto-build — TBot's gold standard; calculates production increase/cost ratio across all planets
- Auto-expeditions — high-value passive income; slot management + auto-resend
- Telegram notifications — attack alerts, fleet-save confirmations, errors
- Configuration system — per-account JSON/YAML config with feature toggles
- Anti-detection basics — random intervals, request throttling, sleep mode

**Should have (competitive — v1.x):**
- Auto-farm — galaxy scan → spy inactives → attack if profitable
- Web dashboard — real-time empire overview, fleet movements, build queues
- Telegram remote control — command interface (30+ commands in TBot)
- Auto-repatriate — resource consolidation to feed auto-build
- Auto-research — automate tech research with target levels
- Multi-account support — separate ogamed instances per account
- Settings hot-reload — change config without restart (critical for 24/7)

**Defer (v2+):**
- Auto-colonize, auto-harvest, lifeform automation — niche or volatile features
- Marketplace/auction automation — high ban risk, low value

### Architecture Approach

The architecture is a layered system with strict boundaries: each layer only talks to the one below it. The ogamed REST client is the single point of contact with OGame — no other code makes HTTP calls. A cached game state manager provides the single source of truth for all workers, preventing redundant API calls that trigger rate limits. Workers are independent tick-based loops that communicate through an event bus and compete for fleet slots through a priority coordinator (defender = critical, fleet-save = high, expeditions = normal, auto-farm = low).

**Major components:**
1. **Ogamed REST Client** — typed wrapper over all ogamed HTTP endpoints with rate limiting and retry logic
2. **Game State Manager** — cached planet/fleet/research state with periodic refresh and event emission on changes
3. **Scheduler + Event Bus** — priority-based fleet slot coordination + cross-worker event communication
4. **Feature Workers** — independent tick-based loops: Defender (attack detection + fleet-save), AutoBuild (ROI calculations), AutoFarm (galaxy scan + espionage + attack), Expeditions (slot management + fleet composition)
5. **Presentation Layer** — Fastify API + WebSocket (dashboard), Telegraf (Telegram notifications + commands)

### Critical Pitfalls

1. **Fleet-save mission fallback failure** — TBot issue #178: bot fell through to "Colonize" at an occupied position after Deploy failed, leaving the entire fleet exposed. Validate destination per-mission, not once globally. Always have a guaranteed-safe fallback.
2. **Captcha freezes the bot** — #1 reported issue across ALL OGame bots. Without automated solving (Ninja Solver API, $0.10/solve), the bot is completely frozen and can't fleet-save. Must detect captcha state and alert via Telegram immediately.
3. **OGame updates break ogamed parsers** — ogamed works by scraping HTML; when Gameforge changes page structure, extractors break silently (ogamed issues #148, #150). Build health-check system that validates API responses and pauses on anomalies.
4. **Datacenter IP blocking** — OGame blocks known datacenter IPs (AWS, DigitalOcean, Hetzner). Residential proxy is a deployment prerequisite, not an optional step. ogamed has built-in proxy support.
5. **Phalanx-unsafe fleet-save** — Only Deploy (with recall) and Harvest missions are invisible to sensor phalanx. Transport, Attack, and other missions expose exact return times to attackers. Default to deploy-with-recall only.
6. **Request rate limiting → ban** — All features' API calls compound. Must use a global rate limiter (target ~1 req/2-3s), add ±20-40% jitter to all intervals, and track total requests across ALL features.
7. **Fuel check before fleet-save** — Must verify `availableDeuterium >= fuelCost + deuteriumToLoad`. Insufficient fuel = fleet stays home = certain death.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Foundation + Fleet Safety
**Rationale:** Fleet protection is the core value proposition and the most technically risky feature. All other features are irrelevant if fleet-save fails. The ogamed client and game state manager are dependencies for everything. Anti-detection infrastructure (rate limiting, jitter) must exist before any feature runs.
**Delivers:** Working ogamed connection, game state polling, attack detection, phalanx-safe fleet-save with recall, Telegram notifications, configuration system, rate limiter with jitter, residential proxy support.
**Addresses:** Attack detection, fleet-save, Telegram notifications, configuration, anti-detection basics (P1 features).
**Avoids:** Pitfalls 1 (fleet-save fallback), 2 (API breakage), 3 (captcha), 4 (IP blocking), 5 (phalanx-unsafe), 6 (rate limiting), 7 (fuel check).
**Architecture components:** Ogamed client, game state manager, scheduler, defender worker, notification layer, config manager.

### Phase 2: Empire Growth (Auto-Build + Expeditions)
**Rationale:** Once fleet safety is proven, the next highest-value features are auto-build (empire growth) and auto-expeditions (passive income). These are the features users activate first after trusting fleet-save. Auto-build requires OGame formula research (building costs, production rates). Expeditions are relatively simple slot management.
**Delivers:** ROI-based auto-build across all planets with max-level caps. Auto-expedition with slot management, fleet composition optimization, and auto-resend.
**Uses:** Game state manager (cached planet data for ROI calc), fleet slot coordinator (expeditions compete for slots), Telegram (build/expedition notifications).
**Implements:** AutoBuild worker, Expeditions worker, ROI calculator, fleet composer.
**Avoids:** Pitfall on behavioral patterns (add sleep mode here, varied build decisions).

### Phase 3: Combat Features (Auto-Farm)
**Rationale:** Auto-farm is a multi-step pipeline (galaxy scan → spy inactives → parse reports → calculate profit → send attacks) and the most complex worker. It should only be built when the simpler workers are proven reliable. It also consumes the most fleet slots and API calls, so rate limiting must be battle-tested first.
**Delivers:** Galaxy scanning in configured ranges, espionage of inactive players, spy report analysis, profit-threshold attack dispatch.
**Uses:** Fleet slot coordinator (LOW priority), rate limiter (galaxy scans are heavy), game state (track farming history).
**Implements:** AutoFarm worker, galaxy scanner, espionage analyzer, attack planner.

### Phase 4: Monitoring + Control (Web Dashboard)
**Rationale:** The dashboard is primarily for monitoring a running bot. It's not useful until there's meaningful data to display (fleet movements, build queues, farm activity). Real-time WebSocket updates require a working event bus. SolidJS dashboard is a self-contained frontend that can be built independently once the API exists.
**Delivers:** Real-time empire overview, fleet movement table, build queue display, activity log, configuration editor. WebSocket for live updates.
**Uses:** Fastify API server, WebSocket plugin, SolidJS frontend, shared types package.
**Implements:** Dashboard backend (REST + WebSocket), Dashboard frontend (SolidJS SPA).

### Phase 5: Enhanced Features (Multi-Account + Telegram Commands + Hot-Reload)
**Rationale:** These features enhance a working bot but aren't needed for single-account operation. Multi-account adds account context isolation. Telegram commands turn notifications into a control interface. Hot-reload eliminates restart pain for 24/7 operation.
**Delivers:** Multiple account support (isolated contexts, separate ogamed instances), Telegram command interface, settings hot-reload, auto-repatriate, auto-research, auto-cargo.
**Uses:** Account manager pattern (one context per account), Telegraf command handlers, file watcher for config.
**Implements:** Account manager, Telegram command dispatcher, config watcher, additional workers.

### Phase Ordering Rationale

- **Safety before growth:** Fleet-save (Phase 1) must be rock-solid before any automation runs. A buggy auto-builder that triggers rate limits while fleet-save is broken is worse than no bot at all.
- **Dependencies flow down:** ogamed client → game state → scheduler → workers → dashboard. Each phase builds on the previous layer.
- **Complexity gradient:** Defender (check attacks, send fleet) → AutoBuild (ROI math) → AutoFarm (multi-step pipeline) → Dashboard (full web app). Simplest workers first.
- **Fleet slot contention:** The priority-based fleet slot manager (built in Phase 1) becomes critical as more workers compete for slots in later phases. Building it first means it's battle-tested by Phase 2-3.
- **Anti-detection compounds:** Rate limiting and jitter in Phase 1, sleep mode in Phase 2, behavioral variation in Phase 2-3. Anti-detection is layered in progressively rather than bolted on.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (Auto-Build):** ROI calculation requires OGame building cost formulas, production rate formulas, energy balance formulas. Need to research exact OGame wiki formulas or extract from TBot source.
- **Phase 2 (Expeditions):** Optimal fleet composition varies by account size, character class (Discoverer bonus), and lifeform bonuses. TBot's auto-calculation logic needs study.
- **Phase 3 (Auto-Farm):** Galaxy scanning patterns, spy report parsing format (ogamed response structure), loot calculation formulas. May need to test against live ogamed responses.
- **Phase 5 (Multi-Account):** ogamed multi-instance setup (different ports, cookie isolation), rate limiting across accounts sharing one IP.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** ogamed REST API is well-documented on their wiki. Worker pattern, game state caching, and fleet slot coordination are standard patterns documented in ARCHITECTURE.md.
- **Phase 4 (Dashboard):** SolidJS + Fastify + WebSocket is a well-documented stack. Dashboard is standard CRUD + real-time updates.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies verified via Context7 docs or official sources. ogamed is the only viable OGame wrapper — confirmed by analyzing 3 reference projects. Version compatibility matrix validated. |
| Features | HIGH | Feature expectations derived from detailed analysis of 3 reference bots (TBot, Cruiser, r4fek) with 173+ total stars and active user communities. Prioritization cross-referenced with TBot wiki configuration guide. |
| Architecture | HIGH | Architecture patterns extracted from TBot source structure (WorkerBase, GameState, TBotOgamedBridge), Cruiser's clean client/engine/bot separation, and ogamed REST API docs. Well-established patterns in game bot domain. |
| Pitfalls | HIGH | Pitfalls cross-referenced from 10+ TBot GitHub issues, 3+ ogamed issues, Cruiser source, and community warnings. Fleet-save failure modes (TBot #178) and captcha issues (6+ TBot issues) are well-documented real-world failures. |

**Overall confidence:** HIGH

### Gaps to Address

- **OGame formula constants:** Building cost formulas, production rate formulas, fuel consumption formulas needed for ROI calculator. Available in OGame wiki but need to be codified in TypeScript. Handle during Phase 2 planning.
- **ogamed response schemas:** Exact JSON response shapes for all REST endpoints need Zod schema definitions. ogamed wiki documents endpoints but response validation is on us. Build during Phase 1 by running ogamed locally and capturing responses.
- **Expedition fleet optimization:** TBot auto-calculates optimal expedition fleet including lifeform bonuses. This logic is complex and may need iterative refinement based on real expedition results. Start simple in Phase 2, optimize in Phase 5.
- **Residential proxy providers:** Which proxy services work reliably with OGame is operational knowledge. Document during Phase 1 deployment testing. ogamed's `loginOnly` proxy parameter may reduce costs.
- **Universe speed multiplier:** OGame universes run at different speeds (x1, x2, x4, x7). All calculations (ROI, flight time, production) must account for this. Need to query server data from OGame public API on startup.

## Sources

### Primary (HIGH confidence)
- **ogamed** — github.com/alaingilbert/ogame, v53.0.0. REST API wiki, issue tracker, release history. 3300+ commits, actively maintained.
- **TBot** — github.com/ogame-tbot/TBot, 94 stars. README, wiki configuration guide, issue tracker (10+ issues analyzed). Most complete reference implementation.
- **Cruiser** — github.com/kweimann/cruiser, 30 stars. Source code architecture, phalanx-safe fleet-save implementation, config patterns.
- **Context7: Fastify** — `/fastify/fastify`. TypeScript setup, plugin system, WebSocket support.
- **Context7: Drizzle ORM** — `/drizzle-team/drizzle-orm`. SQLite + better-sqlite3 setup, migrations.
- **Context7: SolidJS** — `/solidjs/solid`. Fine-grained reactivity, TSX syntax, routing.
- **Context7: Telegraf** — `/telegraf/telegraf`. Bot setup, commands, webhook mode.
- **Context7: pnpm** — `/websites/pnpm_io`. Workspace support, monorepo configuration.

### Secondary (MEDIUM confidence)
- **r4fek/ogame-bot** — github.com/r4fek/ogame-bot, 45 stars (archived 2018). Historical reference for anti-patterns (SMS notifications, scope creep).
- **Context7: BullMQ** — `/websites/bullmq_io`. Evaluated and rejected (requires Redis). Confirmed node-cron is sufficient.
- **Training data** — pino (logging), ofetch (HTTP client), node-cron versions. Well-established libraries with stable APIs.

### Tertiary (LOW confidence)
- **OGame public API (XML)** — `sNNN-LL.ogame.gameforge.com/api/...` for server data. Referenced by TBot but not directly tested. May be useful for universe speed and player data.
- **Ninja Captcha Solver** — referenced in ogamed wiki for automated captcha solving at $0.10/solve. Not directly tested; pricing/API may have changed.

---
*Research completed: 2026-04-25*
*Ready for roadmap: yes*
