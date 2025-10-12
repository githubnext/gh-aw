package logging

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if logger.Logger == nil {
		t.Fatal("Logger.Logger is nil")
	}
}

func TestNewLoggerWithCategory(t *testing.T) {
	tests := []struct {
		name      string
		category  string
		envValue  string
		shouldLog bool
	}{
		{
			name:      "category enabled by default",
			category:  "compiler",
			envValue:  "",
			shouldLog: true,
		},
		{
			name:      "category enabled by filter",
			category:  "compiler",
			envValue:  "compiler,parser",
			shouldLog: true,
		},
		{
			name:      "category disabled by filter",
			category:  "compiler",
			envValue:  "parser,validator",
			shouldLog: false,
		},
		{
			name:      "all categories enabled",
			category:  "compiler",
			envValue:  "all",
			shouldLog: true,
		},
		{
			name:      "empty category always enabled",
			category:  "",
			envValue:  "compiler",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("GH_AW_LOG_FILTER", tt.envValue)
				defer os.Unsetenv("GH_AW_LOG_FILTER")
			} else {
				os.Unsetenv("GH_AW_LOG_FILTER")
			}

			var buf bytes.Buffer
			logger := NewLoggerWithWriterAndCategory(&buf, tt.category)

			if logger == nil {
				t.Fatal("NewLoggerWithCategory returned nil")
			}
			if logger.GetCategory() != tt.category {
				t.Errorf("GetCategory() = %v, want %v", logger.GetCategory(), tt.category)
			}

			// Test if logging works as expected
			logger.Infof("test message")

			output := buf.String()
			hasOutput := strings.Contains(output, "test message")

			if tt.shouldLog && !hasOutput {
				t.Errorf("Expected log output for category %q with filter %q, but got none", tt.category, tt.envValue)
			}
			if !tt.shouldLog && hasOutput {
				t.Errorf("Expected no log output for category %q with filter %q, but got: %s", tt.category, tt.envValue, output)
			}

			// Check if category is in output when set
			if tt.category != "" && hasOutput {
				if !strings.Contains(output, "category="+tt.category) {
					t.Errorf("Expected category=%s in output, got: %s", tt.category, output)
				}
			}
		})
	}
}

func TestNewLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf)

	if logger == nil {
		t.Fatal("NewLoggerWithWriter returned nil")
	}
	if logger.Logger == nil {
		t.Fatal("Logger.Logger is nil")
	}
}

func TestLoggerInfof(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf)

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
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf)

	logger.Debugf("debug message")

	output := buf.String()
	// Debug messages should not appear at INFO level
	if strings.Contains(output, "DEBUG") || strings.Contains(output, "debug message") {
		t.Errorf("Expected no debug output at INFO level, got: %s", output)
	}
}

func TestLoggerWarnf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf)

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
	logger := NewLoggerWithWriter(&buf)

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
	logger := NewLoggerWithWriter(&buf)

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
	logger := NewLoggerWithWriter(&buf)

	logger.DebugWithFields("debug structured", "field", "value")

	output := buf.String()
	// Debug messages should not appear at INFO level
	if strings.Contains(output, "DEBUG") || strings.Contains(output, "debug structured") {
		t.Errorf("Expected no debug output at INFO level, got: %s", output)
	}
}

func TestLoggerWarnWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf)

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
	logger := NewLoggerWithWriter(&buf)

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

func TestCategoryFiltering(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		category  string
		shouldLog bool
	}{
		{
			name:      "no filter - all enabled",
			filter:    "",
			category:  "compiler",
			shouldLog: true,
		},
		{
			name:      "filter matches category",
			filter:    "compiler",
			category:  "compiler",
			shouldLog: true,
		},
		{
			name:      "filter doesn't match category",
			filter:    "parser",
			category:  "compiler",
			shouldLog: false,
		},
		{
			name:      "filter has multiple categories including target",
			filter:    "parser,compiler,validator",
			category:  "compiler",
			shouldLog: true,
		},
		{
			name:      "filter has multiple categories not including target",
			filter:    "parser,validator",
			category:  "compiler",
			shouldLog: false,
		},
		{
			name:      "filter is 'all'",
			filter:    "all",
			category:  "compiler",
			shouldLog: true,
		},
		{
			name:      "filter is 'ALL' (case insensitive)",
			filter:    "ALL",
			category:  "compiler",
			shouldLog: true,
		},
		{
			name:      "filter with whitespace",
			filter:    " compiler , parser ",
			category:  "compiler",
			shouldLog: true,
		},
		{
			name:      "case insensitive matching",
			filter:    "COMPILER",
			category:  "compiler",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.filter != "" {
				os.Setenv("GH_AW_LOG_FILTER", tt.filter)
			} else {
				os.Unsetenv("GH_AW_LOG_FILTER")
			}
			defer os.Unsetenv("GH_AW_LOG_FILTER")

			var buf bytes.Buffer
			logger := NewLoggerWithWriterAndCategory(&buf, tt.category)

			logger.Infof("test message")

			output := buf.String()
			hasOutput := len(output) > 0 && strings.Contains(output, "test message")

			if tt.shouldLog && !hasOutput {
				t.Errorf("Expected log output with filter=%q category=%q, but got none", tt.filter, tt.category)
			}
			if !tt.shouldLog && hasOutput {
				t.Errorf("Expected no log output with filter=%q category=%q, but got: %s", tt.filter, tt.category, output)
			}
		})
	}
}
