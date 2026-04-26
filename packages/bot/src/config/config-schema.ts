import { z } from 'zod';

export const rateLimitConfigSchema = z.object({
  defaultMinDelayMs: z.number().min(500).default(2000),
  defaultMaxDelayMs: z.number().min(1000).default(5000),
  endpointOverrides: z
    .record(
      z.string(),
      z.object({
        minMs: z.number().min(500),
        maxMs: z.number().min(1000),
      }),
    )
    .default(() => ({})),
});

export const configSchema = z.object({
  // Account credentials (password comes from env via ${OGAME_PASSWORD})
  account: z.object({
    universe: z.string().min(1),
    username: z.string().min(1),
    password: z.string().min(1), // Will be ${OGAME_PASSWORD} in YAML, interpolated at load
  }),

  // ogamed connection
  ogamed: z.object({
    url: z.string().url().default('http://ogamed:8080'),
  }),

  // Feature toggles (each feature can be enabled/disabled independently)
  features: z
    .object({
      defender: z
        .object({
          enabled: z.boolean().default(false),
          pollIntervalMs: z.number().min(5000).default(30000),
        })
        .default(() => ({ enabled: false, pollIntervalMs: 30000 })),
      autoBuild: z
        .object({
          enabled: z.boolean().default(false),
          pollIntervalMs: z.number().min(30000).default(120000),
        })
        .default(() => ({ enabled: false, pollIntervalMs: 120000 })),
      autoFarm: z
        .object({
          enabled: z.boolean().default(false),
          pollIntervalMs: z.number().min(30000).default(300000),
        })
        .default(() => ({ enabled: false, pollIntervalMs: 300000 })),
    })
    .default(() => ({
      defender: { enabled: false, pollIntervalMs: 30000 },
      autoBuild: { enabled: false, pollIntervalMs: 120000 },
      autoFarm: { enabled: false, pollIntervalMs: 300000 },
    })),

  // Rate limiting configuration
  rateLimit: rateLimitConfigSchema.default(() => ({
    defaultMinDelayMs: 2000,
    defaultMaxDelayMs: 5000,
    endpointOverrides: {},
  })),

  // Logging
  logLevel: z.enum(['trace', 'debug', 'info', 'warn', 'error', 'fatal']).default('info'),
});

export type Config = z.infer<typeof configSchema>;
