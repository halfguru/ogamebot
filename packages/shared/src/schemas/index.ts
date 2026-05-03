export { ogamedResponseSchema, OgamedError } from './ogamed.js';
export type { OgamedResponse } from './ogamed.js';
export {
  coordinateSchema,
  resourcesSchema,
  planetSchema,
  planetArraySchema,
  resourceBuildingsSchema,
  facilitiesSchema,
  shipsSchema,
  defenceSchema,
} from './planets.js';
export { fleetSchema, fleetArraySchema } from './fleets.js';
export { researchSchema } from './research.js';
export {
  apiResourcesSchema,
  apiBuildingsSchema,
  apiFacilitiesSchema,
  apiPlanetSchema,
  apiPlanetArraySchema,
  apiFleetSchema,
  apiFleetArraySchema,
  apiResearchSchema,
  apiBuildEventSchema,
  apiBuildEventArraySchema,
  apiFleetSaveEventSchema,
  apiFleetSaveEventArraySchema,
  apiFarmAttackSchema,
  apiFarmAttackArraySchema,
  wsMessageSchema,
} from './dashboard.js';
