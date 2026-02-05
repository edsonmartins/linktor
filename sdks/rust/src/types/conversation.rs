use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ConversationStatus {
    Open,
    Pending,
    Resolved,
    Closed,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ConversationPriority {
    Low,
    Medium,
    High,
    Urgent,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MessageType {
    Text,
    Image,
    Video,
    Audio,
    Document,
    Location,
    Contact,
    Sticker,
    Template,
    Interactive,
    System,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MessageStatus {
    Pending,
    Sent,
    Delivered,
    Read,
    Failed,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MessageDirection {
    Inbound,
    Outbound,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Conversation {
    pub id: String,
    pub tenant_id: String,
    pub channel_id: String,
    pub contact_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub assigned_agent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bot_id: Option<String>,
    pub status: ConversationStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub priority: Option<ConversationPriority>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub subject: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<Box<Message>>,
    #[serde(default)]
    pub unread_count: i32,
    #[serde(default)]
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, serde_json::Value>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub first_message_at: Option<chrono::DateTime<chrono::Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message_at: Option<chrono::DateTime<chrono::Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resolved_at: Option<chrono::DateTime<chrono::Utc>>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Message {
    pub id: String,
    pub conversation_id: String,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    pub direction: MessageDirection,
    pub status: MessageStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub text: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MediaContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<LocationContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub contact: Option<ContactContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub template: Option<TemplateContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub interactive: Option<InteractiveContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sender_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sender_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub external_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, serde_json::Value>>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MediaContent {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mime_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub filename: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub caption: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LocationContent {
    pub latitude: f64,
    pub longitude: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub address: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ContactContent {
    pub name: String,
    #[serde(default)]
    pub phones: Vec<PhoneNumber>,
    #[serde(default)]
    pub emails: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PhoneNumber {
    #[serde(rename = "type")]
    pub phone_type: String,
    pub number: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TemplateContent {
    pub name: String,
    pub language: String,
    #[serde(default)]
    pub components: Vec<TemplateComponent>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TemplateComponent {
    #[serde(rename = "type")]
    pub component_type: String,
    #[serde(default)]
    pub parameters: Vec<TemplateParameter>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TemplateParameter {
    #[serde(rename = "type")]
    pub param_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub text: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub image: Option<MediaContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub document: Option<MediaContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub video: Option<MediaContent>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct InteractiveContent {
    #[serde(rename = "type")]
    pub interactive_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub header: Option<InteractiveHeader>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub body: Option<InteractiveBody>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub footer: Option<InteractiveFooter>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action: Option<InteractiveAction>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct InteractiveHeader {
    #[serde(rename = "type")]
    pub header_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub text: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub image: Option<MediaContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub video: Option<MediaContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub document: Option<MediaContent>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InteractiveBody {
    pub text: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InteractiveFooter {
    pub text: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct InteractiveAction {
    #[serde(default)]
    pub buttons: Vec<Button>,
    #[serde(default)]
    pub sections: Vec<Section>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Button {
    #[serde(rename = "type")]
    pub button_type: String,
    pub id: String,
    pub title: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Section {
    pub title: String,
    #[serde(default)]
    pub rows: Vec<SectionRow>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SectionRow {
    pub id: String,
    pub title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
}

// Input types

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ListConversationsParams {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<ConversationStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub priority: Option<ConversationPriority>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub channel_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub contact_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub assigned_agent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tag: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub search: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub limit: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cursor: Option<String>,
}

impl ListConversationsParams {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn status(mut self, status: ConversationStatus) -> Self {
        self.status = Some(status);
        self
    }

    pub fn priority(mut self, priority: ConversationPriority) -> Self {
        self.priority = Some(priority);
        self
    }

    pub fn channel_id(mut self, id: impl Into<String>) -> Self {
        self.channel_id = Some(id.into());
        self
    }

    pub fn limit(mut self, limit: i32) -> Self {
        self.limit = Some(limit);
        self
    }
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SendMessageInput {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub text: Option<String>,
    #[serde(rename = "type", skip_serializing_if = "Option::is_none")]
    pub message_type: Option<MessageType>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MediaContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<LocationContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub contact: Option<ContactContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub template: Option<TemplateContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub interactive: Option<InteractiveContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, serde_json::Value>>,
}

impl SendMessageInput {
    pub fn text(text: impl Into<String>) -> Self {
        Self {
            text: Some(text.into()),
            message_type: Some(MessageType::Text),
            ..Default::default()
        }
    }
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateConversationInput {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<ConversationStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub priority: Option<ConversationPriority>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub assigned_agent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, serde_json::Value>>,
}
