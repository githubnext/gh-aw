package workflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow/logging"
)

func TestCompilerLogger(t *testing.T) {
	c := NewCompiler(false, "", "1.0.0")

	if c.logger == nil {
		t.Fatal("Compiler logger is nil")
	}
}

func TestCompilerWithCustomOutput(t *testing.T) {
	c := NewCompilerWithCustomOutput(false, "", "/tmp/output", "1.0.0")

	if c.logger == nil {
		t.Fatal("Compiler logger is nil")
	}
}

func TestCompilerSetLogger(t *testing.T) {
	c := NewCompiler(false, "", "1.0.0")

	// Create a custom logger with a buffer to capture output
	var buf bytes.Buffer
	customLogger := logging.NewLoggerWithWriter(&buf)

	// Set the custom logger
	c.SetLogger(customLogger)

	// Verify the logger was set
	if c.GetLogger() != customLogger {
		t.Error("SetLogger did not set the custom logger")
	}

	// Test that the logger works
	c.logger.Infof("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in logger output, got: %s", output)
	}
}

func TestCompilerGetLogger(t *testing.T) {
	c := NewCompiler(false, "", "1.0.0")

	logger := c.GetLogger()
	if logger == nil {
		t.Fatal("GetLogger returned nil")
	}

	if logger != c.logger {
		t.Error("GetLogger did not return the compiler's logger")
	}
}
