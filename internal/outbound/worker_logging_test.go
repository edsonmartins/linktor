package outbound

import (
	"context"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

type capturedLog struct {
	level     entity.LogLevel
	tenantID  string
	channelID string
	message   string
	metadata  map[string]string
}

// captureLogger records ChannelLogger calls for assertions.
type captureLogger struct{ entries []capturedLog }

func (c *captureLogger) Info(_ context.Context, tenantID, channelID, message string, metadata map[string]string) {
	c.entries = append(c.entries, capturedLog{entity.LogLevelInfo, tenantID, channelID, message, metadata})
}
func (c *captureLogger) Warn(_ context.Context, tenantID, channelID, message string, metadata map[string]string) {
	c.entries = append(c.entries, capturedLog{entity.LogLevelWarn, tenantID, channelID, message, metadata})
}
func (c *captureLogger) Error(_ context.Context, tenantID, channelID, message string, metadata map[string]string) {
	c.entries = append(c.entries, capturedLog{entity.LogLevelError, tenantID, channelID, message, metadata})
}

func TestLogActivityNilLoggerIsSafe(t *testing.T) {
	w := &Worker{} // logs == nil
	// Must not panic.
	w.logActivity(context.Background(), entity.LogLevelError, &nats.OutboundMessage{}, nil, "boom")
}

func TestLogActivityRoutesLevelAndMetadata(t *testing.T) {
	cap := &captureLogger{}
	w := &Worker{logs: cap}

	raw := &nats.OutboundMessage{
		ID:          "msg-1",
		TenantID:    "tenant-1",
		ChannelID:   "chan-1",
		ChannelType: "whatsapp",
	}
	msg := &Message{To: "+5511999999999"}

	w.logActivity(context.Background(), entity.LogLevelWarn, raw, msg, "temporário")
	w.logActivity(context.Background(), entity.LogLevelError, raw, nil, "permanente")
	w.logActivity(context.Background(), entity.LogLevelInfo, raw, msg, "entregue")

	if len(cap.entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(cap.entries))
	}

	warn := cap.entries[0]
	if warn.level != entity.LogLevelWarn || warn.tenantID != "tenant-1" || warn.channelID != "chan-1" {
		t.Fatalf("warn entry mismatch: %+v", warn)
	}
	// "to" is masked (INV-002): same format the guard blocks use — full
	// recipient numbers never reach message_logs.
	if warn.metadata["message_id"] != "msg-1" || warn.metadata["channel_type"] != "whatsapp" {
		t.Fatalf("warn metadata mismatch: %+v", warn.metadata)
	}
	if warn.metadata["to"] != entity.MaskRecipient("+5511999999999") {
		t.Fatalf("to must be masked, got %q", warn.metadata["to"])
	}
	if strings.Contains(warn.metadata["to"], "5511999999999") {
		t.Fatalf("to leaks the full number: %q", warn.metadata["to"])
	}

	// nil msg -> no "to" key, but still logs.
	errEntry := cap.entries[1]
	if errEntry.level != entity.LogLevelError {
		t.Fatalf("expected error level, got %s", errEntry.level)
	}
	if _, ok := errEntry.metadata["to"]; ok {
		t.Fatalf("did not expect 'to' when msg is nil: %+v", errEntry.metadata)
	}

	if cap.entries[2].level != entity.LogLevelInfo {
		t.Fatalf("expected info level, got %s", cap.entries[2].level)
	}
}
