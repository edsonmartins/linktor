package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class AI {

    public enum AgentStatus {
        @SerializedName("active") ACTIVE,
        @SerializedName("inactive") INACTIVE,
        @SerializedName("draft") DRAFT
    }

    public static class Agent {
        private String id;
        private String tenantId;
        private String name;
        private String description;
        private AgentStatus status;
        private String model;
        private String systemPrompt;
        private double temperature;
        private int maxTokens;
        private List<String> knowledgeBaseIds;
        private List<Tool> tools;
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

        public AgentStatus getStatus() { return status; }
        public void setStatus(AgentStatus status) { this.status = status; }

        public String getModel() { return model; }
        public void setModel(String model) { this.model = model; }

        public String getSystemPrompt() { return systemPrompt; }
        public void setSystemPrompt(String systemPrompt) { this.systemPrompt = systemPrompt; }

        public double getTemperature() { return temperature; }
        public void setTemperature(double temperature) { this.temperature = temperature; }

        public int getMaxTokens() { return maxTokens; }
        public void setMaxTokens(int maxTokens) { this.maxTokens = maxTokens; }

        public List<String> getKnowledgeBaseIds() { return knowledgeBaseIds; }
        public void setKnowledgeBaseIds(List<String> knowledgeBaseIds) { this.knowledgeBaseIds = knowledgeBaseIds; }

        public List<Tool> getTools() { return tools; }
        public void setTools(List<Tool> tools) { this.tools = tools; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class Tool {
        private String name;
        private String description;
        private Map<String, Object> parameters;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public Map<String, Object> getParameters() { return parameters; }
        public void setParameters(Map<String, Object> parameters) { this.parameters = parameters; }
    }

    public static class CreateAgentInput {
        private String name;
        private String description;
        private String model;
        private String systemPrompt;
        private Double temperature;
        private Integer maxTokens;
        private List<String> knowledgeBaseIds;
        private List<Tool> tools;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public String getModel() { return model; }
        public void setModel(String model) { this.model = model; }

        public String getSystemPrompt() { return systemPrompt; }
        public void setSystemPrompt(String systemPrompt) { this.systemPrompt = systemPrompt; }

        public Double getTemperature() { return temperature; }
        public void setTemperature(Double temperature) { this.temperature = temperature; }

        public Integer getMaxTokens() { return maxTokens; }
        public void setMaxTokens(Integer maxTokens) { this.maxTokens = maxTokens; }

        public List<String> getKnowledgeBaseIds() { return knowledgeBaseIds; }
        public void setKnowledgeBaseIds(List<String> knowledgeBaseIds) { this.knowledgeBaseIds = knowledgeBaseIds; }

        public List<Tool> getTools() { return tools; }
        public void setTools(List<Tool> tools) { this.tools = tools; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CreateAgentInput input = new CreateAgentInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder description(String description) { input.description = description; return this; }
            public Builder model(String model) { input.model = model; return this; }
            public Builder systemPrompt(String prompt) { input.systemPrompt = prompt; return this; }
            public Builder temperature(Double temperature) { input.temperature = temperature; return this; }
            public Builder maxTokens(Integer maxTokens) { input.maxTokens = maxTokens; return this; }
            public Builder knowledgeBaseIds(List<String> ids) { input.knowledgeBaseIds = ids; return this; }
            public Builder tools(List<Tool> tools) { input.tools = tools; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public CreateAgentInput build() { return input; }
        }
    }

    public static class ChatMessage {
        private String role;
        private String content;

        public ChatMessage() {}

        public ChatMessage(String role, String content) {
            this.role = role;
            this.content = content;
        }

        public String getRole() { return role; }
        public void setRole(String role) { this.role = role; }

        public String getContent() { return content; }
        public void setContent(String content) { this.content = content; }

        public static ChatMessage user(String content) {
            return new ChatMessage("user", content);
        }

        public static ChatMessage assistant(String content) {
            return new ChatMessage("assistant", content);
        }

        public static ChatMessage system(String content) {
            return new ChatMessage("system", content);
        }
    }

    public static class CompletionInput {
        private List<ChatMessage> messages;
        private String model;
        private Double temperature;
        private Integer maxTokens;
        private boolean stream;
        private List<Tool> tools;
        private Map<String, Object> metadata;

        public List<ChatMessage> getMessages() { return messages; }
        public void setMessages(List<ChatMessage> messages) { this.messages = messages; }

        public String getModel() { return model; }
        public void setModel(String model) { this.model = model; }

        public Double getTemperature() { return temperature; }
        public void setTemperature(Double temperature) { this.temperature = temperature; }

        public Integer getMaxTokens() { return maxTokens; }
        public void setMaxTokens(Integer maxTokens) { this.maxTokens = maxTokens; }

        public boolean isStream() { return stream; }
        public void setStream(boolean stream) { this.stream = stream; }

        public List<Tool> getTools() { return tools; }
        public void setTools(List<Tool> tools) { this.tools = tools; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CompletionInput input = new CompletionInput();

            public Builder messages(List<ChatMessage> messages) { input.messages = messages; return this; }
            public Builder model(String model) { input.model = model; return this; }
            public Builder temperature(Double temperature) { input.temperature = temperature; return this; }
            public Builder maxTokens(Integer maxTokens) { input.maxTokens = maxTokens; return this; }
            public Builder stream(boolean stream) { input.stream = stream; return this; }
            public Builder tools(List<Tool> tools) { input.tools = tools; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public CompletionInput build() { return input; }
        }
    }

    public static class CompletionResponse {
        private String id;
        private String object;
        private long created;
        private String model;
        private List<Choice> choices;
        private Usage usage;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getObject() { return object; }
        public void setObject(String object) { this.object = object; }

        public long getCreated() { return created; }
        public void setCreated(long created) { this.created = created; }

        public String getModel() { return model; }
        public void setModel(String model) { this.model = model; }

        public List<Choice> getChoices() { return choices; }
        public void setChoices(List<Choice> choices) { this.choices = choices; }

        public Usage getUsage() { return usage; }
        public void setUsage(Usage usage) { this.usage = usage; }

        public String getContent() {
            if (choices != null && !choices.isEmpty()) {
                ChatMessage msg = choices.get(0).getMessage();
                if (msg != null) {
                    return msg.getContent();
                }
            }
            return null;
        }
    }

    public static class Choice {
        private int index;
        private ChatMessage message;
        private String finishReason;

        public int getIndex() { return index; }
        public void setIndex(int index) { this.index = index; }

        public ChatMessage getMessage() { return message; }
        public void setMessage(ChatMessage message) { this.message = message; }

        public String getFinishReason() { return finishReason; }
        public void setFinishReason(String finishReason) { this.finishReason = finishReason; }
    }

    public static class Usage {
        private int promptTokens;
        private int completionTokens;
        private int totalTokens;

        public int getPromptTokens() { return promptTokens; }
        public void setPromptTokens(int promptTokens) { this.promptTokens = promptTokens; }

        public int getCompletionTokens() { return completionTokens; }
        public void setCompletionTokens(int completionTokens) { this.completionTokens = completionTokens; }

        public int getTotalTokens() { return totalTokens; }
        public void setTotalTokens(int totalTokens) { this.totalTokens = totalTokens; }
    }

    public static class EmbeddingInput {
        private String input;
        private List<String> inputs;
        private String model;

        public String getInput() { return input; }
        public void setInput(String input) { this.input = input; }

        public List<String> getInputs() { return inputs; }
        public void setInputs(List<String> inputs) { this.inputs = inputs; }

        public String getModel() { return model; }
        public void setModel(String model) { this.model = model; }

        public static EmbeddingInput single(String text) {
            EmbeddingInput input = new EmbeddingInput();
            input.input = text;
            return input;
        }

        public static EmbeddingInput batch(List<String> texts) {
            EmbeddingInput input = new EmbeddingInput();
            input.inputs = texts;
            return input;
        }
    }

    public static class EmbeddingResponse {
        private String object;
        private List<EmbeddingData> data;
        private String model;
        private Usage usage;

        public String getObject() { return object; }
        public void setObject(String object) { this.object = object; }

        public List<EmbeddingData> getData() { return data; }
        public void setData(List<EmbeddingData> data) { this.data = data; }

        public String getModel() { return model; }
        public void setModel(String model) { this.model = model; }

        public Usage getUsage() { return usage; }
        public void setUsage(Usage usage) { this.usage = usage; }

        public double[] getEmbedding() {
            if (data != null && !data.isEmpty()) {
                return data.get(0).getEmbedding();
            }
            return null;
        }
    }

    public static class EmbeddingData {
        private String object;
        private int index;
        private double[] embedding;

        public String getObject() { return object; }
        public void setObject(String object) { this.object = object; }

        public int getIndex() { return index; }
        public void setIndex(int index) { this.index = index; }

        public double[] getEmbedding() { return embedding; }
        public void setEmbedding(double[] embedding) { this.embedding = embedding; }
    }
}
