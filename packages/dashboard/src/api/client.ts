import type { APIPlanet, APIFleet, APIResearch, APIBuildEvent, APIFleetSaveEvent, APIFarmAttack } from '@ogame-bot/shared';

const API_BASE = ''; // relative — Vite proxy in dev, same-origin in prod

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(`${API_BASE}${url}`);
  if (!res.ok) {
    throw new Error(`API error ${res.status}: ${res.statusText} for ${url}`);
  }
  return res.json() as Promise<T>;
}

export async function fetchPlanets(): Promise<APIPlanet[]> {
  return fetchJSON<APIPlanet[]>('/api/planets');
}

export async function fetchFleets(): Promise<APIFleet[]> {
  return fetchJSON<APIFleet[]>('/api/fleets');
}

export async function fetchResearch(): Promise<APIResearch> {
  return fetchJSON<APIResearch>('/api/research');
}

export async function fetchBuildEvents(): Promise<APIBuildEvent[]> {
  return fetchJSON<APIBuildEvent[]>('/api/events/builds');
}

export async function fetchFleetSaveEvents(): Promise<APIFleetSaveEvent[]> {
  return fetchJSON<APIFleetSaveEvent[]>('/api/events/fleet-saves');
}

export async function fetchFarmAttacks(): Promise<APIFarmAttack[]> {
  return fetchJSON<APIFarmAttack[]>('/api/events/farm-attacks');
}

export async function fetchAllState(): Promise<{
  planets: APIPlanet[];
  fleets: APIFleet[];
  research: APIResearch;
}> {
  const [planets, fleets, research] = await Promise.all([
    fetchPlanets(),
    fetchFleets(),
    fetchResearch(),
  ]);
  return { planets, fleets, research };
}
