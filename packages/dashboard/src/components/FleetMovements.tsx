import type { APIFleet } from '@ogame-bot/shared';
import { For, Show, createSignal, onCleanup, onMount } from 'solid-js';

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

function missionBadgeClass(mission: number): string {
  switch (mission) {
    case 1: case 2: return 'attack';
    case 3: return 'transport';
    case 4: return 'deploy';
    case 6: return 'espionage';
    case 8: return 'harvest';
    case 15: return 'expedition';
    default: return 'default';
  }
}

function getCountdown(arrivalTime: number): { text: string; className: string } {
  const now = Date.now();
  const target = arrivalTime * 1000;
  const diff = target - now;
  if (diff <= 0) return { text: 'Arrived', className: 'countdown-arrived' };
  const totalSec = Math.floor(diff / 1000);
  const hrs = Math.floor(totalSec / 3600);
  const mins = Math.floor((totalSec % 3600) / 60);
  const secs = totalSec % 60;
  const parts: string[] = [];
  if (hrs > 0) parts.push(`${hrs}h`);
  if (mins > 0 || hrs > 0) parts.push(`${mins}m`);
  parts.push(`${secs}s`);
  const totalMins = totalSec / 60;
  if (totalMins > 10) return { text: parts.join(' '), className: 'countdown-green' };
  if (totalMins > 5) return { text: parts.join(' '), className: 'countdown-yellow' };
  return { text: parts.join(' '), className: 'countdown-red pulse' };
}

function CountdownCell(props: { arrivalTime: number }) {
  const [tick, setTick] = createSignal(0);
  let interval: ReturnType<typeof setInterval> | undefined;
  onMount(() => {
    interval = setInterval(() => setTick((t) => t + 1), 1000);
  });
  onCleanup(() => {
    if (interval) clearInterval(interval);
  });
  const info = () => {
    tick();
    return getCountdown(props.arrivalTime);
  };
  return <td class={info().className}>{info().text}</td>;
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toString();
}

export default function FleetMovements(props: { fleets: APIFleet[] }) {
  const totalCargo = () =>
    props.fleets.reduce(
      (acc, f) => ({
        metal: acc.metal + f.metal,
        crystal: acc.crystal + f.crystal,
        deuterium: acc.deuterium + f.deuterium,
      }),
      { metal: 0, crystal: 0, deuterium: 0 },
    );

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
              <th>&rarr;</th>
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
                  <td>{fleet.returnFlight ? '\u2190' : '\u2192'}</td>
                  <td>
                    [{fleet.destGalaxy}:{fleet.destSystem}:{fleet.destPosition}]
                  </td>
                  <td>
                    <span class={`mission-badge ${missionBadgeClass(fleet.mission)}`}>
                      {missionNames[fleet.mission] || `#${fleet.mission}`}
                    </span>
                    {fleet.returnFlight ? ' (R)' : ''}
                  </td>
                  <CountdownCell arrivalTime={fleet.arrivalTime} />
                  <td>
                    M:{fleet.metal} C:{fleet.crystal} D:{fleet.deuterium}
                  </td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
        <div class="fleet-summary">
          <span>Total in transit: <strong>M:{formatNumber(totalCargo().metal)}</strong></span>
          <span><strong>C:{formatNumber(totalCargo().crystal)}</strong></span>
          <span><strong>D:{formatNumber(totalCargo().deuterium)}</strong></span>
        </div>
      </Show>
    </section>
  );
}
