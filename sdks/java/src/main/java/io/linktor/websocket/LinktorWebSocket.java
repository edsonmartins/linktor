package io.linktor.websocket;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonDeserializer;
import io.linktor.types.Conversation;
import org.java_websocket.client.WebSocketClient;
import org.java_websocket.handshake.ServerHandshake;

import java.net.URI;
import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.function.Consumer;

public class LinktorWebSocket {
    private final String baseUrl;
    private final String apiKey;
    private final String accessToken;
    private final Gson gson;

    private WebSocketClient client;
    private boolean autoReconnect = true;
    private int reconnectAttempts = 0;
    private static final int MAX_RECONNECT_ATTEMPTS = 5;
    private static final long RECONNECT_DELAY_MS = 5000;

    private final Map<String, Consumer<WebSocketEvent>> eventHandlers = new ConcurrentHashMap<>();
    private Consumer<Conversation.Message> messageHandler;
    private Consumer<WebSocketEvent> connectionHandler;
    private Consumer<Throwable> errorHandler;

    public LinktorWebSocket(String baseUrl, String apiKey, String accessToken) {
        this.baseUrl = baseUrl.replace("http://", "ws://").replace("https://", "wss://");
        this.apiKey = apiKey;
        this.accessToken = accessToken;
        this.gson = new GsonBuilder()
                .registerTypeAdapter(Instant.class, (JsonDeserializer<Instant>) (json, typeOfT, context) ->
                        Instant.parse(json.getAsString()))
                .create();
    }

    /**
     * Connect to the WebSocket server
     */
    public void connect() {
        try {
            String url = baseUrl + "/ws/conversations";
            URI uri = new URI(url);

            Map<String, String> headers = new HashMap<>();
            if (apiKey != null && !apiKey.isEmpty()) {
                headers.put("X-API-Key", apiKey);
            } else if (accessToken != null && !accessToken.isEmpty()) {
                headers.put("Authorization", "Bearer " + accessToken);
            }

            client = new WebSocketClient(uri, headers) {
                @Override
                public void onOpen(ServerHandshake handshake) {
                    reconnectAttempts = 0;
                    if (connectionHandler != null) {
                        connectionHandler.accept(new WebSocketEvent("connected", null));
                    }
                }

                @Override
                public void onMessage(String message) {
                    handleMessage(message);
                }

                @Override
                public void onClose(int code, String reason, boolean remote) {
                    if (connectionHandler != null) {
                        connectionHandler.accept(new WebSocketEvent("disconnected", Map.of(
                                "code", code,
                                "reason", reason,
                                "remote", remote
                        )));
                    }
                    if (autoReconnect && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
                        scheduleReconnect();
                    }
                }

                @Override
                public void onError(Exception ex) {
                    if (errorHandler != null) {
                        errorHandler.accept(ex);
                    }
                }
            };

            client.connect();
        } catch (Exception e) {
            if (errorHandler != null) {
                errorHandler.accept(e);
            }
        }
    }

    /**
     * Disconnect from the WebSocket server
     */
    public void disconnect() {
        autoReconnect = false;
        if (client != null) {
            client.close();
        }
    }

    /**
     * Check if connected
     */
    public boolean isConnected() {
        return client != null && client.isOpen();
    }

    /**
     * Subscribe to a conversation
     */
    public void subscribe(String conversationId) {
        send(new WebSocketMessage("subscribe", Map.of("conversationId", conversationId)));
    }

    /**
     * Unsubscribe from a conversation
     */
    public void unsubscribe(String conversationId) {
        send(new WebSocketMessage("unsubscribe", Map.of("conversationId", conversationId)));
    }

    /**
     * Send a message through WebSocket
     */
    public void send(Object message) {
        if (client != null && client.isOpen()) {
            client.send(gson.toJson(message));
        }
    }

    /**
     * Set handler for new messages
     */
    public void onMessage(Consumer<Conversation.Message> handler) {
        this.messageHandler = handler;
    }

    /**
     * Set handler for connection events
     */
    public void onConnection(Consumer<WebSocketEvent> handler) {
        this.connectionHandler = handler;
    }

    /**
     * Set handler for errors
     */
    public void onError(Consumer<Throwable> handler) {
        this.errorHandler = handler;
    }

    /**
     * Set handler for specific event type
     */
    public void on(String eventType, Consumer<WebSocketEvent> handler) {
        eventHandlers.put(eventType, handler);
    }

    /**
     * Remove handler for specific event type
     */
    public void off(String eventType) {
        eventHandlers.remove(eventType);
    }

    /**
     * Set auto-reconnect behavior
     */
    public void setAutoReconnect(boolean autoReconnect) {
        this.autoReconnect = autoReconnect;
    }

    private void handleMessage(String messageStr) {
        try {
            WebSocketEvent event = gson.fromJson(messageStr, WebSocketEvent.class);

            // Handle specific event types
            if (eventHandlers.containsKey(event.getType())) {
                eventHandlers.get(event.getType()).accept(event);
            }

            // Handle message events
            if ("message.received".equals(event.getType()) || "message.sent".equals(event.getType())) {
                if (messageHandler != null && event.getData() != null) {
                    Conversation.Message msg = gson.fromJson(gson.toJson(event.getData()), Conversation.Message.class);
                    messageHandler.accept(msg);
                }
            }
        } catch (Exception e) {
            if (errorHandler != null) {
                errorHandler.accept(e);
            }
        }
    }

    private void scheduleReconnect() {
        reconnectAttempts++;
        new Thread(() -> {
            try {
                Thread.sleep(RECONNECT_DELAY_MS * reconnectAttempts);
                if (autoReconnect) {
                    connect();
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }).start();
    }

    // Inner classes
    public static class WebSocketEvent {
        private String type;
        private Object data;
        private Instant timestamp;

        public WebSocketEvent() {}

        public WebSocketEvent(String type, Object data) {
            this.type = type;
            this.data = data;
            this.timestamp = Instant.now();
        }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public Object getData() { return data; }
        public void setData(Object data) { this.data = data; }

        public Instant getTimestamp() { return timestamp; }
        public void setTimestamp(Instant timestamp) { this.timestamp = timestamp; }
    }

    public static class WebSocketMessage {
        private String type;
        private Object payload;

        public WebSocketMessage(String type, Object payload) {
            this.type = type;
            this.payload = payload;
        }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public Object getPayload() { return payload; }
        public void setPayload(Object payload) { this.payload = payload; }
    }
}
