<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\Bot;
use Linktor\Types\BotStatus;
use Linktor\Types\PaginatedResponse;

class BotsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function list(array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/bots{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($b) => Bot::fromArray($b), $response->data);
        return $response;
    }

    public function get(string $id): Bot
    {
        $data = $this->client->get("/bots/{$id}");
        return Bot::fromArray($data);
    }

    public function create(array $input): Bot
    {
        $data = $this->client->post('/bots', $input);
        return Bot::fromArray($data);
    }

    public function update(string $id, array $input): Bot
    {
        $data = $this->client->patch("/bots/{$id}", $input);
        return Bot::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/bots/{$id}");
    }

    public function start(string $id): Bot
    {
        $data = $this->client->post("/bots/{$id}/start", []);
        return Bot::fromArray($data);
    }

    public function stop(string $id): Bot
    {
        $data = $this->client->post("/bots/{$id}/stop", []);
        return Bot::fromArray($data);
    }

    public function getStatus(string $id): BotStatus
    {
        $data = $this->client->get("/bots/{$id}/status");
        return BotStatus::fromArray($data);
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
