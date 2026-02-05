/**
 * Linktor SDK for TypeScript/JavaScript
 *
 * @packageDocumentation
 */

// Main client
export { LinktorClient, LinktorClientConfig, createClient } from './client';

// Types
export * from './types';

// Resources
export { AuthResource } from './resources/auth';
export { ConversationsResource } from './resources/conversations';
export { ContactsResource } from './resources/contacts';
export { ChannelsResource } from './resources/channels';
export { BotsResource } from './resources/bots';
export { AIResource } from './resources/ai';
export { KnowledgeBasesResource } from './resources/knowledge-bases';
export { FlowsResource } from './resources/flows';
export { AnalyticsResource, DateRange } from './resources/analytics';

// WebSocket
export {
  LinktorWebSocket,
  WebSocketConfig,
  WebSocketEvent,
  MessageEvent,
  MessageStatusEvent,
  ConversationUpdateEvent,
  TypingEvent,
  WebSocketData,
} from './websocket/client';

// Webhook utilities
export {
  verifyWebhookSignature,
  verifyWebhook,
  computeSignature,
  constructEvent,
  createWebhookHandler,
  isEventType,
  WebhookVerificationOptions,
  WebhookHeaders,
} from './utils/webhook';

// Errors
export {
  LinktorError,
  AuthenticationError,
  AuthorizationError,
  NotFoundError,
  ValidationError,
  RateLimitError,
  ConflictError,
  ServerError,
  NetworkError,
  TimeoutError,
  WebSocketError,
  createErrorFromResponse,
  isRetryableError,
} from './utils/errors';

// HTTP client (for advanced usage)
export { HttpClient, HttpClientConfig, RequestConfig } from './utils/http';
