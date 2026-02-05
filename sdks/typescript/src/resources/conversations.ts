/**
 * Conversations Resource
 */

import type { HttpClient } from '../utils/http';
import type { PaginatedResponse } from '../types/common';
import type {
  Conversation,
  Message,
  ListConversationsParams,
  ListMessagesParams,
  SendMessageInput,
  UpdateConversationInput,
} from '../types/conversation';

export class ConversationsResource {
  constructor(private http: HttpClient) {}

  /**
   * List conversations
   */
  async list(params?: ListConversationsParams): Promise<PaginatedResponse<Conversation>> {
    return this.http.get<PaginatedResponse<Conversation>>('/conversations', {
      params,
    });
  }

  /**
   * Get a single conversation
   */
  async get(id: string): Promise<Conversation> {
    return this.http.get<Conversation>(`/conversations/${id}`);
  }

  /**
   * Update a conversation
   */
  async update(id: string, data: UpdateConversationInput): Promise<Conversation> {
    return this.http.patch<Conversation>(`/conversations/${id}`, data);
  }

  /**
   * Assign conversation to an agent
   */
  async assign(id: string, agentId: string): Promise<Conversation> {
    return this.http.post<Conversation>(`/conversations/${id}/assign`, {
      agentId,
    });
  }

  /**
   * Unassign conversation
   */
  async unassign(id: string): Promise<Conversation> {
    return this.http.post<Conversation>(`/conversations/${id}/unassign`);
  }

  /**
   * Resolve conversation
   */
  async resolve(id: string): Promise<Conversation> {
    return this.update(id, { status: 'resolved' });
  }

  /**
   * Reopen conversation
   */
  async reopen(id: string): Promise<Conversation> {
    return this.update(id, { status: 'open' });
  }

  /**
   * Snooze conversation
   */
  async snooze(id: string, until?: string): Promise<Conversation> {
    return this.http.post<Conversation>(`/conversations/${id}/snooze`, {
      until,
    });
  }

  /**
   * Add tags to conversation
   */
  async addTags(id: string, tags: string[]): Promise<Conversation> {
    return this.http.post<Conversation>(`/conversations/${id}/tags`, {
      tags,
    });
  }

  /**
   * Remove tags from conversation
   */
  async removeTags(id: string, tags: string[]): Promise<Conversation> {
    return this.http.delete<Conversation>(`/conversations/${id}/tags`, {
      data: { tags },
    });
  }

  /**
   * List messages in a conversation
   */
  async listMessages(
    conversationId: string,
    params?: ListMessagesParams
  ): Promise<PaginatedResponse<Message>> {
    return this.http.get<PaginatedResponse<Message>>(
      `/conversations/${conversationId}/messages`,
      { params }
    );
  }

  /**
   * Get a single message
   */
  async getMessage(conversationId: string, messageId: string): Promise<Message> {
    return this.http.get<Message>(`/conversations/${conversationId}/messages/${messageId}`);
  }

  /**
   * Send a message to a conversation
   */
  async sendMessage(conversationId: string, data: SendMessageInput): Promise<Message> {
    return this.http.post<Message>(`/conversations/${conversationId}/messages`, data);
  }

  /**
   * Send a text message
   */
  async sendText(conversationId: string, text: string): Promise<Message> {
    return this.sendMessage(conversationId, { text });
  }

  /**
   * Send an image
   */
  async sendImage(
    conversationId: string,
    url: string,
    caption?: string
  ): Promise<Message> {
    return this.sendMessage(conversationId, {
      media: { type: 'image', url, caption },
    });
  }

  /**
   * Send a document
   */
  async sendDocument(
    conversationId: string,
    url: string,
    filename?: string
  ): Promise<Message> {
    return this.sendMessage(conversationId, {
      media: { type: 'document', url, filename },
    });
  }

  /**
   * Send a location
   */
  async sendLocation(
    conversationId: string,
    latitude: number,
    longitude: number,
    name?: string,
    address?: string
  ): Promise<Message> {
    return this.sendMessage(conversationId, {
      location: { latitude, longitude, name, address },
    });
  }

  /**
   * Mark messages as read
   */
  async markAsRead(conversationId: string, messageIds?: string[]): Promise<void> {
    await this.http.post<void>(`/conversations/${conversationId}/read`, {
      messageIds,
    });
  }

  /**
   * Get conversation count by status
   */
  async getCounts(): Promise<Record<string, number>> {
    return this.http.get<Record<string, number>>('/conversations/counts');
  }

  /**
   * Iterate through all conversations (handles pagination)
   */
  async *iterate(
    params?: Omit<ListConversationsParams, 'cursor'>
  ): AsyncGenerator<Conversation, void, unknown> {
    let cursor: string | undefined;
    let hasMore = true;

    while (hasMore) {
      const response = await this.list({ ...params, cursor });

      for (const conversation of response.data) {
        yield conversation;
      }

      hasMore = response.pagination.hasMore;
      cursor = response.pagination.nextCursor;
    }
  }
}
