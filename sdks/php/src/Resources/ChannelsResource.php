<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\Channel;
use Linktor\Types\ConnectResult;

/**
 * Channels resource.
 *
 * Every backend response is wrapped as `{ success, data }`; the underlying
 * HttpClient already unwraps that envelope, so each method here just shapes
 * the inner `data` into the corresponding type.
 */
class ChannelsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    /**
     * List channels. The backend returns a plain array under `data`
     * (channels have no pagination envelope).
     *
     * @param array<string,mixed> $params optional filters (type, status, search)
     * @return Channel[]
     */
    public function list(array $params = []): array
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/channels{$query}");
        return array_map(fn($c) => Channel::fromArray($c), $data);
    }

    public function get(string $id): Channel
    {
        $data = $this->client->get("/channels/{$id}");
        return Channel::fromArray($data);
    }

    /**
     * Create a channel. Put secrets in `credentials` (write-only) and
     * non-secret settings in `config`.
     *
     * @param array<string,mixed> $input {type, name, identifier?, config?, credentials?, webhook_url?}
     */
    public function create(array $input): Channel
    {
        $data = $this->client->post('/channels', $input);
        return Channel::fromArray($data);
    }

    /**
     * Update a channel (PUT). Omit `credentials` (or send "__redacted__")
     * to keep the stored secrets.
     *
     * @param array<string,mixed> $input {name?, identifier?, config?, credentials?, webhook_url?}
     */
    public function update(string $id, array $input): Channel
    {
        $data = $this->client->put("/channels/{$id}", $input);
        return Channel::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/channels/{$id}");
    }

    /**
     * Connect a channel. For WhatsApp this starts (or refreshes) linking and
     * returns a ConnectResult carrying the QR payload (`qrCode`, `expiresIn`)
     * to render; call connect again to poll for a fresh QR or linked state.
     */
    public function connect(string $id): ConnectResult
    {
        $data = $this->client->post("/channels/{$id}/connect", []);
        return ConnectResult::fromArray($data);
    }

    /**
     * Request a WhatsApp pairing code for a phone number, as an alternative
     * to QR linking.
     */
    public function requestPairCode(string $id, string $phoneNumber): ConnectResult
    {
        $data = $this->client->post("/channels/{$id}/pair", [
            'phone_number' => $phoneNumber,
        ]);
        return ConnectResult::fromArray($data);
    }

    /**
     * Disconnect a channel (deactivate it).
     */
    public function disconnect(string $id): Channel
    {
        $data = $this->client->post("/channels/{$id}/disconnect", []);
        return Channel::fromArray($data);
    }

    /**
     * @param array<string,mixed> $params
     */
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
