import { createSignal, onMount, onCleanup } from 'solid-js';
import { fetchAllState, fetchBuildEvents, fetchFleetSaveEvents, fetchFarmAttacks } from './api/client';
import { createWSClient } from './api/websocket';
import type { APIPlanet, APIFleet, APIResearch, APIBuildEvent, APIFleetSaveEvent, APIFarmAttack } from '@ogame-bot/shared';
import Header from './components/Header';
import EmpireOverview from './components/EmpireOverview';
import FleetMovements from './components/FleetMovements';
import ResearchPanel from './components/ResearchPanel';
import ActivityFeed from './components/ActivityFeed';

export default function App() {
  const [planets, setPlanets] = createSignal<APIPlanet[]>([]);
  const [fleets, setFleets] = createSignal<APIFleet[]>([]);
  const [research, setResearch] = createSignal<APIResearch | null>(null);
  const [buildEvents, setBuildEvents] = createSignal<APIBuildEvent[]>([]);
  const [fleetSaveEvents, setFleetSaveEvents] = createSignal<APIFleetSaveEvent[]>([]);
  const [farmAttacks, setFarmAttacks] = createSignal<APIFarmAttack[]>([]);
  const [connected, setConnected] = createSignal(false);
  const [lastUpdate, setLastUpdate] = createSignal<Date | null>(null);

  onMount(async () => {
    try {
      const [state, builds, saves, attacks] = await Promise.all([
        fetchAllState(),
        fetchBuildEvents(),
        fetchFleetSaveEvents(),
        fetchFarmAttacks(),
      ]);
      setPlanets(state.planets);
      setFleets(state.fleets);
      setResearch(state.research);
      setBuildEvents(builds);
      setFleetSaveEvents(saves);
      setFarmAttacks(attacks);
      setLastUpdate(new Date());
    } catch (err) {
      console.error('Failed to load initial state:', err);
    }

    const ws = createWSClient(
      (msg) => {
        switch (msg.type) {
          case 'state_update':
            setPlanets(msg.data.planets);
            setFleets(msg.data.fleets);
            setResearch(msg.data.research);
            break;
          case 'build_event':
            setBuildEvents((prev) => [msg.data, ...prev].slice(0, 50));
            break;
          case 'fleet_save_event':
            setFleetSaveEvents((prev) => [msg.data, ...prev].slice(0, 20));
            break;
          case 'farm_attack':
            setFarmAttacks((prev) => [msg.data, ...prev].slice(0, 50));
            break;
        }
        setLastUpdate(new Date());
      },
      (status) => setConnected(status),
    );
    ws.connect();
    onCleanup(() => ws.disconnect());
  });

  return (
    <div class="dashboard">
      <Header connected={connected()} lastUpdate={lastUpdate()} />
      <main>
        <EmpireOverview planets={planets()} buildEvents={buildEvents()} />
        <ResearchPanel research={research()} />
        <FleetMovements fleets={fleets()} />
        <ActivityFeed buildEvents={buildEvents()} fleetSaveEvents={fleetSaveEvents()} farmAttacks={farmAttacks()} />
      </main>
    </div>
  );
}
