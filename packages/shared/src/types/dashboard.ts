// Dashboard API types — match Go types in internal/dashboard/types.go

/** Resources on a planet (metal, crystal, deuterium, energy). */
export interface APIResources {
  metal: number;
  crystal: number;
  deuterium: number;
  energy: number;
}

/** Resource building levels on a planet. */
export interface APIBuildings {
  metalMine: number;
  crystalMine: number;
  deuteriumSynthesizer: number;
  solarPlant: number;
  fusionReactor: number;
  metalStorage: number;
  crystalStorage: number;
  deuteriumStorage: number;
}

export interface APIFacilities {
  roboticsFactory: number;
  shipyard: number;
  researchLab: number;
  naniteFactory: number;
}

export interface APIPlanet {
  id: number;
  name: string;
  galaxy: number;
  system: number;
  position: number;
  isMoon: boolean;
  diameter: number;
  fieldsUsed: number;
  fieldsTotal: number;
  temperatureMin: number;
  temperatureMax: number;
  resources: APIResources;
  buildings: APIBuildings;
  facilities: APIFacilities;
}

/** A fleet movement for the dashboard API. */
export interface APIFleet {
  id: number;
  mission: number;
  returnFlight: boolean;
  originGalaxy: number;
  originSystem: number;
  originPosition: number;
  destGalaxy: number;
  destSystem: number;
  destPosition: number;
  arrivalTime: number;
  metal: number;
  crystal: number;
  deuterium: number;
}

/** Research levels for the dashboard API. */
export interface APIResearch {
  energyTechnology: number;
  laserTechnology: number;
  ionTechnology: number;
  hyperspaceTechnology: number;
  plasmaTechnology: number;
  combustionDrive: number;
  impulseDrive: number;
  hyperspaceDrive: number;
  espionageTechnology: number;
  computerTechnology: number;
  astrophysics: number;
  intergalacticResearchNetwork: number;
  gravitonTechnology: number;
  weaponTechnology: number;
  shieldingTechnology: number;
  armourTechnology: number;
}

/** A build event from the auto-builder. */
export interface APIBuildEvent {
  id: number;
  planetId: number;
  buildingName: string;
  fromLevel: number;
  toLevel: number;
  costMetal: number;
  costCrystal: number;
  costDeut: number;
  roiScore: number;
  createdAt: string;
}

/** A fleet-save event from the defender. */
export interface APIFleetSaveEvent {
  id: number;
  planetId: number;
  fleetId: number;
  destPlanetId: number;
  attackId: number;
  sentAt: string;
  recallAt: string | null;
  completed: boolean;
  recalled: boolean;
}

/** A farm attack from the auto-farmer. */
export interface APIFarmAttack {
  id: number;
  fleetId: number;
  planetId: number;
  targetCoord: string;
  shipsSent: number;
  metalLooted: number;
  crystalLooted: number;
  deuteriumLooted: number;
  sentAt: string;
}

export interface PlanetBuildPlan {
  planetId: number;
  planetName: string;
  buildingId: number;
  buildingName: string;
  currentLevel: number;
  targetLevel: number;
  costMetal: number;
  costCrystal: number;
  costDeuterium: number;
  roiScore: number;
  tier: string;
  affordable: boolean;
}

export interface ResearchPlan {
  researchId: number;
  researchName: string;
  currentLevel: number;
  targetLevel: number;
  costMetal: number;
  costCrystal: number;
  costDeuterium: number;
}

export interface BuildPlan {
  planets: PlanetBuildPlan[];
  research: ResearchPlan | null;
}

/** WebSocket message types from the server. */
export type WSMessage =
  | { type: 'state_update'; data: { planets: APIPlanet[]; fleets: APIFleet[]; research: APIResearch } }
  | { type: 'build_event'; data: APIBuildEvent }
  | { type: 'fleet_save_event'; data: APIFleetSaveEvent }
  | { type: 'farm_attack'; data: APIFarmAttack };
