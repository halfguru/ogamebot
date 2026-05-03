# OGameX HTTP Endpoints Research

Source: github.com/lanedirt/OGameX (Laravel PHP app)
Route definitions: `routes/web.php`
Controllers: `app/Http/Controllers/`

## Authentication & CSRF

- Session-based auth (Laravel default: `laravel_session` cookie)
- CSRF token required on all POST requests
- CSRF token can be sent as:
  - Form field `_token` (building/research endpoints explicitly check this)
  - Form field `token` (fleet dispatch endpoints)
  - Header `X-CSRF-TOKEN` (Laravel standard alternative)
- CSRF token is embedded in HTML pages via `csrf_token()` (typically in `<meta>` tag or inline JS)
- JSON responses return `newAjaxToken` — the client must use this for the next AJAX request
- All game routes are behind middleware: `['auth', 'banned', 'globalgame', 'locale', 'firstlogin']`
- Login endpoint: standard Laravel `POST /login` with `email`, `password`, `_token`

---

## 1. Fleet Dispatch — Check Target (Step 1 of 2)

| Property | Value |
|----------|-------|
| **URL** | `POST /ajax/fleet/dispatch/check-target` |
| **Method** | POST |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `galaxy` | int | yes | Target galaxy (1-9) |
| `system` | int | yes | Target system (1-499) |
| `position` | int | yes | Target position (1-16) |
| `type` | int | yes | Planet type: 1=Planet, 2=Debris, 3=Moon |
| `union` | int | no | Union ID for ACS attacks |
| `token` | string | yes | CSRF token |

### Response (JSON)
```json
{
  "shipsData": {
    "202": {
      "id": 202,
      "name": "Small Cargo",
      "baseFuelCapacity": 5000,
      "baseCargoCapacity": 5000,
      "fuelConsumption": 3,
      "speed": 7500
    }
  },
  "status": "success",
  "errors": [],
  "targetInhabited": true,
  "targetPlayerId": 123,
  "targetPlayerName": "Player1",
  "targetPlanet": {
    "galaxy": 1,
    "system": 1,
    "position": 12,
    "type": 1,
    "name": "Homeworld"
  },
  "orders": {
    "1": true, "2": false, "3": true, "4": false,
    "5": false, "6": true, "7": false, "8": false,
    "9": false, "15": false
  },
  "newAjaxToken": "abc123..."
}
```

### Mission Types (orders keys)
| ID | Mission |
|----|---------|
| 1 | Attack |
| 2 | ACS Attack |
| 3 | Transport |
| 4 | Deploy |
| 5 | ACS Defend (Hold) |
| 6 | Espionage |
| 7 | Colonize |
| 8 | Recycle |
| 9 | Destroy Moon |
| 10 | Missile Attack (not in orders) |
| 15 | Expedition |

---

## 2. Fleet Dispatch — Send Fleet (Step 2 of 2)

| Property | Value |
|----------|-------|
| **URL** | `POST /ajax/fleet/dispatch/send-fleet` |
| **Method** | POST |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | yes | CSRF token |
| `am202` | int | no* | Small Cargo count (field prefix `am` + ship ID) |
| `am203` | int | no* | Large Cargo count |
| `am204` | int | no* | Light Fighter count |
| `am205` | int | no* | Heavy Fighter count |
| `am206` | int | no* | Cruiser count |
| `am207` | int | no* | Battleship count |
| `am208` | int | no* | Colony Ship count |
| `am209` | int | no* | Recycler count |
| `am210` | int | no* | Espionage Probe count |
| `am211` | int | no* | Bomber count |
| `am213` | int | no* | Destroyer count |
| `am214` | int | no* | Deathstar count |
| `am215` | int | no* | Battlecruiser count |
| `am218` | int | no* | Reaper count |
| `am219` | int | no* | Pathfinder count |
| `galaxy` | int | yes | Target galaxy |
| `system` | int | yes | Target system |
| `position` | int | yes | Target position |
| `type` | int | yes | Planet type (1=Planet, 2=Debris, 3=Moon) |
| `metal` | int | yes | Metal to load (0 if none) |
| `crystal` | int | yes | Crystal to load (0 if none) |
| `deuterium` | int | yes | Deuterium to load (0 if none) |
| `mission` | int | yes | Mission type ID (see table above) |
| `speed` | float | yes | Speed (1.0-10.0 in 0.5 steps; 10 = 100%) |
| `retreatAfterDefenderRetreat` | int | no | 0 or 1 |
| `lootFoodOnAttack` | int | no | 0 or 1 |
| `union` | int | no | Union ID for ACS (0 = none) |
| `holdingtime` | int | no | Hold hours (expeditions: 1 to astrophysics level) |

*Ship fields: only include ships you want to send. Ship IDs map to the `am{ID}` pattern.

### Response — Success (JSON)
```json
{
  "success": true,
  "message": "Your fleet has been successfully sent.",
  "components": [],
  "newAjaxToken": "new_csrf_token_here",
  "redirectUrl": "/fleet"
}
```

### Response — Failure (JSON)
```json
{
  "success": false,
  "errors": [
    { "message": "Error description", "error": 140020 }
  ],
  "components": [],
  "newAjaxToken": "new_csrf_token_here"
}
```

---

## 3. Fleet Dispatch — Mini Fleet (Galaxy Page Shortcuts)

| Property | Value |
|----------|-------|
| **URL** | `POST /ajax/fleet/dispatch/send-mini-fleet` |
| **Method** | POST |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

Used for espionage and recycle shortcuts from galaxy page. Automatically selects ship type.

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `galaxy` | int | yes | Target galaxy |
| `system` | int | yes | Target system |
| `position` | int | yes | Target position |
| `type` | int | yes | Planet type |
| `mission` | int | yes | Only 6 (Espionage) or 8 (Recycle) |
| `shipCount` | int | no | Ship count (defaults to 1) |

### Response (JSON)
```json
{
  "response": {
    "message": "Fleet dispatched",
    "type": 1,
    "slots": 1,
    "probes": 11,
    "recyclers": 0,
    "shipsSent": 3,
    "coordinates": { "galaxy": 1, "system": 50, "position": 8 },
    "planetType": 1,
    "success": true
  },
  "newAjaxToken": "...",
  "components": []
}
```

---

## 4. Fleet Cancel (Recall)

| Property | Value |
|----------|-------|
| **URL** | `POST /ajax/fleet/dispatch/recall-fleet` |
| **Method** | POST |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `fleet_mission_id` | int | yes | The fleet mission ID to recall |

### Response — Success (JSON)
```json
{
  "components": [],
  "newAjaxToken": "...",
  "success": true
}
```

### Response — Failure (JSON, HTTP 500)
```json
{
  "components": [],
  "newAjaxToken": "...",
  "success": false
}
```

Notes:
- Only the fleet owner can recall
- Cannot recall already-canceled missions
- Missile attacks (type 10) cannot be recalled
- Planet relocation transfers cannot be recalled

---

## 5. Building Upgrade — Resources (Mines, Solar, Storage)

| Property | Value |
|----------|-------|
| **URL** | `POST /resources/add-buildrequest` |
| **Alt URL** | `GET /resources/add-buildrequest` (also supported) |
| **Method** | POST (preferred) |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `technologyId` | string | yes | Building ID (see table below) |
| `_token` | string | yes | CSRF token |
| `mode` | int | no | 3 = downgrade (omit for upgrade) |

### Building IDs (Resources page)
| ID | Building | Machine Name |
|----|----------|--------------|
| 1 | Metal Mine | `metal_mine` |
| 2 | Crystal Mine | `crystal_mine` |
| 3 | Deuterium Synthesizer | `deuterium_synthesizer` |
| 4 | Solar Plant | `solar_plant` |
| 12 | Fusion Reactor | `fusion_plant` |
| 212 | Solar Satellite | `solar_satellite` (routed to ShipyardController) |
| 217 | Crawler | `crawler` (routed to ShipyardController) |
| 22 | Metal Storage | `metal_store` |
| 23 | Crystal Storage | `crystal_store` |
| 24 | Deuterium Storage | `deuterium_store` |

### Response (JSON)
```json
{ "status": "success", "message": "Building construction started." }
```
or on error:
```json
{ "success": false, "message": "Error reason" }
```

---

## 6. Building Upgrade — Facilities (Robotics, Shipyard, Lab, etc.)

| Property | Value |
|----------|-------|
| **URL** | `POST /facilities/add-buildrequest` |
| **Alt URL** | `GET /facilities/add-buildrequest` (also supported) |
| **Method** | POST (preferred) |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

### Form Data
Same format as Resources: `technologyId`, `_token`, optional `mode=3` for downgrade.

### Building IDs (Facilities page — Planet)
| ID | Building | Machine Name |
|----|----------|--------------|
| 14 | Robotics Factory | `robot_factory` |
| 21 | Shipyard | `shipyard` |
| 31 | Research Lab | `research_lab` |
| 34 | Alliance Depot | `alliance_depot` |
| 44 | Missile Silo | `missile_silo` |
| 15 | Nanite Factory | `nano_factory` |
| 33 | Terraformer | `terraformer` |
| 36 | Space Dock | `space_dock` |

### Building IDs (Facilities page — Moon)
| ID | Building | Machine Name |
|----|----------|--------------|
| 14 | Robotics Factory | `robot_factory` |
| 21 | Shipyard | `shipyard` |
| 41 | Lunar Base | `lunar_base` |
| 42 | Sensor Phalanx | `sensor_phalanx` |
| 43 | Jump Gate | `jump_gate` |

---

## 7. Building Cancel

| Property | Value |
|----------|-------|
| **URL** | `POST /resources/cancel-buildrequest` |
| **URL** | `POST /facilities/cancel-buildrequest` |
| **Method** | POST |
| **Response** | JSON |

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `technologyId` | string | yes | Building ID |
| `listId` | string | yes | Queue item ID to cancel |

---

## 8. Research Upgrade

| Property | Value |
|----------|-------|
| **URL** | `POST /research/add-buildrequest` |
| **Alt URL** | `GET /research/add-buildrequest` (also supported) |
| **Method** | POST (preferred) |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `technologyId` | string | yes | Research ID (see table below) |
| `_token` | string | yes | CSRF token |

### Research IDs
| ID | Research | Machine Name |
|----|----------|--------------|
| 113 | Energy Technology | `energy_technology` |
| 120 | Laser Technology | `laser_technology` |
| 121 | Ion Technology | `ion_technology` |
| 114 | Hyperspace Technology | `hyperspace_technology` |
| 122 | Plasma Technology | `plasma_technology` |
| 115 | Combustion Drive | `combustion_drive` |
| 117 | Impulse Drive | `impulse_drive` |
| 118 | Hyperspace Drive | `hyperspace_drive` |
| 106 | Espionage Technology | `espionage_technology` |
| 108 | Computer Technology | `computer_technology` |
| 124 | Astrophysics | `astrophysics` |
| 123 | Intergalactic Research Network | `intergalactic_research_network` |
| 199 | Graviton Technology | `graviton_technology` |
| 109 | Weapon Technology | `weapon_technology` |
| 110 | Shielding Technology | `shielding_technology` |
| 111 | Armor Technology | `armor_technology` |

### Response (JSON)
```json
{ "status": "success", "message": "Building construction started." }
```

### Research Cancel
| Property | Value |
|----------|-------|
| **URL** | `POST /research/cancel-buildrequest` |
| **Form Data** | `technologyId`, `listId` |

---

## 9. Galaxy Scan

| Property | Value |
|----------|-------|
| **URL** | `POST /ajax/galaxy` |
| **Method** | POST |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **Response** | JSON |

### Form Data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `galaxy` | int | yes | Galaxy number (1-9) |
| `system` | int | yes | System number (1-499) |

### Response (JSON)
```json
{
  "components": [],
  "newAjaxToken": "...",
  "reservedPositions": [],
  "success": true,
  "system": {
    "availableMissiles": 0,
    "availablePathfinders": 0,
    "availableProbes": 5,
    "availableRecyclers": 0,
    "canColonize": true,
    "canExpedition": true,
    "canFly": true,
    "canSystemPhalanx": false,
    "currentPlanetId": 123,
    "galaxy": 1,
    "system": 50,
    "galaxyContent": [
      {
        "actions": {
          "canEspionage": true,
          "canPhalanx": false,
          "canSendProbes": true,
          "canMissileAttack": false,
          "canBuddyRequests": true
        },
        "availableMissions": [],
        "galaxy": 1,
        "planets": [
          {
            "activity": { "idleTime": null, "showActivity": 15, "showMinutes": true },
            "availableMissions": [
              { "missionType": 3, "link": "/fleet?...", "name": "Transport" },
              { "missionType": 6, "canSpy": true, "link": "...", "name": "Espionage" },
              { "missionType": 1, "link": "...", "name": "Attack" }
            ],
            "fleet": [],
            "imageInformation": "desert_driedup",
            "isDestroyed": false,
            "planetId": 456,
            "planetName": "Planet1",
            "playerId": 789,
            "planetType": 1
          }
        ],
        "player": {
          "playerId": 789,
          "playerName": "Player1",
          "isAdmin": false,
          "isInactive": false,
          "isLongInactive": false,
          "isNewbie": false,
          "isStrong": false,
          "isOnVacation": false,
          "isBanned": false,
          "allianceId": null,
          "allianceTag": null,
          "actions": {
            "alliance": { "available": false },
            "highscore": { "available": true, "rank": 42 }
          }
        },
        "position": 1,
        "system": 50
      }
    ],
    "hasAdmiral": false,
    "playerId": 100,
    "slotsColonized": 5,
    "usedFleetSlots": 1
  }
}
```

Notes:
- `galaxyContent` is an array of 15 rows (positions 1-15), each with `planets` array
- Empty positions have `planets: []` and `player: { playerId: 99999, playerName: "Deep space" }`
- Moon entries appear as additional items in the `planets` array with `planetType: 3`
- Debris fields appear as additional items in the `planets` array with `planetType: 2`
- Position 16 (expedition) may appear for Discoverer class if debris exists

---

## 10. Messages / Espionage Reports

### List Messages (HTML page)
| Property | Value |
|----------|-------|
| **URL** | `GET /messages` |
| **Query Params** | `tab=fleets`, `subtab=espionage`, `pagination={page}` |
| **Response** | HTML page |

### AJAX Tab Contents (HTML fragment)
| Property | Value |
|----------|-------|
| **URL** | `GET /ajax/messages` |
| **Query Params** | `tab=fleets`, `subtab=espionage`, `pagination={page}` |
| **Response** | HTML fragment |

### Tabs and Subtabs
| Tab | Subtabs |
|-----|---------|
| `fleets` | `espionage`, `combat_reports`, `expeditions`, `transport`, `other` |
| `communication` | `messages`, `information` |
| `economy` | (default) |
| `universe` | (default) |
| `system` | (default) |
| `favorites` | (default) |

### Get Individual Message (HTML fragment)
| Property | Value |
|----------|-------|
| **URL** | `GET /ajax/messages/{messageId}` |
| **Query Params** | `tab=fleets`, `subtab=espionage` |
| **Response** | HTML fragment (full message body with pagination context) |

### Delete Message
| Property | Value |
|----------|-------|
| **URL** | `POST /messages` |
| **Method** | POST |
| **Response** | JSON |

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | int | yes | `103` for delete |
| `messageId` | int | yes | Message ID to delete |

Response:
```json
{ "12345": true }
```

---

## 11. Fleet Movement (overview of active fleets)

| Property | Value |
|----------|-------|
| **URL** | `GET /fleet/movement` |
| **Response** | HTML page |

### Fleet Event Box (AJAX)
| Property | Value |
|----------|-------|
| **URL** | `GET /ajax/fleet/eventbox/fetch` |
| **Response** | JSON |

### Fleet Event List (AJAX)
| Property | Value |
|----------|-------|
| **URL** | `GET /ajax/fleet/eventlist/fetch` |
| **Response** | JSON |

---

## 12. Overview Page (planet resources, buildings, etc.)

| Property | Value |
|----------|-------|
| **URL** | `GET /overview` |
| **Response** | HTML page |

### Resources AJAX
| Property | Value |
|----------|-------|
| **URL** | `GET /ajax/resources` |
| **Response** | JSON (current resource levels, production, etc.) |

### Facilities AJAX
| Property | Value |
|----------|-------|
| **URL** | `GET /ajax/facilities` |
| **Response** | JSON |

### Research AJAX
| Property | Value |
|----------|-------|
| **URL** | `GET /ajax/research` |
| **Response** | JSON |

---

## Planet Switching

The "current planet" is tracked via the session. To switch planets:
- OGameX uses a `cp` query parameter on page loads (e.g., `GET /overview?cp={planetId}`)
- This is typically set by clicking a planet in the top nav
- AJAX endpoints operate on whatever planet is currently set in the session

---

## Ship IDs Reference

| ID | Ship | Machine Name |
|----|------|--------------|
| 202 | Small Cargo | `small_cargo` |
| 203 | Large Cargo | `large_cargo` |
| 204 | Light Fighter | `light_fighter` |
| 205 | Heavy Fighter | `heavy_fighter` |
| 206 | Cruiser | `cruiser` |
| 207 | Battleship | `battle_ship` |
| 208 | Colony Ship | `colony_ship` |
| 209 | Recycler | `recycler` |
| 210 | Espionage Probe | `espionage_probe` |
| 211 | Bomber | `bomber` |
| 213 | Destroyer | `destroyer` |
| 214 | Deathstar | `deathstar` |
| 215 | Battlecruiser | `battlecruiser` |
| 218 | Reaper | `reaper` |
| 219 | Pathfinder | `pathfinder` |

## Planet Types
| Value | Type |
|-------|------|
| 1 | Planet |
| 2 | Debris Field |
| 3 | Moon |

## Speed Values
- Range: 1.0 to 10.0 (representing 10% to 100%)
- Default increment: 1.0 (= 10% steps)
- General class: 0.5 increment (= 5% steps, includes 0.5 = 5%)
- `speed=10` means 100% speed
