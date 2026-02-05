# Linktor SDK for Python

Official Linktor SDK for Python applications.

## Installation

```bash
pip install linktor
```

## Quick Start

```python
from linktor import LinktorClient

client = LinktorClient(
    base_url="https://api.linktor.io",
    api_key="your-api-key",
)

# List conversations
conversations = client.conversations.list(limit=10)

# Send a message
client.conversations.send_text(conversation_id, "Hello!")

# Use AI
answer = client.ai.completions.complete("What is 2 + 2?")
print(answer)  # "4"
```

## Authentication

### API Key (Server-side)

```python
client = LinktorClient(api_key="your-api-key")
```

### Access Token (User sessions)

```python
client = LinktorClient(access_token="user-token")
```

### Login

```python
client = LinktorClient(base_url="https://api.linktor.io")
response = client.auth.login("user@example.com", "password")
print(response.user.name)
```

## Resources

### Conversations

```python
# List conversations
convs = client.conversations.list(status="open", limit=20)

# Get conversation
conv = client.conversations.get("conv-id")

# Send text message
client.conversations.send_text("conv-id", "Hello!")

# Send message with options
client.conversations.send_message("conv-id", text="Hello!", metadata={"key": "value"})

# Resolve conversation
client.conversations.resolve("conv-id")

# Assign to agent
client.conversations.assign("conv-id", "agent-id")
```

### Contacts

```python
# Create contact
contact = client.contacts.create(
    name="John Doe",
    email="john@example.com",
    phone="+1234567890",
)

# Search contacts
results = client.contacts.search("john")

# Update contact
client.contacts.update("contact-id", name="John Smith")

# Delete contact
client.contacts.delete("contact-id")
```

### Channels

```python
# List channels
channels = client.channels.list()

# Create channel
channel = client.channels.create(
    name="WhatsApp Business",
    type="whatsapp",
    config={
        "type": "whatsapp",
        "phoneNumberId": "123456789",
        "businessAccountId": "987654321",
        "accessToken": "token",
        "verifyToken": "verify",
    },
)

# Connect/disconnect channel
client.channels.connect("channel-id")
client.channels.disconnect("channel-id")
```

### Bots

```python
# Create AI bot
bot = client.bots.create(
    name="Support Bot",
    type="ai",
    config={
        "welcomeMessage": "Hello! How can I help?",
        "aiConfig": {
            "model": "gpt-4",
            "systemPrompt": "You are a helpful support agent.",
            "useKnowledgeBase": True,
        },
    },
)

# Activate/deactivate
client.bots.activate("bot-id")
client.bots.deactivate("bot-id")
```

### AI

```python
# Simple completion
answer = client.ai.completions.complete("What is the capital of France?")

# Chat completion
response = client.ai.completions.create(
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"},
    ],
    model="gpt-4",
)

# Invoke agent
result = client.ai.agents.invoke("agent-id", "How do I reset my password?")

# Create embeddings
embeddings = client.ai.embeddings.embed("Hello world")
```

### Knowledge Bases

```python
# Create knowledge base
kb = client.knowledge_bases.create(
    name="Product Documentation",
    description="Product docs and FAQs",
)

# Upload document
with open("faq.pdf", "rb") as f:
    doc = client.knowledge_bases.upload_document("kb-id", f.read(), "faq.pdf")

# Query knowledge base
results = client.knowledge_bases.query("kb-id", "How to reset password?", top_k=5)

# Simple search
texts = client.knowledge_bases.search("kb-id", "reset password")
```

### Flows

```python
# Create flow
flow = client.flows.create(
    name="Welcome Flow",
    nodes=[
        {"id": "start", "type": "start", "position": {"x": 0, "y": 0}, "data": {}},
        {"id": "welcome", "type": "message", "position": {"x": 200, "y": 0}, "data": {"messageContent": "Welcome!"}},
    ],
    edges=[{"id": "e1", "source": "start", "target": "welcome"}],
)

# Execute flow
execution = client.flows.execute("flow-id", conversation_id="conv-id")

# Activate/deactivate
client.flows.activate("flow-id")
client.flows.deactivate("flow-id")
```

### Analytics

```python
# Dashboard metrics
dashboard = client.analytics.get_dashboard(startDate="2024-01-01", endDate="2024-01-31")

# Realtime metrics
realtime = client.analytics.get_realtime()
```

## Webhooks

```python
from linktor import verify_webhook, construct_event

# Verify webhook
is_valid = verify_webhook(
    payload=request.body,
    headers=request.headers,
    secret="your-webhook-secret",
)

# Parse webhook event
event = construct_event(
    payload=request.body,
    headers=request.headers,
    secret="your-webhook-secret",
)

if event.type == "message.received":
    print("New message:", event.data)
```

### Flask Example

```python
from flask import Flask, request
from linktor import construct_event

app = Flask(__name__)

@app.route("/webhook", methods=["POST"])
def webhook():
    try:
        event = construct_event(
            request.data,
            dict(request.headers),
            "your-webhook-secret",
        )

        if event.type == "message.received":
            print("New message:", event.data)

        return "", 200
    except ValueError as e:
        return str(e), 400
```

## Error Handling

```python
from linktor import (
    LinktorError,
    AuthenticationError,
    RateLimitError,
    ValidationError,
)

try:
    conv = client.conversations.get("invalid-id")
except AuthenticationError:
    print("Please login again")
except RateLimitError as e:
    print(f"Rate limited. Retry after {e.retry_after}s")
except ValidationError as e:
    print(f"Validation error: {e.details}")
except LinktorError as e:
    print(f"Error [{e.code}]: {e.message}")
```

## Async Client

```python
import asyncio
from linktor import LinktorAsyncClient

async def main():
    async with LinktorAsyncClient(api_key="your-api-key") as client:
        # Use async methods
        pass

asyncio.run(main())
```

## Context Manager

```python
with LinktorClient(api_key="your-api-key") as client:
    convs = client.conversations.list()
# Client is automatically closed
```

## Configuration

```python
client = LinktorClient(
    base_url="https://api.linktor.io",
    api_key="your-api-key",
    timeout=30.0,  # Request timeout in seconds
    max_retries=3,  # Max retry attempts
    retry_delay=1.0,  # Initial retry delay in seconds
    headers={"X-Custom-Header": "value"},  # Custom headers
)
```

## License

MIT
