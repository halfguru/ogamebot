# OGame Bot

## What This Is

An open-source OGame automation bot that handles the tedious parts of playing OGame — auto-building, auto-farming, fleet-saving, and expedition management. Built on top of `ogamed` (alaingilbert/ogame Go library) as a REST backend, with a TypeScript-based bot logic layer and a web dashboard for monitoring and configuration. Designed to run 24/7 on a server, accessible from anywhere.

## Core Value

The bot must reliably protect your fleet and grow your empire while you're away — if fleet-save fails, everything else is pointless.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Connect to OGame account via ogamed REST API and retrieve game state (planets, resources, fleets, buildings, research)
- [ ] Smart fleet-save: detect incoming attacks and auto-save fleet + resources (phalanx-safe, with recall)
- [ ] ROI-based auto-build: upgrade the most profitable building/research across all planets
- [ ] Auto-farm: scan galaxy ranges, spy inactive players, attack if profitable
- [ ] Auto-expeditions: manage expedition slots, optimize fleet composition, auto-resend
- [ ] Telegram notifications: attack alerts, build completions, bot status
- [ ] Web dashboard: real-time overview of empire, fleet movements, build queues, and logs
- [ ] Configurable behavior: settings for each feature (priority targets, build limits, farm ranges, etc.)
- [ ] Anti-detection: random delays, request throttling, device fingerprinting (handled by ogamed)
- [ ] Multi-account support

### Out of Scope

- Browser-based UI proxy (playing through the bot) — complexity without core value
- Marketplace automation — niche feature, adds complexity
- Lifeform-specific logic — can add later when core is solid
- Combat simulator — use existing tools (speedsim, etc.)
- Mobile app — web dashboard is mobile-responsive instead

## Context

### OGame Ecosystem

- OGame is a browser-based space strategy MMO by Gameforge
- `ogamed` (github.com/alaingilbert/ogame) is an actively maintained Go library (v53, 3300+ commits) that wraps the entire OGame HTTP API
- It provides a REST daemon (`ogamed`) with endpoints for everything: buildings, fleets, research, espionage, etc.
- The Python ogame library is unmaintained since 2021 — Go is the only viable option

### Reference Projects

- **TBot** (github.com/ogame-tbot/TBot, 94 stars) — Most complete OGame bot. C#/.NET on top of ogamed. Has WebUI, Telegram integration, auto-mine (ROI-based), auto-farm, defender, expeditions, multi-account, settings hot-reload. Key inspiration.
- **Cruiser** (github.com/kweimann/cruiser, 30 stars) — Focused on safety. Smart fleet-save (phalanx-safe, auto-recall), Telegram notifications, YAML config, Docker support. Clean architecture with separate OGame client + game engine + bot logic.
- **r4fek/ogame-bot** (45 stars, archived) — Simple Python bot with SMS notifications, combat sim integration, transport manager.

### Key Patterns from Reference Projects

1. **Telegram for remote control + notifications** — every successful bot has this
2. **ROI-based auto-build** — TBot's approach of calculating return on investment is the gold standard
3. **Smart fleet-save** — must be phalanx-safe with recall capability
4. **Web dashboard** — TBot proves this is expected for any modern bot
5. **Config hot-reload** — critical for a 24/7 service
6. **Sleep mode** — reduce activity during specific hours to avoid detection

### Technical Considerations

- ogamed handles device fingerprinting, anti-bot measures, and captcha
- OGame servers rate-limit requests — bot must throttle and respect limits
- Residential proxy support may be needed (datacenter IPs get blocked)
- Bot needs to handle OGame version updates gracefully (ogamed usually updates quickly)

## Constraints

- **Backend**: Must use ogamed (Go) for OGame API interaction — it's the only maintained library
- **Bot Logic**: TypeScript/Node.js for the automation logic and web dashboard
- **Runtime**: Must run 24/7 headless on a VPS
- **Security**: Credentials must be stored securely, never committed
- **Anti-ban**: Scripting/botting violates OGame TOS — anti-detection measures are mandatory, not optional

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| ogamed as REST backend | Only maintained OGame API wrapper; REST API allows language flexibility | — Pending |
| TypeScript for bot + dashboard | Shared language for bot logic and web UI; good ecosystem for automation | — Pending |
| Telegram for notifications | Industry standard for bots; easy to set up; supports commands | — Pending |
| Web dashboard (not terminal) | 24/7 service needs remote monitoring; mobile-accessible | — Pending |
| Open source from the start | User wants others to be able to use it | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-25 after initialization*
