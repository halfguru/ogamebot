import type { Logger } from 'pino';

export interface RateLimitConfig {
  defaultMinDelayMs: number;
  defaultMaxDelayMs: number;
  endpointOverrides?: Record<string, { minMs: number; maxMs: number }>;
}

export class RateLimiter {
  private lastRequestTime = 0;
  private config: RateLimitConfig;
  private log: Logger;

  constructor(config: RateLimitConfig, log: Logger) {
    this.config = config;
    this.log = log.child({ component: 'rate-limiter' });
  }

  async acquire(endpoint: string): Promise<void> {
    const now = Date.now();
    const elapsed = now - this.lastRequestTime;

    const override = this.config.endpointOverrides?.[endpoint];
    const minDelay = override?.minMs ?? this.config.defaultMinDelayMs;
    const maxDelay = override?.maxMs ?? this.config.defaultMaxDelayMs;
    const randomDelay = minDelay + Math.random() * (maxDelay - minDelay);

    const waitTime = Math.max(0, randomDelay - elapsed);
    if (waitTime > 0) {
      this.log.debug({ endpoint, waitMs: Math.round(waitTime) }, 'Rate limiting: waiting');
      await new Promise((resolve) => setTimeout(resolve, waitTime));
    }

    this.lastRequestTime = Date.now();
  }
}
