<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\Channel;
use Linktor\Types\ChannelStatus;
use Linktor\Types\PaginatedResponse;

class ChannelsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function list(array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/channels{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($c) => Channel::fromArray($c), $response->data);
        return $response;
    }

    public function get(string $id): Channel
    {
        $data = $this->client->get("/channels/{$id}");
        return Channel::fromArray($data);
    }

    public function create(array $input): Channel
    {
        $data = $this->client->post('/channels', $input);
        return Channel::fromArray($data);
    }

    public function update(string $id, array $input): Channel
    {
        $data = $this->client->patch("/channels/{$id}", $input);
        return Channel::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/channels/{$id}");
    }

    public function connect(string $id): Channel
    {
        $data = $this->client->post("/channels/{$id}/connect", []);
        return Channel::fromArray($data);
    }

    public function disconnect(string $id): Channel
    {
        $data = $this->client->post("/channels/{$id}/disconnect", []);
        return Channel::fromArray($data);
    }

    public function getStatus(string $id): ChannelStatus
    {
        $data = $this->client->get("/channels/{$id}/status");
        return ChannelStatus::fromArray($data);
    }

    public function test(string $id): Channel
    {
        $data = $this->client->post("/channels/{$id}/test", []);
        return Channel::fromArray($data);
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
