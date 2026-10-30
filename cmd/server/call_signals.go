package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/handlers"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/callsignal"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	apperrors "github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
)

// Chamadas vistas por um aparelho conectado (bridge ou sessão embutida). A
// oferta entra pelo pipeline de mensagens (tipo call, estado tocando). O
// callsignal.Tracker avisa aqui o desfecho, uma vez por chamada: atendida,
// não atendida ou recusada. Ele vira metadata na mensagem da chamada,
// atualização ao vivo no admin e evento call.* para o webhook.
// Recusa automática ainda manda a mensagem configurada no canal pela
// conversa, como uma resposta normal do atendimento.

type callMessageStore interface {
	messageByExternalIDFinder
	Update(ctx context.Context, message *entity.Message) error
}

type conversationFinder interface {
	FindByID(ctx context.Context, id string) (*entity.Conversation, error)
}

type channelFinder interface {
	FindByID(ctx context.Context, id string) (*entity.Channel, error)
}

type messageSender interface {
	Execute(ctx context.Context, input *usecase.SendMessageInput) (*usecase.SendMessageOutput, error)
}

type callOutcomeSink struct {
	messages      callMessageStore
	conversations conversationFinder
	channels      channelFinder
	producer      nats.Publisher
	sender        messageSender
	sleep         func(time.Duration)
	broadcast     func(tenantID, conversationID string, message *entity.Message)
}

func newCallOutcomeSink(messages callMessageStore, conversations conversationFinder, channels channelFinder, producer nats.Publisher, sender messageSender) *callOutcomeSink {
	return &callOutcomeSink{
		messages:      messages,
		conversations: conversations,
		channels:      channels,
		producer:      producer,
		sender:        sender,
		sleep:         time.Sleep,
		broadcast:     handlers.BroadcastMessageUpdated,
	}
}

// Handle é o callsignal.Sink. A busca pelo call_id tenta de novo por um
// instante porque a mensagem da oferta é gravada por outro consumidor e a
// recusa automática chega logo atrás dela.
func (s *callOutcomeSink) Handle(ctx context.Context, u callsignal.Update) {
	message, err := findMessageByExternalID(ctx, s.messages, u.CallID, s.sleep)
	if err != nil || message == nil {
		logger.Warn("call " + u.CallID + ": message not found, state " + u.State + " not recorded")
		return
	}
	if message.Metadata == nil {
		message.Metadata = map[string]string{}
	}
	message.Metadata[callsignal.MetaState] = u.State
	if u.RejectedBy != "" {
		message.Metadata[callsignal.MetaRejectedBy] = u.RejectedBy
	}
	if err := s.messages.Update(ctx, message); err != nil {
		logger.Warn("call " + u.CallID + ": failed to record state " + u.State + ": " + err.Error())
		return
	}

	conversation, err := s.conversations.FindByID(ctx, message.ConversationID)
	if err != nil || conversation == nil {
		return
	}
	s.broadcast(conversation.TenantID, conversation.ID, message)

	if eventType := callEventType(u.State); eventType != "" && s.producer != nil {
		_ = s.producer.PublishEvent(ctx, &nats.Event{
			Type:     eventType,
			TenantID: conversation.TenantID,
			Payload: map[string]interface{}{
				"message_id":      message.ID,
				"conversation_id": conversation.ID,
				"contact_id":      conversation.ContactID,
				"channel_id":      conversation.ChannelID,
				"call_id":         u.CallID,
				"state":           u.State,
				"from":            message.Metadata["phone"],
				"rejected_by":     u.RejectedBy,
			},
			Timestamp: u.At,
		})
	}

	if u.State == callsignal.StateRejected && u.RejectedBy == callsignal.RejectedByAuto {
		s.autoReply(ctx, conversation, u.CallID)
	}
}

// autoReply manda a mensagem de recusa do canal (reject_call_msg) pela
// conversa, para ficar registrada e seguir pelo envio normal.
func (s *callOutcomeSink) autoReply(ctx context.Context, conversation *entity.Conversation, callID string) {
	if s.sender == nil || s.channels == nil {
		return
	}
	channel, err := s.channels.FindByID(ctx, conversation.ChannelID)
	if err != nil || channel == nil {
		return
	}
	settings := channel.GetAdvancedSettings()
	if settings == nil || settings.RejectCallMsg == "" {
		return
	}
	_, err = s.sender.Execute(ctx, &usecase.SendMessageInput{
		TenantID:       conversation.TenantID,
		ConversationID: conversation.ID,
		SenderType:     entity.SenderTypeSystem,
		ContentType:    entity.ContentTypeText,
		Content:        settings.RejectCallMsg,
		Metadata:       map[string]string{callsignal.MetaCallID: callID, "auto_reply": "reject_call"},
	})
	if err != nil {
		logger.Warn("call " + callID + ": reject message not sent: " + err.Error())
	}
}

// callEventType é o evento de webhook de cada desfecho; tocando não tem um
// (é o message.received da própria mensagem).
func callEventType(state string) string {
	switch state {
	case callsignal.StateAnswered:
		return nats.EventCallAnswered
	case callsignal.StateMissed:
		return nats.EventCallMissed
	case callsignal.StateRejected:
		return nats.EventCallRejected
	}
	return ""
}

// callRejecter é o pedaço do callsignal.Tracker que a rota de recusa usa.
type callRejecter interface {
	Reject(ctx context.Context, channelID, callID, by string) error
}

type tenantChannelFinder interface {
	GetByTenantAndID(ctx context.Context, tenantID, id string) (*entity.Channel, error)
}

// rejectCallHandler atende POST /channels/:id/calls/:callId/reject: o
// atendente recusa, pelo admin, uma chamada que está tocando no aparelho.
func rejectCallHandler(channels tenantChannelFinder, calls callRejecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := middleware.MustGetTenantID(c)
		if tenantID == "" {
			return
		}
		channelID, callID := c.Param("id"), c.Param("callId")
		if _, err := channels.GetByTenantAndID(c.Request.Context(), tenantID, channelID); err != nil {
			handlers.RespondError(c, err)
			return
		}
		if err := calls.Reject(c.Request.Context(), channelID, callID, callsignal.RejectedByOperator); err != nil {
			handlers.RespondError(c, callRejectError(err))
			return
		}
		c.JSON(http.StatusOK, gin.H{"call_id": callID, "state": callsignal.StateRejected})
	}
}

// callRejectError traduz os erros da recusa para a API: chamada que não está
// tocando (404), aparelho desligado (503).
func callRejectError(err error) error {
	switch {
	case errors.Is(err, callsignal.ErrCallNotFound):
		return apperrors.NotFound("call")
	case errors.Is(err, callsignal.ErrDeviceOffline):
		e := apperrors.New(apperrors.ErrCodeChannelDisconnected, "device is offline")
		e.StatusCode = http.StatusServiceUnavailable
		return e
	}
	return apperrors.New(apperrors.ErrCodeChannelError, err.Error())
}
