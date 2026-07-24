package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.Channel;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.List;

public class ChannelsResource {
    private final HttpClient http;

    public ChannelsResource(HttpClient http) {
        this.http = http;
    }

    /**
     * List all channels. The backend returns a plain array (no pagination).
     */
    public List<Channel.ChannelModel> list() throws LinktorException {
        Type responseType = new TypeToken<List<Channel.ChannelModel>>(){}.getType();
        return http.get("/channels", responseType);
    }

    /**
     * Get a channel by ID.
     */
    public Channel.ChannelModel get(String channelId) throws LinktorException {
        return http.get("/channels/" + channelId, Channel.ChannelModel.class);
    }

    /**
     * Create a new channel. Secrets are passed via {@code input.credentials}.
     */
    public Channel.ChannelModel create(Channel.CreateChannelInput input) throws LinktorException {
        return http.post("/channels", input, Channel.ChannelModel.class);
    }

    /**
     * Update a channel (PUT; reuses the create body shape).
     */
    public Channel.ChannelModel update(String channelId, Channel.UpdateChannelInput input) throws LinktorException {
        return http.put("/channels/" + channelId, input, Channel.ChannelModel.class);
    }

    /**
     * Delete a channel (204 No Content).
     */
    public void delete(String channelId) throws LinktorException {
        http.delete("/channels/" + channelId);
    }

    /**
     * Connect a channel. Returns a {@link Channel.ConnectResult} which may carry a
     * {@code qrCode} to render (WhatsApp Web linking) plus its {@code expiresIn}.
     */
    public Channel.ConnectResult connect(String channelId) throws LinktorException {
        return http.post("/channels/" + channelId + "/connect", null, Channel.ConnectResult.class);
    }

    /**
     * Request a phone-number pairing code for a channel. Returns a
     * {@link Channel.ConnectResult} carrying the {@code pairCode}.
     */
    public Channel.ConnectResult requestPairCode(String channelId, String phoneNumber) throws LinktorException {
        Channel.PairChannelInput body = new Channel.PairChannelInput(phoneNumber);
        return http.post("/channels/" + channelId + "/pair", body, Channel.ConnectResult.class);
    }

    /**
     * Disconnect a channel. Returns the updated channel.
     */
    public Channel.ChannelModel disconnect(String channelId) throws LinktorException {
        return http.post("/channels/" + channelId + "/disconnect", null, Channel.ChannelModel.class);
    }
}
