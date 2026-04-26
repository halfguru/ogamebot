import type { APIFleet } from '@ogame-bot/shared';
import { For, Show } from 'solid-js';

const missionNames: Record<number, string> = {
  1: 'Attack',
  2: 'ACS Attack',
  3: 'Transport',
  4: 'Deploy',
  5: 'Hold',
  6: 'Espionage',
  8: 'Harvest',
  9: 'Moon Destruction',
  15: 'Expedition',
};

function formatArrival(arrivalTime: number): string {
  const date = new Date(arrivalTime * 1000);
  const now = new Date();
  const diff = date.getTime() - now.getTime();
  if (diff <= 0) return 'Arrived';
  const mins = Math.floor(diff / 60000);
  const hrs = Math.floor(mins / 60);
  if (hrs > 0) return `${hrs}h ${mins % 60}m`;
  return `${mins}m`;
}

export default function FleetMovements(props: { fleets: APIFleet[] }) {
  return (
    <section class="fleet-movements">
      <h2>Fleet Movements ({props.fleets.length} active)</h2>
      <Show
        when={props.fleets.length > 0}
        fallback={<p class="empty">No active fleet movements</p>}
      >
        <table>
          <thead>
            <tr>
              <th>Origin</th>
              <th>→</th>
              <th>Destination</th>
              <th>Mission</th>
              <th>Arrival</th>
              <th>Cargo</th>
            </tr>
          </thead>
          <tbody>
            <For each={props.fleets}>
              {(fleet) => (
                <tr class={fleet.returnFlight ? 'returning' : 'outgoing'}>
                  <td>
                    [{fleet.originGalaxy}:{fleet.originSystem}:{fleet.originPosition}]
                  </td>
                  <td>{fleet.returnFlight ? '←' : '→'}</td>
                  <td>
                    [{fleet.destGalaxy}:{fleet.destSystem}:{fleet.destPosition}]
                  </td>
                  <td>
                    {missionNames[fleet.mission] || `#${fleet.mission}`}
                    {fleet.returnFlight ? ' (R)' : ''}
                  </td>
                  <td>{formatArrival(fleet.arrivalTime)}</td>
                  <td>
                    M:{fleet.metal} C:{fleet.crystal} D:{fleet.deuterium}
                  </td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </Show>
    </section>
  );
}
