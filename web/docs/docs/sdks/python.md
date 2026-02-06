---
sidebar_position: 3
title: Python SDK
---

# Python SDK

The official Python SDK for Linktor provides both synchronous and asynchronous clients with full type hints and IDE support.

## Installation

Install the SDK using pip:

```bash
pip install linktor
```

Or with poetry:

```bash
poetry add linktor
```

## Quick Start

### Initialize the Client

```python
from linktor import LinktorClient

# Synchronous client
client = LinktorClient(
    api_key="your-api-key",
    # Optional configuration
    base_url="https://api.linktor.io",
    timeout=30.0,
    max_retries=3,
)
```

### Send a Message

```python
# Send a message to a conversation
message = client.conversations.send_text(
    conversation_id="conversation-id",
    text="Hello from Python!"
)

print(f"Message sent: {message.id}")
```

### List Conversations

```python
# Get all conversations with pagination
conversations = client.conversations.list(
    limit=20,
    status="open"
)

for conversation in conversations.data:
    print(f"{conversation.id}: {conversation.contact.name}")
```

### Work with Contacts

```python
# Create a new contact
contact = client.contacts.create(
    name="John Doe",
    email="john@example.com",
    phone="+1234567890",
    metadata={
        "customer_id": "cust_123"
    }
)

# Search contacts
results = client.contacts.search("john", limit=10)
```

## Async Client

For asynchronous operations, use the `LinktorAsyncClient`:

```python
import asyncio
from linktor import LinktorAsyncClient

async def main():
    async with LinktorAsyncClient(api_key="your-api-key") as client:
        # All operations are async
        conversations = await client.conversations.list(limit=10)

        for conv in conversations.data:
            print(conv.id)

asyncio.run(main())
```

## Real-time Updates (WebSocket)

Connect to the WebSocket for real-time message updates.

```python
import asyncio
from linktor.websocket import LinktorWebSocket

async def main():
    ws = LinktorWebSocket(
        url="wss://api.linktor.io/ws",
        api_key="your-api-key"
    )

    # Connect to WebSocket
    await ws.connect()

    # Subscribe to a conversation
    await ws.subscribe("conversation-id")

    # Handle incoming messages
    @ws.on_message
    async def handle_message(event):
        print(f"New message: {event.message.text}")
        print(f"From conversation: {event.conversation_id}")

    # Handle message status updates
    @ws.on_message_status
    async def handle_status(event):
        print(f"Message {event.message_id} status: {event.status}")

    # Handle typing indicators
    @ws.on_typing
    async def handle_typing(event):
        if event.is_typing:
            print(f"User {event.user_id} is typing...")

    # Send typing indicator
    await ws.send_typing("conversation-id", True)

    # Keep the connection alive
    await asyncio.sleep(3600)

    # Cleanup
    await ws.disconnect()

asyncio.run(main())
```

## Error Handling

The SDK provides specific exception classes for different error types.

```python
from linktor import LinktorClient
from linktor.utils.errors import (
    LinktorError,
    AuthenticationError,
    NotFoundError,
    RateLimitError,
    ValidationError,
    ServerError,
)

client = LinktorClient(api_key="your-api-key")

try:
    conversation = client.conversations.get("invalid-id")
except NotFoundError as e:
    print(f"Conversation not found: {e.message}")
except AuthenticationError as e:
    print(f"Invalid API key: {e.message}")
except RateLimitError as e:
    print(f"Rate limited. Retry after {e.retry_after} seconds")
except ValidationError as e:
    print(f"Invalid request: {e.details}")
except LinktorError as e:
    print(f"API error [{e.code}]: {e.message}")
    print(f"Request ID: {e.request_id}")
```

### Exception Types

| Exception | Status Code | Description |
|-----------|-------------|-------------|
| `AuthenticationError` | 401 | Invalid or missing API key |
| `AuthorizationError` | 403 | Insufficient permissions |
| `NotFoundError` | 404 | Resource not found |
| `ValidationError` | 400 | Invalid request parameters |
| `RateLimitError` | 429 | Too many requests |
| `ConflictError` | 409 | Resource conflict |
| `ServerError` | 5xx | Server-side error |
| `NetworkError` | - | Network connectivity issue |
| `TimeoutError` | 408 | Request timeout |
| `WebSocketError` | - | WebSocket connection error |

## Webhook Verification

Verify incoming webhooks from Linktor.

```python
from flask import Flask, request
from linktor.utils.webhook import verify_webhook, construct_event

app = Flask(__name__)

@app.route("/webhook", methods=["POST"])
def webhook():
    payload = request.get_data()
    signature = request.headers.get("X-Linktor-Signature")
    timestamp = request.headers.get("X-Linktor-Timestamp")

    webhook_secret = "your-webhook-secret"

    try:
        # Verify and parse the webhook event
        event = construct_event(
            payload=payload,
            signature=signature,
            timestamp=timestamp,
            secret=webhook_secret
        )

        # Handle different event types
        if event.type == "message.received":
            print(f"New message received: {event.data}")
        elif event.type == "conversation.created":
            print(f"New conversation: {event.data}")

        return {"received": True}
    except Exception as e:
        print(f"Webhook verification failed: {e}")
        return {"error": "Invalid signature"}, 400
```

## AI Features

### Completions

```python
# Simple completion
response = client.ai.completions.create(
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "What is the capital of France?"},
    ],
    model="gpt-4"
)

print(response["message"]["content"])

# Simple helper method
answer = client.ai.completions.complete("What is 2 + 2?")
print(answer)  # "4"
```

### Knowledge Bases

```python
# Query a knowledge base
results = client.knowledge_bases.query(
    id="kb-id",
    query="How do I reset my password?",
    top_k=5
)

for chunk in results.get("chunks", []):
    print(f"Match: {chunk['content']}")
    print(f"Score: {chunk['score']}")

# Simple search helper
texts = client.knowledge_bases.search("kb-id", "password reset")
for text in texts:
    print(text)
```

### Embeddings

```python
# Create embeddings
embedding = client.ai.embeddings.embed("Hello, world!")
print(f"Embedding dimension: {len(embedding)}")
```

## Context Manager

The client supports context managers for automatic cleanup:

```python
from linktor import LinktorClient

# Synchronous
with LinktorClient(api_key="your-api-key") as client:
    conversations = client.conversations.list()
# Connection is automatically closed

# Asynchronous
async with LinktorAsyncClient(api_key="your-api-key") as client:
    conversations = await client.conversations.list()
# Connection is automatically closed
```

## Resources

The SDK provides access to all Linktor resources:

- `client.auth` - Authentication
- `client.conversations` - Conversations and messages
- `client.contacts` - Contact management
- `client.channels` - Channel configuration
- `client.bots` - Bot management
- `client.ai` - AI completions and embeddings
- `client.knowledge_bases` - Knowledge base operations
- `client.flows` - Conversation flows
- `client.analytics` - Analytics and metrics

## API Reference

For complete API documentation, see the [Python SDK Reference](https://github.com/linktor/linktor/tree/main/sdks/python).
