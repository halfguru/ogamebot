# AGENTS.md

## Project: OGameX Bot

An open-source OGame automation bot targeting OGameX (github.com/lanedirt/OGameX) — an open-source OGame clone. Go bot engine talks directly to OGameX's web endpoints (session-based auth + CSRF). SolidJS web dashboard for monitoring.

## Architecture

- **OGameX Client** (Go) — HTTP client handling Laravel session auth, CSRF tokens, HTML/JSON parsing
- **Bot Engine** (Go) — automation workers (defender, builder, farmer), game state management, REST API
- **Web Dashboard** (SolidJS/TypeScript) — real-time monitoring and configuration
- **Single Go binary** — no Docker needed, runs natively on Windows

## Planning

Project uses GSD (Get Shit Done) workflow. All planning artifacts in `.planning/`.

| Artifact | Location |
|----------|----------|
| Project context | `.planning/PROJECT.md` |
| Requirements | `.planning/REQUIREMENTS.md` |
| Roadmap | `.planning/ROADMAP.md` |
| State | `.planning/STATE.md` |
| Research | `.planning/research/` |
| Config | `.planning/config.json` |

## Key Decisions

- OGameX (open-source clone) as target — no anti-bot, no Gameforge
- Go HTTP client with session cookies + CSRF token management (no ogamed)
- Swap client layer — reuse all existing workers (defender, builder, farmer)
- Go for bot engine (developer knows Go best)
- SolidJS/TypeScript for web dashboard only
- SQLite + Go standard library (zero-ops, single-user)
- YAML config files for user-facing configuration
- Native Go binary on Windows (no Docker)

## Conventions

- Go module for bot engine (`cmd/bot/`, `internal/`)
- `internal/ogamex/` — OGameX HTTP client (replaces `internal/ogamed/`)
- pnpm workspace for dashboard (`packages/dashboard`, `packages/shared`)
- YAML config files for user-facing configuration
- goquery for HTML parsing (CSRF tokens, planet data from OGameX pages)

## Workflow Enforcement

- Do NOT create new phases — use `/gsd-plan-phase N`
- Do NOT skip phases — execute in order
- Commit after each plan completes
- Run lint/typecheck before committing

## Commands

```bash
go build ./cmd/bot/          # Build bot binary
go test ./...                # Run all tests
```
