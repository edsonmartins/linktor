---
sidebar_position: 4
title: Go SDK
---

# Go SDK

The official Go SDK for Linktor provides a idiomatic Go interface with context support, functional options, and strong type safety.

## Installation

Install the SDK using go get:

```bash
go get github.com/linktor/linktor-go
```

## Quick Start

### Initialize the Client

```go
package main

import (
    "context"
    "fmt"
    "os"

    linktor "github.com/linktor/linktor-go"
)

func main() {
    // Create client with options
    client := linktor.NewClient(
        linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
        // Optional configuration
        linktor.WithBaseURL("https://api.linktor.io"),
        linktor.WithTimeout(30 * time.Second),
        linktor.WithMaxRetries(3),
    )
}
```

### Send a Message

```go
ctx := context.Background()

// Send a message to a conversation
message, err := client.Conversations.SendText(ctx, "conversation-id", "Hello from Go!")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message sent: %s\n", message.ID)
```

### List Conversations

```go
ctx := context.Background()

// Get all conversations with pagination
conversations, err := client.Conversations.List(ctx, &types.ListConversationsParams{
    Limit:  20,
    Status: "open",
})
if err != nil {
    log.Fatal(err)
}

for _, conv := range conversations.Data {
    fmt.Printf("%s: %s\n", conv.ID, conv.Contact.Name)
}
```

### Work with Contacts

```go
ctx := context.Background()

// Create a new contact
contact, err := client.Contacts.Create(ctx, &types.CreateContactInput{
    Name:  "John Doe",
    Email: "john@example.com",
    Phone: "+1234567890",
    Metadata: map[string]interface{}{
        "customerId": "cust_123",
    },
})
if err != nil {
    log.Fatal(err)
}

// List contacts
results, err := client.Contacts.List(ctx, &types.ListContactsParams{
    Search: "john",
    Limit:  10,
})
```

## Real-time Updates (WebSocket)

Connect to the WebSocket for real-time message updates.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    linktor "github.com/linktor/linktor-go"
    "github.com/linktor/linktor-go/websocket"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create WebSocket client
    ws := websocket.NewClient(
        "wss://api.linktor.io/ws",
        websocket.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
        websocket.WithAutoReconnect(true),
    )

    // Connect to WebSocket
    if err := ws.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer ws.Disconnect()

    // Subscribe to a conversation
    ws.Subscribe("conversation-id")

    // Handle incoming messages
    ws.OnMessage(func(event *websocket.MessageEvent) {
        fmt.Printf("New message: %s\n", event.Message.Text)
        fmt.Printf("From conversation: %s\n", event.ConversationID)
    })

    // Handle message status updates
    ws.OnMessageStatus(func(event *websocket.MessageStatusEvent) {
        fmt.Printf("Message %s status: %s\n", event.MessageID, event.Status)
    })

    // Handle typing indicators
    ws.OnTyping(func(event *websocket.TypingEvent) {
        if event.IsTyping {
            fmt.Printf("User %s is typing...\n", event.UserID)
        }
    })

    // Send typing indicator
    ws.SendTyping("conversation-id", true)

    // Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
}
```

## Error Handling

The SDK provides a structured error type for handling API errors.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"

    linktor "github.com/linktor/linktor-go"
)

func main() {
    client := linktor.NewClient(
        linktor.WithAPIKey("your-api-key"),
    )

    ctx := context.Background()

    conversation, err := client.Conversations.Get(ctx, "invalid-id")
    if err != nil {
        var apiErr *linktor.Error
        if errors.As(err, &apiErr) {
            switch apiErr.Status {
            case 404:
                fmt.Println("Conversation not found")
            case 401:
                fmt.Println("Invalid API key")
            case 429:
                fmt.Printf("Rate limited. Retry after checking headers\n")
            case 400:
                fmt.Printf("Invalid request: %s\n", apiErr.Message)
            default:
                fmt.Printf("API error [%s]: %s\n", apiErr.Code, apiErr.Message)
                fmt.Printf("Request ID: %s\n", apiErr.RequestID)
            }
        } else {
            log.Fatal(err)
        }
        return
    }

    fmt.Printf("Conversation: %s\n", conversation.ID)
}
```

### Error Structure

```go
type Error struct {
    Code      string                 `json:"code"`
    Message   string                 `json:"message"`
    Status    int                    `json:"status"`
    RequestID string                 `json:"requestId,omitempty"`
    Details   map[string]interface{} `json:"details,omitempty"`
}
```

## Webhook Verification

Verify incoming webhooks from Linktor.

```go
package main

import (
    "encoding/json"
    "io"
    "net/http"

    linktor "github.com/linktor/linktor-go"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    payload, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }

    signature := r.Header.Get("X-Linktor-Signature")
    secret := "your-webhook-secret"

    // Verify the webhook signature
    if !linktor.VerifyWebhook(payload, signature, secret) {
        http.Error(w, "Invalid signature", http.StatusBadRequest)
        return
    }

    // Parse the event
    var event map[string]interface{}
    if err := json.Unmarshal(payload, &event); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Handle different event types
    switch event["type"].(string) {
    case "message.received":
        // Handle new message
    case "conversation.created":
        // Handle new conversation
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]bool{"received": true})
}

func main() {
    http.HandleFunc("/webhook", webhookHandler)
    http.ListenAndServe(":8080", nil)
}
```

## AI Features

### Completions

```go
ctx := context.Background()

// Simple completion
response, err := client.AI.Completions.Complete(ctx, "What is the capital of France?")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response)  // "Paris"
```

### Knowledge Bases

```go
ctx := context.Background()

// Query a knowledge base
result, err := client.KnowledgeBases.Query(ctx, "kb-id", "How do I reset my password?", 5)
if err != nil {
    log.Fatal(err)
}

for _, chunk := range result.Chunks {
    fmt.Printf("Match: %s\n", chunk.Content)
    fmt.Printf("Score: %f\n", chunk.Score)
}
```

### Embeddings

```go
ctx := context.Background()

// Create embeddings
embedding, err := client.AI.Embeddings.Embed(ctx, "Hello, world!")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Embedding dimension: %d\n", len(embedding))
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
- `client.Analytics` - Analytics and metrics

## API Reference

For complete API documentation, see the [Go SDK Reference](https://github.com/linktor/linktor/tree/main/sdks/go).
