# Phase 1: Core Infrastructure - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-26
**Phase:** 01-core-infrastructure
**Areas discussed:** Project structure, Configuration, State storage, ogamed client, Docker, Logging

---

## Project Structure

| Option | Description | Selected |
|--------|-------------|----------|
| pnpm monorepo (3 packages) | packages/bot, packages/dashboard, packages/shared | ✓ |
| Flat structure | Single package with all code | |
| Turborepo monorepo | Turborepo for build orchestration | |

**User's choice:** Auto-decided (pnpm monorepo — research recommendation)
**Notes:** Shared package for types/schemas enables type safety across bot + dashboard

---

## Configuration Format

| Option | Description | Selected |
|--------|-------------|----------|
| YAML | Human-readable, supports comments, Cruiser pattern | ✓ |
| JSON | Simpler parsing, TBot pattern | |
| TOML | Emerging standard, less common in Node.js | |

**User's choice:** Auto-decided (YAML — research recommendation, human-friendly for bot config)
**Notes:** Secrets via environment variable interpolation, not in config file

---

## State Storage

| Option | Description | Selected |
|--------|-------------|----------|
| SQLite + Drizzle ORM | Zero-ops, single-file DB, type-safe queries | ✓ |
| In-memory only | Fastest, but lost on restart | |
| PostgreSQL + Prisma | Full DB, overkill for single-user bot | |

**User's choice:** Auto-decided (SQLite — research recommendation for single-user bot)
**Notes:** Drizzle is 7KB vs Prisma's 100MB+. better-sqlite3 synchronous API simplifies bot logic.

---

## ogamed Client Design

| Option | Description | Selected |
|--------|-------------|----------|
| Type-safe wrapper + Zod validation | Validates every response, handles breakage gracefully | ✓ |
| Thin HTTP wrapper | Simple fetch calls, manual type casting | |
| Auto-generated client from OpenAPI | Most correct, but ogamed has no OpenAPI spec | |

**User's choice:** Auto-decided (Zod-validated wrapper — pitfalls research shows game updates silently break responses)
**Notes:** Exponential backoff retries + rate limiter wrapping all calls

---

## Docker Networking

| Option | Description | Selected |
|--------|-------------|----------|
| Shared Docker network | Bot calls ogamed by service name | ✓ |
| Host network | Both on host network, localhost communication | |
| Sidecar pattern | ogamed embedded in same container | |

**User's choice:** Auto-decided (shared network — standard Docker Compose pattern)
**Notes:** ogamed official image + custom bot image, data/ volume for SQLite persistence

---

## Logging

| Option | Description | Selected |
|--------|-------------|----------|
| pino structured logging | Fastest Node.js logger, JSON output, 5x faster than Winston | ✓ |
| Winston | Most popular, heavier | |
| console.log | No structured logging | |

**User's choice:** Auto-decided (pino — research recommendation)
**Notes:** Pretty-print in dev, JSON in production. All API calls logged at debug level.

---

## Agent's Discretion

- Exact file naming conventions within packages
- Test framework selection
- Specific Drizzle schema design
- Build/bundling setup for production Docker image
