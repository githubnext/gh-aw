package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "verbose logger",
			verbose: true,
		},
		{
			name:    "non-verbose logger",
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.verbose)
			if logger == nil {
				t.Fatal("NewLogger returned nil")
			}
			if logger.Logger == nil {
				t.Fatal("Logger.Logger is nil")
			}
			if logger.IsVerbose() != tt.verbose {
				t.Errorf("IsVerbose() = %v, want %v", logger.IsVerbose(), tt.verbose)
			}
		})
	}
}

func TestNewLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	if logger == nil {
		t.Fatal("NewLoggerWithWriter returned nil")
	}
	if logger.Logger == nil {
		t.Fatal("Logger.Logger is nil")
	}
	if !logger.IsVerbose() {
		t.Error("Expected verbose logger")
	}
}

func TestLoggerInfof(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	logger.Infof("test message")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO level in output, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
}

func TestLoggerDebugf(t *testing.T) {
	tests := []struct {
		name      string
		verbose   bool
		shouldLog bool
	}{
		{
			name:      "verbose mode logs debug",
			verbose:   true,
			shouldLog: true,
		},
		{
			name:      "non-verbose mode skips debug",
			verbose:   false,
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLoggerWithWriter(tt.verbose, &buf)

			logger.Debugf("debug message")

			output := buf.String()
			if tt.shouldLog {
				if !strings.Contains(output, "DEBUG") {
					t.Errorf("Expected DEBUG level in output, got: %s", output)
				}
				if !strings.Contains(output, "debug message") {
					t.Errorf("Expected 'debug message' in output, got: %s", output)
				}
			} else {
				if strings.Contains(output, "DEBUG") || strings.Contains(output, "debug message") {
					t.Errorf("Expected no debug output, got: %s", output)
				}
			}
		})
	}
}

func TestLoggerWarnf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	logger.Warnf("warning message")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("Expected WARN level in output, got: %s", output)
	}
	if !strings.Contains(output, "warning message") {
		t.Errorf("Expected 'warning message' in output, got: %s", output)
	}
}

func TestLoggerErrorf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	logger.Errorf("error message")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected ERROR level in output, got: %s", output)
	}
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected 'error message' in output, got: %s", output)
	}
}

func TestLoggerInfoWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	logger.InfoWithFields("structured message", "key1", "value1", "key2", 42)

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO level in output, got: %s", output)
	}
	if !strings.Contains(output, "structured message") {
		t.Errorf("Expected 'structured message' in output, got: %s", output)
	}
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("Expected 'key1=value1' in output, got: %s", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("Expected 'key2=42' in output, got: %s", output)
	}
}

func TestLoggerDebugWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	logger.DebugWithFields("debug structured", "field", "value")

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Expected DEBUG level in output, got: %s", output)
	}
	if !strings.Contains(output, "debug structured") {
		t.Errorf("Expected 'debug structured' in output, got: %s", output)
	}
	if !strings.Contains(output, "field=value") {
		t.Errorf("Expected 'field=value' in output, got: %s", output)
	}
}

func TestLoggerWarnWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	logger.WarnWithFields("warning structured", "status", "degraded")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("Expected WARN level in output, got: %s", output)
	}
	if !strings.Contains(output, "warning structured") {
		t.Errorf("Expected 'warning structured' in output, got: %s", output)
	}
	if !strings.Contains(output, "status=degraded") {
		t.Errorf("Expected 'status=degraded' in output, got: %s", output)
	}
}

func TestLoggerErrorWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(true, &buf)

	logger.ErrorWithFields("error structured", "code", 500, "message", "internal error")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected ERROR level in output, got: %s", output)
	}
	if !strings.Contains(output, "error structured") {
		t.Errorf("Expected 'error structured' in output, got: %s", output)
	}
	if !strings.Contains(output, "code=500") {
		t.Errorf("Expected 'code=500' in output, got: %s", output)
	}
	if !strings.Contains(output, "message=\"internal error\"") {
		t.Errorf("Expected 'message=\"internal error\"' in output, got: %s", output)
	}
}

func TestLoggerVerboseBehavior(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		logFunc     func(*Logger)
		expected    string
		notExpected string
	}{
		{
			name:    "verbose mode shows debug",
			verbose: true,
			logFunc: func(l *Logger) {
				l.Debugf("debug info")
			},
			expected:    "DEBUG",
			notExpected: "",
		},
		{
			name:    "non-verbose mode hides debug",
			verbose: false,
			logFunc: func(l *Logger) {
				l.Debugf("debug info")
			},
			expected:    "",
			notExpected: "DEBUG",
		},
		{
			name:    "non-verbose mode shows info",
			verbose: false,
			logFunc: func(l *Logger) {
				l.Infof("info message")
			},
			expected:    "INFO",
			notExpected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLoggerWithWriter(tt.verbose, &buf)

			tt.logFunc(logger)

			output := buf.String()
			if tt.expected != "" && !strings.Contains(output, tt.expected) {
				t.Errorf("Expected '%s' in output, got: %s", tt.expected, output)
			}
			if tt.notExpected != "" && strings.Contains(output, tt.notExpected) {
				t.Errorf("Did not expect '%s' in output, got: %s", tt.notExpected, output)
			}
		})
	}
}
