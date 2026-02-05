package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.Map;

public class Channel {

    public enum ChannelType {
        @SerializedName("whatsapp") WHATSAPP,
        @SerializedName("whatsapp_unofficial") WHATSAPP_UNOFFICIAL,
        @SerializedName("telegram") TELEGRAM,
        @SerializedName("facebook") FACEBOOK,
        @SerializedName("instagram") INSTAGRAM,
        @SerializedName("webchat") WEBCHAT,
        @SerializedName("sms") SMS,
        @SerializedName("email") EMAIL,
        @SerializedName("rcs") RCS
    }

    public enum ChannelStatus {
        @SerializedName("connected") CONNECTED,
        @SerializedName("disconnected") DISCONNECTED,
        @SerializedName("connecting") CONNECTING,
        @SerializedName("error") ERROR
    }

    public static class ChannelModel {
        private String id;
        private String tenantId;
        private String name;
        private ChannelType type;
        private ChannelStatus status;
        private Map<String, Object> config;
        private Map<String, Object> metadata;
        private String errorMessage;
        private Instant connectedAt;
        private Instant lastActivityAt;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public ChannelType getType() { return type; }
        public void setType(ChannelType type) { this.type = type; }

        public ChannelStatus getStatus() { return status; }
        public void setStatus(ChannelStatus status) { this.status = status; }

        public Map<String, Object> getConfig() { return config; }
        public void setConfig(Map<String, Object> config) { this.config = config; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public String getErrorMessage() { return errorMessage; }
        public void setErrorMessage(String errorMessage) { this.errorMessage = errorMessage; }

        public Instant getConnectedAt() { return connectedAt; }
        public void setConnectedAt(Instant connectedAt) { this.connectedAt = connectedAt; }

        public Instant getLastActivityAt() { return lastActivityAt; }
        public void setLastActivityAt(Instant lastActivityAt) { this.lastActivityAt = lastActivityAt; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class CreateChannelInput {
        private String name;
        private ChannelType type;
        private Map<String, Object> config;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public ChannelType getType() { return type; }
        public void setType(ChannelType type) { this.type = type; }

        public Map<String, Object> getConfig() { return config; }
        public void setConfig(Map<String, Object> config) { this.config = config; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CreateChannelInput input = new CreateChannelInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder type(ChannelType type) { input.type = type; return this; }
            public Builder config(Map<String, Object> config) { input.config = config; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public CreateChannelInput build() { return input; }
        }
    }

    public static class UpdateChannelInput {
        private String name;
        private Map<String, Object> config;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public Map<String, Object> getConfig() { return config; }
        public void setConfig(Map<String, Object> config) { this.config = config; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final UpdateChannelInput input = new UpdateChannelInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder config(Map<String, Object> config) { input.config = config; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public UpdateChannelInput build() { return input; }
        }
    }

    public static class ListChannelsParams extends Common.PaginationParams {
        private ChannelType type;
        private ChannelStatus status;
        private String search;

        public ChannelType getType() { return type; }
        public void setType(ChannelType type) { this.type = type; }

        public ChannelStatus getStatus() { return status; }
        public void setStatus(ChannelStatus status) { this.status = status; }

        public String getSearch() { return search; }
        public void setSearch(String search) { this.search = search; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final ListChannelsParams params = new ListChannelsParams();

            public Builder type(ChannelType type) { params.type = type; return this; }
            public Builder status(ChannelStatus status) { params.status = status; return this; }
            public Builder search(String search) { params.search = search; return this; }
            public Builder limit(Integer limit) { params.setLimit(limit); return this; }
            public Builder page(Integer page) { params.setPage(page); return this; }
            public ListChannelsParams build() { return params; }
        }
    }

    public static class ChannelStatusResponse {
        private ChannelStatus status;
        private String errorMessage;
        private Instant connectedAt;
        private Instant lastActivityAt;

        public ChannelStatus getStatus() { return status; }
        public void setStatus(ChannelStatus status) { this.status = status; }

        public String getErrorMessage() { return errorMessage; }
        public void setErrorMessage(String errorMessage) { this.errorMessage = errorMessage; }

        public Instant getConnectedAt() { return connectedAt; }
        public void setConnectedAt(Instant connectedAt) { this.connectedAt = connectedAt; }

        public Instant getLastActivityAt() { return lastActivityAt; }
        public void setLastActivityAt(Instant lastActivityAt) { this.lastActivityAt = lastActivityAt; }
    }
}
