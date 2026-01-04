// Package stringutil provides utility functions for working with strings.
package stringutil

import (
	"fmt"
	"strings"
)

// Truncate truncates a string to a maximum length, adding "..." if truncated.
// If maxLen is 3 or less, the string is truncated without "...".
//
// This is a general-purpose utility for truncating any string to a configurable
// length. For domain-specific workflow command identifiers with newline handling,
// see workflow.ShortenCommand instead.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// NormalizeWhitespace normalizes trailing whitespace and newlines to reduce spurious conflicts.
// It trims trailing whitespace from each line and ensures exactly one trailing newline.
func NormalizeWhitespace(content string) string {
	// Split into lines and trim trailing whitespace from each line
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Join back and ensure exactly one trailing newline if content is not empty
	normalized := strings.Join(lines, "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if len(normalized) > 0 {
		normalized += "\n"
	}

	return normalized
}

// ParseVersionValue converts version values of various types to strings.
// Supports string, int, int64, uint64, and float64 types.
// Returns empty string for unsupported types.
func ParseVersionValue(version any) string {
	switch v := version.(type) {
	case string:
		return v
	case int, int64, uint64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		return ""
	}
}
