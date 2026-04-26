# Phase 5 Research: Web Dashboard

**Date:** 2026-04-26
**Status:** Complete

## Overview

Phase 5 adds a web dashboard to the OGame bot, requiring both a Go HTTP/WebSocket API layer and a SolidJS frontend. This is the only phase that spans two languages/runtimes (Go backend + TypeScript frontend).

## Existing Codebase (Read from codebase)

### Go Backend (Phases 1-4)
- **State Manager** (`internal/state/manager.go`): All game data cached in SQLite with public read methods: `GetPlanets`, `GetResources`, `GetFleets`, `GetResearch`, `GetBuildings`, `GetFacilities`, `GetServerSpeed`
- **SQLite Schema**: 7 tables — `planets`, `resources`, `buildings`, `facilities`, `research`, `fleets`, `fleet_save_events`, `build_events`, `farm_targets`, `farm_attacks`
- **Model types** (`internal/model/types.go`): Rich domain types with JSON tags (PascalCase matching ogamed)
- **Config** (`internal/config/config.go`): No dashboard config yet — needs `DashboardConfig` added
- **Main entrypoint** (`cmd/bot/main.go`): Wire workers in sections 8.5-8.7, need section 8.8 for dashboard server
- **Dependencies**: `modernc.org/sqlite`, `gopkg.in/yaml.v3` — no HTTP/WebSocket deps yet

### TypeScript Workspace (Phase 1 scaffold)
- **pnpm monorepo** with `packages/shared` (Zod schemas + TS types) and `packages/dashboard` (placeholder)
- **Shared types** use camelCase (`planet.ts`: `{id, name, coordinate, ...}`)
- **Shared schemas** use PascalCase matching ogamed (`planets.ts`: `{ID, Name, Coordinate, ...}`)
- **Dashboard package** is empty (placeholder build script only)
- **tsconfig.base.json**: ES2022, Node16 modules, strict mode

### Docker
- **docker-compose.yml**: Two services (ogamed, bot), no dashboard port exposed
- **Dockerfile**: Multi-stage Go build, no static asset embedding

## Technology Decisions

### Go HTTP API: stdlib `net/http`
- No framework needed — just 5-6 JSON endpoints
- `http.NewServeMux` with Go 1.22+ method routing (`mux.HandleFunc("GET /api/planets", ...)`)
- CORS middleware for dev mode (dashboard dev server on different port)

### Go WebSocket: `github.com/gorilla/websocket`
- Industry standard, well-tested, gorilla/chat hub pattern for broadcasting
- Hub goroutine manages client registration/unregistration via channels
- Each client gets readPump + writePump goroutines for concurrent safety
- Alternative considered: `nhooyr.io/websocket` — more modern but gorilla is battle-tested and simpler
- Alternative rejected: `golang.org/x/net/websocket` — deprecated, lacks compression, poor docs

### Frontend: SolidJS + Vite
- Already decided in AGENTS.md
- SolidJS provides fine-grained reactivity without Virtual DOM overhead
- `createSignal` for reactive state, `onMount` for initial data fetch, `For`/`Show` for rendering
- No SolidStart needed — this is a SPA consuming Go API, not SSR

### Build/Deploy: Go `embed` + Vite build output
- `//go:embed dashboard/*` to embed Vite's `dist/` into Go binary
- Single binary deployment, no separate static file server
- Dashboard served at `/` with API at `/api/` and WebSocket at `/ws`

## API Design

### REST Endpoints (JSON)

| Method | Path | Response | Source |
|--------|------|----------|--------|
| GET | `/api/planets` | `[]Planet` with nested resources + buildings | state manager |
| GET | `/api/fleets` | `[]Fleet` | state manager |
| GET | `/api/research` | `Research` | state manager |
| GET | `/api/events/builds` | `[]BuildEvent` (last 50) | SQLite build_events |
| GET | `/api/events/fleet-saves` | `[]FleetSaveEvent` (last 20) | SQLite fleet_save_events |
| GET | `/api/events/farm-attacks` | `[]FarmAttack` (last 50) | SQLite farm_attacks |

### WebSocket Endpoint

| Path | Protocol | Messages |
|------|----------|----------|
| `/ws` | JSON frames | Server pushes `state_update`, `build_event`, `fleet_save_event`, `farm_attack` |

### WebSocket Message Types (server → client)

```json
{"type": "state_update", "data": {"planets": [...], "fleets": [...], "research": {...}}}
{"type": "build_event", "data": {"planet_id": 1, "building_name": "MetalMine", "from_level": 20, "to_level": 21, ...}}
{"type": "fleet_save_event", "data": {"planet_id": 1, "fleet_id": 123, "dest_planet_id": 2, ...}}
{"type": "farm_attack", "data": {"fleet_id": 456, "target_galaxy": 1, "target_system": 100, ...}}
```

### JSON Format: CamelCase API Response

The Go API returns **camelCase** JSON (not PascalCase like ogamed). The dashboard is a *consumer* of the Go bot API (not ogamed directly), so we control the format. CamelCase is standard for web APIs and matches TypeScript conventions.

Go model types for API responses will use `json:"camelCase"` tags (separate from ogamed's `json:"PascalCase"`).

## Architecture

### Go Package Layout

```
internal/
  dashboard/
    server.go      # HTTP server setup, CORS, static file serving
    handlers.go    # REST endpoint handlers
    hub.go         # WebSocket hub (broadcast to connected clients)
    types.go       # API response types (camelCase JSON)
cmd/bot/main.go    # Wire dashboard server in section 8.8
```

### Hub Pattern

The Hub is a singleton that:
1. Maintains a set of connected WebSocket clients
2. Exposes `Broadcast(msgType string, data any)` method
3. State manager calls `hub.Broadcast("state_update", ...)` after each refresh
4. Builder calls `hub.Broadcast("build_event", ...)` after each build
5. Defender calls `hub.Broadcast("fleet_save_event", ...)` after each save
6. Farmer calls `hub.Broadcast("farm_attack", ...)` after each attack

### Frontend Component Tree

```
App
├── Header (connection status, bot uptime)
├── EmpireOverview
│   └── PlanetCard[] (name, coords, resources, buildings, fields)
├── FleetMovements
│   └── FleetRow[] (origin → destination, mission, arrival, cargo)
├── ActivityFeed
│   ├── BuildEvents[] (building, level, cost)
│   ├── FleetSaveEvents[] (planet, destination, recall time)
│   └── FarmAttacks[] (target, loot, status)
└── Footer (last update time)
```

## Configuration

Add to `config.yaml`:

```yaml
dashboard:
  enabled: true
  port: 3000
  corsOrigins:
    - "http://localhost:5173"  # Vite dev server
```

New Go type:

```go
type DashboardConfig struct {
    Enabled      bool     `yaml:"enabled"`
    Port         int      `yaml:"port"`
    CorsOrigins  []string `yaml:"corsOrigins"`
}
```

## Common Pitfalls

1. **SQLite single-writer**: Dashboard reads must not block worker writes. WAL mode is already enabled — dashboard reads are readers in WAL mode and won't block.
2. **WebSocket write concurrency**: gorilla/websocket requires one writer at a time. Use per-client write channel (gorilla chat pattern).
3. **Stale state on reconnect**: Client must fetch full state via REST on connect, then apply incremental WebSocket updates. WebSocket is for real-time pushes, not the sole data source.
4. **Dashboard static files in dev vs prod**: In dev, Vite dev server proxies API/WS to Go server. In prod, Go serves embedded static files. Need Vite proxy config.
5. **CORS**: Must be enabled for dev mode (Vite dev server → Go API). In prod, same-origin (Go serves both).
6. **Type mismatch**: Go model types use PascalCase JSON (ogamed format). API response types use camelCase. Keep them separate.

## Out of Scope

- Dashboard configuration editing (toggle features, change params) — v2
- Authentication/authorization — single-user bot on private network
- Mobile-specific layout — responsive CSS covers it
- Push notifications from dashboard — Telegram integration is v2
- Charting/graphs — v2 enhancement
- Dark/light theme toggle — v2
