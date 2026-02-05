package io.linktor;

import io.linktor.resources.*;
import io.linktor.utils.HttpClient;
import io.linktor.utils.WebhookVerifier;
import io.linktor.websocket.LinktorWebSocket;

public class LinktorClient {
    private final HttpClient http;
    private final LinktorWebSocket webSocket;

    // Resources
    public final AuthResource auth;
    public final ConversationsResource conversations;
    public final ContactsResource contacts;
    public final ChannelsResource channels;
    public final BotsResource bots;
    public final AIResource ai;
    public final KnowledgeBasesResource knowledgeBases;
    public final FlowsResource flows;
    public final AnalyticsResource analytics;

    private LinktorClient(Builder builder) {
        this.http = new HttpClient(
                builder.baseUrl,
                builder.apiKey,
                builder.accessToken,
                builder.timeoutSeconds,
                builder.maxRetries
        );

        this.webSocket = new LinktorWebSocket(builder.baseUrl, builder.apiKey, builder.accessToken);

        // Initialize resources
        this.auth = new AuthResource(http);
        this.conversations = new ConversationsResource(http);
        this.contacts = new ContactsResource(http);
        this.channels = new ChannelsResource(http);
        this.bots = new BotsResource(http);
        this.ai = new AIResource(http);
        this.knowledgeBases = new KnowledgeBasesResource(http);
        this.flows = new FlowsResource(http);
        this.analytics = new AnalyticsResource(http);
    }

    /**
     * Get the WebSocket client
     */
    public LinktorWebSocket webSocket() {
        return webSocket;
    }

    /**
     * Get the HTTP client (for advanced usage)
     */
    public HttpClient getHttpClient() {
        return http;
    }

    /**
     * Create a new builder
     */
    public static Builder builder() {
        return new Builder();
    }

    // Webhook utilities (static methods)

    /**
     * Verify webhook signature
     */
    public static boolean verifyWebhook(byte[] payload, String signature, String secret) {
        return WebhookVerifier.verifySignature(payload, signature, secret);
    }

    /**
     * Verify webhook signature
     */
    public static boolean verifyWebhook(String payload, String signature, String secret) {
        return WebhookVerifier.verifySignature(payload, signature, secret);
    }

    /**
     * Compute webhook signature
     */
    public static String computeSignature(byte[] payload, String secret) {
        return WebhookVerifier.computeSignature(payload, secret);
    }

    /**
     * Compute webhook signature
     */
    public static String computeSignature(String payload, String secret) {
        return WebhookVerifier.computeSignature(payload, secret);
    }

    public static class Builder {
        private String baseUrl = "https://api.linktor.io";
        private String apiKey;
        private String accessToken;
        private int timeoutSeconds = 30;
        private int maxRetries = 3;

        public Builder baseUrl(String baseUrl) {
            this.baseUrl = baseUrl;
            return this;
        }

        public Builder apiKey(String apiKey) {
            this.apiKey = apiKey;
            return this;
        }

        public Builder accessToken(String accessToken) {
            this.accessToken = accessToken;
            return this;
        }

        public Builder timeout(int seconds) {
            this.timeoutSeconds = seconds;
            return this;
        }

        public Builder maxRetries(int retries) {
            this.maxRetries = retries;
            return this;
        }

        public LinktorClient build() {
            if (baseUrl == null || baseUrl.isEmpty()) {
                throw new IllegalArgumentException("baseUrl is required");
            }
            return new LinktorClient(this);
        }
    }
}
