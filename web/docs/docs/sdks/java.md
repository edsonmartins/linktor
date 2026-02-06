---
sidebar_position: 5
title: Java SDK
---

# Java SDK

The official Java SDK for Linktor provides a type-safe interface with builder patterns and WebSocket support for real-time communication.

## Installation

### Maven

Add the dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>io.linktor</groupId>
    <artifactId>linktor-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

Add the dependency to your `build.gradle`:

```groovy
implementation 'io.linktor:linktor-sdk:1.0.0'
```

Or for Kotlin DSL (`build.gradle.kts`):

```kotlin
implementation("io.linktor:linktor-sdk:1.0.0")
```

## Quick Start

### Initialize the Client

```java
import io.linktor.LinktorClient;

public class Main {
    public static void main(String[] args) {
        LinktorClient client = LinktorClient.builder()
            .apiKey(System.getenv("LINKTOR_API_KEY"))
            // Optional configuration
            .baseUrl("https://api.linktor.io")
            .timeout(30)
            .maxRetries(3)
            .build();
    }
}
```

### Send a Message

```java
import io.linktor.types.Message;
import io.linktor.types.SendMessageInput;

// Send a message to a conversation
Message message = client.conversations.sendMessage(
    "conversation-id",
    SendMessageInput.builder()
        .text("Hello from Java!")
        .build()
);

System.out.println("Message sent: " + message.getId());
```

### List Conversations

```java
import io.linktor.types.Conversation;
import io.linktor.types.PaginatedResponse;
import io.linktor.types.ListConversationsParams;

// Get all conversations with pagination
PaginatedResponse<Conversation> conversations = client.conversations.list(
    ListConversationsParams.builder()
        .limit(20)
        .status("open")
        .build()
);

for (Conversation conv : conversations.getData()) {
    System.out.println(conv.getId() + ": " + conv.getContact().getName());
}
```

### Work with Contacts

```java
import io.linktor.types.Contact;
import io.linktor.types.CreateContactInput;

// Create a new contact
Contact contact = client.contacts.create(
    CreateContactInput.builder()
        .name("John Doe")
        .email("john@example.com")
        .phone("+1234567890")
        .metadata(Map.of("customerId", "cust_123"))
        .build()
);

// List contacts
var results = client.contacts.list(
    ListContactsParams.builder()
        .search("john")
        .limit(10)
        .build()
);
```

## Real-time Updates (WebSocket)

Connect to the WebSocket for real-time message updates.

```java
import io.linktor.LinktorClient;
import io.linktor.websocket.LinktorWebSocket;
import io.linktor.websocket.MessageEvent;
import io.linktor.websocket.MessageStatusEvent;
import io.linktor.websocket.TypingEvent;

public class WebSocketExample {
    public static void main(String[] args) {
        LinktorClient client = LinktorClient.builder()
            .apiKey(System.getenv("LINKTOR_API_KEY"))
            .build();

        LinktorWebSocket ws = client.webSocket();

        // Connect to WebSocket
        ws.connect();

        // Subscribe to a conversation
        ws.subscribe("conversation-id");

        // Handle incoming messages
        ws.onMessage((MessageEvent event) -> {
            System.out.println("New message: " + event.getMessage().getText());
            System.out.println("From conversation: " + event.getConversationId());
        });

        // Handle message status updates
        ws.onMessageStatus((MessageStatusEvent event) -> {
            System.out.println("Message " + event.getMessageId() +
                             " status: " + event.getStatus());
        });

        // Handle typing indicators
        ws.onTyping((TypingEvent event) -> {
            if (event.isTyping()) {
                System.out.println("User " + event.getUserId() + " is typing...");
            }
        });

        // Send typing indicator
        ws.sendTyping("conversation-id", true);

        // Connection events
        ws.onConnected(() -> {
            System.out.println("WebSocket connected");
        });

        ws.onDisconnected((code, reason) -> {
            System.out.println("WebSocket disconnected: " + code + " - " + reason);
        });

        ws.onError((error) -> {
            System.err.println("WebSocket error: " + error.getMessage());
        });

        // Keep application running
        Thread.sleep(Long.MAX_VALUE);

        // Cleanup
        ws.disconnect();
    }
}
```

## Error Handling

The SDK provides a `LinktorException` class for handling API errors.

```java
import io.linktor.LinktorClient;
import io.linktor.utils.LinktorException;
import io.linktor.types.Conversation;

public class ErrorHandlingExample {
    public static void main(String[] args) {
        LinktorClient client = LinktorClient.builder()
            .apiKey("your-api-key")
            .build();

        try {
            Conversation conversation = client.conversations.get("invalid-id");
        } catch (LinktorException e) {
            switch (e.getStatusCode()) {
                case 404:
                    System.out.println("Conversation not found");
                    break;
                case 401:
                    System.out.println("Invalid API key");
                    break;
                case 429:
                    System.out.println("Rate limited. Retry after: " +
                                     e.getRetryAfter() + " seconds");
                    break;
                case 400:
                    System.out.println("Invalid request: " + e.getDetails());
                    break;
                default:
                    System.out.println("API error [" + e.getCode() + "]: " +
                                     e.getMessage());
                    System.out.println("Request ID: " + e.getRequestId());
            }
        }
    }
}
```

### Exception Properties

| Property | Type | Description |
|----------|------|-------------|
| `getCode()` | String | Error code (e.g., "NOT_FOUND") |
| `getMessage()` | String | Error message |
| `getStatusCode()` | int | HTTP status code |
| `getRequestId()` | String | Request ID for debugging |
| `getDetails()` | Map | Additional error details |
| `getRetryAfter()` | Integer | Seconds to wait before retrying (rate limit) |

## Webhook Verification

Verify incoming webhooks from Linktor.

```java
import io.linktor.LinktorClient;
import javax.servlet.http.*;
import java.io.*;

public class WebhookServlet extends HttpServlet {
    private static final String WEBHOOK_SECRET = System.getenv("WEBHOOK_SECRET");

    @Override
    protected void doPost(HttpServletRequest req, HttpServletResponse resp)
            throws IOException {

        // Read the request body
        byte[] payload = req.getInputStream().readAllBytes();
        String signature = req.getHeader("X-Linktor-Signature");

        // Verify the webhook signature
        if (!LinktorClient.verifyWebhook(payload, signature, WEBHOOK_SECRET)) {
            resp.setStatus(HttpServletResponse.SC_BAD_REQUEST);
            resp.getWriter().write("{\"error\": \"Invalid signature\"}");
            return;
        }

        // Parse and handle the event
        String json = new String(payload);
        // Process the webhook event...

        resp.setStatus(HttpServletResponse.SC_OK);
        resp.getWriter().write("{\"received\": true}");
    }
}
```

### With Spring Boot

```java
import io.linktor.LinktorClient;
import org.springframework.web.bind.annotation.*;

@RestController
public class WebhookController {

    @Value("${linktor.webhook.secret}")
    private String webhookSecret;

    @PostMapping("/webhook")
    public Map<String, Boolean> handleWebhook(
            @RequestBody byte[] payload,
            @RequestHeader("X-Linktor-Signature") String signature) {

        if (!LinktorClient.verifyWebhook(payload, signature, webhookSecret)) {
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "Invalid signature");
        }

        // Process the webhook event...

        return Map.of("received", true);
    }
}
```

## AI Features

### Completions

```java
import io.linktor.types.CompletionInput;
import io.linktor.types.ChatMessage;
import java.util.List;

// Create a completion
var response = client.ai.completions.create(
    CompletionInput.builder()
        .messages(List.of(
            ChatMessage.system("You are a helpful assistant."),
            ChatMessage.user("What is the capital of France?")
        ))
        .model("gpt-4")
        .build()
);

System.out.println(response.getMessage().getContent());
```

### Knowledge Bases

```java
import io.linktor.types.QueryResult;

// Query a knowledge base
QueryResult result = client.knowledgeBases.query(
    "kb-id",
    "How do I reset my password?",
    5  // topK
);

for (var chunk : result.getChunks()) {
    System.out.println("Match: " + chunk.getContent());
    System.out.println("Score: " + chunk.getScore());
}
```

### Embeddings

```java
// Create embeddings
List<Double> embedding = client.ai.embeddings.embed("Hello, world!");
System.out.println("Embedding dimension: " + embedding.size());
```

## Resources

The SDK provides access to all Linktor resources:

- `client.auth` - Authentication
- `client.conversations` - Conversations and messages
- `client.contacts` - Contact management
- `client.channels` - Channel configuration
- `client.bots` - Bot management
- `client.ai` - AI completions and embeddings
- `client.knowledgeBases` - Knowledge base operations
- `client.flows` - Conversation flows
- `client.analytics` - Analytics and metrics

## API Reference

For complete API documentation, see the [Java SDK Reference](https://github.com/linktor/linktor/tree/main/sdks/java).
