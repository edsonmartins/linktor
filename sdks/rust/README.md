# Linktor SDK for Rust

Official Linktor SDK for Rust applications.

## Installation

Add to your `Cargo.toml`:

```toml
[dependencies]
linktor = "1.0"
tokio = { version = "1", features = ["full"] }
```

## Quick Start

```rust
use linktor::{LinktorClient, types::*};

#[tokio::main]
async fn main() -> Result<(), linktor::Error> {
    let client = LinktorClient::builder()
        .base_url("https://api.linktor.io")
        .api_key("your-api-key")
        .build()?;

    // List conversations
    let convs = client.conversations()
        .list(Some(ListConversationsParams::new().status(ConversationStatus::Open)))
        .await?;
    println!("Found {} conversations", convs.data.len());

    // Send a message
    let msg = client.conversations().send_text("conv-id", "Hello!").await?;
    println!("Sent message: {}", msg.id);

    // Use AI
    let answer = client.ai().completions().complete("What is 2 + 2?").await?;
    println!("{}", answer); // "4"

    Ok(())
}
```

## Authentication

### API Key

```rust
let client = LinktorClient::builder()
    .api_key("your-api-key")
    .build()?;
```

### Access Token

```rust
let client = LinktorClient::builder()
    .access_token("user-token")
    .build()?;
```

### Login

```rust
let client = LinktorClient::builder()
    .base_url("https://api.linktor.io")
    .build()?;

let response = client.auth().login("user@example.com", "password").await?;
println!("{}", response.user.name);
```

## Resources

### Conversations

```rust
// List conversations
let convs = client.conversations()
    .list(Some(ListConversationsParams::new()
        .status(ConversationStatus::Open)
        .limit(10)))
    .await?;

// Get conversation
let conv = client.conversations().get("conv-id").await?;

// Send text message
let msg = client.conversations().send_text("conv-id", "Hello!").await?;

// Send message with options
let msg = client.conversations()
    .send_message("conv-id", SendMessageInput::text("Hello!"))
    .await?;

// Resolve conversation
client.conversations().resolve("conv-id").await?;

// Assign to agent
client.conversations().assign("conv-id", "agent-id").await?;
```

### Contacts

```rust
// Create contact
let contact = client.contacts()
    .create(CreateContactInput::new()
        .name("John Doe")
        .email("john@example.com")
        .phone("+1234567890"))
    .await?;

// Get contact
let contact = client.contacts().get("contact-id").await?;

// Update contact
let contact = client.contacts()
    .update("contact-id", UpdateContactInput {
        name: Some("John Smith".to_string()),
        ..Default::default()
    })
    .await?;

// Delete contact
client.contacts().delete("contact-id").await?;
```

### Channels

```rust
// List channels
let channels = client.channels().list(None).await?;

// Create channel
let channel = client.channels()
    .create(CreateChannelInput::new("WhatsApp Business", ChannelType::Whatsapp)
        .config(serde_json::json!({
            "phoneNumberId": "123456789",
            "businessAccountId": "987654321",
            "accessToken": "token"
        }).as_object().unwrap().clone().into_iter().collect()))
    .await?;

// Connect/disconnect
client.channels().connect("channel-id").await?;
client.channels().disconnect("channel-id").await?;
```

### Bots

```rust
// Create AI bot
let bot = client.bots()
    .create(CreateBotInput::new("Support Bot", BotType::Ai)
        .description("AI-powered support bot")
        .config(serde_json::json!({
            "welcomeMessage": "Hello! How can I help?",
            "aiConfig": {
                "model": "gpt-4",
                "systemPrompt": "You are a helpful support agent."
            }
        }).as_object().unwrap().clone().into_iter().collect()))
    .await?;
```

### AI

```rust
// Simple completion
let answer = client.ai().completions()
    .complete("What is the capital of France?")
    .await?;

// Chat with messages
let response = client.ai().completions()
    .chat(vec![
        ChatMessage::system("You are a helpful assistant."),
        ChatMessage::user("What is 2 + 2?"),
    ])
    .await?;
println!("{}", response.content().unwrap_or_default());

// Create embeddings
let embedding = client.ai().embeddings().embed("Hello world").await?;
```

### Knowledge Bases

```rust
// Create knowledge base
let kb = client.knowledge_bases()
    .create(CreateKnowledgeBaseInput::new("FAQ Knowledge Base")
        .description("Frequently asked questions"))
    .await?;

// Add document
let doc = client.knowledge_bases()
    .add_document("kb-id", AddDocumentInput::new("FAQ")
        .content("Question: How to reset password?\nAnswer: ..."))
    .await?;

// Query knowledge base
let result = client.knowledge_bases()
    .query("kb-id", "How to reset password?", 5)
    .await?;

for chunk in result.chunks {
    println!("{}", chunk.content);
}
```

### Flows

```rust
// Create flow
let flow = client.flows()
    .create(CreateFlowInput::new("Welcome Flow")
        .description("Onboarding flow for new users"))
    .await?;

// Execute flow
let execution = client.flows().execute("flow-id", "conv-id").await?;
```

## Webhooks

### Verify Signature

```rust
use linktor::webhook;
use std::collections::HashMap;

// Verify signature only
let is_valid = webhook::verify_signature(payload, signature, secret);

// Verify with headers and timestamp
let mut headers = HashMap::new();
headers.insert("X-Linktor-Signature".to_string(), signature.to_string());
headers.insert("X-Linktor-Timestamp".to_string(), timestamp.to_string());

let is_valid = webhook::verify(payload, &headers, secret, None);

// Parse and verify event
match webhook::construct_event(payload, &headers, secret, None) {
    Ok(event) => {
        println!("Event type: {}", event.event_type);
        match event.get_event_type() {
            Some(EventType::MessageReceived) => {
                // Handle new message
            }
            Some(EventType::ConversationCreated) => {
                // Handle new conversation
            }
            _ => {}
        }
    }
    Err(e) => {
        eprintln!("Webhook error: {}", e);
    }
}
```

### Axum Example

```rust
use axum::{extract::State, http::HeaderMap, body::Bytes};
use linktor::webhook;

async fn webhook_handler(
    headers: HeaderMap,
    body: Bytes,
) -> impl IntoResponse {
    let secret = std::env::var("LINKTOR_WEBHOOK_SECRET").unwrap();

    let headers_map: HashMap<String, String> = headers
        .iter()
        .filter_map(|(k, v)| {
            Some((k.to_string(), v.to_str().ok()?.to_string()))
        })
        .collect();

    match webhook::construct_event(&body, &headers_map, &secret, None) {
        Ok(event) => {
            println!("Received: {}", event.event_type);
            StatusCode::OK
        }
        Err(e) => {
            eprintln!("Error: {}", e);
            StatusCode::BAD_REQUEST
        }
    }
}
```

## Error Handling

```rust
use linktor::LinktorError;

match client.conversations().get("invalid-id").await {
    Ok(conv) => println!("Found: {}", conv.id),
    Err(LinktorError::NotFound { message, .. }) => {
        println!("Not found: {}", message);
    }
    Err(LinktorError::Authentication { message, .. }) => {
        println!("Auth failed: {}", message);
    }
    Err(LinktorError::RateLimit { retry_after, .. }) => {
        println!("Rate limited. Retry after {} seconds", retry_after);
    }
    Err(e) => {
        println!("Error: {}", e);
    }
}
```

## Configuration

```rust
let client = LinktorClient::builder()
    .base_url("https://api.linktor.io")
    .api_key("your-api-key")
    .timeout(30)          // Timeout in seconds
    .max_retries(3)       // Number of retries
    .build()?;
```

## Requirements

- Rust 1.70 or higher
- Tokio runtime

## License

MIT
