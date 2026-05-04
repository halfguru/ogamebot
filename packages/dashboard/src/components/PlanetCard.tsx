import type { APIPlanet, APIBuildEvent, PlanetBuildPlan, APIBuildings, APIFacilities } from '@ogame-bot/shared';
import { formatNumber } from '@ogame-bot/shared';
import { Show, createSignal, onCleanup, onMount } from 'solid-js';

function ResourceBar(props: { label: string; value: number; type: string; production?: number }) {
  const maxStorage = () => {
    if (props.value <= 0) return 100;
    const mag = Math.pow(10, Math.floor(Math.log10(props.value)));
    return Math.max(mag * 1.5, 100);
  };
  const percent = () => Math.min((props.value / maxStorage()) * 100, 100);

  return (
    <div class="resource-bar">
      <span class="resource-bar-label">{props.label}</span>
      <div class="resource-bar-track">
        <div
          class={`resource-bar-fill ${props.type}`}
          style={{ width: `${percent()}%` }}
        />
        <span class="resource-bar-value">{formatNumber(props.value)}</span>
      </div>
      <Show when={props.production && props.production > 0}>
        <span class="resource-production">+{formatNumber(Math.round(props.production! * 3600))}/hr</span>
      </Show>
    </div>
  );
}

function buildingCurrentLevel(buildings: APIBuildings, facilities: APIFacilities, buildingId: number): number {
  switch (buildingId) {
    case 1: return buildings.metalMine;
    case 2: return buildings.crystalMine;
    case 3: return buildings.deuteriumSynthesizer;
    case 4: return buildings.solarPlant;
    case 12: return buildings.fusionReactor;
    case 14: return facilities.roboticsFactory;
    case 15: return facilities.naniteFactory;
    case 21: return facilities.shipyard;
    case 22: return buildings.metalStorage;
    case 23: return buildings.crystalStorage;
    case 24: return buildings.deuteriumStorage;
    case 31: return facilities.researchLab;
    default: return 0;
  }
}

function BuildCountdown(props: { event: APIBuildEvent; currentLevel: number }) {
  const [tick, setTick] = createSignal(0);
  let interval: ReturnType<typeof setInterval> | undefined;
  onMount(() => {
    interval = setInterval(() => setTick((t) => t + 1), 1000);
  });
  onCleanup(() => {
    if (interval) clearInterval(interval);
  });
  const completed = () => {
    tick();
    return props.currentLevel >= props.event.toLevel;
  };
  const progress = () => {
    tick();
    if (completed()) return 100;
    const created = new Date(props.event.createdAt).getTime();
    const elapsed = Date.now() - created;
    const buildDurationMs = props.event.buildTimeSeconds * 1000;
    if (buildDurationMs <= 0) return 100;
    return Math.min(100, Math.max(0, (elapsed / buildDurationMs) * 100));
  };
  const text = () => {
    tick();
    if (completed()) {
      return `Completed: ${props.event.buildingName} ${props.event.toLevel}`;
    }
    const created = new Date(props.event.createdAt).getTime();
    const elapsed = Date.now() - created;
    const buildDurationMs = props.event.buildTimeSeconds * 1000;
    if (elapsed >= 0) {
      const remaining = Math.max(0, buildDurationMs - elapsed);
      const m = Math.floor(remaining / 60000);
      const s = Math.floor((remaining % 60000) / 1000);
      if (remaining > 0) {
        return `Building ${props.event.buildingName} ${props.event.toLevel} (${m}m ${s}s left)`;
      }
    }
    return `Completed: ${props.event.buildingName} ${props.event.toLevel}`;
  };
  return (
    <div class="build-queue">
      <Show when={!completed()} fallback={
        <div class="build-queue-header">
          <span class="build-complete-icon">✓</span>
          <span class="build-queue-text">{text()}</span>
        </div>
      }>
        <div class="build-progress-track">
          <div class="build-progress-fill" style={{ width: `${progress()}%` }} />
          <span class="build-progress-text">{text()}</span>
        </div>
      </Show>
    </div>
  );
}

export default function PlanetCard(props: { planet: APIPlanet; buildEvent?: APIBuildEvent; buildPlan?: PlanetBuildPlan }) {
  const fieldsPercent = () => (props.planet.fieldsUsed / props.planet.fieldsTotal) * 100;
  const fieldsClass = () =>
    fieldsPercent() >= 90 ? 'fields-critical' : fieldsPercent() >= 70 ? 'fields-warning' : 'fields-ok';

  const fac = () => props.planet.facilities;

  const imageTypeClass = () => props.planet.imageType ? `planet-type-${props.planet.imageType}` : '';

  return (
    <div class={`planet-card ${imageTypeClass()}`}>
      <div class="planet-card-inner">
        <div class="planet-header">
          <h3>{props.planet.name}</h3>
          <span class="coords">
            [{props.planet.galaxy}:{props.planet.system}:{props.planet.position}]
          </span>
          <Show when={props.planet.isMoon}>
            <span class="badge moon">Moon</span>
          </Show>
        </div>
        <div class={`planet-fields ${fieldsClass()}`}>
          <div class="fields-bar-track">
            <div class={`fields-bar-fill ${fieldsClass()}`} style={{ width: `${Math.min(fieldsPercent(), 100)}%` }} />
            <span class="fields-bar-text">{props.planet.fieldsUsed}/{props.planet.fieldsTotal}</span>
          </div>
        </div>
        <div class="planet-resources">
          <ResourceBar label="Metal" value={props.planet.resources.metal} type="metal" production={props.planet.resources.metalProduction} />
          <ResourceBar label="Crystal" value={props.planet.resources.crystal} type="crystal" production={props.planet.resources.crystalProduction} />
          <ResourceBar label="Deut" value={props.planet.resources.deuterium} type="deuterium" production={props.planet.resources.deuteriumProduction} />
          <ResourceBar label="Energy" value={props.planet.resources.energy} type="energy" />
        </div>
        <div class="planet-section-label">Buildings</div>
        <div class="planet-buildings">
          <Show when={props.planet.buildings.metalMine > 0}>
            <span class="pill">Metal {props.planet.buildings.metalMine}</span>
          </Show>
          <Show when={props.planet.buildings.crystalMine > 0}>
            <span class="pill">Crystal {props.planet.buildings.crystalMine}</span>
          </Show>
          <Show when={props.planet.buildings.deuteriumSynthesizer > 0}>
            <span class="pill">Deut {props.planet.buildings.deuteriumSynthesizer}</span>
          </Show>
          <Show when={props.planet.buildings.solarPlant > 0}>
            <span class="pill">Solar {props.planet.buildings.solarPlant}</span>
          </Show>
          <Show when={props.planet.buildings.fusionReactor > 0}>
            <span class="pill">Fusion {props.planet.buildings.fusionReactor}</span>
          </Show>
        </div>
        <Show when={
          fac().roboticsFactory > 0 || fac().shipyard > 0 || fac().researchLab > 0 || fac().naniteFactory > 0
        }>
          <div class="planet-section-label">Facilities</div>
          <div class="planet-facilities">
            <Show when={fac().roboticsFactory > 0}>
              <span class="pill facility">Robot {fac().roboticsFactory}</span>
            </Show>
            <Show when={fac().shipyard > 0}>
              <span class="pill facility">Shipyard {fac().shipyard}</span>
            </Show>
            <Show when={fac().researchLab > 0}>
              <span class="pill facility">Lab {fac().researchLab}</span>
            </Show>
            <Show when={fac().naniteFactory > 0}>
              <span class="pill facility">Nanite {fac().naniteFactory}</span>
            </Show>
          </div>
        </Show>
        <Show when={props.buildPlan}>
          <div class="planet-section-label">Next Build</div>
          <div class="planet-next-build">
            <span class={`pill plan tier-${props.buildPlan!.tier} ${props.buildPlan!.affordable ? '' : 'pending'}`}>
              {props.buildPlan!.buildingName} {props.buildPlan!.currentLevel} → {props.buildPlan!.targetLevel}
            </span>
            <Show when={props.buildPlan!.costMetal > 0 || props.buildPlan!.costCrystal > 0 || props.buildPlan!.costDeuterium > 0}>
              <span class="plan-cost">
                {formatNumber(props.buildPlan!.costMetal)}M {formatNumber(props.buildPlan!.costCrystal)}C {formatNumber(props.buildPlan!.costDeuterium)}D
              </span>
            </Show>
            <Show when={!props.buildPlan!.affordable}>
              <span class="plan-status pending">Waiting for resources</span>
            </Show>
          </div>
        </Show>
        <Show when={props.buildEvent}>
          <BuildCountdown event={props.buildEvent!} currentLevel={buildingCurrentLevel(props.planet.buildings, props.planet.facilities, props.buildEvent!.buildingId)} />
        </Show>
      </div>
    </div>
  );
}
