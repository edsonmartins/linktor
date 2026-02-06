---
sidebar_position: 6
title: Voice
---

# Voice Integration

Add voice calling capabilities to Linktor using providers like Twilio or Vonage. Build IVR systems, enable click-to-call, and integrate voice with your AI bots.

## Overview

The Voice integration enables you to:

- Make and receive phone calls programmatically
- Build interactive voice response (IVR) systems
- Convert speech to text for AI processing
- Generate dynamic text-to-speech responses
- Record calls for quality assurance
- Transfer calls to human agents
- Handle conference calls

## Prerequisites

Before configuring Voice in Linktor, you'll need:

1. **Voice Provider Account**: Twilio or Vonage account with voice capability
2. **Phone Number**: A phone number with voice capability
3. **Webhook URL**: Publicly accessible HTTPS endpoint

### Supported Providers

| Provider | Features | Best For |
|----------|----------|----------|
| **Twilio** | Full-featured, global, STT/TTS | Enterprise, global reach |
| **Vonage** | WebRTC support, competitive pricing | Web-based calling |
| **Plivo** | Affordable, developer-friendly | Cost-sensitive applications |

## Configuration in Linktor

### Twilio Voice Setup

#### Step 1: Get Twilio Voice Credentials

1. Sign up at [twilio.com](https://www.twilio.com)
2. From the Console, note your:
   - **Account SID**
   - **Auth Token**
3. Buy a phone number with Voice capability
4. Note the phone number

#### Step 2: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **Voice**
3. Choose **Twilio** as the provider
4. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name (e.g., "Support Hotline") |
| **Account SID** | Your Twilio Account SID |
| **Auth Token** | Your Twilio Auth Token |
| **Phone Number** | Your Twilio voice number (E.164 format) |
| **Default Voice** | TTS voice (e.g., "Polly.Joanna") |
| **Default Language** | Speech recognition language |

5. Click **Save** to create the channel

#### Step 3: Configure Twilio Webhook

1. Go to **Phone Numbers → Manage → Active Numbers**
2. Select your phone number
3. Under **Voice & Fax**:
   - **A Call Comes In**: Webhook, `https://api.your-domain.com/webhooks/voice/{channelId}`
   - **Call Status Changes**: `https://api.your-domain.com/webhooks/voice/{channelId}/status`

### Vonage Voice Setup

#### Step 1: Create Vonage Voice Application

1. Sign up at [vonage.com](https://www.vonage.com/communications-apis/)
2. Go to **Applications** and create a new application
3. Enable **Voice** capability
4. Set Answer URL and Event URL to your Linktor webhooks
5. Generate and download the private key
6. Link a phone number to the application

#### Step 2: Add Channel in Dashboard

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel |
| **Application ID** | Your Vonage Application ID |
| **Private Key** | Your Vonage private key content |
| **Phone Number** | Your Vonage voice number |

## API Usage

### Making Outbound Calls

#### Basic Outbound Call

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

const call = await client.voice.calls.create({
  channelId: 'ch_voice_123',
  to: '+5511999999999',
  flow: 'outbound_welcome' // IVR flow to execute
})

console.log(`Call initiated: ${call.id}, status: ${call.status}`)
```

#### Call with Custom TwiML/NCCO

```typescript
// Twilio TwiML
const call = await client.voice.calls.create({
  channelId: 'ch_voice_123',
  to: '+5511999999999',
  twiml: `
    <Response>
      <Say voice="Polly.Joanna">Hello! This is a reminder about your appointment tomorrow at 3 PM.</Say>
      <Gather input="speech dtmf" timeout="5" numDigits="1">
        <Say>Press 1 to confirm, or 2 to reschedule.</Say>
      </Gather>
    </Response>
  `
})
```

#### Call with Text-to-Speech

```typescript
const call = await client.voice.calls.create({
  channelId: 'ch_voice_123',
  to: '+5511999999999',
  actions: [
    {
      action: 'say',
      text: 'Hello! Your order has been shipped and will arrive tomorrow.',
      voice: 'Polly.Joanna',
      language: 'en-US'
    },
    {
      action: 'say',
      text: 'Press any key to repeat this message, or hang up.',
      voice: 'Polly.Joanna'
    },
    {
      action: 'gather',
      input: ['dtmf'],
      timeout: 10,
      onGather: 'repeat_message'
    }
  ]
})
```

### Handling Inbound Calls

#### IVR Flow Example

Define an IVR flow that handles incoming calls:

```typescript
// In your Linktor flow configuration
{
  "name": "main_ivr",
  "trigger": {
    "type": "voice.inbound",
    "channelId": "ch_voice_123"
  },
  "steps": [
    {
      "id": "welcome",
      "action": "say",
      "text": "Welcome to Acme Support. ",
      "voice": "Polly.Joanna"
    },
    {
      "id": "menu",
      "action": "gather",
      "input": ["dtmf", "speech"],
      "timeout": 5,
      "speechTimeout": "auto",
      "hints": ["sales", "support", "billing"],
      "prompt": {
        "action": "say",
        "text": "For sales, press 1 or say sales. For support, press 2 or say support. For billing, press 3 or say billing."
      },
      "onGather": "route_call"
    },
    {
      "id": "route_call",
      "action": "conditional",
      "conditions": [
        {
          "if": "{{digits}} == '1' || {{speech}} contains 'sales'",
          "then": "transfer_sales"
        },
        {
          "if": "{{digits}} == '2' || {{speech}} contains 'support'",
          "then": "transfer_support"
        },
        {
          "if": "{{digits}} == '3' || {{speech}} contains 'billing'",
          "then": "transfer_billing"
        }
      ],
      "default": "invalid_option"
    },
    {
      "id": "transfer_sales",
      "action": "dial",
      "number": "+5511888888888",
      "callerId": "{{channel.phoneNumber}}",
      "timeout": 30,
      "onNoAnswer": "voicemail"
    }
  ]
}
```

### Speech Recognition (STT)

#### Gather Speech Input

```typescript
await client.voice.calls.update(callId, {
  actions: [
    {
      action: 'say',
      text: 'Please describe your issue after the beep.'
    },
    {
      action: 'gather',
      input: ['speech'],
      speechTimeout: 'auto',
      language: 'en-US',
      hints: ['refund', 'shipping', 'product', 'account'],
      onGather: async (result) => {
        const transcription = result.speechResult
        // Process with AI
        const response = await processWithAI(transcription)
        return {
          action: 'say',
          text: response
        }
      }
    }
  ]
})
```

### Text-to-Speech (TTS)

#### Available Voices

```typescript
// List available voices
const voices = await client.voice.voices.list({
  channelId: 'ch_voice_123',
  language: 'en-US'
})

// Twilio Polly voices
// - Polly.Joanna (female, US)
// - Polly.Matthew (male, US)
// - Polly.Amy (female, UK)
// - Polly.Brian (male, UK)
// - Polly.Camila (female, Portuguese)
```

#### Using Different Voices

```typescript
await client.voice.calls.update(callId, {
  actions: [
    {
      action: 'say',
      text: 'Hello, this is your English assistant.',
      voice: 'Polly.Joanna',
      language: 'en-US'
    },
    {
      action: 'say',
      text: 'Ol\u00e1, sou sua assistente em portugu\u00eas.',
      voice: 'Polly.Camila',
      language: 'pt-BR'
    }
  ]
})
```

### Call Recording

#### Enable Recording

```typescript
const call = await client.voice.calls.create({
  channelId: 'ch_voice_123',
  to: '+5511999999999',
  record: true,
  recordingStatusCallback: 'https://api.your-domain.com/webhooks/voice/recording',
  flow: 'outbound_call'
})
```

#### Retrieve Recordings

```typescript
const recordings = await client.voice.recordings.list({
  callId: call.id
})

recordings.forEach(recording => {
  console.log(`Recording: ${recording.url}, duration: ${recording.duration}s`)
})
```

### Call Transfer

#### Warm Transfer

```typescript
await client.voice.calls.update(callId, {
  actions: [
    {
      action: 'say',
      text: 'Please hold while I transfer you to an agent.'
    },
    {
      action: 'dial',
      number: '+5511888888888',
      callerId: '{{caller.number}}',
      timeout: 30,
      whisper: {
        action: 'say',
        text: 'Incoming call from {{caller.number}} regarding order issue.'
      }
    }
  ]
})
```

#### Cold Transfer

```typescript
await client.voice.calls.transfer(callId, {
  to: '+5511888888888',
  announce: 'Transferring to support team.'
})
```

### AI Voice Bot Integration

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

// Handle incoming call with AI
client.voice.on('call.answered', async (call) => {
  await client.voice.calls.update(call.id, {
    actions: [
      {
        action: 'say',
        text: 'Hello! I am your AI assistant. How can I help you today?'
      },
      {
        action: 'gather',
        input: ['speech'],
        speechTimeout: 'auto',
        onGather: async (result) => {
          // Send to AI bot
          const aiResponse = await client.bots.chat({
            botId: 'bot_ai_123',
            message: result.speechResult,
            context: {
              channel: 'voice',
              callId: call.id,
              caller: call.from
            }
          })

          return {
            action: 'say',
            text: aiResponse.message
          }
        }
      }
    ]
  })
})
```

### Webhook Events

#### Call Events

```typescript
// Incoming call event
{
  "event": "voice.call.incoming",
  "data": {
    "callId": "call_abc123",
    "channelId": "ch_voice_123",
    "direction": "inbound",
    "from": "+5511999999999",
    "to": "+5511888888888",
    "status": "ringing",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}

// Call answered
{
  "event": "voice.call.answered",
  "data": {
    "callId": "call_abc123",
    "status": "in-progress",
    "answeredBy": "human", // or "machine"
    "timestamp": "2024-01-15T10:30:05Z"
  }
}

// Call completed
{
  "event": "voice.call.completed",
  "data": {
    "callId": "call_abc123",
    "status": "completed",
    "duration": 125,
    "recordingUrl": "https://...",
    "timestamp": "2024-01-15T10:32:10Z"
  }
}
```

#### DTMF Events

```typescript
{
  "event": "voice.dtmf",
  "data": {
    "callId": "call_abc123",
    "digits": "1",
    "timestamp": "2024-01-15T10:30:15Z"
  }
}
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/voice/{channelId}
```

### Webhook Security

**Twilio:**
- Validates `X-Twilio-Signature` header
- Use [Request Validator](https://www.twilio.com/docs/usage/security#validating-requests)

**Vonage:**
- Validates JWT signature
- Verify `Authorization` header

## Common Issues and Troubleshooting

### "Call failed to connect"

**Possible causes:**
- Invalid phone number
- Number not voice-capable
- Carrier issues
- Insufficient balance

**Solution:**
- Verify number format (E.164)
- Check number capabilities in provider dashboard
- Test with different numbers
- Check account balance

### Poor Audio Quality

**Possible causes:**
- Network latency
- Codec issues
- TTS voice quality

**Solution:**
- Use high-quality TTS voices (Polly Neural)
- Check network connectivity
- Use appropriate audio codecs

### Speech Recognition Failures

**Possible causes:**
- Background noise
- Accent not recognized
- Wrong language configured

**Solution:**
- Add speech hints for expected words
- Use appropriate language model
- Consider gathering DTMF as fallback

### Webhook Timeout

**Possible causes:**
- Slow webhook response (>15 seconds for Twilio)
- Complex processing

**Solution:**
- Respond immediately with TwiML
- Use async processing for complex logic
- Implement proper caching

### Recording Not Available

**Possible causes:**
- Recording not enabled
- Call too short
- Storage issues

**Solution:**
- Verify recording is enabled
- Check minimum duration requirements
- Verify storage configuration

## Best Practices

1. **Always Identify Yourself**: State who is calling at the start of outbound calls.

2. **Provide Options**: Always give callers a way to reach a human.

3. **Handle Errors Gracefully**: Have fallback flows for unrecognized input.

4. **Keep IVR Menus Short**: Maximum 3-4 options per menu level.

5. **Use Appropriate Voices**: Match voice gender, accent, and language to your audience.

6. **Record with Consent**: Announce recording at the start of calls where required.

7. **Monitor Call Quality**: Track metrics like answer rates, duration, and drop-offs.

8. **Test Regularly**: Call your numbers regularly to verify they're working.

## Compliance

### TCPA (US)

- Get prior express consent before calling
- Honor Do Not Call requests
- Follow calling hour restrictions (8am-9pm local)
- Provide opt-out mechanism

### Recording Consent

- **One-party consent states**: Only one party needs to consent
- **Two-party consent states**: All parties must consent
- Always announce recording to be safe

## Rate Limits

| Provider | Concurrent Calls | Calls per Second |
|----------|------------------|------------------|
| Twilio | Based on CPS | 1 CPS default (can increase) |
| Vonage | Based on account | 1 CPS default |

## Pricing Considerations

Voice calls are billed by:
- **Per-minute rates**: Varies by destination
- **Phone number rental**: Monthly fee
- **Recording storage**: Per minute stored
- **Transcription**: Per minute transcribed

## Next Steps

- [IVR Flows](/flows/ivr) - Build complex IVR systems
- [AI Voice Bots](/bots/voice) - Create AI-powered voice assistants
- [Call Analytics](/api/analytics) - Track voice metrics
- [Recording Guide](/guides/call-recording) - Recording best practices
