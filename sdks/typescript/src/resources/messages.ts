/**
 * Messages Resource
 *
 * Sending that is not scoped to a conversation the caller already knows.
 */

import type { HttpClient } from '../utils/http';
import type { DirectSendInput, DirectSendResult } from '../types/message';

export class MessagesResource {
  constructor(private http: HttpClient) {}

  /**
   * Send a message on a channel to a recipient, letting Linktor resolve (or
   * create) the identity, contact and conversation. Requires the
   * `messages:send` scope. Returns as soon as the message is queued.
   */
  async sendDirect(data: DirectSendInput): Promise<DirectSendResult> {
    return this.http.post<DirectSendResult>('/messages/send', data);
  }

  /**
   * `sendDirect` for a plain text message.
   */
  async sendText(
    channelId: string,
    to: string,
    text: string,
    metadata?: Record<string, string>
  ): Promise<DirectSendResult> {
    return this.sendDirect({
      channel_id: channelId,
      to,
      content_type: 'text',
      text,
      metadata,
    });
  }
}
