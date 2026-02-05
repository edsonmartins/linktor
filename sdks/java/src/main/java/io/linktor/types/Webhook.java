package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.Map;

public class Webhook {

    public static final String SIGNATURE_HEADER = "X-Linktor-Signature";
    public static final String TIMESTAMP_HEADER = "X-Linktor-Timestamp";
    public static final int DEFAULT_TOLERANCE_SECONDS = 300;

    public enum EventType {
        @SerializedName("message.received") MESSAGE_RECEIVED,
        @SerializedName("message.sent") MESSAGE_SENT,
        @SerializedName("message.delivered") MESSAGE_DELIVERED,
        @SerializedName("message.read") MESSAGE_READ,
        @SerializedName("message.failed") MESSAGE_FAILED,

        @SerializedName("conversation.created") CONVERSATION_CREATED,
        @SerializedName("conversation.updated") CONVERSATION_UPDATED,
        @SerializedName("conversation.resolved") CONVERSATION_RESOLVED,
        @SerializedName("conversation.assigned") CONVERSATION_ASSIGNED,

        @SerializedName("contact.created") CONTACT_CREATED,
        @SerializedName("contact.updated") CONTACT_UPDATED,
        @SerializedName("contact.deleted") CONTACT_DELETED,

        @SerializedName("channel.connected") CHANNEL_CONNECTED,
        @SerializedName("channel.disconnected") CHANNEL_DISCONNECTED,
        @SerializedName("channel.error") CHANNEL_ERROR,

        @SerializedName("bot.started") BOT_STARTED,
        @SerializedName("bot.stopped") BOT_STOPPED,

        @SerializedName("flow.started") FLOW_STARTED,
        @SerializedName("flow.completed") FLOW_COMPLETED,
        @SerializedName("flow.failed") FLOW_FAILED
    }

    public static class WebhookEvent {
        private String id;
        private String type;
        private Instant timestamp;
        private String tenantId;
        private Map<String, Object> data;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public Instant getTimestamp() { return timestamp; }
        public void setTimestamp(Instant timestamp) { this.timestamp = timestamp; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public Map<String, Object> getData() { return data; }
        public void setData(Map<String, Object> data) { this.data = data; }

        public EventType getEventType() {
            if (type == null) return null;
            try {
                return EventType.valueOf(type.toUpperCase().replace(".", "_"));
            } catch (IllegalArgumentException e) {
                return null;
            }
        }
    }

    public static class WebhookConfig {
        private String url;
        private String secret;
        private String[] events;
        private boolean enabled;
        private Map<String, String> headers;

        public String getUrl() { return url; }
        public void setUrl(String url) { this.url = url; }

        public String getSecret() { return secret; }
        public void setSecret(String secret) { this.secret = secret; }

        public String[] getEvents() { return events; }
        public void setEvents(String[] events) { this.events = events; }

        public boolean isEnabled() { return enabled; }
        public void setEnabled(boolean enabled) { this.enabled = enabled; }

        public Map<String, String> getHeaders() { return headers; }
        public void setHeaders(Map<String, String> headers) { this.headers = headers; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final WebhookConfig config = new WebhookConfig();

            public Builder url(String url) { config.url = url; return this; }
            public Builder secret(String secret) { config.secret = secret; return this; }
            public Builder events(String... events) { config.events = events; return this; }
            public Builder enabled(boolean enabled) { config.enabled = enabled; return this; }
            public Builder headers(Map<String, String> headers) { config.headers = headers; return this; }
            public WebhookConfig build() { return config; }
        }
    }
}
