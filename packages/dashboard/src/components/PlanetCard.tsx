import type { APIPlanet, APIBuildEvent } from '@ogame-bot/shared';
import { Show, createSignal, onCleanup, onMount } from 'solid-js';

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toString();
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
    if (elapsed >= 0) {
      const secs = Math.floor(elapsed / 1000);
      const m = Math.floor(secs / 60);
      const s = secs % 60;
      return `Building ${props.event.buildingName} ${props.event.toLevel} (${m}m ${s}s ago)`;
    }
    return `Queued: ${props.event.buildingName} ${props.event.toLevel}`;
  };
  return <div class="build-queue">{text()}</div>;
}

export default function PlanetCard(props: { planet: APIPlanet; buildEvent?: APIBuildEvent }) {
  const fieldsPercent = () => (props.planet.fieldsUsed / props.planet.fieldsTotal) * 100;
  const fieldsClass = () =>
    fieldsPercent() >= 90 ? 'fields-critical' : fieldsPercent() >= 70 ? 'fields-warning' : 'fields-ok';

  const fac = () => props.planet.facilities;

  return (
    <div class="planet-card">
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
        <div class="resource metal">Metal: {formatNumber(props.planet.resources.metal)}</div>
        <div class="resource crystal">Crystal: {formatNumber(props.planet.resources.crystal)}</div>
        <div class="resource deuterium">Deut: {formatNumber(props.planet.resources.deuterium)}</div>
        <div class="resource energy">Energy: {formatNumber(props.planet.resources.energy)}</div>
      </div>
      <div class="planet-buildings">
        <Show when={props.planet.buildings.metalMine > 0}>
          <span>Metal: {props.planet.buildings.metalMine}</span>
        </Show>
        <Show when={props.planet.buildings.crystalMine > 0}>
          <span>Crystal: {props.planet.buildings.crystalMine}</span>
        </Show>
        <Show when={props.planet.buildings.deuteriumSynthesizer > 0}>
          <span>Deut: {props.planet.buildings.deuteriumSynthesizer}</span>
        </Show>
        <Show when={props.planet.buildings.solarPlant > 0}>
          <span>Solar: {props.planet.buildings.solarPlant}</span>
        </Show>
        <Show when={props.planet.buildings.fusionReactor > 0}>
          <span>Fusion: {props.planet.buildings.fusionReactor}</span>
        </Show>
      </div>
      <Show when={
        fac().roboticsFactory > 0 || fac().shipyard > 0 || fac().researchLab > 0 || fac().naniteFactory > 0
      }>
        <div class="planet-facilities">
          <Show when={fac().roboticsFactory > 0}>
            <span>Robot: {fac().roboticsFactory}</span>
          </Show>
          <Show when={fac().shipyard > 0}>
            <span>Shipyard: {fac().shipyard}</span>
          </Show>
          <Show when={fac().researchLab > 0}>
            <span>Lab: {fac().researchLab}</span>
          </Show>
          <Show when={fac().naniteFactory > 0}>
            <span>Nanite: {fac().naniteFactory}</span>
          </Show>
        </div>
      </Show>
      <Show when={props.buildEvent}>
        <BuildCountdown event={props.buildEvent!} />
      </Show>
    </div>
  );
}
