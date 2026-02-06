import { z } from 'zod';

const ConfigSchema = z.object({
  apiUrl: z.string().url().default('http://localhost:8080/api/v1'),
  apiKey: z.string().optional(),
  accessToken: z.string().optional(),
  timeout: z.number().default(30000),
  maxRetries: z.number().default(3),
  retryDelay: z.number().default(1000),
});

export type Config = z.infer<typeof ConfigSchema>;

export function loadConfig(): Config {
  const rawConfig = {
    apiUrl: process.env.LINKTOR_API_URL || 'http://localhost:8080/api/v1',
    apiKey: process.env.LINKTOR_API_KEY,
    accessToken: process.env.LINKTOR_ACCESS_TOKEN,
    timeout: process.env.LINKTOR_TIMEOUT ? parseInt(process.env.LINKTOR_TIMEOUT, 10) : 30000,
    maxRetries: process.env.LINKTOR_MAX_RETRIES ? parseInt(process.env.LINKTOR_MAX_RETRIES, 10) : 3,
    retryDelay: process.env.LINKTOR_RETRY_DELAY ? parseInt(process.env.LINKTOR_RETRY_DELAY, 10) : 1000,
  };

  const config = ConfigSchema.parse(rawConfig);

  if (!config.apiKey && !config.accessToken) {
    console.error('Warning: No LINKTOR_API_KEY or LINKTOR_ACCESS_TOKEN provided');
  }

  return config;
}

export const config = loadConfig();
