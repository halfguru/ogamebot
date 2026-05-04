# Stack Research: OGameX Bot

## Summary

The bot uses a direct HTTP client layer that speaks to OGameX's Laravel/AJAX endpoints. The core insight: **Go's standard library `net/http` with its cookie jar is sufficient for session management** — no external HTTP client library needed. The only additional dependency is **goquery** for HTML parsing to extract CSRF tokens from login pages.

---

## Core Stack (kept from existing)

| Dependency | Version | Purpose | Confidence |
|---|---|---|---|
| Go | 1.26.2 | Language runtime | **High** — current, stable |
| `net/http` (stdlib) | — | HTTP client with `CookieJar` for session cookies | **High** — stdlib, battle-tested |
| `encoding/json` (stdlib) | — | Decode AJAX JSON responses | **High** — already in use |
| `modernc.org/sqlite` | v1.50.0 (current) | State cache, game data persistence | **High** — kept as-is |
| `github.com/gorilla/websocket` | v1.5.3 (current) | Dashboard real-time updates | **High** — kept as-is |
| `gopkg.in/yaml.v3` | v3.0.1 (current) | Configuration files | **High** — kept as-is |
| `golang.org/x/sys` | v0.42.0 (current) | Indirect — syscalls | **High** — transitive |

## New Dependencies Needed

| Dependency | Version | Purpose | Confidence |
|---|---|---|---|
| `github.com/PuerkitoBio/goquery` | v1.12.0 (latest) | HTML parsing for CSRF token extraction from login page | **High** |
| `golang.org/x/net` | v0.53.0 (latest) | Transitive dep of goquery (`html` package) | **High** — automatic |

### Only two new direct imports

That's it. The entire OGameX client can be built with:
- **`net/http`** — cookie jar handles session persistence, manual CSRF token in headers
- **`goquery`** — parse the login page HTML to extract `<meta name="csrf-token">`

## Dependencies (No Longer Used)

| Dependency | Reason |
|---|---|
| Docker for middleware | No longer needed — bot talks directly to OGameX |

The following **internal packages are reused as-is** (import path changes only):
- `internal/model/` — domain types (Planet, Fleet, Resources, etc.)
- `internal/constants/` — ship IDs, building IDs, mission types
- `internal/builder/` — ROI calculator, build engine
- `internal/defender/` — escape route calculator, fleet-save logic
- `internal/farmer/` — farm scanner, attack scheduler
- `internal/state/` — SQLite state manager
- `internal/dashboard/` — web UI server
- `internal/config/` — YAML config (schema changes, same library)

---

## Rationale for Each Choice

### 1. `net/http` + `net/http/cookiejar` (stdlib) — NO external HTTP library

**Why not `go-resty/resty` or `imroc/req`?**
- Session management for Laravel apps is cookie-based. Go's `net/http.Client` with `cookiejar.Jar` handles this natively — the jar automatically stores and sends cookies on subsequent requests.
- CSRF tokens are extracted once at login, then sent as `X-CSRF-TOKEN` header. This is 3 lines of code.
- External HTTP clients (resty, req, heimdall) add complexity without solving any problem we have. They're designed for REST APIs, not browser-session scraping.
- The existing client already uses raw `net/http` — no reason to change the pattern.

**Auth flow (4 steps):**
1. `GET /login` → parse HTML with goquery → extract `<meta name="csrf-token" content="...">`
2. `POST /login` with `email`, `password`, `_token` (CSRF) → cookie jar captures `laravel_session` + `XSRF-TOKEN`
3. On every subsequent request, cookie jar auto-sends both cookies
4. For POST requests, also send `X-CSRF-TOKEN` header (value from `XSRF-TOKEN` cookie, decoded)

**Session refresh:** If a request returns 419 (CSRF token mismatch) or redirects to `/login`, re-login.

### 2. `github.com/PuerkitoBio/goquery` v1.12.0 — HTML parsing

**Why goquery over alternatives?**
- **vs `golang.org/x/net/html` alone**: goquery wraps it with jQuery-like CSS selectors. Extracting `<meta name="csrf-token">` is one line: `doc.Find("meta[name='csrf-token']").Attr("content")`. With raw `x/net/html` you'd write a 20-line tree walker.
- **vs regex**: Fragile, breaks if Laravel changes whitespace/attributes. goquery parses the actual DOM.
- **vs `andybalholm/cascadia` directly**: goquery already includes cascadia — no need for separate import.

**Why not goquery for everything?** Most OGameX endpoints return JSON (AJAX). goquery is only needed for the login page. All game data comes via `ajax/*` endpoints that return JSON directly.

**Minimal usage:** goquery is only used in one function: `extractCSRFToken(htmlBody)`. Everything else is `json.Unmarshal`.

### 3. No browser automation library (no chromedp, rod, etc.)

**Why not?**
- OGameX uses standard HTML forms + AJAX JSON — no JavaScript rendering required for login or API calls.
- All game data comes from `GET /ajax/*` endpoints returning JSON.
- A headless browser would add 100MB+ RAM overhead and an entire Chrome/Chromium dependency.
- The bot needs to be a lightweight single binary.

### 4. No HTTP retry library (keep existing retry logic)

**Why keep custom retry?**
- The existing `retryWithBackoff()` is well-tested (exponential backoff + jitter).
- It will move to a shared package (e.g., `internal/http/retry.go`) with minimal changes.
- Libraries like `hashicorp/go-retryablehttp` are fine but add a dependency for something we already have.

### 5. Rate limiter — keep existing, lower the defaults

**Why keep custom?**
- The existing `RateLimiter` with random jitter per-endpoint is well-suited.
- For OGameX (no anti-bot), defaults should be lower: 200-500ms instead of 2000-5000ms.
- Rate limiting is still useful to avoid hammering the server and getting IP-banned.

---

## Key OGameX Endpoint Map

Derived from `routes/web.php` analysis:

| Bot Operation | Method | Endpoint | Returns |
|---|---|---|---|
| Login | GET | `/login` | HTML (extract CSRF) |
| Login | POST | `/login` | Redirect (session cookie) |
| Resources | GET | `/ajax/resources` | JSON |
| Facilities | GET | `/ajax/facilities` | JSON |
| Research | GET | `/ajax/research` | JSON |
| Shipyard | GET | `/ajax/shipyard` | JSON |
| Defense | GET | `/ajax/defense` | JSON |
| Fleet events | GET | `/ajax/fleet/eventlist/fetch` | JSON |
| Fleet eventbox | GET | `/ajax/fleet/eventbox/fetch` | JSON |
| Build building | POST | `/resources/add-buildrequest` | Redirect/JSON |
| Cancel build | POST | `/resources/cancel-buildrequest` | Redirect/JSON |
| Build ship/def | POST | `/shipyard/add-buildrequest` | Redirect/JSON |
| Send fleet | POST | `/ajax/fleet/dispatch/send-fleet` | JSON |
| Recall fleet | POST | `/ajax/fleet/dispatch/recall-fleet` | JSON |
| Galaxy scan | POST | `/ajax/galaxy` | JSON |
| Messages | GET | `/ajax/messages/{id}` | JSON |
| Planet switch | GET | Any page + `?cp={planetID}` | — |

**Note:** Planet context is set via `?cp=<planet_id>` query parameter on any request. The bot must include this when requesting planet-specific data.

---

## Architecture: New Client Package

```
internal/ogamex/           (OGameX HTTP client)
├── client.go              — Client struct, Login(), session management
├── csrf.go                — CSRF token extraction from HTML
├── ajax.go                — Generic GET/POST wrappers for AJAX endpoints
├── endpoints.go           — ClientInterface implementation (26 methods)
├── types.go               — OGameX-specific response types
└── client_test.go         — Tests with httptest.Server mocks
```

The `ClientInterface` in `client.go` stays identical — workers don't change.

---

## Confidence Levels Summary

| Decision | Confidence | Risk |
|---|---|---|
| `net/http` stdlib for HTTP client | **95%** | None — proven pattern |
| `goquery` for HTML parsing | **95%** | Minimal — only used for login page |
| No browser automation | **95%** | Could change if OGameX adds JS-rendered pages |
| Keep existing retry/rate-limit | **90%** | May need tweaks for OGameX-specific error codes |
| Endpoint map accuracy | **85%** | Based on source code review — may need runtime verification |
| No new HTTP library needed | **95%** | stdlib is sufficient for session cookies + CSRF |
| Session refresh strategy | **80%** | Laravel session lifetime is configurable; bot may need to handle token rotation |

---

## What NOT to Use

| Library / Approach | Why Not |
|---|---|
| `go-resty/resty` | Adds dependency for functionality stdlib provides |
| `imroc/req/v3` | Same as resty — unnecessary abstraction |
| `chromedp` / `rod` | Headless browser — massive overhead, no JS rendering needed |
| `hashicorp/go-retryablehttp` | We have retry logic already |
| `golang.org/x/net/html` directly | Too low-level; goquery wraps it ergonomically |
| Regex for HTML parsing | Fragile; goquery handles DOM properly |
| `andybalholm/cascadia` directly | goquery includes it — redundant to import separately |
| `PuerkitoBio/goquery` for JSON endpoints | Unnecessary — JSON endpoints use `encoding/json` directly |

---

*Researched: 2026-05-03*
*Sources: OGameX GitHub repo (routes/web.php, config/fortify.php), Go module registry, existing codebase analysis*
