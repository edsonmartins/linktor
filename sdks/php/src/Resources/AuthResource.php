<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\LoginResponse;
use Linktor\Types\RefreshTokenResponse;
use Linktor\Types\User;
use Linktor\Types\Tenant;

class AuthResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function login(string $email, string $password): LoginResponse
    {
        $data = $this->client->post('/auth/login', [
            'email' => $email,
            'password' => $password,
        ]);

        $response = LoginResponse::fromArray($data);
        $this->client->setAccessToken($response->accessToken);

        return $response;
    }

    public function logout(): void
    {
        $this->client->post('/auth/logout', []);
        $this->client->setAccessToken(null);
    }

    public function refreshToken(string $refreshToken): RefreshTokenResponse
    {
        $data = $this->client->post('/auth/refresh', [
            'refreshToken' => $refreshToken,
        ]);

        $response = RefreshTokenResponse::fromArray($data);
        $this->client->setAccessToken($response->accessToken);

        return $response;
    }

    public function getCurrentUser(): User
    {
        $data = $this->client->get('/auth/me');
        return User::fromArray($data);
    }

    public function getCurrentTenant(): Tenant
    {
        $data = $this->client->get('/auth/tenant');
        return Tenant::fromArray($data);
    }
}
