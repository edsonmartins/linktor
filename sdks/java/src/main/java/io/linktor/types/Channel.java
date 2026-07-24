package io.linktor.types;

import com.google.gson.JsonElement;
import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.Map;

public class Channel {

    /**
     * Channel type. Matches the backend wire enum exactly.
     */
    public enum ChannelType {
        @SerializedName("webchat") WEBCHAT,
        @SerializedName("whatsapp") WHATSAPP,
        @SerializedName("whatsapp_official") WHATSAPP_OFFICIAL,
        @SerializedName("whatsapp_unofficial") WHATSAPP_UNOFFICIAL,
        @SerializedName("telegram") TELEGRAM,
        @SerializedName("sms") SMS,
        @SerializedName("rcs") RCS,
        @SerializedName("instagram") INSTAGRAM,
        @SerializedName("facebook") FACEBOOK,
        @SerializedName("email") EMAIL,
        @SerializedName("voice") VOICE,
        @SerializedName("teams") TEAMS,
        @SerializedName("slack") SLACK,
        @SerializedName("mattermost") MATTERMOST,
        @SerializedName("direto") DIRETO
    }

    /**
     * Live connection status of the channel.
     */
    public enum ConnectionStatus {
        @SerializedName("disconnected") DISCONNECTED,
        @SerializedName("connecting") CONNECTING,
        @SerializedName("connected") CONNECTED,
        @SerializedName("error") ERROR
    }

    /**
     * WhatsApp coexistence status (only relevant to coexistence channels).
     */
    public enum CoexistenceStatus {
        @SerializedName("inactive") INACTIVE,
        @SerializedName("pending") PENDING,
        @SerializedName("active") ACTIVE,
        @SerializedName("warning") WARNING,
        @SerializedName("disconnected") DISCONNECTED
    }

    /**
     * Channel as returned by the backend. All fields are snake_case on the wire.
     * Credentials are write-only and are NEVER present on a response; secret
     * values inside {@link #config} are redacted as "__redacted__".
     */
    public static class ChannelModel {
        @SerializedName("id")
        private String id;

        @SerializedName("tenant_id")
        private String tenantId;

        @SerializedName("type")
        private ChannelType type;

        @SerializedName("name")
        private String name;

        @SerializedName("identifier")
        private String identifier;

        @SerializedName("enabled")
        private boolean enabled;

        @SerializedName("connection_status")
        private ConnectionStatus connectionStatus;

        @SerializedName("config")
        private Map<String, String> config;

        @SerializedName("webhook_url")
        private String webhookUrl;

        @SerializedName("created_at")
        private Instant createdAt;

        @SerializedName("updated_at")
        private Instant updatedAt;

        @SerializedName("is_coexistence")
        private Boolean isCoexistence;

        @SerializedName("waba_id")
        private String wabaId;

        @SerializedName("last_echo_at")
        private Instant lastEchoAt;

        @SerializedName("coexistence_status")
        private CoexistenceStatus coexistenceStatus;

        @SerializedName("message_template_namespace")
        private String messageTemplateNamespace;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public ChannelType getType() { return type; }
        public void setType(ChannelType type) { this.type = type; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getIdentifier() { return identifier; }
        public void setIdentifier(String identifier) { this.identifier = identifier; }

        public boolean isEnabled() { return enabled; }
        public void setEnabled(boolean enabled) { this.enabled = enabled; }

        public ConnectionStatus getConnectionStatus() { return connectionStatus; }
        public void setConnectionStatus(ConnectionStatus connectionStatus) { this.connectionStatus = connectionStatus; }

        public Map<String, String> getConfig() { return config; }
        public void setConfig(Map<String, String> config) { this.config = config; }

        public String getWebhookUrl() { return webhookUrl; }
        public void setWebhookUrl(String webhookUrl) { this.webhookUrl = webhookUrl; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }

        public Boolean getIsCoexistence() { return isCoexistence; }
        public void setIsCoexistence(Boolean isCoexistence) { this.isCoexistence = isCoexistence; }

        public String getWabaId() { return wabaId; }
        public void setWabaId(String wabaId) { this.wabaId = wabaId; }

        public Instant getLastEchoAt() { return lastEchoAt; }
        public void setLastEchoAt(Instant lastEchoAt) { this.lastEchoAt = lastEchoAt; }

        public CoexistenceStatus getCoexistenceStatus() { return coexistenceStatus; }
        public void setCoexistenceStatus(CoexistenceStatus coexistenceStatus) { this.coexistenceStatus = coexistenceStatus; }

        public String getMessageTemplateNamespace() { return messageTemplateNamespace; }
        public void setMessageTemplateNamespace(String messageTemplateNamespace) { this.messageTemplateNamespace = messageTemplateNamespace; }
    }

    /**
     * Request body for creating a channel (snake_case on the wire).
     * {@code credentials} carries write-only secrets (e.g. access_token, bot_token).
     */
    public static class CreateChannelInput {
        @SerializedName("type")
        private ChannelType type;

        @SerializedName("name")
        private String name;

        @SerializedName("identifier")
        private String identifier;

        @SerializedName("config")
        private Map<String, String> config;

        @SerializedName("credentials")
        private Map<String, String> credentials;

        @SerializedName("webhook_url")
        private String webhookUrl;

        public ChannelType getType() { return type; }
        public void setType(ChannelType type) { this.type = type; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getIdentifier() { return identifier; }
        public void setIdentifier(String identifier) { this.identifier = identifier; }

        public Map<String, String> getConfig() { return config; }
        public void setConfig(Map<String, String> config) { this.config = config; }

        public Map<String, String> getCredentials() { return credentials; }
        public void setCredentials(Map<String, String> credentials) { this.credentials = credentials; }

        public String getWebhookUrl() { return webhookUrl; }
        public void setWebhookUrl(String webhookUrl) { this.webhookUrl = webhookUrl; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CreateChannelInput input = new CreateChannelInput();

            public Builder type(ChannelType type) { input.type = type; return this; }
            public Builder name(String name) { input.name = name; return this; }
            public Builder identifier(String identifier) { input.identifier = identifier; return this; }
            public Builder config(Map<String, String> config) { input.config = config; return this; }
            public Builder credentials(Map<String, String> credentials) { input.credentials = credentials; return this; }
            public Builder webhookUrl(String webhookUrl) { input.webhookUrl = webhookUrl; return this; }
            public CreateChannelInput build() { return input; }
        }
    }

    /**
     * Request body for updating a channel. Reuses the create shape (PUT semantics).
     * A sensitive value equal to "__redacted__" or empty means "keep the stored secret".
     */
    public static class UpdateChannelInput {
        @SerializedName("type")
        private ChannelType type;

        @SerializedName("name")
        private String name;

        @SerializedName("identifier")
        private String identifier;

        @SerializedName("config")
        private Map<String, String> config;

        @SerializedName("credentials")
        private Map<String, String> credentials;

        @SerializedName("webhook_url")
        private String webhookUrl;

        public ChannelType getType() { return type; }
        public void setType(ChannelType type) { this.type = type; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getIdentifier() { return identifier; }
        public void setIdentifier(String identifier) { this.identifier = identifier; }

        public Map<String, String> getConfig() { return config; }
        public void setConfig(Map<String, String> config) { this.config = config; }

        public Map<String, String> getCredentials() { return credentials; }
        public void setCredentials(Map<String, String> credentials) { this.credentials = credentials; }

        public String getWebhookUrl() { return webhookUrl; }
        public void setWebhookUrl(String webhookUrl) { this.webhookUrl = webhookUrl; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final UpdateChannelInput input = new UpdateChannelInput();

            public Builder type(ChannelType type) { input.type = type; return this; }
            public Builder name(String name) { input.name = name; return this; }
            public Builder identifier(String identifier) { input.identifier = identifier; return this; }
            public Builder config(Map<String, String> config) { input.config = config; return this; }
            public Builder credentials(Map<String, String> credentials) { input.credentials = credentials; return this; }
            public Builder webhookUrl(String webhookUrl) { input.webhookUrl = webhookUrl; return this; }
            public UpdateChannelInput build() { return input; }
        }
    }

    /**
     * Request body for {@code POST /channels/{id}/pair}.
     */
    public static class PairChannelInput {
        @SerializedName("phone_number")
        private String phoneNumber;

        public PairChannelInput() {}

        public PairChannelInput(String phoneNumber) { this.phoneNumber = phoneNumber; }

        public String getPhoneNumber() { return phoneNumber; }
        public void setPhoneNumber(String phoneNumber) { this.phoneNumber = phoneNumber; }
    }

    /**
     * Result of a connect/pair operation, returned inside the response envelope's data.
     */
    public static class ConnectResult {
        @SerializedName("channel")
        private ChannelModel channel;

        @SerializedName("qr_code")
        private String qrCode;

        @SerializedName("expires_in")
        private Integer expiresIn;

        @SerializedName("pair_code")
        private String pairCode;

        @SerializedName("passkey_required")
        private Boolean passkeyRequired;

        @SerializedName("passkey_challenge")
        private JsonElement passkeyChallenge;

        public ChannelModel getChannel() { return channel; }
        public void setChannel(ChannelModel channel) { this.channel = channel; }

        public String getQrCode() { return qrCode; }
        public void setQrCode(String qrCode) { this.qrCode = qrCode; }

        public Integer getExpiresIn() { return expiresIn; }
        public void setExpiresIn(Integer expiresIn) { this.expiresIn = expiresIn; }

        public String getPairCode() { return pairCode; }
        public void setPairCode(String pairCode) { this.pairCode = pairCode; }

        public Boolean getPasskeyRequired() { return passkeyRequired; }
        public void setPasskeyRequired(Boolean passkeyRequired) { this.passkeyRequired = passkeyRequired; }

        public JsonElement getPasskeyChallenge() { return passkeyChallenge; }
        public void setPasskeyChallenge(JsonElement passkeyChallenge) { this.passkeyChallenge = passkeyChallenge; }
    }
}
