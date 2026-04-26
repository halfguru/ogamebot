# Stack Research

**Domain:** Game automation bot (OGame) with REST backend + TypeScript bot logic + web dashboard
**Researched:** 2026-04-25
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| **ogamed** | v53.0.0 | OGame REST API backend | Only maintained OGame API wrapper. Go binary exposes REST API on port 8080 covering all game operations: fleets, buildings, research, espionage, galaxy scanning. Docker image available. Handles device fingerprinting and anti-bot measures. 3300+ commits, actively maintained. |
| **Node.js** | 22 LTS | Runtime for bot logic + web server | LTS stability for 24/7 operation. Native ESM support. Excellent ecosystem for HTTP clients, schedulers, and Telegram bots. |
| **TypeScript** | 5.7+ | Language for all TS code | Type safety critical for game logic (ship IDs, mission types, coordinates). Shared types between bot engine and dashboard. Strict mode catches bugs in fleet calculations and ROI math. |
| **pnpm** | 10+ | Package manager + monorepo workspace | Strict dependency isolation prevents phantom deps. Built-in workspace support for monorepo (bot-core, dashboard, shared). Faster installs than npm/yarn. Disk-efficient content-addressable store. |
| **SQLite** (via better-sqlite3) | — | Persistent state database | Single-file DB, zero ops for VPS deployment. Synchronous API (better-sqlite3) simplifies bot logic — no async DB calls mixed with game state updates. Stores: account configs, build history, farm targets, expedition logs, attack records. |
| **Fastify** | 5.x | Web framework for dashboard API | Fastest Node.js framework. TypeScript-first. Plugin system for clean separation (API routes, WebSocket, static serving). Built-in schema validation. `@fastify/cors`, `@fastify/static`, `@fastify/websocket` for dashboard needs. |
| **SolidJS** | 1.9+ | Dashboard frontend framework | Fine-grained reactivity (no virtual DOM overhead). Tiny bundle (~7KB). Perfect for real-time dashboard updates via signals. TSX syntax familiar to React devs but faster. Less complexity than React for a dashboard that's mostly data display + config forms. |
| **Telegraf** | 4.x | Telegram bot framework | Industry standard for Node.js Telegram bots. Full Bot API support (commands, inline keyboards, webhooks). TypeScript support. Used by all reference OGame bots for notifications + remote commands. Middleware system for command parsing. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **Drizzle ORM** | 0.39+ | Type-safe SQLite queries | All database access. Schema-as-code, type-safe queries, migration tooling. Much lighter than Prisma (no engine binary). Native better-sqlite3 driver. `drizzle-kit` for migrations. |
| **Zod** | 3.24+ | Runtime schema validation | Validating ogamed API responses (coordinates, ship IDs, resource counts). Config file validation. API request/response schemas. Runtime type guards where TS can't help (external API data). |
| **pino** | 9.x | Structured logging | All logging. Fastest Node.js logger (5x faster than winston). JSON output for log aggregation. Low overhead critical for 24/7 bot. Child loggers for per-module context. `pino-pretty` for dev. |
| **node-cron** | 3.x | Scheduled task execution | Game tick loops (check attacks every 30s, auto-build every 60s, farm scan every 5min). Cron syntax for sleep-mode schedules. Zero dependencies. No Redis required unlike BullMQ. |
| **ofetch** | 1.x | HTTP client for ogamed API | All calls to ogamed REST API. Built on fetch, auto-parses JSON, better error handling than axios. Works in Node.js natively since Node 18+. Lightweight. |
| **dotenv** | 16.x | Environment variable loading | Loading config from `.env` file (ogamed URL, Telegram token, account credentials). Standard across Node.js projects. |
| **Vite** | 6.x | Dashboard build tool + dev server | SolidJS dashboard bundling. HMR for dashboard development. Config hot-reload support. Much faster than webpack. |
| **@solidjs/router** | 0.15+ | Dashboard client routing | SPA routing for dashboard pages (overview, fleets, builds, settings, logs). |
| **Vitest** | 3.x | Test framework | Unit tests for bot logic (ROI calculations, fleet composition, coordinate math). Jest-compatible API but faster. Native TypeScript + ESM. |
| **Docker** | — | Deployment | Both ogamed and bot run in containers via docker-compose. Reproducible deployments. ogamed provides official Dockerfile. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| **tsx** | TypeScript execution without build step | Dev: `tsx watch src/main.ts` for instant reloads. No compile step during development. |
| **drizzle-kit** | Database migrations | `drizzle-kit generate` creates migration SQL from schema diffs. `drizzle-kit push` for dev. |
| **eslint + typescript-eslint** | Linting | Strict TS rules catch bugs in game logic calculations. |
| **prettier** | Code formatting | Consistent style across monorepo packages. |

## Installation

```bash
# Initialize monorepo
pnpm init

# Core workspace packages (in packages/)
pnpm add typescript @types/node tsx --save-dev -w

# Bot engine (packages/bot)
pnpm add pino ofetch node-cron telegraf dotenv zod --filter bot
pnpm add drizzle-orm better-sqlite3 --filter bot
pnpm add @types/better-sqlite3 --save-dev --filter bot

# Dashboard API (packages/api — also depends on bot)
pnpm add fastify @fastify/cors @fastify/static @fastify/websocket --filter api

# Dashboard frontend (packages/dashboard)
pnpm add solid-js @solidjs/router --filter dashboard
pnpm add vite @solidjs/vite-plugin --save-dev --filter dashboard
pnpm add tailwindcss --save-dev --filter dashboard

# Dev dependencies (root)
pnpm add vitest eslint prettier typescript-eslint --save-dev -w
pnpm add drizzle-kit pino-pretty --save-dev -w
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| **SQLite + Drizzle** | PostgreSQL + Prisma | If multi-user deployment (hosted for many players). SQLite single-file locks under concurrent writes. For personal/single-user bot, SQLite is simpler. |
| **node-cron** | BullMQ (Redis-backed) | If you need persistent job queues, retries, job deduplication, or horizontal scaling. BullMQ requires Redis which adds ops complexity. For a single-instance bot, node-cron with in-memory state is sufficient. |
| **SolidJS** | React | If team already knows React deeply and doesn't want to learn a new framework. SolidJS is smaller/faster but has a smaller ecosystem. |
| **SolidJS** | Svelte | Either works well. Svelte has larger community. SolidJS chosen for more familiar TSX syntax and fine-grained reactivity model that maps well to real-time game data. |
| **Fastify** | Express | Only if you need Express middleware ecosystem. Fastify is faster, has better TypeScript support, and modern plugin architecture. No reason to use Express for a new project. |
| **pino** | Winston | Only if you need Winston's broader transport ecosystem. Pino is 5x+ faster, structured by default, lower overhead for 24/7 bot. |
| **ofetch** | axios | Only if you need axios interceptors or request/response transformation pipeline. ofetch is lighter, uses native fetch, auto-parses JSON. |
| **pnpm** | npm workspaces | Only if team is unfamiliar with pnpm. npm workspaces work but are slower and lack strict dependency isolation. |
| **better-sqlite3** | sql.js (WASM) | If you can't compile native modules. sql.js is pure WASM but slower and doesn't support concurrent access. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| **Prisma** | Heavy engine binary (100MB+), slow cold starts, complex migration story, overkill for SQLite. Generates too much code. | Drizzle ORM — 7KB, no engine, type-safe SQL-like queries, native better-sqlite3 driver. |
| **BullMQ** | Requires Redis. Adds ops complexity (another service to manage, persist, monitor). Unnecessary for single-instance game bot with simple scheduled tasks. | node-cron for scheduling + in-memory state for tracking. Simpler, fewer moving parts. |
| **Express** | Legacy API design, poor TypeScript support, callback-based middleware, slower than modern alternatives. No built-in schema validation. | Fastify — faster, TypeScript-first, plugin system, built-in validation. |
| **Winston** | Slow (synchronous formatting on hot path), complex transport configuration, mutable log levels. | pino — fastest Node.js logger, structured JSON by default, child loggers. |
| **axios** | Large bundle, inconsistent API (`response.data`), interceptor bugs with error handling. | ofetch — native fetch wrapper, auto JSON parse, cleaner error handling. |
| **socket.io** | Adds custom protocol on top of WebSocket. Overkill for dashboard. Requires socket.io client. | Native WebSocket via `@fastify/websocket` — standard protocol, no custom client needed. |
| **NestJS** | Over-engineered for a game bot. Heavy decorator-based DI, Angular-like module system. Adds complexity without value for this domain. | Plain TypeScript with Fastify plugins — simpler, faster to build, easier to debug. |
| **MongoDB** | No relational data in this domain (planets have buildings, fleets have ships, targets have spy reports). Joins are natural. Adds ops complexity. | SQLite — relational, single-file, zero ops. |
| **PM2** | Docker handles process management. PM2 adds another layer of complexity on top of Docker's restart policies. | Docker + docker-compose with `restart: unless-stopped`. |

## Stack Patterns by Variant

**If deploying single-account (personal use):**
- SQLite only. No Redis. Single docker-compose with ogamed + bot containers.
- This is the default recommendation.

**If deploying multi-account (bot manages several OGame accounts):**
- Still SQLite (one DB file). node-cron still sufficient (one timer set per account).
- Add account switching in dashboard. ogamed supports multiple accounts via `AddAccount`.

**If scaling to hosted service (many users, each with own bot instance):**
- Switch to PostgreSQL + Drizzle for concurrent access.
- Switch to BullMQ + Redis for distributed job processing.
- Add authentication to dashboard API (JWT or session-based).
- This is out of scope per PROJECT.md but the architecture should not prevent it.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Compose                        │
│                                                         │
│  ┌──────────────┐     ┌──────────────────────────────┐  │
│  │   ogamed     │     │     Node.js Bot Process       │  │
│  │  (Go binary) │────▶│                              │  │
│  │  Port 8080   │REST │  ┌─────────────┐  ┌────────┐ │  │
│  └──────────────┘     │  │ Bot Engine  │  │ SQLite │ │  │
│                       │  │ (scheduler, │◀─│  DB    │ │  │
│                       │  │ game logic) │  └────────┘ │  │
│                       │  └──────┬──────┘             │  │
│                       │         │                    │  │
│                       │  ┌──────┴──────┐             │  │
│                       │  │ Fastify API │             │  │
│                       │  │ + WebSocket │             │  │
│                       │  └──────┬──────┘             │  │
│                       └─────────┼────────────────────┘  │
│                                 │                       │
│                       ┌─────────┴──────────┐            │
│                       │  SolidJS Dashboard │            │
│                       │  (served as static) │            │
│                       └────────────────────┘            │
│                                                         │
│  ┌──────────────┐                                       │
│  │   Telegram   │◀──── Telegraf (in bot process)        │
│  └──────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

## Monorepo Structure

```
ogame/
├── packages/
│   ├── shared/          # Shared types, constants, schemas (Zod)
│   │   ├── src/
│   │   │   ├── types/   # OGame types (Planet, Fleet, Resources, etc.)
│   │   │   ├── schemas/ # Zod schemas for ogamed API responses
│   │   │   └── constants/ # Ship IDs, mission types, building IDs
│   │   └── package.json
│   ├── bot/             # Bot engine (core logic)
│   │   ├── src/
│   │   │   ├── ogamed-client.ts   # HTTP wrapper for ogamed REST API
│   │   │   ├── engine/            # Game logic modules
│   │   │   │   ├── fleet-saver.ts
│   │   │   │   ├── auto-builder.ts
│   │   │   │   ├── auto-farmer.ts
│   │   │   │   └── expedition-manager.ts
│   │   │   ├── scheduler/         # Cron job definitions
│   │   │   ├── db/                # Drizzle schema + migrations
│   │   │   └── main.ts            # Entry point
│   │   └── package.json
│   ├── api/             # Fastify web API (depends on bot)
│   │   ├── src/
│   │   │   ├── routes/
│   │   │   ├── plugins/
│   │   │   └── main.ts
│   │   └── package.json
│   └── dashboard/       # SolidJS frontend
│       ├── src/
│       └── package.json
├── docker-compose.yml
├── pnpm-workspace.yaml
└── tsconfig.base.json
```

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| drizzle-orm@0.39+ | better-sqlite3@11+ | Drizzle has native better-sqlite3 driver. Use `drizzle-orm/better-sqlite3` import path. |
| SolidJS@1.9 | Vite@6 | Use `@solidjs/vite-plugin` for HMR. |
| Fastify@5 | Node.js@20+ | Fastify 5 requires Node 20+. |
| Telegraf@4 | Node.js@18+ | Full ESM support. Webhook or long polling mode. |
| node-cron@3 | Node.js@18+ | Zero dependencies. No native modules. |
| pino@9 | Node.js@18+ | pino-pretty for dev formatting. |
| Zod@3.24 | TypeScript@5.5+ | Zod 4 exists but is newer; Zod 3 is battle-tested. Consider Zod 4 for smaller bundle. |
| ogamed@53 | — | Go binary. Runs independently. REST API stable across versions. Check changelog for endpoint additions. |

## Key Technical Decisions

### Why not BullMQ + Redis?

The bot runs a **single process on a single VPS**. BullMQ's value proposition (persistent queues, retries, distributed workers, job deduplication) doesn't apply when:
- There's one worker (the bot itself)
- Jobs are simple time-based triggers (check attacks, scan galaxy)
- State is already persisted in SQLite
- No horizontal scaling needed

Adding Redis means another service to install, secure, backup, and monitor. node-cron with in-memory state is sufficient. If retry logic is needed, wrap task execution in a simple retry helper (3 attempts with exponential backoff).

### Why SQLite over PostgreSQL?

- **Single user, single process**: SQLite handles this perfectly. No concurrent write contention.
- **Zero ops**: No database server to configure, secure, or monitor. One file to backup.
- **Synchronous API**: better-sqlite3's synchronous calls simplify bot logic — no async/await chains for simple DB reads.
- **Easy deployment**: SQLite file lives alongside the bot. Copy file = backup. Delete file = reset.
- **Drizzle ORM support**: Full SQLite support with type-safe schema and migrations.

### Why SolidJS over React for the dashboard?

- **Small scope dashboard**: The dashboard is read-heavy data display (empire overview, fleet table, build queue, logs) + config forms. Not a complex web app.
- **Smaller bundle**: ~7KB vs React's ~40KB. Dashboard loads fast even on mobile.
- **Fine-grained reactivity**: When a fleet arrives, only the fleet table row updates — no re-render cascade. Perfect for real-time game state updates.
- **Familiar syntax**: TSX, hooks-like primitives. React devs productive immediately.
- **No virtual DOM**: Direct DOM updates are faster for dashboards that update frequently.

### Why Fastify over Express/Hono?

- **Fastest benchmarks**: Matters less for a personal bot but nice to have.
- **TypeScript-first**: Route handlers, request/reply types all built-in.
- **Plugin system**: Clean separation of API, WebSocket, static serving.
- **Schema validation**: Built-in JSON schema validation for API routes (can use with Zod via `fastify-type-provider-zod`).
- **`@fastify/websocket`**: Native WebSocket support for real-time dashboard updates without socket.io.

## Sources

- **ogamed** — GitHub README (github.com/alaingilbert/ogame), v53.0.0 release. REST API endpoints verified from wiki documentation. HIGH confidence.
- **Fastify** — Context7 `/fastify/fastify`, TypeScript setup, plugins, WebSocket. HIGH confidence.
- **Telegraf** — Context7 `/telegraf/telegraf`, bot setup, commands, webhook mode. HIGH confidence.
- **Drizzle ORM** — Context7 `/drizzle-team/drizzle-orm`, SQLite + better-sqlite3 setup, migrations. HIGH confidence.
- **BullMQ** — Context7 `/websites/bullmq_io`, repeatable jobs, cron scheduling. Evaluated and rejected (needs Redis). HIGH confidence.
- **node-cron** — Context7 `/node-cron/node-cron`, cron syntax scheduling. HIGH confidence.
- **SolidJS** — Context7 `/solidjs/solid`. Fine-grained reactivity, TSX syntax. HIGH confidence.
- **pino** — Training data. Fastest Node.js logger, well-established. MEDIUM confidence on latest version number.
- **ofetch** — Training data. Unjs/ofetch, built on native fetch. MEDIUM confidence on latest version.
- **pnpm** — Context7 `/websites/pnpm_io`, workspace support. HIGH confidence.

---
*Stack research for: OGame automation bot*
*Researched: 2026-04-25*
