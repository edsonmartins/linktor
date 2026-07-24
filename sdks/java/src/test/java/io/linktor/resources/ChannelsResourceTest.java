package io.linktor.resources;

import com.sun.net.httpserver.HttpServer;
import io.linktor.types.Channel;
import io.linktor.utils.HttpClient;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Focused tests proving the ChannelsResource speaks the real wire contract:
 * the {@code {success,data}} envelope is unwrapped, connect() surfaces qr_code,
 * and create() sends write-only credentials.
 */
class ChannelsResourceTest {

    private HttpServer server;
    private ChannelsResource channels;
    private final AtomicReference<String> lastRequestBody = new AtomicReference<>();
    private final Map<String, String> cannedResponses = new HashMap<>();

    @BeforeEach
    void setUp() throws IOException {
        server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/", exchange -> {
            byte[] reqBytes = exchange.getRequestBody().readAllBytes();
            lastRequestBody.set(new String(reqBytes, StandardCharsets.UTF_8));

            String path = exchange.getRequestURI().getPath();
            String method = exchange.getRequestMethod();
            String key = method + " " + path;
            String body = cannedResponses.getOrDefault(key, "{\"success\":true,\"data\":null}");

            byte[] out = body.getBytes(StandardCharsets.UTF_8);
            exchange.getResponseHeaders().add("Content-Type", "application/json");
            exchange.sendResponseHeaders(200, out.length);
            exchange.getResponseBody().write(out);
            exchange.close();
        });
        server.start();

        String baseUrl = "http://127.0.0.1:" + server.getAddress().getPort();
        HttpClient http = new HttpClient(baseUrl, "lk_test", null, 5, 1);
        channels = new ChannelsResource(http);
    }

    @AfterEach
    void tearDown() {
        server.stop(0);
    }

    @Test
    void connectSurfacesQrCodeFromEnvelope() throws Exception {
        cannedResponses.put("POST /channels/c1/connect",
                "{\"success\":true,\"data\":{\"channel\":{\"id\":\"c1\",\"type\":\"whatsapp\","
                        + "\"connection_status\":\"connecting\",\"enabled\":true},"
                        + "\"qr_code\":\"Q\",\"expires_in\":60}}");

        Channel.ConnectResult result = channels.connect("c1");

        assertNotNull(result);
        assertEquals("Q", result.getQrCode());
        assertEquals(60, result.getExpiresIn());
        assertNotNull(result.getChannel());
        assertEquals("c1", result.getChannel().getId());
        assertEquals(Channel.ConnectionStatus.CONNECTING, result.getChannel().getConnectionStatus());
        assertEquals(Channel.ChannelType.WHATSAPP, result.getChannel().getType());
    }

    @Test
    void requestPairCodeSendsPhoneAndSurfacesPairCode() throws Exception {
        cannedResponses.put("POST /channels/c1/pair",
                "{\"success\":true,\"data\":{\"channel\":{\"id\":\"c1\"},\"pair_code\":\"ABCD-1234\"}}");

        Channel.ConnectResult result = channels.requestPairCode("c1", "+5511999999999");

        assertEquals("ABCD-1234", result.getPairCode());
        assertTrue(lastRequestBody.get().contains("\"phone_number\""));
        assertTrue(lastRequestBody.get().contains("+5511999999999"));
    }

    @Test
    void createSendsCredentialsAndUnwrapsChannel() throws Exception {
        cannedResponses.put("POST /channels",
                "{\"success\":true,\"data\":{\"id\":\"c9\",\"type\":\"whatsapp\",\"name\":\"WA\","
                        + "\"enabled\":true,\"connection_status\":\"disconnected\","
                        + "\"config\":{\"access_token\":\"__redacted__\"}}}");

        Map<String, String> creds = new HashMap<>();
        creds.put("access_token", "super-secret-token");
        Channel.CreateChannelInput input = Channel.CreateChannelInput.builder()
                .type(Channel.ChannelType.WHATSAPP)
                .name("WA")
                .credentials(creds)
                .build();

        Channel.ChannelModel created = channels.create(input);

        // Response is unwrapped from the envelope.
        assertEquals("c9", created.getId());
        assertEquals(Channel.ConnectionStatus.DISCONNECTED, created.getConnectionStatus());

        // Credentials were sent on the wire (snake_case, write-only).
        String sent = lastRequestBody.get();
        assertTrue(sent.contains("\"credentials\""), "credentials missing: " + sent);
        assertTrue(sent.contains("\"access_token\""), "access_token missing: " + sent);
        assertTrue(sent.contains("super-secret-token"), "secret value missing: " + sent);
        assertTrue(sent.contains("\"whatsapp\""), "type not serialized as wire enum: " + sent);
    }

    @Test
    void listUnwrapsPlainArray() throws Exception {
        cannedResponses.put("GET /channels",
                "{\"success\":true,\"data\":[{\"id\":\"a\"},{\"id\":\"b\"}],\"meta\":{}}");

        List<Channel.ChannelModel> result = channels.list();

        assertEquals(2, result.size());
        assertEquals("a", result.get(0).getId());
        assertEquals("b", result.get(1).getId());
    }
}
