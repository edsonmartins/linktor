package io.linktor.utils;

public class LinktorException extends RuntimeException {
    private final int statusCode;
    private final String errorCode;
    private final String requestId;

    public LinktorException(String message) {
        super(message);
        this.statusCode = 0;
        this.errorCode = null;
        this.requestId = null;
    }

    public LinktorException(String message, Throwable cause) {
        super(message, cause);
        this.statusCode = 0;
        this.errorCode = null;
        this.requestId = null;
    }

    public LinktorException(String message, int statusCode, String errorCode, String requestId) {
        super(message);
        this.statusCode = statusCode;
        this.errorCode = errorCode;
        this.requestId = requestId;
    }

    public int getStatusCode() { return statusCode; }
    public String getErrorCode() { return errorCode; }
    public String getRequestId() { return requestId; }

    public static class AuthenticationException extends LinktorException {
        public AuthenticationException(String message) {
            super(message, 401, "AUTHENTICATION_ERROR", null);
        }

        public AuthenticationException(String message, String requestId) {
            super(message, 401, "AUTHENTICATION_ERROR", requestId);
        }
    }

    public static class AuthorizationException extends LinktorException {
        public AuthorizationException(String message) {
            super(message, 403, "AUTHORIZATION_ERROR", null);
        }

        public AuthorizationException(String message, String requestId) {
            super(message, 403, "AUTHORIZATION_ERROR", requestId);
        }
    }

    public static class NotFoundException extends LinktorException {
        public NotFoundException(String message) {
            super(message, 404, "NOT_FOUND", null);
        }

        public NotFoundException(String message, String requestId) {
            super(message, 404, "NOT_FOUND", requestId);
        }
    }

    public static class ValidationException extends LinktorException {
        public ValidationException(String message) {
            super(message, 400, "VALIDATION_ERROR", null);
        }

        public ValidationException(String message, String requestId) {
            super(message, 400, "VALIDATION_ERROR", requestId);
        }
    }

    public static class RateLimitException extends LinktorException {
        private final long retryAfter;

        public RateLimitException(String message, long retryAfter) {
            super(message, 429, "RATE_LIMIT", null);
            this.retryAfter = retryAfter;
        }

        public RateLimitException(String message, long retryAfter, String requestId) {
            super(message, 429, "RATE_LIMIT", requestId);
            this.retryAfter = retryAfter;
        }

        public long getRetryAfter() { return retryAfter; }
    }

    public static class ServerException extends LinktorException {
        public ServerException(String message) {
            super(message, 500, "SERVER_ERROR", null);
        }

        public ServerException(String message, String requestId) {
            super(message, 500, "SERVER_ERROR", requestId);
        }
    }

    public static class NetworkException extends LinktorException {
        public NetworkException(String message) {
            super(message);
        }

        public NetworkException(String message, Throwable cause) {
            super(message, cause);
        }
    }

    public static class WebhookVerificationException extends LinktorException {
        public WebhookVerificationException(String message) {
            super(message, 400, "WEBHOOK_VERIFICATION_FAILED", null);
        }
    }
}
