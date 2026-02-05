# Linktor SDK for Java

Official Linktor SDK for Java applications.

## Installation

### Maven

```xml
<dependency>
    <groupId>io.linktor</groupId>
    <artifactId>linktor-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'io.linktor:linktor-sdk:1.0.0'
```

## Quick Start

```java
import io.linktor.LinktorClient;
import io.linktor.types.*;

public class Example {
    public static void main(String[] args) {
        LinktorClient client = LinktorClient.builder()
            .baseUrl("https://api.linktor.io")
            .apiKey("your-api-key")
            .build();

        // List conversations
        var convs = client.conversations.list(
            Conversation.ListConversationsParams.builder()
                .status(Conversation.ConversationStatus.OPEN)
                .limit(10)
                .build()
        );
        System.out.println("Found " + convs.getData().size() + " conversations");

        // Send a message
        var msg = client.conversations.sendText("conv-id", "Hello!");
        System.out.println("Sent message: " + msg.getId());

        // Use AI
        String answer = client.ai.completions.complete("What is 2 + 2?");
        System.out.println(answer); // "4"
    }
}
```

## Authentication

### API Key

```java
LinktorClient client = LinktorClient.builder()
    .apiKey("your-api-key")
    .build();
```

### Access Token

```java
LinktorClient client = LinktorClient.builder()
    .accessToken("user-token")
    .build();
```

### Login

```java
LinktorClient client = LinktorClient.builder()
    .baseUrl("https://api.linktor.io")
    .build();

Auth.LoginResponse response = client.auth.login("user@example.com", "password");
System.out.println(response.getUser().getName());
```

## Resources

### Conversations

```java
// List conversations
var convs = client.conversations.list(
    Conversation.ListConversationsParams.builder()
        .status(Conversation.ConversationStatus.OPEN)
        .build()
);

// Get conversation
var conv = client.conversations.get("conv-id");

// Send text message
var msg = client.conversations.sendText("conv-id", "Hello!");

// Send message with options
var msg = client.conversations.sendMessage("conv-id",
    Conversation.SendMessageInput.builder()
        .text("Hello!")
        .metadata(Map.of("key", "value"))
        .build()
);

// Resolve conversation
client.conversations.resolve("conv-id");

// Assign to agent
client.conversations.assign("conv-id", "agent-id");
```

### Contacts

```java
// Create contact
var contact = client.contacts.create(
    Contact.CreateContactInput.builder()
        .name("John Doe")
        .email("john@example.com")
        .phone("+1234567890")
        .build()
);

// Get contact
var contact = client.contacts.get("contact-id");

// Update contact
var contact = client.contacts.update("contact-id",
    Contact.UpdateContactInput.builder()
        .name("John Smith")
        .build()
);

// Delete contact
client.contacts.delete("contact-id");
```

### Channels

```java
// List channels
var channels = client.channels.list();

// Create channel
var channel = client.channels.create(
    Channel.CreateChannelInput.builder()
        .name("WhatsApp Business")
        .type(Channel.ChannelType.WHATSAPP)
        .config(Map.of(
            "phoneNumberId", "123456789",
            "businessAccountId", "987654321",
            "accessToken", "token",
            "verifyToken", "verify"
        ))
        .build()
);

// Connect/disconnect
client.channels.connect("channel-id");
client.channels.disconnect("channel-id");
```

### Bots

```java
// Create AI bot
var bot = client.bots.create(
    Bot.CreateBotInput.builder()
        .name("Support Bot")
        .type(Bot.BotType.AI)
        .config(Map.of(
            "welcomeMessage", "Hello! How can I help?",
            "aiConfig", Map.of(
                "model", "gpt-4",
                "systemPrompt", "You are a helpful support agent.",
                "useKnowledgeBase", true
            )
        ))
        .build()
);
```

### AI

```java
// Simple completion
String answer = client.ai.completions.complete("What is the capital of France?");

// Chat with messages
var response = client.ai.completions.chat(List.of(
    AI.ChatMessage.system("You are a helpful assistant."),
    AI.ChatMessage.user("What is 2 + 2?")
));
System.out.println(response.getContent());

// Create embeddings
double[] embedding = client.ai.embeddings.embed("Hello world");
```

### Knowledge Bases

```java
// Create knowledge base
var kb = client.knowledgeBases.create(
    Knowledge.CreateKnowledgeBaseInput.builder()
        .name("FAQ Knowledge Base")
        .description("Frequently asked questions")
        .build()
);

// Add document
var doc = client.knowledgeBases.addDocument("kb-id", "FAQ", "Question: How to reset password?\nAnswer: ...");

// Query knowledge base
var result = client.knowledgeBases.query("kb-id", "How to reset password?", 5);
for (var chunk : result.getChunks()) {
    System.out.println(chunk.getContent());
}
```

### Flows

```java
// Create flow
var flow = client.flows.create(
    Flow.CreateFlowInput.builder()
        .name("Welcome Flow")
        .description("Onboarding flow for new users")
        .build()
);

// Execute flow
var execution = client.flows.execute("flow-id", "conv-id");
```

### Analytics

```java
// Dashboard metrics
var dashboard = client.analytics.getDashboard();
System.out.println("Total conversations: " + dashboard.getTotalConversations());

// Realtime metrics
var realtime = client.analytics.getRealtime();
System.out.println("Active conversations: " + realtime.getActiveConversations());
```

## WebSocket

```java
// Connect to WebSocket
client.webSocket().connect();

// Handle messages
client.webSocket().onMessage(message -> {
    System.out.println("New message: " + message.getText());
});

// Handle connection events
client.webSocket().onConnection(event -> {
    System.out.println("Connection event: " + event.getType());
});

// Handle errors
client.webSocket().onError(error -> {
    System.err.println("WebSocket error: " + error.getMessage());
});

// Subscribe to conversation
client.webSocket().subscribe("conv-id");

// Disconnect
client.webSocket().disconnect();
```

## Webhooks

### Verify Signature

```java
import io.linktor.utils.WebhookVerifier;

// Verify signature only
boolean isValid = WebhookVerifier.verifySignature(payload, signature, secret);

// Verify with timestamp
Map<String, String> headers = Map.of(
    "X-Linktor-Signature", signature,
    "X-Linktor-Timestamp", timestamp
);
boolean isValid = WebhookVerifier.verify(payload, headers, secret);

// Parse event
var event = WebhookVerifier.constructEvent(payload, headers, secret);
System.out.println("Event type: " + event.getType());
```

### Spring Boot Example

```java
import io.linktor.utils.WebhookVerifier;
import io.linktor.types.Webhook;

@RestController
public class WebhookController {

    @Value("${linktor.webhook.secret}")
    private String webhookSecret;

    @PostMapping("/webhook")
    public ResponseEntity<String> handleWebhook(
            @RequestBody byte[] payload,
            @RequestHeader Map<String, String> headers) {
        try {
            var event = WebhookVerifier.constructEvent(payload, headers, webhookSecret);

            switch (event.getType()) {
                case "message.received":
                    handleNewMessage(event);
                    break;
                case "conversation.created":
                    handleNewConversation(event);
                    break;
            }

            return ResponseEntity.ok("OK");
        } catch (Exception e) {
            return ResponseEntity.badRequest().body(e.getMessage());
        }
    }
}
```

## Error Handling

```java
import io.linktor.utils.LinktorException;

try {
    var conv = client.conversations.get("invalid-id");
} catch (LinktorException.NotFoundException e) {
    System.out.println("Conversation not found");
} catch (LinktorException.AuthenticationException e) {
    System.out.println("Authentication failed");
} catch (LinktorException.RateLimitException e) {
    System.out.println("Rate limited. Retry after: " + e.getRetryAfter() + " seconds");
} catch (LinktorException e) {
    System.out.println("Error: " + e.getMessage());
    System.out.println("Status: " + e.getStatusCode());
    System.out.println("Code: " + e.getErrorCode());
}
```

## Configuration

```java
LinktorClient client = LinktorClient.builder()
    .baseUrl("https://api.linktor.io")
    .apiKey("your-api-key")
    .timeout(30)        // Timeout in seconds
    .maxRetries(3)      // Number of retries
    .build();
```

## Requirements

- Java 17 or higher
- Dependencies:
  - OkHttp 4.x
  - Gson 2.x
  - Java-WebSocket 1.5.x

## License

MIT
