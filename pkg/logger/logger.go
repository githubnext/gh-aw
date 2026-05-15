package logger

import (
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"sync"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/github/gh-aw/pkg/styles"
	"github.com/github/gh-aw/pkg/timeutil"
)

// Logger represents a debug logger for a specific namespace.
type Logger struct {
	namespace string
	enabled   bool
	lastLog   time.Time
	mu        sync.Mutex
	label     string
}

var (
	// DEBUG environment variable value, read once at initialization.
	// If DEBUG is not set but ACTIONS_RUNNER_DEBUG=true, all loggers are enabled.
	debugEnv = initDebugEnv()

	// DEBUG_COLORS environment variable to control color output.
	debugColors = os.Getenv("DEBUG_COLORS") != "0"

	// Color palette for namespace coloring, using adaptive styles.
	colorPalette = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(styles.ColorInfo),
		lipgloss.NewStyle().Foreground(styles.ColorSuccess),
		lipgloss.NewStyle().Foreground(styles.ColorWarning),
		lipgloss.NewStyle().Foreground(styles.ColorPurple),
		lipgloss.NewStyle().Foreground(styles.ColorYellow),
		lipgloss.NewStyle().Foreground(styles.ColorError),
		lipgloss.NewStyle().Foreground(styles.ColorComment),
		lipgloss.NewStyle().Foreground(styles.ColorForeground),
		lipgloss.NewStyle().Foreground(styles.ColorBorder),
		lipgloss.NewStyle().Foreground(styles.ColorInfo),
		lipgloss.NewStyle().Foreground(styles.ColorSuccess),
		lipgloss.NewStyle().Foreground(styles.ColorPurple),
	}
)

// initDebugEnv resolves the effective debug pattern.
// If DEBUG is set, it takes precedence. Otherwise, if ACTIONS_RUNNER_DEBUG=true,
// all loggers are enabled (equivalent to DEBUG=*).
func initDebugEnv() string {
	if d := os.Getenv("DEBUG"); d != "" {
		return d
	}
	if os.Getenv("ACTIONS_RUNNER_DEBUG") == "true" {
		return "*"
	}
	return ""
}

// New creates a new Logger for the given namespace.
// The enabled state is computed at construction time based on the DEBUG environment variable.
// DEBUG syntax follows https://www.npmjs.com/package/debug patterns:
//
//	DEBUG=*              - enables all loggers
//	DEBUG=namespace:*    - enables all loggers in a namespace
//	DEBUG=ns1,ns2        - enables specific namespaces
//	DEBUG=ns:*,-ns:skip  - enables namespace but excludes specific patterns
//
// Colors are automatically assigned to each namespace if DEBUG_COLORS != "0".
func New(namespace string) *Logger {
	enabled := computeEnabled(namespace)
	label := selectNamespaceLabel(namespace)
	return &Logger{
		namespace: namespace,
		enabled:   enabled,
		lastLog:   time.Now(),
		label:     label,
	}
}

// selectNamespaceLabel renders the namespace label with a hash-selected style.
func selectNamespaceLabel(namespace string) string {
	if !debugColors {
		return namespace
	}

	// Use FNV-1a hash for consistent color assignment
	h := fnv.New32a()
	// hash.Hash.Write never returns an error in practice, but check to satisfy gosec G104
	if _, err := h.Write([]byte(namespace)); err != nil {
		// Return plain namespace (no color) if write somehow fails
		return namespace
	}
	hash := h.Sum32()

	// Select color from palette based on hash and render namespace with it.
	return colorPalette[hash%uint32(len(colorPalette))].Render(namespace)
}

// Enabled returns whether this logger is enabled
func (l *Logger) Enabled() bool {
	return l.enabled
}

// Printf prints a formatted message if the logger is enabled.
// A newline is always added at the end.
// Time diff since last log is displayed like the debug npm package.
func (l *Logger) Printf(format string, args ...any) {
	if !l.enabled {
		return
	}
	l.mu.Lock()
	now := time.Now()
	diff := now.Sub(l.lastLog)
	l.lastLog = now
	l.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	lipgloss.Fprintf(os.Stderr, "%s %s +%s\n", l.label, message, timeutil.FormatDuration(diff))
}

// Print prints a message if the logger is enabled.
// A newline is always added at the end.
// Time diff since last log is displayed like the debug npm package.
func (l *Logger) Print(args ...any) {
	if !l.enabled {
		return
	}
	l.mu.Lock()
	now := time.Now()
	diff := now.Sub(l.lastLog)
	l.lastLog = now
	l.mu.Unlock()

	message := fmt.Sprint(args...)
	lipgloss.Fprintf(os.Stderr, "%s %s +%s\n", l.label, message, timeutil.FormatDuration(diff))
}

// computeEnabled computes whether a namespace matches the DEBUG patterns
func computeEnabled(namespace string) bool {
	patterns := strings.Split(debugEnv, ",")

	enabled := false

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)

		// Handle exclusion patterns (starting with -)
		if after, ok := strings.CutPrefix(pattern, "-"); ok {
			excludePattern := after
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
		if before, ok := strings.CutSuffix(pattern, "*"); ok {
			prefix := before
			return strings.HasPrefix(namespace, prefix)
		}
		if after, ok := strings.CutPrefix(pattern, "*"); ok {
			suffix := after
			return strings.HasSuffix(namespace, suffix)
		}
		// Middle wildcard: split and check both parts
		parts := strings.SplitN(pattern, "*", 2)
		if len(parts) == 2 {
			prefix, suffix := parts[0], parts[1]
			return len(namespace) >= len(prefix)+len(suffix) &&
				strings.HasPrefix(namespace, prefix) &&
				strings.HasSuffix(namespace, suffix)
		}
	}

	return false
}
