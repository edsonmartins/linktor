---
slug: /
sidebar_position: 1
title: Introduction
---

# Welcome to Linktor

**Linktor** is an open-source omnichannel conversation platform powered by AI. Connect all your messaging channels in one place, build intelligent chatbots, and engage customers everywhere.

## Features

- **10+ Channels**: WhatsApp, Telegram, SMS, Email, Voice, WebChat, Facebook, Instagram, RCS
- **AI-Powered Bots**: Build intelligent chatbots with OpenAI, Anthropic, or custom LLM providers
- **Visual Flow Builder**: Design conversation flows with drag-and-drop simplicity
- **Knowledge Base**: Train your AI with documents and FAQs using semantic search
- **Multi-language SDKs**: TypeScript, Python, Go, Java, Rust, .NET, PHP
- **Real-time Analytics**: Monitor conversations and track performance
- **Human Handoff**: Seamlessly escalate to human agents when needed

## Quick Start

Get started with Linktor in under 5 minutes:

```bash
# Clone the repository
git clone https://github.com/linktor/linktor.git
cd linktor

# Start with Docker Compose
docker-compose up -d

# Access the dashboard
open http://localhost:3000
```

## Architecture

Linktor consists of several components:

| Component | Description |
|-----------|-------------|
| **API Server** | Go-based REST API and WebSocket server |
| **Admin Dashboard** | Next.js web application for management |
| **Worker** | Background job processor for async tasks |
| **PostgreSQL** | Primary database for conversations and config |
| **Redis** | Caching and real-time pub/sub |
| **NATS** | Message queue for channel integrations |
| **MinIO** | Object storage for media files |

## Next Steps

- [Installation Guide](/getting-started/installation) - Set up Linktor on your infrastructure
- [Quick Start](/getting-started/quick-start) - Create your first chatbot
- [API Reference](/api/overview) - Explore the REST API
- [SDKs](/sdks/overview) - Integrate with your favorite language
