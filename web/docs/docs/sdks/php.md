---
sidebar_position: 8
title: PHP SDK
---

# PHP SDK

The official PHP SDK for Linktor provides a clean, object-oriented interface with full PSR compliance and comprehensive error handling.

## Installation

Install the SDK using Composer:

```bash
composer require linktor/linktor-php
```

## Requirements

- PHP 8.1 or higher
- Composer
- ext-json
- ext-curl (or any PSR-18 HTTP client)

## Quick Start

### Initialize the Client

```php
<?php

require_once 'vendor/autoload.php';

use Linktor\LinktorClient;

$client = new LinktorClient([
    'api_key' => getenv('LINKTOR_API_KEY'),
    // Optional configuration
    'base_url' => 'https://api.linktor.io',
    'timeout' => 30,
    'max_retries' => 3,
]);
```

### Send a Message

```php
// Send a message to a conversation
$message = $client->conversations->sendMessage('conversation-id', [
    'text' => 'Hello from PHP!'
]);

echo "Message sent: " . $message['id'] . "\n";
```

### List Conversations

```php
// Get all conversations with pagination
$conversations = $client->conversations->list([
    'limit' => 20,
    'status' => 'open',
]);

foreach ($conversations['data'] as $conv) {
    echo $conv['id'] . ": " . $conv['contact']['name'] . "\n";
}
```

### Work with Contacts

```php
// Create a new contact
$contact = $client->contacts->create([
    'name' => 'John Doe',
    'email' => 'john@example.com',
    'phone' => '+1234567890',
    'metadata' => [
        'customer_id' => 'cust_123',
    ],
]);

// Search contacts
$results = $client->contacts->list([
    'search' => 'john',
    'limit' => 10,
]);
```

## Real-time Updates (WebSocket)

For PHP applications, WebSocket connections are typically handled through a separate process or using ReactPHP/Ratchet.

### Using ReactPHP

```php
<?php

require_once 'vendor/autoload.php';

use Ratchet\Client\WebSocket;
use React\EventLoop\Loop;

$loop = Loop::get();
$apiKey = getenv('LINKTOR_API_KEY');

$connector = new \Ratchet\Client\Connector($loop);
$connector("wss://api.linktor.io/ws?api_key={$apiKey}")
    ->then(function (WebSocket $conn) {
        echo "Connected!\n";

        // Subscribe to a conversation
        $conn->send(json_encode([
            'type' => 'subscribe',
            'conversationId' => 'conversation-id',
        ]));

        // Handle incoming messages
        $conn->on('message', function ($msg) {
            $data = json_decode($msg, true);

            switch ($data['type']) {
                case 'message':
                    echo "New message: " . $data['message']['text'] . "\n";
                    break;
                case 'message_status':
                    echo "Message " . $data['messageId'] . " status: " . $data['status'] . "\n";
                    break;
                case 'typing':
                    if ($data['isTyping']) {
                        echo "User " . $data['userId'] . " is typing...\n";
                    }
                    break;
            }
        });

        $conn->on('close', function ($code = null, $reason = null) {
            echo "Connection closed: {$code} - {$reason}\n";
        });

    }, function ($e) {
        echo "Could not connect: {$e->getMessage()}\n";
    });

$loop->run();
```

### Webhook-based Real-time

For most PHP applications, webhooks are the recommended approach for receiving real-time updates:

```php
<?php

require_once 'vendor/autoload.php';

use Linktor\LinktorClient;

// Receive webhook POST
$payload = file_get_contents('php://input');
$headers = getallheaders();

$signature = $headers['X-Linktor-Signature'] ?? '';
$webhookSecret = getenv('WEBHOOK_SECRET');

// Verify the webhook
if (!LinktorClient::verifyWebhookSignature($payload, $signature, $webhookSecret)) {
    http_response_code(400);
    echo json_encode(['error' => 'Invalid signature']);
    exit;
}

// Parse and handle the event
$event = json_decode($payload, true);

switch ($event['type']) {
    case 'message.received':
        // Handle new message
        handleNewMessage($event['data']);
        break;
    case 'conversation.created':
        // Handle new conversation
        handleNewConversation($event['data']);
        break;
}

http_response_code(200);
echo json_encode(['received' => true]);
```

## Error Handling

The SDK provides a `LinktorException` class for handling API errors.

```php
<?php

use Linktor\LinktorClient;
use Linktor\Utils\LinktorException;

$client = new LinktorClient([
    'api_key' => 'your-api-key',
]);

try {
    $conversation = $client->conversations->get('invalid-id');
} catch (LinktorException $e) {
    switch ($e->getStatusCode()) {
        case 404:
            echo "Conversation not found\n";
            break;
        case 401:
            echo "Invalid API key\n";
            break;
        case 429:
            echo "Rate limited. Retry after: " . $e->getRetryAfter() . " seconds\n";
            break;
        case 400:
            echo "Invalid request: " . json_encode($e->getDetails()) . "\n";
            break;
        default:
            echo "API error [{$e->getCode()}]: {$e->getMessage()}\n";
            echo "Request ID: " . $e->getRequestId() . "\n";
    }
}
```

### Exception Methods

| Method | Return Type | Description |
|--------|-------------|-------------|
| `getCode()` | string | Error code (e.g., "NOT_FOUND") |
| `getMessage()` | string | Error message |
| `getStatusCode()` | int | HTTP status code |
| `getRequestId()` | string\|null | Request ID for debugging |
| `getDetails()` | array | Additional error details |
| `getRetryAfter()` | int\|null | Seconds to wait before retrying |

## Webhook Verification

Verify incoming webhooks from Linktor.

### Plain PHP

```php
<?php

require_once 'vendor/autoload.php';

use Linktor\LinktorClient;

$payload = file_get_contents('php://input');
$signature = $_SERVER['HTTP_X_LINKTOR_SIGNATURE'] ?? '';
$webhookSecret = getenv('WEBHOOK_SECRET');

// Verify the webhook signature
if (!LinktorClient::verifyWebhookSignature($payload, $signature, $webhookSecret)) {
    http_response_code(400);
    echo json_encode(['error' => 'Invalid signature']);
    exit;
}

// Construct and handle the event
$headers = [
    'X-Linktor-Signature' => $signature,
    'X-Linktor-Timestamp' => $_SERVER['HTTP_X_LINKTOR_TIMESTAMP'] ?? '',
];

$event = LinktorClient::constructWebhookEvent($payload, $headers, $webhookSecret);

// Handle the event
switch ($event->type) {
    case 'message.received':
        // Handle new message
        break;
    case 'conversation.created':
        // Handle new conversation
        break;
}

echo json_encode(['received' => true]);
```

### Laravel

```php
<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use Linktor\LinktorClient;

class WebhookController extends Controller
{
    public function handle(Request $request)
    {
        $payload = $request->getContent();
        $signature = $request->header('X-Linktor-Signature');
        $webhookSecret = config('services.linktor.webhook_secret');

        // Verify the webhook signature
        if (!LinktorClient::verifyWebhookSignature($payload, $signature, $webhookSecret)) {
            return response()->json(['error' => 'Invalid signature'], 400);
        }

        $event = $request->all();

        // Handle different event types
        match ($event['type']) {
            'message.received' => $this->handleNewMessage($event['data']),
            'conversation.created' => $this->handleNewConversation($event['data']),
            default => null,
        };

        return response()->json(['received' => true]);
    }
}
```

### Symfony

```php
<?php

namespace App\Controller;

use Linktor\LinktorClient;
use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\HttpFoundation\JsonResponse;
use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\Routing\Annotation\Route;

class WebhookController extends AbstractController
{
    #[Route('/webhook', methods: ['POST'])]
    public function handle(Request $request): JsonResponse
    {
        $payload = $request->getContent();
        $signature = $request->headers->get('X-Linktor-Signature');
        $webhookSecret = $this->getParameter('linktor.webhook_secret');

        // Verify the webhook signature
        if (!LinktorClient::verifyWebhookSignature($payload, $signature, $webhookSecret)) {
            return $this->json(['error' => 'Invalid signature'], 400);
        }

        $event = json_decode($payload, true);

        // Handle the event...

        return $this->json(['received' => true]);
    }
}
```

## AI Features

### Completions

```php
// Create a completion
$response = $client->ai->completions->create([
    'messages' => [
        ['role' => 'system', 'content' => 'You are a helpful assistant.'],
        ['role' => 'user', 'content' => 'What is the capital of France?'],
    ],
    'model' => 'gpt-4',
]);

echo $response['message']['content'];

// Simple helper method
$answer = $client->ai->completions->complete('What is 2 + 2?');
echo $answer;  // "4"
```

### Knowledge Bases

```php
// Query a knowledge base
$result = $client->knowledgeBases->query('kb-id', [
    'query' => 'How do I reset my password?',
    'topK' => 5,
]);

foreach ($result['chunks'] as $chunk) {
    echo "Match: " . $chunk['content'] . "\n";
    echo "Score: " . $chunk['score'] . "\n";
}

// Simple search helper
$texts = $client->knowledgeBases->search('kb-id', 'password reset');
foreach ($texts as $text) {
    echo $text . "\n";
}
```

### Embeddings

```php
// Create embeddings
$embedding = $client->ai->embeddings->embed('Hello, world!');
echo "Embedding dimension: " . count($embedding) . "\n";
```

## Resources

The SDK provides access to all Linktor resources:

- `$client->auth` - Authentication
- `$client->conversations` - Conversations and messages
- `$client->contacts` - Contact management
- `$client->channels` - Channel configuration
- `$client->bots` - Bot management
- `$client->ai` - AI completions and embeddings
- `$client->knowledgeBases` - Knowledge base operations
- `$client->flows` - Conversation flows

## Laravel Integration

Create a service provider for Laravel:

```php
<?php

namespace App\Providers;

use Illuminate\Support\ServiceProvider;
use Linktor\LinktorClient;

class LinktorServiceProvider extends ServiceProvider
{
    public function register()
    {
        $this->app->singleton(LinktorClient::class, function ($app) {
            return new LinktorClient([
                'api_key' => config('services.linktor.api_key'),
                'base_url' => config('services.linktor.base_url', 'https://api.linktor.io'),
            ]);
        });
    }
}

// Usage in a controller
class ChatController extends Controller
{
    public function __construct(
        private LinktorClient $linktor
    ) {}

    public function send(Request $request)
    {
        $message = $this->linktor->conversations->sendMessage(
            $request->conversation_id,
            ['text' => $request->message]
        );

        return response()->json($message);
    }
}
```

## API Reference

For complete API documentation, see the [PHP SDK Reference](https://github.com/linktor/linktor/tree/main/sdks/php).
