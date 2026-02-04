package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/msgfy/linktor/internal/infrastructure/config"
)

// Client wraps a NATS connection with JetStream support
type Client struct {
	conn      *nats.Conn
	js        jetstream.JetStream
	clientID  string
	clusterID string
}

// NewClient creates a new NATS client with JetStream
func NewClient(cfg *config.NATSConfig) (*Client, error) {
	// Connect to NATS
	opts := []nats.Option{
		nats.Name(cfg.ClientID),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.PingInterval(20 * time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectBufSize(5 * 1024 * 1024), // 5MB buffer
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &Client{
		conn:      conn,
		js:        js,
		clientID:  cfg.ClientID,
		clusterID: cfg.ClusterID,
	}

	// Initialize streams
	if err := client.initializeStreams(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	return client, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Drain()
		c.conn.Close()
	}
}

// Conn returns the underlying NATS connection
func (c *Client) Conn() *nats.Conn {
	return c.conn
}

// JetStream returns the JetStream context
func (c *Client) JetStream() jetstream.JetStream {
	return c.js
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// initializeStreams creates the required JetStream streams
func (c *Client) initializeStreams(ctx context.Context) error {
	streams := []jetstream.StreamConfig{
		{
			Name:        StreamMessages,
			Description: "Linktor message stream for inbound and outbound messages",
			Subjects: []string{
				SubjectInboundAll,
				SubjectOutboundAll,
				SubjectStatusAll,
			},
			Retention:    jetstream.WorkQueuePolicy,
			MaxConsumers: -1,
			MaxMsgs:      -1,
			MaxBytes:     1024 * 1024 * 1024, // 1GB
			MaxAge:       7 * 24 * time.Hour, // 7 days
			MaxMsgSize:   4 * 1024 * 1024,    // 4MB per message
			Discard:      jetstream.DiscardOld,
			Storage:      jetstream.FileStorage,
			Replicas:     1,
			Duplicates:   5 * time.Minute,
		},
		{
			Name:        StreamEvents,
			Description: "Linktor event stream for system events",
			Subjects: []string{
				SubjectEventsAll,
			},
			Retention:    jetstream.InterestPolicy,
			MaxConsumers: -1,
			MaxMsgs:      -1,
			MaxBytes:     512 * 1024 * 1024, // 512MB
			MaxAge:       24 * time.Hour,    // 1 day
			MaxMsgSize:   1024 * 1024,       // 1MB per message
			Discard:      jetstream.DiscardOld,
			Storage:      jetstream.FileStorage,
			Replicas:     1,
		},
		{
			Name:        StreamWebhooks,
			Description: "Linktor webhook delivery stream",
			Subjects: []string{
				SubjectWebhooksAll,
			},
			Retention:    jetstream.WorkQueuePolicy,
			MaxConsumers: -1,
			MaxMsgs:      -1,
			MaxBytes:     256 * 1024 * 1024, // 256MB
			MaxAge:       3 * 24 * time.Hour, // 3 days
			MaxMsgSize:   1024 * 1024,        // 1MB per message
			Discard:      jetstream.DiscardOld,
			Storage:      jetstream.FileStorage,
			Replicas:     1,
		},
	}

	for _, streamCfg := range streams {
		_, err := c.js.CreateOrUpdateStream(ctx, streamCfg)
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", streamCfg.Name, err)
		}
	}

	return nil
}
