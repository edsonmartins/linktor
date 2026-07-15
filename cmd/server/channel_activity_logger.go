package main

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// channelActivityLogger adapts the outbound worker's ChannelLogger interface
// (and inbound/lifecycle call sites) onto the ObservabilityService so channel
// activity lands in the message_logs store surfaced by the admin observability
// log viewer. The source field tags entries (channel vs webhook) for filtering.
//
// Writes are fire-and-forget on a background context with a short timeout: the
// send/receive path must never block on (or be aborted by) logging, and the
// worker's consume context may already be done by the time the insert runs.
type channelActivityLogger struct {
	svc    *service.ObservabilityService
	source entity.LogSource
}

func (l channelActivityLogger) Info(_ context.Context, tenantID, channelID, message string, metadata map[string]string) {
	l.write(entity.LogLevelInfo, tenantID, channelID, message, metadata)
}

func (l channelActivityLogger) Warn(_ context.Context, tenantID, channelID, message string, metadata map[string]string) {
	l.write(entity.LogLevelWarn, tenantID, channelID, message, metadata)
}

func (l channelActivityLogger) Error(_ context.Context, tenantID, channelID, message string, metadata map[string]string) {
	l.write(entity.LogLevelError, tenantID, channelID, message, metadata)
}

func (l channelActivityLogger) write(level entity.LogLevel, tenantID, channelID, message string, metadata map[string]string) {
	if l.svc == nil {
		return
	}
	source := l.source
	if source == "" {
		source = entity.LogSourceChannel
	}
	var chanPtr *string
	if channelID != "" {
		chanPtr = &channelID
	}
	meta := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		meta[k] = v
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		switch level {
		case entity.LogLevelError:
			_ = l.svc.LogError(ctx, tenantID, chanPtr, source, message, meta)
		case entity.LogLevelWarn:
			_ = l.svc.LogWarn(ctx, tenantID, chanPtr, source, message, meta)
		default:
			_ = l.svc.LogInfo(ctx, tenantID, chanPtr, source, message, meta)
		}
	}()
}
