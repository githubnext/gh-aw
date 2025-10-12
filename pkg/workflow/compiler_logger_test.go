package workflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow/logging"
)

func TestCompilerLogger(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "compiler with verbose logger",
			verbose: true,
		},
		{
			name:    "compiler with non-verbose logger",
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler(tt.verbose, "", "1.0.0")

			if c.logger == nil {
				t.Fatal("Compiler logger is nil")
			}

			if c.logger.IsVerbose() != tt.verbose {
				t.Errorf("Logger.IsVerbose() = %v, want %v", c.logger.IsVerbose(), tt.verbose)
			}
		})
	}
}

func TestCompilerWithCustomOutput(t *testing.T) {
	c := NewCompilerWithCustomOutput(true, "", "/tmp/output", "1.0.0")

	if c.logger == nil {
		t.Fatal("Compiler logger is nil")
	}

	if !c.logger.IsVerbose() {
		t.Error("Expected verbose logger")
	}
}

func TestCompilerSetLogger(t *testing.T) {
	c := NewCompiler(false, "", "1.0.0")

	// Create a custom logger with a buffer to capture output
	var buf bytes.Buffer
	customLogger := logging.NewLoggerWithWriter(true, &buf)

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
	c := NewCompiler(true, "", "1.0.0")

	logger := c.GetLogger()
	if logger == nil {
		t.Fatal("GetLogger returned nil")
	}

	if logger != c.logger {
		t.Error("GetLogger did not return the compiler's logger")
	}
}

func TestCompilerLoggerVerboseBehavior(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		logFunc      func(*logging.Logger)
		shouldLog    bool
		expectedText string
	}{
		{
			name:    "verbose compiler logs debug",
			verbose: true,
			logFunc: func(l *logging.Logger) {
				l.Debugf("debug message")
			},
			shouldLog:    true,
			expectedText: "DEBUG",
		},
		{
			name:    "non-verbose compiler skips debug",
			verbose: false,
			logFunc: func(l *logging.Logger) {
				l.Debugf("debug message")
			},
			shouldLog:    false,
			expectedText: "",
		},
		{
			name:    "non-verbose compiler logs info",
			verbose: false,
			logFunc: func(l *logging.Logger) {
				l.Infof("info message")
			},
			shouldLog:    true,
			expectedText: "INFO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			c := NewCompiler(tt.verbose, "", "1.0.0")
			c.SetLogger(logging.NewLoggerWithWriter(tt.verbose, &buf))

			tt.logFunc(c.logger)

			output := buf.String()
			if tt.shouldLog {
				if !strings.Contains(output, tt.expectedText) {
					t.Errorf("Expected '%s' in output, got: %s", tt.expectedText, output)
				}
			} else {
				if len(output) > 0 {
					t.Errorf("Expected no output for non-verbose debug, got: %s", output)
				}
			}
		})
	}
}
