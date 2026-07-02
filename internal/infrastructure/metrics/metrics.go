// Package metrics exposes Prometheus instrumentation for the LINKTOR messaging
// pipelines and HTTP surface. It uses the default Prometheus registry (which
// already carries the Go runtime and process collectors), so Handler() exposes
// everything at /metrics.
//
// Cardinality note: labels are deliberately bounded — channel_type, a small
// result enum, HTTP method + route template + status. NEVER label by tenant_id,
// contact, or raw URL path (unbounded → registry blow-up).
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "linktor"

var (
	// InboundMessages counts inbound messages leaving the receive pipeline,
	// partitioned by channel and outcome (processed | duplicate | failed).
	InboundMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "inbound",
		Name:      "messages_total",
		Help:      "Inbound messages processed, by channel type and result.",
	}, []string{"channel_type", "result"})

	// InboundDuration measures end-to-end inbound processing latency.
	InboundDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "inbound",
		Name:      "processing_seconds",
		Help:      "Inbound message processing duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"channel_type"})

	// OutboundMessages counts outbound send attempts, by channel and outcome
	// (sent | failed).
	OutboundMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "outbound",
		Name:      "messages_total",
		Help:      "Outbound messages sent, by channel type and result.",
	}, []string{"channel_type", "result"})

	// OutboundDuration measures outbound send latency (adapter round-trip).
	OutboundDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "outbound",
		Name:      "send_seconds",
		Help:      "Outbound message send duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"channel_type"})

	// PublishFailures counts NATS publish errors, by logical kind (event,
	// inbound, outbound, status).
	PublishFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "nats",
		Name:      "publish_failures_total",
		Help:      "NATS publish failures, by message kind.",
	}, []string{"kind"})

	// DLQMessages counts messages routed to a dead-letter stream.
	DLQMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "nats",
		Name:      "dlq_messages_total",
		Help:      "Messages routed to the dead-letter queue, by reason.",
	}, []string{"reason"})

	// NATSUp is 1 when the JetStream connection is live, 0 otherwise.
	NATSUp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "nats",
		Name:      "up",
		Help:      "1 if the NATS/JetStream connection is live, 0 otherwise.",
	})

	// HTTPRequests counts HTTP requests, by method, route template and status.
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "HTTP requests, by method, route and status code.",
	}, []string{"method", "route", "status"})

	// HTTPDuration measures HTTP request latency by method and route template.
	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "request_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "route"})

	// StreamMessages is the current number of messages held in each JetStream
	// stream (the DLQ stream's value is the dead-letter backlog depth).
	StreamMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "stream",
		Name:      "messages",
		Help:      "Messages currently stored in a JetStream stream.",
	}, []string{"stream"})

	// ConsumerPending is the number of messages a consumer has not yet received
	// (delivery backlog / lag).
	ConsumerPending = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "consumer",
		Name:      "pending",
		Help:      "Messages pending (not yet delivered) for a consumer.",
	}, []string{"stream", "consumer"})

	// ConsumerAckPending is the number of in-flight messages awaiting ack.
	ConsumerAckPending = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "consumer",
		Name:      "ack_pending",
		Help:      "In-flight messages awaiting ack for a consumer.",
	}, []string{"stream", "consumer"})

	// ConsumerRedelivered is the number of messages redelivered to a consumer
	// (a rising value signals repeated processing failures).
	ConsumerRedelivered = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "consumer",
		Name:      "redelivered",
		Help:      "Messages redelivered to a consumer.",
	}, []string{"stream", "consumer"})
)

// SetStreamGauge records the current message count for a stream.
func SetStreamGauge(stream string, messages uint64) {
	StreamMessages.WithLabelValues(stream).Set(float64(messages))
}

// SetConsumerGauges records the current lag metrics for a consumer.
func SetConsumerGauges(stream, consumer string, pending, ackPending, redelivered int) {
	ConsumerPending.WithLabelValues(stream, consumer).Set(float64(pending))
	ConsumerAckPending.WithLabelValues(stream, consumer).Set(float64(ackPending))
	ConsumerRedelivered.WithLabelValues(stream, consumer).Set(float64(redelivered))
}

// Result label values.
const (
	ResultProcessed = "processed"
	ResultDuplicate = "duplicate"
	ResultFailed    = "failed"
	ResultSent      = "sent"
)

// RecordInbound records an inbound message outcome and its processing latency.
func RecordInbound(channelType, result string, started time.Time) {
	channelType = normalizeChannel(channelType)
	InboundMessages.WithLabelValues(channelType, result).Inc()
	InboundDuration.WithLabelValues(channelType).Observe(time.Since(started).Seconds())
}

// RecordOutbound records an outbound send outcome and its latency.
func RecordOutbound(channelType, result string, started time.Time) {
	channelType = normalizeChannel(channelType)
	OutboundMessages.WithLabelValues(channelType, result).Inc()
	OutboundDuration.WithLabelValues(channelType).Observe(time.Since(started).Seconds())
}

// IncPublishFailure increments the publish-failure counter for a message kind.
func IncPublishFailure(kind string) { PublishFailures.WithLabelValues(kind).Inc() }

// IncDLQ increments the dead-letter counter for a reason.
func IncDLQ(reason string) { DLQMessages.WithLabelValues(reason).Inc() }

// SetNATSUp reflects the current NATS connection state.
func SetNATSUp(up bool) {
	if up {
		NATSUp.Set(1)
	} else {
		NATSUp.Set(0)
	}
}

// normalizeChannel bounds the channel_type label so an unexpected/empty value
// cannot create an unbounded time series.
func normalizeChannel(channelType string) string {
	if channelType == "" {
		return "unknown"
	}
	return channelType
}

// Handler returns the Prometheus scrape handler for /metrics.
func Handler() http.Handler { return promhttp.Handler() }

// GinMiddleware records request count and latency for every HTTP request using
// the matched route template (never the raw path) to keep cardinality bounded.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched" // 404s and unknown paths collapse to one series
		}
		HTTPRequests.WithLabelValues(c.Request.Method, route, strconv.Itoa(c.Writer.Status())).Inc()
		HTTPDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}
