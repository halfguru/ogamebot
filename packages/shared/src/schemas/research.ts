import { z } from 'zod';

export const researchSchema = z.object({
  EnergyTechnology: z.number().default(0),
  LaserTechnology: z.number().default(0),
  IonTechnology: z.number().default(0),
  HyperspaceTechnology: z.number().default(0),
  PlasmaTechnology: z.number().default(0),
  CombustionDrive: z.number().default(0),
  ImpulseDrive: z.number().default(0),
  HyperspaceDrive: z.number().default(0),
  EspionageTechnology: z.number().default(0),
  ComputerTechnology: z.number().default(0),
  Astrophysics: z.number().default(0),
  IntergalacticResearchNetwork: z.number().default(0),
  GravitonTechnology: z.number().default(0),
  WeaponTechnology: z.number().default(0),
  ShieldingTechnology: z.number().default(0),
  ArmourTechnology: z.number().default(0),
});
