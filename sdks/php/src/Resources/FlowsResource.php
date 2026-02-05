<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\Flow;
use Linktor\Types\FlowExecution;
use Linktor\Types\PaginatedResponse;

class FlowsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function list(array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/flows{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($f) => Flow::fromArray($f), $response->data);
        return $response;
    }

    public function get(string $id): Flow
    {
        $data = $this->client->get("/flows/{$id}");
        return Flow::fromArray($data);
    }

    public function create(array $input): Flow
    {
        $data = $this->client->post('/flows', $input);
        return Flow::fromArray($data);
    }

    public function update(string $id, array $input): Flow
    {
        $data = $this->client->patch("/flows/{$id}", $input);
        return Flow::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/flows/{$id}");
    }

    public function publish(string $id): Flow
    {
        $data = $this->client->post("/flows/{$id}/publish", []);
        return Flow::fromArray($data);
    }

    public function unpublish(string $id): Flow
    {
        $data = $this->client->post("/flows/{$id}/unpublish", []);
        return Flow::fromArray($data);
    }

    public function execute(string $id, array $input): FlowExecution
    {
        $data = $this->client->post("/flows/{$id}/execute", $input);
        return FlowExecution::fromArray($data);
    }

    public function validate(string $id): array
    {
        return $this->client->post("/flows/{$id}/validate", []);
    }

    public function duplicate(string $id, ?string $name = null): Flow
    {
        $input = [];
        if ($name !== null) {
            $input['name'] = $name;
        }
        $data = $this->client->post("/flows/{$id}/duplicate", $input);
        return Flow::fromArray($data);
    }

    public function getNodeTypes(): array
    {
        return $this->client->get('/flows/node-types');
    }

    public function getExecutions(string $flowId, array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/flows/{$flowId}/executions{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($e) => FlowExecution::fromArray($e), $response->data);
        return $response;
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
