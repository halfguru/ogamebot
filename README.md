# OGame Bot

Open-source OGame automation bot. Connects to your OGame account via [ogamed](https://github.com/alaingilbert/ogame) and handles fleet safety, auto-building, auto-farming, and provides a web dashboard for monitoring.

> ⚠️ Botting violates OGame Terms of Service. Use at your own risk.

## Features

- **Fleet Safety** — Detects incoming attacks and auto-saves your fleet with phalanx-safe deploy + recall
- **Auto-Build** — ROI-based building upgrades across all planets, respects max-level caps
- **Auto-Farm** — Scans galaxy for inactive players, spies them, attacks when profitable
- **Web Dashboard** — Real-time empire overview with WebSocket updates, fleet movements, build/activity logs
- **Anti-Detection** — Randomized request intervals, jitter on all actions, configurable rate limiting
- **Zero-Ops** — SQLite database, single Docker Compose command, persistent data volume

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- An OGame account

### 1. Configure

```bash
cp .env.example .env
cp config.example.yaml config.yaml
```

Edit `.env` with your OGame credentials:

```env
OGAMED_UNIVERSE=s123-en.ogame.gameforge.com
OGAMED_USERNAME=your_username
OGAMED_PASSWORD=your_password
OGAMED_LANGUAGE=en
```

Edit `config.yaml` to enable features:

```yaml
features:
  defender:
    enabled: true
    pollIntervalMs: 30000
    recallEnabled: true
  autoBuild:
    enabled: true
    pollIntervalMs: 120000
  autoFarm:
    enabled: true
    pollIntervalMs: 300000
```

### 2. Run

```bash
docker compose up --build -d
```

### 3. Monitor

Open http://localhost:3000 for the web dashboard.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   ogamed (Go)   │     │   Bot (Go)      │
│   OGame API     │◄────│   Automation    │
│   REST daemon   │     │   engine        │
└─────────────────┘     │                 │
                        │  ┌───────────┐  │    ┌──────────────┐
                        │  │ Defender  │  │    │  Dashboard   │
                        │  │ Builder   │  │◄──►│  (SolidJS)   │
                        │  │ Farmer    │  │ WS │              │
                        │  └───────────┘  │    └──────────────┘
                        │  ┌───────────┐  │
                        │  │  SQLite   │  │
                        │  │  (state)  │  │
                        │  └───────────┘  │
                        └─────────────────┘
```

- **ogamed** — REST daemon wrapping OGame's HTTP API (handles auth, anti-bot, captcha)
- **Bot engine** — Go workers for fleet safety, auto-build, auto-farm, with REST API + WebSocket for the dashboard
- **Dashboard** — SolidJS SPA, served embedded from the Go binary, real-time updates via WebSocket

## Project Structure

```
cmd/bot/main.go              Entrypoint
internal/
  config/                    YAML config loader with env interpolation
  constants/                 OGame building/ship/mission IDs
  model/                     Domain types
  ogamed/                    REST client with rate limiter + retry
  state/                     SQLite database + game state cache
  defender/                  Fleet safety worker + escape route calculator
  builder/                   Auto-build worker + ROI calculator
  farmer/                    Auto-farm worker (scan/spy/attack)
  dashboard/                 REST API + WebSocket hub
packages/
  dashboard/                 SolidJS web frontend
```

## Configuration

All settings live in `config.yaml`. Secrets come from `.env` via `${ENV_VAR}` interpolation.

| Setting | Default | Description |
|---------|---------|-------------|
| `features.defender.enabled` | `false` | Enable fleet-save on attack detection |
| `features.defender.recallEnabled` | `true` | Auto-recall fleet after attack passes |
| `features.defender.safetyMarginMs` | `120000` | Min time before attack to trigger save |
| `features.autoBuild.enabled` | `false` | Enable ROI-based building upgrades |
| `features.autoFarm.enabled` | `false` | Enable inactive player farming |
| `rateLimit.defaultMinDelayMs` | `2000` | Min delay between API calls (ms) |
| `dashboard.enabled` | `true` | Enable web dashboard |
| `dashboard.port` | `3000` | Dashboard HTTP port |

## Development

### Run locally (without Docker)

```bash
# Start ogamed separately (requires Go + OGame credentials)
# Then run the bot:
go run ./cmd/bot
```

### Run tests

```bash
go test ./...
```

### Build

```bash
go build -o bot ./cmd/bot
```

## License

MIT
