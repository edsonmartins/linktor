package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordInbound_IncrementsCounter(t *testing.T) {
	before := testutil.ToFloat64(InboundMessages.WithLabelValues("telegram", ResultProcessed))
	RecordInbound("telegram", ResultProcessed, time.Now())
	after := testutil.ToFloat64(InboundMessages.WithLabelValues("telegram", ResultProcessed))
	if after != before+1 {
		t.Fatalf("inbound counter: want %v, got %v", before+1, after)
	}
}

func TestRecordOutbound_And_EmptyChannelNormalized(t *testing.T) {
	before := testutil.ToFloat64(OutboundMessages.WithLabelValues("unknown", ResultFailed))
	RecordOutbound("", ResultFailed, time.Now()) // empty -> "unknown"
	after := testutil.ToFloat64(OutboundMessages.WithLabelValues("unknown", ResultFailed))
	if after != before+1 {
		t.Fatalf("outbound counter: want %v, got %v", before+1, after)
	}
}

func TestSetNATSUp(t *testing.T) {
	SetNATSUp(true)
	if got := testutil.ToFloat64(NATSUp); got != 1 {
		t.Fatalf("nats up: want 1, got %v", got)
	}
	SetNATSUp(false)
	if got := testutil.ToFloat64(NATSUp); got != 0 {
		t.Fatalf("nats up: want 0, got %v", got)
	}
}

func TestGinMiddleware_RecordsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinMiddleware())
	r.GET("/api/v1/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	before := testutil.ToFloat64(HTTPRequests.WithLabelValues("GET", "/api/v1/ping", "200"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))
	after := testutil.ToFloat64(HTTPRequests.WithLabelValues("GET", "/api/v1/ping", "200"))
	if after != before+1 {
		t.Fatalf("http counter: want %v, got %v", before+1, after)
	}
}

func TestHandler_ExposesMetrics(t *testing.T) {
	RecordInbound("whatsapp", ResultProcessed, time.Now())
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("metrics status: want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "linktor_inbound_messages_total") {
		t.Fatalf("metrics output missing linktor_inbound_messages_total")
	}
}

func TestQueueGauges(t *testing.T) {
	SetStreamGauge("LINKTOR_DLQ", 7)
	if got := testutil.ToFloat64(StreamMessages.WithLabelValues("LINKTOR_DLQ")); got != 7 {
		t.Fatalf("stream gauge: want 7, got %v", got)
	}
	SetConsumerGauges("LINKTOR_MESSAGES", "inbound", 3, 2, 5)
	if got := testutil.ToFloat64(ConsumerPending.WithLabelValues("LINKTOR_MESSAGES", "inbound")); got != 3 {
		t.Fatalf("pending gauge: want 3, got %v", got)
	}
	if got := testutil.ToFloat64(ConsumerRedelivered.WithLabelValues("LINKTOR_MESSAGES", "inbound")); got != 5 {
		t.Fatalf("redelivered gauge: want 5, got %v", got)
	}
}
