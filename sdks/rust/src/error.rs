use thiserror::Error;

#[derive(Error, Debug)]
pub enum LinktorError {
    #[error("Authentication failed: {message}")]
    Authentication { message: String, request_id: Option<String> },

    #[error("Authorization failed: {message}")]
    Authorization { message: String, request_id: Option<String> },

    #[error("Resource not found: {message}")]
    NotFound { message: String, request_id: Option<String> },

    #[error("Validation error: {message}")]
    Validation { message: String, request_id: Option<String> },

    #[error("Rate limit exceeded. Retry after {retry_after} seconds")]
    RateLimit { retry_after: u64, message: String, request_id: Option<String> },

    #[error("Server error: {message}")]
    Server { message: String, request_id: Option<String> },

    #[error("Network error: {0}")]
    Network(#[from] reqwest::Error),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("Webhook verification failed: {message}")]
    WebhookVerification { message: String },

    #[error("WebSocket error: {message}")]
    WebSocket { message: String },

    #[error("Unknown error: {message}")]
    Unknown { message: String, status_code: Option<u16> },
}

impl LinktorError {
    pub fn from_status(status: reqwest::StatusCode, message: String, request_id: Option<String>) -> Self {
        match status.as_u16() {
            400 => LinktorError::Validation { message, request_id },
            401 => LinktorError::Authentication { message, request_id },
            403 => LinktorError::Authorization { message, request_id },
            404 => LinktorError::NotFound { message, request_id },
            429 => LinktorError::RateLimit {
                retry_after: 60,
                message,
                request_id,
            },
            500..=599 => LinktorError::Server { message, request_id },
            _ => LinktorError::Unknown {
                message,
                status_code: Some(status.as_u16()),
            },
        }
    }

    pub fn request_id(&self) -> Option<&str> {
        match self {
            LinktorError::Authentication { request_id, .. } => request_id.as_deref(),
            LinktorError::Authorization { request_id, .. } => request_id.as_deref(),
            LinktorError::NotFound { request_id, .. } => request_id.as_deref(),
            LinktorError::Validation { request_id, .. } => request_id.as_deref(),
            LinktorError::RateLimit { request_id, .. } => request_id.as_deref(),
            LinktorError::Server { request_id, .. } => request_id.as_deref(),
            _ => None,
        }
    }
}

pub type Result<T> = std::result::Result<T, LinktorError>;
