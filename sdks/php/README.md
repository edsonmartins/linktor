# Linktor PHP SDK

Official PHP SDK for the Linktor API.

## Requirements

- PHP 8.1 or later
- Composer

## Installation

```bash
composer require linktor/linktor-php
```

## Quick Start

```php
<?php

require_once 'vendor/autoload.php';

use Linktor\LinktorClient;

// Initialize the client
$client = new LinktorClient([
    'base_url' => 'https://api.linktor.io',
    'api_key' => 'your-api-key',
]);

// List conversations
$conversations = $client->conversations->list(['limit' => 10]);
foreach ($conversations->data as $conversation) {
    echo "Conversation: {$conversation->id}\n";
}

// Send a message
$message = $client->conversations->sendMessage($conversationId, [
    'text' => 'Hello from PHP!',
]);
```

## Authentication

### API Key Authentication

```php
$client = new LinktorClient([
    'api_key' => 'your-api-key',
]);
```

### JWT Authentication

```php
$client = new LinktorClient([
    'base_url' => 'https://api.linktor.io',
]);

// Login to get access token
$loginResponse = $client->auth->login('user@example.com', 'password');
// Access token is automatically set on the client

// Get current user
$user = $client->auth->getCurrentUser();

// Refresh token when needed
$refreshResponse = $client->auth->refreshToken($loginResponse->refreshToken);
```

## Resources

### Conversations

```php
// List conversations
$conversations = $client->conversations->list([
    'limit' => 20,
    'status' => 'open',
    'channelId' => 'channel-id',
]);

// Get a conversation
$conversation = $client->conversations->get('conversation-id');

// Create a conversation
$newConversation = $client->conversations->create([
    'channelId' => 'channel-id',
    'contactId' => 'contact-id',
]);

// Send a message
$message = $client->conversations->sendMessage('conversation-id', [
    'text' => 'Hello!',
    'quickReplies' => [
        ['title' => 'Yes', 'payload' => 'yes'],
        ['title' => 'No', 'payload' => 'no'],
    ],
]);

// Get messages
$messages = $client->conversations->getMessages('conversation-id', ['limit' => 50]);

// Assign conversation
$client->conversations->assign('conversation-id', 'user-id');

// Resolve conversation
$client->conversations->resolve('conversation-id');
```

### Contacts

```php
// List contacts
$contacts = $client->contacts->list([
    'limit' => 20,
    'search' => 'john',
]);

// Create a contact
$contact = $client->contacts->create([
    'name' => 'John Doe',
    'email' => 'john@example.com',
    'phone' => '+1234567890',
    'tags' => ['vip', 'enterprise'],
    'customFields' => [
        'company' => 'Acme Inc',
        'plan' => 'premium',
    ],
]);

// Search contacts
$searchResults = $client->contacts->search([
    'query' => 'enterprise',
    'filters' => [
        'tags' => ['vip'],
    ],
]);

// Merge contacts
$mergedContact = $client->contacts->merge('primary-contact-id', ['duplicate-1', 'duplicate-2']);
```

### Channels

```php
// List channels
$channels = $client->channels->list(['type' => 'whatsapp']);

// Create a channel
$channel = $client->channels->create([
    'name' => 'WhatsApp Business',
    'type' => 'whatsapp',
    'config' => [
        'phoneNumberId' => 'your-phone-number-id',
        'accessToken' => 'your-access-token',
    ],
]);

// Connect channel
$client->channels->connect('channel-id');

// Get channel status
$status = $client->channels->getStatus('channel-id');
echo "Connected: " . ($status->isConnected ? 'Yes' : 'No') . "\n";
```

### Bots

```php
// List bots
$bots = $client->bots->list();

// Create a bot
$bot = $client->bots->create([
    'name' => 'Support Bot',
    'agentId' => 'agent-id',
    'channelIds' => ['channel-1', 'channel-2'],
]);

// Start bot
$client->bots->start('bot-id');

// Get bot status
$botStatus = $client->bots->getStatus('bot-id');
echo "Active conversations: {$botStatus->activeConversations}\n";
```

### AI

#### Agents

```php
// List agents
$agents = $client->ai->agents->list();

// Create an agent
$agent = $client->ai->agents->create([
    'name' => 'Customer Support Agent',
    'systemPrompt' => 'You are a helpful customer support agent...',
    'model' => 'gpt-4',
    'knowledgeBaseIds' => ['kb-id'],
]);

// Invoke agent
$response = $client->ai->agents->invoke('agent-id', [
    'message' => 'How do I reset my password?',
    'conversationId' => 'conversation-id',
]);
echo $response['response'];
```

#### Completions

```php
// Simple completion
$completion = $client->ai->completions->create([
    'prompt' => 'Translate to Spanish: Hello, how are you?',
    'maxTokens' => 100,
]);

// Chat completion
$chat = $client->ai->completions->chat([
    'messages' => [
        ['role' => 'user', 'content' => 'What is the capital of France?'],
    ],
    'maxTokens' => 100,
]);
```

#### Embeddings

```php
// Create embeddings
$embeddings = $client->ai->embeddings->create([
    'texts' => ['Hello world', 'How are you?'],
]);

// Similarity search
$searchResults = $client->ai->embeddings->search([
    'query' => 'password reset',
    'knowledgeBaseId' => 'kb-id',
    'topK' => 5,
]);
```

### Knowledge Bases

```php
// Create a knowledge base
$kb = $client->knowledgeBases->create([
    'name' => 'Product Documentation',
    'description' => 'Documentation for our products',
]);

// Add a document
$doc = $client->knowledgeBases->addDocument('kb-id', [
    'title' => 'Getting Started Guide',
    'content' => 'This guide will help you get started...',
]);

// Query the knowledge base
$results = $client->knowledgeBases->query('kb-id', [
    'query' => 'how to get started',
    'topK' => 5,
    'includeContent' => true,
]);

foreach ($results['results'] as $result) {
    echo "Score: {$result['score']}, Content: {$result['content']}\n";
}
```

### Flows

```php
// List flows
$flows = $client->flows->list();

// Create a flow
$flow = $client->flows->create([
    'name' => 'Welcome Flow',
    'nodes' => [
        [
            'id' => 'start',
            'type' => 'trigger',
            'data' => ['event' => 'conversation.created'],
        ],
        [
            'id' => 'welcome',
            'type' => 'send_message',
            'data' => ['text' => 'Welcome!'],
        ],
    ],
    'edges' => [
        ['source' => 'start', 'target' => 'welcome'],
    ],
]);

// Validate flow
$validation = $client->flows->validate('flow-id');
if (!$validation['isValid']) {
    foreach ($validation['errors'] ?? [] as $error) {
        echo "Error in {$error['nodeId']}: {$error['message']}\n";
    }
}

// Publish flow
$client->flows->publish('flow-id');

// Execute flow manually
$execution = $client->flows->execute('flow-id', [
    'conversationId' => 'conversation-id',
    'variables' => [
        'customVariable' => 'value',
    ],
]);
```

## Webhook Verification

```php
use Linktor\LinktorClient;
use Linktor\Utils\WebhookVerificationException;
use Linktor\Types\EventType;

// In your webhook handler (e.g., Laravel, Symfony)
$payload = file_get_contents('php://input');

$headers = [
    'X-Linktor-Signature' => $_SERVER['HTTP_X_LINKTOR_SIGNATURE'] ?? '',
    'X-Linktor-Timestamp' => $_SERVER['HTTP_X_LINKTOR_TIMESTAMP'] ?? '',
];

try {
    $webhookEvent = LinktorClient::constructWebhookEvent(
        $payload,
        $headers,
        'your-webhook-secret',
        300 // tolerance in seconds
    );

    // Handle the event
    switch ($webhookEvent->getEventType()) {
        case EventType::MessageReceived:
            // Handle new message
            break;
        case EventType::ConversationCreated:
            // Handle new conversation
            break;
    }

    http_response_code(200);
    echo json_encode(['status' => 'ok']);
} catch (WebhookVerificationException $e) {
    http_response_code(400);
    echo json_encode(['error' => $e->getMessage()]);
}

// Verify signature only
$isValid = LinktorClient::verifyWebhookSignature(
    $payload,
    $signature,
    'your-webhook-secret'
);

// Compute signature (for testing)
$signature = LinktorClient::computeWebhookSignature($payload, 'your-secret');
```

### Laravel Integration

```php
// app/Http/Controllers/WebhookController.php
namespace App\Http\Controllers;

use Illuminate\Http\Request;
use Linktor\LinktorClient;
use Linktor\Utils\WebhookVerificationException;

class WebhookController extends Controller
{
    public function handle(Request $request)
    {
        $payload = $request->getContent();
        $headers = [
            'X-Linktor-Signature' => $request->header('X-Linktor-Signature'),
            'X-Linktor-Timestamp' => $request->header('X-Linktor-Timestamp'),
        ];

        try {
            $event = LinktorClient::constructWebhookEvent(
                $payload,
                $headers,
                config('services.linktor.webhook_secret')
            );

            // Dispatch to job or handle inline
            dispatch(new ProcessWebhookEvent($event));

            return response()->json(['status' => 'ok']);
        } catch (WebhookVerificationException $e) {
            return response()->json(['error' => $e->getMessage()], 400);
        }
    }
}
```

## Error Handling

```php
use Linktor\Utils\LinktorException;
use Linktor\Utils\NotFoundException;
use Linktor\Utils\AuthenticationException;
use Linktor\Utils\RateLimitException;
use Linktor\Utils\ValidationException;

try {
    $conversation = $client->conversations->get('non-existent-id');
} catch (NotFoundException $e) {
    echo "Not found: {$e->getMessage()}\n";
    echo "Request ID: {$e->getRequestId()}\n";
} catch (AuthenticationException $e) {
    echo "Invalid credentials\n";
} catch (RateLimitException $e) {
    echo "Rate limited. Retry after: {$e->getRetryAfter()} seconds\n";
} catch (ValidationException $e) {
    echo "Validation error: {$e->getMessage()}\n";
} catch (LinktorException $e) {
    echo "API error: {$e->getMessage()}\n";
    echo "Status code: {$e->getStatusCode()}\n";
    echo "Error code: {$e->getErrorCode()}\n";
}
```

## Configuration Options

```php
$client = new LinktorClient([
    'base_url' => 'https://api.linktor.io',  // API base URL
    'api_key' => 'your-api-key',              // API key for authentication
    'access_token' => 'your-access-token',    // Or use access token directly
    'timeout' => 30,                          // Request timeout in seconds (default: 30)
    'max_retries' => 3,                       // Max retry attempts (default: 3)
]);

// Snake_case options are also supported
$client = new LinktorClient([
    'base_url' => 'https://api.linktor.io',
    'api_key' => 'your-api-key',
]);
```

## License

MIT License - see LICENSE file for details.
