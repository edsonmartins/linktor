/**
 * Contacts Resource
 */

import type { HttpClient } from '../utils/http';
import type { PaginatedResponse } from '../types/common';
import type {
  Contact,
  CreateContactInput,
  UpdateContactInput,
  ListContactsParams,
  MergeContactsInput,
} from '../types/contact';
import type { Conversation } from '../types/conversation';

export class ContactsResource {
  constructor(private http: HttpClient) {}

  /**
   * List contacts
   */
  async list(params?: ListContactsParams): Promise<PaginatedResponse<Contact>> {
    return this.http.get<PaginatedResponse<Contact>>('/contacts', { params });
  }

  /**
   * Get a single contact
   */
  async get(id: string): Promise<Contact> {
    return this.http.get<Contact>(`/contacts/${id}`);
  }

  /**
   * Create a new contact
   */
  async create(data: CreateContactInput): Promise<Contact> {
    return this.http.post<Contact>('/contacts', data);
  }

  /**
   * Update a contact
   */
  async update(id: string, data: UpdateContactInput): Promise<Contact> {
    return this.http.patch<Contact>(`/contacts/${id}`, data);
  }

  /**
   * Delete a contact
   */
  async delete(id: string): Promise<void> {
    await this.http.delete<void>(`/contacts/${id}`);
  }

  /**
   * Search contacts
   */
  async search(query: string, params?: Omit<ListContactsParams, 'search'>): Promise<PaginatedResponse<Contact>> {
    return this.list({ ...params, search: query });
  }

  /**
   * Find contact by email
   */
  async findByEmail(email: string): Promise<Contact | null> {
    const response = await this.list({ email, limit: 1 });
    return response.data[0] || null;
  }

  /**
   * Find contact by phone
   */
  async findByPhone(phone: string): Promise<Contact | null> {
    const response = await this.list({ phone, limit: 1 });
    return response.data[0] || null;
  }

  /**
   * Merge contacts
   */
  async merge(data: MergeContactsInput): Promise<Contact> {
    return this.http.post<Contact>('/contacts/merge', data);
  }

  /**
   * Get contact's conversations
   */
  async getConversations(contactId: string): Promise<PaginatedResponse<Conversation>> {
    return this.http.get<PaginatedResponse<Conversation>>(`/contacts/${contactId}/conversations`);
  }

  /**
   * Add tags to contact
   */
  async addTags(id: string, tags: string[]): Promise<Contact> {
    return this.http.post<Contact>(`/contacts/${id}/tags`, { tags });
  }

  /**
   * Remove tags from contact
   */
  async removeTags(id: string, tags: string[]): Promise<Contact> {
    return this.http.delete<Contact>(`/contacts/${id}/tags`, {
      data: { tags },
    });
  }

  /**
   * Update custom fields
   */
  async updateCustomFields(id: string, customFields: Record<string, unknown>): Promise<Contact> {
    return this.update(id, { customFields });
  }

  /**
   * Iterate through all contacts (handles pagination)
   */
  async *iterate(
    params?: Omit<ListContactsParams, 'cursor'>
  ): AsyncGenerator<Contact, void, unknown> {
    let cursor: string | undefined;
    let hasMore = true;

    while (hasMore) {
      const response = await this.list({ ...params, cursor });

      for (const contact of response.data) {
        yield contact;
      }

      hasMore = response.pagination.hasMore;
      cursor = response.pagination.nextCursor;
    }
  }
}
