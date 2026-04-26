# Research: Phase 4 — Auto-Farm

**Researched:** 2026-04-26
**Requirements:** COMB-01, COMB-02, COMB-03

## ogamed API Endpoints for Auto-Farm

### Galaxy Scanning (COMB-01)

**Endpoint:** `GET /bot/galaxy-infos/:galaxy/:system`
**Returns:** `SystemInfos` — array of planet/player data for one solar system.

Key fields in response (per ogamed source, PascalCase JSON tags):
- Each position has: `PlayerID`, `PlayerName`, `Inactive` (bool), `LongInactive` (bool), `Vacation` (bool), `Banned` (bool), `Name` (planet name), `Coordinate` (Galaxy/System/Position)
- Also: `Rank` (player rank), `Alliance` info, `Moon` presence

**Filtering criteria for inactives:**
- `Inactive == true` (7+ days offline)
- `Vacation == false` (vacation mode = can't attack)
- `Banned == false`
- `Name != ""` (position has a planet)

### Espionage (COMB-02)

**Send probes:** `POST /bot/planets/:planetID/send-fleet` with mission=6 (MissionEspionage)
- Ships: `210,count` (EspionageProbe ID=210)
- Speed: 10 (fastest — probes are fast)
- Destination: target coordinates
- No resources sent (Metal=0, Crystal=0, Deuterium=0)

**Read reports:** `GET /bot/get-espionage-report-messages` → returns `[]EspionageReportSummary`
- Each has: `ID` (message ID), `Coordinate`, `Date`

**Detailed report:** `GET /bot/get-espionage-report/:messageID` → returns full `EspionageReport`
- Key fields for auto-farm: `Metal`, `Crystal`, `Deuterium` (resources on planet)
- Defense: `RocketLauncher`, `LightLaser`, `HeavyLaser`, `GaussCannon`, `IonCannon`, `PlasmaTurret`, `SmallShieldDome`, `LargeShieldDome`
- `HasDefensesInformation` bool — depends on probe count vs espionage tech level
- `HasFleetInformation` bool — same dependency
- `IsInactive`, `IsLongInactive` bool
- `Loot(characterClass)` method (but we calculate our own)

**Cleanup:** `POST /bot/delete-all-espionage-reports` — delete old reports after processing

### Attack (COMB-03)

**Send attack:** `POST /bot/planets/:planetID/send-fleet` with mission=1 (MissionAttack)
- Ships: enough SmallCargo or LargeCargo to carry loot
- Speed: configurable (10 = fastest arrival, 1 = slowest = less fuel)
- Loot cap: 50% of resources on planet (75% for inactive if Discoverer class)

## OGame Loot Mechanics

- Plunder ratio: 50% of total resources per attack
- Multiple raids: can send multiple attacks, each gets 50% of remaining
- Cargo capacity needed: metal + crystal + deuterium loot must fit in ship cargo
- SmallCargo: 5,000 (25,000 with impulse 5) cargo
- LargeCargo: 25,000 cargo
- No combat losses if target has zero defense and zero fleet

## Farm Decision Logic

### Profit Threshold
```
estimatedLoot = (metal + crystal + deuterium) * plunderRatio
lootValue = metal + crystal*1.5 + deuterium*2.0  // metal-equivalent
fuelCost = fuelConsumption(distance, speed, ships, research)
netProfit = lootValue - fuelCost

Attack if: netProfit >= configurableMinProfit
```

### Target Scoring
- Higher resources = better target
- Closer system = less fuel = higher net profit
- No defense = guaranteed full loot
- Long inactive > regular inactive (less likely to come back)

### Attack Fleet Composition
- Use SmallCargo (or LargeCargo) only — cheap, fast, maximum cargo
- Number of ships = ceil(estimatedLoot / cargoCapacity)
- Add small buffer (10%) for loot variation
- Send from closest planet to minimize fuel

## Safety Considerations

- Don't attack if fleet slots are nearly full (defender needs slots)
- Don't attack players with rank > our rank (honor loss, possible retaliation)
- Randomize timing between attacks (anti-detection)
- Skip targets that returned failed attacks recently
- Clean up espionage reports after processing (don't accumulate messages)

## New Client Methods Needed

```
GetGalaxyInfos(ctx, galaxy, system) → SystemInfos
GetEspionageReportMessages(ctx) → []EspionageReportSummary
GetEspionageReport(ctx, messageID) → EspionageReport
DeleteAllEspionageReports(ctx) → error
```

## New Domain Types Needed

```go
// SystemInfos — response from galaxy scan
type SystemInfos struct {
    Galaxy   int
    System   int
    Planets  []PlanetPosition
}

type PlanetPosition struct {
    Position     int
    Name         string
    PlayerID     int64
    PlayerName   string
    Inactive     bool
    LongInactive bool
    Vacation     bool
    Banned       bool
    Rank         int
    Coordinate   Coordinate
    Moon         bool
}

// EspionageReportSummary — from message list
type EspionageReportSummary struct {
    ID         int64
    Coordinate Coordinate
    Date       time.Time
}

// EspionageReport — full report
type EspionageReport struct {
    ID                       int64
    Metal                    int64
    Crystal                  int64
    Deuterium                int64
    HasDefensesInformation   bool
    HasFleetInformation      bool
    RocketLauncher           int
    LightLaser               int
    HeavyLaser               int
    GaussCannon              int
    IonCannon                int
    PlasmaTurret             int
    SmallShieldDome          int
    LargeShieldDome          int
    IsInactive               bool
    Coordinate               Coordinate
}
```

## Config Extension

```yaml
features:
  autoFarm:
    enabled: true
    pollIntervalMs: 300000     # 5 min between farm cycles
    galaxyRanges:              # which galaxies/systems to scan
      - galaxy: 1
        systemStart: 1
        systemEnd: 50
    minProfitThreshold: 10000  # minimum metal-equivalent profit
    maxProbesPerTarget: 5      # number of probes to send
    maxAttacksPerCycle: 3      # limit attacks per cycle
    preferCloseTargets: true   # sort by distance
    skipDefended: true         # skip targets with defense
    attackShipType: "smallCargo"  # or "largeCargo"
```

## State Tracking (SQLite)

New tables needed:
- `farm_targets` — discovered inactive planets with last-scan time
- `farm_attacks` — attack history (coordinate, loot, sent_at, fleet_id)

## Architecture: Farmer Worker Pattern

Follow same pattern as Defender and Builder workers:
- `internal/farmer/farmer.go` — FarmerWorker with Run(ctx) loop
- `internal/farmer/scanner.go` — Galaxy scanning logic
- `internal/farmer/targets.go` — Target evaluation and scoring
- Uses ClientInterface + StateReader + DB
- Wired in cmd/bot/main.go like Defender/Builder

### Workflow Per Cycle:
1. Scan galaxy ranges → discover inactive positions
2. For discovered inactives, check if recently scanned
3. Send espionage probes to unscanned/recent inactives
4. Wait for reports to arrive (not immediate — need to poll messages)
5. Parse reports → evaluate loot vs defense
6. Score targets → rank by net profit
7. Dispatch attacks to top N targets (up to maxAttacksPerCycle)
8. Clean up old reports
