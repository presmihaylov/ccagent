package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// Global slog logger instance
var logger *slog.Logger
var currentWriter io.Writer = os.Stdout
var currentLevel slog.Level = slog.Level(1000)

func init() {
	// Initialize with high level to disable logging by default
	logger = slog.New(slog.NewTextHandler(currentWriter, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

// Info logs an info message with optional structured attributes
// Usage: log.Info("message") or log.Info("message", "key", value, ...)
func Info(format string, args ...any) {
	if len(args) > 0 {
		logger.Info(fmt.Sprintf(format, args...))
	} else {
		logger.Info(format)
	}
}

// InfoWith logs an info message with structured key-value pairs
func InfoWith(msg string, attrs ...any) {
	logger.Info(msg, attrs...)
}

// Debug logs a debug message with optional structured attributes
func Debug(format string, args ...any) {
	if len(args) > 0 {
		logger.Debug(fmt.Sprintf(format, args...))
	} else {
		logger.Debug(format)
	}
}

// DebugWith logs a debug message with structured key-value pairs
func DebugWith(msg string, attrs ...any) {
	logger.Debug(msg, attrs...)
}

// Warn logs a warning message with optional structured attributes
func Warn(format string, args ...any) {
	if len(args) > 0 {
		logger.Warn(fmt.Sprintf(format, args...))
	} else {
		logger.Warn(format)
	}
}

// WarnWith logs a warning message with structured key-value pairs
func WarnWith(msg string, attrs ...any) {
	logger.Warn(msg, attrs...)
}

// Error logs an error message with optional structured attributes
func Error(format string, args ...any) {
	if len(args) > 0 {
		logger.Error(fmt.Sprintf(format, args...))
	} else {
		logger.Error(format)
	}
}

// ErrorWith logs an error message with structured key-value pairs
func ErrorWith(msg string, attrs ...any) {
	logger.Error(msg, attrs...)
}

func SetLevel(level slog.Level) {
	currentLevel = level
	logger = slog.New(slog.NewTextHandler(currentWriter, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

func SetWriter(writer io.Writer) {
	currentWriter = writer
	logger = slog.New(slog.NewTextHandler(currentWriter, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

func SetWriterWithLevel(writer io.Writer, level slog.Level) {
	currentWriter = writer
	currentLevel = level
	logger = slog.New(slog.NewTextHandler(currentWriter, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

// Timer tracks elapsed time for an operation
type Timer struct {
	start time.Time
	name  string
}

// StartTimer begins timing an operation
func StartTimer(name string) *Timer {
	return &Timer{
		start: time.Now(),
		name:  name,
	}
}

// LogElapsed logs the elapsed time for the operation
func (t *Timer) LogElapsed(attrs ...any) {
	elapsed := time.Since(t.start)
	allAttrs := append([]any{"operation", t.name, "elapsed_ms", elapsed.Milliseconds()}, attrs...)
	logger.Info("⏱️ Operation completed", allAttrs...)
}

// LogElapsedWith logs the elapsed time with a custom message
func (t *Timer) LogElapsedWith(msg string, attrs ...any) {
	elapsed := time.Since(t.start)
	allAttrs := append([]any{"operation", t.name, "elapsed_ms", elapsed.Milliseconds()}, attrs...)
	logger.Info(msg, allAttrs...)
}

// WithContext returns a new logger with additional context attributes
func WithContext(attrs ...any) func(msg string, extraAttrs ...any) {
	return func(msg string, extraAttrs ...any) {
		allAttrs := append(attrs, extraAttrs...)
		logger.Info(msg, allAttrs...)
	}
}
