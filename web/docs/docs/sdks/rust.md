---
sidebar_position: 6
title: Rust SDK
---

# Rust SDK

The official Rust SDK for Linktor provides an async-first, type-safe interface with builder patterns and comprehensive error handling using `thiserror`.

## Installation

Add the dependency to your `Cargo.toml`:

```toml
[dependencies]
linktor = "1.0"
tokio = { version = "1", features = ["full"] }
```

## Quick Start

### Initialize the Client

```rust
use linktor::LinktorClient;

#[tokio::main]
async fn main() -> linktor::Result<()> {
    // Create client with builder pattern
    let client = LinktorClient::builder()
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        // Optional configuration
        .base_url("https://api.linktor.io")
        .timeout(30)
        .max_retries(3)
        .build()?;

    Ok(())
}
```

### Send a Message

```rust
use linktor::LinktorClient;

#[tokio::main]
async fn main() -> linktor::Result<()> {
    let client = LinktorClient::builder()
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    // Send a message to a conversation
    let message = client
        .conversations()
        .send_text("conversation-id", "Hello from Rust!")
        .await?;

    println!("Message sent: {}", message.id);

    Ok(())
}
```

### List Conversations

```rust
use linktor::{LinktorClient, types::ListConversationsParams};

#[tokio::main]
async fn main() -> linktor::Result<()> {
    let client = LinktorClient::builder()
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    // Get all conversations with pagination
    let params = ListConversationsParams::default()
        .limit(20)
        .status("open");

    let conversations = client
        .conversations()
        .list(Some(params))
        .await?;

    for conv in conversations.data {
        println!("{}: {}", conv.id, conv.contact.name);
    }

    Ok(())
}
```

### Work with Contacts

```rust
use linktor::{LinktorClient, types::CreateContactInput};
use std::collections::HashMap;

#[tokio::main]
async fn main() -> linktor::Result<()> {
    let client = LinktorClient::builder()
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    // Create a new contact
    let mut metadata = HashMap::new();
    metadata.insert("customer_id".to_string(), "cust_123".into());

    let contact = client.contacts().create(
        CreateContactInput::new("John Doe")
            .email("john@example.com")
            .phone("+1234567890")
            .metadata(metadata)
    ).await?;

    // List contacts
    let results = client.contacts()
        .list(Some(ListContactsParams::default()
            .search("john")
            .limit(10)))
        .await?;

    Ok(())
}
```

## Real-time Updates (WebSocket)

Connect to the WebSocket for real-time message updates.

```rust
use linktor::websocket::{LinktorWebSocket, WebSocketConfig};
use tokio::signal;

#[tokio::main]
async fn main() -> linktor::Result<()> {
    // Create WebSocket client
    let ws = LinktorWebSocket::new(WebSocketConfig {
        url: "wss://api.linktor.io/ws".to_string(),
        api_key: Some(std::env::var("LINKTOR_API_KEY")?),
        access_token: None,
        auto_reconnect: true,
        reconnect_interval: 5000,
        max_reconnect_attempts: 10,
    });

    // Connect to WebSocket
    ws.connect().await?;

    // Subscribe to a conversation
    ws.subscribe("conversation-id").await;

    // Handle incoming messages
    let ws_clone = ws.clone();
    tokio::spawn(async move {
        ws_clone.on_message(|event| {
            println!("New message: {}", event.message.text);
            println!("From conversation: {}", event.conversation_id);
        }).await;
    });

    // Handle message status updates
    let ws_clone = ws.clone();
    tokio::spawn(async move {
        ws_clone.on_message_status(|event| {
            println!("Message {} status: {}", event.message_id, event.status);
        }).await;
    });

    // Handle typing indicators
    let ws_clone = ws.clone();
    tokio::spawn(async move {
        ws_clone.on_typing(|event| {
            if event.is_typing {
                println!("User {} is typing...", event.user_id);
            }
        }).await;
    });

    // Send typing indicator
    ws.send_typing("conversation-id", true).await;

    // Wait for shutdown signal
    signal::ctrl_c().await?;

    // Cleanup
    ws.disconnect().await;

    Ok(())
}
```

## Error Handling

The SDK uses the `thiserror` crate for comprehensive error handling.

```rust
use linktor::{LinktorClient, LinktorError};

#[tokio::main]
async fn main() {
    let client = LinktorClient::builder()
        .api_key("your-api-key")
        .build()
        .expect("Failed to create client");

    match client.conversations().get("invalid-id").await {
        Ok(conversation) => {
            println!("Conversation: {}", conversation.id);
        }
        Err(error) => {
            match error {
                LinktorError::NotFound { message, request_id } => {
                    println!("Conversation not found: {}", message);
                }
                LinktorError::Authentication { message, .. } => {
                    println!("Invalid API key: {}", message);
                }
                LinktorError::RateLimit { retry_after, message, .. } => {
                    println!("Rate limited. Retry after {} seconds: {}",
                             retry_after, message);
                }
                LinktorError::Validation { message, .. } => {
                    println!("Invalid request: {}", message);
                }
                LinktorError::Network(e) => {
                    println!("Network error: {}", e);
                }
                LinktorError::Server { message, request_id } => {
                    println!("Server error: {}", message);
                    if let Some(id) = request_id {
                        println!("Request ID: {}", id);
                    }
                }
                _ => {
                    println!("Error: {}", error);
                }
            }
        }
    }
}
```

### Error Types

```rust
#[derive(Error, Debug)]
pub enum LinktorError {
    #[error("Authentication failed: {message}")]
    Authentication { message: String, request_id: Option<String> },

    #[error("Authorization failed: {message}")]
    Authorization { message: String, request_id: Option<String> },

    #[error("Resource not found: {message}")]
    NotFound { message: String, request_id: Option<String> },

    #[error("Validation error: {message}")]
    Validation { message: String, request_id: Option<String> },

    #[error("Rate limit exceeded. Retry after {retry_after} seconds")]
    RateLimit { retry_after: u64, message: String, request_id: Option<String> },

    #[error("Server error: {message}")]
    Server { message: String, request_id: Option<String> },

    #[error("Network error: {0}")]
    Network(#[from] reqwest::Error),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("Webhook verification failed: {message}")]
    WebhookVerification { message: String },

    #[error("WebSocket error: {message}")]
    WebSocket { message: String },
}
```

## Webhook Verification

Verify incoming webhooks from Linktor.

```rust
use linktor::webhook;
use axum::{
    extract::Json,
    http::{HeaderMap, StatusCode},
    response::IntoResponse,
};
use serde_json::Value;

async fn webhook_handler(
    headers: HeaderMap,
    body: String,
) -> impl IntoResponse {
    let signature = headers
        .get("X-Linktor-Signature")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");

    let secret = std::env::var("WEBHOOK_SECRET").unwrap();

    // Verify the webhook signature
    if !webhook::verify_signature(&body, signature, &secret) {
        return (StatusCode::BAD_REQUEST, "Invalid signature").into_response();
    }

    // Parse and handle the event
    let event: Value = serde_json::from_str(&body).unwrap();

    match event["type"].as_str() {
        Some("message.received") => {
            // Handle new message
        }
        Some("conversation.created") => {
            // Handle new conversation
        }
        _ => {}
    }

    (StatusCode::OK, Json(serde_json::json!({"received": true}))).into_response()
}
```

## AI Features

### Completions

```rust
use linktor::types::{CompletionInput, ChatMessage};

#[tokio::main]
async fn main() -> linktor::Result<()> {
    let client = LinktorClient::builder()
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    // Simple completion
    let response = client.ai().completions()
        .complete("What is the capital of France?")
        .await?;

    println!("{}", response);  // "Paris"

    Ok(())
}
```

### Knowledge Bases

```rust
#[tokio::main]
async fn main() -> linktor::Result<()> {
    let client = LinktorClient::builder()
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    // Query a knowledge base
    let result = client.knowledge_bases()
        .query("kb-id", "How do I reset my password?", 5)
        .await?;

    for chunk in result.chunks {
        println!("Match: {}", chunk.content);
        println!("Score: {}", chunk.score);
    }

    Ok(())
}
```

### Embeddings

```rust
// Create embeddings
let embedding = client.ai().embeddings()
    .embed("Hello, world!")
    .await?;

println!("Embedding dimension: {}", embedding.len());
```

## Resources

The SDK provides access to all Linktor resources:

- `client.auth()` - Authentication
- `client.conversations()` - Conversations and messages
- `client.contacts()` - Contact management
- `client.channels()` - Channel configuration
- `client.bots()` - Bot management
- `client.ai()` - AI completions and embeddings
- `client.knowledge_bases()` - Knowledge base operations
- `client.flows()` - Conversation flows

## API Reference

For complete API documentation, see the [Rust SDK Reference](https://github.com/linktor/linktor/tree/main/sdks/rust).
