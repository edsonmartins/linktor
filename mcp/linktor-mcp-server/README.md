# Linktor MCP Server

MCP (Model Context Protocol) Server for Linktor - enabling AI assistants like Claude to interact with the Linktor omnichannel conversation platform.

## Overview

The Linktor MCP Server allows AI assistants to:
- Manage customer conversations across multiple channels
- Send and receive messages
- Access contact information
- Configure and test AI bots
- Query analytics and knowledge bases

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Claude/AI     │────▶│  Linktor MCP    │────▶│  Linktor API    │
│   Assistant     │◀────│     Server      │◀────│  (Go Backend)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ▼
                        stdio transport
```

## Features

- **26 Tools** for managing conversations, messages, contacts, channels, bots, and analytics
- **6 Resources** for reading platform data
- **4 Prompts** for common customer support tasks
- Full TypeScript support with type definitions
- HTTP client with retry logic and error handling
- Works with Claude Desktop and other MCP-compatible clients

## Installation

```bash
# Using npm
npm install @linktor/mcp-server

# Using npx (no installation required)
npx @linktor/mcp-server
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `LINKTOR_API_URL` | Linktor API URL (default: `http://localhost:8080/api/v1`) | No |
| `LINKTOR_API_KEY` | API Key for authentication | Yes* |
| `LINKTOR_ACCESS_TOKEN` | JWT Access Token (alternative to API Key) | Yes* |
| `LINKTOR_TIMEOUT` | Request timeout in ms (default: 30000) | No |
| `LINKTOR_MAX_RETRIES` | Max retry attempts (default: 3) | No |
| `LINKTOR_RETRY_DELAY` | Delay between retries in ms (default: 1000) | No |

*Either `LINKTOR_API_KEY` or `LINKTOR_ACCESS_TOKEN` is required.

### Claude Desktop Configuration

Add to your Claude Desktop config file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "linktor": {
      "command": "npx",
      "args": ["-y", "@linktor/mcp-server"],
      "env": {
        "LINKTOR_API_URL": "https://api.linktor.io",
        "LINKTOR_API_KEY": "your-api-key"
      }
    }
  }
}
```

## Project Structure

```
linktor-mcp-server/
├── package.json
├── tsconfig.json
├── README.md
├── bin/
│   └── linktor-mcp.js          # CLI executable
├── examples/
│   └── claude-desktop.json     # Example Claude Desktop config
└── src/
    ├── index.ts                # Entry point
    ├── server.ts               # MCP Server setup
    ├── config.ts               # Configuration with Zod validation
    ├── api/
    │   ├── client.ts           # HTTP client for Linktor API
    │   └── types.ts            # TypeScript type definitions
    ├── tools/
    │   ├── index.ts            # Tool exports
    │   ├── conversations.ts    # Conversation management
    │   ├── messages.ts         # Message operations
    │   ├── contacts.ts         # Contact management
    │   ├── channels.ts         # Channel operations
    │   ├── bots.ts             # Bot management
    │   ├── analytics.ts        # Analytics queries
    │   └── knowledge.ts        # Knowledge base search
    ├── resources/
    │   ├── index.ts            # Resource exports
    │   └── handlers.ts         # Resource handlers
    └── prompts/
        ├── index.ts            # Prompt exports
        └── templates.ts        # Prompt templates
```

## Available Tools

### Conversations (8 tools)

| Tool | Description |
|------|-------------|
| `list_conversations` | List conversations with filters (status, channel, assigned user) |
| `get_conversation` | Get detailed conversation information |
| `create_conversation` | Create a new conversation with a contact |
| `assign_conversation` | Assign conversation to a user/agent |
| `unassign_conversation` | Remove assignment from conversation |
| `resolve_conversation` | Mark conversation as resolved |
| `reopen_conversation` | Reopen a resolved/closed conversation |
| `close_conversation` | Close conversation permanently |

### Messages (3 tools)

| Tool | Description |
|------|-------------|
| `list_messages` | List messages in a conversation with pagination |
| `get_message` | Get specific message details |
| `send_message` | Send message (text, image, document, etc.) |

### Contacts (5 tools)

| Tool | Description |
|------|-------------|
| `list_contacts` | List contacts with search and tag filters |
| `get_contact` | Get contact details with channel identities |
| `create_contact` | Create new contact with custom fields |
| `update_contact` | Update contact information |
| `delete_contact` | Delete a contact |

### Channels (4 tools)

| Tool | Description |
|------|-------------|
| `list_channels` | List channels (WhatsApp, Telegram, etc.) |
| `get_channel` | Get channel configuration and status |
| `connect_channel` | Connect/activate a channel |
| `disconnect_channel` | Disconnect a channel |

### Bots (5 tools)

| Tool | Description |
|------|-------------|
| `list_bots` | List AI bots with status filter |
| `get_bot` | Get bot configuration and rules |
| `activate_bot` | Activate a bot |
| `deactivate_bot` | Deactivate a bot |
| `test_bot` | Test bot with a sample message |

### Analytics & Knowledge Base (5 tools)

| Tool | Description |
|------|-------------|
| `get_analytics_summary` | Get analytics for date range |
| `get_conversation_stats` | Get conversation statistics |
| `search_knowledge` | Semantic search in knowledge base |
| `list_knowledge_documents` | List KB documents |
| `get_knowledge_document` | Get document content |

## Resources

| URI | Description |
|-----|-------------|
| `linktor://conversations` | Active conversations list |
| `linktor://conversations/{id}` | Specific conversation details |
| `linktor://contacts` | All contacts |
| `linktor://contacts/{id}` | Specific contact details |
| `linktor://channels` | Configured channels |
| `linktor://channels/{id}` | Specific channel details |
| `linktor://bots` | AI bots list |
| `linktor://bots/{id}` | Specific bot details |
| `linktor://users` | Team members/agents |
| `linktor://analytics/summary` | Analytics summary (last 30 days) |

## Prompts

| Prompt | Description | Arguments |
|--------|-------------|-----------|
| `customer_support` | Customer support assistant | `company_name`, `tone?` |
| `conversation_summary` | Summarize a conversation | `conversation_id` |
| `draft_response` | Draft a customer response | `message`, `context?` |
| `analyze_sentiment` | Analyze text sentiment | `text` |

## Usage Examples

### List Open Conversations
```
"Show me all open conversations"
"List conversations assigned to user123"
"Get conversations from the WhatsApp channel"
```

### Send a Message
```
"Send 'Hello, how can I help you?' to conversation abc123"
"Reply to the customer with our refund policy"
```

### Manage Contacts
```
"Create a contact for John Doe with email john@example.com"
"Find contacts tagged as 'VIP'"
"Update the phone number for contact xyz789"
```

### Bot Operations
```
"List all active bots"
"Test the support bot with 'How do I reset my password?'"
"Deactivate bot abc123"
```

### Analytics
```
"Show me conversation statistics for the last week"
"Get analytics summary from January 1st to January 31st"
```

### Knowledge Base
```
"Search the knowledge base for refund policy"
"List all documents in knowledge base kb123"
```

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run in development mode (watch)
npm run dev

# Type check
npm run typecheck

# Start the server
npm run start
```

## Supported Channels

The Linktor platform supports the following channels:
- WebChat
- WhatsApp (Business & Official API)
- Telegram
- SMS
- RCS
- Instagram
- Facebook Messenger
- Email
- Voice

## Error Handling

The MCP Server handles errors gracefully and returns structured error responses:

```json
{
  "error": true,
  "code": "NOT_FOUND",
  "message": "Conversation not found",
  "details": {}
}
```

## License

MIT

## Links

- [Linktor Documentation](https://docs.linktor.io)
- [MCP Protocol Specification](https://modelcontextprotocol.io)
- [Claude Desktop](https://claude.ai/download)
