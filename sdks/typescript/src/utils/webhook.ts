/**
 * Webhook signature verification utilities
 */

import { createHmac, timingSafeEqual } from 'crypto';
import type { WebhookEvent, WebhookEventType } from '../types/webhook';

const SIGNATURE_HEADER = 'x-linktor-signature';
const TIMESTAMP_HEADER = 'x-linktor-timestamp';
const DEFAULT_TOLERANCE = 300; // 5 minutes

export interface WebhookVerificationOptions {
  /**
   * Maximum age of the webhook in seconds
   * @default 300 (5 minutes)
   */
  tolerance?: number;
}

export interface WebhookHeaders {
  [key: string]: string | string[] | undefined;
}

/**
 * Verify webhook signature
 *
 * @param payload - Raw request body as string or Buffer
 * @param signature - Signature from x-linktor-signature header
 * @param secret - Webhook secret
 * @param options - Verification options
 * @returns true if signature is valid
 */
export function verifyWebhookSignature(
  payload: string | Buffer,
  signature: string,
  secret: string,
  options?: WebhookVerificationOptions
): boolean {
  if (!signature || !secret) {
    return false;
  }

  const payloadString = typeof payload === 'string' ? payload : payload.toString('utf8');

  // Compute expected signature
  const expectedSignature = computeSignature(payloadString, secret);

  // Timing-safe comparison
  try {
    const sigBuffer = Buffer.from(signature, 'hex');
    const expectedBuffer = Buffer.from(expectedSignature, 'hex');

    if (sigBuffer.length !== expectedBuffer.length) {
      return false;
    }

    return timingSafeEqual(sigBuffer, expectedBuffer);
  } catch {
    return false;
  }
}

/**
 * Verify webhook with timestamp validation
 *
 * @param payload - Raw request body
 * @param headers - Request headers
 * @param secret - Webhook secret
 * @param options - Verification options
 * @returns true if signature and timestamp are valid
 */
export function verifyWebhook(
  payload: string | Buffer,
  headers: WebhookHeaders,
  secret: string,
  options?: WebhookVerificationOptions
): boolean {
  const signature = getHeader(headers, SIGNATURE_HEADER);
  const timestamp = getHeader(headers, TIMESTAMP_HEADER);

  if (!signature) {
    return false;
  }

  // Verify timestamp if present
  if (timestamp) {
    const tolerance = options?.tolerance ?? DEFAULT_TOLERANCE;
    if (!verifyTimestamp(timestamp, tolerance)) {
      return false;
    }
  }

  return verifyWebhookSignature(payload, signature, secret, options);
}

/**
 * Compute HMAC-SHA256 signature
 */
export function computeSignature(payload: string, secret: string): string {
  return createHmac('sha256', secret).update(payload, 'utf8').digest('hex');
}

/**
 * Verify timestamp is within tolerance
 */
function verifyTimestamp(timestamp: string, toleranceSeconds: number): boolean {
  const webhookTime = parseInt(timestamp, 10);
  if (isNaN(webhookTime)) {
    return false;
  }

  const now = Math.floor(Date.now() / 1000);
  const diff = Math.abs(now - webhookTime);

  return diff <= toleranceSeconds;
}

/**
 * Get header value (handles arrays)
 */
function getHeader(headers: WebhookHeaders, name: string): string | undefined {
  const value = headers[name] || headers[name.toLowerCase()];
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
}

/**
 * Parse and validate webhook event
 *
 * @param payload - Raw request body
 * @param headers - Request headers
 * @param secret - Webhook secret
 * @param options - Verification options
 * @returns Parsed webhook event
 * @throws Error if verification fails or payload is invalid
 */
export function constructEvent<T = unknown>(
  payload: string | Buffer,
  headers: WebhookHeaders,
  secret: string,
  options?: WebhookVerificationOptions
): WebhookEvent<T> {
  if (!verifyWebhook(payload, headers, secret, options)) {
    throw new Error('Webhook signature verification failed');
  }

  const payloadString = typeof payload === 'string' ? payload : payload.toString('utf8');

  try {
    const event = JSON.parse(payloadString) as WebhookEvent<T>;

    if (!event.id || !event.type || !event.timestamp) {
      throw new Error('Invalid webhook event structure');
    }

    return event;
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error('Invalid JSON payload');
    }
    throw error;
  }
}

/**
 * Type guard for webhook event types
 */
export function isEventType<T extends WebhookEventType>(
  event: WebhookEvent,
  type: T
): event is WebhookEvent & { type: T } {
  return event.type === type;
}

/**
 * Create a webhook handler factory for Express-like frameworks
 */
export function createWebhookHandler(
  secret: string,
  handlers: Partial<Record<WebhookEventType, (event: WebhookEvent) => void | Promise<void>>>,
  options?: WebhookVerificationOptions
) {
  return async (req: {
    body: string | Buffer | Record<string, unknown>;
    headers: WebhookHeaders;
  }): Promise<{ status: number; body?: string }> => {
    try {
      // Get raw body
      let rawBody: string | Buffer;
      if (typeof req.body === 'string' || Buffer.isBuffer(req.body)) {
        rawBody = req.body;
      } else {
        rawBody = JSON.stringify(req.body);
      }

      const event = constructEvent(rawBody, req.headers, secret, options);

      const handler = handlers[event.type];
      if (handler) {
        await handler(event);
      }

      return { status: 200 };
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      return { status: 400, body: message };
    }
  };
}
