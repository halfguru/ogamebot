import { z } from 'zod';

/** Zod schema for APIResources. */
export const apiResourcesSchema = z.object({
  metal: z.number(),
  crystal: z.number(),
  deuterium: z.number(),
  energy: z.number(),
  metalStorage: z.number(),
  crystalStorage: z.number(),
  deuteriumStorage: z.number(),
  metalProduction: z.number(),
  crystalProduction: z.number(),
  deuteriumProduction: z.number(),
});

/** Zod schema for APIBuildings. */
export const apiBuildingsSchema = z.object({
  metalMine: z.number(),
  crystalMine: z.number(),
  deuteriumSynthesizer: z.number(),
  solarPlant: z.number(),
  fusionReactor: z.number(),
  metalStorage: z.number(),
  crystalStorage: z.number(),
  deuteriumStorage: z.number(),
});

export const apiFacilitiesSchema = z.object({
  roboticsFactory: z.number(),
  shipyard: z.number(),
  researchLab: z.number(),
  naniteFactory: z.number(),
});

export const apiPlanetSchema = z.object({
  id: z.number(),
  name: z.string(),
  galaxy: z.number(),
  system: z.number(),
  position: z.number(),
  isMoon: z.boolean(),
  diameter: z.number(),
  fieldsUsed: z.number(),
  fieldsTotal: z.number(),
  temperatureMin: z.number(),
  temperatureMax: z.number(),
  imageType: z.string(),
  resources: apiResourcesSchema,
  buildings: apiBuildingsSchema,
  facilities: apiFacilitiesSchema,
});

/** Zod schema for APIPlanet array. */
export const apiPlanetArraySchema = z.array(apiPlanetSchema);

/** Zod schema for APIFleet. */
export const apiFleetSchema = z.object({
  id: z.number(),
  mission: z.number(),
  returnFlight: z.boolean(),
  originGalaxy: z.number(),
  originSystem: z.number(),
  originPosition: z.number(),
  destGalaxy: z.number(),
  destSystem: z.number(),
  destPosition: z.number(),
  arrivalTime: z.number(),
  metal: z.number(),
  crystal: z.number(),
  deuterium: z.number(),
});

/** Zod schema for APIFleet array. */
export const apiFleetArraySchema = z.array(apiFleetSchema);

/** Zod schema for APIResearch. */
export const apiResearchSchema = z.object({
  energyTechnology: z.number(),
  laserTechnology: z.number(),
  ionTechnology: z.number(),
  hyperspaceTechnology: z.number(),
  plasmaTechnology: z.number(),
  combustionDrive: z.number(),
  impulseDrive: z.number(),
  hyperspaceDrive: z.number(),
  espionageTechnology: z.number(),
  computerTechnology: z.number(),
  astrophysics: z.number(),
  intergalacticResearchNetwork: z.number(),
  gravitonTechnology: z.number(),
  weaponTechnology: z.number(),
  shieldingTechnology: z.number(),
  armourTechnology: z.number(),
});

/** Zod schema for APIBuildEvent. */
export const apiBuildEventSchema = z.object({
  id: z.number(),
  planetId: z.number(),
  buildingName: z.string(),
  fromLevel: z.number(),
  toLevel: z.number(),
  costMetal: z.number(),
  costCrystal: z.number(),
  costDeut: z.number(),
  roiScore: z.number(),
  createdAt: z.string(),
});

/** Zod schema for APIBuildEvent array. */
export const apiBuildEventArraySchema = z.array(apiBuildEventSchema);

/** Zod schema for APIFleetSaveEvent. */
export const apiFleetSaveEventSchema = z.object({
  id: z.number(),
  planetId: z.number(),
  fleetId: z.number(),
  destPlanetId: z.number(),
  attackId: z.number(),
  sentAt: z.string(),
  recallAt: z.string().nullable(),
  completed: z.boolean(),
  recalled: z.boolean(),
});

/** Zod schema for APIFleetSaveEvent array. */
export const apiFleetSaveEventArraySchema = z.array(apiFleetSaveEventSchema);

/** Zod schema for APIFarmAttack. */
export const apiFarmAttackSchema = z.object({
  id: z.number(),
  fleetId: z.number(),
  planetId: z.number(),
  targetCoord: z.string(),
  shipsSent: z.number(),
  metalLooted: z.number(),
  crystalLooted: z.number(),
  deuteriumLooted: z.number(),
  sentAt: z.string(),
});

/** Zod schema for APIFarmAttack array. */
export const apiFarmAttackArraySchema = z.array(apiFarmAttackSchema);

/** Zod schema for WSMessage (discriminated union on type field). */
export const wsMessageSchema: z.ZodType = z.discriminatedUnion('type', [
  z.object({
    type: z.literal('state_update'),
    data: z.object({
      planets: apiPlanetArraySchema,
      fleets: apiFleetArraySchema,
      research: apiResearchSchema,
    }),
  }),
  z.object({
    type: z.literal('build_event'),
    data: apiBuildEventSchema,
  }),
  z.object({
    type: z.literal('fleet_save_event'),
    data: apiFleetSaveEventSchema,
  }),
  z.object({
    type: z.literal('farm_attack'),
    data: apiFarmAttackSchema,
  }),
]);
