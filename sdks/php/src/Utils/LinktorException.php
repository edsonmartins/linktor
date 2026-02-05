<?php

declare(strict_types=1);

namespace Linktor\Utils;

class LinktorException extends \Exception
{
    protected int $statusCode;
    protected ?string $errorCode;
    protected ?string $requestId;

    public function __construct(
        string $message,
        int $statusCode = 0,
        ?string $errorCode = null,
        ?string $requestId = null,
        ?\Throwable $previous = null
    ) {
        parent::__construct($message, $statusCode, $previous);
        $this->statusCode = $statusCode;
        $this->errorCode = $errorCode;
        $this->requestId = $requestId;
    }

    public function getStatusCode(): int
    {
        return $this->statusCode;
    }

    public function getErrorCode(): ?string
    {
        return $this->errorCode;
    }

    public function getRequestId(): ?string
    {
        return $this->requestId;
    }

    public static function fromStatus(int $statusCode, string $message, ?string $requestId = null): self
    {
        return match ($statusCode) {
            400 => new ValidationException($message, $requestId),
            401 => new AuthenticationException($message, $requestId),
            403 => new AuthorizationException($message, $requestId),
            404 => new NotFoundException($message, $requestId),
            429 => new RateLimitException($message, 60, $requestId),
            default => $statusCode >= 500 && $statusCode <= 599
                ? new ServerException($message, $requestId)
                : new self($message, $statusCode, null, $requestId),
        };
    }
}

class AuthenticationException extends LinktorException
{
    public function __construct(string $message, ?string $requestId = null)
    {
        parent::__construct($message, 401, 'AUTHENTICATION_ERROR', $requestId);
    }
}

class AuthorizationException extends LinktorException
{
    public function __construct(string $message, ?string $requestId = null)
    {
        parent::__construct($message, 403, 'AUTHORIZATION_ERROR', $requestId);
    }
}

class NotFoundException extends LinktorException
{
    public function __construct(string $message, ?string $requestId = null)
    {
        parent::__construct($message, 404, 'NOT_FOUND', $requestId);
    }
}

class ValidationException extends LinktorException
{
    public function __construct(string $message, ?string $requestId = null)
    {
        parent::__construct($message, 400, 'VALIDATION_ERROR', $requestId);
    }
}

class RateLimitException extends LinktorException
{
    private int $retryAfter;

    public function __construct(string $message, int $retryAfter = 60, ?string $requestId = null)
    {
        parent::__construct($message, 429, 'RATE_LIMIT', $requestId);
        $this->retryAfter = $retryAfter;
    }

    public function getRetryAfter(): int
    {
        return $this->retryAfter;
    }
}

class ServerException extends LinktorException
{
    public function __construct(string $message, ?string $requestId = null)
    {
        parent::__construct($message, 500, 'SERVER_ERROR', $requestId);
    }
}

class WebhookVerificationException extends LinktorException
{
    public function __construct(string $message)
    {
        parent::__construct($message, 400, 'WEBHOOK_VERIFICATION_FAILED', null);
    }
}
