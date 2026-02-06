---
sidebar_position: 1
title: API Overview
---

# API Overview

The Linktor API is a RESTful API that provides programmatic access to all Linktor features. Use the API to send messages, manage conversations, configure channels, and build integrations.

## Base URL

All API requests should be made to:

```
https://api.linktor.io/v1
```

For self-hosted installations, replace with your own domain:

```
https://your-domain.com/api/v1
```

## Authentication

The Linktor API uses Bearer token authentication. Include your API key in the `Authorization` header of every request:

```bash
curl https://api.linktor.io/v1/conversations \
  -H "Authorization: Bearer YOUR_API_KEY"
```

You can generate API keys in the Linktor dashboard under **Settings > API Keys**.

### Authentication Methods

| Method | Use Case |
|--------|----------|
| API Key | Server-to-server integrations |
| JWT Token | User sessions and frontend applications |
| OAuth 2.0 | Third-party application integrations |

See [Authentication](/api/authentication) for detailed documentation on authentication endpoints.

## Request Format

### Headers

All requests should include the following headers:

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer YOUR_API_KEY` | Yes |
| `Content-Type` | `application/json` | Yes (for POST/PUT/PATCH) |
| `Accept` | `application/json` | Recommended |
| `X-Request-ID` | UUID | Optional (for tracing) |

### Request Body

For POST, PUT, and PATCH requests, send data as JSON in the request body:

```bash
curl -X POST https://api.linktor.io/v1/messages \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channelId": "ch_abc123",
    "to": "+5511999999999",
    "content": {
      "type": "text",
      "text": "Hello from Linktor!"
    }
  }'
```

## Response Format

All responses are returned in JSON format with a consistent structure.

### Successful Response

Single resource:

```json
{
  "data": {
    "id": "msg_abc123",
    "type": "message",
    "attributes": {
      "channelId": "ch_xyz789",
      "content": {
        "type": "text",
        "text": "Hello from Linktor!"
      },
      "status": "delivered",
      "createdAt": "2024-01-15T10:30:00Z"
    }
  },
  "meta": {
    "requestId": "req_def456"
  }
}
```

Collection:

```json
{
  "data": [
    {
      "id": "msg_abc123",
      "type": "message",
      "attributes": { ... }
    },
    {
      "id": "msg_def456",
      "type": "message",
      "attributes": { ... }
    }
  ],
  "meta": {
    "requestId": "req_ghi789",
    "pagination": {
      "page": 1,
      "perPage": 20,
      "totalPages": 5,
      "totalCount": 100
    }
  }
}
```

### Error Response

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": [
      {
        "field": "to",
        "message": "Phone number must be in E.164 format"
      }
    ]
  },
  "meta": {
    "requestId": "req_xyz789"
  }
}
```

## Pagination

List endpoints support pagination using the following query parameters:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number (1-indexed) |
| `perPage` | integer | 20 | Items per page (max: 100) |
| `sortBy` | string | `createdAt` | Field to sort by |
| `sortOrder` | string | `desc` | Sort order (`asc` or `desc`) |

### Example

```bash
curl "https://api.linktor.io/v1/conversations?page=2&perPage=50&sortBy=updatedAt&sortOrder=desc" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Pagination Response

```json
{
  "data": [...],
  "meta": {
    "pagination": {
      "page": 2,
      "perPage": 50,
      "totalPages": 10,
      "totalCount": 500,
      "hasNextPage": true,
      "hasPrevPage": true
    }
  }
}
```

## Cursor-Based Pagination

For real-time data streams, some endpoints support cursor-based pagination:

| Parameter | Type | Description |
|-----------|------|-------------|
| `cursor` | string | Cursor from previous response |
| `limit` | integer | Number of items to return (max: 100) |

```bash
curl "https://api.linktor.io/v1/messages?cursor=eyJpZCI6Im1zZ18xMjM0NTYifQ&limit=50" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## Filtering

Most list endpoints support filtering via query parameters:

```bash
# Filter conversations by status
curl "https://api.linktor.io/v1/conversations?status=open" \
  -H "Authorization: Bearer YOUR_API_KEY"

# Filter messages by date range
curl "https://api.linktor.io/v1/messages?startDate=2024-01-01&endDate=2024-01-31" \
  -H "Authorization: Bearer YOUR_API_KEY"

# Multiple filters
curl "https://api.linktor.io/v1/conversations?status=open&channelId=ch_abc123&assigneeId=user_xyz" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## Error Codes

The API uses standard HTTP status codes and custom error codes:

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (successful delete) |
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Invalid or missing API key |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 409 | Conflict - Resource already exists |
| 422 | Unprocessable Entity - Validation failed |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

### Error Codes

| Code | Description |
|------|-------------|
| `AUTHENTICATION_REQUIRED` | No API key provided |
| `INVALID_API_KEY` | API key is invalid or expired |
| `INSUFFICIENT_PERMISSIONS` | API key lacks required permissions |
| `VALIDATION_ERROR` | Request validation failed |
| `RESOURCE_NOT_FOUND` | Requested resource doesn't exist |
| `RESOURCE_ALREADY_EXISTS` | Resource with same identifier exists |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `CHANNEL_ERROR` | Error communicating with channel provider |
| `INTERNAL_ERROR` | Unexpected server error |

## Rate Limiting

The API enforces rate limits to ensure fair usage:

| Tier | Requests/minute | Requests/day |
|------|-----------------|--------------|
| Free | 60 | 1,000 |
| Starter | 300 | 10,000 |
| Pro | 1,000 | 100,000 |
| Enterprise | Custom | Custom |

Rate limit headers are included in every response:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1705312800
```

When rate limited, you'll receive a 429 response:

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please retry after 60 seconds.",
    "retryAfter": 60
  }
}
```

## Idempotency

For POST requests, you can provide an `Idempotency-Key` header to ensure the request is processed only once:

```bash
curl -X POST https://api.linktor.io/v1/messages \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-request-id-123" \
  -d '{
    "channelId": "ch_abc123",
    "to": "+5511999999999",
    "content": {
      "type": "text",
      "text": "Hello!"
    }
  }'
```

Idempotency keys are stored for 24 hours. Replaying a request with the same key will return the original response.

## Versioning

The API is versioned via the URL path. The current version is `v1`:

```
https://api.linktor.io/v1/...
```

When we introduce breaking changes, we'll release a new version. Previous versions remain available for a deprecation period.

## SDKs

Instead of making raw HTTP requests, consider using our official SDKs:

- [TypeScript/JavaScript](/sdks/typescript)
- [Python](/sdks/python)
- [Go](/sdks/go)
- [Java](/sdks/java)
- [Rust](/sdks/rust)
- [.NET](/sdks/dotnet)
- [PHP](/sdks/php)

## Next Steps

- [Authentication](/api/authentication) - Login, register, and token management
- [Conversations](/api/conversations) - Manage conversations
- [Messages](/api/messages) - Send and receive messages
- [Channels](/api/channels) - Configure messaging channels
- [Bots](/api/bots) - Create and manage chatbots
- [Webhooks](/api/webhooks) - Real-time event notifications
