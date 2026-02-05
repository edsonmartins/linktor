use serde::{Deserialize, Serialize};
use std::collections::HashMap;

pub const SIGNATURE_HEADER: &str = "X-Linktor-Signature";
pub const TIMESTAMP_HEADER: &str = "X-Linktor-Timestamp";
pub const DEFAULT_TOLERANCE_SECONDS: i64 = 300;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum EventType {
    #[serde(rename = "message.received")]
    MessageReceived,
    #[serde(rename = "message.sent")]
    MessageSent,
    #[serde(rename = "message.delivered")]
    MessageDelivered,
    #[serde(rename = "message.read")]
    MessageRead,
    #[serde(rename = "message.failed")]
    MessageFailed,

    #[serde(rename = "conversation.created")]
    ConversationCreated,
    #[serde(rename = "conversation.updated")]
    ConversationUpdated,
    #[serde(rename = "conversation.resolved")]
    ConversationResolved,
    #[serde(rename = "conversation.assigned")]
    ConversationAssigned,

    #[serde(rename = "contact.created")]
    ContactCreated,
    #[serde(rename = "contact.updated")]
    ContactUpdated,
    #[serde(rename = "contact.deleted")]
    ContactDeleted,

    #[serde(rename = "channel.connected")]
    ChannelConnected,
    #[serde(rename = "channel.disconnected")]
    ChannelDisconnected,
    #[serde(rename = "channel.error")]
    ChannelError,

    #[serde(rename = "bot.started")]
    BotStarted,
    #[serde(rename = "bot.stopped")]
    BotStopped,

    #[serde(rename = "flow.started")]
    FlowStarted,
    #[serde(rename = "flow.completed")]
    FlowCompleted,
    #[serde(rename = "flow.failed")]
    FlowFailed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebhookEvent {
    pub id: String,
    #[serde(rename = "type")]
    pub event_type: String,
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub tenant_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<HashMap<String, serde_json::Value>>,
}

impl WebhookEvent {
    pub fn get_event_type(&self) -> Option<EventType> {
        match self.event_type.as_str() {
            "message.received" => Some(EventType::MessageReceived),
            "message.sent" => Some(EventType::MessageSent),
            "message.delivered" => Some(EventType::MessageDelivered),
            "message.read" => Some(EventType::MessageRead),
            "message.failed" => Some(EventType::MessageFailed),
            "conversation.created" => Some(EventType::ConversationCreated),
            "conversation.updated" => Some(EventType::ConversationUpdated),
            "conversation.resolved" => Some(EventType::ConversationResolved),
            "conversation.assigned" => Some(EventType::ConversationAssigned),
            "contact.created" => Some(EventType::ContactCreated),
            "contact.updated" => Some(EventType::ContactUpdated),
            "contact.deleted" => Some(EventType::ContactDeleted),
            "channel.connected" => Some(EventType::ChannelConnected),
            "channel.disconnected" => Some(EventType::ChannelDisconnected),
            "channel.error" => Some(EventType::ChannelError),
            "bot.started" => Some(EventType::BotStarted),
            "bot.stopped" => Some(EventType::BotStopped),
            "flow.started" => Some(EventType::FlowStarted),
            "flow.completed" => Some(EventType::FlowCompleted),
            "flow.failed" => Some(EventType::FlowFailed),
            _ => None,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebhookConfig {
    pub url: String,
    pub secret: String,
    #[serde(default)]
    pub events: Vec<String>,
    #[serde(default)]
    pub enabled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub headers: Option<HashMap<String, String>>,
}
