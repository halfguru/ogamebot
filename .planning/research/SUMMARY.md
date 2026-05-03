# Research Summary: OGameX Bot

**Date:** 2026-05-03
**Domain:** OGame bot targeting open-source OGameX clone (github.com/lanedirt/OGameX)

## Key Findings

### Stack
- **Only 1 new dependency needed**: `github.com/PuerkitoBio/goquery` v1.12.0 for HTML parsing (CSRF token extraction, planet/building data from HTML pages)
- Go stdlib `net/http` + `cookiejar` handles session cookies natively
- `encoding/json` decodes AJAX responses (OGameX returns JSON for game data endpoints)
- No browser automation needed — OGameX is traditional form-post + AJAX
- Keep existing: modernc.org/sqlite, gorilla/websocket, yaml.v3
- Remove: all ogamed dependencies, Docker for ogamed

### Table Stakes
7 features, all already implemented in the codebase:
1. Session management (login, CSRF, reconnection)
2. Game state retrieval (planets, resources, fleets, buildings)
3. Fleet-save / defender (attack detection + auto-save)
4. Auto-build (ROI calculator + building execution)
5. Auto-farm (galaxy scan + espionage + attack)
6. Configuration (YAML)
7. Logging

### Differentiators (already built or incremental)
- Web dashboard with real-time WebSocket (already built)
- Auto-expeditions
- Auto-research
- Telegram notifications
- Multi-account support

### OGameX-Specific Opportunities
- **Zero anti-bot evasion** — no captcha, no proxy, no fingerprinting
- **Single Go binary** — no Docker, no ogamed middleware
- **Direct DB access** if self-hosting (MySQL queries for game state)
- **No Lifeforms** — simpler scope than TBot

### Critical Pitfalls
1. **CSRF token rotation on every response** — must be automatic + thread-safe
2. **No structured state endpoints** — must parse HTML for planet lists, building levels
3. **Mixed response formats** — HTML, JSON, and JSON-with-embedded-HTML
4. **Fleet dispatch is two-step** — check-target then send-fleet
5. **Concurrent goroutine token conflicts** — multiple workers sharing rotating CSRF token

### Build Order
1. Login + session management + CSRF
2. Planet list + resource retrieval (HTML parsing)
3. Fleet operations (send, recall, movements)
4. Build actions (upgrade buildings, start research)
5. Galaxy scan + espionage
6. Wire up existing workers (defender, builder, farmer)
7. Cleanup (remove ogamed package, update Docker/config)

---
*Research synthesized: 2026-05-03*
