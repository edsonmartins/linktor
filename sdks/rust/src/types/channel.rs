use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Channel type as it appears on the wire (`type` field). snake_case.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ChannelType {
    Webchat,
    Whatsapp,
    WhatsappOfficial,
    WhatsappUnofficial,
    Telegram,
    Sms,
    Rcs,
    Instagram,
    Facebook,
    Email,
    Voice,
    Teams,
    Slack,
    Mattermost,
    Direto,
}

/// Live connection state of a channel (wire field `connection_status`).
/// Distinct from `enabled`, which is the system-level enable flag.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ConnectionStatus {
    Disconnected,
    Connecting,
    Connected,
    Error,
}

/// WhatsApp Business App + Cloud API coexistence state (wire field
/// `coexistence_status`).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum CoexistenceStatus {
    Inactive,
    Pending,
    Active,
    Warning,
    Disconnected,
}

/// Channel mirrors the backend wire shape exactly (snake_case). Credentials are
/// write-only and never present on a response; secret `config` values are
/// redacted server-side as `"__redacted__"`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Channel {
    pub id: String,
    pub tenant_id: String,
    #[serde(rename = "type")]
    pub channel_type: ChannelType,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub identifier: Option<String>,
    /// System-level enable flag.
    pub enabled: bool,
    /// Live connection state.
    pub connection_status: ConnectionStatus,
    /// Non-secret settings; secret values are redacted as `"__redacted__"`.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub config: Option<HashMap<String, String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub webhook_url: Option<String>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub is_coexistence: Option<bool>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub waba_id: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub last_echo_at: Option<chrono::DateTime<chrono::Utc>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub coexistence_status: Option<CoexistenceStatus>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub message_template_namespace: Option<String>,
}

/// Result of connecting a channel. For WhatsApp Web-style linking it carries the
/// QR payload (`qr_code`) to render and its lifetime (`expires_in` seconds); call
/// `connect` again to refresh an expired code. `pair_code` is the phone-linking
/// code when pairing by number. `passkey_required` is true when the account is
/// passkey-locked and must be linked by signing `passkey_challenge`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectResult {
    /// The channel, when the response carries one. The backend serializes
    /// `"channel"` with no omitempty, so a nil value arrives as `null` (e.g. some
    /// pair/passkey responses) — model it as optional so that never fails to
    /// deserialize.
    #[serde(default)]
    pub channel: Option<Channel>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub qr_code: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub expires_in: Option<i64>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub pair_code: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passkey_required: Option<bool>,
    /// Raw JSON challenge to sign and submit via
    /// `POST /channels/{id}/passkey/response`.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passkey_challenge: Option<serde_json::Value>,
}

/// Body for creating a channel. Put secrets in `credentials` (write-only) and
/// non-secret settings in `config`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateChannelInput {
    #[serde(rename = "type")]
    pub channel_type: ChannelType,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub identifier: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub config: Option<HashMap<String, String>>,
    /// Secrets (e.g. access_token, bot_token). Write-only, never returned.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub credentials: Option<HashMap<String, String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub webhook_url: Option<String>,
}

impl CreateChannelInput {
    pub fn new(name: impl Into<String>, channel_type: ChannelType) -> Self {
        Self {
            channel_type,
            name: name.into(),
            identifier: None,
            config: None,
            credentials: None,
            webhook_url: None,
        }
    }

    pub fn identifier(mut self, identifier: impl Into<String>) -> Self {
        self.identifier = Some(identifier.into());
        self
    }

    pub fn config(mut self, config: HashMap<String, String>) -> Self {
        self.config = Some(config);
        self
    }

    pub fn credentials(mut self, credentials: HashMap<String, String>) -> Self {
        self.credentials = Some(credentials);
        self
    }

    pub fn webhook_url(mut self, webhook_url: impl Into<String>) -> Self {
        self.webhook_url = Some(webhook_url.into());
        self
    }
}

/// Body for updating a channel (PUT). Reuses the create body shape server-side.
/// Omit `credentials` (or send the redacted placeholder) to keep the stored
/// secrets untouched.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct UpdateChannelInput {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub identifier: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub config: Option<HashMap<String, String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub credentials: Option<HashMap<String, String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub webhook_url: Option<String>,
}

/// Query filters for `GET /channels`.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ListChannelsParams {
    #[serde(rename = "type", skip_serializing_if = "Option::is_none")]
    pub channel_type: Option<ChannelType>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<ConnectionStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub search: Option<String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn connect_result_deserializes_qr_and_expiry() {
        let json = r#"{
            "channel": {
                "id": "ch_1",
                "tenant_id": "t_1",
                "type": "whatsapp_unofficial",
                "name": "My WhatsApp",
                "enabled": true,
                "connection_status": "connecting",
                "created_at": "2026-07-24T12:00:00Z",
                "updated_at": "2026-07-24T12:00:00Z"
            },
            "qr_code": "Q",
            "expires_in": 60
        }"#;

        let result: ConnectResult = serde_json::from_str(json).expect("should deserialize");
        assert_eq!(result.qr_code.as_deref(), Some("Q"));
        assert_eq!(result.expires_in, Some(60));
        assert_eq!(result.pair_code, None);
        assert_eq!(result.passkey_required, None);
        assert_eq!(result.passkey_challenge, None);
        let channel = result.channel.expect("channel present");
        assert_eq!(channel.id, "ch_1");
        assert_eq!(channel.channel_type, ChannelType::WhatsappUnofficial);
        assert_eq!(channel.connection_status, ConnectionStatus::Connecting);
        assert!(channel.enabled);
    }

    #[test]
    fn connect_result_tolerates_null_channel() {
        // The backend emits `"channel"` with no omitempty, so a nil pointer
        // arrives as null (e.g. some pair responses). It must not fail to parse.
        let json = r#"{ "channel": null, "pair_code": "ABCD-1234" }"#;
        let result: ConnectResult = serde_json::from_str(json).expect("should deserialize");
        assert!(result.channel.is_none());
        assert_eq!(result.pair_code.as_deref(), Some("ABCD-1234"));
    }

    #[test]
    fn channel_deserializes_coexistence_fields() {
        let json = r#"{
            "id": "ch_2",
            "tenant_id": "t_1",
            "type": "whatsapp_official",
            "name": "Official",
            "enabled": true,
            "connection_status": "connected",
            "config": {"phone_number_id": "123", "access_token": "__redacted__"},
            "created_at": "2026-07-24T12:00:00Z",
            "updated_at": "2026-07-24T12:00:00Z",
            "is_coexistence": true,
            "waba_id": "waba_1",
            "coexistence_status": "active"
        }"#;

        let ch: Channel = serde_json::from_str(json).expect("should deserialize");
        assert_eq!(ch.is_coexistence, Some(true));
        assert_eq!(ch.waba_id.as_deref(), Some("waba_1"));
        assert_eq!(ch.coexistence_status, Some(CoexistenceStatus::Active));
        assert_eq!(
            ch.config.as_ref().unwrap().get("access_token").map(String::as_str),
            Some("__redacted__")
        );
    }
}
