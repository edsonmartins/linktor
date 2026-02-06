---
sidebar_position: 7
title: .NET SDK
---

# .NET SDK

The official .NET SDK for Linktor provides an async-first interface with full support for .NET 6+ and modern C# features.

## Installation

Install the SDK using the .NET CLI:

```bash
dotnet add package Linktor.SDK
```

Or using the Package Manager Console:

```powershell
Install-Package Linktor.SDK
```

Or add to your `.csproj`:

```xml
<PackageReference Include="Linktor.SDK" Version="1.0.0" />
```

## Quick Start

### Initialize the Client

```csharp
using Linktor;

// Create client with configuration
var client = new LinktorClient(new LinktorClientOptions
{
    ApiKey = Environment.GetEnvironmentVariable("LINKTOR_API_KEY"),
    // Optional configuration
    BaseUrl = "https://api.linktor.io",
    TimeoutSeconds = 30,
    MaxRetries = 3
});
```

### Send a Message

```csharp
using Linktor;
using Linktor.Types;

// Send a message to a conversation
var message = await client.Conversations.SendMessageAsync(
    "conversation-id",
    new SendMessageInput { Text = "Hello from .NET!" }
);

Console.WriteLine($"Message sent: {message.Id}");
```

### List Conversations

```csharp
// Get all conversations with pagination
var conversations = await client.Conversations.ListAsync(new ListConversationsParams
{
    Limit = 20,
    Status = "open"
});

foreach (var conv in conversations.Data)
{
    Console.WriteLine($"{conv.Id}: {conv.Contact.Name}");
}
```

### Work with Contacts

```csharp
using Linktor.Types;

// Create a new contact
var contact = await client.Contacts.CreateAsync(new CreateContactInput
{
    Name = "John Doe",
    Email = "john@example.com",
    Phone = "+1234567890",
    Metadata = new Dictionary<string, object>
    {
        { "customerId", "cust_123" }
    }
});

// Search contacts
var results = await client.Contacts.ListAsync(new ListContactsParams
{
    Search = "john",
    Limit = 10
});
```

## Real-time Updates (WebSocket)

Connect to the WebSocket for real-time message updates.

```csharp
using Linktor;
using Linktor.WebSocket;

var client = new LinktorClient(new LinktorClientOptions
{
    ApiKey = Environment.GetEnvironmentVariable("LINKTOR_API_KEY")
});

// Get the WebSocket client
var ws = client.WebSocket;

// Connect to WebSocket
await ws.ConnectAsync();

// Subscribe to a conversation
await ws.SubscribeAsync("conversation-id");

// Handle incoming messages
ws.OnMessage += (sender, e) =>
{
    Console.WriteLine($"New message: {e.Message.Text}");
    Console.WriteLine($"From conversation: {e.ConversationId}");
};

// Handle message status updates
ws.OnMessageStatus += (sender, e) =>
{
    Console.WriteLine($"Message {e.MessageId} status: {e.Status}");
};

// Handle typing indicators
ws.OnTyping += (sender, e) =>
{
    if (e.IsTyping)
    {
        Console.WriteLine($"User {e.UserId} is typing...");
    }
};

// Connection events
ws.OnConnected += (sender, e) =>
{
    Console.WriteLine("WebSocket connected");
};

ws.OnDisconnected += (sender, e) =>
{
    Console.WriteLine($"WebSocket disconnected: {e.Code} - {e.Reason}");
};

ws.OnError += (sender, e) =>
{
    Console.WriteLine($"WebSocket error: {e.Error.Message}");
};

// Send typing indicator
await ws.SendTypingAsync("conversation-id", true);

// Keep application running
await Task.Delay(Timeout.Infinite);

// Cleanup
await ws.DisconnectAsync();
```

## Error Handling

The SDK provides a `LinktorException` class for handling API errors.

```csharp
using Linktor;
using Linktor.Utils;

var client = new LinktorClient(new LinktorClientOptions
{
    ApiKey = "your-api-key"
});

try
{
    var conversation = await client.Conversations.GetAsync("invalid-id");
}
catch (LinktorException ex)
{
    switch (ex.StatusCode)
    {
        case 404:
            Console.WriteLine("Conversation not found");
            break;
        case 401:
            Console.WriteLine("Invalid API key");
            break;
        case 429:
            Console.WriteLine($"Rate limited. Retry after {ex.RetryAfter} seconds");
            break;
        case 400:
            Console.WriteLine($"Invalid request: {ex.Details}");
            break;
        default:
            Console.WriteLine($"API error [{ex.Code}]: {ex.Message}");
            Console.WriteLine($"Request ID: {ex.RequestId}");
            break;
    }
}
```

### Exception Properties

| Property | Type | Description |
|----------|------|-------------|
| `Code` | string | Error code (e.g., "NOT_FOUND") |
| `Message` | string | Error message |
| `StatusCode` | int | HTTP status code |
| `RequestId` | string | Request ID for debugging |
| `Details` | Dictionary | Additional error details |
| `RetryAfter` | int? | Seconds to wait before retrying (rate limit) |

## Webhook Verification

Verify incoming webhooks from Linktor.

### ASP.NET Core

```csharp
using Linktor;
using Linktor.Types;
using Microsoft.AspNetCore.Mvc;

[ApiController]
[Route("webhook")]
public class WebhookController : ControllerBase
{
    private readonly string _webhookSecret;

    public WebhookController(IConfiguration config)
    {
        _webhookSecret = config["Linktor:WebhookSecret"];
    }

    [HttpPost]
    public async Task<IActionResult> HandleWebhook()
    {
        using var reader = new StreamReader(Request.Body);
        var payload = await reader.ReadToEndAsync();

        var signature = Request.Headers["X-Linktor-Signature"].FirstOrDefault();

        // Verify the webhook signature
        if (!LinktorClient.VerifyWebhookSignature(
            Encoding.UTF8.GetBytes(payload),
            signature,
            _webhookSecret))
        {
            return BadRequest(new { error = "Invalid signature" });
        }

        // Parse and handle the event
        var headers = Request.Headers.ToDictionary(
            h => h.Key,
            h => h.Value.FirstOrDefault() ?? ""
        );

        var webhookEvent = LinktorClient.ConstructWebhookEvent(
            Encoding.UTF8.GetBytes(payload),
            headers,
            _webhookSecret
        );

        switch (webhookEvent.Type)
        {
            case "message.received":
                // Handle new message
                break;
            case "conversation.created":
                // Handle new conversation
                break;
        }

        return Ok(new { received = true });
    }
}
```

### Minimal API

```csharp
app.MapPost("/webhook", async (HttpContext context) =>
{
    using var reader = new StreamReader(context.Request.Body);
    var payload = await reader.ReadToEndAsync();

    var signature = context.Request.Headers["X-Linktor-Signature"].FirstOrDefault();
    var secret = builder.Configuration["Linktor:WebhookSecret"];

    if (!LinktorClient.VerifyWebhookSignature(
        Encoding.UTF8.GetBytes(payload),
        signature,
        secret))
    {
        return Results.BadRequest(new { error = "Invalid signature" });
    }

    // Process the webhook...

    return Results.Ok(new { received = true });
});
```

## AI Features

### Completions

```csharp
using Linktor.Types;

// Create a completion
var response = await client.AI.Completions.CreateAsync(new CompletionInput
{
    Messages = new List<ChatMessage>
    {
        ChatMessage.System("You are a helpful assistant."),
        ChatMessage.User("What is the capital of France?")
    },
    Model = "gpt-4"
});

Console.WriteLine(response.Message.Content);
```

### Knowledge Bases

```csharp
// Query a knowledge base
var result = await client.KnowledgeBases.QueryAsync(
    "kb-id",
    "How do I reset my password?",
    topK: 5
);

foreach (var chunk in result.Chunks)
{
    Console.WriteLine($"Match: {chunk.Content}");
    Console.WriteLine($"Score: {chunk.Score}");
}
```

### Embeddings

```csharp
// Create embeddings
var embedding = await client.AI.Embeddings.EmbedAsync("Hello, world!");
Console.WriteLine($"Embedding dimension: {embedding.Count}");
```

## Dependency Injection

Register the client with dependency injection in ASP.NET Core:

```csharp
// In Program.cs or Startup.cs
builder.Services.AddSingleton(sp =>
{
    var config = sp.GetRequiredService<IConfiguration>();
    return new LinktorClient(new LinktorClientOptions
    {
        ApiKey = config["Linktor:ApiKey"],
        BaseUrl = config["Linktor:BaseUrl"] ?? "https://api.linktor.io"
    });
});

// In your controller or service
public class ChatService
{
    private readonly LinktorClient _client;

    public ChatService(LinktorClient client)
    {
        _client = client;
    }

    public async Task<Message> SendMessageAsync(string conversationId, string text)
    {
        return await _client.Conversations.SendMessageAsync(
            conversationId,
            new SendMessageInput { Text = text }
        );
    }
}
```

## Resources

The SDK provides access to all Linktor resources:

- `client.Auth` - Authentication
- `client.Conversations` - Conversations and messages
- `client.Contacts` - Contact management
- `client.Channels` - Channel configuration
- `client.Bots` - Bot management
- `client.AI` - AI completions and embeddings
- `client.KnowledgeBases` - Knowledge base operations
- `client.Flows` - Conversation flows

## API Reference

For complete API documentation, see the [.NET SDK Reference](https://github.com/linktor/linktor/tree/main/sdks/dotnet).
