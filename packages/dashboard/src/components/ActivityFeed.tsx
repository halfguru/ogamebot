import type { APIBuildEvent, APIFleetSaveEvent, APIFarmAttack } from '@ogame-bot/shared';
import { For, Show } from 'solid-js';

type ActivityItem =
  | { kind: 'build'; time: string; data: APIBuildEvent }
  | { kind: 'fleet-save'; time: string; data: APIFleetSaveEvent }
  | { kind: 'farm'; time: string; data: APIFarmAttack };

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toString();
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const secs = Math.floor(diff / 1000);
  if (secs < 60) return `${secs}s ago`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

export default function ActivityFeed(props: {
  buildEvents: APIBuildEvent[];
  fleetSaveEvents: APIFleetSaveEvent[];
  farmAttacks: APIFarmAttack[];
}) {
  const items = (): ActivityItem[] => {
    const all: ActivityItem[] = [
      ...props.buildEvents.map((e) => ({ kind: 'build' as const, time: e.createdAt, data: e })),
      ...props.fleetSaveEvents.map((e) => ({ kind: 'fleet-save' as const, time: e.sentAt, data: e })),
      ...props.farmAttacks.map((e) => ({ kind: 'farm' as const, time: e.sentAt, data: e })),
    ];
    all.sort((a, b) => b.time.localeCompare(a.time));
    return all.slice(0, 30);
  };

  return (
    <section class="activity-feed">
      <h2>Activity Feed</h2>
      <Show when={items().length === 0} fallback={
        <div class="feed-list">
          <For each={items()}>
            {(item) => (
              <div class={`feed-item ${item.kind}`}>
                <Show when={item.kind === 'build'}>
                  <span class="feed-icon">🔨</span>
                  <span class="feed-text">
                    {(item as { kind: 'build'; data: APIBuildEvent }).data.buildingName}{' '}
                    {(item as { kind: 'build'; data: APIBuildEvent }).data.fromLevel}→
                    {(item as { kind: 'build'; data: APIBuildEvent }).data.toLevel} on planet #
                    {(item as { kind: 'build'; data: APIBuildEvent }).data.planetId} (cost:{' '}
                    {formatNumber((item as { kind: 'build'; data: APIBuildEvent }).data.costMetal)} metal)
                  </span>
                </Show>
                <Show when={item.kind === 'fleet-save'}>
                  <span class="feed-icon">🛡️</span>
                  <span class="feed-text">
                    Planet #{(item as { kind: 'fleet-save'; data: APIFleetSaveEvent }).data.planetId} → #
                    {(item as { kind: 'fleet-save'; data: APIFleetSaveEvent }).data.destPlanetId}, recalled:{' '}
                    {(item as { kind: 'fleet-save'; data: APIFleetSaveEvent }).data.recalled ? 'yes' : 'no'}
                  </span>
                </Show>
                <Show when={item.kind === 'farm'}>
                  <span class="feed-icon">⚔️</span>
                  <span class="feed-text">
                    Attack →{' '}
                    {(item as { kind: 'farm'; data: APIFarmAttack }).data.targetCoord}, loot:{' '}
                    {formatNumber((item as { kind: 'farm'; data: APIFarmAttack }).data.metalLooted)} metal,{' '}
                    {formatNumber((item as { kind: 'farm'; data: APIFarmAttack }).data.crystalLooted)} crystal
                  </span>
                </Show>
                <span class="feed-time">{relativeTime(item.time)}</span>
              </div>
            )}
          </For>
        </div>
      }>
        <p class="empty">No recent activity</p>
      </Show>
    </section>
  );
}
