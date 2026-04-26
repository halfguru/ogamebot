import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import yaml from 'js-yaml';
import { config as loadDotenv } from 'dotenv';
import { configSchema, type Config } from './config-schema.js';

export function loadConfig(configPath: string): Config {
  // Load .env file first so env vars are available for interpolation
  loadDotenv();

  const raw = readFileSync(resolve(configPath), 'utf-8');

  // Interpolate ${ENV_VAR} references
  const interpolated = raw.replace(/\$\{(\w+)\}/g, (_, varName: string) => {
    const value = process.env[varName];
    if (value === undefined) {
      throw new Error(
        `Environment variable ${varName} is referenced in config but not set. ` +
          `Set it in .env file or environment.`,
      );
    }
    return value;
  });

  const parsed = yaml.load(interpolated);

  const result = configSchema.safeParse(parsed);
  if (!result.success) {
    console.error('Invalid configuration:');
    for (const issue of result.error.issues) {
      console.error(`  ${issue.path.join('.')}: ${issue.message}`);
    }
    process.exit(1);
  }

  return result.data;
}
