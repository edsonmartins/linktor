use crate::error::{LinktorError, Result};
use crate::types::*;
use reqwest::{Client, Response, StatusCode};
use serde::{de::DeserializeOwned, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;

#[derive(Clone)]
pub struct LinktorClient {
    http: Client,
    base_url: String,
    api_key: Option<String>,
    access_token: Arc<RwLock<Option<String>>>,
    max_retries: u32,
}

impl LinktorClient {
    pub fn builder() -> LinktorClientBuilder {
        LinktorClientBuilder::default()
    }

    pub fn auth(&self) -> AuthResource {
        AuthResource { client: self.clone() }
    }

    pub fn conversations(&self) -> ConversationsResource {
        ConversationsResource { client: self.clone() }
    }

    pub fn contacts(&self) -> ContactsResource {
        ContactsResource { client: self.clone() }
    }

    pub fn channels(&self) -> ChannelsResource {
        ChannelsResource { client: self.clone() }
    }

    pub fn bots(&self) -> BotsResource {
        BotsResource { client: self.clone() }
    }

    pub fn ai(&self) -> AIResource {
        AIResource { client: self.clone() }
    }

    pub fn knowledge_bases(&self) -> KnowledgeBasesResource {
        KnowledgeBasesResource { client: self.clone() }
    }

    pub fn flows(&self) -> FlowsResource {
        FlowsResource { client: self.clone() }
    }

    pub fn vre(&self) -> VREResource {
        VREResource { client: self.clone() }
    }

    pub async fn set_access_token(&self, token: Option<String>) {
        let mut guard = self.access_token.write().await;
        *guard = token;
    }

    pub(crate) async fn request<T: DeserializeOwned>(
        &self,
        method: reqwest::Method,
        path: &str,
        body: Option<impl Serialize>,
    ) -> Result<T> {
        let url = format!("{}{}", self.base_url, path);
        let mut attempts = 0;

        loop {
            attempts += 1;

            let mut request = self.http.request(method.clone(), &url);

            // Add authentication
            if let Some(ref api_key) = self.api_key {
                request = request.header("X-API-Key", api_key);
            } else {
                let token_guard = self.access_token.read().await;
                if let Some(ref token) = *token_guard {
                    request = request.header("Authorization", format!("Bearer {}", token));
                }
            }

            // Add body
            if let Some(ref body) = body {
                request = request.json(body);
            }

            let response = request.send().await?;
            let status = response.status();
            let request_id = response
                .headers()
                .get("X-Request-ID")
                .and_then(|v| v.to_str().ok())
                .map(String::from);

            if status.is_success() {
                let text = response.text().await?;
                if text.is_empty() {
                    return Ok(serde_json::from_str("null")?);
                }

                // Try to parse as ApiResponse first
                if let Ok(api_response) = serde_json::from_str::<ApiResponse<T>>(&text) {
                    if api_response.success {
                        if let Some(data) = api_response.data {
                            return Ok(data);
                        }
                    }
                }

                // Parse directly
                return Ok(serde_json::from_str(&text)?);
            }

            // Handle rate limiting
            if status == StatusCode::TOO_MANY_REQUESTS && attempts < self.max_retries {
                let retry_after = response
                    .headers()
                    .get("Retry-After")
                    .and_then(|v| v.to_str().ok())
                    .and_then(|v| v.parse::<u64>().ok())
                    .unwrap_or(60);
                tokio::time::sleep(Duration::from_secs(retry_after)).await;
                continue;
            }

            // Handle server errors with retry
            if status.is_server_error() && attempts < self.max_retries {
                tokio::time::sleep(Duration::from_secs(2u64.pow(attempts))).await;
                continue;
            }

            let text = response.text().await.unwrap_or_default();
            let message = serde_json::from_str::<ApiError>(&text)
                .map(|e| e.message)
                .unwrap_or_else(|_| text);

            return Err(LinktorError::from_status(status, message, request_id));
        }
    }

    pub(crate) async fn get<T: DeserializeOwned>(&self, path: &str) -> Result<T> {
        self.request(reqwest::Method::GET, path, None::<()>).await
    }

    pub(crate) async fn post<T: DeserializeOwned>(&self, path: &str, body: impl Serialize) -> Result<T> {
        self.request(reqwest::Method::POST, path, Some(body)).await
    }

    pub(crate) async fn patch<T: DeserializeOwned>(&self, path: &str, body: impl Serialize) -> Result<T> {
        self.request(reqwest::Method::PATCH, path, Some(body)).await
    }

    pub(crate) async fn delete(&self, path: &str) -> Result<()> {
        self.request::<serde_json::Value>(reqwest::Method::DELETE, path, None::<()>).await?;
        Ok(())
    }
}

#[derive(Default)]
pub struct LinktorClientBuilder {
    base_url: Option<String>,
    api_key: Option<String>,
    access_token: Option<String>,
    timeout_secs: Option<u64>,
    max_retries: Option<u32>,
}

impl LinktorClientBuilder {
    pub fn base_url(mut self, url: impl Into<String>) -> Self {
        self.base_url = Some(url.into());
        self
    }

    pub fn api_key(mut self, key: impl Into<String>) -> Self {
        self.api_key = Some(key.into());
        self
    }

    pub fn access_token(mut self, token: impl Into<String>) -> Self {
        self.access_token = Some(token.into());
        self
    }

    pub fn timeout(mut self, secs: u64) -> Self {
        self.timeout_secs = Some(secs);
        self
    }

    pub fn max_retries(mut self, retries: u32) -> Self {
        self.max_retries = Some(retries);
        self
    }

    pub fn build(self) -> Result<LinktorClient> {
        let base_url = self.base_url.unwrap_or_else(|| "https://api.linktor.io".to_string());
        let base_url = base_url.trim_end_matches('/').to_string();

        let http = Client::builder()
            .timeout(Duration::from_secs(self.timeout_secs.unwrap_or(30)))
            .build()?;

        Ok(LinktorClient {
            http,
            base_url,
            api_key: self.api_key,
            access_token: Arc::new(RwLock::new(self.access_token)),
            max_retries: self.max_retries.unwrap_or(3),
        })
    }
}

// Resource implementations

pub struct AuthResource {
    client: LinktorClient,
}

impl AuthResource {
    pub async fn login(&self, email: &str, password: &str) -> Result<LoginResponse> {
        let input = LoginInput::new(email, password);
        let response: LoginResponse = self.client.post("/auth/login", input).await?;
        self.client.set_access_token(Some(response.access_token.clone())).await;
        Ok(response)
    }

    pub async fn logout(&self) -> Result<()> {
        self.client.post::<serde_json::Value>("/auth/logout", serde_json::json!({})).await?;
        self.client.set_access_token(None).await;
        Ok(())
    }

    pub async fn refresh_token(&self, refresh_token: &str) -> Result<RefreshTokenResponse> {
        let input = RefreshTokenInput {
            refresh_token: refresh_token.to_string(),
        };
        let response: RefreshTokenResponse = self.client.post("/auth/refresh", input).await?;
        self.client.set_access_token(Some(response.access_token.clone())).await;
        Ok(response)
    }

    pub async fn get_current_user(&self) -> Result<User> {
        self.client.get("/auth/me").await
    }

    pub async fn get_current_tenant(&self) -> Result<Tenant> {
        self.client.get("/auth/tenant").await
    }
}

pub struct ConversationsResource {
    client: LinktorClient,
}

impl ConversationsResource {
    pub async fn list(&self, params: Option<ListConversationsParams>) -> Result<PaginatedResponse<Conversation>> {
        let path = match params {
            Some(p) => format!("/conversations?{}", serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => "/conversations".to_string(),
        };
        self.client.get(&path).await
    }

    pub async fn get(&self, id: &str) -> Result<Conversation> {
        self.client.get(&format!("/conversations/{}", id)).await
    }

    pub async fn update(&self, id: &str, input: UpdateConversationInput) -> Result<Conversation> {
        self.client.patch(&format!("/conversations/{}", id), input).await
    }

    pub async fn send_text(&self, id: &str, text: &str) -> Result<Message> {
        let input = SendMessageInput::text(text);
        self.send_message(id, input).await
    }

    pub async fn send_message(&self, id: &str, input: SendMessageInput) -> Result<Message> {
        self.client.post(&format!("/conversations/{}/messages", id), input).await
    }

    pub async fn get_messages(&self, id: &str, params: Option<PaginationParams>) -> Result<PaginatedResponse<Message>> {
        let path = match params {
            Some(p) => format!("/conversations/{}/messages?{}", id, serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => format!("/conversations/{}/messages", id),
        };
        self.client.get(&path).await
    }

    pub async fn resolve(&self, id: &str) -> Result<Conversation> {
        self.client.post(&format!("/conversations/{}/resolve", id), serde_json::json!({})).await
    }

    pub async fn assign(&self, id: &str, agent_id: &str) -> Result<Conversation> {
        self.client.post(&format!("/conversations/{}/assign", id), serde_json::json!({"agentId": agent_id})).await
    }
}

pub struct ContactsResource {
    client: LinktorClient,
}

impl ContactsResource {
    pub async fn list(&self, params: Option<ListContactsParams>) -> Result<PaginatedResponse<Contact>> {
        let path = match params {
            Some(p) => format!("/contacts?{}", serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => "/contacts".to_string(),
        };
        self.client.get(&path).await
    }

    pub async fn get(&self, id: &str) -> Result<Contact> {
        self.client.get(&format!("/contacts/{}", id)).await
    }

    pub async fn create(&self, input: CreateContactInput) -> Result<Contact> {
        self.client.post("/contacts", input).await
    }

    pub async fn update(&self, id: &str, input: UpdateContactInput) -> Result<Contact> {
        self.client.patch(&format!("/contacts/{}", id), input).await
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        self.client.delete(&format!("/contacts/{}", id)).await
    }
}

pub struct ChannelsResource {
    client: LinktorClient,
}

impl ChannelsResource {
    pub async fn list(&self, params: Option<ListChannelsParams>) -> Result<PaginatedResponse<Channel>> {
        let path = match params {
            Some(p) => format!("/channels?{}", serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => "/channels".to_string(),
        };
        self.client.get(&path).await
    }

    pub async fn get(&self, id: &str) -> Result<Channel> {
        self.client.get(&format!("/channels/{}", id)).await
    }

    pub async fn create(&self, input: CreateChannelInput) -> Result<Channel> {
        self.client.post("/channels", input).await
    }

    pub async fn update(&self, id: &str, input: UpdateChannelInput) -> Result<Channel> {
        self.client.patch(&format!("/channels/{}", id), input).await
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        self.client.delete(&format!("/channels/{}", id)).await
    }

    pub async fn connect(&self, id: &str) -> Result<Channel> {
        self.client.post(&format!("/channels/{}/connect", id), serde_json::json!({})).await
    }

    pub async fn disconnect(&self, id: &str) -> Result<Channel> {
        self.client.post(&format!("/channels/{}/disconnect", id), serde_json::json!({})).await
    }
}

pub struct BotsResource {
    client: LinktorClient,
}

impl BotsResource {
    pub async fn list(&self, params: Option<ListBotsParams>) -> Result<PaginatedResponse<Bot>> {
        let path = match params {
            Some(p) => format!("/bots?{}", serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => "/bots".to_string(),
        };
        self.client.get(&path).await
    }

    pub async fn get(&self, id: &str) -> Result<Bot> {
        self.client.get(&format!("/bots/{}", id)).await
    }

    pub async fn create(&self, input: CreateBotInput) -> Result<Bot> {
        self.client.post("/bots", input).await
    }

    pub async fn update(&self, id: &str, input: UpdateBotInput) -> Result<Bot> {
        self.client.patch(&format!("/bots/{}", id), input).await
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        self.client.delete(&format!("/bots/{}", id)).await
    }
}

pub struct AIResource {
    client: LinktorClient,
}

impl AIResource {
    pub fn completions(&self) -> CompletionsResource {
        CompletionsResource { client: self.client.clone() }
    }

    pub fn embeddings(&self) -> EmbeddingsResource {
        EmbeddingsResource { client: self.client.clone() }
    }

    pub fn agents(&self) -> AgentsResource {
        AgentsResource { client: self.client.clone() }
    }
}

pub struct CompletionsResource {
    client: LinktorClient,
}

impl CompletionsResource {
    pub async fn complete(&self, question: &str) -> Result<String> {
        let messages = vec![ChatMessage::user(question)];
        let response = self.chat(messages).await?;
        Ok(response.content().unwrap_or_default().to_string())
    }

    pub async fn chat(&self, messages: Vec<ChatMessage>) -> Result<CompletionResponse> {
        let input = CompletionInput::new(messages);
        self.create(input).await
    }

    pub async fn create(&self, input: CompletionInput) -> Result<CompletionResponse> {
        self.client.post("/ai/completions", input).await
    }
}

pub struct EmbeddingsResource {
    client: LinktorClient,
}

impl EmbeddingsResource {
    pub async fn embed(&self, text: &str) -> Result<Vec<f64>> {
        let response = self.create(EmbeddingInput::single(text)).await?;
        Ok(response.embedding().unwrap_or_default().to_vec())
    }

    pub async fn create(&self, input: EmbeddingInput) -> Result<EmbeddingResponse> {
        self.client.post("/ai/embeddings", input).await
    }
}

pub struct AgentsResource {
    client: LinktorClient,
}

impl AgentsResource {
    pub async fn list(&self, params: Option<PaginationParams>) -> Result<PaginatedResponse<Agent>> {
        let path = match params {
            Some(p) => format!("/ai/agents?{}", serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => "/ai/agents".to_string(),
        };
        self.client.get(&path).await
    }

    pub async fn get(&self, id: &str) -> Result<Agent> {
        self.client.get(&format!("/ai/agents/{}", id)).await
    }

    pub async fn create(&self, input: CreateAgentInput) -> Result<Agent> {
        self.client.post("/ai/agents", input).await
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        self.client.delete(&format!("/ai/agents/{}", id)).await
    }
}

pub struct KnowledgeBasesResource {
    client: LinktorClient,
}

impl KnowledgeBasesResource {
    pub async fn list(&self, params: Option<PaginationParams>) -> Result<PaginatedResponse<KnowledgeBase>> {
        let path = match params {
            Some(p) => format!("/knowledge-bases?{}", serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => "/knowledge-bases".to_string(),
        };
        self.client.get(&path).await
    }

    pub async fn get(&self, id: &str) -> Result<KnowledgeBase> {
        self.client.get(&format!("/knowledge-bases/{}", id)).await
    }

    pub async fn create(&self, input: CreateKnowledgeBaseInput) -> Result<KnowledgeBase> {
        self.client.post("/knowledge-bases", input).await
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        self.client.delete(&format!("/knowledge-bases/{}", id)).await
    }

    pub async fn query(&self, id: &str, query: &str, top_k: i32) -> Result<QueryResult> {
        let input = QueryKnowledgeBaseInput::new(query).top_k(top_k);
        self.client.post(&format!("/knowledge-bases/{}/query", id), input).await
    }

    pub async fn add_document(&self, id: &str, input: AddDocumentInput) -> Result<Document> {
        self.client.post(&format!("/knowledge-bases/{}/documents", id), input).await
    }
}

pub struct FlowsResource {
    client: LinktorClient,
}

impl FlowsResource {
    pub async fn list(&self, params: Option<PaginationParams>) -> Result<PaginatedResponse<Flow>> {
        let path = match params {
            Some(p) => format!("/flows?{}", serde_urlencoded::to_string(&p).unwrap_or_default()),
            None => "/flows".to_string(),
        };
        self.client.get(&path).await
    }

    pub async fn get(&self, id: &str) -> Result<Flow> {
        self.client.get(&format!("/flows/{}", id)).await
    }

    pub async fn create(&self, input: CreateFlowInput) -> Result<Flow> {
        self.client.post("/flows", input).await
    }

    pub async fn update(&self, id: &str, input: UpdateFlowInput) -> Result<Flow> {
        self.client.patch(&format!("/flows/{}", id), input).await
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        self.client.delete(&format!("/flows/{}", id)).await
    }

    pub async fn execute(&self, id: &str, conversation_id: &str) -> Result<FlowExecution> {
        let input = ExecuteFlowInput::new(conversation_id);
        self.client.post(&format!("/flows/{}/execute", id), input).await
    }
}

pub struct VREResource {
    client: LinktorClient,
}

impl VREResource {
    /// Render a VRE template to an image.
    /// Returns base64-encoded image data that can be sent to messaging channels.
    pub async fn render(&self, request: VRERenderRequest) -> Result<VRERenderResponse> {
        self.client.post("/vre/render", request).await
    }

    /// Render a VRE template and send it directly to a conversation.
    /// Combines rendering and sending in one operation.
    pub async fn render_and_send(&self, request: VRERenderAndSendRequest) -> Result<VRERenderAndSendResponse> {
        self.client.post("/vre/render-and-send", request).await
    }

    /// List available VRE templates with their schemas and example data.
    pub async fn list_templates(&self, tenant_id: Option<&str>) -> Result<VREListTemplatesResponse> {
        let path = match tenant_id {
            Some(id) => format!("/vre/templates?tenant_id={}", id),
            None => "/vre/templates".to_string(),
        };
        self.client.get(&path).await
    }

    /// Preview a VRE template with sample data.
    pub async fn preview(&self, template_id: &str, data: Option<std::collections::HashMap<String, serde_json::Value>>) -> Result<VREPreviewResponse> {
        let request = VREPreviewRequest { data };
        self.client.post(&format!("/vre/templates/{}/preview", template_id), request).await
    }

    /// Render a menu with numbered options.
    pub async fn render_menu(
        &self,
        tenant_id: &str,
        titulo: &str,
        opcoes: Vec<MenuOpcaoData>,
        channel: VREChannelType,
    ) -> Result<VRERenderResponse> {
        let mut data = std::collections::HashMap::new();
        data.insert("titulo".to_string(), serde_json::json!(titulo));
        data.insert("opcoes".to_string(), serde_json::to_value(&opcoes).unwrap_or_default());

        let request = VRERenderRequest::new(tenant_id, "menu_opcoes", data)
            .channel(channel);
        self.render(request).await
    }

    /// Render a product card.
    pub async fn render_product_card(
        &self,
        tenant_id: &str,
        produto: CardProdutoData,
        channel: VREChannelType,
    ) -> Result<VRERenderResponse> {
        let data = serde_json::to_value(&produto)
            .map(|v| v.as_object().cloned().unwrap_or_default())
            .unwrap_or_default()
            .into_iter()
            .map(|(k, v)| (k, v))
            .collect();

        let request = VRERenderRequest::new(tenant_id, "card_produto", data)
            .channel(channel);
        self.render(request).await
    }

    /// Render an order status timeline.
    pub async fn render_order_status(
        &self,
        tenant_id: &str,
        status: StatusPedidoData,
        channel: VREChannelType,
    ) -> Result<VRERenderResponse> {
        let data = serde_json::to_value(&status)
            .map(|v| v.as_object().cloned().unwrap_or_default())
            .unwrap_or_default()
            .into_iter()
            .map(|(k, v)| (k, v))
            .collect();

        let request = VRERenderRequest::new(tenant_id, "status_pedido", data)
            .channel(channel);
        self.render(request).await
    }

    /// Render a product list for comparison.
    pub async fn render_product_list(
        &self,
        tenant_id: &str,
        titulo: &str,
        produtos: Vec<ListaProdutoItem>,
        channel: VREChannelType,
    ) -> Result<VRERenderResponse> {
        let mut data = std::collections::HashMap::new();
        data.insert("titulo".to_string(), serde_json::json!(titulo));
        data.insert("produtos".to_string(), serde_json::to_value(&produtos).unwrap_or_default());

        let request = VRERenderRequest::new(tenant_id, "lista_produtos", data)
            .channel(channel);
        self.render(request).await
    }

    /// Render a confirmation summary.
    pub async fn render_confirmation(
        &self,
        tenant_id: &str,
        valor_total: f64,
        itens: Vec<ConfirmacaoItem>,
        channel: VREChannelType,
    ) -> Result<VRERenderResponse> {
        let mut data = std::collections::HashMap::new();
        data.insert("valor_total".to_string(), serde_json::json!(valor_total));
        data.insert("itens".to_string(), serde_json::to_value(&itens).unwrap_or_default());

        let request = VRERenderRequest::new(tenant_id, "confirmacao", data)
            .channel(channel);
        self.render(request).await
    }

    /// Render a PIX payment QR code.
    pub async fn render_pix_payment(
        &self,
        tenant_id: &str,
        pix: CobrancaPixData,
        channel: VREChannelType,
    ) -> Result<VRERenderResponse> {
        let data = serde_json::to_value(&pix)
            .map(|v| v.as_object().cloned().unwrap_or_default())
            .unwrap_or_default()
            .into_iter()
            .map(|(k, v)| (k, v))
            .collect();

        let request = VRERenderRequest::new(tenant_id, "cobranca_pix", data)
            .channel(channel);
        self.render(request).await
    }
}
