package logger

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

// Init initializes the logger with the given log level
func Init(level string) {
	// Parse log level
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Configure the logger
	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	// Create a JSON handler that writes to stderr
	jsonHandler := slog.NewJSONHandler(os.Stderr, opts)

	// Create the logger
	logger = slog.New(jsonHandler)

	// Set the global logger
	slog.SetDefault(logger)
}

// Get returns the logger instance
func Get() *slog.Logger {
	if logger == nil {
		// Initialize with default level if not already initialized
		Init("info")
	}
	return logger
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}
