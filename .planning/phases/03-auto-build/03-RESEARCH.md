# Phase 3: Auto-Build - Research

**Researched:** 2026-04-26
**Domain:** OGame building upgrade automation with ROI-based decision making
**Confidence:** HIGH

## Summary

Phase 3 adds automated building upgrades driven by ROI (Return on Investment) calculations. The bot iterates over all planets, computes which building upgrade yields the highest production gain relative to its cost, and queues the best candidate when a build slot is free. The system must respect per-building max-level caps and handle energy balance constraints.

The core mathematical challenge is computing accurate production increases and build costs using verified OGame formulas, then ranking candidates by ROI. The architectural challenge is fitting the auto-build worker into the existing poll-loop + StateReader pattern established by the defender worker in Phase 2, while adding two new ogamed client methods (BuildBuilding, GetConstructions) that don't yet exist in the codebase.

**Primary recommendation:** Build a pure `internal/builder/roi.go` calculator package (no dependencies on ogamed or state), then wire it into a `internal/builder/builder.go` worker that follows the defender's poll-loop pattern. Extend the ogamed client with BuildBuilding + GetConstructions, and add AutoBuildConfig with max-level caps to the config system.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GROW-01 | Bot calculates ROI (production increase / build cost) for every upgradeable building across all planets | OGame cost and production formulas verified from ogamed source (`pkg/ogame/metalMine.go`, `crystalMine.go`, `deuteriumSynthesizer.go`, `solarPlant.go`, `fusionReactor.go`, `baseLevelable.go`) — see Code Examples section |
| GROW-02 | Bot automatically queues the most profitable building upgrade based on ROI calculation | Worker pattern from defender (poll loop + StateReader DI); ogamed endpoint `POST /bot/planets/:planetID/build/building/:ogameID` [VERIFIED: ogamed wiki] |
| GROW-03 | Bot respects configurable max-level caps per building type per planet | Config system already supports nested YAML structs; AutoBuildConfig extension follows DefenderConfig pattern [VERIFIED: existing `config.go`] |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| ROI calculation (cost + production formulas) | Bot Engine (pure Go) | — | Pure math, no I/O. Must be unit-testable without mocks. |
| Build slot availability check | Bot Engine | ogamed client | State cache for fast check, ogamed for real-time confirmation |
| Build execution (API call) | ogamed client | — | Only ogamed talks to OGame. Client extends with BuildBuilding method. |
| Max-level cap enforcement | Bot Engine (config) | — | Config loaded at startup, enforced in ROI filter. |
| Energy balance validation | Bot Engine (pure Go) | — | Must verify energy surplus before upgrading energy-consuming buildings. |
| Build decision logging | Bot Engine | — | slog structured logging, same as defender pattern. |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (math) | — | Power/exponent/ceiling for OGame formulas | No external dependency needed for cost/production math |
| gopkg.in/yaml.v3 | already in go.mod | Config with max-level caps | Already used by config.go |
| modernc.org/sqlite | already in go.mod | Optional: build history tracking | Consistent with existing persistence layer |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib (math/rand) | — | Jitter for anti-detection poll intervals | All worker poll loops |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom ROI calculator | Pre-computed lookup table | Lookup table is faster but less flexible — custom calculator handles any level and universe speed correctly. Calculator is the right choice. |
| SQLite build history | In-memory only | In-memory loses history on restart; SQLite lets us track "what did we build and when" for the future dashboard. Use SQLite. |

**No new dependencies required.** All needed packages are already in go.mod.

## Architecture Patterns

### System Architecture Diagram

```
                              ┌──────────────────────────────────────────┐
                              │           cmd/bot/main.go                │
                              │  wires: config → state → builder → run  │
                              └───────────────┬──────────────────────────┘
                                              │
                  ┌───────────────────────────┼───────────────────────────┐
                  ▼                           ▼                           ▼
    ┌──────────────────────┐   ┌──────────────────────┐   ┌──────────────────────┐
    │  State Manager       │   │  Builder Worker       │   │  ogamed Client       │
    │  (read-only access)  │   │                      │   │  (extended)          │
    │                      │   │  poll loop:          │   │                      │
    │  GetPlanets()        │◄──┤  1. Get all planets  │──►│  GetConstructions()  │
    │  GetBuildings()      │   │  2. Get buildings    │   │  BuildBuilding()     │
    │  GetResources()      │   │  3. Get resources    │   │  GetResourceBuildings│
    │  GetResearch()       │   │  4. Get research     │   │                      │
    │                      │   │  5. Check slots      │   │  rate-limited, retry │
    │  SQLite cache        │   │  6. ROI calc         │   └──────────────────────┘
    │  (60s refresh)       │   │  7. Build best       │           │
    └──────────────────────┘   │                      │           ▼
                               │  ROI Calculator:     │    ┌──────────────┐
                               │  - cost formulas     │    │  ogamed      │
                               │  - production formulas│   │  REST daemon │
                               │  - energy balance    │    │  (Docker)    │
                               │  - cap enforcement   │    └──────┬───────┘
                               └──────────────────────┘           │
                                                                   ▼
                                                          ┌──────────────┐
                                                          │  OGame       │
                                                          │  Servers     │
                                                          └──────────────┘
```

### Recommended Project Structure
```
internal/
├── builder/           # Auto-build feature (NEW)
│   ├── roi.go         # Pure ROI calculation (cost, production, energy)
│   ├── roi_test.go    # Unit tests for all formulas
│   ├── builder.go     # Builder worker (poll loop, decision logic)
│   └── builder_test.go# Integration tests with mock client
├── ogamed/
│   └── client.go      # Extended: BuildBuilding, GetConstructions
├── config/
│   └── config.go      # Extended: AutoBuildConfig with caps
├── state/
│   └── manager.go     # Extended: GetBuildings, GetFacilities read methods
└── defender/          # Existing (pattern reference only)
```

### Pattern 1: Pure ROI Calculator Package

**What:** `internal/builder/roi.go` contains all OGame formulas as pure functions with no external dependencies (no ogamed, no state, no I/O). Takes building levels, research levels, universe speed, temperature as inputs. Returns cost, production, ROI.

**When to use:** Every calculation in the auto-build system. Must be unit-testable without mocks.

**Example:**
```go
// BuildingCost returns the resources needed to upgrade to the given level.
// Formula verified from ogamed source: baseCost * increaseFactor^(level-1)
// Source: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/baseLevelable.go
func BuildingCost(baseCost Resources, factor float64, level int) Resources {
    return Resources{
        Metal:   int(float64(baseCost.Metal) * math.Pow(factor, float64(level-1))),
        Crystal: int(float64(baseCost.Crystal) * math.Pow(factor, float64(level-1))),
    }
}
```

### Pattern 2: Builder Worker (Follows Defender Pattern)

**What:** The builder worker follows the exact same poll-loop + StateReader DI pattern as the defender. The only differences are: (1) it reads buildings/facilities instead of fleets, (2) it calls BuildBuilding instead of SendFleet, (3) it doesn't need a tracking table (OGame tracks construction state server-side via GetConstructions).

**When to use:** The auto-build worker loop.

**Example:**
```go
// Builder orchestrates auto-building across all planets.
type Builder struct {
    client   ogamed.ClientInterface
    stateMgr StateReader
    cfg      config.AutoBuildConfig
    log      *slog.Logger
}

// Run starts the builder poll loop. Blocks until context is cancelled.
func (b *Builder) Run(ctx context.Context) {
    interval := time.Duration(b.cfg.PollIntervalMs) * time.Millisecond
    for {
        jitter := time.Duration(rand.Intn(int(interval.Milliseconds()/2)+1)) * time.Millisecond
        select {
        case <-ctx.Done():
            return
        case <-time.After(interval + jitter):
            b.poll(ctx)
        }
    }
}
```

### Anti-Patterns to Avoid
- **Computing ROI every poll tick:** Cache ROI results and only recompute when state changes (building completes, resources change significantly). Recomputing for 8 planets × 9 buildings every 2 minutes is wasteful. [VERIFIED: PITFALLS.md "Performance Traps"]
- **Ignoring energy balance:** Upgrading a mine without checking energy surplus means the mine may produce at reduced rate (0% if energy is negative). Always verify energy headroom before upgrading energy-consuming buildings.
- **Not checking construction slot:** OGame allows one building under construction per planet. Must call GetConstructions to verify the slot is free before attempting to build.
- **Building on full planets:** If `FieldsUsed >= FieldsTotal`, no more buildings can be placed. Must skip these planets.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Building cost formula | Custom exponentiation logic | Verified formulas from ogamed source (`baseCost * factor^(level-1)`) | ogamed is the reference implementation; formulas match OGame exactly |
| Production rate formula | Guess at coefficients | Verified formulas from ogamed source (`metalMine.go`, etc.) | Crystal and deuterium have different plasma bonuses (0.66%, 0.33% vs 1% for metal); getting these wrong silently corrupts ROI |
| Rate limiting | Per-worker throttling | Existing `RateLimiter` in `internal/ogamed/` | Global rate limiter already shared across all workers; building a second one would allow exceeding OGame limits |
| State access | Direct ogamed calls in worker | `StateReader` interface reading from cached SQLite | State manager already refreshes buildings/resources; reading from cache avoids duplicate API calls |

**Key insight:** The OGame production formulas are deceptively complex. Each resource type has a different plasma technology bonus multiplier (metal: 1%, crystal: 0.66%, deuterium: 0.33%), and deuterium production is temperature-dependent. Copy the exact formulas from the verified ogamed source.

## Common Pitfalls

### Pitfall 1: Energy Deficit Reduces Production to Zero

**What goes wrong:** The bot upgrades Metal Mine level 20 → 21 without enough energy surplus. The solar plant can't support the increased energy consumption. Production drops to near-zero because the production ratio goes negative, making the ROI calculation completely wrong — the upgrade was actually harmful.

**Why it happens:** ROI calculation only considers the raw production increase without factoring in energy balance. Metal mine at level 21 consumes `ceil(10 * 21 * 1.1^21) = 793` energy. If the planet only has 500 energy surplus, the mine runs at reduced capacity.

**How to avoid:** Before recommending a mine upgrade, verify that current energy surplus ≥ the mine's additional energy consumption at the new level. If not, recommend Solar Plant or Fusion Reactor upgrade first. Include energy-producing buildings in the ROI pool.

**Warning signs:** ROI calculations show high returns but actual production after upgrade decreases.

### Pitfall 2: Building on Planets with No Free Fields

**What goes wrong:** The bot tries to build on a planet where all fields are used. The BuildBuilding API call fails, the bot retries on the next tick, fails again — infinite loop of failed builds.

**Why it happens:** Not checking `Planet.FieldsUsed < Planet.FieldsTotal` before attempting a build. Some planets (especially homeworlds in old universes) can be completely full.

**How to avoid:** Always check `FieldsUsed < FieldsTotal` before including a planet's buildings in the ROI candidate pool. Log a warning when a planet is full so the user knows they may need Terraformer.

**Warning signs:** Repeated "failed to build" errors in logs for the same planet.

### Pitfall 3: Ignoring Construction Slot Per-Planet

**What goes wrong:** The bot calls BuildBuilding on a planet that already has a building under construction. The API call fails.

**Why it happens:** OGame allows exactly one building construction per planet at a time. The bot must check GetConstructions to see if the building slot is free before attempting to build.

**How to avoid:** For each planet, call GetConstructions (or read from state cache) and skip planets where `Building.ID != 0`.

**Warning signs:** API errors when attempting to build on planets with active construction.

### Pitfall 4: ROI Calculation Doesn't Account for Universe Speed

**What goes wrong:** The ROI formula uses a fixed production rate that doesn't account for the universe economy speed multiplier. In a 7x speed universe, mines produce 7x more, making ROI calculations wildly wrong if speed isn't included.

**Why it happens:** The bot developer hardcodes speed=1 in the production formula, or forgets to call GetServerSpeed.

**How to avoid:** Always multiply production by universe speed (from `GetServerSpeed`). Cache the speed value — it never changes for a given universe.

**Warning signs:** ROI calculations seem reasonable in testing (1x universe) but wrong in production (7x universe). [VERIFIED: PITFALLS.md "Looks Done But Isn't" checklist]

### Pitfall 5: Bot Always Picks the Mathematically Optimal Building (Anti-Detection)

**What goes wrong:** The bot always builds the highest-ROI building with zero variation. This is a behavioral pattern that screams "bot" — no human plays with perfect mathematical optimization.

**Why it happens:** Developers focus on the algorithm and forget about behavioral anti-detection.

**How to avoid:** Occasionally (5-10% of the time, configurable) pick the second-best ROI building instead. Add random delay between build decisions. [VERIFIED: PITFALLS.md Pitfall 7]

**Warning signs:** Build logs show perfectly ordered ROI decisions with no deviation.

### Pitfall 6: Not Considering Build Time in ROI

**What goes wrong:** Two buildings have similar ROI, but one takes 2 hours and the other takes 12 hours. The bot picks the 12-hour building because its raw ROI is marginally higher. During those 12 hours, the construction slot is blocked, preventing 6 shorter builds that would have yielded more total production.

**Why it happens:** ROI is calculated as `production_increase / cost`, ignoring the time value — a building that pays for itself in 2 hours is better than one that pays for itself in 12 hours if both have similar absolute ROI.

**How to avoid:** Use time-adjusted ROI: `hourly_production_value / build_cost`, where `hourly_production_value` is the per-hour production increase valued in a common currency (e.g., metal-equivalent). This naturally favors faster payback.

## Code Examples

Verified patterns from ogamed source code:

### Building Cost Formula

```go
// GetPrice returns the cost to build the given level.
// Formula: baseCost * increaseFactor^(level-1)
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/baseLevelable.go]
func GetPrice(baseCostMetal, baseCostCrystal int, factor float64, level int) (metal, crystal int) {
    metal = int(float64(baseCostMetal) * math.Pow(factor, float64(level-1)))
    crystal = int(float64(baseCostCrystal) * math.Pow(factor, float64(level-1)))
    return
}

// Building definitions with base costs and increase factors:
// Source: [VERIFIED: ogamed source pkg/ogame/metalMine.go, crystalMine.go, etc.]
var BuildingDefs = map[int]BuildingDef{
    // ID: {Name, BaseMetal, BaseCrystal, BaseDeut, Factor, EnergyConsumer}
    1:  {"Metal Mine", 60, 15, 0, 1.5, true},           // MetalMine
    2:  {"Crystal Mine", 48, 24, 0, 1.6, true},          // CrystalMine
    3:  {"Deuterium Synthesizer", 225, 75, 0, 1.5, true}, // DeuteriumSynthesizer
    4:  {"Solar Plant", 75, 30, 0, 1.5, false},          // SolarPlant (produces energy)
    12: {"Fusion Reactor", 900, 360, 180, 1.8, false},   // FusionReactor (produces energy, consumes deuterium)
    // Storage buildings: not ROI-relevant for production, but may be needed when production exceeds capacity
    22: {"Metal Storage", 1000, 0, 0, 2.0, false},
    23: {"Crystal Storage", 1000, 500, 0, 2.0, false},
    24: {"Deuterium Tank", 1000, 1000, 0, 2.0, false},
}
```

### Metal Production Formula

```go
// MetalProduction returns hourly metal production at the given level.
// Formula: 30 * (1 + plasmaTech/100) * speed * level * 1.1^level + 30*speed (basic income)
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/metalMine.go]
func MetalProduction(level, plasmaTech, universeSpeed int) int {
    if level == 0 {
        return 30 * universeSpeed // basic income only
    }
    basicIncome := 30.0 * float64(universeSpeed)
    levelProduction := 30.0 * (1.0 + float64(plasmaTech)/100.0) * float64(universeSpeed) *
        float64(level) * math.Pow(1.1, float64(level))
    return int(levelProduction + basicIncome)
}
```

### Crystal Production Formula

```go
// CrystalProduction returns hourly crystal production at the given level.
// Note: plasma bonus is 0.66% per level (not 1% like metal).
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/crystalMine.go]
func CrystalProduction(level, plasmaTech, universeSpeed int) int {
    if level == 0 {
        return 15 * universeSpeed // basic income only
    }
    basicIncome := 15.0 * float64(universeSpeed)
    levelProduction := 20.0 * float64(universeSpeed) *
        (1.0 + float64(plasmaTech)*0.0066) *
        float64(level) * math.Pow(1.1, float64(level))
    return int(levelProduction + basicIncome)
}
```

### Deuterium Production Formula

```go
// DeuteriumProduction returns hourly deuterium production at the given level.
// Note: depends on average planet temperature. Plasma bonus is 0.33% per level.
// Formula: 10 * (1 + plasmaTech*0.0033) * level * 1.1^level * (-0.004*avgTemp + 1.36) * speed
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/deuteriumSynthesizer.go]
func DeuteriumProduction(level, plasmaTech, avgTemperature, universeSpeed int) int {
    if level == 0 {
        return 0 // no basic deuterium income
    }
    production := 10.0 * (1.0 + float64(plasmaTech)*0.0033) *
        float64(level) * math.Pow(1.1, float64(level)) *
        (-0.004*float64(avgTemperature) + 1.36) * float64(universeSpeed)
    return int(math.Round(production))
}
```

### Energy Consumption for Mines

```go
// EnergyConsumption returns energy consumed by a mine at the given level.
// Both metal and crystal mines use the same formula.
// Source: [VERIFIED: ogamed source metalMine.go, crystalMine.go]
func MineEnergyConsumption(level int) int {
    return int(math.Ceil(10.0 * float64(level) * math.Pow(1.1, float64(level))))
}

// Deuterium mine uses 20 instead of 10 as the base multiplier:
func DeutMineEnergyConsumption(level int) int {
    return int(math.Ceil(20.0 * float64(level) * math.Pow(1.1, float64(level))))
}
```

### Solar Plant Energy Production

```go
// SolarProduction returns energy produced by solar plant at given level.
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/solarPlant.go]
func SolarProduction(level int) int {
    return int(math.Floor(20.0 * float64(level) * math.Pow(1.1, float64(level))))
}
```

### Fusion Reactor Energy Production

```go
// FusionProduction returns energy produced by fusion reactor.
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/fusionReactor.go]
func FusionProduction(level, energyTech int) int {
    return int(math.Round(30.0 * float64(level) * math.Pow(1.05+float64(energyTech)*0.01, float64(level))))
}
```

### Building Construction Time

```go
// ConstructionTime returns how long a building takes to construct.
// Formula: (metalCost + crystalCost) / (2500 * (1 + roboticsFactory) * speed * 2^naniteFactory) hours
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/blob/master/pkg/ogame/baseBuilding.go]
func ConstructionTime(metalCost, crystalCost int, roboticsFactory, naniteFactory, universeSpeed int) time.Duration {
    hours := float64(metalCost+crystalCost) /
        (2500.0 * (1.0 + float64(roboticsFactory)) * float64(universeSpeed) * math.Pow(2, float64(naniteFactory)))
    seconds := hours * 3600
    if seconds < 1 {
        seconds = 1
    }
    return time.Duration(int(math.Floor(seconds))) * time.Second
}
```

### BuildBuilding API Call (to add to ogamed client)

```go
// BuildBuilding starts constructing the given building on the specified planet.
// ogamed endpoint: POST /bot/planets/:planetID/build/building/:ogameID
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation]
func (c *Client) BuildBuilding(ctx context.Context, planetID, buildingID int) error {
    path := fmt.Sprintf("/bot/planets/%d/build/building/%d", planetID, buildingID)
    _, err := postTyped[any](c, ctx, path, url.Values{})
    return err
}
```

### GetConstructions API Call (to add to ogamed client)

```go
// Construction represents an active construction on a planet.
type Construction struct {
    ID        int    `json:"ID"`
    Level     int    `json:"Level"`
    Countdown int64  `json:"Countdown"` // seconds remaining
}

// Constructions represents all active constructions on a planet.
type Constructions struct {
    Building Construction `json:"Building"`
    Research Construction `json:"Research"`
}

// GetConstructions returns the current construction status for a planet.
// ogamed endpoint: GET /bot/planets/:planetID/constructions
// Source: [VERIFIED: https://github.com/alaingilbert/ogame/wiki/ogamed-full-documentation]
func (c *Client) GetConstructions(ctx context.Context, planetID int) (Constructions, error) {
    path := fmt.Sprintf("/bot/planets/%d/constructions", planetID)
    return getTyped[Constructions](c, ctx, path)
}
```

### AutoBuildConfig Extension (to add to config.go)

```go
// AutoBuildConfig holds the auto-build feature settings including per-building caps.
type AutoBuildConfig struct {
    FeatureConfig `yaml:",inline"`
    MaxLevels     map[string]int           `yaml:"maxLevels"`     // global defaults: {"MetalMine": 30, "CrystalMine": 28, ...}
    PlanetOverrides map[string]map[string]int `yaml:"planetOverrides"` // per-planet: {"Homeworld": {"MetalMine": 35}}
}
```

### ROI Calculation Core

```go
// ROIResult represents the ROI analysis for a single building upgrade candidate.
type ROIResult struct {
    PlanetID          int
    BuildingID        int
    BuildingName      string
    CurrentLevel      int
    TargetLevel       int
    CostMetal         int
    CostCrystal       int
    CostDeuterium     int
    ProductionIncrease float64 // hourly production increase (metal-equivalent value)
    ROIScore          float64  // productionIncrease / totalCostValue
    BuildTime         time.Duration
}

// CalculateROI computes ROI for upgrading buildingID on a specific planet.
// Pure function — no side effects, no external dependencies.
func CalculateROI(
    buildingID int,
    currentLevel int,
    planet model.Planet,
    buildings model.ResourceBuildings,
    facilities model.Facilities,
    research model.Research,
    resources model.Resources,
    universeSpeed int,
    maxLevel int,
) (ROIResult, bool) {
    // Check max level cap
    if currentLevel >= maxLevel {
        return ROIResult{}, false
    }

    targetLevel := currentLevel + 1
    def, ok := BuildingDefs[buildingID]
    if !ok {
        return ROIResult{}, false
    }

    // Calculate build cost
    costMetal, costCrystal, costDeut := GetPrice(def.BaseMetal, def.BaseCrystal, def.BaseDeut, def.Factor, targetLevel)

    // Check if we can afford it
    if resources.Metal < costMetal || resources.Crystal < costCrystal || resources.Deuterium < costDeut {
        return ROIResult{}, false
    }

    // Calculate production increase
    prodBefore := productionAtLevel(buildingID, currentLevel, research, planet, universeSpeed)
    prodAfter := productionAtLevel(buildingID, targetLevel, research, planet, universeSpeed)
    prodIncrease := prodAfter - prodBefore

    // Check energy balance for energy-consuming buildings
    if def.EnergyConsumer {
        energyConsumedBefore := energyConsumption(buildingID, currentLevel)
        energyConsumedAfter := energyConsumption(buildingID, targetLevel)
        additionalEnergyNeeded := energyConsumedAfter - energyConsumedBefore
        // Verify planet has enough energy surplus
        // (simplified — actual check needs current energy production vs consumption)
    }

    // Calculate ROI score (production increase / total cost valued in metal-equivalent)
    // Standard value ratios: metal=1, crystal=1.5, deuterium=2 (trade ratios)
    totalCostValue := float64(costMetal) + float64(costCrystal)*1.5 + float64(costDeut)*2.0
    roiScore := prodIncrease / totalCostValue

    // Calculate build time
    buildTime := ConstructionTime(costMetal, costCrystal, facilities.RoboticsFactory, facilities.NaniteFactory, universeSpeed)

    return ROIResult{
        PlanetID:          planet.ID,
        BuildingID:        buildingID,
        BuildingName:      def.Name,
        CurrentLevel:      currentLevel,
        TargetLevel:       targetLevel,
        CostMetal:         costMetal,
        CostCrystal:       costCrystal,
        CostDeuterium:     costDeut,
        ProductionIncrease: prodIncrease,
        ROIScore:          roiScore,
        BuildTime:         buildTime,
    }, true
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Fixed build order (metal→crystal→deut) | ROI-based dynamic ranking | Community standard since ~2010 | Must use ROI, not static priority lists |
| Metal-equivalent ROI only | Per-resource ROI with trade ratios | Standard across all modern bots | Crystal and deuterium have different value — use trade ratio weighting |
| Build immediately when affordable | Check construction slot + energy balance | OGame mechanic since always | Must check both constraints |
| Global max levels | Per-planet max level overrides | Best practice from TBot/Cruiser | Allows different strategies for different planets |

**Deprecated/outdated:**
- Static build priority lists: Always inferior to ROI-based approach. Some bots still use them, but they're suboptimal.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | OGame only allows one building construction per planet at a time | Common Pitfalls | If multiple slots allowed, the builder should queue multiple buildings per planet per tick |
| A2 | ogamed `GET /bot/planets/:planetID/constructions` returns JSON matching the Constructions struct from the ogame Go package | Code Examples | If the response format differs, the parser will need adjustment |
| A3 | Resource trade ratios for metal-equivalent valuation are metal=1, crystal=1.5, deuterium=2.0 | Code Examples | Different ratios would change ROI rankings; should be configurable |
| A4 | Storage buildings (Metal Storage, Crystal Storage, Deuterium Tank) are not included in ROI calculation because they don't increase production | Architecture | If storage buildings are needed to prevent resource waste from exceeding capacity, they need separate logic |
| A5 | The `Planet.FieldsUsed` from the API accurately reflects current building count | Common Pitfalls | If it doesn't update until construction completes, we may try to build on nearly-full planets |

## Open Questions

1. **Should storage buildings be included in ROI calculation?**
   - What we know: Storage buildings don't increase production. But if metal production exceeds storage capacity, excess is wasted.
   - What's unclear: Whether v1 needs to handle the "storage full → waste" scenario.
   - Recommendation: Start without storage buildings. Add them in v2 if needed (GROW-06 area).

2. **Should facility buildings (Robotics Factory, Nanite Factory) be included?**
   - What we know: They reduce build times, indirectly increasing ROI of future buildings.
   - What's unclear: How to value build-time reduction in ROI terms.
   - Recommendation: Start with resource buildings only (Metal Mine, Crystal Mine, Deuterium Synthesizer, Solar Plant, Fusion Reactor). Facilities and research are v2 territory (GROW-05).

3. **Should the builder track build history in SQLite?**
   - What we know: The dashboard (Phase 5) needs to show "recent bot actions". Currently no mechanism exists.
   - What's unclear: Whether Phase 3 should create the schema now or defer.
   - Recommendation: Create a simple `build_events` table now — the planner can decide scope.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies beyond existing ogamed Docker setup — all dependencies already in go.mod)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — see Wave 0 |
| Quick run command | `go test ./internal/builder/... -count=1 -timeout=30s` |
| Full suite command | `go test ./... -count=1 -timeout=60s` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GROW-01 | ROI calculation for all building types at various levels | unit | `go test ./internal/builder/... -run TestROI -count=1` | ❌ Wave 0 |
| GROW-01 | Cost formula matches expected values for known levels | unit | `go test ./internal/builder/... -run TestCost -count=1` | ❌ Wave 0 |
| GROW-01 | Production formula matches expected values | unit | `go test ./internal/builder/... -run TestProduction -count=1` | ❌ Wave 0 |
| GROW-02 | Builder picks highest ROI candidate | unit | `go test ./internal/builder/... -run TestPickBest -count=1` | ❌ Wave 0 |
| GROW-02 | Builder calls BuildBuilding when slot is free | integration | `go test ./internal/builder/... -run TestBuild -count=1` | ❌ Wave 0 |
| GROW-02 | Builder skips when all construction slots occupied | integration | `go test ./internal/builder/... -run TestSkipBusy -count=1` | ❌ Wave 0 |
| GROW-03 | Builder respects max-level caps | unit | `go test ./internal/builder/... -run TestMaxLevel -count=1` | ❌ Wave 0 |
| GROW-03 | Per-planet max-level overrides work | unit | `go test ./internal/builder/... -run TestPlanetOverride -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/builder/... -count=1 -timeout=30s`
- **Per wave merge:** `go test ./... -count=1 -timeout=60s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/builder/roi.go` — ROI calculator with all formulas
- [ ] `internal/builder/roi_test.go` — unit tests for formulas and ROI logic
- [ ] `internal/builder/builder.go` — builder worker
- [ ] `internal/builder/builder_test.go` — integration tests with mock client

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | ogamed handles auth |
| V3 Session Management | no | ogamed handles sessions |
| V4 Access Control | no | single-user bot |
| V5 Input Validation | yes | Go type system + config validation for max-level caps |
| V6 Cryptography | no | no crypto in this phase |

### Known Threat Patterns for Auto-Build

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Config injection via YAML | Tampering | Validate max-level ranges (1-100), reject invalid building names |
| BuildBuilding API abuse (too many builds) | Denial of Service | Rate limiter already enforced at client level |
| Resource miscalculation leading to API errors | Tampering | Pre-validate affordability before BuildBuilding call |

## Sources

### Primary (HIGH confidence)
- ogamed source code (`pkg/ogame/metalMine.go`, `crystalMine.go`, `deuteriumSynthesizer.go`, `solarPlant.go`, `fusionReactor.go`, `baseLevelable.go`, `baseBuilding.go`) — all OGame formulas verified from the reference Go implementation
- ogamed wiki REST API documentation — BuildBuilding, GetConstructions endpoints confirmed
- Existing codebase: `internal/ogamed/client.go`, `internal/state/manager.go`, `internal/config/config.go`, `internal/defender/defender.go`, `internal/defender/escape.go` — patterns verified by reading source

### Secondary (MEDIUM confidence)
- OGame community knowledge on trade ratios (metal=1, crystal=1.5, deuterium=2) — standard across community tools but not officially documented [ASSUMED: A3]
- OGame single construction slot per planet — widely documented mechanic [ASSUMED: A1]

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all formulas verified from ogamed source
- Architecture: HIGH — follows established defender pattern exactly; minimal new patterns
- Pitfalls: HIGH — formulas verified, edge cases identified from community knowledge and PITFALLS.md
- OGame formulas: HIGH — directly verified from ogamed source code (alaingilbert/ogame)

**Research date:** 2026-04-26
**Valid until:** 2026-05-26 (stable — OGame formulas don't change between game versions; only new buildings might be added with lifeforms which are out of scope)
