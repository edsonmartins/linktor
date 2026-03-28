package events

import (
	"context"
	"log"
	"time"
)

// EventPayload represents a system event to be dispatched
type EventPayload struct {
	TenantID  string
	EventType string // from valueobject constants
	Payload   map[string]interface{}
	Timestamp time.Time
}

// EventProducer is a generic interface for event delivery backends
// (NATS, Webhook, WebSocket, etc.)
type EventProducer interface {
	Name() string
	Produce(ctx context.Context, event *EventPayload) error
}

// Dispatcher routes events to all registered producers
type Dispatcher struct {
	producers []EventProducer
}

// NewDispatcher creates a Dispatcher with the given initial producers
func NewDispatcher(producers ...EventProducer) *Dispatcher {
	return &Dispatcher{
		producers: producers,
	}
}

// Dispatch sends the event to every registered producer.
// Errors are logged but do not short-circuit delivery to remaining producers.
func (d *Dispatcher) Dispatch(ctx context.Context, event *EventPayload) error {
	for _, p := range d.producers {
		if err := p.Produce(ctx, event); err != nil {
			log.Printf("[events.Dispatcher] producer %q failed for event %q: %v", p.Name(), event.EventType, err)
		}
	}
	return nil
}

// AddProducer appends a producer to the dispatcher
func (d *Dispatcher) AddProducer(p EventProducer) {
	d.producers = append(d.producers, p)
}

// Producers returns the current list of registered producers
func (d *Dispatcher) Producers() []EventProducer {
	return d.producers
}
