# Project State: OGameX Bot

**Updated:** 2026-05-03
**Current Phase:** 1 — Core Infrastructure
**Status:** Ready to begin

## Phase Status

| Phase | Name | Status | Plans | Requirements |
|-------|------|--------|-------|-------------|
| 1 | Core Infrastructure | **Current** | 2 | INFRA-01..04 |
| 2 | Game State | Not started | 2 | STATE-01..07 |
| 3 | Fleet Safety | Not started | 2 | SAFE-01..04 |
| 4 | Auto-Build | Not started | 2 | BUILD-01..04 |
| 5 | Auto-Farm | Not started | 1 | FARM-01..03 |
| 6 | Web Dashboard | Not started | 2 | DASH-01..03 |

## Current Work

**Phase 1 — Core Infrastructure**
- **Next plan:** 1.1 — Config + OGameX client skeleton
- **Blockers:** None
- **Notes:** Brownfield pivot — existing workers, state manager, and dashboard are reused

## Milestones

| Milestone | Phase | Status | Description |
|-----------|-------|--------|-------------|
| M1: Login works | Phase 1 | Pending | Bot authenticates with OGameX, CSRF token rotates correctly |
| M2: State cached | Phase 2 | Pending | SQLite populated with live planet/resource/fleet data |
| M3: Fleet protected | Phase 3 | Pending | Defender detects attacks and fleet-saves autonomously |
| M4: Empire grows | Phase 4 | Pending | Builder upgrades buildings by ROI across all planets |
| M5: Farming active | Phase 5 | Pending | Farmer scans, spies, and attacks profitable inactives |
| M6: Fully operational | Phase 6 | Pending | Dashboard + all features working, single Go binary |

## Completed Plans

(none yet)

## Risk Log

| Risk | Impact | Mitigation | Phase |
|------|--------|------------|-------|
| CSRF token race conditions | Fleet-save fails mid-request | Mutex-protected token store, re-auth on 419 | 1 |
| OGameX HTML structure changes | All parsers break | Pin to specific OGameX version, add parser tests | 2 |
| Building/ship IDs differ from ogamed | Wrong actions executed | Validate IDs against OGameX DOM early in Phase 2 | 2 |
| Fleet dispatch two-step fails | Fleet-save cannot execute | Test check-target + send-fleet thoroughly | 3 |
| Rate limiting on galaxy scan | Farmer too slow | Space out requests, cache galaxy data | 5 |

## Architecture Notes

### What Changes (this roadmap)
- `internal/ogamed/` → `internal/ogamex/` (new client package)
- `config.OgamedConfig` → `config.OGameXConfig` (new config fields)
- `cmd/bot/main.go` wiring (ogamex client instead of ogamed)
- Docker Compose (remove ogamed service)

### What Stays (reuse as-is)
- `internal/model/` — domain types (may need JSON tag adjustments)
- `internal/constants/` — ship/building/mission IDs (verify against OGameX)
- `internal/defender/` — attack detection, escape routes, fleet-save logic
- `internal/builder/` — ROI calculator, build queue logic
- `internal/farmer/` — galaxy scan, espionage, profit evaluation
- `internal/state/` — SQLite cache, state manager, read methods
- `internal/dashboard/` — HTTP handlers, WebSocket hub
- `internal/config/` — YAML loading, validation (extend, don't rewrite)
- `packages/dashboard/` — SolidJS frontend (read-only from API)

### Key Interface
```go
// ogamed.ClientInterface — 26 methods, unchanged
// ogamex.Client must satisfy this same interface
// Workers never import ogamed directly — they use the interface
```

## Dependencies

### Runtime
- Go stdlib (`net/http`, `encoding/json`, `database/sql`)
- `modernc.org/sqlite` — embedded SQLite
- `github.com/PuerkitoBio/goquery` — HTML parsing (NEW)
- `gopkg.in/yaml.v3` — config
- `github.com/gorilla/websocket` — dashboard

### Removed
- ogamed Docker container
- All ogamed REST API calls

---
*State initialized: 2026-05-03*
