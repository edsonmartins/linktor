---
sidebar_position: 2
title: Node Types
---

# Node Types

Flows are built using different types of nodes, each serving a specific purpose. This guide covers all available node types and their configuration options.

## Node Categories

| Category | Purpose |
|----------|---------|
| **Message** | Send content to customers |
| **Input** | Collect information from customers |
| **Logic** | Control flow direction |
| **Action** | Execute integrations and automations |
| **AI** | Incorporate AI-powered responses |
| **Control** | Manage flow state |

## Message Nodes

### Text Message

Send a text message to the customer.

```typescript
{
  id: 'welcome_msg',
  type: 'message',
  content: 'Hello! Welcome to Acme Support. How can I help you today?',
  next: 'main_menu'
}
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `content` | string | Message text (supports variables) |
| `delay` | number | Delay before sending (ms) |
| `typing` | boolean | Show typing indicator |
| `next` | string | Next node ID |

### Rich Message

Send messages with buttons, images, or cards.

```typescript
{
  id: 'product_card',
  type: 'rich_message',
  content: {
    type: 'card',
    title: 'Premium Plan',
    subtitle: '$99/month',
    imageUrl: 'https://cdn.example.com/premium.png',
    buttons: [
      { label: 'Learn More', action: 'goto', target: 'premium_details' },
      { label: 'Buy Now', action: 'url', target: 'https://example.com/buy' }
    ]
  },
  next: 'after_card'
}
```

**Content Types:**

#### Buttons

```typescript
{
  type: 'buttons',
  text: 'How would you like to proceed?',
  buttons: [
    { label: 'Option A', value: 'a' },
    { label: 'Option B', value: 'b' },
    { label: 'Option C', value: 'c' }
  ]
}
```

#### Quick Replies

```typescript
{
  type: 'quick_replies',
  text: 'Choose a topic:',
  replies: ['Sales', 'Support', 'Billing', 'Other']
}
```

#### Carousel

```typescript
{
  type: 'carousel',
  cards: [
    {
      title: 'Product 1',
      subtitle: 'Description',
      imageUrl: '...',
      buttons: [{ label: 'Select', value: 'product_1' }]
    },
    {
      title: 'Product 2',
      subtitle: 'Description',
      imageUrl: '...',
      buttons: [{ label: 'Select', value: 'product_2' }]
    }
  ]
}
```

#### List

```typescript
{
  type: 'list',
  header: 'Our Services',
  items: [
    { title: 'Consulting', description: 'Expert advice', value: 'consulting' },
    { title: 'Training', description: 'Learn new skills', value: 'training' },
    { title: 'Support', description: '24/7 assistance', value: 'support' }
  ]
}
```

### Media Message

Send images, documents, audio, or video.

```typescript
{
  id: 'send_brochure',
  type: 'media',
  media: {
    type: 'document',
    url: 'https://cdn.example.com/brochure.pdf',
    filename: 'Product_Brochure.pdf',
    caption: 'Here is our product brochure'
  },
  next: 'after_brochure'
}
```

**Media Types:**

| Type | Properties |
|------|------------|
| `image` | `url`, `caption` |
| `video` | `url`, `caption`, `thumbnail` |
| `audio` | `url` |
| `document` | `url`, `filename`, `caption` |
| `sticker` | `url` |
| `location` | `latitude`, `longitude`, `name`, `address` |

## Input Nodes

### Question

Ask a question and store the response.

```typescript
{
  id: 'ask_email',
  type: 'question',
  content: 'What is your email address?',
  variable: 'customer_email',
  validation: {
    type: 'email',
    errorMessage: 'Please enter a valid email address.'
  },
  next: 'confirm_email'
}
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `content` | string | Question text |
| `variable` | string | Variable name to store response |
| `validation` | object | Input validation rules |
| `required` | boolean | Whether answer is required |
| `retryLimit` | number | Max retry attempts |
| `timeout` | number | Response timeout (ms) |
| `timeoutNode` | string | Node to go to on timeout |

**Validation Types:**

```typescript
// Text validation
{ type: 'text', minLength: 2, maxLength: 100 }

// Email validation
{ type: 'email' }

// Phone validation
{ type: 'phone', countryCode: 'BR' }

// Number validation
{ type: 'number', min: 0, max: 100 }

// Date validation
{ type: 'date', format: 'YYYY-MM-DD', minDate: 'today' }

// Regex validation
{ type: 'regex', pattern: '^[A-Z]{2}\\d{6}$', errorMessage: 'Invalid format' }

// Custom validation (via webhook)
{ type: 'custom', webhookUrl: 'https://api.example.com/validate' }
```

### Choice Question

Present options and capture selection.

```typescript
{
  id: 'ask_department',
  type: 'choice',
  content: 'Which department do you need?',
  variable: 'department',
  options: [
    { label: 'Sales', value: 'sales', next: 'sales_flow' },
    { label: 'Support', value: 'support', next: 'support_flow' },
    { label: 'Billing', value: 'billing', next: 'billing_flow' }
  ],
  displayAs: 'buttons',  // 'buttons' | 'quick_replies' | 'list'
  allowFreeText: false
}
```

### Multi-Select Question

Allow multiple selections.

```typescript
{
  id: 'ask_interests',
  type: 'multi_select',
  content: 'Select all topics you are interested in:',
  variable: 'interests',
  options: [
    { label: 'Product Updates', value: 'updates' },
    { label: 'Industry News', value: 'news' },
    { label: 'Tips & Tricks', value: 'tips' },
    { label: 'Events', value: 'events' }
  ],
  minSelections: 1,
  maxSelections: 3,
  next: 'confirm_interests'
}
```

### File Upload

Request a file from the customer.

```typescript
{
  id: 'upload_document',
  type: 'file_upload',
  content: 'Please upload a photo of your ID document.',
  variable: 'id_document',
  allowedTypes: ['image/jpeg', 'image/png', 'application/pdf'],
  maxSize: 10 * 1024 * 1024,  // 10MB
  next: 'verify_document'
}
```

### Location Request

Request the customer's location.

```typescript
{
  id: 'ask_location',
  type: 'location_request',
  content: 'Please share your location so we can find nearby stores.',
  variable: 'customer_location',
  next: 'find_stores'
}
```

## Logic Nodes

### Condition

Branch based on conditions.

```typescript
{
  id: 'check_tier',
  type: 'condition',
  conditions: [
    {
      if: '{{customer_tier}} == "enterprise"',
      then: 'enterprise_support'
    },
    {
      if: '{{customer_tier}} == "pro"',
      then: 'pro_support'
    },
    {
      else: 'standard_support'
    }
  ]
}
```

**Operators:**

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal | `{{tier}} == "pro"` |
| `!=` | Not equal | `{{status}} != "active"` |
| `>` | Greater than | `{{age}} > 18` |
| `>=` | Greater or equal | `{{score}} >= 80` |
| `<` | Less than | `{{items}} < 5` |
| `<=` | Less or equal | `{{total}} <= 100` |
| `contains` | String contains | `{{email}} contains "@gmail"` |
| `startsWith` | String starts with | `{{phone}} startsWith "+1"` |
| `endsWith` | String ends with | `{{email}} endsWith ".com"` |
| `in` | Value in list | `{{choice}} in ["a", "b", "c"]` |
| `notIn` | Value not in list | `{{status}} notIn ["banned"]` |
| `isEmpty` | Value is empty | `{{name}} isEmpty` |
| `isNotEmpty` | Value is not empty | `{{email}} isNotEmpty` |
| `matches` | Regex match | `{{code}} matches "^[A-Z]{3}\\d{3}$"` |

**Combining Conditions:**

```typescript
{
  if: '{{age}} >= 18 AND {{country}} == "US"',
  then: 'eligible_us'
}
```

```typescript
{
  if: '{{tier}} == "enterprise" OR {{spending}} > 10000',
  then: 'vip_treatment'
}
```

### Switch

Multi-way branch based on a value.

```typescript
{
  id: 'route_by_language',
  type: 'switch',
  variable: '{{detected_language}}',
  cases: {
    'en': 'english_flow',
    'es': 'spanish_flow',
    'pt': 'portuguese_flow',
    'fr': 'french_flow'
  },
  default: 'english_flow'
}
```

### Random Split

Randomly route to different paths (useful for A/B testing).

```typescript
{
  id: 'ab_test',
  type: 'random_split',
  splits: [
    { weight: 50, next: 'variant_a' },
    { weight: 50, next: 'variant_b' }
  ]
}
```

### Wait

Pause the flow for a duration or until an event.

```typescript
// Time-based wait
{
  id: 'wait_delay',
  type: 'wait',
  duration: 60000,  // 60 seconds
  next: 'follow_up'
}

// Event-based wait
{
  id: 'wait_response',
  type: 'wait',
  event: 'user_message',
  timeout: 300000,  // 5 minutes
  timeoutNode: 'no_response',
  next: 'process_response'
}
```

## Action Nodes

### HTTP Request

Make API calls to external services.

```typescript
{
  id: 'lookup_order',
  type: 'http_request',
  request: {
    method: 'GET',
    url: 'https://api.example.com/orders/{{order_id}}',
    headers: {
      'Authorization': 'Bearer {{API_TOKEN}}'
    }
  },
  response: {
    variable: 'order_data',
    mapping: {
      'status': 'order_status',
      'items': 'order_items',
      'total': 'order_total'
    }
  },
  errorNode: 'order_not_found',
  next: 'show_order'
}
```

**Request Options:**

```typescript
{
  method: 'POST',
  url: 'https://api.example.com/leads',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer {{token}}'
  },
  body: {
    name: '{{customer_name}}',
    email: '{{customer_email}}',
    source: 'chatbot'
  },
  timeout: 10000,
  retries: 3
}
```

### Set Variable

Set or modify variables.

```typescript
{
  id: 'set_vars',
  type: 'set_variable',
  variables: {
    'full_name': '{{first_name}} {{last_name}}',
    'is_qualified': true,
    'score': '{{score}} + 10',
    'tags': ['lead', 'interested']
  },
  next: 'continue'
}
```

### Send Email

Send an email notification.

```typescript
{
  id: 'send_confirmation',
  type: 'send_email',
  email: {
    to: '{{customer_email}}',
    subject: 'Your request has been received',
    template: 'confirmation_template',
    data: {
      name: '{{customer_name}}',
      ticketId: '{{ticket_id}}'
    }
  },
  next: 'email_sent'
}
```

### Create Record

Create a record in Linktor or external CRM.

```typescript
{
  id: 'create_lead',
  type: 'create_record',
  record: {
    type: 'contact',
    data: {
      name: '{{customer_name}}',
      email: '{{customer_email}}',
      phone: '{{customer_phone}}',
      tags: ['chatbot-lead'],
      customFields: {
        source: 'chatbot',
        interest: '{{product_interest}}'
      }
    }
  },
  resultVariable: 'created_contact',
  next: 'lead_created'
}
```

### Assign Conversation

Route the conversation.

```typescript
{
  id: 'route_to_sales',
  type: 'assign',
  assignment: {
    type: 'inbox',    // 'inbox' | 'agent' | 'team' | 'bot'
    id: 'inbox_sales',
    priority: 'high',
    note: 'Qualified lead from chatbot'
  },
  next: 'handoff_message'
}
```

### Tag Conversation

Add or remove tags.

```typescript
{
  id: 'tag_vip',
  type: 'tag',
  addTags: ['vip', 'high-value'],
  removeTags: ['new-lead'],
  next: 'continue'
}
```

### Webhook

Trigger a custom webhook.

```typescript
{
  id: 'notify_webhook',
  type: 'webhook',
  webhook: {
    url: 'https://hooks.example.com/linktor',
    method: 'POST',
    payload: {
      event: 'flow_completed',
      conversationId: '{{conversation.id}}',
      data: {
        name: '{{customer_name}}',
        email: '{{customer_email}}'
      }
    }
  },
  async: true,  // Don't wait for response
  next: 'continue'
}
```

## AI Nodes

### AI Response

Generate an AI-powered response.

```typescript
{
  id: 'ai_answer',
  type: 'ai_response',
  botId: 'bot_support',
  prompt: 'Answer the customer question based on their message: {{last_message}}',
  context: {
    includeKnowledgeBase: true,
    includeConversationHistory: true,
    maxTokens: 500
  },
  next: 'after_ai'
}
```

### AI Classification

Use AI to classify user input.

```typescript
{
  id: 'classify_intent',
  type: 'ai_classify',
  input: '{{last_message}}',
  categories: [
    { name: 'sales', description: 'Questions about pricing, features, or purchasing' },
    { name: 'support', description: 'Technical issues or help requests' },
    { name: 'billing', description: 'Payment, invoices, or subscription questions' },
    { name: 'other', description: 'Anything else' }
  ],
  resultVariable: 'intent',
  confidenceVariable: 'intent_confidence',
  next: 'route_by_intent'
}
```

### AI Extraction

Extract structured data from text.

```typescript
{
  id: 'extract_info',
  type: 'ai_extract',
  input: '{{last_message}}',
  schema: {
    productName: { type: 'string', description: 'Name of the product mentioned' },
    quantity: { type: 'number', description: 'Quantity requested' },
    urgency: { type: 'string', enum: ['low', 'medium', 'high'], description: 'How urgent is the request' }
  },
  resultVariable: 'extracted_data',
  next: 'process_order'
}
```

### AI Sentiment

Analyze sentiment of user message.

```typescript
{
  id: 'analyze_sentiment',
  type: 'ai_sentiment',
  input: '{{last_message}}',
  resultVariable: 'sentiment_score',  // -1 to 1
  labelVariable: 'sentiment_label',   // 'negative', 'neutral', 'positive'
  next: 'check_sentiment'
}
```

## Control Nodes

### Start

The entry point of a flow (automatically added).

```typescript
{
  id: 'start',
  type: 'start',
  next: 'welcome'
}
```

### End

Terminates the flow.

```typescript
{
  id: 'end_success',
  type: 'end',
  status: 'completed',  // 'completed' | 'abandoned' | 'error'
  message: 'Thanks for chatting! Have a great day.',
  analytics: {
    outcome: 'lead_qualified',
    score: '{{qualification_score}}'
  }
}
```

### Jump

Jump to another node or flow.

```typescript
// Jump to node in same flow
{
  id: 'retry',
  type: 'jump',
  target: 'ask_email'
}

// Jump to different flow
{
  id: 'switch_flow',
  type: 'jump',
  flow: 'flow_support',
  node: 'start',
  preserveVariables: true
}
```

### Sub-Flow

Execute another flow as a sub-routine.

```typescript
{
  id: 'collect_address',
  type: 'sub_flow',
  flowId: 'flow_address_collection',
  inputVariables: {
    'customer_name': '{{name}}'
  },
  outputVariables: {
    'collected_address': 'address'
  },
  next: 'confirm_order'
}
```

### Loop

Repeat a section of the flow.

```typescript
{
  id: 'item_loop',
  type: 'loop',
  collection: '{{cart_items}}',
  itemVariable: 'current_item',
  indexVariable: 'item_index',
  bodyStartNode: 'show_item',
  bodyEndNode: 'item_end',
  next: 'checkout'
}
```

## Node Properties Reference

### Common Properties

All nodes share these properties:

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `type` | string | Node type |
| `next` | string | Next node ID |
| `metadata` | object | Custom metadata for analytics |

### Error Handling

Configure error behavior:

```typescript
{
  id: 'api_call',
  type: 'http_request',
  // ... request config
  errorHandling: {
    onError: 'goto_node',    // 'goto_node' | 'retry' | 'continue' | 'end'
    errorNode: 'error_handler',
    retryCount: 3,
    retryDelay: 1000
  }
}
```

## Next Steps

- [Triggers](/flows/triggers) - Learn how to start flows
- [Flows Overview](/flows/overview) - Flow concepts and best practices
- [Bots Overview](/bots/overview) - Combine flows with AI bots
