use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Output format for rendered images
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum VREOutputFormat {
    Png,
    Webp,
    Jpeg,
}

/// Channel type for VRE rendering
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum VREChannelType {
    Whatsapp,
    Telegram,
    Web,
    Email,
}

/// Available template types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum VRETemplateType {
    MenuOpcoes,
    CardProduto,
    StatusPedido,
    ListaProdutos,
    Confirmacao,
    CobrancaPix,
}

/// Order status for status_pedido template
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum OrderStatus {
    Recebido,
    Separacao,
    Faturado,
    Transporte,
    Entregue,
}

/// Stock status for products
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum StockStatus {
    Disponivel,
    Baixo,
    Indisponivel,
}

/// Render request
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct VRERenderRequest {
    pub tenant_id: String,
    pub template_id: String,
    pub data: HashMap<String, serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub channel: Option<VREChannelType>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub format: Option<VREOutputFormat>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub quality: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scale: Option<f64>,
}

impl VRERenderRequest {
    pub fn new(tenant_id: impl Into<String>, template_id: impl Into<String>, data: HashMap<String, serde_json::Value>) -> Self {
        Self {
            tenant_id: tenant_id.into(),
            template_id: template_id.into(),
            data,
            channel: None,
            format: None,
            width: None,
            quality: None,
            scale: None,
        }
    }

    pub fn channel(mut self, channel: VREChannelType) -> Self {
        self.channel = Some(channel);
        self
    }

    pub fn format(mut self, format: VREOutputFormat) -> Self {
        self.format = Some(format);
        self
    }

    pub fn width(mut self, width: i32) -> Self {
        self.width = Some(width);
        self
    }

    pub fn quality(mut self, quality: i32) -> Self {
        self.quality = Some(quality);
        self
    }

    pub fn scale(mut self, scale: f64) -> Self {
        self.scale = Some(scale);
        self
    }
}

/// Render response
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct VRERenderResponse {
    pub image_base64: String,
    pub caption: String,
    pub width: i32,
    pub height: i32,
    pub format: VREOutputFormat,
    pub render_time_ms: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub size_bytes: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cache_hit: Option<bool>,
}

/// Render and send request
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct VRERenderAndSendRequest {
    pub conversation_id: String,
    pub template_id: String,
    pub data: HashMap<String, serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub caption: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub follow_up_text: Option<String>,
}

impl VRERenderAndSendRequest {
    pub fn new(conversation_id: impl Into<String>, template_id: impl Into<String>, data: HashMap<String, serde_json::Value>) -> Self {
        Self {
            conversation_id: conversation_id.into(),
            template_id: template_id.into(),
            data,
            caption: None,
            follow_up_text: None,
        }
    }

    pub fn caption(mut self, caption: impl Into<String>) -> Self {
        self.caption = Some(caption.into());
        self
    }

    pub fn follow_up_text(mut self, follow_up_text: impl Into<String>) -> Self {
        self.follow_up_text = Some(follow_up_text.into());
        self
    }
}

/// Render and send response
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct VRERenderAndSendResponse {
    pub message_id: String,
    pub image_url: String,
    pub caption: String,
}

/// Template definition
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct VRETemplate {
    pub id: String,
    pub name: String,
    pub description: String,
    pub schema: HashMap<String, serde_json::Value>,
}

/// List templates response
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct VREListTemplatesResponse {
    pub templates: Vec<VRETemplate>,
}

/// Preview request
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct VREPreviewRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<HashMap<String, serde_json::Value>>,
}

impl VREPreviewRequest {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn data(mut self, data: HashMap<String, serde_json::Value>) -> Self {
        self.data = Some(data);
        self
    }
}

/// Preview response
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct VREPreviewResponse {
    pub image_base64: String,
    pub width: i32,
    pub height: i32,
}

/// Menu option for menu_opcoes template
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MenuOpcaoData {
    pub label: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub descricao: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub icone: Option<String>,
}

impl MenuOpcaoData {
    pub fn new(label: impl Into<String>) -> Self {
        Self {
            label: label.into(),
            descricao: None,
            icone: None,
        }
    }

    pub fn descricao(mut self, descricao: impl Into<String>) -> Self {
        self.descricao = Some(descricao.into());
        self
    }

    pub fn icone(mut self, icone: impl Into<String>) -> Self {
        self.icone = Some(icone.into());
        self
    }
}

/// Product card data
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct CardProdutoData {
    pub nome: String,
    pub preco: f64,
    pub unidade: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sku: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub estoque: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub imagem_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub destaque: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mensagem: Option<String>,
}

impl CardProdutoData {
    pub fn new(nome: impl Into<String>, preco: f64, unidade: impl Into<String>) -> Self {
        Self {
            nome: nome.into(),
            preco,
            unidade: unidade.into(),
            sku: None,
            estoque: None,
            imagem_url: None,
            destaque: None,
            mensagem: None,
        }
    }
}

/// Order status data
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct StatusPedidoData {
    pub numero_pedido: String,
    pub status_atual: OrderStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub itens_resumo: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub valor_total: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub previsao_entrega: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub motorista: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mensagem: Option<String>,
}

impl StatusPedidoData {
    pub fn new(numero_pedido: impl Into<String>, status_atual: OrderStatus) -> Self {
        Self {
            numero_pedido: numero_pedido.into(),
            status_atual,
            itens_resumo: None,
            valor_total: None,
            previsao_entrega: None,
            motorista: None,
            mensagem: None,
        }
    }
}

/// Product list item
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct ListaProdutoItem {
    pub nome: String,
    pub preco: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unidade: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub estoque_status: Option<StockStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sku: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub emoji: Option<String>,
}

impl ListaProdutoItem {
    pub fn new(nome: impl Into<String>, preco: f64) -> Self {
        Self {
            nome: nome.into(),
            preco,
            unidade: None,
            estoque_status: None,
            sku: None,
            emoji: None,
        }
    }
}

/// Confirmation item
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ConfirmacaoItem {
    pub nome: String,
    pub preco: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub quantidade: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub emoji: Option<String>,
}

impl ConfirmacaoItem {
    pub fn new(nome: impl Into<String>, preco: f64) -> Self {
        Self {
            nome: nome.into(),
            preco,
            quantidade: None,
            emoji: None,
        }
    }
}

/// PIX payment data
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub struct CobrancaPixData {
    pub valor: f64,
    pub pix_payload: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub numero_pedido: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expiracao: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mensagem: Option<String>,
}

impl CobrancaPixData {
    pub fn new(valor: f64, pix_payload: impl Into<String>) -> Self {
        Self {
            valor,
            pix_payload: pix_payload.into(),
            numero_pedido: None,
            expiracao: None,
            mensagem: None,
        }
    }
}
