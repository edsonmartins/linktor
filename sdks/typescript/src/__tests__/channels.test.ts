import { describe, it, expect, vi } from 'vitest';
import { ChannelsResource } from '../resources/channels';
import type { HttpClient } from '../utils/http';

// Minimal HttpClient stub: each verb resolves to a preset envelope and records
// the call so we can assert on path/body.
function stubHttp(responses: Record<string, unknown>) {
  const calls: Array<{ verb: string; path: string; body?: unknown }> = [];
  const make = (verb: string) =>
    vi.fn(async (path: string, body?: unknown) => {
      calls.push({ verb, path, body });
      return responses[`${verb} ${path}`] ?? responses[path];
    });
  const http = {
    get: make('GET'),
    post: make('POST'),
    put: make('PUT'),
    patch: make('PATCH'),
    delete: make('DELETE'),
  } as unknown as HttpClient;
  return { http, calls };
}

describe('ChannelsResource', () => {
  it('connect unwraps the envelope and surfaces the QR payload', async () => {
    const { http } = stubHttp({
      'POST /channels/ch1/connect': {
        success: true,
        data: {
          channel: { id: 'ch1', name: 'wa', type: 'whatsapp', connection_status: 'connecting' },
          qr_code: 'QR-PAYLOAD-123',
          expires_in: 60,
        },
      },
    });
    const res = await new ChannelsResource(http).connect('ch1');
    expect(res.qr_code).toBe('QR-PAYLOAD-123');
    expect(res.expires_in).toBe(60);
    expect(res.channel.id).toBe('ch1');
    expect(res.channel.connection_status).toBe('connecting');
  });

  it('create sends credentials and returns the channel', async () => {
    const { http, calls } = stubHttp({
      'POST /channels': {
        success: true,
        data: { id: 'ch9', name: 'wa', type: 'whatsapp', connection_status: 'disconnected' },
      },
    });
    const ch = await new ChannelsResource(http).create({
      name: 'wa',
      type: 'whatsapp',
      credentials: { access_token: 'secret' },
    });
    expect(ch.id).toBe('ch9');
    const body = calls[0].body as { credentials?: Record<string, string> };
    expect(body.credentials?.access_token).toBe('secret');
  });

  it('update uses PUT', async () => {
    const { http, calls } = stubHttp({
      'PUT /channels/ch1': { success: true, data: { id: 'ch1', name: 'renamed' } },
    });
    await new ChannelsResource(http).update('ch1', { name: 'renamed' });
    expect(calls[0].verb).toBe('PUT');
    expect(calls[0].path).toBe('/channels/ch1');
  });

  it('requestPairCode posts phone_number and returns the pair code', async () => {
    const { http, calls } = stubHttp({
      'POST /channels/ch1/pair': { success: true, data: { pair_code: 'ABCD-1234' } },
    });
    const res = await new ChannelsResource(http).requestPairCode('ch1', '+5511999999999');
    expect(res.pair_code).toBe('ABCD-1234');
    expect((calls[0].body as { phone_number: string }).phone_number).toBe('+5511999999999');
  });

  it('list unwraps the plain data array', async () => {
    const { http } = stubHttp({
      'GET /channels': { success: true, data: [{ id: 'a' }, { id: 'b' }] },
    });
    const list = await new ChannelsResource(http).list();
    expect(list.map((c) => c.id)).toEqual(['a', 'b']);
  });
});
