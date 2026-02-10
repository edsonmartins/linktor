"""
VRE (Visual Response Engine) types
"""

from enum import Enum
from typing import Any, Optional
from pydantic import BaseModel, Field


class VREOutputFormat(str, Enum):
    """Output format for rendered images"""

    PNG = "png"
    WEBP = "webp"
    JPEG = "jpeg"


class VREChannelType(str, Enum):
    """Channel type for VRE rendering"""

    WHATSAPP = "whatsapp"
    TELEGRAM = "telegram"
    WEB = "web"
    EMAIL = "email"


class VRETemplateType(str, Enum):
    """Available template types"""

    MENU_OPCOES = "menu_opcoes"
    CARD_PRODUTO = "card_produto"
    STATUS_PEDIDO = "status_pedido"
    LISTA_PRODUTOS = "lista_produtos"
    CONFIRMACAO = "confirmacao"
    COBRANCA_PIX = "cobranca_pix"


class OrderStatus(str, Enum):
    """Order status for status_pedido template"""

    RECEBIDO = "recebido"
    SEPARACAO = "separacao"
    FATURADO = "faturado"
    TRANSPORTE = "transporte"
    ENTREGUE = "entregue"


class StockStatus(str, Enum):
    """Stock status for products"""

    DISPONIVEL = "disponivel"
    BAIXO = "baixo"
    INDISPONIVEL = "indisponivel"


# ============================================
# Render Request/Response
# ============================================


class VRERenderRequest(BaseModel):
    """Request to render a VRE template"""

    tenant_id: str = Field(alias="tenantId")
    template_id: VRETemplateType = Field(alias="templateId")
    data: dict[str, Any]
    channel: Optional[VREChannelType] = None
    format: Optional[VREOutputFormat] = None
    width: Optional[int] = None
    quality: Optional[int] = None
    scale: Optional[float] = None

    class Config:
        populate_by_name = True


class VRERenderResponse(BaseModel):
    """Response from rendering a template"""

    image_base64: str = Field(alias="imageBase64")
    caption: str
    width: int
    height: int
    format: VREOutputFormat
    render_time_ms: int = Field(alias="renderTimeMs")
    size_bytes: Optional[int] = Field(None, alias="sizeBytes")
    cache_hit: Optional[bool] = Field(None, alias="cacheHit")

    class Config:
        populate_by_name = True


# ============================================
# Render and Send
# ============================================


class VRERenderAndSendRequest(BaseModel):
    """Request to render a template and send it directly to a conversation"""

    conversation_id: str = Field(alias="conversationId")
    template_id: VRETemplateType = Field(alias="templateId")
    data: dict[str, Any]
    caption: Optional[str] = None
    follow_up_text: Optional[str] = Field(None, alias="followUpText")

    class Config:
        populate_by_name = True


class VRERenderAndSendResponse(BaseModel):
    """Response from render and send"""

    message_id: str = Field(alias="messageId")
    image_url: str = Field(alias="imageUrl")
    caption: str

    class Config:
        populate_by_name = True


# ============================================
# Templates
# ============================================


class VRETemplate(BaseModel):
    """VRE template definition"""

    id: VRETemplateType
    name: str
    description: str
    schema: dict[str, Any]


class VREListTemplatesResponse(BaseModel):
    """Response from listing templates"""

    templates: list[VRETemplate]


class VREPreviewRequest(BaseModel):
    """Request to preview a template"""

    template_id: VRETemplateType = Field(alias="templateId")
    data: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class VREPreviewResponse(BaseModel):
    """Response from template preview"""

    image_base64: str = Field(alias="imageBase64")
    width: int
    height: int

    class Config:
        populate_by_name = True


# ============================================
# Template Data Types
# ============================================


class MenuOpcaoData(BaseModel):
    """Menu option for menu_opcoes template"""

    label: str
    descricao: Optional[str] = None
    icone: Optional[str] = None


class MenuOpcoesData(BaseModel):
    """Data for menu_opcoes template"""

    titulo: str
    subtitulo: Optional[str] = None
    opcoes: list[MenuOpcaoData]
    mensagem_antes: Optional[str] = Field(None, alias="mensagemAntes")

    class Config:
        populate_by_name = True


class CardProdutoData(BaseModel):
    """Data for card_produto template"""

    nome: str
    sku: Optional[str] = None
    preco: float
    unidade: str
    estoque: Optional[int] = None
    imagem_url: Optional[str] = Field(None, alias="imagemUrl")
    destaque: Optional[str] = None
    mensagem: Optional[str] = None

    class Config:
        populate_by_name = True


class StatusPedidoStep(BaseModel):
    """Timeline step for status_pedido template"""

    status: str
    icon: str
    label: str


class StatusPedidoData(BaseModel):
    """Data for status_pedido template"""

    numero_pedido: str = Field(alias="numeroPedido")
    status_atual: OrderStatus = Field(alias="statusAtual")
    itens_resumo: Optional[str] = Field(None, alias="itensResumo")
    valor_total: Optional[float] = Field(None, alias="valorTotal")
    previsao_entrega: Optional[str] = Field(None, alias="previsaoEntrega")
    motorista: Optional[str] = None
    mensagem: Optional[str] = None

    class Config:
        populate_by_name = True


class ListaProdutoItem(BaseModel):
    """Product item for lista_produtos template"""

    nome: str
    preco: float
    unidade: Optional[str] = None
    estoque_status: Optional[StockStatus] = Field(None, alias="estoqueStatus")
    sku: Optional[str] = None
    emoji: Optional[str] = None

    class Config:
        populate_by_name = True


class ListaProdutosData(BaseModel):
    """Data for lista_produtos template"""

    titulo: str
    produtos: list[ListaProdutoItem]
    mensagem: Optional[str] = None


class ConfirmacaoItem(BaseModel):
    """Confirmation item for confirmacao template"""

    nome: str
    quantidade: Optional[str] = None
    preco: float
    emoji: Optional[str] = None


class ConfirmacaoData(BaseModel):
    """Data for confirmacao template"""

    titulo: Optional[str] = None
    subtitulo: Optional[str] = None
    itens: list[ConfirmacaoItem]
    valor_total: float = Field(alias="valorTotal")
    previsao_entrega: Optional[str] = Field(None, alias="previsaoEntrega")
    mensagem: Optional[str] = None

    class Config:
        populate_by_name = True


class CobrancaPixData(BaseModel):
    """Data for cobranca_pix template"""

    valor: float
    numero_pedido: Optional[str] = Field(None, alias="numeroPedido")
    pix_payload: str = Field(alias="pixPayload")
    expiracao: Optional[str] = None
    mensagem: Optional[str] = None

    class Config:
        populate_by_name = True


__all__ = [
    "VREOutputFormat",
    "VREChannelType",
    "VRETemplateType",
    "OrderStatus",
    "StockStatus",
    "VRERenderRequest",
    "VRERenderResponse",
    "VRERenderAndSendRequest",
    "VRERenderAndSendResponse",
    "VRETemplate",
    "VREListTemplatesResponse",
    "VREPreviewRequest",
    "VREPreviewResponse",
    "MenuOpcaoData",
    "MenuOpcoesData",
    "CardProdutoData",
    "StatusPedidoStep",
    "StatusPedidoData",
    "ListaProdutoItem",
    "ListaProdutosData",
    "ConfirmacaoItem",
    "ConfirmacaoData",
    "CobrancaPixData",
]
