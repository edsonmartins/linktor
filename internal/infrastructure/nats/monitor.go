package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// Monitor provides methods to monitor NATS JetStream streams and consumers
type Monitor struct {
	client *Client
}

// NewMonitor creates a new NATS monitor
func NewMonitor(client *Client) *Monitor {
	return &Monitor{client: client}
}

// GetQueueStats returns statistics for all streams
func (m *Monitor) GetQueueStats(ctx context.Context) (*entity.QueueStats, error) {
	js := m.client.JetStream()

	// Stream names to monitor
	streamNames := []string{
		StreamMessages,
		StreamEvents,
		StreamWebhooks,
	}

	var streams []entity.StreamInfo
	var totalMessages uint64
	var totalPending int

	for _, streamName := range streamNames {
		info, err := m.getStreamInfo(ctx, js, streamName)
		if err != nil {
			// Stream might not exist, skip it
			continue
		}

		streams = append(streams, *info)
		totalMessages += info.Messages

		for _, consumer := range info.Consumers {
			totalPending += consumer.Pending + consumer.AckPending
		}
	}

	return &entity.QueueStats{
		Streams:       streams,
		TotalMessages: totalMessages,
		TotalPending:  totalPending,
	}, nil
}

// getStreamInfo retrieves information about a specific stream
func (m *Monitor) getStreamInfo(ctx context.Context, js jetstream.JetStream, streamName string) (*entity.StreamInfo, error) {
	stream, err := js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info for %s: %w", streamName, err)
	}

	// Get consumer information
	consumers, err := m.getConsumersForStream(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumers for stream %s: %w", streamName, err)
	}

	return &entity.StreamInfo{
		Name:        info.Config.Name,
		Description: info.Config.Description,
		Messages:    info.State.Msgs,
		Bytes:       info.State.Bytes,
		Consumers:   consumers,
		FirstSeq:    info.State.FirstSeq,
		LastSeq:     info.State.LastSeq,
		Created:     info.Created,
	}, nil
}

// getConsumersForStream retrieves all consumers for a stream
func (m *Monitor) getConsumersForStream(ctx context.Context, stream jetstream.Stream) ([]entity.ConsumerInfo, error) {
	var consumers []entity.ConsumerInfo

	consumerLister := stream.ListConsumers(ctx)
	for consumerInfo := range consumerLister.Info() {
		consumers = append(consumers, entity.ConsumerInfo{
			Name:          consumerInfo.Name,
			Pending:       int(consumerInfo.NumPending),
			AckPending:    int(consumerInfo.NumAckPending),
			Redelivered:   int(consumerInfo.NumRedelivered),
			LastDelivered: consumerInfo.Delivered.Last,
		})
	}

	if err := consumerLister.Err(); err != nil {
		return nil, err
	}

	return consumers, nil
}

// GetStreamInfo returns information about a specific stream
func (m *Monitor) GetStreamInfo(ctx context.Context, streamName string) (*entity.StreamInfo, error) {
	js := m.client.JetStream()
	return m.getStreamInfo(ctx, js, streamName)
}

// ResetConsumer deletes and recreates a consumer to reset its position
func (m *Monitor) ResetConsumer(ctx context.Context, streamName, consumerName string) error {
	js := m.client.JetStream()

	stream, err := js.Stream(ctx, streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	// Get the existing consumer configuration
	consumer, err := stream.Consumer(ctx, consumerName)
	if err != nil {
		return fmt.Errorf("failed to get consumer %s: %w", consumerName, err)
	}

	consumerInfo, err := consumer.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to get consumer info for %s: %w", consumerName, err)
	}

	// Delete the consumer
	if err := stream.DeleteConsumer(ctx, consumerName); err != nil {
		return fmt.Errorf("failed to delete consumer %s: %w", consumerName, err)
	}

	// Recreate the consumer with the same configuration
	_, err = stream.CreateOrUpdateConsumer(ctx, consumerInfo.Config)
	if err != nil {
		return fmt.Errorf("failed to recreate consumer %s: %w", consumerName, err)
	}

	return nil
}

// PurgeStream removes all messages from a stream
func (m *Monitor) PurgeStream(ctx context.Context, streamName string) error {
	js := m.client.JetStream()

	stream, err := js.Stream(ctx, streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	if err := stream.Purge(ctx); err != nil {
		return fmt.Errorf("failed to purge stream %s: %w", streamName, err)
	}

	return nil
}

// GetConsumerInfo returns information about a specific consumer
func (m *Monitor) GetConsumerInfo(ctx context.Context, streamName, consumerName string) (*entity.ConsumerInfo, error) {
	js := m.client.JetStream()

	stream, err := js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	consumer, err := stream.Consumer(ctx, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer %s: %w", consumerName, err)
	}

	info, err := consumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info for %s: %w", consumerName, err)
	}

	return &entity.ConsumerInfo{
		Name:          info.Name,
		Pending:       int(info.NumPending),
		AckPending:    int(info.NumAckPending),
		Redelivered:   int(info.NumRedelivered),
		LastDelivered: info.Delivered.Last,
	}, nil
}
