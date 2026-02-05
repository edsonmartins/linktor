package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.Common;
import io.linktor.types.Flow;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.Map;

public class FlowsResource {
    private final HttpClient http;

    public FlowsResource(HttpClient http) {
        this.http = http;
    }

    /**
     * List flows
     */
    public Common.PaginatedResponse<Flow.FlowModel> list(Common.PaginationParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Flow.FlowModel>>(){}.getType();
        return http.get("/flows", queryParams, responseType);
    }

    /**
     * List all flows (no pagination)
     */
    public Common.PaginatedResponse<Flow.FlowModel> list() throws LinktorException {
        return list(null);
    }

    /**
     * Get a flow by ID
     */
    public Flow.FlowModel get(String flowId) throws LinktorException {
        return http.get("/flows/" + flowId, Flow.FlowModel.class);
    }

    /**
     * Create a new flow
     */
    public Flow.FlowModel create(Flow.CreateFlowInput input) throws LinktorException {
        return http.post("/flows", input, Flow.FlowModel.class);
    }

    /**
     * Update a flow
     */
    public Flow.FlowModel update(String flowId, Flow.UpdateFlowInput input) throws LinktorException {
        return http.patch("/flows/" + flowId, input, Flow.FlowModel.class);
    }

    /**
     * Delete a flow
     */
    public void delete(String flowId) throws LinktorException {
        http.delete("/flows/" + flowId);
    }

    /**
     * Execute a flow for a conversation
     */
    public Flow.FlowExecution execute(String flowId, String conversationId) throws LinktorException {
        Flow.ExecuteFlowInput input = Flow.ExecuteFlowInput.builder()
                .conversationId(conversationId)
                .build();
        return execute(flowId, input);
    }

    /**
     * Execute a flow with variables
     */
    public Flow.FlowExecution execute(String flowId, String conversationId, Map<String, Object> variables) throws LinktorException {
        Flow.ExecuteFlowInput input = Flow.ExecuteFlowInput.builder()
                .conversationId(conversationId)
                .variables(variables)
                .build();
        return execute(flowId, input);
    }

    /**
     * Execute a flow with full options
     */
    public Flow.FlowExecution execute(String flowId, Flow.ExecuteFlowInput input) throws LinktorException {
        return http.post("/flows/" + flowId + "/execute", input, Flow.FlowExecution.class);
    }

    /**
     * Get flow execution status
     */
    public Flow.FlowExecution getExecution(String flowId, String executionId) throws LinktorException {
        return http.get("/flows/" + flowId + "/executions/" + executionId, Flow.FlowExecution.class);
    }

    /**
     * Cancel a flow execution
     */
    public Flow.FlowExecution cancelExecution(String flowId, String executionId) throws LinktorException {
        return http.post("/flows/" + flowId + "/executions/" + executionId + "/cancel", null, Flow.FlowExecution.class);
    }

    /**
     * Activate a flow
     */
    public Flow.FlowModel activate(String flowId) throws LinktorException {
        Flow.UpdateFlowInput input = Flow.UpdateFlowInput.builder().status(Flow.FlowStatus.ACTIVE).build();
        return update(flowId, input);
    }

    /**
     * Deactivate a flow
     */
    public Flow.FlowModel deactivate(String flowId) throws LinktorException {
        Flow.UpdateFlowInput input = Flow.UpdateFlowInput.builder().status(Flow.FlowStatus.INACTIVE).build();
        return update(flowId, input);
    }

    /**
     * Duplicate a flow
     */
    public Flow.FlowModel duplicate(String flowId, String newName) throws LinktorException {
        Map<String, String> body = new HashMap<>();
        body.put("name", newName);
        return http.post("/flows/" + flowId + "/duplicate", body, Flow.FlowModel.class);
    }
}
