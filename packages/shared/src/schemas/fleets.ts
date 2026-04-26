import { z } from 'zod';
import { coordinateSchema } from './planets.js';

const shipCountSchema = z.object({
  ID: z.number(),
  Number: z.number().default(0),
});

export const fleetSchema = z.object({
  ID: z.number(),
  Mission: z.number(),
  ReturnFlight: z.boolean().default(false),
  Origin: coordinateSchema,
  Destination: coordinateSchema,
  ArrivalTime: z.number(),
  Ships: z.array(shipCountSchema).default([]),
  Metal: z.number().default(0),
  Crystal: z.number().default(0),
  Deuterium: z.number().default(0),
});

export const fleetArraySchema = z.array(fleetSchema);
