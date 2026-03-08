package testutil

import (
	"context"
	"sync"

	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// Ensure MockProducer implements nats.Publisher
var _ nats.Publisher = (*MockProducer)(nil)

// MockProducer is a mock implementation of nats.Publisher for testing
// It captures all published messages for assertion
type MockProducer struct {
	mu               sync.Mutex
	OutboundMessages []*nats.OutboundMessage
	InboundMessages  []*nats.InboundMessage
	StatusUpdates    []*nats.StatusUpdate
	Events           []*nats.Event
	ReturnError      error
}

// NewMockProducer creates a new MockProducer
func NewMockProducer() *MockProducer {
	return &MockProducer{}
}

func (m *MockProducer) PublishInbound(ctx context.Context, msg *nats.InboundMessage) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InboundMessages = append(m.InboundMessages, msg)
	return nil
}

func (m *MockProducer) PublishOutbound(ctx context.Context, msg *nats.OutboundMessage) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OutboundMessages = append(m.OutboundMessages, msg)
	return nil
}

func (m *MockProducer) PublishStatusUpdate(ctx context.Context, status *nats.StatusUpdate) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusUpdates = append(m.StatusUpdates, status)
	return nil
}

func (m *MockProducer) PublishEvent(ctx context.Context, event *nats.Event) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, event)
	return nil
}

func (m *MockProducer) PublishWebhookDelivery(ctx context.Context, webhook *nats.WebhookDelivery) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	return nil
}
