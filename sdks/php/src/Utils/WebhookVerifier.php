<?php

declare(strict_types=1);

namespace Linktor\Utils;

use Linktor\Types\WebhookConstants;
use Linktor\Types\WebhookEvent;

class WebhookVerifier
{
    /**
     * Compute HMAC-SHA256 signature for a payload
     */
    public static function computeSignature(string $payload, string $secret): string
    {
        return hash_hmac('sha256', $payload, $secret);
    }

    /**
     * Verify webhook signature using timing-safe comparison
     */
    public static function verifySignature(string $payload, string $signature, string $secret): bool
    {
        if (empty($signature) || empty($secret)) {
            return false;
        }

        $expected = self::computeSignature($payload, $secret);
        return hash_equals(strtolower($expected), strtolower($signature));
    }

    /**
     * Verify webhook with headers and optional timestamp tolerance
     */
    public static function verify(
        string $payload,
        array $headers,
        string $secret,
        ?int $toleranceSeconds = null
    ): bool {
        $tolerance = $toleranceSeconds ?? WebhookConstants::DEFAULT_TOLERANCE_SECONDS;

        // Get signature (case-insensitive header lookup)
        $signature = self::getHeader($headers, WebhookConstants::SIGNATURE_HEADER);
        if (empty($signature)) {
            return false;
        }

        // Verify timestamp if present
        $timestampStr = self::getHeader($headers, WebhookConstants::TIMESTAMP_HEADER);
        if (!empty($timestampStr)) {
            $timestamp = (int) $timestampStr;
            $now = time();
            if (abs($now - $timestamp) > $tolerance) {
                return false;
            }
        }

        return self::verifySignature($payload, $signature, $secret);
    }

    /**
     * Construct and verify a webhook event from payload and headers
     *
     * @throws WebhookVerificationException
     */
    public static function constructEvent(
        string $payload,
        array $headers,
        string $secret,
        ?int $toleranceSeconds = null
    ): WebhookEvent {
        $tolerance = $toleranceSeconds === 0
            ? WebhookConstants::DEFAULT_TOLERANCE_SECONDS
            : ($toleranceSeconds ?? WebhookConstants::DEFAULT_TOLERANCE_SECONDS);

        if (!self::verify($payload, $headers, $secret, $tolerance)) {
            throw new WebhookVerificationException('Webhook signature verification failed');
        }

        try {
            $data = json_decode($payload, true, 512, JSON_THROW_ON_ERROR);
        } catch (\JsonException $e) {
            throw new WebhookVerificationException("Failed to parse webhook event: {$e->getMessage()}");
        }

        $event = WebhookEvent::fromArray($data);

        if (empty($event->id) || empty($event->type)) {
            throw new WebhookVerificationException('Invalid webhook event structure');
        }

        return $event;
    }

    /**
     * Get header value with case-insensitive lookup
     */
    private static function getHeader(array $headers, string $name): ?string
    {
        // Try exact match first
        if (isset($headers[$name])) {
            return $headers[$name];
        }

        // Try lowercase
        $lowerName = strtolower($name);
        if (isset($headers[$lowerName])) {
            return $headers[$lowerName];
        }

        // Search case-insensitively
        foreach ($headers as $key => $value) {
            if (strtolower($key) === $lowerName) {
                return $value;
            }
        }

        return null;
    }
}
