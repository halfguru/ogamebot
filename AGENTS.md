# AGENTS.md

## Project: OGame Bot

An open-source OGame automation bot built on ogamed (Go REST backend) + TypeScript bot logic + web dashboard.

## Architecture

- **ogamed** (Go) — REST daemon handling all OGame API communication (runs in Docker)
- **Bot Engine** (Go) — automation decisions, scheduling, game state management, exposes REST API for dashboard
- **Web Dashboard** (SolidJS/TypeScript) — real-time monitoring and configuration

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

- ogamed REST backend (only maintained OGame API wrapper)
- Go for bot engine (developer knows Go best, shares language with ogamed)
- SolidJS/TypeScript for web dashboard only
- SQLite + Go standard library (zero-ops, single-user)
- YAML config files for user-facing configuration
- Docker Compose for deployment (ogamed + bot)

## Conventions

- Go module for bot engine (`cmd/bot/`, `internal/`)
- pnpm workspace for dashboard (`packages/dashboard`, `packages/shared`)
- Zod for runtime validation of dashboard API responses
- YAML config files for user-facing configuration
- Docker Compose for deployment (ogamed + bot)

## Workflow Enforcement

- Do NOT create new phases — use `/gsd-plan-phase N`
- Do NOT skip phases — execute in order
- Commit after each plan completes
- Run lint/typecheck before committing

## Commands

(To be defined during Phase 1 — project scaffolding)
