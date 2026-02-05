package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class Flow {

    public enum FlowStatus {
        @SerializedName("active") ACTIVE,
        @SerializedName("inactive") INACTIVE,
        @SerializedName("draft") DRAFT
    }

    public enum FlowExecutionStatus {
        @SerializedName("running") RUNNING,
        @SerializedName("waiting") WAITING,
        @SerializedName("completed") COMPLETED,
        @SerializedName("failed") FAILED,
        @SerializedName("cancelled") CANCELLED
    }

    public static class FlowModel {
        private String id;
        private String tenantId;
        private String name;
        private String description;
        private FlowStatus status;
        private int version;
        private List<FlowNode> nodes;
        private List<FlowEdge> edges;
        private List<FlowVariable> variables;
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

        public FlowStatus getStatus() { return status; }
        public void setStatus(FlowStatus status) { this.status = status; }

        public int getVersion() { return version; }
        public void setVersion(int version) { this.version = version; }

        public List<FlowNode> getNodes() { return nodes; }
        public void setNodes(List<FlowNode> nodes) { this.nodes = nodes; }

        public List<FlowEdge> getEdges() { return edges; }
        public void setEdges(List<FlowEdge> edges) { this.edges = edges; }

        public List<FlowVariable> getVariables() { return variables; }
        public void setVariables(List<FlowVariable> variables) { this.variables = variables; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class FlowNode {
        private String id;
        private String type;
        private Map<String, Double> position;
        private Map<String, Object> data;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public Map<String, Double> getPosition() { return position; }
        public void setPosition(Map<String, Double> position) { this.position = position; }

        public Map<String, Object> getData() { return data; }
        public void setData(Map<String, Object> data) { this.data = data; }
    }

    public static class FlowEdge {
        private String id;
        private String source;
        private String target;
        private String sourceHandle;
        private String targetHandle;
        private String label;
        private String condition;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getSource() { return source; }
        public void setSource(String source) { this.source = source; }

        public String getTarget() { return target; }
        public void setTarget(String target) { this.target = target; }

        public String getSourceHandle() { return sourceHandle; }
        public void setSourceHandle(String sourceHandle) { this.sourceHandle = sourceHandle; }

        public String getTargetHandle() { return targetHandle; }
        public void setTargetHandle(String targetHandle) { this.targetHandle = targetHandle; }

        public String getLabel() { return label; }
        public void setLabel(String label) { this.label = label; }

        public String getCondition() { return condition; }
        public void setCondition(String condition) { this.condition = condition; }
    }

    public static class FlowVariable {
        private String name;
        private String type;
        private Object defaultValue;
        private String description;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public Object getDefaultValue() { return defaultValue; }
        public void setDefaultValue(Object defaultValue) { this.defaultValue = defaultValue; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
    }

    public static class FlowExecution {
        private String id;
        private String flowId;
        private String conversationId;
        private FlowExecutionStatus status;
        private String currentNodeId;
        private Map<String, Object> variables;
        private List<FlowExecutionStep> history;
        private Instant startedAt;
        private Instant completedAt;
        private String error;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getFlowId() { return flowId; }
        public void setFlowId(String flowId) { this.flowId = flowId; }

        public String getConversationId() { return conversationId; }
        public void setConversationId(String conversationId) { this.conversationId = conversationId; }

        public FlowExecutionStatus getStatus() { return status; }
        public void setStatus(FlowExecutionStatus status) { this.status = status; }

        public String getCurrentNodeId() { return currentNodeId; }
        public void setCurrentNodeId(String currentNodeId) { this.currentNodeId = currentNodeId; }

        public Map<String, Object> getVariables() { return variables; }
        public void setVariables(Map<String, Object> variables) { this.variables = variables; }

        public List<FlowExecutionStep> getHistory() { return history; }
        public void setHistory(List<FlowExecutionStep> history) { this.history = history; }

        public Instant getStartedAt() { return startedAt; }
        public void setStartedAt(Instant startedAt) { this.startedAt = startedAt; }

        public Instant getCompletedAt() { return completedAt; }
        public void setCompletedAt(Instant completedAt) { this.completedAt = completedAt; }

        public String getError() { return error; }
        public void setError(String error) { this.error = error; }
    }

    public static class FlowExecutionStep {
        private String nodeId;
        private String nodeType;
        private Instant startedAt;
        private Instant completedAt;
        private Map<String, Object> input;
        private Map<String, Object> output;
        private String error;

        public String getNodeId() { return nodeId; }
        public void setNodeId(String nodeId) { this.nodeId = nodeId; }

        public String getNodeType() { return nodeType; }
        public void setNodeType(String nodeType) { this.nodeType = nodeType; }

        public Instant getStartedAt() { return startedAt; }
        public void setStartedAt(Instant startedAt) { this.startedAt = startedAt; }

        public Instant getCompletedAt() { return completedAt; }
        public void setCompletedAt(Instant completedAt) { this.completedAt = completedAt; }

        public Map<String, Object> getInput() { return input; }
        public void setInput(Map<String, Object> input) { this.input = input; }

        public Map<String, Object> getOutput() { return output; }
        public void setOutput(Map<String, Object> output) { this.output = output; }

        public String getError() { return error; }
        public void setError(String error) { this.error = error; }
    }

    public static class CreateFlowInput {
        private String name;
        private String description;
        private List<FlowNode> nodes;
        private List<FlowEdge> edges;
        private List<FlowVariable> variables;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public List<FlowNode> getNodes() { return nodes; }
        public void setNodes(List<FlowNode> nodes) { this.nodes = nodes; }

        public List<FlowEdge> getEdges() { return edges; }
        public void setEdges(List<FlowEdge> edges) { this.edges = edges; }

        public List<FlowVariable> getVariables() { return variables; }
        public void setVariables(List<FlowVariable> variables) { this.variables = variables; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CreateFlowInput input = new CreateFlowInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder description(String description) { input.description = description; return this; }
            public Builder nodes(List<FlowNode> nodes) { input.nodes = nodes; return this; }
            public Builder edges(List<FlowEdge> edges) { input.edges = edges; return this; }
            public Builder variables(List<FlowVariable> variables) { input.variables = variables; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public CreateFlowInput build() { return input; }
        }
    }

    public static class UpdateFlowInput {
        private String name;
        private String description;
        private FlowStatus status;
        private List<FlowNode> nodes;
        private List<FlowEdge> edges;
        private List<FlowVariable> variables;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public FlowStatus getStatus() { return status; }
        public void setStatus(FlowStatus status) { this.status = status; }

        public List<FlowNode> getNodes() { return nodes; }
        public void setNodes(List<FlowNode> nodes) { this.nodes = nodes; }

        public List<FlowEdge> getEdges() { return edges; }
        public void setEdges(List<FlowEdge> edges) { this.edges = edges; }

        public List<FlowVariable> getVariables() { return variables; }
        public void setVariables(List<FlowVariable> variables) { this.variables = variables; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final UpdateFlowInput input = new UpdateFlowInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder description(String description) { input.description = description; return this; }
            public Builder status(FlowStatus status) { input.status = status; return this; }
            public Builder nodes(List<FlowNode> nodes) { input.nodes = nodes; return this; }
            public Builder edges(List<FlowEdge> edges) { input.edges = edges; return this; }
            public Builder variables(List<FlowVariable> variables) { input.variables = variables; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public UpdateFlowInput build() { return input; }
        }
    }

    public static class ExecuteFlowInput {
        private String conversationId;
        private Map<String, Object> variables;

        public String getConversationId() { return conversationId; }
        public void setConversationId(String conversationId) { this.conversationId = conversationId; }

        public Map<String, Object> getVariables() { return variables; }
        public void setVariables(Map<String, Object> variables) { this.variables = variables; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final ExecuteFlowInput input = new ExecuteFlowInput();

            public Builder conversationId(String conversationId) { input.conversationId = conversationId; return this; }
            public Builder variables(Map<String, Object> variables) { input.variables = variables; return this; }
            public ExecuteFlowInput build() { return input; }
        }
    }
}
