import { ofetch } from 'ofetch';
import { z } from 'zod';
import type { Logger } from 'pino';
import {
  ogamedResponseSchema,
  OgamedError,
  planetArraySchema,
  resourcesSchema,
  resourceBuildingsSchema,
  facilitiesSchema,
  shipsSchema,
  defenceSchema,
  fleetArraySchema,
  researchSchema,
} from '@ogame-bot/shared';
import { RateLimiter } from './rate-limiter.js';
import { retryWithBackoff } from './retry.js';

export class OgamedClient {
  private baseUrl: string;
  private rateLimiter: RateLimiter;
  private log: Logger;

  constructor(baseUrl: string, rateLimiter: RateLimiter, log: Logger) {
    this.baseUrl = baseUrl;
    this.rateLimiter = rateLimiter;
    this.log = log.child({ component: 'ogamed-client' });
  }

  // --- Authentication ---

  async login(): Promise<void> {
    await this.get('/bot/login', z.null());
  }

  async logout(): Promise<void> {
    await this.get('/bot/logout', z.null());
  }

  // --- Health Check ---

  async getServerTime(): Promise<string> {
    return this.get('/bot/server/time', z.string());
  }

  // --- User Info ---

  async isUnderAttack(): Promise<boolean> {
    return this.get('/bot/is-under-attack', z.boolean());
  }

  // --- Planets ---

  async getPlanets() {
    return this.get('/bot/planets', planetArraySchema);
  }

  async getResources(planetId: number) {
    return this.get(`/bot/planets/${planetId}/resources`, resourcesSchema);
  }

  async getResourceBuildings(planetId: number) {
    return this.get(`/bot/planets/${planetId}/resources-buildings`, resourceBuildingsSchema);
  }

  async getFacilities(planetId: number) {
    return this.get(`/bot/planets/${planetId}/facilities`, facilitiesSchema);
  }

  async getShips(planetId: number) {
    return this.get(`/bot/planets/${planetId}/ships`, shipsSchema);
  }

  async getDefence(planetId: number) {
    return this.get(`/bot/planets/${planetId}/defence`, defenceSchema);
  }

  // --- Fleets ---

  async getFleets() {
    return this.get('/bot/fleets', fleetArraySchema);
  }

  // --- Research ---

  async getResearch() {
    return this.get('/bot/get-research', researchSchema);
  }

  // --- Server Info ---

  async getServerSpeed(): Promise<number> {
    return this.get('/bot/server/speed', z.number());
  }

  async getServerVersion(): Promise<string> {
    return this.get('/bot/server/version', z.string());
  }

  // --- Core request method ---

  private async get<T>(path: string, resultSchema: z.ZodType<T>): Promise<T> {
    await this.rateLimiter.acquire(path);
    return retryWithBackoff(async () => {
      const start = Date.now();
      const raw = await ofetch<{
        Status: string;
        Code: number;
        Message: string;
        Result: unknown;
      }>(`${this.baseUrl}${path}`);

      const duration = Date.now() - start;
      this.log.debug({ path, duration, status: raw.Status }, 'API call completed');

      const envelope = ogamedResponseSchema(resultSchema).parse(raw);
      if (envelope.Status !== 'ok') {
        throw new OgamedError(envelope.Message, envelope.Code);
      }
      return envelope.Result;
    });
  }
}
