package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/gateway"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type callStoreStub struct {
	message *entity.Message
	updated *entity.Message
}

func (s *callStoreStub) FindByExternalID(_ context.Context, id string) (*entity.Message, error) {
	if s.message != nil && s.message.ExternalID == id {
		return s.message, nil
	}
	return nil, errors.New("not found")
}

func (s *callStoreStub) Update(_ context.Context, m *entity.Message) error {
	s.updated = m
	return nil
}

type conversationStub struct{ conversation *entity.Conversation }

func (s conversationStub) FindByID(context.Context, string) (*entity.Conversation, error) {
	return s.conversation, nil
}

type channelStub struct{ channel *entity.Channel }

func (s channelStub) FindByID(context.Context, string) (*entity.Channel, error) {
	return s.channel, nil
}

type senderStub struct{ inputs []*usecase.SendMessageInput }

func (s *senderStub) Execute(_ context.Context, in *usecase.SendMessageInput) (*usecase.SendMessageOutput, error) {
	s.inputs = append(s.inputs, in)
	return &usecase.SendMessageOutput{}, nil
}

func newSinkUnderTest(t *testing.T, rejectMsg string) (*bridgeCallSink, *callStoreStub, *testutil.MockProducer, *senderStub, *[]string) {
	t.Helper()
	store := &callStoreStub{message: &entity.Message{
		ID: "m-1", ConversationID: "c-1", ExternalID: "CALL1",
		ContentType: entity.ContentTypeCall,
		Metadata:    map[string]string{"phone": "5511999999999", gateway.MetaCallState: gateway.CallStateRinging},
	}}
	producer := testutil.NewMockProducer()
	sender := &senderStub{}
	channel := &entity.Channel{ID: "ch-1", TenantID: "t-1", Config: map[string]string{"reject_call": "true", "reject_call_msg": rejectMsg}}
	sink := newBridgeCallSink(store, conversationStub{&entity.Conversation{ID: "c-1", TenantID: "t-1", ContactID: "ct-1", ChannelID: "ch-1"}}, channelStub{channel}, producer, sender)
	sink.sleep = func(time.Duration) {}
	var broadcasts []string
	sink.broadcast = func(tenantID, conversationID string, m *entity.Message) {
		broadcasts = append(broadcasts, tenantID+"/"+conversationID+"/"+m.Metadata[gateway.MetaCallState])
	}
	return sink, store, producer, sender, &broadcasts
}

// Cada desfecho vira metadata na mensagem, aviso ao admin e evento call.*;
// a duração só na encerrada e quem recusou só na recusada.
func TestBridgeCallSink_RecordsStateAndPublishes(t *testing.T) {
	sink, store, producer, sender, broadcasts := newSinkUnderTest(t, "")
	at := time.Unix(1700000000, 0)

	sink.Handle(context.Background(), gateway.CallUpdate{ChannelID: "ch-1", CallID: "CALL1", State: gateway.CallStateAnswered, At: at})
	require.Equal(t, gateway.CallStateAnswered, store.updated.Metadata[gateway.MetaCallState])
	require.Equal(t, []string{"t-1/c-1/answered"}, *broadcasts)
	require.Len(t, producer.Events, 1)
	ev := producer.Events[0]
	require.Equal(t, "call.answered", ev.Type)
	require.Equal(t, "t-1", ev.TenantID)
	require.Equal(t, "m-1", ev.Payload["message_id"])
	require.Equal(t, "ch-1", ev.Payload["channel_id"])
	require.Equal(t, "ct-1", ev.Payload["contact_id"])
	require.Equal(t, "5511999999999", ev.Payload["from"])
	require.Equal(t, at, ev.Timestamp)

	sink.Handle(context.Background(), gateway.CallUpdate{CallID: "CALL1", State: gateway.CallStateEnded, Duration: 42 * time.Second, At: at})
	require.Equal(t, "42", store.updated.Metadata[gateway.MetaCallDuration])
	require.Equal(t, "call.ended", producer.Events[1].Type)
	require.Equal(t, 42, producer.Events[1].Payload["duration_seconds"])

	sink.Handle(context.Background(), gateway.CallUpdate{CallID: "CALL1", State: gateway.CallStateRejected, RejectedBy: gateway.RejectedByOperator, At: at})
	require.Equal(t, gateway.RejectedByOperator, store.updated.Metadata[gateway.MetaCallRejectedBy])
	require.Equal(t, "call.rejected", producer.Events[2].Type)
	require.Empty(t, sender.inputs, "recusa pelo atendente não manda a mensagem automática")
}

// Recusa automática manda a mensagem configurada no canal pela conversa,
// como mensagem do sistema; sem mensagem configurada, nada sai.
func TestBridgeCallSink_AutoRejectReplies(t *testing.T) {
	sink, _, _, sender, _ := newSinkUnderTest(t, "Este número não atende chamadas.")
	sink.Handle(context.Background(), gateway.CallUpdate{CallID: "CALL1", State: gateway.CallStateRejected, RejectedBy: gateway.RejectedByAuto, At: time.Now()})
	require.Len(t, sender.inputs, 1)
	in := sender.inputs[0]
	require.Equal(t, "c-1", in.ConversationID)
	require.Equal(t, "t-1", in.TenantID)
	require.Equal(t, entity.SenderTypeSystem, in.SenderType)
	require.Equal(t, "Este número não atende chamadas.", in.Content)
	require.Equal(t, "CALL1", in.Metadata[gateway.MetaCallID])

	sink, _, _, sender, _ = newSinkUnderTest(t, "")
	sink.Handle(context.Background(), gateway.CallUpdate{CallID: "CALL1", State: gateway.CallStateRejected, RejectedBy: gateway.RejectedByAuto, At: time.Now()})
	require.Empty(t, sender.inputs)
}

// Mensagem da chamada que não existe (ou ainda não foi gravada após as
// tentativas) não derruba nada: só não registra.
func TestBridgeCallSink_UnknownCall(t *testing.T) {
	sink, store, producer, _, broadcasts := newSinkUnderTest(t, "")
	sink.Handle(context.Background(), gateway.CallUpdate{CallID: "OTHER", State: gateway.CallStateMissed, At: time.Now()})
	require.Nil(t, store.updated)
	require.Empty(t, producer.Events)
	require.Empty(t, *broadcasts)
}

type rejecterStub struct {
	err  error
	got  []string
	call string
}

func (r *rejecterStub) RejectCall(_ context.Context, channelID, callID, by string) error {
	r.got = append(r.got, channelID+"/"+callID+"/"+by)
	return r.err
}

type tenantChannelStub struct{ err error }

func (s tenantChannelStub) GetByTenantAndID(context.Context, string, string) (*entity.Channel, error) {
	return &entity.Channel{ID: "ch-1"}, s.err
}

func rejectRequest(t *testing.T, rejecter *rejecterStub) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", "t-1") })
	r.POST("/channels/:id/bridge-calls/:callId/reject", rejectBridgeCallHandler(tenantChannelStub{}, rejecter))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/channels/ch-1/bridge-calls/CALL1/reject", nil))
	return w
}

// A rota pede a recusa em nome do atendente e traduz os erros do gateway:
// chamada desconhecida 404, já não toca 409, bridge desligado 503.
func TestRejectBridgeCallHandler(t *testing.T) {
	rejecter := &rejecterStub{}
	w := rejectRequest(t, rejecter)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []string{"ch-1/CALL1/operator"}, rejecter.got)
	require.Contains(t, w.Body.String(), `"state":"rejected"`)

	for _, tc := range []struct {
		err  error
		want int
	}{
		{gateway.ErrCallNotFound, http.StatusNotFound},
		{gateway.ErrCallNotRinging, http.StatusConflict},
		{gateway.ErrNoBridge, http.StatusServiceUnavailable},
		{errors.New("bridge could not reject call: boom"), http.StatusInternalServerError},
	} {
		w := rejectRequest(t, &rejecterStub{err: tc.err})
		require.Equal(t, tc.want, w.Code, "erro %v", tc.err)
	}
}
