import type { APIPlanet, APIBuildEvent } from '@ogame-bot/shared';
import PlanetCard from './PlanetCard';

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toString();
}

export default function EmpireOverview(props: {
  planets: APIPlanet[];
  buildEvents: APIBuildEvent[];
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

  return (
    <section class="empire-overview">
      <h2>Empire Overview ({props.planets.length} planets)</h2>
      <div class="resource-totals">
        Total: Metal {formatNumber(totals().metal)} | Crystal {formatNumber(totals().crystal)} | Deuterium {formatNumber(totals().deuterium)} | Energy {formatNumber(totals().energy)}
      </div>
      <div class="planet-grid">
        {props.planets.map((p) => (
          <PlanetCard planet={p} buildEvent={buildEventMap().get(p.id)} />
        ))}
      </div>
    </section>
  );
}
