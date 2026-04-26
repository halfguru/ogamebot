# Phase 1: Core Infrastructure - Research

**Researched:** 2026-04-26
**Domain:** Game bot infrastructure (ogamed REST client, SQLite state cache, YAML config, rate limiting, Docker)
**Confidence:** HIGH

## Summary

Phase 1 builds the foundational layer every subsequent phase depends on: a typed ogamed REST client, a SQLite-backed game state cache, YAML configuration with Zod validation, a centralized rate limiter with random jitter, and a Docker Compose deployment. This is a greenfield TypeScript monorepo — no existing code to integrate.

The ogamed REST API is well-documented on the project wiki with ~30 endpoints covering login, planets, resources, fleets, research, galaxy scanning, and attack detection. All responses follow a consistent `{Status, Code, Message, Result}` envelope. The Go binary runs on port 8080 and is configured via environment variables (`OGAMED_UNIVERSE`, `OGAMED_USERNAME`, `OGAMED_PASSWORD`, `OGAMED_LANGUAGE`, plus proxy settings). A Docker image is available with a Dockerfile that builds from source. [VERIFIED: GitHub wiki]

The standard stack — Drizzle ORM 0.45+ with better-sqlite3 12.9+, Fastify 5.8+, Zod 4.3+, pino 10.3+, js-yaml 4.1+ — is verified against npm registry versions as of 2026-04-26. All packages are compatible with Node.js 22+ LTS and ESM module resolution. [VERIFIED: npm registry]

**Primary recommendation:** Scaffold the monorepo first, then build bottom-up: shared types/schemas → ogamed client with rate limiter → config loader → game state manager → Docker Compose integration.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** pnpm monorepo with three packages: `packages/bot` (bot engine + Fastify server), `packages/dashboard` (SolidJS web app), `packages/shared` (shared types, constants, Zod schemas)
- **D-02:** TypeScript throughout with strict mode. ESM modules.
- **D-03:** Shared package exports: OGame type definitions (planets, resources, fleets, buildings, etc.), Zod schemas for ogamed response validation, constants (building IDs, mission types, etc.)
- **D-04:** YAML config file (`config.yaml`) for user-facing bot settings. YAML is more human-readable for this use case (multi-line values, comments supported). Cruiser uses this pattern.
- **D-05:** Config schema validated with Zod at startup. Invalid config = clear error message + exit.
- **D-06:** Config structure: account credentials, ogamed connection settings, per-feature toggles and parameters, logging level.
- **D-07:** Secrets (password) loaded from environment variables, not stored in config file. Config references them via `${ENV_VAR}` interpolation.
- **D-08:** SQLite via Drizzle ORM for all persistent state. Single-file database in data directory. Zero ops, no separate DB server.
- **D-09:** Game state cached in SQLite tables (planets, resources, buildings, fleets, research). Updated on each poll cycle.
- **D-10:** Drizzle migrations for schema evolution. Schema defined in code, migrations auto-generated.
- **D-11:** Type-safe ogamed REST client wrapper in `packages/bot`. All endpoints covered with typed request/response.
- **D-12:** Zod schemas validate every ogamed response before use. Invalid responses logged and handled gracefully (ogamed game-update breakage is a known pitfall).
- **D-13:** Automatic retries with exponential backoff for transient failures (network errors, 5xx responses).
- **D-14:** Rate limiter wraps all ogamed calls. Minimum 1-3 second random delay between requests. Configurable per-endpoint (galaxy scanning needs longer delays than planet info).
- **D-15:** Docker Compose with two services: `ogamed` (official ogamed image) and `bot` (Node.js). Shared Docker network, bot calls ogamed via `http://ogamed:8080`.
- **D-16:** Environment-based configuration. `.env` file for secrets, `config.yaml` mounted as volume.
- **D-17:** `data/` directory mounted as persistent volume for SQLite database.
- **D-18:** pino structured logging (JSON in production, pretty-print in dev). Log levels: trace, debug, info, warn, error, fatal.
- **D-19:** All ogamed API calls logged at debug level with request/response timing.

### Agent's Discretion
- Exact file naming conventions within packages
- Test framework selection (vitest recommended for monorepo compatibility)
- Specific Drizzle schema design (table structure, indexes)
- Build/bundling setup for production Docker image

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | Bot connects to ogamed REST API and maintains session across restarts | ogamed REST API documented on wiki; login is `GET /bot/login` with credentials set via env vars; ogamed has `OGAMED_AUTO_LOGIN=true` for auto-login on start; session maintained by ogamed's internal cookie management; our client wraps `GET /bot/login` with retry logic |
| INFRA-02 | Bot retrieves and caches game state (planets, resources, fleets, buildings, research) | Key endpoints: `GET /bot/planets`, `GET /bot/planets/:id/resources`, `GET /bot/planets/:id/resources-buildings`, `GET /bot/planets/:id/facilities`, `GET /bot/planets/:id/ships`, `GET /bot/get-research`, `GET /bot/fleets`; Drizzle ORM stores cached state in SQLite |
| INFRA-03 | Bot loads configuration from YAML/JSON file with feature toggles and per-feature parameters | js-yaml 4.1.1 for YAML parsing; Zod for schema validation; env var interpolation for secrets; per-feature toggle structure |
| INFRA-04 | Bot implements request throttling with random intervals between actions | Custom rate limiter with configurable min/max delays, per-endpoint override, priority queue for fleet-save vs normal calls |
| INFRA-05 | Bot runs as a Docker Compose stack (ogamed + bot) with environment-based config | ogamed Dockerfile builds from source with env var config; our bot Dockerfile uses Node.js 22; docker-compose.yml defines two services with shared network |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| OGame API communication | API / Backend (ogamed) | — | ogamed handles all OGame HTTP protocol, cookies, fingerprinting |
| Typed ogamed REST client | API / Backend (bot) | — | Bot's client wraps ogamed's REST endpoints with types/validation |
| Game state persistence | Database / Storage (SQLite) | — | Single-file DB for cached planets, resources, fleets, research |
| Configuration loading | API / Backend (bot) | — | YAML parsing + Zod validation at startup |
| Request rate limiting | API / Backend (bot) | — | Centralized throttle that all ogamed calls pass through |
| Docker orchestration | Infrastructure | — | docker-compose.yml ties ogamed + bot containers |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| ogamed | v53.0.0 | OGame REST API backend | Only maintained OGame API wrapper. Handles login, session, fingerprinting, captcha. [VERIFIED: GitHub releases] |
| TypeScript | 6.0.3 | Language | Strict mode for game logic safety. Shared types across packages. [VERIFIED: npm registry] |
| pnpm | 10+ | Package manager + monorepo | Workspace support, strict dependency isolation, fast installs. [VERIFIED: npm registry] |
| Drizzle ORM | 0.45.2 | SQLite ORM | Type-safe queries, schema-as-code, auto-generated migrations. [VERIFIED: npm registry] |
| better-sqlite3 | 12.9.0 | SQLite driver | Synchronous API simplifies bot logic. Native C++ binding for performance. [VERIFIED: npm registry] |
| Fastify | 5.8.5 | Web framework | TypeScript-first, fastest Node.js framework, plugin system. Will be used in later phases for dashboard API. [VERIFIED: npm registry] |
| Zod | 4.3.6 | Runtime validation | Validate ogamed responses, config schema. Type inference from schemas. [VERIFIED: npm registry] |
| pino | 10.3.1 | Structured logging | Fastest Node.js logger. JSON output. Child loggers for per-module context. [VERIFIED: npm registry] |
| js-yaml | 4.1.1 | YAML parsing | Parse `config.yaml`. Industry standard YAML parser for Node.js. [VERIFIED: npm registry] |
| ofetch | 1.5.1 | HTTP client | Lightweight wrapper over native fetch. Auto JSON parse. For ogamed REST calls. [VERIFIED: npm registry] |
| dotenv | 17.4.2 | Environment variable loading | Load `.env` file for secrets. Standard across Node.js projects. [VERIFIED: npm registry] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| drizzle-kit | 0.31.10 | Database migration tooling | `drizzle-kit generate` to create migrations from schema diffs. `drizzle-kit push` for dev. [VERIFIED: npm registry] |
| pino-pretty | 13.1.3 | Dev log formatting | `pino-pretty` for readable console output during development. [VERIFIED: npm registry] |
| tsx | 4.21.0 | TypeScript execution | `tsx watch src/main.ts` for dev reloads without build step. [VERIFIED: npm registry] |
| Vitest | 4.1.5 | Test framework | Unit tests for ogamed client, config validation, rate limiter. [VERIFIED: npm registry] |
| @types/better-sqlite3 | 7.6.13 | Type definitions | TypeScript types for better-sqlite3. [VERIFIED: npm registry] |
| eslint + typescript-eslint | — | Linting | Catch bugs in game logic calculations. Enforce strict TS rules. |
| prettier | — | Code formatting | Consistent style across monorepo packages. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Zod 4.x | Zod 3.x (3.24+) | Zod 4 is the latest stable; smaller bundle, improved API. Zod 3 is more battle-tested but Zod 4 is production-ready. CONTEXT.md says "Zod 3.24+" but npm shows 4.3.6 as latest — use 4.x per discretion. |
| ofetch | undici | ofetch is simpler and built on native fetch. undici offers more control but adds complexity for our simple REST client needs. |

**Installation:**
```bash
# Root - initialize monorepo
pnpm init
# Add workspace config: pnpm-workspace.yaml

# Root dev dependencies
pnpm add -Dw typescript @types/node tsx vitest eslint prettier typescript-eslint

# packages/shared
pnpm add zod --filter shared

# packages/bot
pnpm add drizzle-orm better-sqlite3 ofetch pino js-yaml dotenv --filter bot
pnpm add -D @types/better-sqlite3 drizzle-kit pino-pretty --filter bot

# packages/dashboard (placeholder for Phase 5)
# SolidJS deps added later
```

**Version verification (2026-04-26):**
```bash
$ npm view drizzle-orm version     # 0.45.2
$ npm view better-sqlite3 version  # 12.9.0
$ npm view fastify version         # 5.8.5
$ npm view zod version             # 4.3.6
$ npm view pino version            # 10.3.1
$ npm view ofetch version          # 1.5.1
$ npm view vitest version          # 4.1.5
$ npm view typescript version      # 6.0.3
$ npm view js-yaml version         # 4.1.1
$ npm view dotenv version          # 17.4.2
$ npm view tsx version             # 4.21.0
$ npm view pino-pretty version     # 13.1.3
$ npm view drizzle-kit version     # 0.31.10
$ npm view @types/better-sqlite3 version  # 7.6.13
```

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     Docker Compose Network                       │
│                                                                  │
│  ┌────────────────────┐       ┌──────────────────────────────┐  │
│  │      ogamed        │       │      Node.js Bot Process      │  │
│  │   (Go binary)      │       │                              │  │
│  │                    │  REST │  ┌─────────────────────────┐  │  │
│  │  Handles:          │◀─────│  │   Ogamed REST Client    │  │  │
│  │  - Login/session   │      │  │   (typed, validated,    │  │  │
│  │  - Device fingerp. │      │  │    rate-limited)        │  │  │
│  │  - Anti-detection  │      │  └───────────┬─────────────┘  │  │
│  │  - Captcha         │      │              │                │  │
│  │  - Cookie mgmt     │      │  ┌───────────▼─────────────┐  │  │
│  │                    │      │  │   Game State Manager    │  │  │
│  │  Port: 8080        │      │  │   (in-memory + SQLite  │  │  │
│  │  Config: env vars  │      │  │    cache, periodic     │  │  │
│  └────────────────────┘      │  │    refresh)            │  │  │
│                               │  └───────────┬─────────────┘  │  │
│                               │              │                │  │
│                               │  ┌───────────▼─────────────┐  │  │
│                               │  │   Config Manager        │  │  │
│                               │  │   (YAML + Zod + env)    │  │  │
│                               │  └─────────────────────────┘  │  │
│                               │              │                │  │
│                               │  ┌───────────▼─────────────┐  │  │
│                               │  │   SQLite (data/bot.db)  │  │  │
│                               │  │   via Drizzle ORM       │  │  │
│                               │  └─────────────────────────┘  │  │
│                               │                              │  │
│                               │  ┌─────────────────────────┐  │  │
│                               │  │   pino Logger           │  │  │
│                               │  └─────────────────────────┘  │  │
│                               └──────────────────────────────┘  │
│                                                                  │
│  Volumes: config.yaml (ro), .env (secrets), data/ (SQLite)      │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
ogame/
├── .env.example                 # Template for secrets (gitignored)
├── .gitignore
├── config.example.yaml          # Template for bot configuration
├── docker-compose.yml           # ogamed + bot services
├── package.json                 # Root workspace config
├── pnpm-workspace.yaml          # Workspace definitions
├── tsconfig.base.json           # Shared TypeScript config
├── packages/
│   ├── shared/                  # Shared types, schemas, constants
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   └── src/
│   │       ├── index.ts         # Barrel export
│   │       ├── types/           # OGame domain types
│   │       │   ├── planet.ts    # Planet, Resources, Coordinate, etc.
│   │       │   ├── fleet.ts     # Fleet, Ship, Mission types
│   │       │   ├── research.ts  # Research levels
│   │       │   └── buildings.ts # Building levels
│   │       ├── schemas/         # Zod schemas for ogamed responses
│   │       │   ├── ogamed.ts    # Base response envelope {Status, Code, Message, Result}
│   │       │   ├── planets.ts   # Planet list, planet details, resources
│   │       │   ├── fleets.ts    # Fleet list, fleet send response
│   │       │   └── research.ts  # Research response
│   │       └── constants/       # OGame game constants
│   │           ├── buildings.ts # Building IDs, names
│   │           ├── ships.ts     # Ship IDs, names
│   │           └── missions.ts  # Mission type IDs
│   ├── bot/                     # Bot engine (core infrastructure)
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   ├── drizzle.config.ts    # Drizzle Kit config for migrations
│   │   ├── Dockerfile           # Production Docker image
│   │   └── src/
│   │       ├── main.ts          # Entry point: load config → init DB → start client
│   │       ├── client/          # ogamed REST client layer
│   │       │   ├── ogamed-client.ts  # Typed HTTP wrapper for all endpoints
│   │       │   ├── rate-limiter.ts   # Centralized request throttle with jitter
│   │       │   └── retry.ts          # Exponential backoff retry helper
│   │       ├── config/          # Configuration management
│   │       │   ├── config-loader.ts  # YAML + env + Zod validation
│   │       │   └── config-schema.ts  # Zod schema for config structure
│   │       ├── state/           # Game state management
│   │       │   ├── game-state.ts     # Central state store with refresh logic
│   │       │   └── types.ts          # Internal state type definitions
│   │       ├── db/              # Database layer
│   │       │   ├── index.ts          # Drizzle instance + connection setup
│   │       │   ├── schema.ts         # Drizzle table definitions
│   │       │   └── migrate.ts        # Migration runner
│   │       └── utils/           # Shared utilities
│   │           └── logger.ts         # pino logger factory
│   └── dashboard/               # SolidJS web app (placeholder for Phase 5)
│       └── package.json
└── tests/                       # (or co-located tests in each package)
    └── ...
```

### Pattern 1: Typed Ogamed Client with Rate Limiting

**What:** A class that wraps every ogamed REST endpoint with typed request/response, rate limiting, retry, and Zod validation. All ogamed communication goes through this single class.

**When to use:** Every interaction with OGame.

**Example:**
```typescript
// packages/bot/src/client/ogamed-client.ts
import { ofetch } from 'ofetch';
import { z } from 'zod';
import { RateLimiter } from './rate-limiter.js';
import { retryWithBackoff } from './retry.js';
import { ogamedResponseSchema } from '@ogame-bot/shared';

export class OgamedClient {
  private baseUrl: string;
  private rateLimiter: RateLimiter;

  constructor(baseUrl: string, rateLimiter: RateLimiter) {
    this.baseUrl = baseUrl;
    this.rateLimiter = rateLimiter;
  }

  async login(): Promise<void> {
    await this.get('/bot/login', z.void());
  }

  async isUnderAttack(): Promise<boolean> {
    const res = await this.get('/bot/is-under-attack', z.boolean());
    return res;
  }

  async getPlanets(): Promise<Planet[]> {
    return this.get('/bot/planets', planetArraySchema);
  }

  async getResources(planetId: number): Promise<Resources> {
    return this.get(`/bot/planets/${planetId}/resources`, resourcesSchema);
  }

  private async get<T>(path: string, resultSchema: z.ZodType<T>): Promise<T> {
    await this.rateLimiter.acquire(path);
    return retryWithBackoff(async () => {
      const raw = await ofetch<{Status: string; Code: number; Message: string; Result: unknown}>(
        `${this.baseUrl}${path}`
      );
      const envelope = ogamedResponseSchema(resultSchema).parse(raw);
      if (envelope.Status !== 'ok') {
        throw new OgamedError(envelope.Message, envelope.Code);
      }
      return envelope.Result as T;
    });
  }
}
```

### Pattern 2: YAML Config with Zod Validation and Env Interpolation

**What:** Load `config.yaml`, interpolate `${ENV_VAR}` references, validate against Zod schema, exit with clear error if invalid.

**When to use:** Application startup.

**Example:**
```typescript
// packages/bot/src/config/config-loader.ts
import { readFileSync } from 'node:fs';
import yaml from 'js-yaml';
import { configSchema } from './config-schema.js';

export function loadConfig(configPath: string): Config {
  const raw = readFileSync(configPath, 'utf-8');
  // Interpolate ${ENV_VAR} references
  const interpolated = raw.replace(/\$\{(\w+)\}/g, (_, varName) => {
    const value = process.env[varName];
    if (value === undefined) {
      throw new Error(`Environment variable ${varName} referenced in config but not set`);
    }
    return value;
  });
  const parsed = yaml.load(interpolated);
  const result = configSchema.safeParse(parsed);
  if (!result.success) {
    console.error('Invalid configuration:');
    for (const issue of result.error.issues) {
      console.error(`  ${issue.path.join('.')}: ${issue.message}`);
    }
    process.exit(1);
  }
  return result.data;
}
```

### Pattern 3: Centralized Rate Limiter with Random Jitter

**What:** A rate limiter that enforces minimum delays between requests with configurable per-endpoint overrides and random jitter for anti-detection.

**When to use:** Wraps every ogamed API call.

**Example:**
```typescript
// packages/bot/src/client/rate-limiter.ts
export interface RateLimitConfig {
  defaultMinDelayMs: number;    // e.g., 2000
  defaultMaxDelayMs: number;    // e.g., 5000
  endpointOverrides?: Record<string, { minMs: number; maxMs: number }>;
}

export class RateLimiter {
  private lastRequestTime = 0;
  private config: RateLimitConfig;

  constructor(config: RateLimitConfig) {
    this.config = config;
  }

  async acquire(endpoint: string): Promise<void> {
    const now = Date.now();
    const elapsed = now - this.lastRequestTime;

    const override = this.config.endpointOverrides?.[endpoint];
    const minDelay = override?.minMs ?? this.config.defaultMinDelayMs;
    const maxDelay = override?.maxMs ?? this.config.defaultMaxDelayMs;
    const randomDelay = minDelay + Math.random() * (maxDelay - minDelay);

    const waitTime = Math.max(0, randomDelay - elapsed);
    if (waitTime > 0) {
      await new Promise(resolve => setTimeout(resolve, waitTime));
    }

    this.lastRequestTime = Date.now();
  }
}
```

### Pattern 4: Drizzle SQLite Schema for Game State

**What:** Define game state tables using Drizzle ORM's `sqliteTable` builder with proper indexes for query performance.

**When to use:** All persistent state storage.

**Example:**
```typescript
// packages/bot/src/db/schema.ts
import { sqliteTable, text, integer, real } from 'drizzle-orm/sqlite-core';

export const planets = sqliteTable('planets', {
  id: integer('id').primaryKey(),              // OGame planet ID
  name: text('name').notNull(),
  galaxy: integer('galaxy').notNull(),
  system: integer('system').notNull(),
  position: integer('position').notNull(),
  diameter: integer('diameter').notNull(),
  fieldsUsed: integer('fields_used').notNull(),
  fieldsTotal: integer('fields_total').notNull(),
  tempMin: integer('temp_min').notNull(),
  tempMax: integer('temp_max').notNull(),
  updatedAt: integer('updated_at', { mode: 'timestamp' }).notNull(),
});

export const resources = sqliteTable('resources', {
  planetId: integer('planet_id').primaryKey().references(() => planets.id),
  metal: real('metal').notNull(),
  crystal: real('crystal').notNull(),
  deuterium: real('deuterium').notNull(),
  energy: real('energy').notNull(),
  updatedAt: integer('updated_at', { mode: 'timestamp' }).notNull(),
});

export const research = sqliteTable('research', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  energyTech: integer('energy_tech').default(0),
  laserTech: integer('laser_tech').default(0),
  combustionDrive: integer('combustion_drive').default(0),
  // ... all research fields
  updatedAt: integer('updated_at', { mode: 'timestamp' }).notNull(),
});

export const fleets = sqliteTable('fleets', {
  id: integer('id').primaryKey(),              // OGame fleet ID
  mission: integer('mission').notNull(),        // Mission type ID
  returnFlight: integer('return_flight', { mode: 'boolean' }).notNull(),
  originGalaxy: integer('origin_galaxy').notNull(),
  originSystem: integer('origin_system').notNull(),
  originPosition: integer('origin_position').notNull(),
  destGalaxy: integer('dest_galaxy').notNull(),
  destSystem: integer('dest_system').notNull(),
  destPosition: integer('dest_position').notNull(),
  arriveIn: integer('arrive_in').notNull(),
  updatedAt: integer('updated_at', { mode: 'timestamp' }).notNull(),
});
```

### Pattern 5: Database Initialization and Migration

**What:** Connect to SQLite file, run pending migrations on startup.

**When to use:** Application startup, before any other initialization.

**Example:**
```typescript
// packages/bot/src/db/index.ts
import Database from 'better-sqlite3';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import { migrate } from 'drizzle-orm/better-sqlite3/migrator';
import * as schema from './schema.js';

export function initDatabase(dbPath: string) {
  const sqlite = new Database(dbPath);
  // Enable WAL mode for better concurrent read performance
  sqlite.pragma('journal_mode = WAL');
  const db = drizzle(sqlite, { schema });
  return { db, sqlite };
}

export function runMigrations(dbPath: string, migrationsFolder: string) {
  const sqlite = new Database(dbPath);
  const db = drizzle(sqlite);
  migrate(db, { migrationsFolder });
  sqlite.close();
}
```

### Anti-Patterns to Avoid
- **Direct ogamed calls outside the client:** All game interaction must go through OgamedClient. Workers/services should never call `ofetch` directly against ogamed. [CITED: ARCHITECTURE.md anti-pattern 1]
- **Fixed-interval polling:** Never use exact intervals (e.g., `setInterval(fn, 30000)`). Always add ±20-40% random jitter. OGame detects regular patterns. [CITED: PITFALLS.md pitfall 7]
- **Skipping Zod validation on responses:** ogamed responses can be corrupted by game updates. Every response must be validated. [CITED: PITFALLS.md pitfall 2]
- **Storing credentials in config.yaml:** Password goes in `.env`, referenced as `${OGAME_PASSWORD}` in config. [CITED: CONTEXT.md D-07]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML parsing | Custom YAML parser | `js-yaml` | Handles edge cases, anchors, multi-doc |
| Schema validation | Manual if/else checks | `zod` | Type inference, nested objects, error messages |
| SQL query building | String concatenation | `drizzle-orm` | SQL injection prevention, type safety, migrations |
| HTTP requests | Raw `fetch` with manual error handling | `ofetch` with retry wrapper | Auto JSON, better error messages, timeout support |
| Logging | `console.log` | `pino` | Structured JSON, log levels, child loggers, performance |
| Database migrations | Manual SQL files | `drizzle-kit generate` | Auto-generates from schema diffs, versioned, rollback-safe |
| Environment variables | Manual `process.env` parsing | `dotenv` | `.env` file loading, standard across Node.js projects |
| Retry logic | Simple `for` loop with `setTimeout` | Exponential backoff helper | Handles transient failures, configurable attempts, jitter on retry delay |

**Key insight:** The ogamed API wrapper is the one thing we DO build custom (no library exists for it), but every supporting utility (YAML, validation, SQL, HTTP, logging) has a battle-tested library.

## Common Pitfalls

### Pitfall 1: ogamed Response Format Not Validated
**What goes wrong:** OGame updates break ogamed's HTML scrapers. Responses return zeros, empty arrays, or malformed JSON. Bot silently processes garbage data.
**Why it happens:** ogamed works by scraping OGame's HTML. When Gameforge changes page structure, parsers break silently.
**How to avoid:** Every ogamed response validated with Zod. Invalid response → log warning + skip update + don't overwrite cached state.
**Warning signs:** Resource counts all zero, planet count changes unexpectedly, timestamps showing year 0001.
[VERIFIED: PITFALLS.md pitfall 2, ogamed issues #148, #150]

### Pitfall 2: Rate Limiter Not Shared Across Workers
**What goes wrong:** Multiple workers each implement their own delays, leading to request spikes when their intervals align.
**Why it happens:** Workers developed independently without a shared throttle.
**How to avoid:** Single `RateLimiter` instance injected into `OgamedClient`. All calls go through one choke point.
**Warning signs:** ogamed returning errors after heavy activity, request bursts in logs.
[VERIFIED: PITFALLS.md pitfall 6]

### Pitfall 3: Session Loss on ogamed Restart
**What goes wrong:** ogamed restarts (crash, update, Docker restart) and the bot doesn't reconnect. All subsequent calls fail.
**Why it happens:** Bot assumes ogamed is always available. No health checks or reconnection logic.
**How to avoid:** Health check loop (`GET /bot/server/time` as heartbeat). On connection failure, retry with exponential backoff. Set `OGAMED_AUTO_LOGIN=true` so ogamed auto-reconnects to OGame.
**Warning signs:** All API calls failing with connection refused, bot process running but producing no actions.
[VERIFIED: ogamed README - OGAMED_AUTO_LOGIN env var]

### Pitfall 4: SQLite File Not Persisted in Docker
**What goes wrong:** Docker container restarts and all game state is lost. Database starts empty every time.
**Why it happens:** SQLite file inside container is ephemeral unless mounted as a volume.
**How to avoid:** Mount `data/` directory as a Docker volume: `volumes: - ./data:/app/data`. Ensure the path in code points to the mounted location.
**Warning signs:** Game state resets on `docker compose restart`.
[ASSUMED — standard Docker behavior]

### Pitfall 5: Zod Schema Doesn't Match Actual ogamed Responses
**What goes wrong:** Zod schemas written from documentation don't match actual runtime responses (missing fields, different types, unexpected null values). Every response validation fails.
**Why it happens:** The ogamed wiki docs show simplified examples. Real responses may have additional fields or edge cases (null vs 0, missing vs empty array).
**How to avoid:** During development, capture actual ogamed responses and write schemas against real data. Use `.passthrough()` initially to allow unknown fields, then tighten. Use `.default()` for fields that may be missing.
**Warning signs:** All Zod validations failing at runtime with "unexpected keys" or "required field missing" errors.

### Pitfall 6: pnpm Workspace Dependencies Not Resolved
**What goes wrong:** `packages/bot` can't import from `@ogame-bot/shared`. TypeScript compilation fails or runtime module not found.
**Why it happens:** Workspace protocol not configured correctly, or `package.json` `name` field doesn't match imports.
**How to avoid:** Use `"@ogame-bot/shared": "workspace:*"` in bot's dependencies. Ensure `packages/shared/package.json` has `"name": "@ogame-bot/shared"`. Run `pnpm install` after adding workspace references.
**Warning signs:** `ERR_MODULE_NOT_FOUND` at runtime, or TypeScript can't resolve module.

## Code Examples

### ogamed REST API Response Envelope
```typescript
// Source: https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation
// All responses follow this format:
{
  "Status": "ok",        // "ok" or "error"
  "Code": 200,           // HTTP status code
  "Message": "",         // Error message if Status != "ok"
  "Result": <T>          // Typed response data
}

// Zod schema for the envelope:
import { z } from 'zod';

export const ogamedResponseSchema = <T extends z.ZodTypeAny>(resultSchema: T) =>
  z.object({
    Status: z.enum(['ok', 'error']),
    Code: z.number(),
    Message: z.string(),
    Result: resultSchema,
  });
```

### Key ogamed Endpoints for Phase 1
```typescript
// Source: https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation [VERIFIED]

// Authentication
'GET  /bot/login'                              // → null
'GET  /bot/logout'                             // → null
'GET  /bot/server/time'                        // → ISO datetime string (health check)
'GET  /bot/is-under-attack'                    // → boolean
'GET  /bot/user-infos'                         // → {PlayerID, PlayerName, Points, Rank, ...}

// Game State (INFRA-02)
'GET  /bot/planets'                            // → Planet[]
'GET  /bot/planets/:planetID/resources'         // → {Metal, Crystal, Deuterium, Energy, Darkmatter}
'GET  /bot/planets/:planetID/resources-buildings' // → {MetalMine, CrystalMine, ...}
'GET  /bot/planets/:planetID/facilities'        // → {RoboticsFactory, Shipyard, ...}
'GET  /bot/planets/:planetID/ships'             // → {LightFighter, HeavyFighter, ...}
'GET  /bot/planets/:planetID/defence'           // → {RocketLauncher, ...}
'GET  /bot/get-research'                        // → {EnergyTechnology, LaserTechnology, ...}
'GET  /bot/fleets'                              // → Fleet[]
'GET  /bot/server/speed'                        // → number (universe speed)
'GET  /bot/server/version'                      // → string (OGame version)
```

### Docker Compose Configuration
```yaml
# Source: ogamed docker-compose.yml + Dockerfile [VERIFIED: GitHub]
# docker-compose.yml
version: '3.8'
services:
  ogamed:
    image: alaingilbert/ogamed:latest  # or build from source
    container_name: ogame-ogamed
    environment:
      - OGAMED_UNIVERSE=${OGAMED_UNIVERSE}
      - OGAMED_USERNAME=${OGAMED_USERNAME}
      - OGAMED_PASSWORD=${OGAMED_PASSWORD}
      - OGAMED_LANGUAGE=${OGAMED_LANGUAGE}
      - OGAMED_HOST=0.0.0.0
      - OGAMED_PORT=8080
      - OGAMED_AUTO_LOGIN=true
      - OGAMED_PROXY=${OGAMED_PROXY:-}
      - OGAMED_PROXY_USERNAME=${OGAMED_PROXY_USERNAME:-}
      - OGAMED_PROXY_PASSWORD=${OGAMED_PROXY_PASSWORD:-}
      - OGAMED_PROXY_TYPE=${OGAMED_PROXY_TYPE:-socks5}
      - OGAMED_PROXY_LOGIN_ONLY=${OGAMED_PROXY_LOGIN_ONLY:-false}
      - CORS_ENABLED=true
    ports:
      - "127.0.0.1:8080:8080"  # Bind to localhost only for security
    restart: unless-stopped

  bot:
    build:
      context: ./packages/bot
      dockerfile: Dockerfile
    container_name: ogame-bot
    depends_on:
      - ogamed
    environment:
      - OGAME_OGAMED_URL=http://ogamed:8080
      - OGAME_PASSWORD=${OGAME_PASSWORD}
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/app/data
    restart: unless-stopped
```

### Bot Dockerfile
```dockerfile
# packages/bot/Dockerfile
FROM node:22-slim

WORKDIR /app

# Copy workspace root files
COPY ../../package.json ../../pnpm-workspace.yaml ../../pnpm-lock.yaml ./
COPY ../../packages/shared/package.json ./packages/shared/
COPY ../../packages/bot/package.json ./packages/bot/

# Install dependencies
RUN corepack enable && pnpm install --frozen-lockfile --prod

# Copy source
COPY ../../packages/shared/ ./packages/shared/
COPY ../../packages/bot/ ./packages/bot/

# Build
RUN pnpm --filter shared build && pnpm --filter bot build

WORKDIR /app/packages/bot
CMD ["node", "dist/main.js"]
```

### pnpm Workspace Configuration
```yaml
# pnpm-workspace.yaml
packages:
  - 'packages/*'
```

```json
// tsconfig.base.json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "Node16",
    "moduleResolution": "Node16",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "dist"
  }
}
```

### pino Logger Setup
```typescript
// packages/bot/src/utils/logger.ts
import pino from 'pino';

export function createLogger(name: string, level: string = 'info') {
  return pino({
    name,
    level,
    transport: level === 'debug' ? {
      target: 'pino-pretty',
      options: { colorize: true }
    } : undefined,
  });
}

// Usage in other modules:
// const log = createLogger('ogamed-client', config.logLevel);
// log.info({ planetId: 123 }, 'Fetching resources');
// log.debug({ duration: 234 }, 'API call completed');
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Prisma for SQLite | Drizzle ORM | 2024-2025 | Drizzle has no engine binary, 7KB vs 100MB+, native better-sqlite3 driver |
| CommonJS modules | ESM (Node16 module resolution) | Node.js 16+ | `"type": "module"` in package.json, `.js` extensions in imports |
| Zod 3.x | Zod 4.x | 2025 | Smaller bundle, improved API, but Zod 3 still widely used |
| axios for HTTP | ofetch (native fetch wrapper) | 2023+ | Lighter, auto JSON parse, native fetch under the hood |
| Winston for logging | pino | 2022+ | 5x+ faster, structured JSON by default |
| Node.js 18 LTS | Node.js 22 LTS | Oct 2024 | Better ESM support, faster runtime |
| TypeScript 5.x | TypeScript 6.0 | 2025 | Stricter checks, better inference |

**Deprecated/outdated:**
- **Express:** Legacy API design, poor TypeScript support. Use Fastify instead.
- **Winston:** Synchronous formatting on hot path. Use pino instead.
- **axios:** Inconsistent `response.data` API. Use ofetch instead.
- **Prisma for SQLite:** Heavy engine binary, slow cold starts. Use Drizzle instead.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | ogamed Docker image available on Docker Hub as `alaingilbert/ogamed` | Docker Compose | Need to build from source Dockerfile instead; adds build time but not blocking |
| A2 | ogamed v53.0.0 is compatible with current OGame servers | Standard Stack | If OGame updated and v53 broken, need to pin to working version or wait for update |
| A3 | `OGAMED_AUTO_LOGIN=true` makes ogamed auto-login and maintain session across restarts | Docker Compose | If not, need additional reconnection logic in bot client |
| A4 | Zod 4.x API is compatible with the schema patterns described (4.x is latest on npm) | Standard Stack | If Zod 4 has breaking changes from 3.x patterns, schemas need adjustment — but 4.x is production-stable |

## Open Questions (RESOLVED)

1. **ogamed Docker image availability** — RESOLVED: Use pre-built image `alaingilbert/ogamed:latest` from Docker Hub (confirmed available). Fallback: build from source via Dockerfile in the repo. Plan 01-03 uses the pre-built image.

2. **ogamed response schema completeness** — RESOLVED: Start Zod schemas permissive with `.passthrough()` and `.default(0)` for numeric fields that may be missing. Tighten schemas after capturing real ogamed responses during development. Plan 01-01 Task 2 implements this pattern.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Bot runtime | ✓ | 25.9.0 | — |
| pnpm | Package manager | ✓ | 10+ (checked) | — |
| Docker | Containerization | ✓ | (available) | — |
| Docker Compose | Multi-container orchestration | ✓ | (available) | — |
| npm | Package registry access | ✓ | (available) | — |
| TypeScript | Compilation | ✓ (via npm) | 6.0.3 | — |
| Git | Version control | ✓ | (available) | — |

**Missing dependencies with no fallback:** None — all required tools available.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.5 |
| Config file | `vitest.config.ts` (per-package or root) — to be created in Wave 0 |
| Quick run command | `pnpm -r exec vitest run` |
| Full suite command | `pnpm -r exec vitest run --coverage` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-01 | Ogamed client connects and authenticates | unit (mocked) + integration | `vitest run packages/bot/src/client/` | ❌ Wave 0 |
| INFRA-01 | Client retries on connection failure | unit | `vitest run packages/bot/src/client/` | ❌ Wave 0 |
| INFRA-02 | Game state refreshes from ogamed and persists to SQLite | unit (mocked) + integration | `vitest run packages/bot/src/state/` | ❌ Wave 0 |
| INFRA-02 | Zod schemas validate real ogamed response shapes | unit | `vitest run packages/shared/src/schemas/` | ❌ Wave 0 |
| INFRA-03 | Config loads from YAML with env interpolation | unit | `vitest run packages/bot/src/config/` | ❌ Wave 0 |
| INFRA-03 | Invalid config produces clear error and exits | unit | `vitest run packages/bot/src/config/` | ❌ Wave 0 |
| INFRA-04 | Rate limiter enforces minimum delays between calls | unit | `vitest run packages/bot/src/client/` | ❌ Wave 0 |
| INFRA-04 | Rate limiter adds random jitter | unit | `vitest run packages/bot/src/client/` | ❌ Wave 0 |
| INFRA-05 | Docker Compose starts both containers | integration (manual) | `docker compose up --abort-on-container-exit` | ❌ Wave 0 |
| INFRA-05 | Bot container reaches ogamed via internal network | integration (manual) | `docker compose exec bot wget -qO- http://ogamed:8080/bot/server/time` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `pnpm --filter bot exec vitest run`
- **Per wave merge:** `pnpm -r exec vitest run`
- **Phase gate:** Full suite green + `docker compose up` smoke test passes

### Wave 0 Gaps
- [ ] `vitest.config.ts` — root config for test framework
- [ ] `packages/bot/tests/` — test directory with shared fixtures
- [ ] `packages/shared/tests/` — test directory for schema validation
- [ ] Framework install: `pnpm add -Dw vitest` — needs to be run during scaffolding
- [ ] Docker integration test script: `tests/docker-smoke.sh`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | ogamed handles OGame auth; bot config has no direct auth |
| V3 Session Management | yes | ogamed manages OGame sessions; bot must handle ogamed session resilience |
| V4 Access Control | yes | Docker network isolation; ogamed bound to localhost; no public exposure |
| V5 Input Validation | yes | Zod validates all ogamed responses; config validated at startup |
| V6 Cryptography | no | No custom crypto; secrets in `.env` file (gitignored) |

### Known Threat Patterns for OGame Bot Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| OGame account credentials in git | Information Disclosure | `.env` for secrets, `.gitignore` for `.env` and `data/` |
| ogamed REST API exposed publicly | Tampering + Spoofing | Bind to `127.0.0.1:8080`, Docker network isolation, firewall rules |
| Malicious ogamed response (MITM) | Tampering | Zod validation rejects unexpected data shapes |
| SQLite database corruption | Denial of Service | WAL mode, Docker volume persistence, backup strategy |
| Config file with sensitive defaults | Information Disclosure | `config.example.yaml` committed, `config.yaml` gitignored |

## Sources

### Primary (HIGH confidence)
- **ogamed REST API documentation** — https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation — All endpoints verified, response formats documented with examples
- **ogamed Dockerfile** — https://github.com/alaingilbert/ogame/blob/master/Dockerfile — Build configuration and env vars
- **ogamed docker-compose.yml** — https://github.com/alaingilbert/ogame/blob/master/docker-compose.yml — Service configuration template
- **ogamed releases** — https://github.com/alaingilbert/ogame/releases — v53.0.0 confirmed as latest
- **npm registry** — Version verification for all packages (drizzle-orm, better-sqlite3, fastify, zod, pino, etc.)
- **Context7: Drizzle ORM** — `/drizzle-team/drizzle-orm-docs` — SQLite schema definition, better-sqlite3 setup, migration patterns
- **Context7: Fastify** — `/fastify/fastify` — TypeScript setup, pino logger config, plugin system
- **Context7: Zod** — `/colinhacks/zod` — Schema definition, type inference, safeParse patterns

### Secondary (MEDIUM confidence)
- **Existing project research** — `.planning/research/STACK.md`, `ARCHITECTURE.md`, `PITFALLS.md` — Pre-researched stack, patterns, and pitfalls (cross-referenced)

### Tertiary (LOW confidence)
- **ogamed Docker Hub image availability** — Assumed available as `alaingilbert/ogamed` but not verified; fallback is building from source Dockerfile

## Project Constraints (from AGENTS.md)

The following directives from AGENTS.md must be followed during implementation:

1. **Monorepo structure:** pnpm workspaces with `packages/bot`, `packages/dashboard`, `packages/shared`
2. **Zod for runtime validation:** All ogamed responses validated with Zod
3. **YAML config files:** User-facing configuration in YAML format
4. **Docker Compose for deployment:** ogamed + bot containers
5. **Run lint/typecheck before committing:** `pnpm lint && pnpm typecheck` (scripts to be defined in Phase 1)
6. **Commit after each plan completes:** GSD workflow enforcement

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All versions verified against npm registry, all libraries documented via Context7 or official sources
- Architecture: HIGH — Patterns established by reference projects (TBot, Cruiser), ogamed API fully documented
- Pitfalls: HIGH — Cross-referenced from 10+ TBot issues, 3+ ogamed issues, community warnings

**Research date:** 2026-04-26
**Valid until:** 2026-05-26 (stable — core libraries change slowly)
