use crate::error::{LinktorError, Result};
use crate::types::webhook::{WebhookEvent, SIGNATURE_HEADER, TIMESTAMP_HEADER, DEFAULT_TOLERANCE_SECONDS};
use chrono::Utc;
use hmac::{Hmac, Mac};
use sha2::Sha256;
use std::collections::HashMap;

type HmacSha256 = Hmac<Sha256>;

/// Compute HMAC-SHA256 signature for the given payload
pub fn compute_signature(payload: &[u8], secret: &str) -> String {
    let mut mac = HmacSha256::new_from_slice(secret.as_bytes())
        .expect("HMAC can take key of any size");
    mac.update(payload);
    let result = mac.finalize();
    hex::encode(result.into_bytes())
}

/// Verify webhook signature only (no timestamp validation)
pub fn verify_signature(payload: &[u8], signature: &str, secret: &str) -> bool {
    if signature.is_empty() || secret.is_empty() {
        return false;
    }

    let expected = compute_signature(payload, secret);

    // Constant-time comparison
    if expected.len() != signature.len() {
        return false;
    }

    let mut result = 0u8;
    for (a, b) in expected.bytes().zip(signature.bytes()) {
        result |= a ^ b;
    }
    result == 0
}

/// Verify webhook with signature and timestamp validation
pub fn verify(payload: &[u8], headers: &HashMap<String, String>, secret: &str, tolerance_seconds: Option<i64>) -> bool {
    let tolerance = tolerance_seconds.unwrap_or(DEFAULT_TOLERANCE_SECONDS);

    // Get signature from headers (case-insensitive)
    let signature = headers
        .get(SIGNATURE_HEADER)
        .or_else(|| headers.get(&SIGNATURE_HEADER.to_lowercase()))
        .map(String::as_str)
        .unwrap_or("");

    if signature.is_empty() {
        return false;
    }

    // Verify timestamp if present
    let timestamp_str = headers
        .get(TIMESTAMP_HEADER)
        .or_else(|| headers.get(&TIMESTAMP_HEADER.to_lowercase()));

    if let Some(ts_str) = timestamp_str {
        if let Ok(timestamp) = ts_str.parse::<i64>() {
            let now = Utc::now().timestamp();
            if (now - timestamp).abs() > tolerance {
                return false;
            }
        } else {
            return false;
        }
    }

    verify_signature(payload, signature, secret)
}

/// Construct and verify a webhook event
pub fn construct_event(
    payload: &[u8],
    headers: &HashMap<String, String>,
    secret: &str,
    tolerance_seconds: Option<i64>,
) -> Result<WebhookEvent> {
    let tolerance = if tolerance_seconds == Some(0) {
        DEFAULT_TOLERANCE_SECONDS
    } else {
        tolerance_seconds.unwrap_or(DEFAULT_TOLERANCE_SECONDS)
    };

    if !verify(payload, headers, secret, Some(tolerance)) {
        return Err(LinktorError::WebhookVerification {
            message: "Webhook signature verification failed".to_string(),
        });
    }

    let event: WebhookEvent = serde_json::from_slice(payload).map_err(|e| {
        LinktorError::WebhookVerification {
            message: format!("Failed to parse webhook event: {}", e),
        }
    })?;

    if event.id.is_empty() || event.event_type.is_empty() {
        return Err(LinktorError::WebhookVerification {
            message: "Invalid webhook event structure".to_string(),
        });
    }

    Ok(event)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_compute_signature() {
        let payload = b"test payload";
        let secret = "test-secret";
        let signature = compute_signature(payload, secret);
        assert!(!signature.is_empty());
        assert_eq!(signature.len(), 64); // SHA256 produces 32 bytes = 64 hex chars
    }

    #[test]
    fn test_verify_signature() {
        let payload = b"test payload";
        let secret = "test-secret";
        let signature = compute_signature(payload, secret);
        assert!(verify_signature(payload, &signature, secret));
        assert!(!verify_signature(payload, "wrong-signature", secret));
    }
}
