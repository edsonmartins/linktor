package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Client is the HTTP client for Linktor API
type Client struct {
	baseURL     string
	apiKey      string
	accessToken string
	httpClient  *http.Client
}

// User represents a Linktor user
type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	TenantID   string `json:"tenantId"`
	TenantName string `json:"tenantName"`
}

// LoginResponse represents the login API response
type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	User         *User  `json:"user"`
}

// APIKey represents an API key
type APIKey struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsed  time.Time `json:"lastUsedAt"`
}

// Channel represents a messaging channel
type Channel struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// Conversation represents a conversation
type Conversation struct {
	ID           string    `json:"id"`
	ChannelID    string    `json:"channelId"`
	ContactID    string    `json:"contactId"`
	ContactName  string    `json:"contactName"`
	Status       string    `json:"status"`
	AssignedTo   string    `json:"assignedTo"`
	MessageCount int       `json:"messageCount"`
	LastMessage  *Message  `json:"lastMessage"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Message represents a message
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	Direction      string    `json:"direction"`
	ContentType    string    `json:"contentType"`
	Text           string    `json:"text"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Contact represents a contact
type Contact struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Email        string                 `json:"email"`
	Phone        string                 `json:"phone"`
	Tags         []string               `json:"tags"`
	CustomFields map[string]interface{} `json:"customFields"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// Bot represents a bot
type Bot struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	AgentID             string    `json:"agentId"`
	Status              string    `json:"status"`
	ChannelIDs          []string  `json:"channelIds"`
	Enabled             bool      `json:"enabled"`
	ActiveConversations int       `json:"activeConversations"`
	CreatedAt           time.Time `json:"createdAt"`
}

// BotLog represents a bot log entry
type BotLog struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Data      map[string]interface{} `json:"data"`
}

// Flow represents a flow
type Flow struct {
	ID             string                   `json:"id"`
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Status         string                   `json:"status"`
	Nodes          []map[string]interface{} `json:"nodes"`
	Triggers       []string                 `json:"triggers"`
	ExecutionCount int                      `json:"executionCount"`
	CreatedAt      time.Time                `json:"createdAt"`
	UpdatedAt      time.Time                `json:"updatedAt"`
	PublishedAt    time.Time                `json:"publishedAt"`
}

// FlowExecutionResult represents the result of a flow execution
type FlowExecutionResult struct {
	ExecutionID string `json:"executionId"`
	Status      string `json:"status"`
}

// FlowValidationResult represents the result of flow validation
type FlowValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// KnowledgeBase represents a knowledge base
type KnowledgeBase struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	DocumentCount int       `json:"documentCount"`
	ChunkCount    int       `json:"chunkCount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Document represents a document in a knowledge base
type Document struct {
	ID              string    `json:"id"`
	KnowledgeBaseID string    `json:"knowledgeBaseId"`
	Title           string    `json:"title"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	ChunkCount      int       `json:"chunkCount"`
	CreatedAt       time.Time `json:"createdAt"`
}

// QueryResult represents a knowledge base query result
type QueryResult struct {
	Score   float64 `json:"score"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	DocID   string  `json:"documentId"`
}

// Webhook represents a webhook configuration
type Webhook struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Enabled   bool      `json:"enabled"`
	Secret    string    `json:"secret"`
	CreatedAt time.Time `json:"createdAt"`
}

// WebhookEvent represents a webhook delivery event
type WebhookEvent struct {
	ID         string    `json:"id"`
	WebhookID  string    `json:"webhookId"`
	EventType  string    `json:"eventType"`
	StatusCode int       `json:"statusCode"`
	Duration   int       `json:"duration"`
	Timestamp  time.Time `json:"timestamp"`
}

// PaginatedResponse wraps paginated API responses
type PaginatedResponse[T any] struct {
	Data       []T    `json:"data"`
	Total      int    `json:"total"`
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// New creates a new client using stored credentials
func New() (*Client, error) {
	c := &Client{
		baseURL:    viper.GetString("base_url"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Try API key first
	if apiKey := viper.GetString("api_key"); apiKey != "" {
		c.apiKey = apiKey
		return c, nil
	}

	// Try stored credentials
	creds, err := loadCredentials()
	if err == nil {
		if token, ok := creds["access_token"]; ok {
			c.accessToken = token
			return c, nil
		}
		if apiKey, ok := creds["api_key"]; ok {
			c.apiKey = apiKey
			return c, nil
		}
	}

	return nil, fmt.Errorf("not authenticated. Run: msgfy auth login")
}

// NewWithAPIKey creates a new client with the given API key
func NewWithAPIKey(apiKey string) (*Client, error) {
	return &Client{
		baseURL:    viper.GetString("base_url"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// NewAnonymous creates a new client without authentication (for login)
func NewAnonymous() (*Client, error) {
	return &Client{
		baseURL:    viper.GetString("base_url"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func loadCredentials() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	credPath := filepath.Join(home, ".msgfy", "credentials")
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, err
	}

	var creds map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return creds, nil
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	} else if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	return c.httpClient.Do(req)
}

func (c *Client) get(path string, result interface{}) error {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) post(path string, body, result interface{}) error {
	resp, err := c.doRequest("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) patch(path string, body, result interface{}) error {
	resp, err := c.doRequest("PATCH", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) delete(path string) error {
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}

	return nil
}

func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}

	if result == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Try to parse as API response wrapper
	var wrapper struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Success && wrapper.Data != nil {
		return json.Unmarshal(wrapper.Data, result)
	}

	return json.Unmarshal(body, result)
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return fmt.Errorf("%s", apiErr.Message)
	}

	return fmt.Errorf("API error: %d", resp.StatusCode)
}

// Auth methods

func (c *Client) Login(email, password string) (*LoginResponse, error) {
	var result LoginResponse
	err := c.post("/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &result)
	return &result, err
}

func (c *Client) GetCurrentUser() (*User, error) {
	var result User
	err := c.get("/auth/me", &result)
	return &result, err
}

func (c *Client) ListAPIKeys() ([]APIKey, error) {
	var result []APIKey
	err := c.get("/auth/api-keys", &result)
	return result, err
}

// Channel methods

func (c *Client) ListChannels(params map[string]string) (*PaginatedResponse[Channel], error) {
	path := "/channels"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Channel]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) GetChannel(id string) (*Channel, error) {
	var result Channel
	err := c.get("/channels/"+id, &result)
	return &result, err
}

func (c *Client) CreateChannel(input map[string]interface{}) (*Channel, error) {
	var result Channel
	err := c.post("/channels", input, &result)
	return &result, err
}

func (c *Client) UpdateChannel(id string, input map[string]interface{}) (*Channel, error) {
	var result Channel
	err := c.patch("/channels/"+id, input, &result)
	return &result, err
}

func (c *Client) DeleteChannel(id string) error {
	return c.delete("/channels/" + id)
}

func (c *Client) ConnectChannel(id string) (*Channel, error) {
	var result Channel
	err := c.post("/channels/"+id+"/connect", nil, &result)
	return &result, err
}

func (c *Client) DisconnectChannel(id string) (*Channel, error) {
	var result Channel
	err := c.post("/channels/"+id+"/disconnect", nil, &result)
	return &result, err
}

// Conversation methods

func (c *Client) ListConversations(params map[string]string) (*PaginatedResponse[Conversation], error) {
	path := "/conversations"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Conversation]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) GetConversation(id string) (*Conversation, error) {
	var result Conversation
	err := c.get("/conversations/"+id, &result)
	return &result, err
}

func (c *Client) GetConversationMessages(id string, params map[string]string) (*PaginatedResponse[Message], error) {
	path := "/conversations/" + id + "/messages"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Message]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) SendMessage(conversationID string, input map[string]interface{}) (*Message, error) {
	var result Message
	err := c.post("/conversations/"+conversationID+"/messages", input, &result)
	return &result, err
}

func (c *Client) CloseConversation(id string) (*Conversation, error) {
	var result Conversation
	err := c.post("/conversations/"+id+"/resolve", nil, &result)
	return &result, err
}

func (c *Client) ReopenConversation(id string) (*Conversation, error) {
	var result Conversation
	err := c.post("/conversations/"+id+"/reopen", nil, &result)
	return &result, err
}

// Contact methods

func (c *Client) ListContacts(params map[string]string) (*PaginatedResponse[Contact], error) {
	path := "/contacts"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Contact]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) GetContact(id string) (*Contact, error) {
	var result Contact
	err := c.get("/contacts/"+id, &result)
	return &result, err
}

func (c *Client) CreateContact(input map[string]interface{}) (*Contact, error) {
	var result Contact
	err := c.post("/contacts", input, &result)
	return &result, err
}

func (c *Client) UpdateContact(id string, input map[string]interface{}) (*Contact, error) {
	var result Contact
	err := c.patch("/contacts/"+id, input, &result)
	return &result, err
}

func (c *Client) DeleteContact(id string) error {
	return c.delete("/contacts/" + id)
}

// Bot methods

func (c *Client) ListBots(params map[string]string) (*PaginatedResponse[Bot], error) {
	path := "/bots"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Bot]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) GetBot(id string) (*Bot, error) {
	var result Bot
	err := c.get("/bots/"+id, &result)
	return &result, err
}

func (c *Client) CreateBot(input map[string]interface{}) (*Bot, error) {
	var result Bot
	err := c.post("/bots", input, &result)
	return &result, err
}

func (c *Client) UpdateBot(id string, input map[string]interface{}) (*Bot, error) {
	var result Bot
	err := c.patch("/bots/"+id, input, &result)
	return &result, err
}

func (c *Client) DeleteBot(id string) error {
	return c.delete("/bots/" + id)
}

func (c *Client) StartBot(id string) (*Bot, error) {
	var result Bot
	err := c.post("/bots/"+id+"/start", nil, &result)
	return &result, err
}

func (c *Client) StopBot(id string) (*Bot, error) {
	var result Bot
	err := c.post("/bots/"+id+"/stop", nil, &result)
	return &result, err
}

func (c *Client) GetBotLogs(id string, params map[string]string) ([]BotLog, error) {
	path := "/bots/" + id + "/logs"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result []BotLog
	err := c.get(path, &result)
	return result, err
}

// Flow methods

func (c *Client) ListFlows(params map[string]string) (*PaginatedResponse[Flow], error) {
	path := "/flows"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Flow]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) GetFlow(id string) (*Flow, error) {
	var result Flow
	err := c.get("/flows/"+id, &result)
	return &result, err
}

func (c *Client) CreateFlow(input map[string]interface{}) (*Flow, error) {
	var result Flow
	err := c.post("/flows", input, &result)
	return &result, err
}

func (c *Client) DeleteFlow(id string) error {
	return c.delete("/flows/" + id)
}

func (c *Client) PublishFlow(id string) (*Flow, error) {
	var result Flow
	err := c.post("/flows/"+id+"/publish", nil, &result)
	return &result, err
}

func (c *Client) UnpublishFlow(id string) (*Flow, error) {
	var result Flow
	err := c.post("/flows/"+id+"/unpublish", nil, &result)
	return &result, err
}

func (c *Client) ExecuteFlow(id string, input map[string]interface{}) (*FlowExecutionResult, error) {
	var result FlowExecutionResult
	err := c.post("/flows/"+id+"/execute", input, &result)
	return &result, err
}

func (c *Client) ValidateFlow(id string) (*FlowValidationResult, error) {
	var result FlowValidationResult
	err := c.post("/flows/"+id+"/validate", nil, &result)
	return &result, err
}

// Knowledge Base methods

func (c *Client) ListKnowledgeBases(params map[string]string) (*PaginatedResponse[KnowledgeBase], error) {
	path := "/knowledge-bases"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[KnowledgeBase]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) GetKnowledgeBase(id string) (*KnowledgeBase, error) {
	var result KnowledgeBase
	err := c.get("/knowledge-bases/"+id, &result)
	return &result, err
}

func (c *Client) CreateKnowledgeBase(input map[string]interface{}) (*KnowledgeBase, error) {
	var result KnowledgeBase
	err := c.post("/knowledge-bases", input, &result)
	return &result, err
}

func (c *Client) ListDocuments(kbID string, params map[string]string) (*PaginatedResponse[Document], error) {
	path := "/knowledge-bases/" + kbID + "/documents"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Document]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) DeleteKnowledgeBase(id string) error {
	return c.delete("/knowledge-bases/" + id)
}

func (c *Client) QueryKnowledgeBase(kbID string, input map[string]interface{}) ([]QueryResult, error) {
	var result struct {
		Results []QueryResult `json:"results"`
	}
	err := c.post("/knowledge-bases/"+kbID+"/query", input, &result)
	return result.Results, err
}

func (c *Client) GetDocument(kbID, docID string) (*Document, error) {
	var result Document
	err := c.get("/knowledge-bases/"+kbID+"/documents/"+docID, &result)
	return &result, err
}

func (c *Client) AddDocument(kbID string, input map[string]interface{}) (*Document, error) {
	var result Document
	err := c.post("/knowledge-bases/"+kbID+"/documents", input, &result)
	return &result, err
}

func (c *Client) UploadDocument(kbID, filePath string, input map[string]interface{}) (*Document, error) {
	// For file upload, we need multipart form data
	// This is a simplified version - actual implementation would use multipart
	input["filePath"] = filePath
	var result Document
	err := c.post("/knowledge-bases/"+kbID+"/documents/upload", input, &result)
	return &result, err
}

func (c *Client) DeleteDocument(kbID, docID string) error {
	return c.delete("/knowledge-bases/" + kbID + "/documents/" + docID)
}

func (c *Client) ReprocessDocument(kbID, docID string) (*Document, error) {
	var result Document
	err := c.post("/knowledge-bases/"+kbID+"/documents/"+docID+"/reprocess", nil, &result)
	return &result, err
}

// Webhook methods

func (c *Client) ListWebhooks(params map[string]string) (*PaginatedResponse[Webhook], error) {
	path := "/webhooks"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result PaginatedResponse[Webhook]
	err := c.get(path, &result)
	return &result, err
}

func (c *Client) ListWebhookEvents(params map[string]string) ([]WebhookEvent, error) {
	path := "/webhooks/events"
	if len(params) > 0 {
		path += "?" + buildQuery(params)
	}
	var result []WebhookEvent
	err := c.get(path, &result)
	return result, err
}

// Helper function to build query string
func buildQuery(params map[string]string) string {
	var parts []string
	for k, v := range params {
		if v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	return joinStrings(parts, "&")
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
