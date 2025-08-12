package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
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

func Info(format string, args ...any) {
	if len(args) > 0 {
		logger.Info(fmt.Sprintf(format, args...))
	} else {
		logger.Info(format)
	}
}

func Debug(format string, args ...any) {
	if len(args) > 0 {
		logger.Debug(fmt.Sprintf(format, args...))
	} else {
		logger.Debug(format)
	}
}

func Warn(format string, args ...any) {
	if len(args) > 0 {
		logger.Warn(fmt.Sprintf(format, args...))
	} else {
		logger.Warn(format)
	}
}

func Error(format string, args ...any) {
	if len(args) > 0 {
		logger.Error(fmt.Sprintf(format, args...))
	} else {
		logger.Error(format)
	}
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
