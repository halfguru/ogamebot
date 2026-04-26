# AGENTS.md

## Project: OGame Bot

An open-source OGame automation bot built on ogamed (Go REST backend) + TypeScript bot logic + web dashboard.

## Architecture

- **ogamed** (Go) — REST daemon handling all OGame API communication (runs in Docker)
- **Bot Engine** (TypeScript/Node.js) — automation decisions, scheduling, game state management
- **Web Dashboard** (SolidJS) — real-time monitoring and configuration

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
- TypeScript for bot + dashboard (shared language)
- SQLite + Drizzle ORM (zero-ops, single-user)
- Fastify 5 for API/WebSocket server
- SolidJS for web dashboard
- Telegraf for Telegram integration (v2)

## Conventions

- Monorepo with pnpm workspaces (`packages/bot`, `packages/dashboard`, `packages/shared`)
- Zod for runtime validation of ogamed responses
- YAML config files for user-facing configuration
- Docker Compose for deployment (ogamed + bot)

## Workflow Enforcement

- Do NOT create new phases — use `/gsd-plan-phase N`
- Do NOT skip phases — execute in order
- Commit after each plan completes
- Run lint/typecheck before committing

## Commands

(To be defined during Phase 1 — project scaffolding)
