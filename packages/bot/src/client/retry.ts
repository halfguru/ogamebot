import type { Logger } from 'pino';
import { OgamedError } from '@ogame-bot/shared';

export interface RetryConfig {
  maxAttempts: number; // default 3
  baseDelayMs: number; // default 1000
  maxDelayMs: number; // default 30000
  jitterFactor: number; // default 0.25 (±25% jitter)
}

const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxAttempts: 3,
  baseDelayMs: 1000,
  maxDelayMs: 30000,
  jitterFactor: 0.25,
};

export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  config: Partial<RetryConfig> = {},
  log?: Logger,
): Promise<T> {
  const opts = { ...DEFAULT_RETRY_CONFIG, ...config };

  for (let attempt = 1; attempt <= opts.maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      // Don't retry on Zod validation errors or 4xx client errors
      if (error instanceof OgamedError && error.code >= 400 && error.code < 500) {
        throw error;
      }
      // Don't retry on ZodError (validation failure = bad data, retrying won't help)
      if (error instanceof Error && error.name === 'ZodError') {
        throw error;
      }

      if (attempt >= opts.maxAttempts) {
        throw error;
      }

      // Exponential backoff with jitter
      const baseDelay = Math.min(opts.baseDelayMs * Math.pow(2, attempt - 1), opts.maxDelayMs);
      const jitter = baseDelay * opts.jitterFactor * (Math.random() * 2 - 1);
      const delay = Math.max(0, baseDelay + jitter);

      log?.warn(
        { attempt, maxAttempts: opts.maxAttempts, delayMs: Math.round(delay), error: String(error) },
        'Request failed, retrying',
      );

      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }

  // Unreachable but TypeScript needs it
  throw new Error('Retry loop exited unexpectedly');
}
