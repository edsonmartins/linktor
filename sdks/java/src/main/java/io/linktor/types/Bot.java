package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class Bot {

    public enum BotStatus {
        @SerializedName("active") ACTIVE,
        @SerializedName("inactive") INACTIVE,
        @SerializedName("draft") DRAFT
    }

    public enum BotType {
        @SerializedName("flow") FLOW,
        @SerializedName("ai") AI,
        @SerializedName("hybrid") HYBRID
    }

    public static class BotModel {
        private String id;
        private String tenantId;
        private String name;
        private String description;
        private BotStatus status;
        private BotType type;
        private Map<String, Object> config;
        private List<String> channelIds;
        private String flowId;
        private String agentId;
        private List<String> knowledgeBaseIds;
        private Map<String, Object> metadata;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public BotStatus getStatus() { return status; }
        public void setStatus(BotStatus status) { this.status = status; }

        public BotType getType() { return type; }
        public void setType(BotType type) { this.type = type; }

        public Map<String, Object> getConfig() { return config; }
        public void setConfig(Map<String, Object> config) { this.config = config; }

        public List<String> getChannelIds() { return channelIds; }
        public void setChannelIds(List<String> channelIds) { this.channelIds = channelIds; }

        public String getFlowId() { return flowId; }
        public void setFlowId(String flowId) { this.flowId = flowId; }

        public String getAgentId() { return agentId; }
        public void setAgentId(String agentId) { this.agentId = agentId; }

        public List<String> getKnowledgeBaseIds() { return knowledgeBaseIds; }
        public void setKnowledgeBaseIds(List<String> knowledgeBaseIds) { this.knowledgeBaseIds = knowledgeBaseIds; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class CreateBotInput {
        private String name;
        private String description;
        private BotType type;
        private Map<String, Object> config;
        private List<String> channelIds;
        private String flowId;
        private String agentId;
        private List<String> knowledgeBaseIds;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public BotType getType() { return type; }
        public void setType(BotType type) { this.type = type; }

        public Map<String, Object> getConfig() { return config; }
        public void setConfig(Map<String, Object> config) { this.config = config; }

        public List<String> getChannelIds() { return channelIds; }
        public void setChannelIds(List<String> channelIds) { this.channelIds = channelIds; }

        public String getFlowId() { return flowId; }
        public void setFlowId(String flowId) { this.flowId = flowId; }

        public String getAgentId() { return agentId; }
        public void setAgentId(String agentId) { this.agentId = agentId; }

        public List<String> getKnowledgeBaseIds() { return knowledgeBaseIds; }
        public void setKnowledgeBaseIds(List<String> knowledgeBaseIds) { this.knowledgeBaseIds = knowledgeBaseIds; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CreateBotInput input = new CreateBotInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder description(String description) { input.description = description; return this; }
            public Builder type(BotType type) { input.type = type; return this; }
            public Builder config(Map<String, Object> config) { input.config = config; return this; }
            public Builder channelIds(List<String> channelIds) { input.channelIds = channelIds; return this; }
            public Builder flowId(String flowId) { input.flowId = flowId; return this; }
            public Builder agentId(String agentId) { input.agentId = agentId; return this; }
            public Builder knowledgeBaseIds(List<String> ids) { input.knowledgeBaseIds = ids; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public CreateBotInput build() { return input; }
        }
    }

    public static class UpdateBotInput {
        private String name;
        private String description;
        private BotStatus status;
        private Map<String, Object> config;
        private List<String> channelIds;
        private String flowId;
        private String agentId;
        private List<String> knowledgeBaseIds;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public BotStatus getStatus() { return status; }
        public void setStatus(BotStatus status) { this.status = status; }

        public Map<String, Object> getConfig() { return config; }
        public void setConfig(Map<String, Object> config) { this.config = config; }

        public List<String> getChannelIds() { return channelIds; }
        public void setChannelIds(List<String> channelIds) { this.channelIds = channelIds; }

        public String getFlowId() { return flowId; }
        public void setFlowId(String flowId) { this.flowId = flowId; }

        public String getAgentId() { return agentId; }
        public void setAgentId(String agentId) { this.agentId = agentId; }

        public List<String> getKnowledgeBaseIds() { return knowledgeBaseIds; }
        public void setKnowledgeBaseIds(List<String> knowledgeBaseIds) { this.knowledgeBaseIds = knowledgeBaseIds; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final UpdateBotInput input = new UpdateBotInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder description(String description) { input.description = description; return this; }
            public Builder status(BotStatus status) { input.status = status; return this; }
            public Builder config(Map<String, Object> config) { input.config = config; return this; }
            public Builder channelIds(List<String> channelIds) { input.channelIds = channelIds; return this; }
            public Builder flowId(String flowId) { input.flowId = flowId; return this; }
            public Builder agentId(String agentId) { input.agentId = agentId; return this; }
            public Builder knowledgeBaseIds(List<String> ids) { input.knowledgeBaseIds = ids; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public UpdateBotInput build() { return input; }
        }
    }

    public static class ListBotsParams extends Common.PaginationParams {
        private BotStatus status;
        private BotType type;
        private String channelId;
        private String search;

        public BotStatus getStatus() { return status; }
        public void setStatus(BotStatus status) { this.status = status; }

        public BotType getType() { return type; }
        public void setType(BotType type) { this.type = type; }

        public String getChannelId() { return channelId; }
        public void setChannelId(String channelId) { this.channelId = channelId; }

        public String getSearch() { return search; }
        public void setSearch(String search) { this.search = search; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final ListBotsParams params = new ListBotsParams();

            public Builder status(BotStatus status) { params.status = status; return this; }
            public Builder type(BotType type) { params.type = type; return this; }
            public Builder channelId(String channelId) { params.channelId = channelId; return this; }
            public Builder search(String search) { params.search = search; return this; }
            public Builder limit(Integer limit) { params.setLimit(limit); return this; }
            public Builder page(Integer page) { params.setPage(page); return this; }
            public ListBotsParams build() { return params; }
        }
    }
}
