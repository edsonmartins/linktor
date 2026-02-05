package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.AI;
import io.linktor.types.Common;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class AIResource {
    private final HttpClient http;
    public final AgentsResource agents;
    public final CompletionsResource completions;
    public final EmbeddingsResource embeddings;

    public AIResource(HttpClient http) {
        this.http = http;
        this.agents = new AgentsResource(http);
        this.completions = new CompletionsResource(http);
        this.embeddings = new EmbeddingsResource(http);
    }

    // Inner class for Agents
    public static class AgentsResource {
        private final HttpClient http;

        public AgentsResource(HttpClient http) {
            this.http = http;
        }

        /**
         * List all agents
         */
        public Common.PaginatedResponse<AI.Agent> list(Common.PaginationParams params) throws LinktorException {
            Map<String, String> queryParams = new HashMap<>();
            if (params != null) {
                if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
                if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
            }

            Type responseType = new TypeToken<Common.PaginatedResponse<AI.Agent>>(){}.getType();
            return http.get("/ai/agents", queryParams, responseType);
        }

        /**
         * List all agents (no pagination)
         */
        public Common.PaginatedResponse<AI.Agent> list() throws LinktorException {
            return list(null);
        }

        /**
         * Get an agent by ID
         */
        public AI.Agent get(String agentId) throws LinktorException {
            return http.get("/ai/agents/" + agentId, AI.Agent.class);
        }

        /**
         * Create a new agent
         */
        public AI.Agent create(AI.CreateAgentInput input) throws LinktorException {
            return http.post("/ai/agents", input, AI.Agent.class);
        }

        /**
         * Update an agent
         */
        public AI.Agent update(String agentId, AI.CreateAgentInput input) throws LinktorException {
            return http.patch("/ai/agents/" + agentId, input, AI.Agent.class);
        }

        /**
         * Delete an agent
         */
        public void delete(String agentId) throws LinktorException {
            http.delete("/ai/agents/" + agentId);
        }

        /**
         * Invoke an agent with a message
         */
        public AI.CompletionResponse invoke(String agentId, String message) throws LinktorException {
            Map<String, Object> body = new HashMap<>();
            body.put("message", message);
            return http.post("/ai/agents/" + agentId + "/invoke", body, AI.CompletionResponse.class);
        }

        /**
         * Invoke an agent with conversation context
         */
        public AI.CompletionResponse invoke(String agentId, List<AI.ChatMessage> messages) throws LinktorException {
            Map<String, Object> body = new HashMap<>();
            body.put("messages", messages);
            return http.post("/ai/agents/" + agentId + "/invoke", body, AI.CompletionResponse.class);
        }
    }

    // Inner class for Completions
    public static class CompletionsResource {
        private final HttpClient http;

        public CompletionsResource(HttpClient http) {
            this.http = http;
        }

        /**
         * Simple completion with just a question
         */
        public String complete(String question) throws LinktorException {
            AI.CompletionResponse response = chat(Arrays.asList(AI.ChatMessage.user(question)));
            return response.getContent();
        }

        /**
         * Create a chat completion
         */
        public AI.CompletionResponse chat(List<AI.ChatMessage> messages) throws LinktorException {
            AI.CompletionInput input = AI.CompletionInput.builder()
                    .messages(messages)
                    .build();
            return create(input);
        }

        /**
         * Create a chat completion with custom options
         */
        public AI.CompletionResponse create(AI.CompletionInput input) throws LinktorException {
            return http.post("/ai/completions", input, AI.CompletionResponse.class);
        }

        /**
         * Chat with a specific model
         */
        public AI.CompletionResponse chat(List<AI.ChatMessage> messages, String model) throws LinktorException {
            AI.CompletionInput input = AI.CompletionInput.builder()
                    .messages(messages)
                    .model(model)
                    .build();
            return create(input);
        }

        /**
         * Chat with model and temperature
         */
        public AI.CompletionResponse chat(List<AI.ChatMessage> messages, String model, Double temperature) throws LinktorException {
            AI.CompletionInput input = AI.CompletionInput.builder()
                    .messages(messages)
                    .model(model)
                    .temperature(temperature)
                    .build();
            return create(input);
        }
    }

    // Inner class for Embeddings
    public static class EmbeddingsResource {
        private final HttpClient http;

        public EmbeddingsResource(HttpClient http) {
            this.http = http;
        }

        /**
         * Create embedding for a single text
         */
        public double[] embed(String text) throws LinktorException {
            AI.EmbeddingResponse response = create(AI.EmbeddingInput.single(text));
            return response.getEmbedding();
        }

        /**
         * Create embeddings for multiple texts
         */
        public AI.EmbeddingResponse embedBatch(List<String> texts) throws LinktorException {
            return create(AI.EmbeddingInput.batch(texts));
        }

        /**
         * Create embeddings with full options
         */
        public AI.EmbeddingResponse create(AI.EmbeddingInput input) throws LinktorException {
            return http.post("/ai/embeddings", input, AI.EmbeddingResponse.class);
        }

        /**
         * Create embedding with specific model
         */
        public double[] embed(String text, String model) throws LinktorException {
            AI.EmbeddingInput input = AI.EmbeddingInput.single(text);
            input.setModel(model);
            AI.EmbeddingResponse response = create(input);
            return response.getEmbedding();
        }
    }
}
