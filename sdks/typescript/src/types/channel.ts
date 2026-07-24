/**
 * Channel types — modeled on the backend wire contract (snake_case).
 * Credentials are write-only and never present on a response.
 */

import type { ChannelType, PaginationParams } from './common';

/** Live connection state (wire field `connection_status`), distinct from `enabled`. */
export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error';

/** WhatsApp Business App + Cloud API coexistence state. */
export type CoexistenceStatus =
  | 'inactive'
  | 'pending'
  | 'active'
  | 'warning'
  | 'disconnected';

/**
 * Channel `config` is a flat string map on the wire (e.g.
 * `{ phone_number_id: "...", waba_id: "..." }`). Secret values are redacted to
 * `"__redacted__"` in responses.
 */
export type ChannelConfig = Record<string, string>;

export interface Channel {
  id: string;
  tenant_id: string;
  type: ChannelType;
  name: string;
  identifier?: string;
  /** System-level enable flag (distinct from connection_status). */
  enabled: boolean;
  connection_status: ConnectionStatus;
  config?: ChannelConfig;
  webhook_url?: string;
  created_at: string;
  updated_at: string;
  // WhatsApp coexistence
  is_coexistence?: boolean;
  waba_id?: string;
  last_echo_at?: string;
  coexistence_status?: CoexistenceStatus;
  message_template_namespace?: string;
}

// Request types

/**
 * `config` holds non-secret settings (phone_number_id, waba_id, ...).
 * `credentials` holds secrets (access_token, bot_token, ...) — stored encrypted,
 * never returned. `webhook_url` is the external endpoint Linktor delivers signed
 * inbound/status events to.
 */
export interface CreateChannelInput {
  name: string;
  type: ChannelType;
  identifier?: string;
  config?: Record<string, string>;
  credentials?: Record<string, string>;
  webhook_url?: string;
}

/**
 * Update reuses the create body shape. `credentials`, when present, replace the
 * stored secrets; omit it (or send the redacted placeholder) to keep them.
 */
export interface UpdateChannelInput {
  name?: string;
  identifier?: string;
  config?: Record<string, string>;
  credentials?: Record<string, string>;
  webhook_url?: string;
}

/**
 * Result of connecting a channel. For WhatsApp Web-style linking, `qr_code`
 * carries the payload to render and `expires_in` its lifetime in seconds — call
 * `connect` again to refresh an expired code. `pair_code` is the phone-linking
 * code. When `passkey_required` is true the account is passkey-locked and must
 * be linked by signing `passkey_challenge` (submit via the passkey endpoint),
 * not by QR.
 */
export interface ConnectResult {
  /** Present when the response carries a channel; the wire may send null. */
  channel?: Channel;
  qr_code?: string;
  expires_in?: number;
  pair_code?: string;
  passkey_required?: boolean;
  passkey_challenge?: unknown;
}

/** Body for requesting a WhatsApp pairing code. */
export interface PairCodeInput {
  phone_number: string;
}

export interface ListChannelsParams extends PaginationParams {
  type?: ChannelType;
  status?: ConnectionStatus;
  search?: string;
}

export interface ChannelCapabilities {
  supportedContentTypes: string[];
  supportsMedia: boolean;
  supportsButtons: boolean;
  supportsLists: boolean;
  supportsTemplates: boolean;
  supportsLocation: boolean;
  maxMessageLength: number;
  maxMediaSize: number;
}
