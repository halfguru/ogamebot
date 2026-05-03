import type { APIPlanet, APIBuildEvent, BuildPlan } from '@ogame-bot/shared';
import { Show } from 'solid-js';
import PlanetCard from './PlanetCard';

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toString();
}

export default function EmpireOverview(props: {
  planets: APIPlanet[];
  buildEvents: APIBuildEvent[];
  buildPlan: BuildPlan;
}) {
  const totals = () =>
    props.planets.reduce(
      (acc, p) => ({
        metal: acc.metal + p.resources.metal,
        crystal: acc.crystal + p.resources.crystal,
        deuterium: acc.deuterium + p.resources.deuterium,
        energy: acc.energy + p.resources.energy,
      }),
      { metal: 0, crystal: 0, deuterium: 0, energy: 0 },
    );

  const buildEventMap = () => {
    const m = new Map<number, APIBuildEvent>();
    for (const e of props.buildEvents) {
      if (!m.has(e.planetId)) {
        m.set(e.planetId, e);
      }
    }
    return m;
  };

  const buildPlanMap = () => {
    const m = new Map<number, typeof props.buildPlan.planets[0]>();
    for (const p of props.buildPlan.planets) {
      m.set(p.planetId, p);
    }
    return m;
  };

  return (
    <section class="empire-overview">
      <h2>Empire Overview ({props.planets.length} planets)</h2>
      <div class="resource-totals">
        <span class="total-item metal">
          <span class="total-label">Metal</span>
          <span class="total-value">{formatNumber(totals().metal)}</span>
        </span>
        <span class="total-item crystal">
          <span class="total-label">Crystal</span>
          <span class="total-value">{formatNumber(totals().crystal)}</span>
        </span>
        <span class="total-item deuterium">
          <span class="total-label">Deuterium</span>
          <span class="total-value">{formatNumber(totals().deuterium)}</span>
        </span>
        <span class="total-item energy">
          <span class="total-label">Energy</span>
          <span class="total-value">{formatNumber(totals().energy)}</span>
        </span>
      </div>
      <div class="planet-grid">
        {props.planets.map((p) => (
          <PlanetCard planet={p} buildEvent={buildEventMap().get(p.id)} buildPlan={buildPlanMap().get(p.id)} />
        ))}
      </div>
      <Show when={props.buildPlan.research}>
        <div class="research-plan">
          <span class="plan-label">Next Research:</span>
          <span class="pill research">{props.buildPlan.research!.researchName} {props.buildPlan.research!.currentLevel} → {props.buildPlan.research!.targetLevel}</span>
        </div>
      </Show>
    </section>
  );
}
