package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// captureStderr captures stderr output during test execution
func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		debugEnv  string
		namespace string
		enabled   bool
	}{
		{
			name:      "empty DEBUG disables all loggers",
			debugEnv:  "",
			namespace: "test:logger",
			enabled:   false,
		},
		{
			name:      "wildcard enables all loggers",
			debugEnv:  "*",
			namespace: "test:logger",
			enabled:   true,
		},
		{
			name:      "exact match enables logger",
			debugEnv:  "test:logger",
			namespace: "test:logger",
			enabled:   true,
		},
		{
			name:      "exact match different namespace disabled",
			debugEnv:  "test:logger",
			namespace: "other:logger",
			enabled:   false,
		},
		{
			name:      "namespace wildcard enables matching loggers",
			debugEnv:  "test:*",
			namespace: "test:logger",
			enabled:   true,
		},
		{
			name:      "namespace wildcard matches deeply nested",
			debugEnv:  "test:*",
			namespace: "test:sub:logger",
			enabled:   true,
		},
		{
			name:      "namespace wildcard does not match different prefix",
			debugEnv:  "test:*",
			namespace: "other:logger",
			enabled:   false,
		},
		{
			name:      "multiple patterns with comma",
			debugEnv:  "test:*,other:*",
			namespace: "test:logger",
			enabled:   true,
		},
		{
			name:      "multiple patterns second matches",
			debugEnv:  "test:*,other:*",
			namespace: "other:logger",
			enabled:   true,
		},
		{
			name:      "exclusion pattern disables specific logger",
			debugEnv:  "test:*,-test:skip",
			namespace: "test:skip",
			enabled:   false,
		},
		{
			name:      "exclusion does not affect other loggers",
			debugEnv:  "test:*,-test:skip",
			namespace: "test:logger",
			enabled:   true,
		},
		{
			name:      "exclusion with wildcard",
			debugEnv:  "*,-test:*",
			namespace: "test:logger",
			enabled:   false,
		},
		{
			name:      "exclusion with wildcard allows others",
			debugEnv:  "*,-test:*",
			namespace: "other:logger",
			enabled:   true,
		},
		{
			name:      "suffix wildcard",
			debugEnv:  "*:logger",
			namespace: "test:logger",
			enabled:   true,
		},
		{
			name:      "suffix wildcard no match",
			debugEnv:  "*:logger",
			namespace: "test:other",
			enabled:   false,
		},
		{
			name:      "middle wildcard",
			debugEnv:  "test:*:end",
			namespace: "test:middle:end",
			enabled:   true,
		},
		{
			name:      "middle wildcard no match prefix",
			debugEnv:  "test:*:end",
			namespace: "other:middle:end",
			enabled:   false,
		},
		{
			name:      "middle wildcard no match suffix",
			debugEnv:  "test:*:end",
			namespace: "test:middle:other",
			enabled:   false,
		},
		{
			name:      "spaces in patterns are trimmed",
			debugEnv:  "test:* , other:*",
			namespace: "other:logger",
			enabled:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cache and set environment for this test
			patternCacheLock.Lock()
			patternCache = make(map[string]bool)
			debugEnv = tt.debugEnv
			patternCacheLock.Unlock()

			logger := New(tt.namespace)
			if logger.Enabled() != tt.enabled {
				t.Errorf("New(%q) with DEBUG=%q: enabled = %v, want %v",
					tt.namespace, tt.debugEnv, logger.Enabled(), tt.enabled)
			}
		})
	}
}

func TestLogger_Printf(t *testing.T) {
	tests := []struct {
		name      string
		debugEnv  string
		namespace string
		format    string
		args      []interface{}
		wantLog   bool
	}{
		{
			name:      "enabled logger prints",
			debugEnv:  "*",
			namespace: "test:logger",
			format:    "hello %s",
			args:      []interface{}{"world"},
			wantLog:   true,
		},
		{
			name:      "disabled logger does not print",
			debugEnv:  "",
			namespace: "test:logger",
			format:    "hello %s",
			args:      []interface{}{"world"},
			wantLog:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cache and set environment
			patternCacheLock.Lock()
			patternCache = make(map[string]bool)
			debugEnv = tt.debugEnv
			patternCacheLock.Unlock()

			logger := New(tt.namespace)

			output := captureStderr(func() {
				logger.Printf(tt.format, tt.args...)
			})

			if tt.wantLog {
				if output == "" {
					t.Errorf("Printf() should have logged but got empty output")
				}
				if !strings.Contains(output, tt.namespace) {
					t.Errorf("Printf() output should contain namespace %q, got %q", tt.namespace, output)
				}
				expectedMessage := "hello world"
				if !strings.Contains(output, expectedMessage) {
					t.Errorf("Printf() output should contain %q, got %q", expectedMessage, output)
				}
			} else {
				if output != "" {
					t.Errorf("Printf() should not have logged but got %q", output)
				}
			}
		})
	}
}

func TestLogger_Print(t *testing.T) {
	// Reset cache and set environment
	patternCacheLock.Lock()
	patternCache = make(map[string]bool)
	debugEnv = "*"
	patternCacheLock.Unlock()

	logger := New("test:print")

	output := captureStderr(func() {
		logger.Print("hello", " ", "world")
	})

	if !strings.Contains(output, "test:print") {
		t.Errorf("Print() output should contain namespace, got %q", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("Print() output should contain message, got %q", output)
	}
}

func TestLogger_Println(t *testing.T) {
	// Reset cache and set environment
	patternCacheLock.Lock()
	patternCache = make(map[string]bool)
	debugEnv = "*"
	patternCacheLock.Unlock()

	logger := New("test:println")

	output := captureStderr(func() {
		logger.Println("hello world")
	})

	if !strings.Contains(output, "test:println") {
		t.Errorf("Println() output should contain namespace, got %q", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("Println() output should contain message, got %q", output)
	}
}

func TestLogger_LazyPrintf(t *testing.T) {
	tests := []struct {
		name         string
		debugEnv     string
		namespace    string
		shouldInvoke bool
	}{
		{
			name:         "enabled logger invokes lazy function",
			debugEnv:     "*",
			namespace:    "test:lazy",
			shouldInvoke: true,
		},
		{
			name:         "disabled logger does not invoke lazy function",
			debugEnv:     "",
			namespace:    "test:lazy",
			shouldInvoke: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cache and set environment
			patternCacheLock.Lock()
			patternCache = make(map[string]bool)
			debugEnv = tt.debugEnv
			patternCacheLock.Unlock()

			logger := New(tt.namespace)

			invoked := false
			output := captureStderr(func() {
				logger.LazyPrintf(func() string {
					invoked = true
					return "lazy message"
				})
			})

			if invoked != tt.shouldInvoke {
				t.Errorf("LazyPrintf() lazy function invoked = %v, want %v", invoked, tt.shouldInvoke)
			}

			if tt.shouldInvoke {
				if !strings.Contains(output, "lazy message") {
					t.Errorf("LazyPrintf() output should contain lazy message, got %q", output)
				}
			} else {
				if output != "" {
					t.Errorf("LazyPrintf() should not have logged but got %q", output)
				}
			}
		})
	}
}

func TestLogger_EnabledCaching(t *testing.T) {
	// Reset cache and set environment
	patternCacheLock.Lock()
	patternCache = make(map[string]bool)
	debugEnv = "test:*"
	patternCacheLock.Unlock()

	// Create first logger
	logger1 := New("test:cache")
	if !logger1.Enabled() {
		t.Error("First logger should be enabled")
	}

	// Create second logger with same namespace - should use cache
	logger2 := New("test:cache")
	if !logger2.Enabled() {
		t.Error("Second logger should be enabled (from cache)")
	}

	// Verify cache was used by checking cache size
	patternCacheLock.RLock()
	if len(patternCache) != 1 {
		t.Errorf("Cache should have 1 entry, got %d", len(patternCache))
	}
	patternCacheLock.RUnlock()
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		pattern   string
		want      bool
	}{
		{"exact match", "test:logger", "test:logger", true},
		{"no match", "test:logger", "other:logger", false},
		{"wildcard all", "test:logger", "*", true},
		{"prefix wildcard", "test:logger", "test:*", true},
		{"prefix wildcard no match", "test:logger", "other:*", false},
		{"suffix wildcard", "test:logger", "*:logger", true},
		{"suffix wildcard no match", "test:logger", "*:other", false},
		{"middle wildcard", "test:middle:logger", "test:*:logger", true},
		{"middle wildcard no match prefix", "other:middle:logger", "test:*:logger", false},
		{"middle wildcard no match suffix", "test:middle:other", "test:*:logger", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.namespace, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.namespace, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestComputeEnabled(t *testing.T) {
	tests := []struct {
		name      string
		debugEnv  string
		namespace string
		want      bool
	}{
		{"single pattern match", "test:*", "test:logger", true},
		{"single pattern no match", "test:*", "other:logger", false},
		{"multiple patterns first match", "test:*,other:*", "test:logger", true},
		{"multiple patterns second match", "test:*,other:*", "other:logger", true},
		{"multiple patterns no match", "test:*,other:*", "third:logger", false},
		{"exclusion disables", "test:*,-test:skip", "test:skip", false},
		{"exclusion allows others", "test:*,-test:skip", "test:logger", true},
		{"exclusion wildcard", "*,-test:*", "test:logger", false},
		{"exclusion wildcard allows", "*,-test:*", "other:logger", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set DEBUG for this test
			debugEnv = tt.debugEnv
			got := computeEnabled(tt.namespace)
			if got != tt.want {
				t.Errorf("computeEnabled(%q) with DEBUG=%q = %v, want %v",
					tt.namespace, tt.debugEnv, got, tt.want)
			}
		})
	}
}
