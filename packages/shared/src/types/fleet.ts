import type { MissionType } from '../constants/missions.js';
import type { ShipId } from '../constants/ships.js';
import type { Coordinate } from './planet.js';

export interface Fleet {
  id: number;
  mission: MissionType;
  returnFlight: boolean;
  origin: Coordinate;
  destination: Coordinate;
  arrivalTime: number; // Unix timestamp
  ships: ShipCount[];
  metal: number;
  crystal: number;
  deuterium: number;
}

export interface ShipCount {
  id: ShipId;
  count: number;
}

export interface FleetSlots {
  total: number;
  inUse: number;
}
