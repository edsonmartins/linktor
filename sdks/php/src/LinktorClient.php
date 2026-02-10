<?php

declare(strict_types=1);

namespace Linktor;

use Linktor\Resources\AuthResource;
use Linktor\Resources\ConversationsResource;
use Linktor\Resources\ContactsResource;
use Linktor\Resources\ChannelsResource;
use Linktor\Resources\BotsResource;
use Linktor\Resources\AIResource;
use Linktor\Resources\KnowledgeBasesResource;
use Linktor\Resources\FlowsResource;
use Linktor\Resources\VREResource;
use Linktor\Utils\HttpClient;
use Linktor\Utils\WebhookVerifier;
use Linktor\Types\WebhookEvent;

class LinktorClient
{
    private HttpClient $http;

    public AuthResource $auth;
    public ConversationsResource $conversations;
    public ContactsResource $contacts;
    public ChannelsResource $channels;
    public BotsResource $bots;
    public AIResource $ai;
    public KnowledgeBasesResource $knowledgeBases;
    public FlowsResource $flows;
    public VREResource $vre;

    public function __construct(array $options = [])
    {
        $baseUrl = $options['baseUrl'] ?? $options['base_url'] ?? 'https://api.linktor.io';
        $apiKey = $options['apiKey'] ?? $options['api_key'] ?? null;
        $accessToken = $options['accessToken'] ?? $options['access_token'] ?? null;
        $timeoutSeconds = $options['timeoutSeconds'] ?? $options['timeout'] ?? 30;
        $maxRetries = $options['maxRetries'] ?? $options['max_retries'] ?? 3;

        $this->http = new HttpClient(
            $baseUrl,
            $apiKey,
            $accessToken,
            $timeoutSeconds,
            $maxRetries
        );

        // Initialize resources
        $this->auth = new AuthResource($this);
        $this->conversations = new ConversationsResource($this);
        $this->contacts = new ContactsResource($this);
        $this->channels = new ChannelsResource($this);
        $this->bots = new BotsResource($this);
        $this->ai = new AIResource($this);
        $this->knowledgeBases = new KnowledgeBasesResource($this);
        $this->flows = new FlowsResource($this);
        $this->vre = new VREResource($this);
    }

    public function setAccessToken(?string $token): void
    {
        $this->http->setAccessToken($token);
    }

    // HTTP methods exposed for resources
    public function get(string $path): array
    {
        return $this->http->get($path);
    }

    public function post(string $path, ?array $body = null): array
    {
        return $this->http->post($path, $body);
    }

    public function patch(string $path, array $body): array
    {
        return $this->http->patch($path, $body);
    }

    public function delete(string $path): void
    {
        $this->http->delete($path);
    }

    // Static webhook utilities
    public static function verifyWebhookSignature(string $payload, string $signature, string $secret): bool
    {
        return WebhookVerifier::verifySignature($payload, $signature, $secret);
    }

    public static function computeWebhookSignature(string $payload, string $secret): string
    {
        return WebhookVerifier::computeSignature($payload, $secret);
    }

    public static function constructWebhookEvent(
        string $payload,
        array $headers,
        string $secret,
        ?int $tolerance = null
    ): WebhookEvent {
        return WebhookVerifier::constructEvent($payload, $headers, $secret, $tolerance);
    }
}
