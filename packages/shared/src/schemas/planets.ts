import { z } from 'zod';

export const coordinateSchema = z.object({
  galaxy: z.number(),
  system: z.number(),
  position: z.number(),
  type: z.enum(['planet', 'moon']),
});

export const resourcesSchema = z.object({
  Metal: z.number().default(0),
  Crystal: z.number().default(0),
  Deuterium: z.number().default(0),
  Energy: z.number().default(0),
  Darkmatter: z.number().optional().default(0),
});

// ogamed returns arrays of planet objects with specific keys
export const planetSchema = z.object({
  ID: z.number(),
  Name: z.string(),
  Coordinate: coordinateSchema,
  Diameter: z.number(),
  FieldsUsed: z.number(),
  FieldsTotal: z.number(),
  TemperatureMin: z.number(),
  TemperatureMax: z.number(),
  IsMoon: z.boolean().default(false),
});

export const planetArraySchema = z.array(planetSchema);

// Resource buildings response
export const resourceBuildingsSchema = z.object({
  MetalMine: z.number().default(0),
  CrystalMine: z.number().default(0),
  DeuteriumSynthesizer: z.number().default(0),
  SolarPlant: z.number().default(0),
  FusionReactor: z.number().default(0),
  MetalStorage: z.number().default(0),
  CrystalStorage: z.number().default(0),
  DeuteriumStorage: z.number().default(0),
});

// Facilities response
export const facilitiesSchema = z.object({
  RoboticsFactory: z.number().default(0),
  Shipyard: z.number().default(0),
  ResearchLab: z.number().default(0),
  AllianceDepot: z.number().default(0),
  MissileSilo: z.number().default(0),
  NaniteFactory: z.number().default(0),
  Terraformer: z.number().default(0),
  SpaceDock: z.number().default(0),
});

// Ships response
export const shipsSchema = z.object({
  LightFighter: z.number().default(0),
  HeavyFighter: z.number().default(0),
  Cruiser: z.number().default(0),
  Battleship: z.number().default(0),
  Battlecruiser: z.number().default(0),
  Bomber: z.number().default(0),
  Destroyer: z.number().default(0),
  Deathstar: z.number().default(0),
  SmallCargo: z.number().default(0),
  LargeCargo: z.number().default(0),
  ColonyShip: z.number().default(0),
  Recycler: z.number().default(0),
  EspionageProbe: z.number().default(0),
  SolarSatellite: z.number().default(0),
});

// Defence response
export const defenceSchema = z.object({
  RocketLauncher: z.number().default(0),
  LightLaser: z.number().default(0),
  HeavyLaser: z.number().default(0),
  GaussCannon: z.number().default(0),
  IonCannon: z.number().default(0),
  PlasmaTurret: z.number().default(0),
  SmallShield: z.number().default(0),
  LargeShield: z.number().default(0),
  AntiBallisticMissile: z.number().default(0),
  InterplanetaryMissile: z.number().default(0),
});
