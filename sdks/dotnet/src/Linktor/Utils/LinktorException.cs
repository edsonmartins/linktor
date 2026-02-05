namespace Linktor.Utils;

public class LinktorException : Exception
{
    public int StatusCode { get; }
    public string? ErrorCode { get; }
    public string? RequestId { get; }

    public LinktorException(string message) : base(message)
    {
        StatusCode = 0;
    }

    public LinktorException(string message, Exception inner) : base(message, inner)
    {
        StatusCode = 0;
    }

    public LinktorException(string message, int statusCode, string? errorCode, string? requestId)
        : base(message)
    {
        StatusCode = statusCode;
        ErrorCode = errorCode;
        RequestId = requestId;
    }

    public static LinktorException FromStatus(int statusCode, string message, string? requestId)
    {
        return statusCode switch
        {
            400 => new ValidationException(message, requestId),
            401 => new AuthenticationException(message, requestId),
            403 => new AuthorizationException(message, requestId),
            404 => new NotFoundException(message, requestId),
            429 => new RateLimitException(message, 60, requestId),
            >= 500 and <= 599 => new ServerException(message, requestId),
            _ => new LinktorException(message, statusCode, null, requestId)
        };
    }
}

public class AuthenticationException : LinktorException
{
    public AuthenticationException(string message, string? requestId = null)
        : base(message, 401, "AUTHENTICATION_ERROR", requestId) { }
}

public class AuthorizationException : LinktorException
{
    public AuthorizationException(string message, string? requestId = null)
        : base(message, 403, "AUTHORIZATION_ERROR", requestId) { }
}

public class NotFoundException : LinktorException
{
    public NotFoundException(string message, string? requestId = null)
        : base(message, 404, "NOT_FOUND", requestId) { }
}

public class ValidationException : LinktorException
{
    public ValidationException(string message, string? requestId = null)
        : base(message, 400, "VALIDATION_ERROR", requestId) { }
}

public class RateLimitException : LinktorException
{
    public long RetryAfter { get; }

    public RateLimitException(string message, long retryAfter, string? requestId = null)
        : base(message, 429, "RATE_LIMIT", requestId)
    {
        RetryAfter = retryAfter;
    }
}

public class ServerException : LinktorException
{
    public ServerException(string message, string? requestId = null)
        : base(message, 500, "SERVER_ERROR", requestId) { }
}

public class WebhookVerificationException : LinktorException
{
    public WebhookVerificationException(string message)
        : base(message, 400, "WEBHOOK_VERIFICATION_FAILED", null) { }
}
