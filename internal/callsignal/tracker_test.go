package callsignal

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/require"
)

type channelStub map[string]*entity.Channel

func (c channelStub) FindByID(_ context.Context, id string) (*entity.Channel, error) {
	if ch, ok := c[id]; ok {
		return ch, nil
	}
	return nil, errors.New("not found")
}

func newTracker(t *testing.T, cfg map[string]string) (*Tracker, chan Update) {
	t.Helper()
	tr := NewTracker(channelStub{"ch-1": {ID: "ch-1", Config: cfg}})
	updates := make(chan Update, 8)
	tr.SetSink(func(_ context.Context, u Update) { updates <- u })
	return tr, updates
}

type offerProbe struct {
	published atomic.Int32
	rejected  atomic.Int32
	rejectErr error
}

func (p *offerProbe) offer(callID string) Offer {
	return Offer{
		ChannelID: "ch-1",
		CallID:    callID,
		Publish:   func(context.Context) error { p.published.Add(1); return nil },
		Reject: func(context.Context) error {
			p.rejected.Add(1)
			return p.rejectErr
		},
	}
}

func next(t *testing.T, updates chan Update) Update {
	t.Helper()
	select {
	case u := <-updates:
		return u
	case <-time.After(2 * time.Second):
		t.Fatal("sink não recebeu o desfecho")
		return Update{}
	}
}

func none(t *testing.T, updates chan Update) {
	t.Helper()
	select {
	case u := <-updates:
		t.Fatalf("desfecho inesperado: %+v", u)
	case <-time.After(50 * time.Millisecond):
	}
}

// A oferta publica uma vez; repetida (reconexão do aparelho) não publica.
func TestOfferPublishesOnce(t *testing.T) {
	tr, updates := newTracker(t, nil)
	p := &offerProbe{}
	require.NoError(t, tr.Offer(context.Background(), p.offer("A")))
	require.NoError(t, tr.Offer(context.Background(), p.offer("A")))
	require.Equal(t, int32(1), p.published.Load())
	require.True(t, tr.Ringing("A"))
	none(t, updates)
}

// Falha ao publicar não deixa a chamada no registro.
func TestOfferPublishFailureDropsCall(t *testing.T) {
	tr, _ := newTracker(t, nil)
	o := (&offerProbe{}).offer("F")
	o.Publish = func(context.Context) error { return errors.New("nats") }
	require.Error(t, tr.Offer(context.Background(), o))
	require.False(t, tr.Ringing("F"))
}

// Cada chamada tem um só desfecho, o do primeiro sinal; o término que segue
// um aceite (aparelho principal atendeu) não muda nada.
func TestSignalOutcomes(t *testing.T) {
	tr, updates := newTracker(t, nil)
	p := &offerProbe{}

	require.NoError(t, tr.Offer(context.Background(), p.offer("A")))
	tr.Signal("A", SignalAccept, time.Unix(114, 0))
	tr.Signal("A", SignalTerminate, time.Unix(114, 0))
	u := next(t, updates)
	require.Equal(t, Update{ChannelID: "ch-1", CallID: "A", State: StateAnswered, At: time.Unix(114, 0)}, u)
	none(t, updates)

	require.NoError(t, tr.Offer(context.Background(), p.offer("B")))
	tr.Signal("B", SignalTerminate, time.Unix(230, 0))
	require.Equal(t, StateMissed, next(t, updates).State)

	require.NoError(t, tr.Offer(context.Background(), p.offer("C")))
	tr.Signal("C", SignalReject, time.Unix(310, 0))
	u = next(t, updates)
	require.Equal(t, StateRejected, u.State)
	require.Equal(t, RejectedByDevice, u.RejectedBy)

	tr.Signal("Z", SignalAccept, time.Now())
	tr.Signal("C", "preaccept", time.Now())
	none(t, updates)
	require.False(t, tr.Ringing("A"))
}

// Sem nenhum desfecho (aparelho caiu no meio do toque) vira não atendida.
func TestRingTimeout(t *testing.T) {
	tr, updates := newTracker(t, nil)
	tr.ringTimeout = 20 * time.Millisecond
	require.NoError(t, tr.Offer(context.Background(), (&offerProbe{}).offer("T")))
	require.Equal(t, StateMissed, next(t, updates).State)
}

// Recusa pelo atendente usa o Reject da origem e registra quem pediu; outro
// canal ou chamada desconhecida é ErrCallNotFound; falha do aparelho mantém
// a chamada tocando.
func TestReject(t *testing.T) {
	tr, updates := newTracker(t, nil)
	p := &offerProbe{}
	require.NoError(t, tr.Offer(context.Background(), p.offer("R")))

	require.ErrorIs(t, tr.Reject(context.Background(), "ch-other", "R", RejectedByOperator), ErrCallNotFound)
	require.ErrorIs(t, tr.Reject(context.Background(), "ch-1", "NOPE", RejectedByOperator), ErrCallNotFound)

	p.rejectErr = errors.New("bridge offline")
	require.ErrorContains(t, tr.Reject(context.Background(), "ch-1", "R", RejectedByOperator), "bridge offline")
	require.True(t, tr.Ringing("R"))
	none(t, updates)

	p.rejectErr = nil
	require.NoError(t, tr.Reject(context.Background(), "ch-1", "R", RejectedByOperator))
	u := next(t, updates)
	require.Equal(t, StateRejected, u.State)
	require.Equal(t, RejectedByOperator, u.RejectedBy)
	require.ErrorIs(t, tr.Reject(context.Background(), "ch-1", "R", RejectedByOperator), ErrCallNotFound)
}

// Canal com reject_call recusa sozinho depois de publicar.
func TestAutoReject(t *testing.T) {
	tr, updates := newTracker(t, map[string]string{"reject_call": "true"})
	p := &offerProbe{}
	require.NoError(t, tr.Offer(context.Background(), p.offer("AUTO")))
	u := next(t, updates)
	require.Equal(t, StateRejected, u.State)
	require.Equal(t, RejectedByAuto, u.RejectedBy)
	require.Equal(t, int32(1), p.published.Load())
	require.Equal(t, int32(1), p.rejected.Load())
}

func TestMessageMetadataAndContent(t *testing.T) {
	require.Equal(t, map[string]string{MetaCallID: "X", MetaState: StateRinging}, MessageMetadata("X", false))
	require.Equal(t, "true", MessageMetadata("X", true)[MetaVideo])
	require.Equal(t, "Chamada de voz", Content(false))
	require.Equal(t, "Chamada de vídeo", Content(true))
}
