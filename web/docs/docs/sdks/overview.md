---
sidebar_position: 1
title: SDKs Overview
---

# SDKs Overview

Linktor provides official SDKs for 7 programming languages, making it easy to integrate with any tech stack.

## Available SDKs

| Language | Package | Version | Docs |
|----------|---------|---------|------|
| [TypeScript](/sdks/typescript) | `@linktor/sdk` | 1.0.0 | [API Reference](https://github.com/linktor/linktor/tree/main/sdks/typescript) |
| [Python](/sdks/python) | `linktor` | 1.0.0 | [API Reference](https://github.com/linktor/linktor/tree/main/sdks/python) |
| [Go](/sdks/go) | `linktor-go` | 1.0.0 | [API Reference](https://github.com/linktor/linktor/tree/main/sdks/go) |
| [Java](/sdks/java) | `io.linktor:linktor-sdk` | 1.0.0 | [API Reference](https://github.com/linktor/linktor/tree/main/sdks/java) |
| [Rust](/sdks/rust) | `linktor` | 1.0.0 | [API Reference](https://github.com/linktor/linktor/tree/main/sdks/rust) |
| [.NET](/sdks/dotnet) | `Linktor.SDK` | 1.0.0 | [API Reference](https://github.com/linktor/linktor/tree/main/sdks/dotnet) |
| [PHP](/sdks/php) | `linktor/linktor-php` | 1.0.0 | [API Reference](https://github.com/linktor/linktor/tree/main/sdks/php) |

## Features

All SDKs provide:

- **Full API Coverage**: Access all Linktor endpoints
- **Type Safety**: Strong typing for IDE support and error prevention
- **WebSocket Support**: Real-time message streaming
- **Automatic Retries**: Built-in retry logic for transient failures
- **Error Handling**: Detailed error types and messages

## Quick Comparison

```typescript
// TypeScript
const client = new Linktor({ apiKey: 'YOUR_API_KEY' })
await client.messages.send({ channelId, to, content: 'Hello!' })
```

```python
# Python
client = Linktor(api_key='YOUR_API_KEY')
client.messages.send(channel_id=channel_id, to=to, content='Hello!')
```

```go
// Go
client := linktor.New("YOUR_API_KEY")
client.Messages.Send(ctx, channelID, to, "Hello!")
```

```java
// Java
Linktor client = new Linktor("YOUR_API_KEY");
client.messages().send(channelId, to, "Hello!");
```

```rust
// Rust
let client = Client::new("YOUR_API_KEY");
client.messages().send(channel_id, to, "Hello!").await?;
```

```csharp
// .NET
var client = new LinktorClient("YOUR_API_KEY");
await client.Messages.SendAsync(channelId, to, "Hello!");
```

```php
// PHP
$client = new Client('YOUR_API_KEY');
$client->messages->send($channelId, $to, 'Hello!');
```

## Installation

Choose your language and follow the installation guide:

- [TypeScript](/sdks/typescript#installation)
- [Python](/sdks/python#installation)
- [Go](/sdks/go#installation)
- [Java](/sdks/java#installation)
- [Rust](/sdks/rust#installation)
- [.NET](/sdks/dotnet#installation)
- [PHP](/sdks/php#installation)
