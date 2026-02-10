import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { LinktorClient, createClient, LinktorClientConfig } from '../client';

describe('LinktorClient', () => {
  describe('constructor', () => {
    it('should create client with default config', () => {
      const client = new LinktorClient();

      expect(client).toBeInstanceOf(LinktorClient);
      expect(client.auth).toBeDefined();
      expect(client.conversations).toBeDefined();
      expect(client.contacts).toBeDefined();
      expect(client.channels).toBeDefined();
      expect(client.bots).toBeDefined();
      expect(client.ai).toBeDefined();
      expect(client.knowledgeBases).toBeDefined();
      expect(client.flows).toBeDefined();
      expect(client.analytics).toBeDefined();
      expect(client.webhooks).toBeDefined();
    });

    it('should create client with custom config', () => {
      const config: LinktorClientConfig = {
        baseUrl: 'https://custom.api.com',
        apiKey: 'test-api-key',
        timeout: 60000,
        maxRetries: 5,
        retryDelay: 2000,
      };

      const client = new LinktorClient(config);

      expect(client).toBeInstanceOf(LinktorClient);
    });

    it('should create client with access token', () => {
      const config: LinktorClientConfig = {
        accessToken: 'test-access-token',
      };

      const client = new LinktorClient(config);

      expect(client).toBeInstanceOf(LinktorClient);
    });

    it('should create client with custom headers', () => {
      const config: LinktorClientConfig = {
        apiKey: 'test-api-key',
        headers: {
          'X-Custom-Header': 'custom-value',
        },
      };

      const client = new LinktorClient(config);

      expect(client).toBeInstanceOf(LinktorClient);
    });
  });

  describe('createClient', () => {
    it('should create client using factory function', () => {
      const client = createClient();

      expect(client).toBeInstanceOf(LinktorClient);
    });

    it('should create client with config using factory function', () => {
      const client = createClient({
        apiKey: 'test-api-key',
        baseUrl: 'https://custom.api.com',
      });

      expect(client).toBeInstanceOf(LinktorClient);
    });
  });

  describe('setApiKey', () => {
    it('should update API key', () => {
      const client = new LinktorClient();

      expect(() => client.setApiKey('new-api-key')).not.toThrow();
    });
  });

  describe('setAccessToken', () => {
    it('should update access token', () => {
      const client = new LinktorClient();

      expect(() => client.setAccessToken('new-access-token')).not.toThrow();
    });
  });

  describe('close', () => {
    it('should close connections', () => {
      const client = new LinktorClient();

      expect(() => client.close()).not.toThrow();
    });
  });

  describe('webhooks', () => {
    it('should have webhook utilities', () => {
      const client = new LinktorClient();

      expect(client.webhooks).toBeDefined();
      expect(client.webhooks.verify).toBeTypeOf('function');
      expect(client.webhooks.verifyWithTimestamp).toBeTypeOf('function');
      expect(client.webhooks.computeSignature).toBeTypeOf('function');
      expect(client.webhooks.constructEvent).toBeTypeOf('function');
      expect(client.webhooks.createHandler).toBeTypeOf('function');
      expect(client.webhooks.isEventType).toBeTypeOf('function');
    });
  });

  describe('resources', () => {
    let client: LinktorClient;

    beforeEach(() => {
      client = new LinktorClient({
        apiKey: 'test-api-key',
        baseUrl: 'https://api.test.com',
      });
    });

    afterEach(() => {
      client.close();
    });

    it('should have auth resource', () => {
      expect(client.auth).toBeDefined();
    });

    it('should have conversations resource', () => {
      expect(client.conversations).toBeDefined();
    });

    it('should have contacts resource', () => {
      expect(client.contacts).toBeDefined();
    });

    it('should have channels resource', () => {
      expect(client.channels).toBeDefined();
    });

    it('should have bots resource', () => {
      expect(client.bots).toBeDefined();
    });

    it('should have ai resource', () => {
      expect(client.ai).toBeDefined();
    });

    it('should have knowledge bases resource', () => {
      expect(client.knowledgeBases).toBeDefined();
    });

    it('should have flows resource', () => {
      expect(client.flows).toBeDefined();
    });

    it('should have analytics resource', () => {
      expect(client.analytics).toBeDefined();
    });
  });
});
