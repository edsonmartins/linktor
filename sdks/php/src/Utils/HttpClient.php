<?php

declare(strict_types=1);

namespace Linktor\Utils;

use GuzzleHttp\Client;
use GuzzleHttp\Exception\ClientException;
use GuzzleHttp\Exception\ServerException as GuzzleServerException;
use GuzzleHttp\Exception\ConnectException;
use GuzzleHttp\RequestOptions;

class HttpClient
{
    private Client $client;
    private string $baseUrl;
    private ?string $apiKey;
    private ?string $accessToken;
    private int $maxRetries;

    public function __construct(
        string $baseUrl,
        ?string $apiKey = null,
        ?string $accessToken = null,
        int $timeoutSeconds = 30,
        int $maxRetries = 3
    ) {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->apiKey = $apiKey;
        $this->accessToken = $accessToken;
        $this->maxRetries = $maxRetries;

        $this->client = new Client([
            'base_uri' => $this->baseUrl,
            'timeout' => $timeoutSeconds,
            'http_errors' => false,
        ]);
    }

    public function setAccessToken(?string $token): void
    {
        $this->accessToken = $token;
    }

    public function get(string $path): array
    {
        return $this->request('GET', $path);
    }

    public function post(string $path, ?array $body = null): array
    {
        return $this->request('POST', $path, $body);
    }

    public function patch(string $path, array $body): array
    {
        return $this->request('PATCH', $path, $body);
    }

    public function delete(string $path): void
    {
        $this->request('DELETE', $path);
    }

    private function request(string $method, string $path, ?array $body = null): array
    {
        $attempts = 0;

        while (true) {
            $attempts++;

            $options = [
                RequestOptions::HEADERS => $this->getHeaders(),
            ];

            if ($body !== null) {
                $options[RequestOptions::JSON] = $body;
            }

            try {
                $response = $this->client->request($method, $path, $options);
            } catch (ConnectException $e) {
                if ($attempts < $this->maxRetries) {
                    usleep((int) (pow(2, $attempts) * 1000000)); // Exponential backoff
                    continue;
                }
                throw new LinktorException("Connection failed: {$e->getMessage()}");
            }

            $statusCode = $response->getStatusCode();
            $requestId = $response->getHeaderLine('X-Request-ID') ?: null;
            $content = $response->getBody()->getContents();

            if ($statusCode >= 200 && $statusCode < 300) {
                if (empty($content) || $content === 'null') {
                    return [];
                }

                $data = json_decode($content, true);

                // Try to unwrap ApiResponse
                if (isset($data['success']) && $data['success'] === true && isset($data['data'])) {
                    return $data['data'];
                }

                return $data ?? [];
            }

            // Handle rate limiting
            if ($statusCode === 429 && $attempts < $this->maxRetries) {
                $retryAfter = (int) ($response->getHeaderLine('Retry-After') ?: 60);
                sleep($retryAfter);
                continue;
            }

            // Handle server errors with retry
            if ($statusCode >= 500 && $attempts < $this->maxRetries) {
                usleep((int) (pow(2, $attempts) * 1000000));
                continue;
            }

            // Parse error message
            $message = 'Request failed';
            if (!empty($content)) {
                $errorData = json_decode($content, true);
                if (isset($errorData['message'])) {
                    $message = $errorData['message'];
                }
            }

            throw LinktorException::fromStatus($statusCode, $message, $requestId);
        }
    }

    private function getHeaders(): array
    {
        $headers = [
            'Content-Type' => 'application/json',
            'Accept' => 'application/json',
        ];

        if (!empty($this->apiKey)) {
            $headers['X-API-Key'] = $this->apiKey;
        } elseif (!empty($this->accessToken)) {
            $headers['Authorization'] = "Bearer {$this->accessToken}";
        }

        return $headers;
    }
}
