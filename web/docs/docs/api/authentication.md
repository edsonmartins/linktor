---
sidebar_position: 2
title: Authentication
---

# Authentication

The Authentication API allows you to register users, authenticate, and manage access tokens.

## Overview

Linktor supports multiple authentication methods:

- **API Keys**: Long-lived tokens for server-to-server integrations
- **JWT Tokens**: Short-lived tokens for user sessions
- **OAuth 2.0**: For third-party application integrations

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/register` | Register a new user account |
| POST | `/auth/login` | Authenticate and get tokens |
| POST | `/auth/refresh` | Refresh an expired access token |
| POST | `/auth/logout` | Revoke tokens |
| GET | `/auth/me` | Get current user info |
| POST | `/auth/password/reset` | Request password reset |
| POST | `/auth/password/confirm` | Confirm password reset |

---

## Register

Create a new user account.

```
POST /auth/register
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | Yes | User's email address |
| `password` | string | Yes | Password (min 8 characters) |
| `name` | string | Yes | User's display name |
| `organizationName` | string | No | Organization name (creates new org) |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword123",
    "name": "John Doe",
    "organizationName": "Acme Corp"
  }'
```

### Response

```json
{
  "data": {
    "id": "user_abc123",
    "type": "user",
    "attributes": {
      "email": "john@example.com",
      "name": "John Doe",
      "emailVerified": false,
      "createdAt": "2024-01-15T10:30:00Z"
    },
    "relationships": {
      "organization": {
        "id": "org_xyz789",
        "name": "Acme Corp"
      }
    }
  },
  "meta": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "rt_abc123def456",
    "expiresIn": 3600
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid email format or weak password |
| 409 | `EMAIL_ALREADY_EXISTS` | Email is already registered |

---

## Login

Authenticate a user and receive access tokens.

```
POST /auth/login
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | Yes | User's email address |
| `password` | string | Yes | User's password |
| `rememberMe` | boolean | No | Extend token expiry (default: false) |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword123",
    "rememberMe": true
  }'
```

### Response

```json
{
  "data": {
    "id": "user_abc123",
    "type": "user",
    "attributes": {
      "email": "john@example.com",
      "name": "John Doe",
      "emailVerified": true,
      "lastLoginAt": "2024-01-15T10:30:00Z"
    },
    "relationships": {
      "organization": {
        "id": "org_xyz789",
        "name": "Acme Corp"
      }
    }
  },
  "meta": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "rt_abc123def456",
    "expiresIn": 3600,
    "tokenType": "Bearer"
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Missing or invalid fields |
| 401 | `INVALID_CREDENTIALS` | Email or password is incorrect |
| 403 | `ACCOUNT_LOCKED` | Account locked due to too many attempts |
| 403 | `EMAIL_NOT_VERIFIED` | Email verification required |

---

## Refresh Token

Exchange a refresh token for a new access token.

```
POST /auth/refresh
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `refreshToken` | string | Yes | Valid refresh token |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "rt_abc123def456"
  }'
```

### Response

```json
{
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "rt_new789ghi012",
    "expiresIn": 3600,
    "tokenType": "Bearer"
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Missing refresh token |
| 401 | `INVALID_REFRESH_TOKEN` | Refresh token is invalid or expired |
| 401 | `TOKEN_REVOKED` | Refresh token has been revoked |

---

## Logout

Revoke the current access and refresh tokens.

```
POST /auth/logout
```

### Headers

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer YOUR_ACCESS_TOKEN` | Yes |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `refreshToken` | string | No | Revoke specific refresh token |
| `allDevices` | boolean | No | Revoke all tokens for user |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/auth/logout \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "allDevices": false
  }'
```

### Response

```json
{
  "data": {
    "message": "Successfully logged out"
  }
}
```

---

## Get Current User

Retrieve the authenticated user's profile.

```
GET /auth/me
```

### Headers

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer YOUR_ACCESS_TOKEN` | Yes |

### Example Request

```bash
curl https://api.linktor.io/v1/auth/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Response

```json
{
  "data": {
    "id": "user_abc123",
    "type": "user",
    "attributes": {
      "email": "john@example.com",
      "name": "John Doe",
      "avatar": "https://cdn.linktor.io/avatars/user_abc123.jpg",
      "emailVerified": true,
      "createdAt": "2024-01-01T00:00:00Z",
      "lastLoginAt": "2024-01-15T10:30:00Z"
    },
    "relationships": {
      "organization": {
        "id": "org_xyz789",
        "name": "Acme Corp",
        "role": "admin"
      }
    }
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 401 | `AUTHENTICATION_REQUIRED` | No token provided |
| 401 | `INVALID_TOKEN` | Token is invalid or expired |

---

## Request Password Reset

Send a password reset email to the user.

```
POST /auth/password/reset
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | Yes | User's email address |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/auth/password/reset \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com"
  }'
```

### Response

```json
{
  "data": {
    "message": "If an account exists with this email, a password reset link has been sent."
  }
}
```

**Note**: The response is intentionally vague to prevent email enumeration attacks.

---

## Confirm Password Reset

Reset the password using a reset token.

```
POST /auth/password/confirm
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | Password reset token from email |
| `password` | string | Yes | New password (min 8 characters) |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/auth/password/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "token": "reset_token_from_email",
    "password": "newsecurepassword456"
  }'
```

### Response

```json
{
  "data": {
    "message": "Password has been reset successfully"
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Weak password |
| 400 | `INVALID_RESET_TOKEN` | Token is invalid or expired |

---

## API Keys

API keys are used for server-to-server integrations. Manage them via the dashboard or API.

### List API Keys

```bash
curl https://api.linktor.io/v1/api-keys \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Create API Key

```bash
curl -X POST https://api.linktor.io/v1/api-keys \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production Server",
    "permissions": ["messages:read", "messages:write", "conversations:read"],
    "expiresAt": "2025-01-01T00:00:00Z"
  }'
```

### Response

```json
{
  "data": {
    "id": "key_abc123",
    "type": "api_key",
    "attributes": {
      "name": "Production Server",
      "key": "lk_live_abc123def456...",
      "permissions": ["messages:read", "messages:write", "conversations:read"],
      "lastUsedAt": null,
      "expiresAt": "2025-01-01T00:00:00Z",
      "createdAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

**Important**: The `key` field is only returned once when the API key is created. Store it securely.

### Delete API Key

```bash
curl -X DELETE https://api.linktor.io/v1/api-keys/key_abc123 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

---

## Permissions

API keys can be scoped with specific permissions:

| Permission | Description |
|------------|-------------|
| `messages:read` | Read messages |
| `messages:write` | Send messages |
| `conversations:read` | Read conversations |
| `conversations:write` | Create/update conversations |
| `channels:read` | Read channel configurations |
| `channels:write` | Create/update channels |
| `bots:read` | Read bot configurations |
| `bots:write` | Create/update bots |
| `webhooks:read` | Read webhook configurations |
| `webhooks:write` | Create/update webhooks |
| `analytics:read` | Read analytics data |
| `admin` | Full administrative access |

---

## Token Expiration

| Token Type | Default Expiry | Maximum Expiry |
|------------|----------------|----------------|
| Access Token | 1 hour | 24 hours |
| Refresh Token | 7 days | 30 days |
| API Key | No expiry | Custom |
| Password Reset Token | 1 hour | 1 hour |

---

## Security Best Practices

1. **Store tokens securely**: Never expose tokens in client-side code or logs
2. **Use HTTPS**: All API calls must use HTTPS
3. **Rotate API keys**: Regularly rotate production API keys
4. **Minimal permissions**: Grant only necessary permissions to API keys
5. **Monitor usage**: Track API key usage for suspicious activity
