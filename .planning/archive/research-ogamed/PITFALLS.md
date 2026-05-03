# Pitfalls Research

**Domain:** OGame automation bot (anti-detection, fleet-save, API wrapper fragility)
**Researched:** 2026-04-25
**Confidence:** HIGH (cross-referenced TBot issues, ogamed issues, Cruiser source, community warnings)

## Critical Pitfalls

### Pitfall 1: Fleet-Save Fallback to Invalid Missions

**What goes wrong:**
When the primary fleet-save mission (Deploy) fails, the bot falls through to fallback missions without validating destination. TBot issue #178 shows the bot trying to "Colonize" at an already-inhabited position after Deploy failed — which errors with "Planet is already inhabited!" and the fleet stays home un-saved. The entire main fleet was left exposed.

**Why it happens:**
Developers write mission selection as a priority list (Deploy → Harvest → Colonize) but don't validate that the fallback destination is valid for that mission type. Deploy requires a friendly planet/moon. Colonize requires an empty slot. Harvest requires a debris field. The destinations that work for each mission are completely different.

**How to avoid:**
- Validate destination per-mission, not once globally
- When a fleet-save mission fails, recalculate the destination for the next mission type
- Always have a guaranteed-safe fallback: deploy to own planet with recall capability
- Never use "Colonize" as a fleet-save fallback unless you've verified the slot is empty via galaxy scan
- Log the full fallback chain with reasons so failures are diagnosable

**Warning signs:**
- Fleet-save logs show "checking Colonize destination..." after Deploy fails
- Error messages about "Planet is already inhabited" in fleet-save context
- Fleet scheduler stack traces around `ManageResponse` / `SendFleet`

**Phase to address:**
Phase 1 (Fleet-Save) — This is the single most important feature. If fleet-save fails, months of progress can be destroyed in seconds. Build fleet-save first, test it exhaustively with every mission type and edge case before adding any other feature.

---

### Pitfall 2: OGame Game Updates Breaking the API Wrapper

**What goes wrong:**
OGame updates (v11.13, v11.15) changed HTML structure and API responses, breaking ogamed's parsers. `GetEspionageReport` stopped working (ogamed issue #150), `ExtractConstructions` broke (issue #148). The bot silently gets wrong data or crashes.

**Why it happens:**
ogamed works by scraping OGame's HTML responses and parsing page content. When Gameforge changes the page structure, the extractors break. This has happened repeatedly across ogamed's history (268 releases, major migration guides for v48 and v53). The bot layer has no way to know the data is wrong — it just gets stale or garbage data.

**How to avoid:**
- Pin ogamed to a specific version and test after every OGame update
- Implement data validation: if `GetResources` returns all zeros, if timestamps parse to year 0001 (TBot issue #200), or if planet counts suddenly change, alert and pause
- Monitor the ogamed GitHub releases and issues for breakage reports
- Build a health-check system that periodically verifies API responses look reasonable
- Structure the bot so ogamed is the ONLY thing touching OGame — making updates a single-point fix
- Have a "maintenance mode" that pauses all actions when API responses look corrupted

**Warning signs:**
- Date fields returning "1/01/0001" or similar defaults
- Resource counts showing as zero when you know they shouldn't be
- ogamed returning errors or empty results for previously-working endpoints
- OGame announces a version update on their forums

**Phase to address:**
Phase 1 (Core) — Build resilience from day one. Every API call should have error handling that treats unexpected responses as "bot goes blind, alert user, stop autonomous actions."

---

### Pitfall 3: Captcha Kills the Bot Dead

**What goes wrong:**
Captcha is the #1 reported issue across ALL OGame bot projects. TBot has 6+ open captcha-related issues (#170, #165, #158, #184, #189, #203). When Gameforge triggers a captcha, ogamed pauses and waits for solving. Without automated solving, the bot is completely frozen until a human intervenes. A frozen bot can't fleet-save.

**Why it happens:**
Gameforge uses captcha as a primary anti-bot measure. Captcha can be triggered by: too many requests, suspicious IP patterns, login from new location, or random checks. ogamed supports manual solving (via web UI at `host:port/bot/captcha`) and automated solving (Ninja Captcha solver, $0.10/solve). But if you don't set up automated solving, every captcha = bot downtime.

**How to avoid:**
- Implement automated captcha solving from day one (Ninja Solver API or Telegram-based solver)
- The bot MUST detect when it's captcha-blocked and immediately alert via Telegram
- When captcha-blocked, the bot should enter a "safe state" — don't queue new actions, but flag that manual intervention is needed
- Track captcha frequency — if it spikes, the bot is being too aggressive and should throttle back
- Never let captcha block fleet-save monitoring — that must be the last thing to stop

**Warning signs:**
- ogamed logs show captcha prompts
- Bot actions stop happening but process is still running
- "Problem captcha?" or "where to put my captcha" in your own notes

**Phase to address:**
Phase 1 (Core) — Captcha handling must be part of the initial ogamed integration, not an afterthought. Design the alert system to notify on captcha from the start.

---

### Pitfall 4: Datacenter IP Blocking

**What goes wrong:**
OGame actively blocks IP addresses from known datacenter ranges (AWS, DigitalOcean, Hetzner, etc.). TBot's README explicitly warns: "Ogame is positively blocking IPs from datacenters. You will probably need a residential proxy." The bot gets "Forbidden" errors on login (ogamed issue #145) and can never connect.

**Why it happens:**
Gameforge maintains blocklists of datacenter IP ranges. Since no legitimate player would be accessing OGame from an AWS instance, this is a trivial detection signal. This isn't rate-limiting — it's a hard block.

**How to avoid:**
- Plan for residential proxy support from the start
- ogamed has built-in proxy support: `SetProxy(proxyAddress, username, password, proxyType, loginOnly, config)`
- Test with a residential proxy before deploying — don't discover this after setup
- Document proxy configuration as a setup prerequisite, not an optional step
- Support proxy rotation for multi-account setups (each account through a different residential IP)
- The `loginOnly` proxy parameter lets you proxy only the login request through a residential IP while routing game traffic through the datacenter — this may be sufficient and cheaper

**Warning signs:**
- "Forbidden" errors on login despite correct credentials
- Login works from your home computer but not from the VPS
- ogamed logs show connection refused at the login stage

**Phase to address:**
Phase 1 (Core) — Proxy configuration must be documented and tested as part of initial deployment setup. This is a prerequisite, not a nice-to-have.

---

### Pitfall 5: Phalanx-Unsafe Fleet-Save Patterns

**What goes wrong:**
Sensor Phalanx (moon building) lets players scan a planet to see all fleet movements. If fleet-save uses a mission that shows arrival times (like Harvest), an attacker can time their fleet to arrive seconds after yours returns — destroying everything. This is the #1 way experienced players destroy bot users' fleets.

**Why it happens:**
Only certain mission types are "phalanx-safe":
- **Deploy mission**: Shows on scan BUT can be recalled, making the return invisible to phalanx. This is the gold standard.
- **Harvest mission**: Completely invisible to phalanx. Safe, but requires a debris field.
- **Attack/Transport/Colonize/Expedition**: All visible to phalanx with exact return times. Dangerously unsafe.

Developers pick the wrong mission type or don't implement recall for deploy missions.

**How to avoid:**
- Default to Deploy mission with recall for fleet-save (Cruiser's approach)
- If using Harvest, ensure there's actually a debris field at the destination (send a probe to crash first if needed)
- Never use Transport, Attack, or Espionage as fleet-save missions — they're all phalanx-visible
- Implement deploy-with-recall: send fleet on deploy to a friendly planet, then recall before it arrives. The recalled fleet is invisible to phalanx.
- Time the recall so the fleet returns after the attack window has passed
- Support moon-to-moon deploy as an option (moons can't be phalanxed unless attacker has a moon in same system)

**Warning signs:**
- Fleet-save uses Transport or Harvest without debris field verification
- No recall mechanism for Deploy missions
- Fleet-save destination is the player's own planet but mission type is Transport (visible to phalanx)

**Phase to address:**
Phase 1 (Fleet-Save) — Phalanx safety must be a core design requirement, not a feature you add later. The fleet-save algorithm must choose phalanx-safe missions by default.

---

### Pitfall 6: Request Rate Limiting → IP/Account Ban

**What goes wrong:**
OGame servers enforce requests-per-second limits. Exceeding them triggers IP bans or account bans. TBot warns: "Ogame's servers have a limit on the number of requests per second. If you run too many instances, you may get IP banned." Multi-account setups compound this dramatically.

**Why it happens:**
Bot loops poll aggressively: check attacks, check fleets, check builds, check resources. Each is a request. With multiple features running and multiple accounts, you quickly exceed limits. Developers don't account for the compounding effect of all features running simultaneously.

**How to avoid:**
- Implement a global request throttle/queue — all ogamed API calls go through a single rate limiter
- Target ~1 request per 2-3 seconds minimum (conservative; real players are much slower)
- Track total requests per minute across ALL features, not per-feature
- Add random jitter to all polling intervals (±20-40%)
- Priority queue: fleet-save and attack detection are high priority; auto-build and galaxy scans are low priority
- When running multi-account, stagger the polling loops — don't have all accounts checking at the same second
- Monitor for rate-limit responses from OGame and back off exponentially if detected

**Warning signs:**
- ogamed returns error responses after periods of heavy activity
- Account gets temporary bans that escalate
- Multiple features all have 30-second polling intervals firing simultaneously

**Phase to address:**
Phase 1 (Core) — The rate limiter is infrastructure that every feature depends on. Build it before any feature logic.

---

### Pitfall 7: Bot Behavior Patterns That Scream "I'm a Bot"

**What goes wrong:**
Even with ogamed handling device fingerprinting, the bot's behavioral patterns can trigger detection. Perfect 30-second polling intervals, instant reactions to attacks, building exactly the ROI-optimal building every time with no variation — these are patterns no human exhibits.

**Why it happens:**
Developers focus on the technical anti-detection (user-agent, fingerprinting) but neglect behavioral anti-detection. Gameforge employs behavioral analysis. Key detection vectors:
- Exact-interval polling (every 30.0s on the dot)
- Instant reaction to incoming attacks (humans take minutes to respond)
- 24/7 activity with no sleep periods
- Perfect optimization (always building the mathematically optimal thing)
- Galaxy scanning in perfectly sequential order

**How to avoid:**
- **Randomize all intervals**: If the base interval is 30s, use `30s + random(0, 30s)` — not `30s ± 1s`
- **Implement sleep mode** (TBot and Cruiser both have this): reduce or stop activity during configurable hours
- **Add reaction delay**: Don't react to attacks instantly. Wait a random 30-120 seconds before fleet-saving
- **Vary build decisions**: Occasionally (5-10% of the time) pick the second-best ROI building instead of the optimal one
- **Don't scan galaxies sequentially**: Randomize the scan pattern within the configured range
- **Add "away time" simulation**: Periods of reduced activity that look like a human stepping away
- **TBot's approach**: configurable sleep mode with automatic fleet-save before sleep and resume on wake

**Warning signs:**
- Polling logs show requests at exactly :00, :30, :00, :30 timestamps
- Attack responses always fire within 1 second of detection
- The bot never has any idle time in its action logs

**Phase to address:**
Phase 2 (Anti-Detection + Auto-Build) — The rate limiter from Phase 1 should include jitter from the start, but full behavioral anti-detection (sleep mode, varied decisions) comes when building the automation features.

---

### Pitfall 8: Not Checking Fuel Before Fleet-Save

**What goes wrong:**
Fleet-save calculates it needs X deuterium for fuel, but the planet only has X-100. The send fails. Fleet stays home. TBot's fleet-save failure shows fuel calculation (229,543 deuterium) was done but the send still failed — likely because available fuel was insufficient after accounting for what was being loaded onto transports.

**Why it happens:**
Fuel calculation happens independently from resource loading calculation. The bot calculates fuel cost, calculates how many resources to load onto ships, but doesn't verify that fuel + loaded resources ≤ available resources. Or the bot sends resources to the fleet-save destination that leaves insufficient fuel for the trip.

**How to avoid:**
- Always verify `availableDeuterium >= fuelCost + deuteriumToLoad`
- If insufficient fuel, reduce loaded resources to prioritize fuel
- As a last resort, send fleet at slower speed (saves fuel) with reduced cargo
- Never leave fleet-save to chance — if the primary attempt fails, immediately try the backup plan at slower speed or different destination
- Test edge cases: planet with 0 deuterium, planet with just enough fuel but not for cargo

**Warning signs:**
- Fleet-save log shows fuel calculation but then "Unable to send fleet" error
- Resource amounts in fleet-save logs are close to total planet resources
- Failed fleet-saves happen on low-deuterium planets but not on main planets

**Phase to address:**
Phase 1 (Fleet-Save) — Fuel verification is a hard requirement for fleet-save. Build it into the initial fleet-save logic, not as a bug fix later.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Polling ogamed on fixed intervals | Simple to implement, predictable | Detection signature, rate limit hits | Never in production; OK during initial development |
| Skipping fleet slot availability checks | Faster feature development | Fleet send failures when slots full | Only during early testing with 1-2 planets |
| Hardcoded mission types for fleet-save | Simpler fleet-save logic | Misses edge cases (no friendly planet available) | Never — fleet-save must be robust from day one |
| No ogamed health monitoring | Less infrastructure code | Bot runs blind when ogamed crashes or loses connection | Acceptable in MVP if Telegram alerts are implemented |
| Single-threaded bot logic | Simpler state management | Can't fleet-save while galaxy scanning; one slow operation blocks everything | Acceptable initially, but event loop must be non-blocking |
| Storing credentials in plain config files | Easy setup | Security risk, accidental commits | Development only — use env vars or secrets manager for production |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| ogamed REST API | Assuming responses are always valid JSON | Wrap every ogamed call in try/catch with validation; treat any unexpected response as "API broken" |
| ogamed sessions | Assuming login persists indefinitely | Implement reconnection logic with exponential backoff; monitor for session expiry signals |
| ogamed captcha | Ignoring captcha until it happens | Set up automated captcha solving (Ninja Solver) from day one; implement captcha-state detection |
| ogamed fleet operations | Not checking fleet slots before sending | Always call `GET /bot/fleets` to check available slots before attempting fleet operations |
| ogamed server time | Using local clock for timing | Always use `GET /bot/server/time` for any time-sensitive operations; local clock can drift |
| Telegram notifications | Sending every log event | Only send actionable alerts (attacks, errors, fleet-save events); too many notifications = ignored notifications |
| OGame public API (api.xml) | Not using it for read-only data | Use the public XML API for player rankings, universe data, server info — reduces authenticated requests |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Full empire state refresh every tick | Response time grows with planet count; rate limit hits | Cache state; only refresh what changed; use event-driven updates where possible | 3+ planets |
| Galaxy scanning entire range every cycle | Hundreds of API calls; rate limit bans; detection flag | Scan in chunks across multiple cycles; randomize scan order; cache inactive player lists | Any active auto-farm |
| Synchronous ogamed calls blocking the event loop | Bot becomes unresponsive during slow operations | All ogamed calls should be async with timeouts; don't await indefinitely | First deployment |
| Building ROI calculation for all planets every cycle | CPU spikes; unnecessary recalculations | Recalculate only when build completes or resources change significantly | 5+ planets |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing OGame credentials in config.json | Accidental commit to GitHub; credential theft | Use environment variables or encrypted config; `.gitignore` config files with credentials |
| Exposing ogamed port to the internet | Anyone can control your account via ogamed REST API | Bind ogamed to localhost only; use firewall rules; never expose port 8080 publicly |
| Exposing web dashboard without auth | Anyone can see your empire state and control the bot | Add authentication to web dashboard; use HTTPS; consider VPN-only access |
| Logging sensitive data (credentials, session tokens) | Log files become attack vectors | Sanitize logs; never log passwords or session tokens; rotate log files |
| Sharing cookies file between different lobby accounts | Cross-account contamination; easier to detect as bot network | One cookies file per lobby account (TBot explicitly warns about this) |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silent failures in fleet-save | User thinks fleet is saved, logs in to find fleet destroyed | Fleet-save failure must be a red-alert Telegram notification with siren emoji |
| Too many Telegram notifications | Notification fatigue → user ignores important alerts | Three tiers: 🔴 Critical (attack, fleet-save fail), 🟡 Warning (build errors), ⚪ Info (build complete). Configurable per tier. |
| No way to see what the bot is doing right now | User anxiety → disabling the bot | Web dashboard with live activity feed; "last action" timestamp visible everywhere |
| Config requires restart to take effect | User must choose between "stop bot (lose protection)" or "live with wrong config" | Hot-reload config (TBot supports this); apply changes without restart |
| No "pause" button | User must choose between "kill the bot entirely" or "let it keep doing things" | Granular pause: pause auto-build but keep fleet-save active, or pause everything |

## "Looks Done But Isn't" Checklist

- [ ] **Fleet-Save:** Verify it works when no friendly planet is available for deploy mission — often only tested with multiple colonies
- [ ] **Fleet-Save:** Verify it works when fleet is already in-flight (returning deploy, returning harvest) — must handle in-flight fleets gracefully
- [ ] **Fleet-Save:** Verify fuel calculation accounts for speed modifier — slower speeds save fuel but may not cover attack window
- [ ] **Fleet-Save:** Verify deploy-with-recall timing — recalled fleet must return after attack window
- [ ] **Attack Detection:** Verify it detects ACS (Alliance Combat System) attacks, not just single-player attacks
- [ ] **Attack Detection:** Verify it doesn't false-positive on own returning fleets — TBot has had issues with self-detection
- [ ] **Auto-Build:** Verify ROI calculation accounts for universe speed — a x7 universe has very different ROI timelines than x1
- [ ] **Auto-Build:** Verify it handles "planet full" scenario gracefully — no infinite retry loops
- [ ] **Auto-Farm:** Verify cargo capacity calculation accounts for deuterium cost of the return trip — TBot issue #169
- [ ] **Auto-Farm:** Verify it skips alliance members if configured — TBot issue #185
- [ ] **Expeditions:** Verify expedition slot counting accounts for already-in-flight expeditions — don't over-send
- [ ] **Expeditions:** Verify fleet composition optimization for character class (Discoverer gets different bonuses)
- [ ] **ogamed Integration:** Verify reconnection works after ogamed process restart — not just after session expiry
- [ ] **Telegram:** Verify bot can send messages (not just to bots) — TBot issue #197 about bots-can't-message-bots
- [ ] **Multi-Account:** Verify different ports per instance — TBot explicitly requires this
- [ ] **Multi-Account:** Verify cookie file separation between different lobby accounts

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Fleet destroyed due to fleet-save failure | HIGH (months of progress lost) | No recovery possible. Prevention is the only strategy. Alert user immediately on failure. |
| Account banned for botting | HIGH (account permanently lost) | No recovery possible. Appeal rarely works. Prevention via anti-detection is critical. |
| ogamed broken by game update | MEDIUM (bot offline for hours-days) | Wait for ogamed update; pin to last working version; implement maintenance mode |
| IP banned for rate limiting | MEDIUM (hours to days offline) | Switch proxy/VPN; reduce request frequency; wait for ban to expire |
| Bot process crashes | LOW (minutes of downtime) | Process manager (systemd, PM2) auto-restart; save state to disk for resume |
| Captcha freeze | LOW (minutes if auto-solving; hours if manual) | Implement Ninja Solver; Telegram alert on captcha; manual web UI fallback |
| Config corruption | LOW (minutes) | Git-track config templates; validate config on load; auto-backup working configs |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Fleet-save mission fallback failure | Phase 1: Fleet-Save | Test with: no friendly planets available, occupied destination, empty fleet slots |
| OGame updates breaking API | Phase 1: Core | Test with: mock ogamed returning broken responses; verify health-check triggers |
| Captcha killing bot | Phase 1: Core | Test with: Ninja Solver integration; verify Telegram alert on captcha state |
| Datacenter IP blocking | Phase 1: Core (setup) | Test by: deploying to VPS with residential proxy; verify login succeeds |
| Phalanx-unsafe fleet-save | Phase 1: Fleet-Save | Test with: simulate phalanx-visible missions; verify only deploy+recall and harvest are used |
| Request rate limiting | Phase 1: Core | Test with: all features running simultaneously; verify total requests/minute stays under limit |
| Bot behavior patterns | Phase 2: Anti-Detection | Test with: review polling interval variance in logs; verify sleep mode activates |
| Fuel check before fleet-save | Phase 1: Fleet-Save | Test with: planet at 0 deuterium; planet with barely enough fuel |
| Auto-farm cargo miscalculation | Phase 3: Auto-Farm | Test with: various cargo-to-loot ratios; verify round-trip deuterium included |
| Cookie/session management | Phase 1: Core | Test with: kill ogamed mid-session; verify clean reconnection |
| Multi-account resource contention | Phase 4: Multi-Account | Test with: multiple accounts on same IP; verify rate limiting across accounts |

## Sources

- TBot GitHub issues (open + closed): #178 (failed fleet-save), #169 (cargo calc), #198 (sleep mode errors), #200 (date parsing), #185 (alliance farm), #197 (Telegram bot message issue), multiple captcha issues (#170, #165, #158, #184, #189, #203) — https://github.com/ogame-tbot/TBot/issues
- ogamed GitHub issues: #150 (espionage report broken by v11.15), #148 (constructions broken by v11.13), #145 (Forbidden on startup) — https://github.com/alaingilbert/ogame/issues
- ogamed wiki: auto-captcha using ninja solver, full REST API documentation — https://github.com/alaingilbert/ogame/wiki
- TBot README: explicit warnings about datacenter IP blocking, rate limiting, proxy requirements — https://github.com/ogame-tbot/TBot
- Cruiser source code (bot.py): phalanx-safe fleet-save implementation, deploy-with-recall pattern — https://github.com/kweimann/cruiser
- OGame community knowledge: phalanx mechanics, mission visibility rules, fleet-save best practices — domain expertise from reference projects

---
*Pitfalls research for: OGame automation bot*
*Researched: 2026-04-25*
