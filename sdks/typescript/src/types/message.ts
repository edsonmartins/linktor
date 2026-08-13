/**
 * Direct send types (POST /messages/send)
 */

/**
 * Body of a direct send: a channel + recipient send that does not require the
 * caller to know a conversation. Linktor resolves (or creates) the recipient's
 * identity, contact and conversation inside the tenant before persisting and
 * queueing the message.
 */
export interface DirectSendInput {
  channel_id: string;
  /** Recipient address: phone number, email, or channel-specific id. */
  to: string;
  /** Currently restricted to 'text' (the default when omitted). */
  content_type?: 'text';
  text: string;
  /**
   * Carried end to end: persisted on the message and delivered to the channel
   * adapter. Two keys change behaviour rather than merely riding along:
   *
   * - `idempotency_key` — unique per tenant. Repeating a call with the same key
   *   returns the original message instead of sending a second one.
   * - `subject` — used as the subject line by the email channel.
   *
   * Linktor-internal fields (reply threading, reaction targets, campaign
   * bookkeeping, routing ids) are rejected, not silently stripped.
   */
  metadata?: Record<string, string>;
}

/**
 * Answer to a direct send. The message is queued, not yet delivered: `status`
 * is 'queued' and the delivery outcome arrives later on the channel's webhook
 * as `message.sent` / `message.failed`.
 */
export interface DirectSendResult {
  id: string;
  conversation_id: string;
  channel_id: string;
  status: string;
}
