import { z } from 'zod';

// Factory that creates a schema for the ogamed response envelope with a typed Result
export const ogamedResponseSchema = <T extends z.ZodTypeAny>(resultSchema: T) =>
  z.object({
    Status: z.enum(['ok', 'error']),
    Code: z.number(),
    Message: z.string(),
    Result: resultSchema,
  });

// Type inference helper
export type OgamedResponse<T> = {
  Status: 'ok' | 'error';
  Code: number;
  Message: string;
  Result: T;
};

// Error class for ogamed errors
export class OgamedError extends Error {
  constructor(
    message: string,
    public readonly code: number,
  ) {
    super(message);
    this.name = 'OgamedError';
  }
}
