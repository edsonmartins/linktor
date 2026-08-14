package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/adapters/email"
	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/mattermost"
	"github.com/msgfy/linktor/internal/adapters/rcs"
	"github.com/msgfy/linktor/internal/adapters/slack"
	"github.com/msgfy/linktor/internal/adapters/sms"
	"github.com/msgfy/linktor/internal/adapters/teams"
	"github.com/msgfy/linktor/internal/adapters/telegram"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// ChannelHandler handles channel endpoints
type ChannelHandler struct {
	channelService *service.ChannelService
	producer       nats.Publisher
	audit          *service.AuditService
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(channelService *service.ChannelService, producer nats.Publisher, audit *service.AuditService) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		producer:       producer,
		audit:          audit,
	}
}

// channelAuditChanges collects the audit-relevant, NON-SECRET declarations of
// a channel request (INV-023): environment defines the sandbox boundary,
// credential_environment is the declared credential binding (a declaration,
// not a credential value) and phone_number_id is deliberately non-secret
// config. Credential VALUES never enter the trail (INV-002).
func channelAuditChanges(channelType, environment string, config, credentials map[string]string) map[string]interface{} {
	changes := map[string]interface{}{
		"type":        channelType,
		"environment": environment,
	}
	if v := credentials[service.CredentialEnvironmentKey]; v != "" {
		changes["credential_environment"] = v
	}
	if channelType == string(entity.ChannelTypeWhatsAppOfficial) {
		if v := config["phone_number_id"]; v != "" {
			changes["phone_number_id"] = v
		}
	}
	return changes
}

// CreateChannelRequest represents a create channel request
type CreateChannelRequest struct {
	Type       string `json:"type" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Identifier string `json:"identifier"`
	// Environment is "production" (default when omitted) or "sandbox" and is
	// immutable after creation. Sandbox channels additionally require
	// credentials["credential_environment"]="sandbox" (see ChannelService).
	Environment string            `json:"environment"`
	Config      map[string]string `json:"config"`
	Credentials map[string]string `json:"credentials"`
	// WebhookURL is the external consumer endpoint (e.g. DeskLenz) Linktor
	// delivers signed inbound/status events to for this channel.
	WebhookURL string `json:"webhook_url"`
}

// UpdateChannelRequest represents a partial channel update. Every field is a
// pointer/map so an omitted field is left untouched (nil), rather than being
// overwritten with a zero value. Unlike create, `type`/`name` are NOT required —
// a caller may send just the fields it wants to change (e.g. only `name`, or
// only `credentials`). A channel's type is immutable and is ignored here.
type UpdateChannelRequest struct {
	Name       *string `json:"name"`
	Identifier *string `json:"identifier"`
	// Environment is included only so the service can explicitly reject an
	// attempt to change it (it is immutable after creation). nil = not touched.
	Environment *string           `json:"environment"`
	Config      map[string]string `json:"config"`
	Credentials map[string]string `json:"credentials"`
	WebhookURL  *string           `json:"webhook_url"`
}

// derefOr returns *p when non-nil, otherwise def. Used to feed the non-secret
// audit trail from a partial update whose fields are optional pointers.
func derefOr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}

type TestChannelRequest struct {
	Type        string            `json:"type"`
	Config      map[string]string `json:"config"`
	Credentials map[string]string `json:"credentials"`
}

// List godoc
// @Summary      List channels
// @Description  Returns all channels for the current tenant
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} Response{data=[]entity.Channel}
// @Failure      401 {object} Response
// @Router       /channels [get]
func (h *ChannelHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	channels, err := h.channelService.List(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}

	display := make([]*entity.Channel, len(channels))
	for i, ch := range channels {
		display[i] = withDisplayConfig(ch)
	}
	RespondSuccess(c, display)
}

// Create godoc
// @Summary      Create channel
// @Description  Create a new channel for the current tenant
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateChannelRequest true "Channel data"
// @Success      201 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /channels [post]
func (h *ChannelHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateChannelInput{
		TenantID:    tenantID,
		Type:        req.Type,
		Name:        req.Name,
		Identifier:  req.Identifier,
		Environment: req.Environment,
		Config:      req.Config,
		Credentials: req.Credentials,
		WebhookURL:  req.WebhookURL,
	}

	channel, err := h.channelService.Create(c.Request.Context(), input)
	if err != nil {
		// Rejections from the environment/credential binding validation are a
		// security signal (attempt to cross environments) and low-volume, so
		// they get a trail entry; ordinary validation noise does not.
		if service.IsEnvironmentBindingError(err) {
			changes := channelAuditChanges(req.Type, req.Environment, req.Config, req.Credentials)
			changes["rejection_reason"] = err.Error()
			h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
				"channel.create_rejected", "channel", "", changes)
		}
		RespondError(c, err)
		return
	}

	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
		"channel.create", "channel", channel.ID,
		channelAuditChanges(string(channel.Type), string(channel.Environment), channel.Config, req.Credentials))
	RespondCreated(c, channel)
}

// TestConnection validates channel credentials without creating a channel.
func (h *ChannelHandler) TestConnection(c *gin.Context) {
	h.testConnection(c, "")
}

func (h *ChannelHandler) TestWhatsAppConnection(c *gin.Context) {
	h.testConnection(c, "whatsapp_official")
}

func (h *ChannelHandler) TestTelegramConnection(c *gin.Context) {
	h.testConnection(c, "telegram")
}

func (h *ChannelHandler) TestTwilioConnection(c *gin.Context) {
	h.testConnection(c, "sms")
}

func (h *ChannelHandler) TestFacebookConnection(c *gin.Context) {
	h.testConnection(c, "facebook")
}

func (h *ChannelHandler) TestInstagramConnection(c *gin.Context) {
	h.testConnection(c, "instagram")
}

func (h *ChannelHandler) TestTeamsConnection(c *gin.Context) {
	h.testConnection(c, "teams")
}

func (h *ChannelHandler) TestSlackConnection(c *gin.Context) {
	h.testConnection(c, "slack")
}

func (h *ChannelHandler) TestMattermostConnection(c *gin.Context) {
	h.testConnection(c, "mattermost")
}

func (h *ChannelHandler) TestEmailConnection(c *gin.Context) {
	h.testConnection(c, "email")
}

func (h *ChannelHandler) testConnection(c *gin.Context, forcedType string) {
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	channelType := forcedType
	if channelType == "" {
		channelType = stringFromMap(raw, "type")
	}
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	if channelType == "" {
		RespondValidationError(c, "type is required", nil)
		return
	}

	config := flattenChannelTestConfig(raw)
	if err := validateChannelTestConfig(channelType, config); err != nil {
		RespondValidationError(c, err.Error(), nil)
		return
	}

	// Live provider check for connectors that support one (Teams/Slack/Mattermost):
	// actually reach the provider with the supplied credentials.
	message := "configuration accepted"
	if detail, ok, err := liveChannelCheck(c.Request.Context(), channelType, config); ok {
		if err != nil {
			RespondValidationError(c, "connection test failed: "+err.Error(), nil)
			return
		}
		message = detail
	}

	RespondSuccess(c, gin.H{
		"status":  "ok",
		"type":    channelType,
		"valid":   true,
		"message": message,
	})
}

// liveChannelCheck performs a real provider round-trip for connectors that
// support one. The second return value reports whether a live check exists for
// the type (false → caller keeps the static result). The check is time-bounded.
func liveChannelCheck(ctx context.Context, channelType string, config map[string]string) (detail string, supported bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch channelType {
	case "teams":
		return "credentials valid (AAD token acquired)", true,
			teams.NewClientFromCredentials(config).Ping(ctx)
	case "slack":
		team, user, e := slack.NewClient(config[slack.CredBotToken]).AuthTest(ctx)
		if e != nil {
			return "", true, e
		}
		return fmt.Sprintf("connected to Slack (team %q, bot %q)", team, user), true, nil
	case "mattermost":
		_, username, e := mattermost.NewClient(mattermost.Config{
			BaseURL:  config[mattermost.CredBaseURL],
			BotToken: config[mattermost.CredBotToken],
		}).Me(ctx)
		if e != nil {
			return "", true, e
		}
		return fmt.Sprintf("connected to Mattermost (bot %q)", username), true, nil
	case "email":
		return liveEmailCheck(ctx, config)
	default:
		return "", false, nil
	}
}

// liveEmailCheck exercita a configuração de e-mail de verdade, nas DUAS direções
// quando ambas estiverem configuradas.
//
// Testar só a saída deixaria passar o erro mais caro do canal de e-mail: com
// IMAP mal configurado o canal envia normalmente e nunca recebe resposta —
// falha silenciosa que só aparece quando um cliente reclama que ninguém
// respondeu. Por isso a entrada, quando declarada, também é verificada.
//
// Nada é enviado: os provedores validam credencial por chamada de API, e SMTP
// e IMAP apenas conectam e autenticam.
func liveEmailCheck(ctx context.Context, config map[string]string) (string, bool, error) {
	cfg := email.ConfigFromMap(config)

	// A validação estática já sabe o que cada provedor exige; usá-la aqui dá
	// mensagem específica ("sendgrid_api_key is required") em vez de um erro de
	// conexão confuso lá na frente.
	if err := cfg.Validate(); err != nil {
		return "", true, err
	}

	client, err := email.NewClient(cfg)
	if err != nil {
		return "", true, err
	}
	if err := client.TestConnection(ctx); err != nil {
		return "", true, fmt.Errorf("envio (%s): %w", client.GetProviderName(), err)
	}

	detail := fmt.Sprintf("envio ok via %s", client.GetProviderName())

	// IMAP é opcional: só o SMTP tem recebimento próprio. Os demais provedores
	// recebem por webhook, que não dá para verificar daqui.
	if cfg.IMAPHost == "" {
		return detail + " (recebimento por webhook, não verificável aqui)", true, nil
	}

	imapClient, err := email.NewIMAPClient(cfg)
	if err != nil {
		return "", true, fmt.Errorf("recebimento (IMAP): %w", err)
	}
	if err := imapClient.Connect(ctx); err != nil {
		return "", true, fmt.Errorf("recebimento (IMAP em %s): %w", cfg.IMAPHost, err)
	}
	defer imapClient.Disconnect()

	return detail + fmt.Sprintf(", recebimento ok via IMAP em %s", cfg.IMAPHost), true, nil
}

// Get godoc
// @Summary      Get channel
// @Description  Returns a channel by ID
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id} [get]
func (h *ChannelHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	channel, err := h.channelService.GetByTenantAndID(c.Request.Context(), tenantID, id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, withDisplayConfig(channel))
}

// nonSecretCredentialKeys lists, per channel type, credential keys safe to surface
// back to the admin UI for edit-form prefill. Secrets (tokens, passwords, signing
// secrets) are deliberately excluded and never returned.
var nonSecretCredentialKeys = map[string][]string{
	teams.ChannelType:      {teams.CredAppID, teams.CredTenantID, teams.CredServiceURL},
	slack.ChannelType:      {slack.CredAppID, slack.CredBotUserID},
	mattermost.ChannelType: {mattermost.CredBaseURL, mattermost.CredBotUserID},
}

// withDisplayConfig returns a shallow copy of the channel with non-secret
// credential fields merged into Config for UI prefill. The original entity is
// never mutated and Credentials (json:"-") stay hidden. Existing Config keys win.
func withDisplayConfig(channel *entity.Channel) *entity.Channel {
	if channel == nil {
		return channel
	}
	keys := nonSecretCredentialKeys[string(channel.Type)]
	if len(keys) == 0 {
		return channel
	}

	clone := *channel
	clone.Config = map[string]string{}
	for k, v := range channel.Config {
		clone.Config[k] = v
	}
	for _, k := range keys {
		if _, exists := clone.Config[k]; exists {
			continue
		}
		if v := channel.Credentials[k]; v != "" {
			clone.Config[k] = v
		}
	}
	return &clone
}

// Update godoc
// @Summary      Update channel
// @Description  Update a channel's configuration
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body UpdateChannelRequest true "Channel update data (partial)"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id} [put]
func (h *ChannelHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Only fields present in the request are forwarded; a nil pointer means
	// "leave unchanged" (the service treats nil the same way).
	input := &service.UpdateChannelInput{
		Name:        req.Name,
		Identifier:  req.Identifier,
		Config:      req.Config,
		Credentials: req.Credentials,
		WebhookURL:  req.WebhookURL,
	}
	// Passed through (nil = untouched) so the service can reject an environment
	// change explicitly (immutable after creation) instead of ignoring it.
	input.Environment = req.Environment

	channel, err := h.channelService.UpdateForTenant(c.Request.Context(), tenantID, id, input)
	if err != nil {
		if service.IsEnvironmentBindingError(err) {
			// type is immutable on update, so it is not part of the request.
			changes := channelAuditChanges("", derefOr(req.Environment, ""), req.Config, req.Credentials)
			changes["rejection_reason"] = err.Error()
			h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
				"channel.update_rejected", "channel", id, changes)
		}
		RespondError(c, err)
		return
	}

	// Delta of the audit-relevant fields the request actually carried; secrets
	// never enter the trail (INV-002).
	delta := channelAuditChanges(string(channel.Type), string(channel.Environment), channel.Config, req.Credentials)
	if req.Name != nil && *req.Name != "" {
		delta["name"] = *req.Name
	}
	if req.Identifier != nil && *req.Identifier != "" {
		delta["identifier"] = *req.Identifier
	}
	if req.WebhookURL != nil && *req.WebhookURL != "" {
		delta["webhook_url"] = *req.WebhookURL
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
		"channel.update", "channel", channel.ID, delta)
	RespondSuccess(c, channel)
}

// Delete godoc
// @Summary      Delete channel
// @Description  Delete a channel by ID
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      204 "No Content"
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id} [delete]
func (h *ChannelHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	if err := h.channelService.DeleteForTenant(c.Request.Context(), tenantID, id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

// Reencrypt re-encrypts all of the tenant's channel credentials with the
// current primary key (used after a crypto key rotation).
func (h *ChannelHandler) Reencrypt(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	count, err := h.channelService.ReencryptCredentials(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, gin.H{"reencrypted": count})
}

// Connect godoc
// @Summary      Connect channel
// @Description  Connect a channel to start receiving messages
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      200 {object} Response{data=object}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/connect [post]
func (h *ChannelHandler) Connect(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	result, err := h.channelService.ConnectForTenant(c.Request.Context(), tenantID, id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

// maxPasskeyAssertionBytes caps the WebAuthn assertion body. A real assertion is
// a few hundred bytes to a couple KB; this is a generous safety limit.
const maxPasskeyAssertionBytes = 64 << 10

// SubmitPasskeyResponse godoc
// @Summary      Submit a WhatsApp passkey assertion
// @Description  Completes passkey-locked WhatsApp linking with the WebAuthn assertion produced by the account owner's browser
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      202 {object} Response
// @Failure      400 {object} Response
// @Failure      409 {object} Response
// @Router       /channels/{id}/passkey/response [post]
func (h *ChannelHandler) SubmitPasskeyResponse(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Read the assertion verbatim. It must not be re-encoded on the way to
	// whatsmeow (its base64url fields would break the server's verification).
	assertion, err := c.GetRawData()
	if err != nil {
		RespondValidationError(c, "Could not read the passkey assertion", nil)
		return
	}
	if len(assertion) == 0 {
		RespondValidationError(c, "A passkey assertion is required", nil)
		return
	}
	if len(assertion) > maxPasskeyAssertionBytes {
		RespondValidationError(c, "Passkey assertion is too large", nil)
		return
	}

	if err := h.channelService.SubmitPasskeyResponse(c.Request.Context(), tenantID, id, assertion); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"status": "accepted"})
}

// Disconnect godoc
// @Summary      Disconnect channel
// @Description  Disconnect a channel to stop receiving messages
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      200 {object} Response{data=object{message=string}}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/disconnect [post]
func (h *ChannelHandler) Disconnect(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	if err := h.channelService.DisconnectForTenant(c.Request.Context(), tenantID, id); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Channel disconnected"})
}

// UpdateStatusRequest represents a request to update channel status
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

// UpdateStatus godoc
// @Summary      Update channel status
// @Description  Update a channel's status (active/inactive)
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body UpdateStatusRequest true "New status"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/status [put]
func (h *ChannelHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid status. Must be 'active' or 'inactive'", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	channel, err := h.channelService.UpdateStatusForTenant(c.Request.Context(), tenantID, id, req.Status)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, channel)
}

// UpdateEnabled godoc
// @Summary      Update channel enabled state
// @Description  Enable or disable a channel (system-level, independent of connection status)
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body UpdateEnabledRequest true "Enabled state"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/enabled [put]
func (h *ChannelHandler) UpdateEnabled(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req UpdateEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	channel, err := h.channelService.UpdateEnabledForTenant(c.Request.Context(), tenantID, id, req.Enabled)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, channel)
}

// UpdateEnabledRequest represents a request to enable/disable a channel
type UpdateEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// PairCodeRequest represents a request for WhatsApp pair code
type PairCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

// RequestPairCode godoc
// @Summary      Request pair code
// @Description  Request a pair code for WhatsApp authentication (alternative to QR code)
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body PairCodeRequest true "Phone number"
// @Success      200 {object} Response{data=service.ConnectResult}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/pair [post]
func (h *ChannelHandler) RequestPairCode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req PairCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "phone_number is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	result, err := h.channelService.RequestPairCodeForTenant(c.Request.Context(), tenantID, id, req.PhoneNumber)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

func flattenChannelTestConfig(raw map[string]interface{}) map[string]string {
	config := make(map[string]string)

	for key, value := range raw {
		switch key {
		case "type", "config", "credentials":
			continue
		default:
			if str, ok := value.(string); ok {
				config[normalizeConfigKey(key)] = str
			}
		}
	}

	mergeStringMap(config, raw["config"])
	mergeStringMap(config, raw["credentials"])

	return config
}

func mergeStringMap(dst map[string]string, value interface{}) {
	values, ok := value.(map[string]interface{})
	if !ok {
		return
	}

	for key, rawValue := range values {
		if str, ok := rawValue.(string); ok {
			dst[normalizeConfigKey(key)] = str
		}
	}
}

func normalizeConfigKey(key string) string {
	replacements := map[string]string{
		"accessToken":         "access_token",
		"phoneNumberId":       "phone_number_id",
		"phoneNumberID":       "phone_number_id",
		"apiVersion":          "api_version",
		"botToken":            "bot_token",
		"accountSid":          "account_sid",
		"accountSID":          "account_sid",
		"authToken":           "auth_token",
		"apiKeySid":           "api_key_sid",
		"apiKeySID":           "api_key_sid",
		"apiKeySecret":        "api_key_secret",
		"messagingServiceSid": "messaging_service_sid",
		"messagingServiceSID": "messaging_service_sid",
		"pageId":              "page_id",
		"pageID":              "page_id",
		"pageAccessToken":     "page_access_token",
		"instagramId":         "instagram_id",
		"instagramID":         "instagram_id",
		"appId":               "app_id",
		"appID":               "app_id",
		"appSecret":           "app_secret",
		"verifyToken":         "verify_token",
		"agentId":             "agent_id",
		"agentID":             "agent_id",
		"apiKey":              "api_key",
		"apiSecret":           "api_secret",
		"baseUrl":             "base_url",
		"baseURL":             "base_url",
		"senderId":            "sender_id",
		"senderID":            "sender_id",
		"webhookUrl":          "webhook_url",
		"webhookURL":          "webhook_url",
		"webhookSecret":       "webhook_secret",
		"statusCallbackUrl":   "status_callback_url",
		"statusCallbackURL":   "status_callback_url",
		"userAccessToken":     "user_access_token",
	}
	if replacement, ok := replacements[key]; ok {
		return replacement
	}
	return key
}

func stringFromMap(values map[string]interface{}, key string) string {
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return value
}

func validateChannelTestConfig(channelType string, config map[string]string) error {
	switch channelType {
	case "whatsapp", "whatsapp_official":
		if strings.TrimSpace(config["access_token"]) == "" {
			return fmt.Errorf("access_token is required")
		}
		if strings.TrimSpace(config["phone_number_id"]) == "" {
			return fmt.Errorf("phone_number_id is required")
		}
		return nil
	case "telegram":
		return telegram.NewAdapter().Initialize(config)
	case "sms", "twilio":
		return sms.NewAdapter().Initialize(config)
	case "facebook":
		cfg := &facebook.FacebookConfig{
			PageID:          config["page_id"],
			PageAccessToken: config["page_access_token"],
		}
		return cfg.Validate()
	case "instagram":
		cfg := &instagram.InstagramConfig{
			InstagramID:     config["instagram_id"],
			AccessToken:     config["access_token"],
			PageAccessToken: config["page_access_token"],
		}
		return cfg.Validate()
	case "rcs":
		cfg := &rcs.Config{
			Provider: rcs.Provider(config["provider"]),
			AgentID:  config["agent_id"],
			APIKey:   config["api_key"],
		}
		if cfg.Provider == "" {
			cfg.Provider = rcs.ProviderZenvia
		}
		return cfg.Validate()
	case "teams":
		if strings.TrimSpace(config[teams.CredAppID]) == "" {
			return fmt.Errorf("app_id is required")
		}
		if strings.TrimSpace(config[teams.CredAppPassword]) == "" {
			return fmt.Errorf("app_password is required")
		}
		return nil
	case "slack":
		if strings.TrimSpace(config[slack.CredBotToken]) == "" {
			return fmt.Errorf("bot_token is required")
		}
		return nil
	case "mattermost":
		if strings.TrimSpace(config[mattermost.CredBaseURL]) == "" {
			return fmt.Errorf("base_url is required")
		}
		if strings.TrimSpace(config[mattermost.CredBotToken]) == "" {
			return fmt.Errorf("bot_token is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported channel type: %s", channelType)
	}
}
