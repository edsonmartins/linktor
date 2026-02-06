---
sidebar_position: 7
title: WebChat
---

# WebChat Widget Integration

Embed a customizable chat widget on your website to engage visitors in real-time. WebChat provides instant support without requiring users to leave your site or download any apps.

## Overview

The WebChat integration enables you to:

- Embed a chat widget on any website
- Customize appearance to match your brand
- Support text, images, files, and rich messages
- Show typing indicators and read receipts
- Pre-chat forms to collect visitor information
- Proactive messages to engage visitors
- Multi-language support
- Mobile-responsive design

## Prerequisites

Before configuring WebChat in Linktor, you'll need:

1. **Linktor Account**: An active Linktor account with API access
2. **Website Access**: Ability to add JavaScript to your website
3. **HTTPS**: Your website must use HTTPS (required for secure communication)

## Configuration in Linktor

### Step 1: Create WebChat Channel

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **WebChat**
3. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name (e.g., "Website Chat") |
| **Allowed Domains** | Domains where widget can be embedded (e.g., `example.com, *.example.com`) |
| **Default Language** | Primary language for the widget |

4. Click **Save** to create the channel

### Step 2: Customize Appearance

Navigate to the **Appearance** tab to customize:

| Setting | Description |
|---------|-------------|
| **Primary Color** | Main color for buttons and headers |
| **Widget Position** | Bottom-right or bottom-left |
| **Widget Size** | Compact, standard, or large |
| **Launcher Icon** | Chat bubble or custom icon |
| **Company Logo** | Logo displayed in chat header |
| **Welcome Message** | Initial greeting message |
| **Offline Message** | Message when agents unavailable |

### Step 3: Configure Behavior

Navigate to the **Behavior** tab:

| Setting | Description |
|---------|-------------|
| **Pre-chat Form** | Collect name/email before chat |
| **File Uploads** | Allow users to send files |
| **Emojis** | Enable emoji picker |
| **Sound Notifications** | Play sound for new messages |
| **Proactive Messages** | Trigger messages based on conditions |

### Step 4: Get Embed Code

1. Go to the **Installation** tab
2. Copy the embed code snippet
3. Add it to your website before the closing `</body>` tag

## Installation

### Basic Installation

Add this code to your website:

```html
<!-- Linktor WebChat Widget -->
<script>
  (function(l,i,n,k,t,o,r){
    l['LinktorObject']=t;l[t]=l[t]||function(){
    (l[t].q=l[t].q||[]).push(arguments)};l[t].l=1*new Date();
    o=i.createElement(n);r=i.getElementsByTagName(n)[0];
    o.async=1;o.src=k;r.parentNode.insertBefore(o,r);
  })(window,document,'script','https://widget.linktor.io/v1/linktor.js','linktor');

  linktor('init', {
    channelId: 'ch_webchat_123'
  });
</script>
```

### Advanced Configuration

```html
<script>
  linktor('init', {
    channelId: 'ch_webchat_123',

    // Appearance
    primaryColor: '#0066FF',
    position: 'bottom-right',
    size: 'standard',
    zIndex: 9999,

    // Localization
    language: 'en',
    translations: {
      welcomeMessage: 'Hi there! How can we help?',
      inputPlaceholder: 'Type your message...',
      sendButton: 'Send',
      offlineMessage: 'We are currently offline. Leave a message!'
    },

    // Behavior
    autoOpen: false,
    openDelay: 0,
    soundEnabled: true,
    showLauncher: true,

    // Pre-chat form
    preChatForm: {
      enabled: true,
      fields: [
        { name: 'name', label: 'Name', type: 'text', required: true },
        { name: 'email', label: 'Email', type: 'email', required: true },
        { name: 'company', label: 'Company', type: 'text', required: false }
      ]
    },

    // User identification (if logged in)
    user: {
      id: 'user_123',
      name: 'John Doe',
      email: 'john@example.com',
      avatar: 'https://example.com/avatar.jpg',
      metadata: {
        plan: 'enterprise',
        accountId: 'acc_456'
      }
    },

    // Callbacks
    onReady: function() {
      console.log('Linktor widget loaded');
    },
    onOpen: function() {
      console.log('Chat opened');
    },
    onClose: function() {
      console.log('Chat closed');
    },
    onMessage: function(message) {
      console.log('New message:', message);
    }
  });
</script>
```

### React Installation

```jsx
// Install the package
// npm install @linktor/react-webchat

import { LinktorChat } from '@linktor/react-webchat';

function App() {
  return (
    <div>
      <YourAppContent />

      <LinktorChat
        channelId="ch_webchat_123"
        primaryColor="#0066FF"
        position="bottom-right"
        user={{
          id: currentUser.id,
          name: currentUser.name,
          email: currentUser.email
        }}
        onMessage={(message) => console.log('New message:', message)}
      />
    </div>
  );
}
```

### Vue.js Installation

```vue
<!-- Install the package -->
<!-- npm install @linktor/vue-webchat -->

<template>
  <div>
    <YourAppContent />

    <LinktorChat
      channel-id="ch_webchat_123"
      primary-color="#0066FF"
      position="bottom-right"
      :user="currentUser"
      @message="handleMessage"
    />
  </div>
</template>

<script>
import { LinktorChat } from '@linktor/vue-webchat';

export default {
  components: { LinktorChat },
  data() {
    return {
      currentUser: {
        id: 'user_123',
        name: 'John Doe',
        email: 'john@example.com'
      }
    };
  },
  methods: {
    handleMessage(message) {
      console.log('New message:', message);
    }
  }
};
</script>
```

### Angular Installation

```typescript
// Install the package
// npm install @linktor/angular-webchat

// app.module.ts
import { LinktorChatModule } from '@linktor/angular-webchat';

@NgModule({
  imports: [
    LinktorChatModule.forRoot({
      channelId: 'ch_webchat_123'
    })
  ]
})
export class AppModule {}

// app.component.html
<linktor-chat
  [primaryColor]="'#0066FF'"
  [position]="'bottom-right'"
  [user]="currentUser"
  (onMessage)="handleMessage($event)">
</linktor-chat>
```

## API Usage

### JavaScript API

Control the widget programmatically:

```javascript
// Open the chat widget
linktor('open');

// Close the chat widget
linktor('close');

// Toggle the chat widget
linktor('toggle');

// Send a message
linktor('sendMessage', {
  text: 'Hello, I need help!'
});

// Update user information
linktor('updateUser', {
  id: 'user_123',
  name: 'John Doe',
  email: 'john@example.com',
  metadata: {
    plan: 'premium'
  }
});

// Show a proactive message
linktor('showProactiveMessage', {
  text: 'Need help finding something?',
  delay: 5000 // Show after 5 seconds
});

// Hide the launcher
linktor('hideLauncher');

// Show the launcher
linktor('showLauncher');

// Destroy the widget
linktor('destroy');

// Reset conversation (new session)
linktor('reset');
```

### Server-Side API

Send messages from your backend:

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

// Send message to a visitor
await client.messages.send({
  channelId: 'ch_webchat_123',
  to: 'visitor_abc123', // Visitor ID from session
  content: {
    type: 'text',
    text: 'Thanks for your patience! Here is the information you requested.'
  }
})

// Send rich message with buttons
await client.messages.send({
  channelId: 'ch_webchat_123',
  to: 'visitor_abc123',
  content: {
    type: 'interactive',
    interactive: {
      type: 'buttons',
      body: {
        text: 'How would you rate your experience?'
      },
      buttons: [
        { id: 'rating_great', label: 'Great!' },
        { id: 'rating_good', label: 'Good' },
        { id: 'rating_poor', label: 'Could be better' }
      ]
    }
  }
})

// Send a card with image
await client.messages.send({
  channelId: 'ch_webchat_123',
  to: 'visitor_abc123',
  content: {
    type: 'card',
    card: {
      title: 'iPhone 15 Pro',
      subtitle: '$999 - Free shipping',
      imageUrl: 'https://example.com/iphone.jpg',
      buttons: [
        { id: 'view', label: 'View Details', url: 'https://example.com/products/iphone-15' },
        { id: 'buy', label: 'Add to Cart' }
      ]
    }
  }
})
```

### Webhook Events

```typescript
// New conversation started
{
  "event": "webchat.conversation.started",
  "data": {
    "conversationId": "conv_abc123",
    "channelId": "ch_webchat_123",
    "visitor": {
      "id": "visitor_xyz",
      "name": "John Doe",
      "email": "john@example.com",
      "metadata": {
        "page": "/pricing",
        "referrer": "https://google.com",
        "userAgent": "Mozilla/5.0..."
      }
    },
    "timestamp": "2024-01-15T10:30:00Z"
  }
}

// Message received from visitor
{
  "event": "message.received",
  "data": {
    "id": "msg_abc123",
    "channelId": "ch_webchat_123",
    "channelType": "webchat",
    "direction": "inbound",
    "from": "visitor_xyz",
    "content": {
      "type": "text",
      "text": "Do you offer enterprise pricing?"
    },
    "timestamp": "2024-01-15T10:30:05Z",
    "metadata": {
      "conversationId": "conv_abc123",
      "page": "/pricing"
    }
  }
}

// Button clicked
{
  "event": "webchat.button.clicked",
  "data": {
    "conversationId": "conv_abc123",
    "messageId": "msg_abc123",
    "buttonId": "rating_great",
    "visitor": "visitor_xyz",
    "timestamp": "2024-01-15T10:31:00Z"
  }
}
```

## Customization

### Custom CSS

Override default styles with custom CSS:

```css
/* Custom styles for Linktor WebChat */
.linktor-widget {
  --linktor-primary: #0066FF;
  --linktor-primary-dark: #0052CC;
  --linktor-text: #333333;
  --linktor-background: #FFFFFF;
  --linktor-border-radius: 12px;
}

.linktor-launcher {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.linktor-header {
  background: linear-gradient(135deg, var(--linktor-primary), var(--linktor-primary-dark));
}

.linktor-message-agent {
  background-color: #F0F4F8;
  border-radius: 16px 16px 16px 4px;
}

.linktor-message-visitor {
  background-color: var(--linktor-primary);
  color: white;
  border-radius: 16px 16px 4px 16px;
}
```

### Custom Launcher

Replace the default launcher with your own:

```javascript
linktor('init', {
  channelId: 'ch_webchat_123',
  showLauncher: false // Hide default launcher
});

// Create custom launcher
document.getElementById('my-chat-button').addEventListener('click', function() {
  linktor('toggle');
});
```

### Proactive Messages

Trigger messages based on user behavior:

```javascript
linktor('init', {
  channelId: 'ch_webchat_123',
  proactiveMessages: [
    {
      trigger: 'time',
      delay: 30000, // 30 seconds
      message: 'Hi! Need any help?',
      onlyOnce: true
    },
    {
      trigger: 'pageview',
      page: '/pricing',
      delay: 10000,
      message: 'Have questions about pricing? I am here to help!'
    },
    {
      trigger: 'scroll',
      percentage: 75,
      message: 'Looks like you are interested! Want to chat?'
    },
    {
      trigger: 'exit_intent',
      message: 'Wait! Before you go, can we help with anything?'
    }
  ]
});
```

## Features

### File Uploads

```javascript
linktor('init', {
  channelId: 'ch_webchat_123',
  fileUpload: {
    enabled: true,
    maxSize: 10 * 1024 * 1024, // 10MB
    allowedTypes: ['image/*', 'application/pdf', '.doc', '.docx'],
    multiple: true
  }
});
```

### Typing Indicators

Typing indicators are automatic. Control them server-side:

```typescript
// Show typing indicator
await client.webchat.typing.start({
  channelId: 'ch_webchat_123',
  conversationId: 'conv_abc123'
});

// Stop typing indicator
await client.webchat.typing.stop({
  channelId: 'ch_webchat_123',
  conversationId: 'conv_abc123'
});
```

### Message History

Messages are persisted and restored when visitors return:

```javascript
linktor('init', {
  channelId: 'ch_webchat_123',
  persistSession: true, // Enable session persistence
  sessionDuration: 7 * 24 * 60 * 60 * 1000 // 7 days
});
```

### Offline Mode

Handle when agents are unavailable:

```javascript
linktor('init', {
  channelId: 'ch_webchat_123',
  offline: {
    enabled: true,
    message: 'We are currently offline. Leave a message and we will get back to you!',
    form: {
      fields: [
        { name: 'email', label: 'Email', type: 'email', required: true },
        { name: 'message', label: 'Message', type: 'textarea', required: true }
      ],
      submitButton: 'Send Message'
    }
  }
});
```

## Common Issues and Troubleshooting

### Widget Not Loading

**Possible causes:**
- Script not loaded
- Invalid channel ID
- Domain not allowed

**Solution:**
- Check browser console for errors
- Verify channel ID is correct
- Add your domain to allowed domains list
- Ensure HTTPS is used

### Messages Not Sending

**Possible causes:**
- WebSocket connection failed
- API rate limit
- Network issues

**Solution:**
- Check network connectivity
- Verify WebSocket connection in Network tab
- Check Linktor status page

### Widget Not Showing on Mobile

**Possible causes:**
- CSS conflicts
- Z-index issues
- Position conflicts

**Solution:**
- Check for CSS conflicts
- Increase z-index value
- Test on actual devices

### Cross-Origin Errors

**Possible causes:**
- Widget embedded on non-allowed domain
- Mixed HTTP/HTTPS content

**Solution:**
- Add domain to allowed list
- Ensure all resources use HTTPS

### User Data Not Updating

**Possible causes:**
- updateUser called before init
- Invalid user object format

**Solution:**
- Ensure init completes before updateUser
- Verify user object structure

## Best Practices

1. **Fast Initial Load**: Use the async script to avoid blocking page load.

2. **Identify Users**: Pass user information when available for better context.

3. **Strategic Placement**: Position widget where it's visible but not intrusive.

4. **Proactive Engagement**: Use proactive messages sparingly and relevantly.

5. **Quick Responses**: Set expectations for response times, especially when using bots.

6. **Mobile Optimization**: Test on various mobile devices and screen sizes.

7. **Accessibility**: Ensure the widget is keyboard-navigable and screen-reader friendly.

8. **Performance Monitoring**: Track load times and interaction metrics.

## Security

### Domain Restrictions

Only allow widget on your domains:

```javascript
// In Linktor dashboard, set allowed domains:
// example.com, *.example.com, staging.example.com
```

### Content Security Policy

Add to your CSP header:

```
script-src 'self' https://widget.linktor.io;
connect-src 'self' https://api.linktor.io wss://realtime.linktor.io;
frame-src 'self' https://widget.linktor.io;
```

### Data Privacy

```javascript
linktor('init', {
  channelId: 'ch_webchat_123',
  privacy: {
    cookieConsent: true, // Show cookie consent
    dataRetention: 30, // Days to retain data
    anonymizeIP: true // Anonymize visitor IPs
  }
});
```

## Next Steps

- [Flows](/flows/overview) - Build automated WebChat flows
- [AI Bots](/bots/overview) - Add AI to your WebChat
- [Customization](/guides/webchat-customization) - Advanced customization guide
- [Analytics](/api/analytics) - Track WebChat metrics
