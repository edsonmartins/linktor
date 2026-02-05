# Linktor SDK for Go

Official Linktor SDK for Go applications.

## Installation

```bash
go get github.com/linktor/linktor-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/linktor/linktor-go"
)

func main() {
    client := linktor.NewClient(
        linktor.WithBaseURL("https://api.linktor.io"),
        linktor.WithAPIKey("your-api-key"),
    )

    ctx := context.Background()

    // List conversations
    convs, err := client.Conversations.List(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d conversations\n", len(convs.Data))

    // Send a message
    msg, err := client.Conversations.SendText(ctx, "conv-id", "Hello!")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Sent message: %s\n", msg.ID)

    // Use AI
    answer, err := client.AI.Completions.Complete(ctx, "What is 2 + 2?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(answer) // "4"
}
```

## Authentication

### API Key

```go
client := linktor.NewClient(
    linktor.WithAPIKey("your-api-key"),
)
```

### Access Token

```go
client := linktor.NewClient(
    linktor.WithAccessToken("user-token"),
)
```

### Login

```go
client := linktor.NewClient(
    linktor.WithBaseURL("https://api.linktor.io"),
)

ctx := context.Background()
response, err := client.Auth.Login(ctx, "user@example.com", "password")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response.User.Name)
```

## Resources

### Conversations

```go
ctx := context.Background()

// List conversations
convs, _ := client.Conversations.List(ctx, &types.ListConversationsParams{
    Status: types.ConversationStatusOpen,
})

// Get conversation
conv, _ := client.Conversations.Get(ctx, "conv-id")

// Send text message
msg, _ := client.Conversations.SendText(ctx, "conv-id", "Hello!")

// Send message with options
msg, _ := client.Conversations.SendMessage(ctx, "conv-id", &types.SendMessageInput{
    Text: "Hello!",
    Metadata: map[string]interface{}{"key": "value"},
})

// Resolve conversation
client.Conversations.Resolve(ctx, "conv-id")

// Assign to agent
client.Conversations.Assign(ctx, "conv-id", "agent-id")
```

### Contacts

```go
// Create contact
contact, _ := client.Contacts.Create(ctx, &types.CreateContactInput{
    Name:  "John Doe",
    Email: "john@example.com",
    Phone: "+1234567890",
})

// Get contact
contact, _ := client.Contacts.Get(ctx, "contact-id")

// Update contact
contact, _ := client.Contacts.Update(ctx, "contact-id", &types.UpdateContactInput{
    Name: "John Smith",
})

// Delete contact
client.Contacts.Delete(ctx, "contact-id")
```

### Channels

```go
// List channels
channels, _ := client.Channels.List(ctx, nil)

// Create channel
channel, _ := client.Channels.Create(ctx, &types.CreateChannelInput{
    Name: "WhatsApp Business",
    Type: types.ChannelTypeWhatsApp,
    Config: map[string]interface{}{
        "phoneNumberId":     "123456789",
        "businessAccountId": "987654321",
        "accessToken":       "token",
        "verifyToken":       "verify",
    },
})

// Connect/disconnect
client.Channels.Connect(ctx, "channel-id")
client.Channels.Disconnect(ctx, "channel-id")
```

### Bots

```go
// Create AI bot
bot, _ := client.Bots.Create(ctx, &types.CreateBotInput{
    Name: "Support Bot",
    Type: types.BotTypeAI,
    Config: map[string]interface{}{
        "welcomeMessage": "Hello! How can I help?",
        "aiConfig": map[string]interface{}{
            "model":            "gpt-4",
            "systemPrompt":     "You are a helpful support agent.",
            "useKnowledgeBase": true,
        },
    },
})
```

### AI

```go
// Simple completion
answer, _ := client.AI.Completions.Complete(ctx, "What is the capital of France?")

// Create embeddings
embeddings, _ := client.AI.Embeddings.Embed(ctx, "Hello world")
```

### Knowledge Bases

```go
// Query knowledge base
result, _ := client.KnowledgeBases.Query(ctx, "kb-id", "How to reset password?", 5)

for _, chunk := range result.Chunks {
    fmt.Println(chunk.Content)
}
```

### Flows

```go
// Execute flow
execution, _ := client.Flows.Execute(ctx, "flow-id", "conv-id")
```

### Analytics

```go
// Dashboard metrics
dashboard, _ := client.Analytics.GetDashboard(ctx)

// Realtime metrics
realtime, _ := client.Analytics.GetRealtime(ctx)
```

## Webhooks

```go
package main

import (
    "log"
    "net/http"

    "github.com/linktor/linktor-go"
)

func main() {
    handlers := map[string]func(*linktor.WebhookEvent){
        "message.received": func(event *linktor.WebhookEvent) {
            log.Printf("New message: %v", event.Data)
        },
        "conversation.created": func(event *linktor.WebhookEvent) {
            log.Printf("New conversation: %v", event.Data)
        },
    }

    http.HandleFunc("/webhook", linktor.WebhookHandler("your-webhook-secret", handlers))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Manual Verification

```go
import "github.com/linktor/linktor-go"

// Verify signature
isValid := linktor.VerifyWebhookSignature(payload, signature, secret)

// Verify with timestamp
isValid := linktor.VerifyWebhook(payload, headers, secret, 300)

// Parse event
event, err := linktor.ConstructEvent(payload, headers, secret, 0)
if err != nil {
    log.Fatal(err)
}
```

## Error Handling

```go
conv, err := client.Conversations.Get(ctx, "invalid-id")
if err != nil {
    if apiErr, ok := err.(*linktor.Error); ok {
        switch apiErr.Status {
        case 401:
            log.Println("Authentication failed")
        case 404:
            log.Println("Not found")
        case 429:
            log.Println("Rate limited")
        default:
            log.Printf("Error [%s]: %s", apiErr.Code, apiErr.Message)
        }
    }
}
```

## Configuration

```go
client := linktor.NewClient(
    linktor.WithBaseURL("https://api.linktor.io"),
    linktor.WithAPIKey("your-api-key"),
    linktor.WithTimeout(30 * time.Second),
    linktor.WithMaxRetries(3),
)
```

## License

MIT
