package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger represents a debug logger for a specific namespace
type Logger struct {
	namespace string
	enabled   bool
	lastLog   time.Time
	mu        sync.Mutex
}

var (
	// DEBUG environment variable value, read once at initialization
	debugEnv = os.Getenv("DEBUG")
)

// New creates a new Logger for the given namespace.
// The enabled state is computed at construction time based on the DEBUG environment variable.
// DEBUG syntax follows https://www.npmjs.com/package/debug patterns:
//
//	DEBUG=*              - enables all loggers
//	DEBUG=namespace:*    - enables all loggers in a namespace
//	DEBUG=ns1,ns2        - enables specific namespaces
//	DEBUG=ns:*,-ns:skip  - enables namespace but excludes specific patterns
func New(namespace string) *Logger {
	enabled := computeEnabled(namespace)
	return &Logger{
		namespace: namespace,
		enabled:   enabled,
		lastLog:   time.Now(),
	}
}

// Enabled returns whether this logger is enabled
func (l *Logger) Enabled() bool {
	return l.enabled
}

// Printf prints a formatted message if the logger is enabled.
// A newline is always added at the end.
// Time diff since last log is displayed like the debug npm package.
func (l *Logger) Printf(format string, args ...interface{}) {
	if !l.enabled {
		return
	}
	l.mu.Lock()
	now := time.Now()
	diff := now.Sub(l.lastLog)
	l.lastLog = now
	l.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s +%s\n", l.namespace, message, formatDuration(diff))
}

// Print prints a message if the logger is enabled.
// A newline is always added at the end.
// Time diff since last log is displayed like the debug npm package.
func (l *Logger) Print(args ...interface{}) {
	if !l.enabled {
		return
	}
	l.mu.Lock()
	now := time.Now()
	diff := now.Sub(l.lastLog)
	l.lastLog = now
	l.mu.Unlock()

	message := fmt.Sprint(args...)
	fmt.Fprintf(os.Stderr, "%s %s +%s\n", l.namespace, message, formatDuration(diff))
}

// formatDuration formats a duration for display like the debug npm package
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// computeEnabled computes whether a namespace matches the DEBUG patterns
func computeEnabled(namespace string) bool {
	patterns := strings.Split(debugEnv, ",")

	enabled := false

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)

		// Handle exclusion patterns (starting with -)
		if strings.HasPrefix(pattern, "-") {
			excludePattern := strings.TrimPrefix(pattern, "-")
			if matchPattern(namespace, excludePattern) {
				return false // Exclusions take precedence
			}
			continue
		}

		// Check if pattern matches
		if matchPattern(namespace, pattern) {
			enabled = true
		}
	}

	return enabled
}

// matchPattern checks if a namespace matches a pattern
// Supports wildcards (*) for pattern matching
func matchPattern(namespace, pattern string) bool {
	// Exact match or wildcard-all
	if pattern == "*" || pattern == namespace {
		return true
	}

	// Pattern with wildcard
	if strings.Contains(pattern, "*") {
		// Replace * with .* for regex-like matching, but keep it simple
		// Convert pattern to prefix/suffix matching
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			return strings.HasPrefix(namespace, prefix)
		}
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(namespace, suffix)
		}
		// Middle wildcard: split and check both parts
		parts := strings.SplitN(pattern, "*", 2)
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]
			return strings.HasPrefix(namespace, prefix) && strings.HasSuffix(namespace, suffix)
		}
	}

	return false
}
