---
sidebar_position: 5
title: Email
---

# Email Integration

Connect email to Linktor for two-way email conversations. Support customers through their inbox, send transactional emails, and manage email-based support tickets.

## Overview

The Email integration enables you to:

- Send and receive emails with full HTML support
- Handle attachments and inline images
- Track email opens and clicks (optional)
- Thread conversations automatically
- Support multiple email addresses per inbox
- Integrate with SMTP/IMAP or email APIs (SendGrid, Mailgun, AWS SES)

## Prerequisites

Before configuring Email in Linktor, you'll need:

1. **Email Provider**: SMTP/IMAP credentials or an email API provider account
2. **Domain Configuration**: DNS records for SPF, DKIM, and DMARC
3. **Email Address**: The address(es) you'll send/receive from

### Supported Providers

| Provider | Type | Features | Best For |
|----------|------|----------|----------|
| **Custom SMTP/IMAP** | Protocol | Full control | Existing mail servers |
| **SendGrid** | API | High deliverability, analytics | Transactional & marketing |
| **Mailgun** | API | Developer-friendly, EU hosting | Developers, EU compliance |
| **AWS SES** | API | AWS integration, low cost | AWS users, high volume |
| **Postmark** | API | Excellent deliverability | Transactional only |
| **Microsoft 365** | API | Office integration | Enterprise |
| **Google Workspace** | API | Gmail integration | Google users |

## Configuration in Linktor

### SMTP/IMAP Setup

#### Step 1: Gather SMTP/IMAP Credentials

You'll need:
- **SMTP Server**: e.g., `smtp.gmail.com`
- **SMTP Port**: Usually 587 (TLS) or 465 (SSL)
- **IMAP Server**: e.g., `imap.gmail.com`
- **IMAP Port**: Usually 993 (SSL)
- **Username**: Your email address
- **Password**: Your password or app-specific password

#### Step 2: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **Email**
3. Choose **SMTP/IMAP** as the provider
4. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name (e.g., "Support Email") |
| **Email Address** | The email address for this channel |
| **Display Name** | Name shown to recipients (e.g., "Acme Support") |
| **SMTP Host** | SMTP server hostname |
| **SMTP Port** | SMTP port (587 or 465) |
| **SMTP Security** | TLS or SSL |
| **IMAP Host** | IMAP server hostname |
| **IMAP Port** | IMAP port (usually 993) |
| **Username** | Authentication username |
| **Password** | Authentication password |

5. Click **Test Connection** to verify settings
6. Click **Save** to create the channel

### SendGrid Setup

#### Step 1: Get SendGrid Credentials

1. Sign up at [sendgrid.com](https://sendgrid.com)
2. Go to **Settings → API Keys**
3. Create an API key with full access or restricted to Mail Send
4. Configure domain authentication in **Settings → Sender Authentication**

#### Step 2: Configure Inbound Parse

1. In SendGrid, go to **Settings → Inbound Parse**
2. Add your domain and point to: `https://api.your-domain.com/webhooks/email/{channelId}`
3. Add MX record to your DNS pointing to `mx.sendgrid.net`

#### Step 3: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **Email**
3. Choose **SendGrid** as the provider
4. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel |
| **Email Address** | Your verified sender email |
| **API Key** | Your SendGrid API key |
| **Webhook Signing Key** | For webhook validation (optional) |

### AWS SES Setup

#### Step 1: Configure AWS SES

1. Open AWS SES Console
2. Verify your domain or email address
3. Create SMTP credentials or use IAM for API access
4. Move out of sandbox mode for production

#### Step 2: Configure Receiving

1. Create an S3 bucket for incoming emails
2. Set up a receipt rule in SES to:
   - Store emails in S3
   - Trigger a Lambda function or SNS notification
3. Configure Linktor webhook to receive notifications

#### Step 3: Add Channel in Dashboard

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel |
| **Email Address** | Your verified sender email |
| **AWS Region** | SES region (e.g., us-east-1) |
| **Access Key ID** | AWS access key |
| **Secret Access Key** | AWS secret key |
| **Configuration Set** | Optional, for tracking |

## API Usage

### Sending Emails

#### Basic Text Email

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.messages.send({
  channelId: 'ch_email_123',
  to: 'customer@example.com',
  content: {
    type: 'email',
    email: {
      subject: 'Your order has shipped',
      text: 'Hi John, your order #12345 has shipped and will arrive in 2-3 days.'
    }
  }
})
```

#### HTML Email

```typescript
await client.messages.send({
  channelId: 'ch_email_123',
  to: 'customer@example.com',
  content: {
    type: 'email',
    email: {
      subject: 'Your order has shipped',
      html: `
        <h1>Order Shipped!</h1>
        <p>Hi John,</p>
        <p>Your order <strong>#12345</strong> has shipped.</p>
        <a href="https://track.example.com/12345">Track your package</a>
      `,
      text: 'Hi John, your order #12345 has shipped.'
    }
  }
})
```

#### Email with Attachments

```typescript
await client.messages.send({
  channelId: 'ch_email_123',
  to: 'customer@example.com',
  content: {
    type: 'email',
    email: {
      subject: 'Your Invoice #2024-001',
      html: '<p>Please find your invoice attached.</p>',
      attachments: [
        {
          filename: 'invoice-2024-001.pdf',
          content: base64EncodedPdf,
          contentType: 'application/pdf'
        },
        {
          filename: 'receipt.pdf',
          url: 'https://storage.example.com/receipts/2024-001.pdf'
        }
      ]
    }
  }
})
```

#### Email with CC and BCC

```typescript
await client.messages.send({
  channelId: 'ch_email_123',
  to: 'customer@example.com',
  content: {
    type: 'email',
    email: {
      subject: 'Contract for Review',
      html: '<p>Please review the attached contract.</p>',
      cc: ['manager@example.com', 'legal@example.com'],
      bcc: ['records@example.com'],
      replyTo: 'contracts@example.com'
    }
  }
})
```

#### Reply to Thread

```typescript
await client.messages.send({
  channelId: 'ch_email_123',
  to: 'customer@example.com',
  content: {
    type: 'email',
    email: {
      subject: 'Re: Support Request #12345',
      html: '<p>Thanks for your patience. Here is the update...</p>',
      inReplyTo: 'original-message-id@example.com',
      references: ['original-message-id@example.com']
    }
  }
})
```

### Email Templates

#### Create a Template

```typescript
await client.templates.create({
  name: 'welcome_email',
  channel: 'email',
  content: {
    subject: 'Welcome to {{company_name}}, {{first_name}}!',
    html: `
      <h1>Welcome, {{first_name}}!</h1>
      <p>Thanks for joining {{company_name}}.</p>
      <p>Here are your next steps:</p>
      <ol>
        <li>Complete your profile</li>
        <li>Explore our features</li>
        <li>Reach out if you need help</li>
      </ol>
      <a href="{{dashboard_url}}">Go to Dashboard</a>
    `,
    text: 'Welcome, {{first_name}}! Thanks for joining {{company_name}}.'
  }
})
```

#### Send Using Template

```typescript
await client.messages.send({
  channelId: 'ch_email_123',
  to: 'newuser@example.com',
  template: {
    name: 'welcome_email',
    variables: {
      first_name: 'John',
      company_name: 'Acme Inc',
      dashboard_url: 'https://app.acme.com/dashboard'
    }
  }
})
```

### Receiving Emails

#### Webhook Events

```typescript
// Email received event
{
  "event": "message.received",
  "data": {
    "id": "msg_abc123",
    "channelId": "ch_email_123",
    "channelType": "email",
    "direction": "inbound",
    "from": "customer@example.com",
    "to": "support@yourcompany.com",
    "content": {
      "type": "email",
      "email": {
        "subject": "Help with my order",
        "text": "Hi, I need help with order #12345...",
        "html": "<p>Hi, I need help with order #12345...</p>",
        "attachments": [
          {
            "filename": "screenshot.png",
            "contentType": "image/png",
            "size": 102400,
            "url": "https://storage.linktor.io/attachments/abc123.png"
          }
        ]
      }
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "messageId": "<abc123@mail.example.com>",
      "inReplyTo": "<original@mail.yourcompany.com>",
      "headers": {
        "from": "John Doe <customer@example.com>",
        "date": "Mon, 15 Jan 2024 10:30:00 +0000"
      }
    }
  }
}
```

### Tracking

#### Enable Tracking

```typescript
await client.messages.send({
  channelId: 'ch_email_123',
  to: 'customer@example.com',
  content: {
    type: 'email',
    email: {
      subject: 'Check out our new features',
      html: '<p>We have exciting new features!</p>'
    }
  },
  tracking: {
    opens: true,
    clicks: true
  }
})
```

#### Tracking Events

```typescript
// Open event
{
  "event": "email.opened",
  "data": {
    "messageId": "msg_abc123",
    "recipient": "customer@example.com",
    "timestamp": "2024-01-15T10:35:00Z",
    "metadata": {
      "userAgent": "Mozilla/5.0...",
      "ip": "192.168.1.1"
    }
  }
}

// Click event
{
  "event": "email.clicked",
  "data": {
    "messageId": "msg_abc123",
    "recipient": "customer@example.com",
    "url": "https://example.com/features",
    "timestamp": "2024-01-15T10:36:00Z"
  }
}
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/email/{channelId}
```

### Inbound Email Parsing

Different providers have different webhook formats. Linktor normalizes all of them:

**SendGrid Inbound Parse:**
- Multipart form data with email fields
- Attachments as file uploads

**Mailgun:**
- JSON payload with email data
- Attachments as URLs

**AWS SES:**
- SNS notification with S3 location
- Linktor fetches email from S3

## Common Issues and Troubleshooting

### Emails Going to Spam

**Possible causes:**
- Missing SPF, DKIM, or DMARC records
- New domain without reputation
- Spammy content or subject lines
- High bounce rate

**Solution:**
- Configure all DNS authentication records:
  ```
  # SPF
  v=spf1 include:sendgrid.net ~all

  # DKIM (add provider's DKIM record)

  # DMARC
  _dmarc.yourdomain.com TXT "v=DMARC1; p=quarantine; rua=mailto:dmarc@yourdomain.com"
  ```
- Warm up new domains gradually
- Test with mail-tester.com before sending
- Clean your email list regularly

### Not Receiving Inbound Emails

**Possible causes:**
- MX records not configured
- Webhook URL not accessible
- Inbound parsing not enabled

**Solution:**
- Verify MX records point to your provider
- Test webhook endpoint accessibility
- Check provider's inbound settings

### SMTP Authentication Failed

**Possible causes:**
- Wrong credentials
- 2FA without app password
- SMTP access disabled

**Solution:**
- Verify username and password
- For Gmail/Google Workspace: use App Password
- Enable "Less secure app access" or use OAuth

### Attachments Not Received

**Possible causes:**
- Attachment too large
- Blocked file type
- Parsing configuration issue

**Solution:**
- Check attachment size limits (usually 10-25MB)
- Verify allowed file types
- Check inbound parse settings

### Threading Not Working

**Possible causes:**
- Missing In-Reply-To header
- Subject line changed
- Different From address

**Solution:**
- Include proper In-Reply-To and References headers
- Keep original subject (with Re: prefix)
- Use consistent sender address

## Best Practices

1. **Authenticate Your Domain**: Set up SPF, DKIM, and DMARC for all sending domains.

2. **Use a Consistent From Address**: Build reputation with consistent sender addresses.

3. **Include Plain Text**: Always provide a text version alongside HTML.

4. **Mobile-Friendly Design**: Over 50% of emails are opened on mobile devices.

5. **Clear Subject Lines**: Be concise and descriptive, avoid spam triggers.

6. **Easy Unsubscribe**: Include an unsubscribe link in marketing emails.

7. **Monitor Deliverability**: Track bounce rates, spam complaints, and engagement.

8. **Thread Conversations**: Use proper headers to maintain conversation context.

## Email Authentication

### SPF Record

```
v=spf1 include:sendgrid.net include:mailgun.org ~all
```

### DKIM

Add the DKIM record provided by your email provider. Example:

```
s1._domainkey.yourdomain.com TXT "k=rsa; p=MIGfMA0GCSqGSIb3DQEBA..."
```

### DMARC

```
_dmarc.yourdomain.com TXT "v=DMARC1; p=quarantine; pct=100; rua=mailto:dmarc-reports@yourdomain.com"
```

## Rate Limits

| Provider | Limit | Notes |
|----------|-------|-------|
| SendGrid Free | 100/day | Upgrade for more |
| SendGrid Essentials | 40,000/month | Starting tier |
| AWS SES | 200/sec | Can request increase |
| Mailgun | 300/min | Free tier |
| SMTP (Gmail) | 500/day | Personal accounts |
| SMTP (Google Workspace) | 2,000/day | Business accounts |

## Next Steps

- [Flows](/flows/overview) - Build email automation flows
- [Templates](/api/templates) - Create reusable email templates
- [Inboxes](/guides/inboxes) - Manage team email inboxes
- [Analytics](/api/analytics) - Track email performance
