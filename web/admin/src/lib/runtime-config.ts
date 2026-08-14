/**
 * Deployment configuration for the browser bundle.
 *
 * NEXT_PUBLIC_* values are inlined by Next at build time, which pins a published
 * image to a single deployment: CI builds with the SaaS URLs, so the same image
 * cannot serve an on-prem install that talks to another host. The root layout
 * therefore re-publishes these values at request time, read from env vars that
 * are *not* inlined (LINKTOR_ADMIN_*). Build-time values stay as the fallback,
 * so a deployment that sets none of them behaves exactly as before.
 *
 * They travel as a data attribute on <html> rather than as an inline script.
 * Next emits its bundles as `async` scripts in the head, and an async script may
 * execute before the parser reaches anything in the body — so a script-based
 * handoff would race with module evaluation, and the loser would silently read
 * the build-time value. An attribute on the document element is parsed before
 * any script in the document can run, which removes the ordering question
 * instead of betting on it.
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
  /** Versão desta build do admin, para conferir o que está no ar. */
  adminVersion: string
}

/**
 * Attribute on <html> carrying the serialised config. `data-linktor-config`
 * reaches the DOM as `dataset.linktorConfig`.
 */
export const RUNTIME_CONFIG_ATTRIBUTE = 'data-linktor-config'

const BUILD_API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081/api/v1'
const BUILD_WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081/api/v1/ws'
const BUILD_ADMIN_VERSION = ''

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
    // Vem da tag da imagem, injetada pelo compose. Vazio quando não
    // configurado: é informação de operação, não contrato.
    adminVersion: process.env.LINKTOR_ADMIN_VERSION || BUILD_ADMIN_VERSION,
    wsUrl: process.env.LINKTOR_ADMIN_WS_URL || BUILD_WS_URL,
    webhookBaseUrl: (
      process.env.LINKTOR_ADMIN_WEBHOOK_BASE_URL || derivedWebhookBase
    ).replace(/\/$/, ''),
  }
}

// Parsed once: the attribute is fixed for the life of the document.
let published: Partial<RuntimeConfig> | null = null

function fromDocument(): Partial<RuntimeConfig> {
  if (published) return published
  try {
    published = JSON.parse(
      document.documentElement.getAttribute(RUNTIME_CONFIG_ATTRIBUTE) || '{}'
    ) as Partial<RuntimeConfig>
  } catch {
    published = {}
  }
  return published
}

function configured(key: keyof RuntimeConfig, buildFallback: string): string {
  if (typeof window === 'undefined') {
    return resolveRuntimeConfig()[key]
  }
  // Missing only if the page was not rendered by this layout; the build-time
  // value then keeps the SaaS deployment working.
  return fromDocument()[key] || buildFallback
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

/** Versão desta build do admin (vazio quando não configurada). */
export function getAdminVersion(): string {
  return configured('adminVersion', BUILD_ADMIN_VERSION)
}

/**
 * Origem pública da API, sem o /api/v1 — é onde vivem /health e /ready.
 *
 * Derivada da mesma fonte que getApiBaseUrl, e não de webhookBaseUrl: aquela é
 * configurável à parte e pode legitimamente apontar para outro endereço.
 */
export function getApiOrigin(): string {
  return getApiBaseUrl().replace(/\/api\/v1\/?$/, '')
}
