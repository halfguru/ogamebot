import type { APIPlanet } from '@ogame-bot/shared';
import PlanetCard from './PlanetCard';

export default function EmpireOverview(props: { planets: APIPlanet[] }) {
  return (
    <section class="empire-overview">
      <h2>Empire Overview ({props.planets.length} planets)</h2>
      <div class="planet-grid">
        {props.planets.map((p) => (
          <PlanetCard planet={p} />
        ))}
      </div>
    </section>
  );
}
