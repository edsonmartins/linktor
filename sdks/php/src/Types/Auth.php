<?php

declare(strict_types=1);

namespace Linktor\Types;

class User
{
    public string $id = '';
    public string $email = '';
    public ?string $name = null;
    public ?string $avatarUrl = null;
    public string $role = '';
    public string $tenantId = '';
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->email = $data['email'] ?? '';
        $instance->name = $data['name'] ?? null;
        $instance->avatarUrl = $data['avatarUrl'] ?? null;
        $instance->role = $data['role'] ?? '';
        $instance->tenantId = $data['tenantId'] ?? '';
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        return $instance;
    }
}

class Tenant
{
    public string $id = '';
    public string $name = '';
    public ?string $slug = null;
    public ?string $logoUrl = null;
    public array $settings = [];
    public ?\DateTimeImmutable $createdAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->slug = $data['slug'] ?? null;
        $instance->logoUrl = $data['logoUrl'] ?? null;
        $instance->settings = $data['settings'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        return $instance;
    }
}

class LoginResponse
{
    public string $accessToken = '';
    public string $refreshToken = '';
    public int $expiresIn = 0;
    public string $tokenType = 'Bearer';
    public ?User $user = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->accessToken = $data['accessToken'] ?? '';
        $instance->refreshToken = $data['refreshToken'] ?? '';
        $instance->expiresIn = $data['expiresIn'] ?? 0;
        $instance->tokenType = $data['tokenType'] ?? 'Bearer';
        $instance->user = isset($data['user']) ? User::fromArray($data['user']) : null;
        return $instance;
    }
}

class RefreshTokenResponse
{
    public string $accessToken = '';
    public int $expiresIn = 0;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->accessToken = $data['accessToken'] ?? '';
        $instance->expiresIn = $data['expiresIn'] ?? 0;
        return $instance;
    }
}
