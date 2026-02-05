package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.Bot;
import io.linktor.types.Common;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.Map;

public class BotsResource {
    private final HttpClient http;

    public BotsResource(HttpClient http) {
        this.http = http;
    }

    /**
     * List bots with optional filters
     */
    public Common.PaginatedResponse<Bot.BotModel> list(Bot.ListBotsParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getStatus() != null) queryParams.put("status", params.getStatus().name().toLowerCase());
            if (params.getType() != null) queryParams.put("type", params.getType().name().toLowerCase());
            if (params.getChannelId() != null) queryParams.put("channelId", params.getChannelId());
            if (params.getSearch() != null) queryParams.put("search", params.getSearch());
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Bot.BotModel>>(){}.getType();
        return http.get("/bots", queryParams, responseType);
    }

    /**
     * List all bots (no filters)
     */
    public Common.PaginatedResponse<Bot.BotModel> list() throws LinktorException {
        return list(null);
    }

    /**
     * Get a bot by ID
     */
    public Bot.BotModel get(String botId) throws LinktorException {
        return http.get("/bots/" + botId, Bot.BotModel.class);
    }

    /**
     * Create a new bot
     */
    public Bot.BotModel create(Bot.CreateBotInput input) throws LinktorException {
        return http.post("/bots", input, Bot.BotModel.class);
    }

    /**
     * Update a bot
     */
    public Bot.BotModel update(String botId, Bot.UpdateBotInput input) throws LinktorException {
        return http.patch("/bots/" + botId, input, Bot.BotModel.class);
    }

    /**
     * Delete a bot
     */
    public void delete(String botId) throws LinktorException {
        http.delete("/bots/" + botId);
    }

    /**
     * Activate a bot
     */
    public Bot.BotModel activate(String botId) throws LinktorException {
        Bot.UpdateBotInput input = Bot.UpdateBotInput.builder().status(Bot.BotStatus.ACTIVE).build();
        return update(botId, input);
    }

    /**
     * Deactivate a bot
     */
    public Bot.BotModel deactivate(String botId) throws LinktorException {
        Bot.UpdateBotInput input = Bot.UpdateBotInput.builder().status(Bot.BotStatus.INACTIVE).build();
        return update(botId, input);
    }
}
