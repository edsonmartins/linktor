<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\Agent;
use Linktor\Types\PaginatedResponse;

class AIResource
{
    private LinktorClient $client;
    public AgentsResource $agents;
    public CompletionsResource $completions;
    public EmbeddingsResource $embeddings;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
        $this->agents = new AgentsResource($client);
        $this->completions = new CompletionsResource($client);
        $this->embeddings = new EmbeddingsResource($client);
    }
}

class AgentsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function list(array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/ai/agents{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($a) => Agent::fromArray($a), $response->data);
        return $response;
    }

    public function get(string $id): Agent
    {
        $data = $this->client->get("/ai/agents/{$id}");
        return Agent::fromArray($data);
    }

    public function create(array $input): Agent
    {
        $data = $this->client->post('/ai/agents', $input);
        return Agent::fromArray($data);
    }

    public function update(string $id, array $input): Agent
    {
        $data = $this->client->patch("/ai/agents/{$id}", $input);
        return Agent::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/ai/agents/{$id}");
    }

    public function invoke(string $id, array $input): array
    {
        return $this->client->post("/ai/agents/{$id}/invoke", $input);
    }

    private function buildQuery(array $params): string
    {
        if (empty($params)) {
            return '';
        }

        $parts = [];
        foreach ($params as $key => $value) {
            if ($value !== null) {
                $parts[] = urlencode($key) . '=' . urlencode((string) $value);
            }
        }

        return empty($parts) ? '' : '?' . implode('&', $parts);
    }
}

class CompletionsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function create(array $input): array
    {
        return $this->client->post('/ai/completions', $input);
    }

    public function chat(array $input): array
    {
        return $this->client->post('/ai/completions/chat', $input);
    }
}

class EmbeddingsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function create(array $input): array
    {
        return $this->client->post('/ai/embeddings', $input);
    }

    public function search(array $input): array
    {
        return $this->client->post('/ai/embeddings/search', $input);
    }
}
