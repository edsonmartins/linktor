package logger

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLoggerManager(t *testing.T) {
	base := zaptest.NewLogger(t)
	m := NewLoggerManager(base)

	require.NotNil(t, m)
	assert.NotNil(t, m.loggers)
	assert.Empty(t, m.loggers)
	assert.NotNil(t, m.base)
}

func TestLoggerManager_GetLogger_CreatesNew(t *testing.T) {
	base := zaptest.NewLogger(t)
	m := NewLoggerManager(base)

	cl := m.GetLogger("chan-1", "tenant-1")

	require.NotNil(t, cl)
	assert.Equal(t, "chan-1", cl.channelID)
	assert.Equal(t, "tenant-1", cl.tenantID)
	assert.Len(t, m.loggers, 1)
}

func TestLoggerManager_GetLogger_ReturnsCached(t *testing.T) {
	base := zaptest.NewLogger(t)
	m := NewLoggerManager(base)

	cl1 := m.GetLogger("chan-1", "tenant-1")
	cl2 := m.GetLogger("chan-1", "tenant-1")

	assert.Same(t, cl1, cl2, "expected the same cached pointer")
	assert.Len(t, m.loggers, 1)
}

func TestLoggerManager_GetLogger_DifferentChannels(t *testing.T) {
	base := zaptest.NewLogger(t)
	m := NewLoggerManager(base)

	cl1 := m.GetLogger("chan-1", "tenant-1")
	cl2 := m.GetLogger("chan-2", "tenant-1")
	cl3 := m.GetLogger("chan-1", "tenant-2")

	assert.NotSame(t, cl1, cl2)
	assert.NotSame(t, cl1, cl3)
	assert.NotSame(t, cl2, cl3)
	assert.Len(t, m.loggers, 3)
}

func TestChannelLogger_Info(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	base := zap.New(core)
	m := NewLoggerManager(base)

	cl := m.GetLogger("chan-info", "tenant-info")
	cl.Info("hello info", zap.String("extra", "value"))

	entries := recorded.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "hello info", entries[0].Message)
	assert.Equal(t, zapcore.InfoLevel, entries[0].Level)

	fieldMap := fieldsToMap(entries[0].ContextMap())
	assert.Equal(t, "chan-info", fieldMap["channel_id"])
	assert.Equal(t, "tenant-info", fieldMap["tenant_id"])
	assert.Equal(t, "value", fieldMap["extra"])
}

func TestChannelLogger_With(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	base := zap.New(core)
	m := NewLoggerManager(base)

	cl := m.GetLogger("chan-w", "tenant-w")
	cl2 := cl.With(zap.String("request_id", "req-123"))

	// The derived logger must retain channel/tenant identity.
	assert.Equal(t, "chan-w", cl2.channelID)
	assert.Equal(t, "tenant-w", cl2.tenantID)
	assert.NotSame(t, cl, cl2)

	cl2.Info("with test")

	entries := recorded.All()
	require.Len(t, entries, 1)
	fm := entries[0].ContextMap()
	assert.Equal(t, "chan-w", fm["channel_id"])
	assert.Equal(t, "tenant-w", fm["tenant_id"])
	assert.Equal(t, "req-123", fm["request_id"])
}

func TestLoggerManager_Close(t *testing.T) {
	base := zaptest.NewLogger(t)
	m := NewLoggerManager(base)

	m.GetLogger("chan-1", "tenant-1")
	m.GetLogger("chan-2", "tenant-1")
	require.Len(t, m.loggers, 2)

	m.Close()
	assert.Empty(t, m.loggers)
}

func TestChannelLogger_AllLevels(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	base := zap.New(core)
	m := NewLoggerManager(base)
	cl := m.GetLogger("chan-all", "tenant-all")

	cl.Debug("debug msg")
	cl.Info("info msg")
	cl.Warn("warn msg")
	cl.Error("error msg")

	entries := recorded.All()
	require.Len(t, entries, 4)

	expected := []struct {
		msg   string
		level zapcore.Level
	}{
		{"debug msg", zapcore.DebugLevel},
		{"info msg", zapcore.InfoLevel},
		{"warn msg", zapcore.WarnLevel},
		{"error msg", zapcore.ErrorLevel},
	}

	for i, e := range expected {
		assert.Equal(t, e.msg, entries[i].Message)
		assert.Equal(t, e.level, entries[i].Level)

		fm := entries[i].ContextMap()
		assert.Equal(t, "chan-all", fm["channel_id"])
		assert.Equal(t, "tenant-all", fm["tenant_id"])
	}
}

func TestLoggerManager_Concurrent(t *testing.T) {
	base := zaptest.NewLogger(t)
	m := NewLoggerManager(base)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]*ChannelLogger, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			// All goroutines request the same channel logger.
			results[idx] = m.GetLogger("shared-chan", "shared-tenant")
		}(i)
	}

	wg.Wait()

	// All goroutines must have received the same pointer.
	for i := 1; i < goroutines; i++ {
		assert.Same(t, results[0], results[i], "goroutine %d got a different logger", i)
	}
	assert.Len(t, m.loggers, 1)
}

// fieldsToMap converts a context map to map[string]interface{} (identity, but
// makes the helper explicit).
func fieldsToMap(m map[string]interface{}) map[string]interface{} {
	return m
}
