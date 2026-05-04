import type { APIBuildEvent, APIFleetSaveEvent, APIFarmAttack } from '@ogame-bot/shared';
import { For, Show } from 'solid-js';

type ActivityItem =
  | { kind: 'build'; time: string; data: APIBuildEvent }
  | { kind: 'fleet-save'; time: string; data: APIFleetSaveEvent }
  | { kind: 'farm'; time: string; data: APIFarmAttack };

interface GroupedItem {
  item: ActivityItem;
  count: number;
}

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

function itemKey(item: ActivityItem): string {
  switch (item.kind) {
    case 'build':
      return `build:${item.data.buildingName}:${item.data.fromLevel}:${item.data.toLevel}`;
    case 'fleet-save':
      return `fleet-save:${item.data.planetId}:${item.data.destPlanetId}`;
    case 'farm':
      return `farm:${item.data.targetCoord}`;
  }
}

function groupItems(items: ActivityItem[]): GroupedItem[] {
  const result: GroupedItem[] = [];
  for (const item of items) {
    const key = itemKey(item);
    const last = result[result.length - 1];
    if (last && itemKey(last.item) === key) {
      last.count++;
    } else {
      result.push({ item, count: 1 });
    }
  }
  return result;
}

function BuildText(props: { data: APIBuildEvent; count: number }) {
  return (
    <>
      {props.data.buildingName} {props.data.fromLevel}→{props.data.toLevel} on planet #{props.data.planetId} (cost: {formatNumber(props.data.costMetal)} metal)
      <Show when={props.count > 1}>
        <span class="feed-group-count"> ×{props.count}</span>
      </Show>
    </>
  );
}

function FleetSaveText(props: { data: APIFleetSaveEvent; count: number }) {
  return (
    <>
      Planet #{props.data.planetId} → #{props.data.destPlanetId}, recalled: {props.data.recalled ? 'yes' : 'no'}
      <Show when={props.count > 1}>
        <span class="feed-group-count"> ×{props.count}</span>
      </Show>
    </>
  );
}

function FarmText(props: { data: APIFarmAttack; count: number }) {
  return (
    <>
      Attack → {props.data.targetCoord}, loot: {formatNumber(props.data.metalLooted)} metal, {formatNumber(props.data.crystalLooted)} crystal
      <Show when={props.count > 1}>
        <span class="feed-group-count"> ×{props.count}</span>
      </Show>
    </>
  );
}

export default function ActivityFeed(props: {
  buildEvents: APIBuildEvent[];
  fleetSaveEvents: APIFleetSaveEvent[];
  farmAttacks: APIFarmAttack[];
}) {
  const grouped = (): GroupedItem[] => {
    const all: ActivityItem[] = [
      ...props.buildEvents.map((e) => ({ kind: 'build' as const, time: e.createdAt, data: e })),
      ...props.fleetSaveEvents.map((e) => ({ kind: 'fleet-save' as const, time: e.sentAt, data: e })),
      ...props.farmAttacks.map((e) => ({ kind: 'farm' as const, time: e.sentAt, data: e })),
    ];
    all.sort((a, b) => b.time.localeCompare(a.time));
    return groupItems(all.slice(0, 50));
  };

  return (
    <section class="activity-feed">
      <h2>Activity Feed</h2>
      <Show when={grouped().length > 0} fallback={
        <p class="empty">No recent activity</p>
      }>
        <div class="feed-list">
          <For each={grouped()}>
            {(grouped, idx) => {
              const item = grouped.item;
              const isFirst = idx() === 0;
              return (
                <div class={`feed-item ${item.kind} ${isFirst ? 'feed-item-latest' : ''}`}>
                  <Show when={item.kind === 'build'}>
                    <span class="feed-icon">🔨</span>
                    <span class="feed-text">
                      <BuildText data={(item as { kind: 'build'; data: APIBuildEvent }).data} count={grouped.count} />
                    </span>
                  </Show>
                  <Show when={item.kind === 'fleet-save'}>
                    <span class="feed-icon">🛡️</span>
                    <span class="feed-text">
                      <FleetSaveText data={(item as { kind: 'fleet-save'; data: APIFleetSaveEvent }).data} count={grouped.count} />
                    </span>
                  </Show>
                  <Show when={item.kind === 'farm'}>
                    <span class="feed-icon">⚔️</span>
                    <span class="feed-text">
                      <FarmText data={(item as { kind: 'farm'; data: APIFarmAttack }).data} count={grouped.count} />
                    </span>
                  </Show>
                  <span class="feed-time">{relativeTime(item.time)}</span>
                </div>
              );
            }}
          </For>
        </div>
      </Show>
    </section>
  );
}
