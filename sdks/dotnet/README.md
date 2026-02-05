# Linktor .NET SDK

Official .NET SDK for the Linktor API.

## Installation

### Package Manager

```bash
Install-Package Linktor
```

### .NET CLI

```bash
dotnet add package Linktor
```

### PackageReference

```xml
<PackageReference Include="Linktor" Version="1.0.0" />
```

## Requirements

- .NET 8.0 or later

## Quick Start

```csharp
using Linktor;

// Initialize the client
var client = new LinktorClient(new LinktorClientOptions
{
    BaseUrl = "https://api.linktor.io",
    ApiKey = "your-api-key"
});

// List conversations
var conversations = await client.Conversations.ListAsync(new ListConversationsParams
{
    Limit = 10,
    Status = "open"
});

foreach (var conv in conversations.Data)
{
    Console.WriteLine($"Conversation: {conv.Id}");
}

// Send a message
var message = await client.Conversations.SendMessageAsync(conversationId, new SendMessageInput
{
    Text = "Hello from .NET!"
});
```

## Authentication

### API Key Authentication

```csharp
var client = new LinktorClient(new LinktorClientOptions
{
    ApiKey = "your-api-key"
});
```

### JWT Authentication

```csharp
var client = new LinktorClient(new LinktorClientOptions
{
    BaseUrl = "https://api.linktor.io"
});

// Login to get access token
var loginResponse = await client.Auth.LoginAsync("user@example.com", "password");
// Access token is automatically set on the client

// Get current user
var user = await client.Auth.GetCurrentUserAsync();

// Refresh token when needed
var refreshResponse = await client.Auth.RefreshTokenAsync(loginResponse.RefreshToken);
```

## Resources

### Conversations

```csharp
// List conversations
var conversations = await client.Conversations.ListAsync(new ListConversationsParams
{
    Limit = 20,
    Status = "open",
    ChannelId = "channel-id"
});

// Get a conversation
var conversation = await client.Conversations.GetAsync("conversation-id");

// Create a conversation
var newConversation = await client.Conversations.CreateAsync(new CreateConversationInput
{
    ChannelId = "channel-id",
    ContactId = "contact-id"
});

// Send a message
var message = await client.Conversations.SendMessageAsync("conversation-id", new SendMessageInput
{
    Text = "Hello!",
    QuickReplies = new List<QuickReply>
    {
        new() { Title = "Yes", Payload = "yes" },
        new() { Title = "No", Payload = "no" }
    }
});

// Get messages
var messages = await client.Conversations.GetMessagesAsync("conversation-id", new ListMessagesParams
{
    Limit = 50
});

// Assign conversation
await client.Conversations.AssignAsync("conversation-id", new AssignConversationInput
{
    UserId = "user-id"
});

// Resolve conversation
await client.Conversations.ResolveAsync("conversation-id");
```

### Contacts

```csharp
// List contacts
var contacts = await client.Contacts.ListAsync(new ListContactsParams
{
    Limit = 20,
    Search = "john"
});

// Create a contact
var contact = await client.Contacts.CreateAsync(new CreateContactInput
{
    Name = "John Doe",
    Email = "john@example.com",
    Phone = "+1234567890",
    Tags = new List<string> { "vip", "enterprise" },
    CustomFields = new Dictionary<string, object>
    {
        ["company"] = "Acme Inc",
        ["plan"] = "premium"
    }
});

// Search contacts
var searchResults = await client.Contacts.SearchAsync(new SearchContactsInput
{
    Query = "enterprise",
    Filters = new Dictionary<string, object>
    {
        ["tags"] = new[] { "vip" }
    }
});

// Merge contacts
var mergedContact = await client.Contacts.MergeAsync(new MergeContactsInput
{
    PrimaryId = "primary-contact-id",
    SecondaryIds = new List<string> { "duplicate-1", "duplicate-2" }
});
```

### Channels

```csharp
// List channels
var channels = await client.Channels.ListAsync(new ListChannelsParams
{
    Type = "whatsapp"
});

// Create a channel
var channel = await client.Channels.CreateAsync(new CreateChannelInput
{
    Name = "WhatsApp Business",
    Type = "whatsapp",
    Config = new Dictionary<string, object>
    {
        ["phoneNumberId"] = "your-phone-number-id",
        ["accessToken"] = "your-access-token"
    }
});

// Connect channel
await client.Channels.ConnectAsync("channel-id");

// Get channel status
var status = await client.Channels.GetStatusAsync("channel-id");
Console.WriteLine($"Connected: {status.IsConnected}");
```

### Bots

```csharp
// List bots
var bots = await client.Bots.ListAsync();

// Create a bot
var bot = await client.Bots.CreateAsync(new CreateBotInput
{
    Name = "Support Bot",
    AgentId = "agent-id",
    ChannelIds = new List<string> { "channel-1", "channel-2" }
});

// Start bot
await client.Bots.StartAsync("bot-id");

// Get bot status
var botStatus = await client.Bots.GetStatusAsync("bot-id");
Console.WriteLine($"Active conversations: {botStatus.ActiveConversations}");
```

### AI

#### Agents

```csharp
// List agents
var agents = await client.AI.Agents.ListAsync();

// Create an agent
var agent = await client.AI.Agents.CreateAsync(new CreateAgentInput
{
    Name = "Customer Support Agent",
    SystemPrompt = "You are a helpful customer support agent...",
    Model = "gpt-4",
    KnowledgeBaseIds = new List<string> { "kb-id" }
});

// Invoke agent
var response = await client.AI.Agents.InvokeAsync("agent-id", new AgentInvokeInput
{
    Message = "How do I reset my password?",
    ConversationId = "conversation-id"
});
Console.WriteLine(response.Response);
```

#### Completions

```csharp
// Simple completion
var completion = await client.AI.Completions.CreateAsync(new CompletionInput
{
    Prompt = "Translate to Spanish: Hello, how are you?",
    MaxTokens = 100
});

// Chat completion
var chat = await client.AI.Completions.ChatAsync(new ChatCompletionInput
{
    Messages = new List<ChatMessage>
    {
        new() { Role = "user", Content = "What is the capital of France?" }
    },
    MaxTokens = 100
});
```

#### Embeddings

```csharp
// Create embeddings
var embeddings = await client.AI.Embeddings.CreateAsync(new EmbeddingInput
{
    Texts = new List<string> { "Hello world", "How are you?" }
});

// Similarity search
var searchResults = await client.AI.Embeddings.SearchAsync(new SimilaritySearchInput
{
    Query = "password reset",
    KnowledgeBaseId = "kb-id",
    TopK = 5
});
```

### Knowledge Bases

```csharp
// Create a knowledge base
var kb = await client.KnowledgeBases.CreateAsync(new CreateKnowledgeBaseInput
{
    Name = "Product Documentation",
    Description = "Documentation for our products"
});

// Add a document
var doc = await client.KnowledgeBases.AddDocumentAsync("kb-id", new AddDocumentInput
{
    Title = "Getting Started Guide",
    Content = "This guide will help you get started..."
});

// Query the knowledge base
var results = await client.KnowledgeBases.QueryAsync("kb-id", new QueryInput
{
    Query = "how to get started",
    TopK = 5,
    IncludeContent = true
});

foreach (var result in results.Results)
{
    Console.WriteLine($"Score: {result.Score}, Content: {result.Content}");
}
```

### Flows

```csharp
// List flows
var flows = await client.Flows.ListAsync();

// Create a flow
var flow = await client.Flows.CreateAsync(new CreateFlowInput
{
    Name = "Welcome Flow",
    Nodes = new List<FlowNode>
    {
        new()
        {
            Id = "start",
            Type = "trigger",
            Data = new Dictionary<string, object> { ["event"] = "conversation.created" }
        },
        new()
        {
            Id = "welcome",
            Type = "send_message",
            Data = new Dictionary<string, object> { ["text"] = "Welcome!" }
        }
    },
    Edges = new List<FlowEdge>
    {
        new() { Source = "start", Target = "welcome" }
    }
});

// Validate flow
var validation = await client.Flows.ValidateAsync("flow-id");
if (!validation.IsValid)
{
    foreach (var error in validation.Errors ?? new List<FlowValidationError>())
    {
        Console.WriteLine($"Error in {error.NodeId}: {error.Message}");
    }
}

// Publish flow
await client.Flows.PublishAsync("flow-id");

// Execute flow manually
var execution = await client.Flows.ExecuteAsync("flow-id", new FlowExecuteInput
{
    ConversationId = "conversation-id",
    Variables = new Dictionary<string, object>
    {
        ["customVariable"] = "value"
    }
});
```

## Webhook Verification

```csharp
using Linktor.Utils;

// In your webhook handler
[HttpPost("webhook")]
public IActionResult HandleWebhook()
{
    var payload = await new StreamReader(Request.Body).ReadToEndAsync();
    var payloadBytes = Encoding.UTF8.GetBytes(payload);

    var headers = new Dictionary<string, string>
    {
        ["X-Linktor-Signature"] = Request.Headers["X-Linktor-Signature"],
        ["X-Linktor-Timestamp"] = Request.Headers["X-Linktor-Timestamp"]
    };

    try
    {
        var webhookEvent = LinktorClient.ConstructWebhookEvent(
            payloadBytes,
            headers,
            "your-webhook-secret",
            toleranceSeconds: 300
        );

        // Handle the event
        switch (webhookEvent.GetEventType())
        {
            case EventType.MessageReceived:
                // Handle new message
                break;
            case EventType.ConversationCreated:
                // Handle new conversation
                break;
        }

        return Ok();
    }
    catch (WebhookVerificationException ex)
    {
        return BadRequest(ex.Message);
    }
}

// Verify signature only
var isValid = LinktorClient.VerifyWebhookSignature(
    payloadBytes,
    signature,
    "your-webhook-secret"
);

// Compute signature (for testing)
var signature = LinktorClient.ComputeWebhookSignature(payloadBytes, "your-secret");
```

## Error Handling

```csharp
using Linktor.Utils;

try
{
    var conversation = await client.Conversations.GetAsync("non-existent-id");
}
catch (NotFoundException ex)
{
    Console.WriteLine($"Not found: {ex.Message}");
    Console.WriteLine($"Request ID: {ex.RequestId}");
}
catch (AuthenticationException ex)
{
    Console.WriteLine("Invalid credentials");
}
catch (RateLimitException ex)
{
    Console.WriteLine($"Rate limited. Retry after: {ex.RetryAfter} seconds");
}
catch (ValidationException ex)
{
    Console.WriteLine($"Validation error: {ex.Message}");
}
catch (LinktorException ex)
{
    Console.WriteLine($"API error: {ex.Message}");
    Console.WriteLine($"Status code: {ex.StatusCode}");
    Console.WriteLine($"Error code: {ex.ErrorCode}");
}
```

## Configuration Options

```csharp
var client = new LinktorClient(new LinktorClientOptions
{
    BaseUrl = "https://api.linktor.io",  // API base URL
    ApiKey = "your-api-key",              // API key for authentication
    AccessToken = "your-access-token",    // Or use access token directly
    TimeoutSeconds = 30,                  // Request timeout (default: 30)
    MaxRetries = 3                        // Max retry attempts (default: 3)
});
```

## Cancellation Support

All async methods support `CancellationToken`:

```csharp
using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(10));

try
{
    var conversations = await client.Conversations.ListAsync(ct: cts.Token);
}
catch (OperationCanceledException)
{
    Console.WriteLine("Request was cancelled");
}
```

## License

MIT License - see LICENSE file for details.
