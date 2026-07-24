<?php

declare(strict_types=1);

namespace Linktor\Types;

/**
 * Channel model — mirrors the backend wire shape (snake_case).
 *
 * There is no `status` field on the wire: live state is `connectionStatus`
 * (connection_status) and the system enable flag is `enabled`.
 * Credentials are write-only and are never present on a response; `config`
 * is a flat map<string,string> whose secret values arrive redacted as
 * "__redacted__".
 */
class Channel
{
    public string $id = '';
    public string $tenantId = '';
    public string $type = '';
    public string $name = '';
    public ?string $identifier = null;
    public bool $enabled = false;
    /** One of: disconnected, connecting, connected, error */
    public string $connectionStatus = 'disconnected';
    /** @var array<string,string> */
    public array $config = [];
    public ?string $webhookUrl = null;
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;

    // WhatsApp coexistence
    public ?bool $isCoexistence = null;
    public ?string $wabaId = null;
    public ?\DateTimeImmutable $lastEchoAt = null;
    /** One of: inactive, pending, active, warning, disconnected */
    public ?string $coexistenceStatus = null;
    public ?string $messageTemplateNamespace = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->tenantId = $data['tenant_id'] ?? '';
        $instance->type = $data['type'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->identifier = $data['identifier'] ?? null;
        $instance->enabled = $data['enabled'] ?? false;
        $instance->connectionStatus = $data['connection_status'] ?? 'disconnected';
        $instance->config = $data['config'] ?? [];
        $instance->webhookUrl = $data['webhook_url'] ?? null;
        $instance->createdAt = isset($data['created_at']) ? new \DateTimeImmutable($data['created_at']) : null;
        $instance->updatedAt = isset($data['updated_at']) ? new \DateTimeImmutable($data['updated_at']) : null;
        $instance->isCoexistence = $data['is_coexistence'] ?? null;
        $instance->wabaId = $data['waba_id'] ?? null;
        $instance->lastEchoAt = isset($data['last_echo_at']) ? new \DateTimeImmutable($data['last_echo_at']) : null;
        $instance->coexistenceStatus = $data['coexistence_status'] ?? null;
        $instance->messageTemplateNamespace = $data['message_template_namespace'] ?? null;
        return $instance;
    }
}

/**
 * Result of connecting a channel (POST /channels/{id}/connect or /pair).
 *
 * For WhatsApp Web-style linking, `qrCode` carries the payload to render and
 * `expiresIn` its lifetime in seconds — call connect again to refresh an
 * expired code. `pairCode` is the phone-linking code. When `passkeyRequired`
 * is true the account is passkey-locked and must be linked by signing
 * `passkeyChallenge`, not by QR.
 */
class ConnectResult
{
    public ?Channel $channel = null;
    public ?string $qrCode = null;
    public ?int $expiresIn = null;
    public ?string $pairCode = null;
    public ?bool $passkeyRequired = null;
    /** @var mixed raw passkey challenge JSON */
    public mixed $passkeyChallenge = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->channel = isset($data['channel']) && is_array($data['channel'])
            ? Channel::fromArray($data['channel'])
            : null;
        $instance->qrCode = $data['qr_code'] ?? null;
        $instance->expiresIn = $data['expires_in'] ?? null;
        $instance->pairCode = $data['pair_code'] ?? null;
        $instance->passkeyRequired = $data['passkey_required'] ?? null;
        $instance->passkeyChallenge = $data['passkey_challenge'] ?? null;
        return $instance;
    }
}
