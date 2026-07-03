package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// mu guards the package logger. Init (write), Default (read + lazy init) and
// every log helper touch the same global, and tests/goroutines call them
// concurrently — without this lock `go test -race` flags a data race between a
// test's Init and a still-running goroutine's Default/Info.
var (
	mu  sync.RWMutex
	log *zap.Logger
)

// Init initializes the logger with the given configuration
func Init(level, format string) error {
	var config zap.Config

	if format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	built, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}

	mu.Lock()
	log = built
	mu.Unlock()
	return nil
}

// Default returns the configured logger, lazily creating a development logger if
// Init wasn't called. The returned *zap.Logger is itself safe for concurrent use;
// only the global pointer needs guarding.
func Default() *zap.Logger {
	mu.RLock()
	l := log
	mu.RUnlock()
	if l != nil {
		return l
	}

	mu.Lock()
	defer mu.Unlock()
	if log == nil { // re-check: another goroutine may have set it
		log, _ = zap.NewDevelopment()
	}
	return log
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Default().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Default().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Default().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Default().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Default().Fatal(msg, fields...)
	os.Exit(1)
}

// With creates a child logger with additional fields
func With(fields ...zap.Field) *zap.Logger {
	return Default().With(fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return Default().Sync()
}
