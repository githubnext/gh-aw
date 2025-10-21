package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Logger represents a debug logger for a specific namespace
type Logger struct {
	namespace string
	enabled   bool
}

var (
	// Cache for compiled pattern checks to avoid recompiling on every logger creation
	patternCache     = make(map[string]bool)
	patternCacheLock sync.RWMutex

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
	enabled := isEnabled(namespace)
	return &Logger{
		namespace: namespace,
		enabled:   enabled,
	}
}

// Enabled returns whether this logger is enabled
func (l *Logger) Enabled() bool {
	return l.enabled
}

// Printf prints a formatted message if the logger is enabled
func (l *Logger) Printf(format string, args ...interface{}) {
	if !l.enabled {
		return
	}
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", l.namespace, message)
}

// Print prints a message if the logger is enabled
func (l *Logger) Print(args ...interface{}) {
	if !l.enabled {
		return
	}
	message := fmt.Sprint(args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", l.namespace, message)
}

// Println prints a message with a newline if the logger is enabled
func (l *Logger) Println(args ...interface{}) {
	if !l.enabled {
		return
	}
	message := fmt.Sprint(args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", l.namespace, message)
}

// LazyPrintf evaluates the lazy function only if the logger is enabled,
// then prints the result. This is useful for expensive string operations.
func (l *Logger) LazyPrintf(lazy func() string) {
	if !l.enabled {
		return
	}
	message := lazy()
	fmt.Fprintf(os.Stderr, "%s %s\n", l.namespace, message)
}

// isEnabled determines if a namespace should be enabled based on DEBUG environment variable
func isEnabled(namespace string) bool {
	if debugEnv == "" {
		return false
	}

	// Check cache first
	patternCacheLock.RLock()
	if enabled, found := patternCache[namespace]; found {
		patternCacheLock.RUnlock()
		return enabled
	}
	patternCacheLock.RUnlock()

	// Compute and cache the result
	enabled := computeEnabled(namespace)

	patternCacheLock.Lock()
	patternCache[namespace] = enabled
	patternCacheLock.Unlock()

	return enabled
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
