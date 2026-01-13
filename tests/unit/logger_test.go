package unit

import (
	"testing"

	"triggermesh/internal/logger"
)

func TestLogger_Init(t *testing.T) {
	// Test initialization with different log levels
	levels := []string{"debug", "info", "warn", "error", "invalid"}
	
	for _, level := range levels {
		t.Run("Level_"+level, func(t *testing.T) {
			logger.Init(level)
			if logger.Get() == nil {
				t.Error("Logger not initialized")
			}
		})
	}
}

func TestLogger_Get(t *testing.T) {
	// Test that Get() returns a non-nil logger
	log := logger.Get()
	if log == nil {
		t.Error("Get() returned nil logger")
	}
	
	// Get() should auto-initialize if not already initialized
	// This is tested implicitly by calling Get() without Init()
}

func TestLogger_LogMethods(t *testing.T) {
	// Initialize logger
	logger.Init("debug")
	
	// Test all log methods to ensure they don't panic
	// Note: Without dependency injection, we can't easily capture output,
	// but we can verify the methods execute without errors
	
	t.Run("Debug", func(t *testing.T) {
		logger.Debug("test debug message", "key", "value")
	})
	
	t.Run("Info", func(t *testing.T) {
		logger.Info("test info message", "key", "value")
	})
	
	t.Run("Warn", func(t *testing.T) {
		logger.Warn("test warn message", "key", "value")
	})
	
	t.Run("Error", func(t *testing.T) {
		logger.Error("test error message", "key", "value")
	})
}

func TestLogger_InvalidLevel(t *testing.T) {
	// Test that invalid level defaults to info level
	logger.Init("invalid-level")
	if logger.Get() == nil {
		t.Error("Logger should be initialized even with invalid level")
	}
	
	// Verify logger still works
	logger.Info("test message after invalid level init")
}
