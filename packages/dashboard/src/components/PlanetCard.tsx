import type { APIPlanet, APIBuildEvent, PlanetBuildPlan } from '@ogame-bot/shared';
import { formatNumber } from '@ogame-bot/shared';
import { Show, createSignal, onCleanup, onMount } from 'solid-js';

function ResourceBar(props: { label: string; value: number; type: string }) {
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
    </div>
  );
}

function BuildCountdown(props: { event: APIBuildEvent }) {
  const [tick, setTick] = createSignal(0);
  let interval: ReturnType<typeof setInterval> | undefined;
  onMount(() => {
    interval = setInterval(() => setTick((t) => t + 1), 1000);
  });
  onCleanup(() => {
    if (interval) clearInterval(interval);
  });
  const text = () => {
    tick();
    const created = new Date(props.event.createdAt).getTime();
    const elapsed = Date.now() - created;
    const buildDurationMs = props.event.buildTimeSeconds * 1000;
    if (buildDurationMs > 0 && elapsed >= buildDurationMs) {
      return `Completed: ${props.event.buildingName} ${props.event.toLevel}`;
    }
    if (elapsed >= 0) {
      const remaining = Math.max(0, buildDurationMs - elapsed);
      const m = Math.floor(remaining / 60000);
      const s = Math.floor((remaining % 60000) / 1000);
      return `Building ${props.event.buildingName} ${props.event.toLevel} (${m}m ${s}s left)`;
    }
    return `Queued: ${props.event.buildingName} ${props.event.toLevel}`;
  };
  return <div class="build-queue">{text()}</div>;
}

export default function PlanetCard(props: { planet: APIPlanet; buildEvent?: APIBuildEvent; buildPlan?: PlanetBuildPlan }) {
  const fieldsPercent = () => (props.planet.fieldsUsed / props.planet.fieldsTotal) * 100;
  const fieldsClass = () =>
    fieldsPercent() >= 90 ? 'fields-critical' : fieldsPercent() >= 70 ? 'fields-warning' : 'fields-ok';

  const fac = () => props.planet.facilities;

  return (
    <div class="planet-card">
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
          Fields: {props.planet.fieldsUsed}/{props.planet.fieldsTotal}
        </div>
        <div class="planet-resources">
          <ResourceBar label="Metal" value={props.planet.resources.metal} type="metal" />
          <ResourceBar label="Crystal" value={props.planet.resources.crystal} type="crystal" />
          <ResourceBar label="Deut" value={props.planet.resources.deuterium} type="deuterium" />
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
          <BuildCountdown event={props.buildEvent!} />
        </Show>
      </div>
    </div>
  );
}
