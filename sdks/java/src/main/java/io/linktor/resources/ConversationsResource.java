package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.Common;
import io.linktor.types.Conversation;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ConversationsResource {
    private final HttpClient http;

    public ConversationsResource(HttpClient http) {
        this.http = http;
    }

    /**
     * List conversations with optional filters
     */
    public Common.PaginatedResponse<Conversation.ConversationModel> list(Conversation.ListConversationsParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getStatus() != null) queryParams.put("status", params.getStatus().name().toLowerCase());
            if (params.getPriority() != null) queryParams.put("priority", params.getPriority().name().toLowerCase());
            if (params.getChannelId() != null) queryParams.put("channelId", params.getChannelId());
            if (params.getContactId() != null) queryParams.put("contactId", params.getContactId());
            if (params.getAssignedAgentId() != null) queryParams.put("assignedAgentId", params.getAssignedAgentId());
            if (params.getTag() != null) queryParams.put("tag", params.getTag());
            if (params.getSearch() != null) queryParams.put("search", params.getSearch());
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
            if (params.getCursor() != null) queryParams.put("cursor", params.getCursor());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Conversation.ConversationModel>>(){}.getType();
        return http.get("/conversations", queryParams, responseType);
    }

    /**
     * List all conversations (no filters)
     */
    public Common.PaginatedResponse<Conversation.ConversationModel> list() throws LinktorException {
        return list(null);
    }

    /**
     * Get a conversation by ID
     */
    public Conversation.ConversationModel get(String conversationId) throws LinktorException {
        return http.get("/conversations/" + conversationId, Conversation.ConversationModel.class);
    }

    /**
     * Update a conversation
     */
    public Conversation.ConversationModel update(String conversationId, Conversation.UpdateConversationInput input) throws LinktorException {
        return http.patch("/conversations/" + conversationId, input, Conversation.ConversationModel.class);
    }

    /**
     * Send a text message to a conversation
     */
    public Conversation.Message sendText(String conversationId, String text) throws LinktorException {
        Conversation.SendMessageInput input = Conversation.SendMessageInput.builder().text(text).build();
        return sendMessage(conversationId, input);
    }

    /**
     * Send a message to a conversation
     */
    public Conversation.Message sendMessage(String conversationId, Conversation.SendMessageInput input) throws LinktorException {
        return http.post("/conversations/" + conversationId + "/messages", input, Conversation.Message.class);
    }

    /**
     * Get messages from a conversation
     */
    public Common.PaginatedResponse<Conversation.Message> getMessages(String conversationId, Common.PaginationParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
            if (params.getCursor() != null) queryParams.put("cursor", params.getCursor());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Conversation.Message>>(){}.getType();
        return http.get("/conversations/" + conversationId + "/messages", queryParams, responseType);
    }

    /**
     * Get messages from a conversation (no pagination)
     */
    public Common.PaginatedResponse<Conversation.Message> getMessages(String conversationId) throws LinktorException {
        return getMessages(conversationId, null);
    }

    /**
     * Resolve a conversation
     */
    public Conversation.ConversationModel resolve(String conversationId) throws LinktorException {
        return http.post("/conversations/" + conversationId + "/resolve", null, Conversation.ConversationModel.class);
    }

    /**
     * Reopen a conversation
     */
    public Conversation.ConversationModel reopen(String conversationId) throws LinktorException {
        return http.post("/conversations/" + conversationId + "/reopen", null, Conversation.ConversationModel.class);
    }

    /**
     * Assign conversation to an agent
     */
    public Conversation.ConversationModel assign(String conversationId, String agentId) throws LinktorException {
        Map<String, String> body = new HashMap<>();
        body.put("agentId", agentId);
        return http.post("/conversations/" + conversationId + "/assign", body, Conversation.ConversationModel.class);
    }

    /**
     * Unassign conversation
     */
    public Conversation.ConversationModel unassign(String conversationId) throws LinktorException {
        return http.post("/conversations/" + conversationId + "/unassign", null, Conversation.ConversationModel.class);
    }

    /**
     * Add tags to a conversation
     */
    public Conversation.ConversationModel addTags(String conversationId, List<String> tags) throws LinktorException {
        Map<String, Object> body = new HashMap<>();
        body.put("tags", tags);
        return http.post("/conversations/" + conversationId + "/tags", body, Conversation.ConversationModel.class);
    }

    /**
     * Remove tags from a conversation
     */
    public Conversation.ConversationModel removeTags(String conversationId, List<String> tags) throws LinktorException {
        Map<String, Object> body = new HashMap<>();
        body.put("tags", tags);
        return http.post("/conversations/" + conversationId + "/tags/remove", body, Conversation.ConversationModel.class);
    }

    /**
     * Set conversation priority
     */
    public Conversation.ConversationModel setPriority(String conversationId, Conversation.ConversationPriority priority) throws LinktorException {
        Map<String, String> body = new HashMap<>();
        body.put("priority", priority.name().toLowerCase());
        return http.patch("/conversations/" + conversationId, body, Conversation.ConversationModel.class);
    }
}
