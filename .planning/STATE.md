# Project State: OGameX Bot

**Updated:** 2026-05-04
**Current Phase:** 6 — Web Dashboard + Cleanup
**Status:** All phases complete, ogamed fully removed

## Phase Status

| Phase | Name | Status | Plans | Requirements |
|-------|------|--------|-------|-------------|
| 1 | Core Infrastructure | **Complete** | 2 | INFRA-01..04 |
| 2 | Game State | **Complete** | 2 | STATE-01..07 |
| 3 | Fleet Safety | **Complete** | 2 | SAFE-01..04 |
| 4 | Auto-Build | **Complete** | 2 | BUILD-01..04 |
| 5 | Auto-Farm | **Complete** | 1 | FARM-01..03 |
| 6 | Web Dashboard | **Complete** | 2 | DASH-01..03 |

## Current Work

**All phases complete.**
- OGameX client (`internal/ogamex/`) fully implemented and wired
- All workers (defender, builder, farmer) connected to OGameX client
- ogamed package fully removed — no fallback, no Docker dependencies
- Single Go binary builds with `go build ./cmd/bot/`

## Milestones

| Milestone | Phase | Status | Description |
|-----------|-------|--------|-------------|
| M1: Login works | Phase 1 | **Done** | Bot authenticates with OGameX, CSRF token rotates correctly |
| M2: State cached | Phase 2 | **Done** | All read-only ClientInterface methods implemented |
| M3: Fleet protected | Phase 3 | **Done** | SendFleet + CancelFleet implemented, defender wired |
| M4: Empire grows | Phase 4 | **Done** | BuildBuilding implemented, builder wired |
| M5: Farming active | Phase 5 | **Done** | Galaxy scan + espionage + farming methods implemented |
| M6: Fully operational | Phase 6 | **Done** | Dashboard + all features working, single Go binary |

## Completed Plans

- Phase 1: client skeleton + login/session/CSRF
- Phase 2: parser infrastructure + read-only state methods
- Phase 3-5: SendFleet, CancelFleet, BuildBuilding, GetGalaxyInfos, espionage methods

## Risk Log

| Risk | Impact | Mitigation | Phase |
|------|--------|------------|-------|
| CSRF token race conditions | Fleet-save fails mid-request | Mutex-protected token store, re-auth on 419 | 1 |
| OGameX HTML structure changes | All parsers break | Pin to specific OGameX version, add parser tests | 2 |
| Building/ship IDs differ from expected | Wrong actions executed | Validate IDs against OGameX DOM early in Phase 2 | 2 |
| Fleet dispatch two-step fails | Fleet-save cannot execute | Test check-target + send-fleet thoroughly | 3 |
| Rate limiting on galaxy scan | Farmer too slow | Space out requests, cache galaxy data | 5 |

## Architecture Notes

### Architecture

- `internal/ogamex/` — OGameX HTTP client (session auth, CSRF, AJAX endpoints)
- `config.OGameXConfig` — config fields for OGameX connection
- `cmd/bot/main.go` — wiring with ogamex client

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
// ClientInterface — 26 methods, unchanged
// ogamex.Client satisfies this interface
// Workers never import concrete client — they use the interface
```

## Dependencies

### Runtime
- Go stdlib (`net/http`, `encoding/json`, `database/sql`)
- `modernc.org/sqlite` — embedded SQLite
- `github.com/PuerkitoBio/goquery` — HTML parsing
- `gopkg.in/yaml.v3` — config
- `github.com/gorilla/websocket` — dashboard

---
*State updated: 2026-05-04 — all phases complete*
