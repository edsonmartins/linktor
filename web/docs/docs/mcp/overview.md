---
title: MCP Overview
description: Model Context Protocol integration for Linktor
sidebar_position: 1
---

# Model Context Protocol (MCP)

Linktor provides a Model Context Protocol (MCP) server that enables AI assistants like Claude to interact with your Linktor instance programmatically.

## What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io) is an open protocol that standardizes how AI assistants connect to external data sources and tools. It enables:

- **Tools**: Executable functions that AI can call to perform actions
- **Resources**: Data sources that AI can read for context
- **Prompts**: Pre-built prompt templates for common tasks

## Linktor MCP Server

The `@linktor/mcp-server` package provides full access to Linktor's capabilities through MCP:

### Tools (51)

| Category | Description |
|----------|-------------|
| **Conversations** | List, create, update, close, and assign conversations |
| **Messages** | Send messages, send via channel, poll received channel messages, retrieve history, handle attachments |
| **Contacts** | Manage contacts, identities, and custom fields |
| **Channels** | Create, update, test, connect, disconnect, pair, and manage WhatsApp, Telegram, SMS, RCS, Instagram, Facebook, email, voice, and webchat channels |
| **Bots** | Configure and test AI bots |
| **Analytics** | Access metrics, reports, and performance data |
| **Knowledge** | Search and manage knowledge base articles |
| **VRE** | Render visual response templates and send rich image responses |

### Resources (6)

Static and dynamic resources for reading Linktor data:

- Active conversations
- Contact list
- Channel configuration
- Bot settings
- Analytics summaries
- Team members

### Prompts (4)

Pre-built prompt templates:

- **customer_support**: Customer support assistant
- **conversation_summary**: Summarize conversations
- **draft_response**: Draft customer responses
- **analyze_sentiment**: Sentiment analysis

## Installation

### NPM

```bash
npm install @linktor/mcp-server
```

### From Source

```bash
cd mcp/linktor-mcp-server
npm install
npm run build
```

## Usage

### Stdio Transport (Default)

For use with Claude Desktop and other MCP clients:

```bash
npx @linktor/mcp-server
```

### HTTP Transport

For browser-based playgrounds and REST-style access:

```bash
npm run start:http
```

## Configuration

The MCP server uses the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `LINKTOR_API_KEY` | API key for authentication | - |
| `LINKTOR_API_URL` | Linktor API base URL | `http://localhost:8081/api/v1` |
| `MCP_HTTP_PORT` | HTTP server port | `3001` |

## Channel Messaging Flow

The Linktor API stores messages inside conversations. For MCP clients, the fastest channel-oriented flow is:

1. Use `list_channels` or `get_channel` to select a channel.
2. Use `send_channel_message` with an existing `conversation_id`, or with `channel_id` and `contact_id` to create the conversation and send.
3. Use `receive_channel_messages` to poll conversations and messages populated by inbound webhooks for that channel.

Inbound delivery is still handled by each channel webhook. The MCP server reads the resulting conversations/messages; it is not a push subscription transport.

## Next Steps

- [MCP Playground](./playground) - Test tools interactively in your browser
- [API Documentation](/api/overview) - Full REST API documentation
- [SDKs](/sdks/overview) - Client libraries for various languages
