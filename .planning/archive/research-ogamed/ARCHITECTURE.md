# Architecture Research

**Domain:** OGame automation bot (game bot + REST backend + web dashboard)
**Researched:** 2026-04-25
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Presentation Layer                               │
│  ┌─────────────────────┐  ┌──────────────────────┐                      │
│  │   Web Dashboard     │  │  Telegram Bot         │                      │
│  │   (Next.js/React)   │  │  (node-telegram-bot)  │                      │
│  └──────────┬──────────┘  └───────────┬──────────┘                      │
│             │ REST/WebSocket           │                                  │
├─────────────┴─────────────────────────┴──────────────────────────────────┤
│                         Bot Engine (Node.js)                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  Defender    │  │  AutoBuild   │  │  AutoFarm    │  │  Expeditions │ │
│  │  Worker      │  │  Worker      │  │  Worker      │  │  Worker      │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │
│         │                 │                  │                  │         │
│  ┌──────┴─────────────────┴──────────────────┴──────────────────┴──────┐ │
│  │                      Scheduler / Event Bus                          │ │
│  └──────────────────────────────┬──────────────────────────────────────┘ │
│                                 │                                         │
│  ┌──────────────────────────────┴──────────────────────────────────────┐ │
│  │                    Game State Manager                                │ │
│  │  (cached planets, resources, fleets, research, build queues)        │ │
│  └──────────────────────────────┬──────────────────────────────────────┘ │
│                                 │                                         │
├─────────────────────────────────┴────────────────────────────────────────┤
│                      OGame Client Layer (TypeScript)                     │
│  ┌──────────────────────────────────────────────────────────────────────┐│
│  │                    ogamed REST Client                                ││
│  │  (typed wrapper around ogamed HTTP API, rate limiting, retries)     ││
│  └──────────────────────────────┬──────────────────────────────────────┘│
├─────────────────────────────────┴────────────────────────────────────────┤
│                      External: ogamed (Go binary, one per account)       │
│  ┌──────────────────────────────────────────────────────────────────────┐│
│  │  Handles: login, sessions, device fingerprinting, anti-detection,   ││
│  │  captcha, cookie management, HTTP to OGame servers                   ││
│  └──────────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────────┘
         │
         │  (ogamed communicates directly with OGame servers)
         ▼
   ┌───────────────┐
   │  OGame Servers │
   └───────────────┘
```

### Component Responsibilities

| Component | Responsibility | Implementation |
|-----------|----------------|----------------|
| **ogamed** | OGame protocol, anti-detection, session mgmt, captcha | External Go binary (alaingilbert/ogame), one process per account |
| **ogamed REST Client** | Typed wrapper over ogamed HTTP API, rate limiting, retries | TypeScript class with methods mapping 1:1 to REST endpoints |
| **Game State Manager** | Cache and distribute game state (planets, fleets, research, etc.) | In-memory state with periodic refresh; event emitter on state changes |
| **Scheduler / Event Bus** | Coordinate worker execution, manage fleet slots, prevent conflicts | Priority-based job scheduler; EventEmitter for cross-worker communication |
| **Defender Worker** | Monitor attacks, trigger fleet-save, coordinate recall | Polling loop (check `is-under-attack`), response planner |
| **AutoBuild Worker** | ROI-based building/research upgrade decisions | Calculation engine + build queue manager |
| **AutoFarm Worker** | Galaxy scan, spy inactive players, attack if profitable | Galaxy scanner + espionage analyzer + attack dispatcher |
| **Expeditions Worker** | Manage expedition slots, fleet composition, auto-resend | Slot tracker + fleet optimizer + send/recall logic |
| **Web Dashboard** | Real-time monitoring, configuration, manual controls, logs | Next.js app, connects to bot engine via WebSocket + REST |
| **Telegram Bot** | Remote notifications, commands, status queries | node-telegram-bot-api, command handler pattern |
| **Config Manager** | Load, validate, hot-reload YAML/JSON config | File watcher + typed config schema + validation |

## Recommended Project Structure

```
ogame-bot/
├── docker-compose.yml          # ogamed + bot + dashboard
├── package.json
├── tsconfig.json
├── config/                     # Configuration files
│   ├── settings.yaml           # Global settings (telegram, dashboard, accounts)
│   └── accounts/               # Per-account config
│       └── account1.yaml       # Instance-specific config (features, targets, etc.)
├── src/
│   ├── index.ts                # Entry point - starts bot engine
│   ├── client/                 # OGame Client Layer
│   │   ├── ogamed-client.ts    # REST client wrapping ogamed API
│   │   ├── types.ts            # OGame data types (Planet, Fleet, Resources, etc.)
│   │   └── rate-limiter.ts     # Request throttling per ogamed instance
│   ├── state/                  # Game State Manager
│   │   ├── game-state.ts       # Central state store (planets, fleets, research)
│   │   ├── fleet-tracker.ts    # Track fleet movements across all planets
│   │   └── planet-cache.ts     # Per-planet resource/building cache with TTL
│   ├── engine/                 # Scheduler / Event Bus
│   │   ├── scheduler.ts        # Priority job scheduler
│   │   ├── event-bus.ts        # Cross-worker event system
│   │   ├── fleet-lock.ts       # Fleet slot coordination (prevent double-booking)
│   │   └── sleep-mode.ts       # Sleep/wake scheduling for anti-detection
│   ├── workers/                # Feature workers
│   │   ├── worker-base.ts      # Abstract worker with lifecycle hooks
│   │   ├── defender/
│   │   │   ├── defender-worker.ts      # Attack monitoring loop
│   │   │   ├── fleet-saver.ts          # Fleet-save logic (phalanx-safe)
│   │   │   └── attack-analyzer.ts      # Parse attack events, estimate danger
│   │   ├── auto-build/
│   │   │   ├── auto-build-worker.ts    # Build decision loop
│   │   │   ├── roi-calculator.ts       # ROI math for buildings/research
│   │   │   └── build-planner.ts        # Queue and execute builds across planets
│   │   ├── auto-farm/
│   │   │   ├── auto-farm-worker.ts     # Farming loop
│   │   │   ├── galaxy-scanner.ts       # Scan ranges, find targets
│   │   │   ├── espionage-analyzer.ts   # Parse spy reports, assess profit
│   │   │   └── attack-planner.ts       # Calculate fleet needed, send attacks
│   │   └── expeditions/
│   │       ├── expedition-worker.ts     # Expedition management loop
│   │       └── fleet-composer.ts        # Optimal expedition fleet composition
│   ├── notifications/          # Notification layer
│   │   ├── notifier.ts         # Abstract notification interface
│   │   ├── telegram.ts         # Telegram bot integration
│   │   └── templates.ts        # Message templates (attack alerts, etc.)
│   ├── config/                 # Config management
│   │   ├── config-loader.ts    # Load and validate config files
│   │   ├── config-watcher.ts   # Hot-reload on file changes
│   │   └── schema.ts           # Zod/JSON schema for config validation
│   ├── utils/                  # Shared utilities
│   │   ├── logger.ts           # Structured logging
│   │   ├── random.ts           # Random delays for anti-detection
│   │   └── ogame-math.ts       # Flight time, fuel, ROI calculations
│   └── dashboard/              # Web dashboard (or separate package)
│       ├── server.ts           # HTTP + WebSocket server
│       ├── api/                # REST API routes for dashboard
│       └── ws/                 # WebSocket handlers for real-time updates
├── dashboard/                  # Frontend (Next.js or Vite React)
│   ├── package.json
│   └── src/
│       ├── pages/              # Dashboard pages
│       └── components/         # UI components
└── tests/
    ├── client/
    ├── workers/
    └── utils/
```

### Structure Rationale

- **`client/`**: Isolated from game logic. Maps 1:1 to ogamed REST endpoints. Can be tested by mocking HTTP. Other OGame bots (TBot, Cruiser) use the same pattern of a thin client wrapping the protocol layer.
- **`state/`**: Single source of truth for game data. Workers read from state, don't call ogamed directly for reads. This mirrors TBot's `TBotOgamedBridge` + cached player state pattern.
- **`engine/`**: Coordination layer that prevents workers from stepping on each other. Critical for fleet slot management — you can't have Defender and AutoFarm both trying to send fleets when only 1 slot is free.
- **`workers/`**: Each feature is an independent worker with clear boundaries. Based on TBot's worker pattern (WorkerBase → specific workers). Each worker has its own tick interval and can be enabled/disabled independently.
- **`notifications/`**: Abstract interface so Telegram can be swapped or supplemented. TBot and Cruiser both demonstrate this pattern.
- **`config/`**: Hot-reload is critical for 24/7 operation (proven by TBot's SettingsFileWatcher).

## Architectural Patterns

### Pattern 1: Worker with Scheduled Ticks

**What:** Each feature runs as an independent loop with configurable tick intervals. Workers don't call each other directly — they communicate through the event bus and compete for fleet slots through the scheduler.

**When to use:** Every feature module (defender, auto-build, auto-farm, expeditions).

**Trade-offs:** Decoupled workers are easy to develop/test independently, but require careful coordination for shared resources (fleet slots, game API rate limits).

**Example:**
```typescript
abstract class WorkerBase {
  protected tickIntervalMs: number;
  protected running: boolean = false;

  abstract get name(): string;
  protected abstract onTick(): Promise<void>;

  async start(): Promise<void> {
    this.running = true;
    while (this.running) {
      if (!sleepMode.isSleeping() && this.isEnabled()) {
        try {
          await this.onTick();
        } catch (error) {
          logger.error(`${this.name} tick failed`, { error });
        }
      }
      await sleep(this.tickIntervalMs + randomDelay(0, 2000)); // anti-detection jitter
    }
  }

  stop(): void { this.running = false; }
  protected abstract isEnabled(): boolean;
}
```

### Pattern 2: Cached Game State with Event Emission

**What:** Central game state that periodically refreshes from ogamed and emits change events. Workers subscribe to events they care about rather than polling raw API.

**When to use:** For any data that multiple workers need (planet resources, fleet movements, build queues).

**Trade-offs:** Slightly stale data vs. reduced API calls. In OGame, data changes slowly enough that a 30-60 second cache is fine. Critical paths (defender checking attacks) bypass cache.

**Example:**
```typescript
class GameState {
  private planets: Map<CelestialID, PlanetState>;
  private fleets: Fleet[];
  private research: Researches;

  // Refresh all state from ogamed on a timer
  async refresh(): Promise<void> {
    const [planets, fleets, research] = await Promise.all([
      this.client.getPlanets(),
      this.client.getFleets(),
      this.client.getResearch(),
    ]);
    this.planets = new Map(planets.map(p => [p.id, p]));
    const prevFleets = this.fleets;
    this.fleets = fleets;
    this.research = research;

    // Emit events for workers that care about fleet changes
    this.emit('fleets:updated', { current: fleets, previous: prevFleets });
    this.emit('state:refreshed');
  }

  getAvailableFleetSlots(): number {
    const total = this.fleets.length; // from Slots in ogamed response
    return this.slotsTotal - this.slotsInUse;
  }
}
```

### Pattern 3: Fleet Slot Coordination (Semaphore)

**What:** Fleet slots are the scarcest resource in OGame. A semaphore/lock ensures workers don't overbook fleet slots. Defender takes priority.

**When to use:** Any worker that sends fleets (defender, auto-farm, expeditions, fleet-save).

**Trade-offs:** Adds complexity but prevents the most common bug in OGame bots — two workers trying to send fleets when only one slot is free.

**Example:**
```typescript
class FleetSlotManager {
  private queue: Array<{ priority: number; execute: () => Promise<FleetID> }> = [];

  async sendFleet(priority: 'critical' | 'high' | 'normal' | 'low',
                  sendFn: () => Promise<FleetID>): Promise<FleetID> {
    // Wait until a slot is available
    while (this.gameState.getAvailableFleetSlots() <= 0) {
      await sleep(5000);
    }
    // Reserve slot, send fleet
    const fleetId = await sendFn();
    this.gameState.refresh(); // Update slot count
    return fleetId;
  }
}
// Priority: defender=critical, fleet-save=high, expeditions=normal, auto-farm=low
```

### Pattern 4: Ogamed Bridge (REST Client)

**What:** A typed TypeScript client that wraps all ogamed REST endpoints with consistent error handling, rate limiting, and retry logic. This is the single point of contact with ogamed — no other code should make HTTP calls to ogamed.

**When to use:** Every interaction with OGame goes through this client.

**Trade-offs:** Thin abstraction is correct here because ogamed IS the API. Don't try to hide OGame complexity behind another abstraction — ogamed already handles that.

**Example:**
```typescript
class OgamedClient {
  private baseUrl: string;

  constructor(ogamedUrl: string, private rateLimiter: RateLimiter) {
    this.baseUrl = ogamedUrl;
  }

  async isUnderAttack(): Promise<boolean> {
    const res = await this.get<{ Result: boolean }>('/bot/is-under-attack');
    return res.Result;
  }

  async sendFleet(planetID: number, params: SendFleetParams): Promise<FleetID> {
    const res = await this.post<{ Result: FleetID }>(
      `/bot/planets/${planetID}/send-fleet`, params
    );
    return res.Result;
  }

  // ... one method per ogamed endpoint
}
```

## Data Flow

### Attack Detection and Fleet-Save Flow

```
[Defender Worker - every 30-60s]
    │
    ├──► [OgamedClient] GET /bot/is-under-attack
    │         │
    │         ▼
    │    true? ├──► GET /bot/attacks (attack details)
    │         │
    │         ▼
    │    [Attack Analyzer] — estimate arrival time, attacker strength
    │         │
    │         ▼
    │    [Fleet Saver] — calculate safe destination, pick ships + resources
    │         │
    │         ▼
    │    [Fleet Slot Manager] — reserve slot (CRITICAL priority)
    │         │
    │         ▼
    │    [OgamedClient] POST /bot/planets/:id/send-fleet
    │         │
    │         ▼
    │    [Notifier] — Telegram: "Attack incoming! Fleet saved to X:XXX:X"
    │         │
    │         ▼
    │    [Scheduler] — schedule recall check after attack passes
    │
    false? ──► (no action, continue monitoring)
```

### Auto-Build Flow

```
[AutoBuild Worker - every 2-5 min]
    │
    ├──► [GameState] — get all planets, current buildings, research
    │
    ├──► [ROI Calculator] — for each planet, calculate:
    │    - Which building gives best ROI (production increase / cost)
    │    - Which research gives best ROI across all planets
    │    - Can we afford it? (check resources)
    │
    ├──► [Build Planner] — sort by ROI, pick top candidate
    │         │
    │         ▼
    │    Is slot free? (check constructions being built)
    │         │
    │    yes ├──► [OgamedClient] POST /bot/planets/:id/build/building/:ogameID
    │         │
    │    no  ──► (skip, check next planet or wait)
    │
    └──► [Event Bus] emit 'build:started' (dashboard picks this up)
```

### Auto-Farm Flow

```
[AutoFarm Worker - every 15-30 min]
    │
    ├──► [Galaxy Scanner] — scan configured galaxy/system ranges
    │    [OgamedClient] GET /bot/galaxy-infos/:galaxy/:system
    │         │
    │         ▼
    │    Filter: inactive players, no vacation, not too strong
    │         │
    │         ▼
    ├──► [Espionage] — send probes to filtered targets
    │    [Fleet Slot Manager] — reserve slot (LOW priority)
    │    [OgamedClient] POST /bot/planets/:id/send-fleet (spy mission)
    │         │
    │         ▼
    │    Wait for spy reports...
    │    [OgamedClient] GET espionage report messages
    │         │
    │         ▼
    ├──► [Espionage Analyzer] — parse reports, calculate loot vs. cost
    │         │
    │         ▼
    │    Filter: only targets with profit above threshold
    │         │
    │         ▼
    ├──► [Attack Planner] — calculate required ships, send attacks
    │    [Fleet Slot Manager] — reserve slot (LOW priority)
    │    [OgamedClient] POST /bot/planets/:id/send-fleet (attack mission)
    │
    └──► [Event Bus] emit 'farm:attacks_sent' (dashboard shows activity)
```

### Dashboard Real-Time Update Flow

```
[Browser]
    │
    ├──► WebSocket connect to /ws
    │         │
    │         ▼
    [Bot Engine WebSocket Handler]
    │         │
    │    subscribes to event bus
    │         │
    │    On 'state:refreshed'    ──► push planet/fleet summary
    │    On 'build:started'     ──► push build notification
    │    On 'defender:alert'    ──► push attack warning
    │    On 'farm:attacks_sent' ──► push farm activity
    │    On 'log:*'             ──► push log entries
    │
    ├──► REST API for configuration
    │    GET  /api/state         ──► full game state snapshot
    │    GET  /api/config        ──► current configuration
    │    POST /api/config        ──► update configuration (hot-reload)
    │    POST /api/fleet/send    ──► manual fleet send
    │    POST /api/build/:planet ──► manual build
    │    GET  /api/logs          ──► recent log entries
    │
    └──► Static Next.js pages
```

## Multi-Account Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Bot Process                       │
│                                                     │
│  ┌─────────────────┐  ┌─────────────────┐          │
│  │ Account Manager  │  │  Dashboard       │          │
│  │                  │  │  (single server) │          │
│  │  ┌────────────┐  │  │                 │          │
│  │  │ Account 1  │  │  │  serves UI for  │          │
│  │  │ ┌────────┐ │  │  │  all accounts   │          │
│  │  │ │Workers │ │  │  └────────┬────────┘          │
│  │  │ │State   │ │  │           │                    │
│  │  │ │Client──┼─┼──┼───────────┤                    │
│  │  │ └────────┘ │  │           │                    │
│  │  └────────────┘  │           │                    │
│  │       │          │           │                    │
│  │  ┌────────────┐  │           │                    │
│  │  │ Account 2  │  │           │                    │
│  │  │ ┌────────┐ │  │           │                    │
│  │  │ │Workers │ │  │           │                    │
│  │  │ │State   │ │  │           │                    │
│  │  │ │Client──┼─┼──┼───────────┤                    │
│  │  │ └────────┘ │  │           │                    │
│  │  └────────────┘  │           │                    │
│  └─────────────────┘           │                    │
└─────────────────────────────────┤────────────────────┘
                                  │
          ┌───────────────────────┤────────────────────┐
          │                       │                     │
    ┌─────┴──────┐       ┌───────┴──────┐       ┌─────┴──────┐
    │  ogamed #1 │       │  ogamed #2   │       │  ogamed #N │
    │  :8080     │       │  :8081       │       │  :8082     │
    └────────────┘       └──────────────┘       └────────────┘
```

**Key decision:** One bot process manages all accounts. Each account gets its own ogamed instance on a different port. The bot engine creates an isolated "account context" per account with its own workers, state, and client — but shares the dashboard server, Telegram bot, and config manager.

This matches TBot's `InstanceManager` pattern. Each account context is fully independent except for shared infrastructure.

## Build Order (Dependencies)

Based on the architecture, this is the recommended build order derived from component dependencies:

```
1. ogamed Client Layer (client/)
   └── No internal dependencies. Wraps REST API. Must exist first.
   └── Test: Can connect to ogamed, fetch planets/resources.

2. Game State Manager (state/)
   └── Depends on: ogamed Client
   └── Test: Can cache and refresh game state.

3. Engine Core (engine/)
   └── Depends on: Game State (for slot tracking)
   └── Test: Scheduler runs jobs, fleet slot coordination works.

4. Defender Worker (workers/defender/)
   └── Depends on: Client, State, Engine (fleet slots), Notifications
   └── FIRST worker because fleet safety is the core value proposition.
   └── Test: Detects attack, sends fleet to safe coordinate.

5. Notification Layer (notifications/)
   └── Can be built in parallel with workers, but Defender needs it.
   └── Start with Telegram since it's the primary notification channel.

6. Expeditions Worker (workers/expeditions/)
   └── Depends on: Client, State, Engine, Notifications
   └── Second worker — relatively simple, high value.

7. Auto-Build Worker (workers/auto-build/)
   └── Depends on: Client, State, Engine, Notifications
   └── Third worker — ROI calculation is the complex part.

8. Auto-Farm Worker (workers/auto-farm/)
   └── Depends on: Client, State, Engine, Notifications
   └── Fourth worker — most complex (galaxy scanning + espionage + attack).

9. Dashboard Backend (src/dashboard/)
   └── Depends on: State, Event Bus
   └── API routes + WebSocket for real-time updates.

10. Dashboard Frontend (dashboard/)
    └── Depends on: Dashboard Backend API
    └── Next.js/React UI.
```

## Anti-Patterns

### Anti-Pattern 1: Workers Calling ogamed Directly for Reads

**What people do:** Each worker independently calls ogamed to get planet resources, fleet slots, etc.
**Why it's wrong:** Causes 5-10x more API requests than necessary. OGame rate-limits aggressively. Multiple workers refreshing the same data independently will get the account banned.
**Do this instead:** Use the Game State Manager as the single reader. Workers read from cached state. Only the state manager refreshes from ogamed on a controlled schedule.

### Anti-Pattern 2: Ignoring Fleet Slot Contention

**What people do:** Workers independently send fleets without checking if another worker just used the last slot.
**Why it's wrong:** Results in failed fleet sends, which for Defender means potentially losing your entire fleet.
**Do this instead:** Fleet Slot Manager with priority queuing. Defender always wins. Expeditions and farming yield when slots are scarce.

### Anti-Pattern 3: Blocking Main Thread with Game Logic

**What people do:** Synchronous loops that block Node.js event loop while waiting for API responses.
**Why it's wrong:** Blocks WebSocket updates to dashboard, delays defender response, makes the whole system feel frozen.
**Do this instead:** All workers are async. Use `setInterval` / `setTimeout` for scheduling, never `while(true)` without yields. The scheduler should be non-blocking.

### Anti-Pattern 4: Hardcoded Delays Without Jitter

**What people do:** Workers tick at exact intervals (every 60.000 seconds).
**Why it's wrong:** OGame can detect bot-like regularity in request patterns. TBot and Cruiser both add random jitter.
**Do this instead:** Add ±10-20% random jitter to all intervals. The `randomDelay()` utility should be used everywhere.

### Anti-Pattern 5: Storing OGame Credentials in Config Files

**What people do:** Put username/password in settings.json that gets committed to git.
**Why it's wrong:** Credentials leak. OGame accounts get stolen.
**Do this instead:** Environment variables or a `.env` file (gitignored). Config files only store behavior settings, not credentials.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| **ogamed** | HTTP REST (localhost) | One instance per account on different ports. JSON responses `{Status, Code, Message, Result}`. Default port 8080. |
| **OGame servers** | Via ogamed only — never direct | ogamed handles cookies, fingerprinting, anti-bot. Our code should NEVER talk to OGame directly. |
| **Telegram Bot API** | HTTP long-polling or webhook | `node-telegram-bot-api` or `grammy`. Commands for remote control. Notifications for alerts. |
| **OGame Public API** | Optional: XML API for server data | `https://sNNN-LL.ogame.gameforge.com/api/...` — server info, player rankings. No auth needed. Useful for target selection but not required. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Client ↔ State | State calls Client on refresh timer | Client is stateless — just wraps HTTP |
| State ↔ Workers | Workers read state synchronously; subscribe to events | State is the single source of truth |
| Workers ↔ Fleet Manager | Workers request fleet slots via async API | Priority queuing prevents conflicts |
| Workers ↔ Event Bus | Workers emit and subscribe to events | Loose coupling between features |
| Workers ↔ Notifications | Workers call notifier for alerts | Notifications are fire-and-forget |
| Dashboard ↔ Bot Engine | REST API + WebSocket | Dashboard is read-only mostly; config changes via REST |
| Account Contexts | No direct communication | Each account is fully isolated |

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1 account | Single process, everything in memory. Simple config files. |
| 2-5 accounts | Same process, multiple account contexts. Each has its own ogamed instance (different port). Shared dashboard and Telegram. |
| 5-10 accounts | Consider separate ogamed instances on different machines (residential proxies needed). Bot process stays single but connects to remote ogamed instances. |

**This is NOT a high-concurrency system.** It's a game bot managing 1-5 accounts. The scaling concerns are about OGame's rate limits and IP bans, not our server capacity.

### Scaling Priorities

1. **First bottleneck:** OGame rate limiting (requests per second). Mitigation: aggressive caching, batched reads, respect ogamed's internal rate limiting.
2. **Second bottleneck:** Fleet slot contention across workers. Mitigation: Priority-based fleet slot manager (already designed above).

## Sources

- ogamed REST API documentation: https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation (HIGH confidence)
- TBot source structure (Workers, Services, Infrastructure): https://github.com/ogame-tbot/TBot (HIGH confidence)
- Cruiser architecture (client/engine/bot separation): https://github.com/kweimann/cruiser (HIGH confidence)
- OGame game mechanics (fleet slots, phalanx, rate limits): domain knowledge from reference projects (HIGH confidence)

---
*Architecture research for: OGame automation bot*
*Researched: 2026-04-25*
