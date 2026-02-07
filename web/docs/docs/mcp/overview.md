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

### Tools (30+)

| Category | Description |
|----------|-------------|
| **Conversations** | List, create, update, close, and assign conversations |
| **Messages** | Send messages, retrieve history, handle attachments |
| **Contacts** | Manage contacts, identities, and custom fields |
| **Channels** | List and manage WhatsApp, Telegram, and other channels |
| **Bots** | Configure and test AI bots |
| **Analytics** | Access metrics, reports, and performance data |
| **Knowledge** | Search and manage knowledge base articles |

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
| `LINKTOR_API_URL` | Linktor API base URL | `https://api.linktor.io` |
| `MCP_HTTP_PORT` | HTTP server port | `3001` |

## Next Steps

- [MCP Playground](./playground) - Test tools interactively in your browser
- [API Documentation](/api/overview) - Full REST API documentation
- [SDKs](/sdks/overview) - Client libraries for various languages
