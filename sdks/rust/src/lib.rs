//! # Linktor SDK for Rust
//!
//! Official Linktor SDK for Rust applications.
//!
//! ## Quick Start
//!
//! ```rust,no_run
//! use linktor::{LinktorClient, types::*};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), linktor::Error> {
//!     let client = LinktorClient::builder()
//!         .base_url("https://api.linktor.io")
//!         .api_key("your-api-key")
//!         .build()?;
//!
//!     // List conversations
//!     let convs = client.conversations()
//!         .list(Some(ListConversationsParams::new().status(ConversationStatus::Open)))
//!         .await?;
//!     println!("Found {} conversations", convs.data.len());
//!
//!     // Send a message
//!     let msg = client.conversations().send_text("conv-id", "Hello!").await?;
//!     println!("Sent message: {}", msg.id);
//!
//!     // Use AI
//!     let answer = client.ai().completions().complete("What is 2 + 2?").await?;
//!     println!("{}", answer); // "4"
//!
//!     Ok(())
//! }
//! ```
//!
//! ## Authentication
//!
//! ### API Key
//!
//! ```rust,no_run
//! # use linktor::LinktorClient;
//! let client = LinktorClient::builder()
//!     .api_key("your-api-key")
//!     .build()?;
//! # Ok::<(), linktor::Error>(())
//! ```
//!
//! ### Access Token
//!
//! ```rust,no_run
//! # use linktor::LinktorClient;
//! let client = LinktorClient::builder()
//!     .access_token("user-token")
//!     .build()?;
//! # Ok::<(), linktor::Error>(())
//! ```
//!
//! ## Webhooks
//!
//! ```rust,no_run
//! use linktor::webhook;
//! use std::collections::HashMap;
//!
//! fn handle_webhook(payload: &[u8], headers: HashMap<String, String>) {
//!     let secret = "your-webhook-secret";
//!
//!     // Verify and parse
//!     match webhook::construct_event(payload, &headers, secret, None) {
//!         Ok(event) => {
//!             println!("Received event: {}", event.event_type);
//!         }
//!         Err(e) => {
//!             println!("Webhook error: {}", e);
//!         }
//!     }
//! }
//! ```

pub mod client;
pub mod error;
pub mod types;
pub mod webhook;

pub use client::{
    LinktorClient, LinktorClientBuilder,
    AuthResource, ConversationsResource, ContactsResource,
    ChannelsResource, BotsResource, AIResource,
    KnowledgeBasesResource, FlowsResource,
    CompletionsResource, EmbeddingsResource, AgentsResource,
};
pub use error::{LinktorError, Result};
pub use types::*;

/// Type alias for the main error type
pub type Error = LinktorError;
