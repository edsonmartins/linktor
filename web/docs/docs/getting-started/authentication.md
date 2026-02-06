---
sidebar_position: 3
title: Authentication
---

# Authentication

Linktor uses JWT-based authentication for the API and dashboard.

## API Authentication

### API Keys

The recommended way to authenticate API requests is using API keys:

1. Go to **Settings** → **API Keys** in the dashboard
2. Click **Generate New Key**
3. Copy the key (it won't be shown again)
4. Use it in the `Authorization` header:

```bash
curl -X GET http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### JWT Tokens

For user-based authentication, use JWT tokens:

```bash
# Login to get a token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "your-password"
  }'

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "user_123",
    "email": "user@example.com",
    "role": "admin"
  }
}

# Use the token
curl -X GET http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

## SDK Authentication

All SDKs support API key authentication:

```typescript
// TypeScript
import { Linktor } from '@linktor/sdk'
const client = new Linktor({ apiKey: 'YOUR_API_KEY' })
```

```python
# Python
from linktor import Linktor
client = Linktor(api_key='YOUR_API_KEY')
```

```go
// Go
client := linktor.New("YOUR_API_KEY")
```

## User Roles

Linktor supports three user roles:

| Role | Permissions |
|------|-------------|
| **Admin** | Full access to all features |
| **Agent** | Handle conversations, view analytics |
| **Viewer** | Read-only access to dashboards |

## Security Best Practices

1. **Never expose API keys** in client-side code
2. **Rotate keys regularly** - generate new keys periodically
3. **Use environment variables** - don't hardcode keys
4. **Limit key permissions** - use the minimum required scope
5. **Enable HTTPS** - always use TLS in production

## Rate Limiting

API requests are rate-limited:

| Endpoint | Limit |
|----------|-------|
| Authentication | 10 req/min |
| Messages | 100 req/min |
| Other endpoints | 60 req/min |

Rate limit headers are included in responses:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1640000000
```

## Next Steps

- [API Reference](/api/overview) - Explore all endpoints
- [SDKs](/sdks/overview) - Use official client libraries
