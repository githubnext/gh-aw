//go:build !integration

package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewSlogLoggerWithHandler(t *testing.T) {
	// Only run if DEBUG is enabled
	if os.Getenv("DEBUG") == "" {
		t.Skip("Skipping test: DEBUG environment variable not set")
	}

	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create logger and then slog logger from it
	logger := New("test:handler")
	slogLogger := NewSlogLoggerWithHandler(logger)

	// Test logging
	slogLogger.Info("test message from handler")

	// Close write end and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Restore stderr
	os.Stderr = oldStderr

	// Verify output contains expected message
	if !strings.Contains(output, "test:handler") {
		t.Errorf("Expected 'test:handler' namespace in output, got: %s", output)
	}
	if !strings.Contains(output, "[INFO] test message from handler") {
		t.Errorf("Expected info message in output, got: %s", output)
	}
}
