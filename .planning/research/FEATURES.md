# Feature Research

**Domain:** OGame automation bot
**Researched:** 2026-04-25
**Confidence:** HIGH (based on detailed analysis of 3 reference projects + TBot wiki)

## Feature Landscape

### Table Stakes (Users Expect These)

Features every OGame bot user assumes exist. Missing any = instant dealbreaker.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Attack detection** | THE reason people use bots — protecting fleet while away | LOW | Poll ogamed for hostile fleet events; every bot does this. Cruiser polls on configurable interval. TBot checks every 1-22 min (randomized). |
| **Fleet-save** | If your fleet dies, weeks/months of progress are gone | HIGH | Must save fleet + resources before attack lands. Requires: destination selection, fleet composition, mission type, timing. Phalanx-safe (deploy with recall) is the gold standard — Cruiser built its identity on this. |
| **Telegram notifications** | Every successful bot has this. Users run bots on remote servers and need mobile alerts. | LOW | Send alerts on attack, fleet-save actions, errors. TBot and Cruiser both implement. Simple HTTP API. |
| **Auto-build (AutoMine)** | Growing your empire is the core OGame loop. A bot that doesn't build is just an alarm clock. | MEDIUM | ROI-based algorithm is the standard (TBot). Must calculate production increase per cost across all planets, pick the best investment. Also needs max-level caps per building type. |
| **Auto-expeditions** | Expeditions are the highest-value passive income in modern OGame. Not automating them wastes enormous potential. | MEDIUM | Manage expedition slots, send fleets, auto-resend on return. Must handle fleet composition (auto-optimize based on account size). TBot auto-calculates optimal fleet including lifeform bonuses. |
| **Auto-farm** | Attacking inactive players for resources is tedious but essential. Without it, bot users fall behind manual players who farm. | MEDIUM | Scan galaxy ranges, spy inactives, analyze spy reports, attack if profitable. TBot: scan ranges, spy inactives, attack with specified ship type above profit threshold. |
| **Configuration system** | Users need to control what the bot does — which features are active, parameters per feature. | LOW | JSON or YAML config file per account. All three bots implement this. |
| **Multi-account support** | Many OGame players run multiple accounts. Single-account bots are a non-starter for power users. | MEDIUM | TBot: instances array in settings, each with own config file. Different ports per instance. Must handle cookie isolation between lobby accounts. |

### Differentiators (Competitive Advantage)

Features that set a bot apart. Not required for v1, but they're what make users choose one bot over another.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **ROI-based auto-build** | TBot's gold standard. Calculates production increase / cost ratio across ALL planets to find the single most profitable upgrade. Far superior to naive "build whatever is cheapest" or fixed-priority approaches. | HIGH | Requires: production formulas, building costs (exponential), current production rates, energy balance. TBot adds MaxDaysOfInvestmentReturn to cap ROI. This IS the differentiator for auto-build — without ROI calculation, auto-build is trivial and unimpressive. |
| **Phalanx-safe fleet-save with recall** | Cruiser's signature feature. Sends fleet on deploy mission, then recalls so it returns after the attack. Invisible to sensor phalanx. Shows deep game knowledge. | HIGH | Deploy mission → recall at calculated time. Must account for: returning fleets, deployment fleets already in flight, fuel constraints, timing precision. Cruiser also considers min fuel consumption. |
| **Telegram remote control** | Not just notifications — full command interface. TBot has 30+ commands: `/ghost`, `/deploy`, `/build`, `/recall`, `/sleep`, `/getinfo`, etc. This turns Telegram into a mobile command center. | MEDIUM | Bot command parser + action dispatch. TBot lets you control every feature, build ships, deploy fleets, edit settings remotely. High user engagement. |
| **Web dashboard** | Real-time empire overview: resources, fleet movements, build queues, logs. TBot proves this is expected for modern bots. Also enables mobile monitoring. | HIGH | Full web app with real-time updates. TBot's WebUI: settings editor, filterable logs, manual game interaction, multi-account view. Significant frontend investment. |
| **Settings hot-reload** | For a 24/7 service, restarting to change config is painful. TBot watches config files and applies changes without restart. | MEDIUM | File watcher + config merge + feature toggle on/off. TBot supports this for instance settings (partial for adding/removing instances). |
| **Sleep mode** | Reduce activity during specific hours to avoid detection. TBot stops all game interaction during sleep hours, with optional auto-fleet-save before sleeping. | MEDIUM | Timer-based. Fleet-save all assets before sleep, recall on wake. Prevents "this account is active 24/7" detection pattern. |
| **Auto-repatriate (resource consolidation)** | Automatically moves all resources to a central planet/moon. Essential for feeding auto-build and auto-research from a single hub. | MEDIUM | TBot: configurable target, cargo type, exclusion list, deuterium-to-leave on moons. Low complexity but high operational value. |
| **Auto-cargo** | Automatically builds transport ships when planets lack cargo capacity. Prevents the "resources stuck on planet" problem. | LOW | TBot: detect insufficient cargo → build ships. Configurable: cargo type, max to build, max to keep, exclude moons. |
| **Auto-research** | Automates technology research. TBot: set target levels per tech, prioritizes Astrophysics/Plasma/Energy/IRN in early game. | MEDIUM | Requires research tree knowledge, lab requirements, resource transport to research planet. |
| **Auto-harvest (debris field)** | Automatically sends recyclers/pathfinders to harvest expedition debris and own debris fields. Free resources left on the table otherwise. | LOW | TBot: harvest expedition DF in celestial systems + own DFs. Cruiser: harvest expo debris (discoverer class only). |
| **Auto-colonize** | Automates new colony creation. TBot: target coordinates, abandon bad planets (temperature filter), intensive research mode. | MEDIUM | Colony ship management, temperature-based planet quality assessment, abandon/retry logic. |
| **Anti-detection measures** | Random intervals between actions, request throttling, device fingerprinting. Not a visible feature but critical for user trust. | MEDIUM | ogamed handles device fingerprinting and captcha. Bot must add: random delays between actions, sleep mode, activity pattern variation. |

### Anti-Features (Deliberately Do NOT Build)

Features that seem appealing but create problems. Documented to prevent scope creep.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Browser proxy (play through bot)** | "Play from the same IP as the bot" — avoids dual-IP detection | Bot is unaware of manual actions. Player builds something while bot tries to build → conflicts. TBot explicitly warns "TBot is not aware of what you do in the browser." Massive complexity for game-state synchronization. | Don't mix bot and manual play. If user needs manual access, they should pause the bot first. |
| **Marketplace automation** | Automate trading on the in-game marketplace | Niche feature. Marketplace rules change frequently. High ban risk (marketplace manipulation is heavily monitored). Adds significant complexity for low value. | User can handle marketplace manually. Marketplace is not time-critical like fleet-save. |
| **Combat simulator** | "I want to know if I'll win before attacking" | Multiple excellent external tools exist (speedsim, trashsim, o Gotcha). Building one is a significant project unto itself. r4fek's bot had one but it's a distraction. | Link to existing combat simulators in the UI. Provide spy report data export for external tools. |
| **Auction automation** | "Auto-bid on auctions for items" | TBot implements this but it's extremely niche. Auction timing is unpredictable, bid wars with other bots create suspicious activity patterns. High request rate during auctions increases ban risk. | Not worth the complexity. Users can manually check auctions. |
| **Message attacker** | "Send a message to anyone attacking me" | Seems clever but escalates situations. Alerts attackers that you're a bot user (no human responds in seconds at 3 AM). TBot implements it but it's a known risk factor. | Just fleet-save silently. Don't draw attention. |
| **Lifeform-specific automation** | "Automate lifeform buildings and research" | TBot recently added this (LifeformAutoMine, LifeformAutoResearch, AutoDiscovery). Lifeforms are a relatively new OGame feature with frequent balance changes. Massive config surface (6 buildings × 3 tiers per planet). Not core value. | Add as v2 feature once core is stable. Lifeform data is more volatile than core game data. |
| **SMS notifications** | "I don't use Telegram" | r4fek's bot used SMS but it's 2018-era. SMS costs money, requires third-party gateway, less capable than Telegram (no commands, no rich messages). Telegram is free and supports bot commands. | Telegram only. It's the industry standard for bot notifications + control. |
| **Mobile app** | "Native app for my phone" | Web dashboard is mobile-responsive. Building a native app is a massive separate project. TBot doesn't have one. No OGame bot does. | Mobile-responsive web dashboard. Telegram for push notifications. |

## Feature Dependencies

```
Game State Layer (ogamed connection)
    ├──requires──> Account credentials + ogamed REST API
    │
    ├──enables──> Attack Detection
    │                └──requires──> Fleet-save (fleet-save needs attack info)
    │                    └──requires──> Fleet management (send/recall fleets)
    │                        └──requires──> Game state (know what ships exist, where)
    │
    ├──enables──> Auto-build (ROI)
    │                └──requires──> Building cost formulas + production rates
    │                └──enhanced-by──> Auto-repatriate (centralize resources for building)
    │                └──enhanced-by──> Auto-cargo (ensure transport capacity)
    │                └──enhanced-by──> Auto-research (unlock better buildings)
    │
    ├──enables──> Auto-farm
    │                └──requires──> Galaxy scanning (ogamed API)
    │                └──requires──> Spy report parsing
    │                └──requires──> Profit calculation (loot formula)
    │                └──requires──> Fleet management (send attack fleets)
    │
    ├──enables──> Auto-expeditions
    │                └──requires──> Fleet management (send expeditions)
    │                └──requires──> Slot management (track active expeditions)
    │                └──enhanced-by──> Auto-harvest (collect expedition debris)
    │
    └──enables──> Notifications (Telegram)
                     └──enhanced-by──> Telegram remote control (commands)

Sleep Mode
    └──requires──> Fleet-save (save all fleets before sleeping)
    └──requires──> Timer/scheduler system

Web Dashboard
    └──requires──> Game state layer (data to display)
    └──enhanced-by──> Settings hot-reload (change config from UI)

Multi-account
    └──requires──> Account isolation (separate configs, cookies, ports)
    └──enhanced-by──> Web dashboard (view all accounts)
```

### Dependency Notes

- **Fleet-save requires attack detection:** Fleet-save is triggered BY incoming attacks. Without detection, fleet-save has nothing to react to. This is the most critical dependency — attack detection → fleet-save must work flawlessly.
- **Auto-build requires game state + formulas:** ROI calculation needs current building levels, production rates, and cost formulas for every building. These are deterministic from OGame wiki data + current planet state.
- **Auto-farm requires galaxy scanning + spy reports:** Must scan galaxy for inactive players (i-icon), send espionage probes, parse the resulting spy report for resources/defense, then calculate profitability. This is a multi-step pipeline.
- **Auto-expeditions requires slot management:** Must track how many expedition slots are available (based on Astrophysics level), which expeditions are active, when they return. Cannot oversend.
- **Sleep mode requires fleet-save:** Before sleeping, ALL fleet-bearing planets/moons must be saved. Otherwise sleeping = certain death if attacked. This creates a circular dependency (fleet-save during sleep), which TBot handles with AutoFleetSave within SleepMode.
- **Web dashboard requires game state:** Dashboard is useless without data. Must have the game state polling layer working first.
- **Telegram remote control enhances notifications:** Commands are built ON TOP of the notification system. Build notifications first, then add command parsing.

## MVP Definition

### Launch With (v1)

Minimum viable product — what's needed for users to trust the bot with their account.

- [ ] **ogamed connection + game state polling** — foundation everything else builds on. Must reliably connect, maintain session, poll game state.
- [ ] **Attack detection + fleet-save** — THE core value. If this doesn't work, nothing else matters. Must be phalanx-safe (deploy with recall). Must handle moons separately.
- [ ] **Telegram notifications** — attack alerts, fleet-save confirmations, error alerts. Users need to know the bot is working without checking.
- [ ] **Auto-build (ROI-based)** — empire growth while away. ROI algorithm makes this genuinely useful vs trivial. Include max-level caps.
- [ ] **Auto-expeditions** — high-value passive income. Must handle slot management and auto-resend.
- [ ] **Configuration system** — JSON/YAML config per account. Feature toggles, parameters.
- [ ] **Anti-detection basics** — random intervals between actions, request throttling, sleep mode.

### Add After Validation (v1.x)

Features to add once core is proven reliable.

- [ ] **Auto-farm** — scan galaxy, spy inactives, attack if profitable. Trigger: once fleet-save and auto-build are stable. Adds significant resource income.
- [ ] **Web dashboard** — real-time empire overview, fleet movements, build queues. Trigger: once bot is running reliably 24/7 and users want monitoring.
- [ ] **Auto-repatriate** — centralize resources. Trigger: once auto-build + auto-research need centralized resources to function well.
- [ ] **Auto-research** — automate tech research. Trigger: once auto-build is stable and users want full automation.
- [ ] **Auto-cargo** — build transport ships automatically. Trigger: once resource management features need it.
- [ ] **Multi-account support** — run multiple accounts. Trigger: once single-account is rock-solid.
- [ ] **Settings hot-reload** — change config without restart. Trigger: once users are running 24/7 and config changes are painful.
- [ ] **Telegram commands** — remote control via Telegram. Trigger: once notifications are stable, add command interface.

### Future Consideration (v2+)

Features to defer until core product is mature.

- [ ] **Auto-colonize** — automate new colonies. Defer: complex (temperature evaluation, abandon/retry) and most players already have their planets.
- [ ] **Auto-harvest** — collect debris fields. Defer: nice-to-have, doesn't affect core safety or growth significantly.
- [ ] **Lifeform automation** — lifeform buildings/research. Defer: new feature, frequently changing balance, massive config surface.
- [ ] **Auto-discovery** — send lifeform discovery missions. Defer: depends on lifeform support.
- [ ] **Buy offer of the day** — automate trader purchases. Defer: extremely niche.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Attack detection | CRITICAL | LOW | P1 |
| Fleet-save (phalanx-safe + recall) | CRITICAL | HIGH | P1 |
| Telegram notifications | HIGH | LOW | P1 |
| Auto-build (ROI-based) | HIGH | MEDIUM | P1 |
| Auto-expeditions | HIGH | MEDIUM | P1 |
| Configuration system | HIGH | LOW | P1 |
| Anti-detection (random delays, throttling) | HIGH | LOW | P1 |
| Sleep mode | MEDIUM | MEDIUM | P1 |
| Auto-farm | HIGH | MEDIUM | P2 |
| Web dashboard | MEDIUM | HIGH | P2 |
| Auto-repatriate | MEDIUM | MEDIUM | P2 |
| Auto-research | MEDIUM | MEDIUM | P2 |
| Multi-account | MEDIUM | MEDIUM | P2 |
| Telegram commands | MEDIUM | MEDIUM | P2 |
| Settings hot-reload | MEDIUM | MEDIUM | P2 |
| Auto-cargo | LOW | LOW | P2 |
| Auto-harvest | LOW | LOW | P3 |
| Auto-colonize | LOW | MEDIUM | P3 |
| Lifeform automation | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for launch — the bot is useless/dangerous without these
- P2: Should have, add when possible — bot works but users will request these
- P3: Nice to have, future consideration — specialized use cases

## Competitor Feature Analysis

| Feature | TBot (94★) | Cruiser (30★) | r4fek/ogame-bot (45★, archived) | Our Approach |
|---------|-------------|----------------|----------------------------------|--------------|
| **Fleet-save** | Defender + AutoFleetSave (sleep mode). Deploy with recall. | Phalanx-safe, auto-recall, min fuel. Core identity. | Basic fleet save. | Phalanx-safe deploy+recall (from Cruiser). Also during sleep mode (from TBot). Must be rock-solid. |
| **Auto-build** | ROI-based AutoMine. Max levels per building. MaxDaysOfInvestmentReturn. | Not implemented. | Basic building. | ROI-based (from TBot). This is non-negotiable — any other approach is inferior. |
| **Auto-farm** | Scan ranges, spy inactives, attack if profitable. Ship type config. | Not implemented. | Farming (basic). | TBot-style: configurable galaxy ranges, profit threshold, ship type. |
| **Expeditions** | Auto-optimize fleet, multi-origin, military expos, lifeform bonus calc. | Simple per-expedition config, auto-resend. | Basic expeditions. | TBot-style auto-optimize. Must handle slot management + auto-resend. |
| **Telegram** | 30+ commands, full remote control, auto-ping. | Notifications only (attack alerts, actions taken). | SMS only (no Telegram). | Start with notifications (like Cruiser), expand to commands (like TBot). |
| **Web UI** | Full WebUI: settings editor, filterable logs, manual game play. | None. | None. | Build after core features are stable. Focus on monitoring first, control later. |
| **Multi-account** | Instances array, per-instance config, shared/different cookies. | Single account. | Single account. | P2. Follow TBot's pattern: instances array with separate config files. |
| **Sleep mode** | Full sleep with AutoFleetSave. Wake timer. Telegram notification. | Configurable sleep intervals between checks. | Not implemented. | TBot-style: scheduled sleep + fleet-save before sleeping. Critical for anti-detection. |
| **Language** | C#/.NET 6 | Python 3.7+ | Python 2/3 | TypeScript/Node.js + ogamed REST. Modern stack, shared language for bot + web dashboard. |
| **Anti-detection** | ogamed fingerprinting, random intervals, sleep mode, proxy support. | Random sleep intervals, delay between requests. | Minimal. | ogamed handles fingerprinting/captcha. Bot adds random intervals, sleep mode, activity variation. |

## Key Insights from Competitor Analysis

### What TBot Gets Right (model this)
1. **ROI-based auto-build** — genuinely intelligent, not just "build cheapest thing." This is the #1 reason users choose TBot.
2. **Telegram as command center** — 30+ commands means users can fully manage their account from their phone.
3. **Config hot-reload** — a 24/7 service MUST NOT require restarts for config changes.
4. **Sleep mode + AutoFleetSave** — anti-detection AND fleet protection during low-activity hours.
5. **Extensive configurability** — almost every feature has 5-10 tuning parameters. Power users love this.

### What Cruiser Gets Right (learn from this)
1. **Phalanx-safe fleet-save as identity** — one feature done perfectly > many features done poorly.
2. **Clean architecture** — separate OGame client, game engine (calculations), bot logic. Easy to test and extend.
3. **Auto-adjusts to account state** — "Cruiser automatically adjusts to the current state of your account. Can be restarted at any time." This is critical — the bot must be stateless-safe.
4. **Minimal viable config** — account info + that's it for basic protection. Progressive complexity.

### What r4fek Gets Wrong (avoid this)
1. **SMS notifications** — dead end. Telegram is the standard.
2. **Combat simulator built-in** — scope creep. External tools exist.
3. **Archived since 2018** — Python ogame library unmaintained. Shows the risk of depending on unmaintained dependencies.

### Critical Gap We Can Fill
No existing bot combines:
- **TypeScript/modern stack** (TBot is C#, Cruiser is Python, r4fek is archived Python)
- **Safety-first fleet-save** (Cruiser-level)
- **Full automation** (TBot-level features)
- **Web dashboard** (TBot's is basic, we can do better with modern TS ecosystem)
- **Stateless-safe design** (Cruiser's restart-anywhere philosophy)

## Sources

- **TBot**: github.com/ogame-tbot/TBot — README + Wiki Configuration Guide analyzed in full. 1,173 commits, 83 releases, actively maintained (last release May 2024, v0.3.4). [HIGH confidence]
- **Cruiser**: github.com/kweimann/cruiser — README + config.yaml analyzed in full. 48 commits, no releases, Docker-first. [HIGH confidence]
- **r4fek/ogame-bot**: github.com/r4fek/ogame-bot — README analyzed. 6 commits, archived March 2018. [HIGH confidence]
- **ogamed**: github.com/alaingilbert/ogame — referenced for API capabilities (device fingerprinting, captcha, REST endpoints). [HIGH confidence]

---
*Feature research for: OGame automation bot*
*Researched: 2026-04-25*
