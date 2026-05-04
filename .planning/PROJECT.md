# OGameX Bot

## What This Is

An automation bot for OGameX (open-source OGame clone at main.ogamex.dev) that handles fleet-saving, auto-building, auto-farming, and expedition management. Built as a Go binary that talks directly to OGameX's web endpoints (session-based auth + CSRF tokens via AJAX/JSON), with a SolidJS dashboard for real-time monitoring.

## Core Value

The bot must reliably protect your fleet and grow your empire while you're away — if fleet-save fails, everything else is pointless.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Connect to OGameX via HTTP (session login with CSRF tokens) and retrieve game state (planets, resources, fleets, buildings, research)
- [ ] Smart fleet-save: detect incoming attacks and auto-save fleet + resources (phalanx-safe, with recall)
- [ ] ROI-based auto-build: upgrade the most profitable building/research across all planets
- [ ] Auto-farm: scan galaxy ranges, spy inactive players, attack if profitable
- [ ] Auto-expeditions: manage expedition slots, optimize fleet composition, auto-resend
- [ ] Web dashboard: real-time overview of empire, fleet movements, build queues, and logs
- [ ] Configurable behavior: settings for each feature (priority targets, build limits, farm ranges, etc.)
- [ ] Multi-account support

### Out of Scope

- Official OGame support — Gameforge anti-bot is insurmountable without residential proxies
- Browser-based UI proxy (playing through the bot) — complexity without core value
- Marketplace automation — niche feature
- Combat simulator — use existing tools
- Mobile app — web dashboard is mobile-responsive instead
- Adding a REST API to OGameX itself — bot will work with existing AJAX endpoints

## Context

### Targeting OGameX

This bot targets OGameX (github.com/lanedirt/OGameX), an open-source OGame clone. OGameX has no anti-bot protections, making it an ideal automation target.

### OGameX Architecture

- OGameX is a PHP/Laravel application with server-rendered HTML pages and AJAX/JSON endpoints
- Authentication: Laravel Fortify session-based auth (email/password → session cookie + CSRF token)
- All game actions happen via AJAX POST endpoints returning JSON (e.g., `/ajax/fleet/dispatch/send-fleet`)
- Game data available via AJAX GET endpoints (e.g., `/ajax/resources`, `/ajax/fleet/eventlist/fetch`)
- Planet switching via `?cp=<planet_id>` query parameter
- Mission types: 1=Attack, 3=Transport, 4=Deploy, 6=Espionage, 7=Colonize, 8=Recycle, 9=Moon Destruct, 15=Expedition

### Existing Codebase

The bot has clean architecture with a `ClientInterface` abstraction (26 methods). All workers (defender, builder, farmer) depend on this interface, never the concrete client. The OGameX client implements this interface, reusing:
- Domain types (Planet, Fleet, Resources, Ships, etc.)
- Game constants (ship IDs, building IDs, mission types)
- Pure-math engines (ROI calculator, escape route calculator)
- Workers (defender, builder, farmer) with minimal import path changes
- State manager (SQLite cache)
- Web dashboard (SolidJS + Go REST/WebSocket server)

### Reference Projects

- **TBot** (github.com/ogame-tbot/TBot) — Most complete OGame bot. C#/.NET with its own HTTP client.
- **Cruiser** (github.com/kweimann/cruiser) — Focused on safety. Smart fleet-save.

## Constraints

- **Backend**: Go for the bot engine — developer knows Go best, shares language with existing codebase
- **Target**: OGameX at main.ogamex.dev (configurable for any OGameX instance)
- **Auth**: Must handle Laravel session cookies + CSRF token extraction/rotation
- **Dashboard**: SolidJS/TypeScript for web dashboard only
- **Deployment**: Native Go binary on Windows (no Docker needed for the bot itself)
- **No anti-bot**: OGameX has no captcha, fingerprinting, or rate limiting beyond basic login throttle

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Target OGameX directly | OGameX has zero anti-bot; ideal automation target | — Complete |
| OGameX client with ClientInterface | Clean abstraction makes this 70% reuse of existing workers | — Complete |
| Target main.ogamex.dev live demo | No need to self-host; demo is always available | — Complete |
| Native Go binary (no Docker) | Single binary is simpler; no middleware dependencies | — Complete |
| Go for bot engine | Developer knows Go best; goroutines for parallel ops | — Complete |
| SolidJS for dashboard only | Existing dashboard code works; mobile-responsive | — Complete |
| Session-based auth with CSRF | OGameX uses Laravel Fortify; bot must maintain session + token | — Complete |

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
*Last updated: 2026-05-04 — all ogamed references removed*
