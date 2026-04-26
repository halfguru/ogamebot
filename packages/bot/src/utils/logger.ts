import pino from 'pino';

export function createLogger(name: string, level: string = 'info') {
  return pino({
    name,
    level,
    transport:
      level === 'debug' || level === 'trace'
        ? { target: 'pino-pretty', options: { colorize: true } }
        : undefined,
  });
}
