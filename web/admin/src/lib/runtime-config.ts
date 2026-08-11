/**
 * Deployment configuration for the browser bundle.
 *
 * NEXT_PUBLIC_* values are inlined by Next at build time, which pins a published
 * image to a single deployment: CI builds with the SaaS URLs, so the same image
 * cannot serve an on-prem install that talks to another host. The root layout
 * therefore re-publishes these values at request time as
 * `window.__LINKTOR_CONFIG__`, read from env vars that are *not* inlined
 * (LINKTOR_ADMIN_*). Build-time values stay as the fallback, so a deployment
 * that sets none of them behaves exactly as before.
 *
 * Server and browser resolve to the same string — the layout renders per request
 * (next-intl reads cookies/headers) — so SSR and hydration agree.
 *
 * A value starting with "/" is resolved against the page origin when the
 * connection is opened, which lets a single-origin deploy (admin and API behind
 * one reverse proxy) work on any hostname or IP without knowing it at install
 * time. Only the API and WebSocket URLs accept that form: they are used to open
 * connections, never rendered. WEBHOOK_BASE_URL is shown on screen and copied
 * into provider consoles, so it stays whatever the deployment configured.
 */

export type RuntimeConfig = {
  apiUrl: string
  wsUrl: string
  webhookBaseUrl: string
}

declare global {
  interface Window {
    __LINKTOR_CONFIG__?: Partial<RuntimeConfig>
  }
}

/** Name of the global the root layout writes and the browser reads. */
export const RUNTIME_CONFIG_GLOBAL = '__LINKTOR_CONFIG__'

const BUILD_API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081/api/v1'
const BUILD_WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081/api/v1/ws'
const BUILD_WEBHOOK_BASE_URL =
  process.env.NEXT_PUBLIC_WEBHOOK_BASE_URL ||
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/api\/v1\/?$/, '') ||
  'http://localhost:8081'

/**
 * The values the root layout serialises into the page. Server-side only:
 * LINKTOR_ADMIN_* are ordinary server env vars, read from the live process and
 * never inlined into the client bundle.
 */
export function resolveRuntimeConfig(): RuntimeConfig {
  const apiUrl = process.env.LINKTOR_ADMIN_API_URL || BUILD_API_URL

  // When only the API URL is overridden, derive the webhook origin from it
  // rather than falling back to the build-time value — a leftover SaaS URL on
  // an on-prem screen would be actively misleading. A single-origin install
  // derives an empty string, leaving the displayed URLs relative to the page,
  // which is why the on-prem deployment sets this one explicitly.
  const derivedWebhookBase = process.env.LINKTOR_ADMIN_API_URL
    ? apiUrl.replace(/\/api\/v1\/?$/, '')
    : BUILD_WEBHOOK_BASE_URL

  return {
    apiUrl,
    wsUrl: process.env.LINKTOR_ADMIN_WS_URL || BUILD_WS_URL,
    webhookBaseUrl: (
      process.env.LINKTOR_ADMIN_WEBHOOK_BASE_URL || derivedWebhookBase
    ).replace(/\/$/, ''),
  }
}

function configured(key: keyof RuntimeConfig, buildFallback: string): string {
  if (typeof window === 'undefined') {
    return resolveRuntimeConfig()[key]
  }
  // Absent only if the layout script failed to run; the build-time value then
  // keeps the SaaS deployment working.
  return window[RUNTIME_CONFIG_GLOBAL]?.[key] || buildFallback
}

/** Resolves an origin-relative value ("/api/v1") against the current page. */
function absolute(value: string, scheme: 'http' | 'ws'): string {
  if (!value.startsWith('/') || typeof window === 'undefined') {
    return value
  }
  const origin = window.location.origin
  return (scheme === 'ws' ? origin.replace(/^http/, 'ws') : origin) + value
}

/** Base URL of the REST API, e.g. "https://api.linktor.dev/api/v1". */
export function getApiBaseUrl(): string {
  return absolute(configured('apiUrl', BUILD_API_URL), 'http')
}

/** WebSocket endpoint, e.g. "wss://api.linktor.dev/api/v1/ws". */
export function getWsBaseUrl(): string {
  return absolute(configured('wsUrl', BUILD_WS_URL), 'ws')
}

/**
 * Public origin of the API, used to compose the webhook URLs shown in the
 * channel screens and the WebChat embed snippet.
 */
export const WEBHOOK_BASE_URL = configured('webhookBaseUrl', BUILD_WEBHOOK_BASE_URL)
