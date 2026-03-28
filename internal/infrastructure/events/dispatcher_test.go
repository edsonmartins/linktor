package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProducer struct {
	name  string
	calls []*EventPayload
	err   error
}

func (m *mockProducer) Name() string { return m.name }

func (m *mockProducer) Produce(_ context.Context, event *EventPayload) error {
	m.calls = append(m.calls, event)
	return m.err
}

func TestNewDispatcher(t *testing.T) {
	p1 := &mockProducer{name: "p1"}
	p2 := &mockProducer{name: "p2"}

	d := NewDispatcher(p1, p2)
	require.NotNil(t, d)
	assert.Len(t, d.Producers(), 2)
}

func TestNewDispatcher_Empty(t *testing.T) {
	d := NewDispatcher()
	require.NotNil(t, d)
	assert.Empty(t, d.Producers())
}

func TestDispatcher_AddProducer(t *testing.T) {
	d := NewDispatcher()
	assert.Empty(t, d.Producers())

	d.AddProducer(&mockProducer{name: "added"})
	assert.Len(t, d.Producers(), 1)
	assert.Equal(t, "added", d.Producers()[0].Name())
}

func TestDispatcher_Dispatch_AllProducersCalled(t *testing.T) {
	p1 := &mockProducer{name: "nats"}
	p2 := &mockProducer{name: "webhook"}
	d := NewDispatcher(p1, p2)

	event := &EventPayload{
		TenantID:  "tenant-1",
		EventType: "message.inbound",
		Payload:   map[string]interface{}{"body": "hello"},
		Timestamp: time.Now(),
	}

	err := d.Dispatch(context.Background(), event)
	require.NoError(t, err)
	assert.Len(t, p1.calls, 1)
	assert.Len(t, p2.calls, 1)
	assert.Equal(t, event, p1.calls[0])
	assert.Equal(t, event, p2.calls[0])
}

func TestDispatcher_Dispatch_ContinuesOnError(t *testing.T) {
	failing := &mockProducer{name: "failing", err: errors.New("boom")}
	healthy := &mockProducer{name: "healthy"}
	d := NewDispatcher(failing, healthy)

	event := &EventPayload{
		TenantID:  "tenant-2",
		EventType: "contact.created",
		Timestamp: time.Now(),
	}

	err := d.Dispatch(context.Background(), event)
	require.NoError(t, err, "dispatcher should not propagate producer errors")
	assert.Len(t, failing.calls, 1, "failing producer should still have been called")
	assert.Len(t, healthy.calls, 1, "healthy producer should be called despite earlier failure")
}

func TestDispatcher_Dispatch_Empty(t *testing.T) {
	d := NewDispatcher()

	event := &EventPayload{
		TenantID:  "tenant-3",
		EventType: "system.health",
		Timestamp: time.Now(),
	}

	err := d.Dispatch(context.Background(), event)
	require.NoError(t, err, "dispatching with no producers should succeed")
}

func TestDispatcher_MultipleDispatches(t *testing.T) {
	p := &mockProducer{name: "counter"}
	d := NewDispatcher(p)

	for i := 0; i < 5; i++ {
		d.Dispatch(context.Background(), &EventPayload{
			EventType: "message.inbound",
			Timestamp: time.Now(),
		})
	}

	assert.Len(t, p.calls, 5)
}
