# Pitfalls Research: OGameX Bot

Research based on analysis of OGameX source code (github.com/lanedirt/OGameX, Laravel 12.x), the OGameX client (`internal/ogamex/`), and common OGame bot failure modes.

---

## Laravel Session/CSRF Pitfalls

### 1. CSRF Token Rotation After Every AJAX Call

OGameX rotates CSRF tokens on nearly every AJAX response. Every JSON response includes `newAjaxToken` — the old token is immediately invalidated. If your client doesn't capture the new token from each response, the next POST will fail with HTTP 419 (CSRF token mismatch).

**Evidence**: FleetController `dispatchSendFleet` returns `'newAjaxToken' => csrf_token()`. GalaxyController `ajax` returns `'newAjaxToken' => csrf_token()`. FleetEventsController `fetchEventBox` returns `'newAjaxToken' => csrf_token()`. Every. Single. Response.

**Warning signs**: HTTP 419 errors that only appear after the first successful request. Errors disappear on restart (fresh login = fresh token) then recur.

**Prevention**: The client must extract and store the `newAjaxToken` field from every response, not just login. Design the HTTP layer to parse this from all responses automatically. Token storage must be thread-safe if goroutines make concurrent requests.

**Phase**: Phase 1 (client layer) — build token rotation into the base HTTP transport, not individual endpoint methods.

### 2. Session Cookie Lifecycle and Expiry

Laravel sessions expire after a configurable lifetime (default 120 minutes of inactivity in `config/session.php`). The `laravel_session` cookie must be preserved across all requests. Unlike a stateless bearer token, Laravel sessions are stateful — the server tracks last activity.

**Warning signs**: After a quiet period (no API calls for 2+ hours), all requests return redirect to `/login` or 401. The bot appears to be running but is actually getting auth errors silently.

**Prevention**:
- Implement a heartbeat that makes a lightweight request (e.g., `GET /ajax/fleet/eventbox/fetch`) every 30-60 minutes even when idle.
- Detect session expiry by checking for redirect responses or missing `laravel_session` cookie changes.
- Auto-re-login on session expiry (re-POST `/login` with credentials + new CSRF token).

**Phase**: Phase 1 — session keeper goroutine in the client.

### 3. Login is Multi-Step (Not a Single Call)

OGameX login via Laravel Fortify is:
1. `GET /login` → extract CSRF token from HTML `<meta name="csrf-token">` or hidden `_token` field
2. `POST /login` with `email`, `password`, `_token` → receives `laravel_session` cookie + redirect to `/overview`
3. Extract the new CSRF token from the redirect page (it won't be in the login POST response)

**Warning signs**: Login POST succeeds (302 redirect) but subsequent AJAX calls fail with 419 — you captured the session cookie but not the post-login CSRF token.

**Prevention**: After login, make one GET request (e.g., `/overview`) to establish the working CSRF token. Parse the token from the HTML meta tag or a lightweight AJAX response.

**Phase**: Phase 1 — the `Login()` method must handle all three steps.

### 4. Cookie Jar Must Be Shared Across All Requests

Go's `http.Client` has a built-in cookie jar, but it must be explicitly configured. If you create new clients or forget to set the jar, session cookies won't persist between requests.

**Warning signs**: Every request appears to work in isolation but the server doesn't recognize the session. Login works but immediate subsequent calls fail.

**Prevention**: Use a single `http.Client` instance with `cookiejar.Jar` for the entire session lifecycle. Never create per-request clients.

**Phase**: Phase 1 — single client construction.

---

## OGameX API-Specific Pitfalls

### 5. Mixed Response Formats (HTML + JSON + HTML-inside-JSON)

This is the single biggest difference from a typical REST API. The previous client had a clean JSON envelope for everything. OGameX mixes three response types:

| Endpoint | Response Type | Notes |
|----------|--------------|-------|
| `/ajax/fleet/eventbox/fetch` | JSON | Fleet summary counts |
| `/ajax/fleet/eventlist/fetch` | **HTML** (Blade view) | Full fleet movement list — NOT JSON |
| `/ajax/resources?technology=1` | JSON with HTML string | `content.technologydetails` contains rendered HTML |
| `/ajax/fleet/dispatch/send-fleet` | JSON | `{success, message, newAjaxToken}` |
| `/ajax/galaxy` | JSON | Galaxy scan results |
| `/resources/add-buildrequest` | JSON | `{status, message}` |
| `/overview` | Full HTML page | Planet data embedded in page |
| `/fleet` | Full HTML page | Ship counts, fleet slots in page |

**Warning signs**: JSON unmarshal errors on endpoints that return HTML. Golang code trying to parse Blade templates as JSON. Silent failures when response format doesn't match expectations.

**Prevention**: Classify every endpoint by response type during Phase 1. Create separate parsing methods:
- `getJSON()` — for pure JSON endpoints (most `/ajax/` endpoints)
- `getHTML()` — for page endpoints that return Blade views
- The fleet event list (`/ajax/fleet/eventlist/fetch`) needs HTML parsing or you need to find an alternative way to get fleet data

**Phase**: Phase 1 — endpoint classification + per-type parsers.

### 6. No Single "Get Full Game State" Endpoint

OGameX has NO single state endpoint. State is scattered:

- **Planet list**: Embedded in the overview HTML page as a planet menu widget
- **Current planet resources**: Available via the resource bar (rendered in every page's header)
- **Buildings on a planet**: Must load the resources/facilities page and parse HTML, OR use the AJAX endpoint per-building
- **Active fleets**: `/ajax/fleet/eventbox/fetch` gives counts only; `/ajax/fleet/eventlist/fetch` gives an HTML fragment
- **Research levels**: Embedded in research page HTML

**Warning signs**: You build a `GetPlanets()` method but there's no `/ajax/planets` endpoint. You try to get resources but the AJAX endpoint requires a `technology` parameter for individual building details, not bulk state.

**Prevention**: Plan to parse HTML for initial state discovery. The resource bar (present on every page) contains current planet resources, planet list, and current planet ID. Build an HTML parser for the overview page early.

**Phase**: Phase 1 — HTML parsing infrastructure. This is the most underestimated work item.

### 7. Fleet Dispatch is a Two-Step Process

OGameX requires:

1. **Step 1**: `POST /ajax/fleet/dispatch/check-target` with `{galaxy, system, position, type, am202: 1, ...}` → returns `orders` (available missions), `shipsData`, and target info
2. **Step 2**: `POST /ajax/fleet/dispatch/send-fleet` with `{galaxy, system, position, type, mission, speed, am202: count, metal, crystal, deuterium, token}` → returns `{success, newAjaxToken}`

The `token` field in step 2 must be the `newAjaxToken` from step 1. Speed is sent as a float 0.5–10.0 (not percentage), where 10 = 100%. General class gets 0.5 increments; others get 1.0 increments.

**Warning signs**: Fleet dispatch fails silently (returns `success: false`) because you skipped `check-target` or used a stale token. Or speed validation fails because you sent `100` instead of `10`.

**Prevention**: Always call `check-target` before `send-fleet`. Pass the token from check-target to send-fleet. Map speed from the bot's internal representation (1-100%) to OGameX format (0.5-10).

**Phase**: Phase 2 (fleet safety) — implement in the `SendFleet` method.

### 8. Ship Quantities Sent as Individual Form Fields, Not Arrays

Other OGame bots use `ships=202,5&ships=203,10` (repeated parameter with comma-separated values). OGameX uses individual fields: `am202=5&am203=10` where the field name is `am` + the ship ID.

**Evidence**: FleetController `dispatchSendFleet` extracts ships via `$this->getUnitsFromRequest()` which loops all input fields prefixed with "am".

**Warning signs**: Fleet dispatch returns "no ships selected" error. The ships you sent aren't recognized.

**Prevention**: Convert the `[]ShipQty` array to `am{shipID}={count}` form fields. Remember to include ships with count 0 if you need to indicate "no ships of this type" (or just omit them).

**Phase**: Phase 1 — in the SendFleet request builder.

### 9. Planet Switching via Query Parameter

OGameX determines the "current planet" from the `?cp=<planet_id>` query parameter. Without it, the server uses the player's default/current planet. This means:

- Every request must include `?cp=<planet_id>` if you're querying a specific planet
- After switching planets, all subsequent requests use the new planet until you switch again
- This is stateful on the server side — the session remembers the last planet

**Warning signs**: GetResources returns data for the wrong planet. Build requests go to the wrong planet. Intermittent failures that depend on which planet was last accessed.

**Prevention**: Always include `?cp=<planet_id>` on every request. Never rely on server-side "current planet" state. Build this into the base request method.

**Phase**: Phase 1 — base request method parameter.

### 10. Event List Returns HTML, Not Structured Fleet Data

The fleet event list endpoint (`/ajax/fleet/eventlist/fetch`) returns a rendered Blade template, not JSON. The event box endpoint (`/ajax/fleet/eventbox/fetch`) returns JSON but only has summary counts (friendly/hostile/neutral counts and next event time).

For the defender worker, you need to know about specific incoming hostile fleets (origin, arrival time, mission type). The event list HTML contains this data but requires parsing.

**Warning signs**: You try to JSON-unmarshal the event list response and get parse errors. Or you only use eventbox data and can't determine which attacks are incoming vs. outgoing.

**Prevention**: Either:
- (a) Build an HTML parser for the event list Blade template
- (b) Use the fleet movement page (`/fleet/movement`) which also has fleet data in HTML
- (c) Find or request a JSON endpoint for fleet data (OGameX is open-source — you could contribute one)

**Phase**: Phase 1 (investigation) / Phase 2 (implementation for defender).

### 11. Building Construction Uses `technologyId`, Not Building Machine Name

The build endpoint (`/resources/add-buildrequest`) accepts `technologyId` (numeric ID like `1` for metal mine) and `_token` (CSRF token). The response format varies:
- Success: `{status: "success", message: "..."}`
- Error: `{success: false, message: "..."}` or `{success: false, errors: [{message: "..."}]}`

Note the inconsistency: success uses `status`, error uses `success`. And the CSRF token is explicitly checked via `hash_equals()` in `AbstractBuildingsController`.

**Warning signs**: Build requests fail with "Invalid token" even though you're sending a CSRF token. You're sending the header token but the controller reads `_token` from form body.

**Prevention**: Send `_token` as a form field (not header). The controller checks `$request->input('_token')`, not headers. Use the standard OGame building IDs (1=metal mine, 2=crystal mine, etc.).

**Phase**: Phase 1 (client) / Phase 3 (builder worker).

### 12. OGameX May Diverge from Official OGame IDs

OGameX uses standard OGame object IDs (202=Small Cargo, 1=Metal Mine) but is an independent implementation. There may be edge cases where IDs differ, especially for newer features or OGameX-specific additions.

The OGameX `ObjectService::getObjectById()` maps IDs to internal objects, and the response format uses these IDs. But OGameX doesn't implement all OGame features (e.g., Lifeforms), so some IDs may not exist.

**Warning signs**: Building/ship IDs that work on official OGame return "object not found" errors on OGameX. Or OGameX has additional objects with IDs the bot doesn't know about.

**Prevention**: During Phase 1, enumerate all valid object IDs by querying the techtree or AJAX endpoints. Don't assume the full official OGame ID space is available. Make the constants table configurable/overridable.

**Phase**: Phase 1 — validate ID mapping against live OGameX instance.

---

## Migration Pitfalls (Client Implementation)

### 13. Response Format Variations

Each endpoint uses its own response structure:
- `{success: bool, message: string, newAjaxToken: string}`
- `{status: string, message: string}`
- `{success: bool, errors: [{message: string, error: int}]}`
- `{components: [], newAjaxToken: string, hostile: int, ...}` (eventbox)

**Warning signs**: You try to reuse `getTyped[T]` and every endpoint fails with "unexpected JSON structure". Generic unmarshaling returns zero values.

**Prevention**: Create per-endpoint response types. Don't try to force a single envelope pattern. Accept that OGameX response structure is endpoint-specific.

**Phase**: Phase 1 — new response types for each endpoint.

### 14. Case Sensitivity in JSON Field Names

OGameX uses a mix of camelCase and snake_case (`newAjaxToken`, `targetPlayerId`, `fleet_unit_count`).

**Warning signs**: Struct fields are always zero/empty after unmarshaling. You copy tags from a different source.

**Prevention**: Use the actual field names from OGameX responses. Use `json:` tags matching OGameX's actual output.

**Phase**: Phase 1 — all response structs.

### 15. Error Handling Is Fundamentally Different

Error responses vary: HTTP 419 (CSRF), HTTP 500 (server error), JSON `{success: false, errors: [...]}`, or a redirect to `/login` (expired session).

**Warning signs**: Error checking code expects a single error format but OGameX returns different formats. HTTP redirects are followed silently, returning the login page HTML as if it were a successful response.

**Prevention**: 
- Don't follow redirects automatically (check for 302 to `/login`).
- Check HTTP status codes first (419, 500, 302).
- Parse response body for `success: false` or `status: "failure"`.
- Handle multiple error formats.

**Phase**: Phase 1 — error detection layer.

### 16. ClientInterface Methods Don't Map 1:1 to OGameX

The `ClientInterface` has methods like `GetAttacks()`, `GetSlots()`, `IsUnderAttack()` that require creative mapping to OGameX endpoints:

| Method | OGameX Equivalent |

**Warning signs**: You implement the interface but each method requires complex HTML parsing instead of a simple API call. Methods return partial or incorrect data.

**Prevention**: Accept that `ClientInterface` will need modification. Some methods may return aggregated data from multiple endpoints. Consider expanding the interface or adding helper methods that combine multiple OGameX calls.

**Phase**: Phase 1 — redesign ClientInterface to match OGameX reality.

### 17. Rate Limiting Has Different Constraints

OGameX has no explicit rate limiting, but Laravel has built-in throttling on login (default 5 attempts/minute). More importantly, OGameX is a full web app — each request does more work (DB queries, session handling, view rendering) than a lightweight API.

**Warning signs**: After running the bot for a while, responses become slow. The OGameX demo server (main.ogamex.dev) may throttle or IP-ban aggressive clients.

**Prevention**: Keep a rate limiter. Add longer delays between heavy operations (galaxy scans, fleet dispatches). Be respectful of the shared demo server.

**Phase**: Phase 1 — carry over rate limiter, add OGameX-specific delays.

---

## Architecture Pitfalls

### 18. HTML Parsing Fragility

OGameX renders HTML via Laravel Blade templates. Any OGameX update can change the HTML structure, breaking parsers. The main.ogamex.dev demo runs the latest `main` branch and can update at any time.

**Warning signs**: Bot works one day, breaks the next with no code changes. Parse errors on HTML that "looks fine" but has a subtle whitespace or attribute change.

**Prevention**:
- Use CSS selectors (goquery) rather than regex or string matching for HTML parsing.
- Build a "health check" that validates parsed data looks reasonable (e.g., resource count > 0, planet count > 0).
- Log raw HTML responses at debug level so you can diagnose breakage.
- Consider contributing a proper JSON API to OGameX upstream.

**Phase**: Phase 1 — HTML parsing infrastructure with health checks.

### 19. Concurrent Request Token Conflicts

The bot has multiple goroutines (defender, builder, farmer, state manager) all making requests through the same client. If goroutine A's response rotates the CSRF token while goroutine B is constructing a POST with the old token, B's request fails.

**Warning signs**: Intermittent 419 errors that increase with bot activity. Errors that don't reproduce in single-threaded tests.

**Prevention**: Serialize CSRF token updates. Use a mutex around the token read/write. Consider a request queue that processes one request at a time, ensuring token rotation is sequential. Or use a per-goroutine token that's refreshed after each response.

**Phase**: Phase 1 — thread-safe token management.

### 20. Don't Over-Engineer the HTML Parser

Resist the urge to build a generic "OGameX page parser" that handles every page. You only need data for specific bot functions:

- Overview page: planet list, current resources, current planet ID
- Fleet event list: incoming attacks, active fleets, slot counts
- Research page: research levels

Building a general parser is a bottomless time sink.

**Warning signs**: You're spending more time on HTML parsing than on bot logic. The parser handles edge cases that don't affect the bot.

**Prevention**: Parse only what you need. Start with the minimum viable data extraction for each function. Add parsing incrementally as needed.

**Phase**: All phases — keep parser scope tight.

### 21. Native Binary Means No Container Networking

The bot is a single Go binary with no Docker dependencies. This means:
- Direct URL connections (e.g., `https://main.ogamex.dev`)
- No container restart policies
- No built-in log aggregation

**Prevention**: Use direct URLs in configuration. The bot is a native binary — treat it as such.

**Phase**: Phase 1 — config cleanup.

---

## Prevention Strategies per Phase

### Phase 1: OGameX Client Layer

This is the highest-risk phase. Pitfalls are concentrated here.

| Pitfall | Prevention | Priority |
|---------|-----------|----------|
| CSRF token rotation (#1) | Auto-extract from every response, mutex-protected storage | **Critical** |
| Session cookie lifecycle (#2) | Heartbeat goroutine, auto-re-login | **Critical** |
| Multi-step login (#3) | Login = GET token + POST credentials + GET overview | **Critical** |
| Mixed response formats (#5) | Per-endpoint response type classification | **Critical** |
| No get-state endpoint (#6) | HTML parser for overview page | **Critical** |
| HTML fleet events (#10) | HTML parser for event list OR use eventbox + movement page | High |
| Planet switching (#9) | Always send `?cp=` parameter | High |
| Concurrent token conflicts (#19) | Mutex or sequential request queue | High |
| Envelope change (#13) | New per-endpoint response types | High |
| Case sensitivity (#14) | New structs with correct json tags | Medium |
| Error handling (#15) | Multi-format error detection | High |
| ClientInterface mismatch (#16) | Redesign interface for OGameX reality | High |
| HTML parsing fragility (#18) | CSS selectors + health checks | Medium |

### Phase 2: Fleet Safety (Defender)

| Pitfall | Prevention | Priority |
|---------|-----------|----------|
| Fleet dispatch is two-step (#7) | Always check-target before send-fleet | **Critical** |
| Ship form fields (#8) | Convert to `am{ID}={count}` format | High |
| Attack detection via HTML (#10) | Parse event list for hostile missions | **Critical** |
| Token staleness during emergency (#19) | Ensure fresh token before fleet-save | High |

### Phase 3: Auto-Build

| Pitfall | Prevention | Priority |
|---------|-----------|----------|
| Build uses `technologyId` (#11) | Send `_token` as form field + numeric building ID | High |
| Inconsistent success/error format (#11) | Check both `status` and `success` fields | Medium |
| Wrong planet builds (#9) | Always include `?cp=` | High |

### Phase 4: Auto-Farm

| Pitfall | Prevention | Priority |
|---------|-----------|----------|
| Galaxy scan is expensive (#17) | Longer delays between scans | Medium |
| Mini-fleet espionage (#7) | Use `dispatchSendMiniFleet` for galaxy-page espionage | Medium |
| ID divergence (#12) | Validate object IDs against live server | Low |

### Phase 5: Web Dashboard

| Pitfall | Prevention | Priority |
|---------|-----------|----------|
| No Docker (#21) | Use direct URLs in config | Low |

---

## Summary of Top-5 Risks

1. **CSRF token rotation** (#1) — Will silently break all POST operations if not handled. Must be automatic and thread-safe.
2. **No structured state endpoints** (#6) — The bot must parse HTML to get basic game state. This is fragile and underestimated.
3. **Mixed response formats** (#5) — Endpoints return HTML, JSON, or JSON-with-HTML. Treating everything as JSON will cause pervasive parse failures.
4. **Multi-step fleet dispatch** (#7) — Fleet-save (the core value) requires two sequential requests with token handoff. A bug here means the bot can't save your fleet.
5. **Concurrent token conflicts** (#19) — Multiple goroutines sharing a rotating token is a race condition that only manifests under load.
