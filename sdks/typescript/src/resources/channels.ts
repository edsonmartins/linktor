/**
 * Channels Resource
 *
 * Every backend response is wrapped as `{ success, data }`; these methods
 * unwrap it and return the inner `data`.
 */

import type { HttpClient } from '../utils/http';
import type { ApiResponse } from '../types/common';
import type {
  Channel,
  ChannelCapabilities,
  ConnectResult,
  CreateChannelInput,
  UpdateChannelInput,
  ListChannelsParams,
} from '../types/channel';

export class ChannelsResource {
  constructor(private http: HttpClient) {}

  /**
   * List channels. The backend returns a plain array under `data` (no
   * pagination envelope for channels).
   */
  async list(params?: ListChannelsParams): Promise<Channel[]> {
    const res = await this.http.get<ApiResponse<Channel[]>>('/channels', { params });
    return res.data ?? [];
  }

  /**
   * Get a single channel
   */
  async get(id: string): Promise<Channel> {
    const res = await this.http.get<ApiResponse<Channel>>(`/channels/${id}`);
    return res.data as Channel;
  }

  /**
   * Create a new channel. Put secrets in `credentials` (write-only) and
   * non-secret settings in `config`.
   */
  async create(data: CreateChannelInput): Promise<Channel> {
    const res = await this.http.post<ApiResponse<Channel>>('/channels', data);
    return res.data as Channel;
  }

  /**
   * Update a channel (PUT). Omit `credentials` to keep the stored secrets.
   */
  async update(id: string, data: UpdateChannelInput): Promise<Channel> {
    const res = await this.http.put<ApiResponse<Channel>>(`/channels/${id}`, data);
    return res.data as Channel;
  }

  /**
   * Delete a channel
   */
  async delete(id: string): Promise<void> {
    await this.http.delete<void>(`/channels/${id}`);
  }

  /**
   * Connect a channel. For WhatsApp this starts (or refreshes) linking and
   * returns a ConnectResult carrying the QR payload (`qr_code`, `expires_in`) to
   * render; call `connect` again to poll for a fresh QR or the linked state.
   */
  async connect(id: string): Promise<ConnectResult> {
    const res = await this.http.post<ApiResponse<ConnectResult>>(`/channels/${id}/connect`);
    return res.data as ConnectResult;
  }

  /**
   * Request a WhatsApp pairing code for a phone number, as an alternative to QR
   * linking.
   */
  async requestPairCode(id: string, phoneNumber: string): Promise<ConnectResult> {
    const res = await this.http.post<ApiResponse<ConnectResult>>(`/channels/${id}/pair`, {
      phone_number: phoneNumber,
    });
    return res.data as ConnectResult;
  }

  /**
   * Disconnect a channel (deactivate it)
   */
  async disconnect(id: string): Promise<Channel> {
    const res = await this.http.post<ApiResponse<Channel>>(`/channels/${id}/disconnect`);
    return res.data as Channel;
  }

  /**
   * Get channel capabilities
   */
  async getCapabilities(id: string): Promise<ChannelCapabilities> {
    const res = await this.http.get<ApiResponse<ChannelCapabilities>>(
      `/channels/${id}/capabilities`
    );
    return res.data as ChannelCapabilities;
  }
}
