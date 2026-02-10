// Package linktor provides the official Go SDK for the Linktor API.
package linktor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/linktor/linktor-go/types"
)

// ClientConfig holds the configuration for the Linktor client
type ClientConfig struct {
	BaseURL     string
	APIKey      string
	AccessToken string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	HTTPClient  *http.Client
}

// Option is a function that configures the client
type Option func(*ClientConfig)

// WithBaseURL sets the base URL
func WithBaseURL(url string) Option {
	return func(c *ClientConfig) {
		c.BaseURL = url
	}
}

// WithAPIKey sets the API key
func WithAPIKey(key string) Option {
	return func(c *ClientConfig) {
		c.APIKey = key
	}
}

// WithAccessToken sets the access token
func WithAccessToken(token string) Option {
	return func(c *ClientConfig) {
		c.AccessToken = token
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *ClientConfig) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the max retries
func WithMaxRetries(retries int) Option {
	return func(c *ClientConfig) {
		c.MaxRetries = retries
	}
}

// Client is the main Linktor SDK client
type Client struct {
	config *ClientConfig
	http   *http.Client

	// Resources
	Auth          *AuthResource
	Conversations *ConversationsResource
	Contacts      *ContactsResource
	Channels      *ChannelsResource
	Bots          *BotsResource
	AI            *AIResource
	KnowledgeBases *KnowledgeBasesResource
	Flows         *FlowsResource
	Analytics     *AnalyticsResource
	VRE           *VREResource
}

// NewClient creates a new Linktor client
func NewClient(opts ...Option) *Client {
	config := &ClientConfig{
		BaseURL:    "https://api.linktor.io",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}

	for _, opt := range opts {
		opt(config)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	client := &Client{
		config: config,
		http:   httpClient,
	}

	// Initialize resources
	client.Auth = &AuthResource{client: client}
	client.Conversations = &ConversationsResource{client: client}
	client.Contacts = &ContactsResource{client: client}
	client.Channels = &ChannelsResource{client: client}
	client.Bots = &BotsResource{client: client}
	client.AI = &AIResource{client: client}
	client.KnowledgeBases = &KnowledgeBasesResource{client: client}
	client.Flows = &FlowsResource{client: client}
	client.Analytics = &AnalyticsResource{client: client}
	client.VRE = &VREResource{client: client}

	return client
}

// SetAPIKey updates the API key
func (c *Client) SetAPIKey(key string) {
	c.config.APIKey = key
}

// SetAccessToken updates the access token
func (c *Client) SetAccessToken(token string) {
	c.config.AccessToken = token
}

// Error represents a Linktor API error
type Error struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Status    int                    `json:"status"`
	RequestID string                 `json:"requestId,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// request makes an HTTP request
func (c *Client) request(ctx context.Context, method, path string, body, result interface{}) error {
	u, err := url.Parse(c.config.BaseURL + path)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.config.APIKey != "" {
		req.Header.Set("X-API-Key", c.config.APIKey)
	} else if c.config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	}

	// Execute with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.config.MaxRetries {
				time.Sleep(c.config.RetryDelay * time.Duration(1<<attempt))
				continue
			}
			return err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode >= 400 {
			var apiErr Error
			if err := json.Unmarshal(respBody, &apiErr); err != nil {
				apiErr = Error{
					Code:    "UNKNOWN_ERROR",
					Message: string(respBody),
					Status:  resp.StatusCode,
				}
			}
			apiErr.Status = resp.StatusCode
			apiErr.RequestID = resp.Header.Get("X-Request-ID")

			// Retry on 5xx errors
			if resp.StatusCode >= 500 && attempt < c.config.MaxRetries {
				lastErr = &apiErr
				time.Sleep(c.config.RetryDelay * time.Duration(1<<attempt))
				continue
			}

			return &apiErr
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return err
			}
		}

		return nil
	}

	return lastErr
}

// get makes a GET request
func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	return c.request(ctx, http.MethodGet, path, nil, result)
}

// post makes a POST request
func (c *Client) post(ctx context.Context, path string, body, result interface{}) error {
	return c.request(ctx, http.MethodPost, path, body, result)
}

// patch makes a PATCH request
func (c *Client) patch(ctx context.Context, path string, body, result interface{}) error {
	return c.request(ctx, http.MethodPatch, path, body, result)
}

// delete makes a DELETE request
func (c *Client) delete(ctx context.Context, path string) error {
	return c.request(ctx, http.MethodDelete, path, nil, nil)
}

// AuthResource handles authentication
type AuthResource struct {
	client *Client
}

// Login authenticates a user
func (r *AuthResource) Login(ctx context.Context, email, password string) (*types.LoginResponse, error) {
	var result types.LoginResponse
	err := r.client.post(ctx, "/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &result)
	if err != nil {
		return nil, err
	}
	r.client.SetAccessToken(result.AccessToken)
	return &result, nil
}

// Logout logs out the current user
func (r *AuthResource) Logout(ctx context.Context) error {
	return r.client.post(ctx, "/auth/logout", nil, nil)
}

// GetCurrentUser gets the current user
func (r *AuthResource) GetCurrentUser(ctx context.Context) (*types.User, error) {
	var result types.User
	err := r.client.get(ctx, "/auth/me", &result)
	return &result, err
}

// ConversationsResource handles conversations
type ConversationsResource struct {
	client *Client
}

// List lists conversations
func (r *ConversationsResource) List(ctx context.Context, params *types.ListConversationsParams) (*types.PaginatedResponse[types.Conversation], error) {
	var result types.PaginatedResponse[types.Conversation]
	err := r.client.get(ctx, "/conversations", &result)
	return &result, err
}

// Get gets a conversation
func (r *ConversationsResource) Get(ctx context.Context, id string) (*types.Conversation, error) {
	var result types.Conversation
	err := r.client.get(ctx, "/conversations/"+id, &result)
	return &result, err
}

// Update updates a conversation
func (r *ConversationsResource) Update(ctx context.Context, id string, input *types.UpdateConversationInput) (*types.Conversation, error) {
	var result types.Conversation
	err := r.client.patch(ctx, "/conversations/"+id, input, &result)
	return &result, err
}

// SendMessage sends a message
func (r *ConversationsResource) SendMessage(ctx context.Context, conversationID string, input *types.SendMessageInput) (*types.Message, error) {
	var result types.Message
	err := r.client.post(ctx, "/conversations/"+conversationID+"/messages", input, &result)
	return &result, err
}

// SendText sends a text message
func (r *ConversationsResource) SendText(ctx context.Context, conversationID, text string) (*types.Message, error) {
	return r.SendMessage(ctx, conversationID, &types.SendMessageInput{Text: text})
}

// Resolve resolves a conversation
func (r *ConversationsResource) Resolve(ctx context.Context, id string) (*types.Conversation, error) {
	return r.Update(ctx, id, &types.UpdateConversationInput{Status: types.ConversationStatusResolved})
}

// Assign assigns a conversation
func (r *ConversationsResource) Assign(ctx context.Context, id, agentID string) (*types.Conversation, error) {
	var result types.Conversation
	err := r.client.post(ctx, "/conversations/"+id+"/assign", map[string]string{"agentId": agentID}, &result)
	return &result, err
}

// ContactsResource handles contacts
type ContactsResource struct {
	client *Client
}

// List lists contacts
func (r *ContactsResource) List(ctx context.Context, params *types.ListContactsParams) (*types.PaginatedResponse[types.Contact], error) {
	var result types.PaginatedResponse[types.Contact]
	err := r.client.get(ctx, "/contacts", &result)
	return &result, err
}

// Get gets a contact
func (r *ContactsResource) Get(ctx context.Context, id string) (*types.Contact, error) {
	var result types.Contact
	err := r.client.get(ctx, "/contacts/"+id, &result)
	return &result, err
}

// Create creates a contact
func (r *ContactsResource) Create(ctx context.Context, input *types.CreateContactInput) (*types.Contact, error) {
	var result types.Contact
	err := r.client.post(ctx, "/contacts", input, &result)
	return &result, err
}

// Update updates a contact
func (r *ContactsResource) Update(ctx context.Context, id string, input *types.UpdateContactInput) (*types.Contact, error) {
	var result types.Contact
	err := r.client.patch(ctx, "/contacts/"+id, input, &result)
	return &result, err
}

// Delete deletes a contact
func (r *ContactsResource) Delete(ctx context.Context, id string) error {
	return r.client.delete(ctx, "/contacts/"+id)
}

// ChannelsResource handles channels
type ChannelsResource struct {
	client *Client
}

// List lists channels
func (r *ChannelsResource) List(ctx context.Context, params *types.ListChannelsParams) (*types.PaginatedResponse[types.Channel], error) {
	var result types.PaginatedResponse[types.Channel]
	err := r.client.get(ctx, "/channels", &result)
	return &result, err
}

// Get gets a channel
func (r *ChannelsResource) Get(ctx context.Context, id string) (*types.Channel, error) {
	var result types.Channel
	err := r.client.get(ctx, "/channels/"+id, &result)
	return &result, err
}

// Create creates a channel
func (r *ChannelsResource) Create(ctx context.Context, input *types.CreateChannelInput) (*types.Channel, error) {
	var result types.Channel
	err := r.client.post(ctx, "/channels", input, &result)
	return &result, err
}

// Connect connects a channel
func (r *ChannelsResource) Connect(ctx context.Context, id string) (*types.Channel, error) {
	var result types.Channel
	err := r.client.post(ctx, "/channels/"+id+"/connect", nil, &result)
	return &result, err
}

// Disconnect disconnects a channel
func (r *ChannelsResource) Disconnect(ctx context.Context, id string) (*types.Channel, error) {
	var result types.Channel
	err := r.client.post(ctx, "/channels/"+id+"/disconnect", nil, &result)
	return &result, err
}

// BotsResource handles bots
type BotsResource struct {
	client *Client
}

// List lists bots
func (r *BotsResource) List(ctx context.Context, params *types.ListBotsParams) (*types.PaginatedResponse[types.Bot], error) {
	var result types.PaginatedResponse[types.Bot]
	err := r.client.get(ctx, "/bots", &result)
	return &result, err
}

// Get gets a bot
func (r *BotsResource) Get(ctx context.Context, id string) (*types.Bot, error) {
	var result types.Bot
	err := r.client.get(ctx, "/bots/"+id, &result)
	return &result, err
}

// Create creates a bot
func (r *BotsResource) Create(ctx context.Context, input *types.CreateBotInput) (*types.Bot, error) {
	var result types.Bot
	err := r.client.post(ctx, "/bots", input, &result)
	return &result, err
}

// AIResource handles AI operations
type AIResource struct {
	client *Client
	Agents      *AgentsSubResource
	Completions *CompletionsSubResource
	Embeddings  *EmbeddingsSubResource
}

// AgentsSubResource handles AI agents
type AgentsSubResource struct {
	client *Client
}

// CompletionsSubResource handles completions
type CompletionsSubResource struct {
	client *Client
}

// Complete creates a simple completion
func (r *CompletionsSubResource) Complete(ctx context.Context, prompt string) (string, error) {
	var result map[string]interface{}
	err := r.client.post(ctx, "/ai/completions", map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}, &result)
	if err != nil {
		return "", err
	}
	if msg, ok := result["message"].(map[string]interface{}); ok {
		if content, ok := msg["content"].(string); ok {
			return content, nil
		}
	}
	return "", fmt.Errorf("unexpected response format")
}

// EmbeddingsSubResource handles embeddings
type EmbeddingsSubResource struct {
	client *Client
}

// Embed creates an embedding
func (r *EmbeddingsSubResource) Embed(ctx context.Context, text string) ([]float64, error) {
	var result map[string]interface{}
	err := r.client.post(ctx, "/ai/embeddings", map[string]interface{}{
		"input": text,
	}, &result)
	if err != nil {
		return nil, err
	}
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if item, ok := data[0].(map[string]interface{}); ok {
			if emb, ok := item["embedding"].([]interface{}); ok {
				embedding := make([]float64, len(emb))
				for i, v := range emb {
					if f, ok := v.(float64); ok {
						embedding[i] = f
					}
				}
				return embedding, nil
			}
		}
	}
	return nil, fmt.Errorf("unexpected response format")
}

// KnowledgeBasesResource handles knowledge bases
type KnowledgeBasesResource struct {
	client *Client
}

// List lists knowledge bases
func (r *KnowledgeBasesResource) List(ctx context.Context) (*types.PaginatedResponse[types.KnowledgeBase], error) {
	var result types.PaginatedResponse[types.KnowledgeBase]
	err := r.client.get(ctx, "/knowledge-bases", &result)
	return &result, err
}

// Get gets a knowledge base
func (r *KnowledgeBasesResource) Get(ctx context.Context, id string) (*types.KnowledgeBase, error) {
	var result types.KnowledgeBase
	err := r.client.get(ctx, "/knowledge-bases/"+id, &result)
	return &result, err
}

// Query queries a knowledge base
func (r *KnowledgeBasesResource) Query(ctx context.Context, id, query string, topK int) (*types.QueryResult, error) {
	var result types.QueryResult
	err := r.client.post(ctx, "/knowledge-bases/"+id+"/query", map[string]interface{}{
		"query": query,
		"topK":  topK,
	}, &result)
	return &result, err
}

// FlowsResource handles flows
type FlowsResource struct {
	client *Client
}

// List lists flows
func (r *FlowsResource) List(ctx context.Context) (*types.PaginatedResponse[types.Flow], error) {
	var result types.PaginatedResponse[types.Flow]
	err := r.client.get(ctx, "/flows", &result)
	return &result, err
}

// Get gets a flow
func (r *FlowsResource) Get(ctx context.Context, id string) (*types.Flow, error) {
	var result types.Flow
	err := r.client.get(ctx, "/flows/"+id, &result)
	return &result, err
}

// Execute executes a flow
func (r *FlowsResource) Execute(ctx context.Context, id, conversationID string) (*types.FlowExecution, error) {
	var result types.FlowExecution
	err := r.client.post(ctx, "/flows/"+id+"/execute", map[string]string{
		"conversationId": conversationID,
	}, &result)
	return &result, err
}

// AnalyticsResource handles analytics
type AnalyticsResource struct {
	client *Client
}

// GetDashboard gets dashboard metrics
func (r *AnalyticsResource) GetDashboard(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := r.client.get(ctx, "/analytics/dashboard", &result)
	return result, err
}

// GetRealtime gets realtime metrics
func (r *AnalyticsResource) GetRealtime(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := r.client.get(ctx, "/analytics/realtime", &result)
	return result, err
}

// VREResource handles VRE (Visual Response Engine) operations
type VREResource struct {
	client *Client
}

// Render renders a VRE template to an image
func (r *VREResource) Render(ctx context.Context, input *types.VRERenderRequest) (*types.VRERenderResponse, error) {
	var result types.VRERenderResponse
	err := r.client.post(ctx, "/vre/render", input, &result)
	return &result, err
}

// RenderAndSend renders a template and sends it directly to a conversation
func (r *VREResource) RenderAndSend(ctx context.Context, input *types.VRERenderAndSendRequest) (*types.VRERenderAndSendResponse, error) {
	var result types.VRERenderAndSendResponse
	err := r.client.post(ctx, "/vre/render-and-send", input, &result)
	return &result, err
}

// ListTemplates lists available VRE templates
func (r *VREResource) ListTemplates(ctx context.Context, tenantID string) (*types.VREListTemplatesResponse, error) {
	path := "/vre/templates"
	if tenantID != "" {
		path += "?tenant_id=" + tenantID
	}
	var result types.VREListTemplatesResponse
	err := r.client.get(ctx, path, &result)
	return &result, err
}

// Preview previews a VRE template with sample data
func (r *VREResource) Preview(ctx context.Context, templateID string, data map[string]interface{}) (*types.VREPreviewResponse, error) {
	var result types.VREPreviewResponse
	body := map[string]interface{}{}
	if data != nil {
		body["data"] = data
	}
	err := r.client.post(ctx, "/vre/templates/"+templateID+"/preview", body, &result)
	return &result, err
}

// RenderMenu renders a menu with numbered options
func (r *VREResource) RenderMenu(ctx context.Context, tenantID, titulo string, opcoes []types.MenuOpcaoData, channel types.VREChannelType) (*types.VRERenderResponse, error) {
	return r.Render(ctx, &types.VRERenderRequest{
		TenantID:   tenantID,
		TemplateID: types.VRETemplateMenuOpcoes,
		Data: map[string]interface{}{
			"titulo": titulo,
			"opcoes": opcoes,
		},
		Channel: channel,
	})
}

// RenderProductCard renders a product card
func (r *VREResource) RenderProductCard(ctx context.Context, tenantID string, produto *types.CardProdutoData, channel types.VREChannelType) (*types.VRERenderResponse, error) {
	return r.Render(ctx, &types.VRERenderRequest{
		TenantID:   tenantID,
		TemplateID: types.VRETemplateCardProduto,
		Data: map[string]interface{}{
			"nome":       produto.Nome,
			"preco":      produto.Preco,
			"unidade":    produto.Unidade,
			"sku":        produto.SKU,
			"estoque":    produto.Estoque,
			"imagem_url": produto.ImagemURL,
			"destaque":   produto.Destaque,
			"mensagem":   produto.Mensagem,
		},
		Channel: channel,
	})
}

// RenderOrderStatus renders an order status timeline
func (r *VREResource) RenderOrderStatus(ctx context.Context, tenantID string, status *types.StatusPedidoData, channel types.VREChannelType) (*types.VRERenderResponse, error) {
	return r.Render(ctx, &types.VRERenderRequest{
		TenantID:   tenantID,
		TemplateID: types.VRETemplateStatusPedido,
		Data: map[string]interface{}{
			"numero_pedido":    status.NumeroPedido,
			"status_atual":     status.StatusAtual,
			"itens_resumo":     status.ItensResumo,
			"valor_total":      status.ValorTotal,
			"previsao_entrega": status.PrevisaoEntrega,
			"motorista":        status.Motorista,
			"mensagem":         status.Mensagem,
		},
		Channel: channel,
	})
}

// RenderProductList renders a product list for comparison
func (r *VREResource) RenderProductList(ctx context.Context, tenantID, titulo string, produtos []types.ListaProdutoItem, channel types.VREChannelType) (*types.VRERenderResponse, error) {
	return r.Render(ctx, &types.VRERenderRequest{
		TenantID:   tenantID,
		TemplateID: types.VRETemplateListaProdutos,
		Data: map[string]interface{}{
			"titulo":   titulo,
			"produtos": produtos,
		},
		Channel: channel,
	})
}

// RenderConfirmation renders a confirmation summary
func (r *VREResource) RenderConfirmation(ctx context.Context, tenantID string, confirmacao *types.ConfirmacaoData, channel types.VREChannelType) (*types.VRERenderResponse, error) {
	return r.Render(ctx, &types.VRERenderRequest{
		TenantID:   tenantID,
		TemplateID: types.VRETemplateConfirmacao,
		Data: map[string]interface{}{
			"titulo":           confirmacao.Titulo,
			"subtitulo":        confirmacao.Subtitulo,
			"itens":            confirmacao.Itens,
			"valor_total":      confirmacao.ValorTotal,
			"previsao_entrega": confirmacao.PrevisaoEntrega,
			"mensagem":         confirmacao.Mensagem,
		},
		Channel: channel,
	})
}

// RenderPixPayment renders a PIX payment QR code
func (r *VREResource) RenderPixPayment(ctx context.Context, tenantID string, pix *types.CobrancaPixData, channel types.VREChannelType) (*types.VRERenderResponse, error) {
	return r.Render(ctx, &types.VRERenderRequest{
		TenantID:   tenantID,
		TemplateID: types.VRETemplateCobrancaPix,
		Data: map[string]interface{}{
			"valor":         pix.Valor,
			"pix_payload":   pix.PixPayload,
			"numero_pedido": pix.NumeroPedido,
			"expiracao":     pix.Expiracao,
			"mensagem":      pix.Mensagem,
		},
		Channel: channel,
	})
}
