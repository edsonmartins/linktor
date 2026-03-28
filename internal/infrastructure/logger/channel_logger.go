package logger

import (
	"sync"

	"go.uber.org/zap"
)

// ChannelLogger provides per-channel structured logging.
// Every log entry automatically includes channelID and tenantID fields.
type ChannelLogger struct {
	base      *zap.Logger
	channelID string
	tenantID  string
}

// LoggerManager manages per-channel loggers with caching.
type LoggerManager struct {
	mu      sync.RWMutex
	loggers map[string]*ChannelLogger
	base    *zap.Logger
}

// NewLoggerManager creates a new LoggerManager that derives child loggers
// from the provided base zap.Logger.
func NewLoggerManager(base *zap.Logger) *LoggerManager {
	return &LoggerManager{
		loggers: make(map[string]*ChannelLogger),
		base:    base,
	}
}

// GetLogger returns a cached ChannelLogger for the given channel/tenant pair,
// creating one if it does not already exist.
func (m *LoggerManager) GetLogger(channelID, tenantID string) *ChannelLogger {
	key := tenantID + ":" + channelID

	// Fast path: read lock.
	m.mu.RLock()
	if cl, ok := m.loggers[key]; ok {
		m.mu.RUnlock()
		return cl
	}
	m.mu.RUnlock()

	// Slow path: write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if cl, ok := m.loggers[key]; ok {
		return cl
	}

	child := m.base.With(
		zap.String("channel_id", channelID),
		zap.String("tenant_id", tenantID),
	)

	cl := &ChannelLogger{
		base:      child,
		channelID: channelID,
		tenantID:  tenantID,
	}
	m.loggers[key] = cl
	return cl
}

// Close syncs all cached loggers and clears the cache.
func (m *LoggerManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cl := range m.loggers {
		_ = cl.base.Sync()
	}
	m.loggers = make(map[string]*ChannelLogger)
}

// Info logs an info-level message with the channel context fields.
func (l *ChannelLogger) Info(msg string, fields ...zap.Field) {
	l.base.Info(msg, fields...)
}

// Error logs an error-level message with the channel context fields.
func (l *ChannelLogger) Error(msg string, fields ...zap.Field) {
	l.base.Error(msg, fields...)
}

// Warn logs a warn-level message with the channel context fields.
func (l *ChannelLogger) Warn(msg string, fields ...zap.Field) {
	l.base.Warn(msg, fields...)
}

// Debug logs a debug-level message with the channel context fields.
func (l *ChannelLogger) Debug(msg string, fields ...zap.Field) {
	l.base.Debug(msg, fields...)
}

// With returns a new ChannelLogger that carries additional default fields.
func (l *ChannelLogger) With(fields ...zap.Field) *ChannelLogger {
	return &ChannelLogger{
		base:      l.base.With(fields...),
		channelID: l.channelID,
		tenantID:  l.tenantID,
	}
}
