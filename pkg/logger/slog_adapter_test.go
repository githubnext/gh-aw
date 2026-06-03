//go:build !integration

package logger

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSlogLoggerWithHandler(t *testing.T) {
	setupDebugEnv(t, "*")

	logger := New("test:handler")
	slogLogger := NewSlogLoggerWithHandler(logger)

	output := captureStderr(func() {
		slogLogger.Info("test message from handler")
	})

	assert.Contains(t, output, "test:handler", "slog logger output should include logger namespace")
	assert.Contains(t, output, "· test message from handler", "info output should include dot prefix and message")
}

func TestSlogHandlerEnabled_DisabledLogger(t *testing.T) {
	setupDebugEnv(t, "")

	logger := New("test:disabled")
	handler := NewSlogHandler(logger)

	assert.False(t, handler.Enabled(context.Background(), slog.LevelInfo), "handler should be disabled when DEBUG does not match namespace")
}

func TestSlogHandlerHandle_WithAttrs(t *testing.T) {
	setupDebugEnv(t, "*")

	handler := NewSlogHandler(New("test:attrs"))
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "processing request", 0)
	record.AddAttrs(
		slog.String("user", "alice"),
		slog.Int("attempt", 3),
	)

	output := captureStderr(func() {
		err := handler.Handle(context.Background(), record)
		assert.NoError(t, err, "Handle should not return an error for a valid slog record")
	})

	assert.Contains(t, output, "test:attrs", "output should include namespace")
	assert.Contains(t, output, "· processing request", "info records should include dot prefix and message")
	assert.Contains(t, output, "user=alice", "output should render string slog attributes")
	assert.Contains(t, output, "attempt=3", "output should render numeric slog attributes")
}

func TestSlogHandlerHandle_LevelPrefixes(t *testing.T) {
	setupDebugEnv(t, "*")

	tests := []struct {
		name   string
		level  slog.Level
		prefix string
	}{
		{name: "debug uses dot prefix", level: slog.LevelDebug, prefix: "· "},
		{name: "info uses dot prefix", level: slog.LevelInfo, prefix: "· "},
		{name: "warn uses warning prefix", level: slog.LevelWarn, prefix: "⚠ "},
		{name: "error uses error prefix", level: slog.LevelError, prefix: "✗ "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSlogHandler(New("test:levels"))
			record := slog.NewRecord(time.Now(), tt.level, "level message", 0)

			output := captureStderr(func() {
				err := handler.Handle(context.Background(), record)
				assert.NoError(t, err, "Handle should succeed for %s level records", tt.level.String())
			})

			assert.Contains(t, output, "test:levels", "output should include logger namespace for %s level", tt.level.String())
			assert.Contains(t, output, tt.prefix+"level message", "output should include expected %s prefix", tt.level.String())
		})
	}
}

func setupDebugEnv(t *testing.T, debugValue string) {
	t.Helper()
	t.Setenv("DEBUG", debugValue)
	t.Setenv("ACTIONS_RUNNER_DEBUG", "")

	originalDebugEnv := debugEnv
	debugEnv = initDebugEnv()
	t.Cleanup(func() {
		debugEnv = originalDebugEnv
	})
}
