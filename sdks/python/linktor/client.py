"""
Linktor Client - Main SDK entry point
"""

from typing import Any, AsyncIterator, Callable, Optional

from linktor.utils.http import HttpClient, AsyncHttpClient
from linktor.types import (
    PaginatedResponse,
    Conversation,
    Message,
    Contact,
    Channel,
    Bot,
    Agent,
    KnowledgeBase,
    Document,
    Flow,
    FlowExecution,
    User,
    LoginResponse,
    RefreshTokenResponse,
    VRERenderResponse,
    VRERenderAndSendResponse,
    VREListTemplatesResponse,
    VREPreviewResponse,
    VRETemplateType,
    VREChannelType,
    VREOutputFormat,
)


class AuthResource:
    """Auth resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def login(self, email: str, password: str) -> LoginResponse:
        """Login with email and password"""
        data = self._http.post("/auth/login", {"email": email, "password": password})
        response = LoginResponse(**data)
        self._http.set_access_token(response.access_token)
        return response

    def logout(self) -> None:
        """Logout"""
        self._http.post("/auth/logout")

    def refresh_token(self, refresh_token: str) -> RefreshTokenResponse:
        """Refresh access token"""
        data = self._http.post("/auth/refresh", {"refreshToken": refresh_token})
        response = RefreshTokenResponse(**data)
        self._http.set_access_token(response.access_token)
        return response

    def get_current_user(self) -> User:
        """Get current user"""
        data = self._http.get("/auth/me")
        return User(**data)


class ConversationsResource:
    """Conversations resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def list(self, **params: Any) -> PaginatedResponse[Conversation]:
        """List conversations"""
        data = self._http.get("/conversations", params=params)
        return PaginatedResponse[Conversation](**data)

    def get(self, id: str) -> Conversation:
        """Get conversation"""
        data = self._http.get(f"/conversations/{id}")
        return Conversation(**data)

    def update(self, id: str, **kwargs: Any) -> Conversation:
        """Update conversation"""
        data = self._http.patch(f"/conversations/{id}", kwargs)
        return Conversation(**data)

    def send_message(self, conversation_id: str, **kwargs: Any) -> Message:
        """Send message"""
        data = self._http.post(f"/conversations/{conversation_id}/messages", kwargs)
        return Message(**data)

    def send_text(self, conversation_id: str, text: str) -> Message:
        """Send text message"""
        return self.send_message(conversation_id, text=text)

    def list_messages(self, conversation_id: str, **params: Any) -> PaginatedResponse[Message]:
        """List messages"""
        data = self._http.get(f"/conversations/{conversation_id}/messages", params=params)
        return PaginatedResponse[Message](**data)

    def resolve(self, id: str) -> Conversation:
        """Resolve conversation"""
        return self.update(id, status="resolved")

    def assign(self, id: str, agent_id: str) -> Conversation:
        """Assign conversation"""
        data = self._http.post(f"/conversations/{id}/assign", {"agentId": agent_id})
        return Conversation(**data)


class ContactsResource:
    """Contacts resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def list(self, **params: Any) -> PaginatedResponse[Contact]:
        """List contacts"""
        data = self._http.get("/contacts", params=params)
        return PaginatedResponse[Contact](**data)

    def get(self, id: str) -> Contact:
        """Get contact"""
        data = self._http.get(f"/contacts/{id}")
        return Contact(**data)

    def create(self, **kwargs: Any) -> Contact:
        """Create contact"""
        data = self._http.post("/contacts", kwargs)
        return Contact(**data)

    def update(self, id: str, **kwargs: Any) -> Contact:
        """Update contact"""
        data = self._http.patch(f"/contacts/{id}", kwargs)
        return Contact(**data)

    def delete(self, id: str) -> None:
        """Delete contact"""
        self._http.delete(f"/contacts/{id}")

    def search(self, query: str, **params: Any) -> PaginatedResponse[Contact]:
        """Search contacts"""
        return self.list(search=query, **params)


class ChannelsResource:
    """Channels resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def list(self, **params: Any) -> PaginatedResponse[Channel]:
        """List channels"""
        data = self._http.get("/channels", params=params)
        return PaginatedResponse[Channel](**data)

    def get(self, id: str) -> Channel:
        """Get channel"""
        data = self._http.get(f"/channels/{id}")
        return Channel(**data)

    def create(self, **kwargs: Any) -> Channel:
        """Create channel"""
        data = self._http.post("/channels", kwargs)
        return Channel(**data)

    def update(self, id: str, **kwargs: Any) -> Channel:
        """Update channel"""
        data = self._http.patch(f"/channels/{id}", kwargs)
        return Channel(**data)

    def delete(self, id: str) -> None:
        """Delete channel"""
        self._http.delete(f"/channels/{id}")

    def connect(self, id: str) -> Channel:
        """Connect channel"""
        data = self._http.post(f"/channels/{id}/connect")
        return Channel(**data)

    def disconnect(self, id: str) -> Channel:
        """Disconnect channel"""
        data = self._http.post(f"/channels/{id}/disconnect")
        return Channel(**data)


class BotsResource:
    """Bots resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def list(self, **params: Any) -> PaginatedResponse[Bot]:
        """List bots"""
        data = self._http.get("/bots", params=params)
        return PaginatedResponse[Bot](**data)

    def get(self, id: str) -> Bot:
        """Get bot"""
        data = self._http.get(f"/bots/{id}")
        return Bot(**data)

    def create(self, **kwargs: Any) -> Bot:
        """Create bot"""
        data = self._http.post("/bots", kwargs)
        return Bot(**data)

    def update(self, id: str, **kwargs: Any) -> Bot:
        """Update bot"""
        data = self._http.patch(f"/bots/{id}", kwargs)
        return Bot(**data)

    def delete(self, id: str) -> None:
        """Delete bot"""
        self._http.delete(f"/bots/{id}")

    def activate(self, id: str) -> Bot:
        """Activate bot"""
        return self.update(id, status="active")

    def deactivate(self, id: str) -> Bot:
        """Deactivate bot"""
        return self.update(id, status="inactive")


class AIResource:
    """AI resource (agents, completions, embeddings)"""

    def __init__(self, http: HttpClient):
        self._http = http
        self.agents = AgentsSubResource(http)
        self.completions = CompletionsSubResource(http)
        self.embeddings = EmbeddingsSubResource(http)


class AgentsSubResource:
    """Agents sub-resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def list(self, **params: Any) -> PaginatedResponse[Agent]:
        """List agents"""
        data = self._http.get("/ai/agents", params=params)
        return PaginatedResponse[Agent](**data)

    def get(self, id: str) -> Agent:
        """Get agent"""
        data = self._http.get(f"/ai/agents/{id}")
        return Agent(**data)

    def create(self, **kwargs: Any) -> Agent:
        """Create agent"""
        data = self._http.post("/ai/agents", kwargs)
        return Agent(**data)

    def update(self, id: str, **kwargs: Any) -> Agent:
        """Update agent"""
        data = self._http.patch(f"/ai/agents/{id}", kwargs)
        return Agent(**data)

    def delete(self, id: str) -> None:
        """Delete agent"""
        self._http.delete(f"/ai/agents/{id}")

    def invoke(self, id: str, message: str, **kwargs: Any) -> dict[str, Any]:
        """Invoke agent"""
        return self._http.post(f"/ai/agents/{id}/invoke", {"message": message, **kwargs})


class CompletionsSubResource:
    """Completions sub-resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def create(self, messages: list[dict[str, str]], **kwargs: Any) -> dict[str, Any]:
        """Create completion"""
        return self._http.post("/ai/completions", {"messages": messages, **kwargs})

    def complete(self, prompt: str, **kwargs: Any) -> str:
        """Simple completion"""
        response = self.create([{"role": "user", "content": prompt}], **kwargs)
        return response["message"]["content"]


class EmbeddingsSubResource:
    """Embeddings sub-resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def create(self, input: str | list[str], **kwargs: Any) -> dict[str, Any]:
        """Create embeddings"""
        return self._http.post("/ai/embeddings", {"input": input, **kwargs})

    def embed(self, text: str, **kwargs: Any) -> list[float]:
        """Embed single text"""
        response = self.create(text, **kwargs)
        return response["data"][0]["embedding"]


class KnowledgeBasesResource:
    """Knowledge bases resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def list(self, **params: Any) -> PaginatedResponse[KnowledgeBase]:
        """List knowledge bases"""
        data = self._http.get("/knowledge-bases", params=params)
        return PaginatedResponse[KnowledgeBase](**data)

    def get(self, id: str) -> KnowledgeBase:
        """Get knowledge base"""
        data = self._http.get(f"/knowledge-bases/{id}")
        return KnowledgeBase(**data)

    def create(self, **kwargs: Any) -> KnowledgeBase:
        """Create knowledge base"""
        data = self._http.post("/knowledge-bases", kwargs)
        return KnowledgeBase(**data)

    def update(self, id: str, **kwargs: Any) -> KnowledgeBase:
        """Update knowledge base"""
        data = self._http.patch(f"/knowledge-bases/{id}", kwargs)
        return KnowledgeBase(**data)

    def delete(self, id: str) -> None:
        """Delete knowledge base"""
        self._http.delete(f"/knowledge-bases/{id}")

    def query(self, id: str, query: str, top_k: int = 5, **kwargs: Any) -> dict[str, Any]:
        """Query knowledge base"""
        return self._http.post(f"/knowledge-bases/{id}/query", {"query": query, "topK": top_k, **kwargs})

    def search(self, id: str, query: str, top_k: int = 5) -> list[str]:
        """Simple search returning text results"""
        result = self.query(id, query, top_k)
        return [chunk["content"] for chunk in result.get("chunks", [])]

    def list_documents(self, kb_id: str, **params: Any) -> PaginatedResponse[Document]:
        """List documents"""
        data = self._http.get(f"/knowledge-bases/{kb_id}/documents", params=params)
        return PaginatedResponse[Document](**data)

    def upload_document(self, kb_id: str, file: bytes, filename: str, **kwargs: Any) -> Document:
        """Upload document"""
        data = self._http.upload(f"/knowledge-bases/{kb_id}/documents", file, filename, **kwargs)
        return Document(**data)

    def delete_document(self, kb_id: str, doc_id: str) -> None:
        """Delete document"""
        self._http.delete(f"/knowledge-bases/{kb_id}/documents/{doc_id}")


class FlowsResource:
    """Flows resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def list(self, **params: Any) -> PaginatedResponse[Flow]:
        """List flows"""
        data = self._http.get("/flows", params=params)
        return PaginatedResponse[Flow](**data)

    def get(self, id: str) -> Flow:
        """Get flow"""
        data = self._http.get(f"/flows/{id}")
        return Flow(**data)

    def create(self, **kwargs: Any) -> Flow:
        """Create flow"""
        data = self._http.post("/flows", kwargs)
        return Flow(**data)

    def update(self, id: str, **kwargs: Any) -> Flow:
        """Update flow"""
        data = self._http.patch(f"/flows/{id}", kwargs)
        return Flow(**data)

    def delete(self, id: str) -> None:
        """Delete flow"""
        self._http.delete(f"/flows/{id}")

    def execute(self, id: str, conversation_id: str, **kwargs: Any) -> FlowExecution:
        """Execute flow"""
        data = self._http.post(f"/flows/{id}/execute", {"conversationId": conversation_id, **kwargs})
        return FlowExecution(**data)

    def activate(self, id: str) -> Flow:
        """Activate flow"""
        return self.update(id, status="active")

    def deactivate(self, id: str) -> Flow:
        """Deactivate flow"""
        return self.update(id, status="inactive")


class AnalyticsResource:
    """Analytics resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def get_dashboard(self, **params: Any) -> dict[str, Any]:
        """Get dashboard metrics"""
        return self._http.get("/analytics/dashboard", params=params)

    def get_conversation_metrics(self, **params: Any) -> dict[str, Any]:
        """Get conversation metrics"""
        return self._http.get("/analytics/conversations", params=params)

    def get_message_metrics(self, **params: Any) -> dict[str, Any]:
        """Get message metrics"""
        return self._http.get("/analytics/messages", params=params)

    def get_realtime(self) -> dict[str, Any]:
        """Get realtime metrics"""
        return self._http.get("/analytics/realtime")


class VREResource:
    """VRE (Visual Response Engine) resource"""

    def __init__(self, http: HttpClient):
        self._http = http

    def render(
        self,
        tenant_id: str,
        template_id: str,
        data: dict[str, Any],
        channel: Optional[str] = None,
        format: Optional[str] = None,
        width: Optional[int] = None,
        quality: Optional[int] = None,
        scale: Optional[float] = None,
    ) -> VRERenderResponse:
        """
        Render a VRE template to an image.

        Args:
            tenant_id: Tenant ID
            template_id: Template to render (menu_opcoes, card_produto, etc.)
            data: Template data
            channel: Target channel for optimization (whatsapp, telegram, web, email)
            format: Output format (png, webp, jpeg)
            width: Override default width
            quality: Quality 0-100 (for webp/jpeg)
            scale: Scale factor (1.0-2.0)

        Returns:
            VRERenderResponse with image_base64 and caption
        """
        payload = {
            "tenant_id": tenant_id,
            "template_id": template_id,
            "data": data,
        }
        if channel:
            payload["channel"] = channel
        if format:
            payload["format"] = format
        if width:
            payload["width"] = width
        if quality:
            payload["quality"] = quality
        if scale:
            payload["scale"] = scale

        response = self._http.post("/vre/render", payload)
        return VRERenderResponse(**response)

    def render_and_send(
        self,
        conversation_id: str,
        template_id: str,
        data: dict[str, Any],
        caption: Optional[str] = None,
        follow_up_text: Optional[str] = None,
    ) -> VRERenderAndSendResponse:
        """
        Render a template and send it directly to a conversation.

        Args:
            conversation_id: Conversation to send to
            template_id: Template to render
            data: Template data
            caption: Optional custom caption
            follow_up_text: Optional text to send after the image

        Returns:
            VRERenderAndSendResponse with message_id
        """
        payload = {
            "conversation_id": conversation_id,
            "template_id": template_id,
            "data": data,
        }
        if caption:
            payload["caption"] = caption
        if follow_up_text:
            payload["follow_up_text"] = follow_up_text

        response = self._http.post("/vre/render-and-send", payload)
        return VRERenderAndSendResponse(**response)

    def list_templates(self, tenant_id: Optional[str] = None) -> VREListTemplatesResponse:
        """
        List available VRE templates.

        Args:
            tenant_id: Optional tenant ID to include custom templates

        Returns:
            VREListTemplatesResponse with templates list
        """
        params = {}
        if tenant_id:
            params["tenant_id"] = tenant_id
        response = self._http.get("/vre/templates", params=params if params else None)
        return VREListTemplatesResponse(**response)

    def preview(
        self, template_id: str, data: Optional[dict[str, Any]] = None
    ) -> VREPreviewResponse:
        """
        Preview a VRE template with sample data.

        Args:
            template_id: Template to preview
            data: Optional custom data (uses defaults if not provided)

        Returns:
            VREPreviewResponse with image_base64
        """
        payload = {}
        if data:
            payload["data"] = data
        response = self._http.post(f"/vre/templates/{template_id}/preview", payload if payload else None)
        return VREPreviewResponse(**response)

    # Convenience methods

    def render_menu(
        self,
        tenant_id: str,
        titulo: str,
        opcoes: list[dict[str, Any]],
        subtitulo: Optional[str] = None,
        channel: str = "whatsapp",
    ) -> VRERenderResponse:
        """
        Render a menu with numbered options.

        Args:
            tenant_id: Tenant ID
            titulo: Menu title
            opcoes: List of options with label, icone, descricao
            subtitulo: Optional subtitle
            channel: Target channel

        Returns:
            VRERenderResponse
        """
        data = {"titulo": titulo, "opcoes": opcoes}
        if subtitulo:
            data["subtitulo"] = subtitulo
        return self.render(tenant_id, "menu_opcoes", data, channel=channel)

    def render_product_card(
        self,
        tenant_id: str,
        nome: str,
        preco: float,
        unidade: str,
        sku: Optional[str] = None,
        estoque: Optional[int] = None,
        imagem_url: Optional[str] = None,
        destaque: Optional[str] = None,
        channel: str = "whatsapp",
    ) -> VRERenderResponse:
        """
        Render a product card.

        Args:
            tenant_id: Tenant ID
            nome: Product name
            preco: Product price
            unidade: Unit (kg, un, cx, etc.)
            sku: Product SKU
            estoque: Stock quantity
            imagem_url: Product image URL
            destaque: Highlight badge
            channel: Target channel

        Returns:
            VRERenderResponse
        """
        data: dict[str, Any] = {"nome": nome, "preco": preco, "unidade": unidade}
        if sku:
            data["sku"] = sku
        if estoque is not None:
            data["estoque"] = estoque
        if imagem_url:
            data["imagem_url"] = imagem_url
        if destaque:
            data["destaque"] = destaque
        return self.render(tenant_id, "card_produto", data, channel=channel)

    def render_order_status(
        self,
        tenant_id: str,
        numero_pedido: str,
        status_atual: str,
        itens_resumo: Optional[str] = None,
        valor_total: Optional[float] = None,
        previsao_entrega: Optional[str] = None,
        motorista: Optional[str] = None,
        channel: str = "whatsapp",
    ) -> VRERenderResponse:
        """
        Render an order status timeline.

        Args:
            tenant_id: Tenant ID
            numero_pedido: Order number
            status_atual: Current status (recebido, separacao, faturado, transporte, entregue)
            itens_resumo: Items summary
            valor_total: Total value
            previsao_entrega: Delivery estimate
            motorista: Driver name
            channel: Target channel

        Returns:
            VRERenderResponse
        """
        data: dict[str, Any] = {
            "numero_pedido": numero_pedido,
            "status_atual": status_atual,
        }
        if itens_resumo:
            data["itens_resumo"] = itens_resumo
        if valor_total is not None:
            data["valor_total"] = valor_total
        if previsao_entrega:
            data["previsao_entrega"] = previsao_entrega
        if motorista:
            data["motorista"] = motorista
        return self.render(tenant_id, "status_pedido", data, channel=channel)

    def render_product_list(
        self,
        tenant_id: str,
        titulo: str,
        produtos: list[dict[str, Any]],
        mensagem: Optional[str] = None,
        channel: str = "whatsapp",
    ) -> VRERenderResponse:
        """
        Render a product list for comparison.

        Args:
            tenant_id: Tenant ID
            titulo: List title
            produtos: List of products with nome, preco, unidade, etc.
            mensagem: Optional message
            channel: Target channel

        Returns:
            VRERenderResponse
        """
        data: dict[str, Any] = {"titulo": titulo, "produtos": produtos}
        if mensagem:
            data["mensagem"] = mensagem
        return self.render(tenant_id, "lista_produtos", data, channel=channel)

    def render_confirmation(
        self,
        tenant_id: str,
        valor_total: float,
        itens: list[dict[str, Any]],
        titulo: Optional[str] = None,
        subtitulo: Optional[str] = None,
        previsao_entrega: Optional[str] = None,
        mensagem: Optional[str] = None,
        channel: str = "whatsapp",
    ) -> VRERenderResponse:
        """
        Render a confirmation summary.

        Args:
            tenant_id: Tenant ID
            valor_total: Total value
            itens: List of items with nome, quantidade, preco
            titulo: Confirmation title
            subtitulo: Subtitle
            previsao_entrega: Delivery estimate
            mensagem: Optional message
            channel: Target channel

        Returns:
            VRERenderResponse
        """
        data: dict[str, Any] = {"valor_total": valor_total, "itens": itens}
        if titulo:
            data["titulo"] = titulo
        if subtitulo:
            data["subtitulo"] = subtitulo
        if previsao_entrega:
            data["previsao_entrega"] = previsao_entrega
        if mensagem:
            data["mensagem"] = mensagem
        return self.render(tenant_id, "confirmacao", data, channel=channel)

    def render_pix_payment(
        self,
        tenant_id: str,
        valor: float,
        pix_payload: str,
        numero_pedido: Optional[str] = None,
        expiracao: Optional[str] = None,
        mensagem: Optional[str] = None,
        channel: str = "whatsapp",
    ) -> VRERenderResponse:
        """
        Render a PIX payment QR code.

        Args:
            tenant_id: Tenant ID
            valor: Payment amount
            pix_payload: PIX EMV/BRCode payload
            numero_pedido: Order number
            expiracao: Expiration time
            mensagem: Optional message
            channel: Target channel

        Returns:
            VRERenderResponse
        """
        data: dict[str, Any] = {"valor": valor, "pix_payload": pix_payload}
        if numero_pedido:
            data["numero_pedido"] = numero_pedido
        if expiracao:
            data["expiracao"] = expiracao
        if mensagem:
            data["mensagem"] = mensagem
        return self.render(tenant_id, "cobranca_pix", data, channel=channel)


class LinktorClient:
    """Linktor SDK Client (Synchronous)"""

    def __init__(
        self,
        base_url: str = "https://api.linktor.io",
        api_key: Optional[str] = None,
        access_token: Optional[str] = None,
        timeout: float = 30.0,
        max_retries: int = 3,
        retry_delay: float = 1.0,
        headers: Optional[dict[str, str]] = None,
        on_token_refresh: Optional[Callable[[], str]] = None,
    ):
        self._http = HttpClient(
            base_url=base_url,
            api_key=api_key,
            access_token=access_token,
            timeout=timeout,
            max_retries=max_retries,
            retry_delay=retry_delay,
            headers=headers,
            on_token_refresh=on_token_refresh,
        )

        self.auth = AuthResource(self._http)
        self.conversations = ConversationsResource(self._http)
        self.contacts = ContactsResource(self._http)
        self.channels = ChannelsResource(self._http)
        self.bots = BotsResource(self._http)
        self.ai = AIResource(self._http)
        self.knowledge_bases = KnowledgeBasesResource(self._http)
        self.flows = FlowsResource(self._http)
        self.analytics = AnalyticsResource(self._http)
        self.vre = VREResource(self._http)

    def set_api_key(self, api_key: str) -> None:
        """Set API key"""
        self._http.set_api_key(api_key)

    def set_access_token(self, access_token: str) -> None:
        """Set access token"""
        self._http.set_access_token(access_token)

    def close(self) -> None:
        """Close client"""
        self._http.close()

    def __enter__(self) -> "LinktorClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


class LinktorAsyncClient:
    """Linktor SDK Client (Asynchronous)"""

    def __init__(
        self,
        base_url: str = "https://api.linktor.io",
        api_key: Optional[str] = None,
        access_token: Optional[str] = None,
        timeout: float = 30.0,
        max_retries: int = 3,
        retry_delay: float = 1.0,
        headers: Optional[dict[str, str]] = None,
    ):
        self._http = AsyncHttpClient(
            base_url=base_url,
            api_key=api_key,
            access_token=access_token,
            timeout=timeout,
            max_retries=max_retries,
            retry_delay=retry_delay,
            headers=headers,
        )
        # Async resources would be similar but use async methods
        # For brevity, omitting full async implementation

    async def close(self) -> None:
        """Close client"""
        await self._http.close()

    async def __aenter__(self) -> "LinktorAsyncClient":
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.close()


__all__ = ["LinktorClient", "LinktorAsyncClient"]
