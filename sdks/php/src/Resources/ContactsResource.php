<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\Contact;
use Linktor\Types\PaginatedResponse;

class ContactsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function list(array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/contacts{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($c) => Contact::fromArray($c), $response->data);
        return $response;
    }

    public function get(string $id): Contact
    {
        $data = $this->client->get("/contacts/{$id}");
        return Contact::fromArray($data);
    }

    public function create(array $input): Contact
    {
        $data = $this->client->post('/contacts', $input);
        return Contact::fromArray($data);
    }

    public function update(string $id, array $input): Contact
    {
        $data = $this->client->patch("/contacts/{$id}", $input);
        return Contact::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/contacts/{$id}");
    }

    public function search(array $input): PaginatedResponse
    {
        $data = $this->client->post('/contacts/search', $input);
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($c) => Contact::fromArray($c), $response->data);
        return $response;
    }

    public function merge(string $primaryId, array $secondaryIds): Contact
    {
        $data = $this->client->post('/contacts/merge', [
            'primaryId' => $primaryId,
            'secondaryIds' => $secondaryIds,
        ]);
        return Contact::fromArray($data);
    }

    public function getByExternalId(string $externalId): Contact
    {
        $data = $this->client->get('/contacts/external/' . urlencode($externalId));
        return Contact::fromArray($data);
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
