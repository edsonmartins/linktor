package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.Channel;
import io.linktor.types.Common;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.Map;

public class ChannelsResource {
    private final HttpClient http;

    public ChannelsResource(HttpClient http) {
        this.http = http;
    }

    /**
     * List channels with optional filters
     */
    public Common.PaginatedResponse<Channel.ChannelModel> list(Channel.ListChannelsParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getType() != null) queryParams.put("type", params.getType().name().toLowerCase());
            if (params.getStatus() != null) queryParams.put("status", params.getStatus().name().toLowerCase());
            if (params.getSearch() != null) queryParams.put("search", params.getSearch());
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Channel.ChannelModel>>(){}.getType();
        return http.get("/channels", queryParams, responseType);
    }

    /**
     * List all channels (no filters)
     */
    public Common.PaginatedResponse<Channel.ChannelModel> list() throws LinktorException {
        return list(null);
    }

    /**
     * Get a channel by ID
     */
    public Channel.ChannelModel get(String channelId) throws LinktorException {
        return http.get("/channels/" + channelId, Channel.ChannelModel.class);
    }

    /**
     * Create a new channel
     */
    public Channel.ChannelModel create(Channel.CreateChannelInput input) throws LinktorException {
        return http.post("/channels", input, Channel.ChannelModel.class);
    }

    /**
     * Update a channel
     */
    public Channel.ChannelModel update(String channelId, Channel.UpdateChannelInput input) throws LinktorException {
        return http.patch("/channels/" + channelId, input, Channel.ChannelModel.class);
    }

    /**
     * Delete a channel
     */
    public void delete(String channelId) throws LinktorException {
        http.delete("/channels/" + channelId);
    }

    /**
     * Connect a channel
     */
    public Channel.ChannelModel connect(String channelId) throws LinktorException {
        return http.post("/channels/" + channelId + "/connect", null, Channel.ChannelModel.class);
    }

    /**
     * Disconnect a channel
     */
    public Channel.ChannelModel disconnect(String channelId) throws LinktorException {
        return http.post("/channels/" + channelId + "/disconnect", null, Channel.ChannelModel.class);
    }

    /**
     * Get channel status
     */
    public Channel.ChannelStatusResponse getStatus(String channelId) throws LinktorException {
        return http.get("/channels/" + channelId + "/status", Channel.ChannelStatusResponse.class);
    }

    /**
     * Test channel connection
     */
    public Channel.ChannelStatusResponse test(String channelId) throws LinktorException {
        return http.post("/channels/" + channelId + "/test", null, Channel.ChannelStatusResponse.class);
    }
}
