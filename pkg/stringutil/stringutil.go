// Package stringutil provides utility functions for working with strings.
package stringutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var stringutilLog = logger.New("stringutil:stringutil")

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
		stringutilLog.Printf("Truncate: hard-cut at maxLen=%d (no ellipsis)", maxLen)
		return s[:maxLen]
	}
	stringutilLog.Printf("Truncate: shortened from %d to %d chars", len(s), maxLen)
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
	if normalized != "" {
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
		stringutilLog.Printf("ParseVersionValue: unsupported type %T, returning empty string", version)
		return ""
	}
}

// FormatList formats a slice of strings as a natural-language comma-separated list
// with an Oxford comma and "and" before the final item.
//
// Examples:
//
//	FormatList([]string{})              // returns ""
//	FormatList([]string{"a"})           // returns "a"
//	FormatList([]string{"a", "b"})      // returns "a and b"
//	FormatList([]string{"a", "b", "c"}) // returns "a, b, and c"
func FormatList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

// NormalizeLeadingWhitespace removes consistent leading whitespace from all lines
// of a multi-line string. It finds the minimum indentation across all non-empty
// lines and strips that many leading whitespace characters (spaces or tabs) from
// every line.
//
// This is useful for cleaning up content generated with extra indentation,
// such as heredoc bodies.
func NormalizeLeadingWhitespace(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Find minimum leading whitespace (excluding empty lines)
	minLeading := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // Skip empty lines
		}
		leading := len(line) - len(strings.TrimLeft(line, " \t"))
		if minLeading == -1 || leading < minLeading {
			minLeading = leading
		}
	}

	// If no content or no leading whitespace, return as-is
	if minLeading <= 0 {
		return content
	}

	stringutilLog.Printf("NormalizeLeadingWhitespace: stripping %d leading whitespace chars from %d lines", minLeading, len(lines))

	// Remove the minimum leading whitespace from all lines
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		if strings.TrimSpace(line) == "" {
			// Keep empty lines as empty
			result.WriteString("")
		} else if len(line) >= minLeading {
			// Remove leading whitespace
			result.WriteString(line[minLeading:])
		} else {
			result.WriteString(line)
		}
	}

	return result.String()
}

// IsPositiveInteger checks if a string is a positive integer.
// Returns true for strings like "1", "123", "999" but false for:
//   - Zero ("0")
//   - Negative numbers ("-5")
//   - Numbers with leading zeros ("007")
//   - Floating point numbers ("3.14")
//   - Non-numeric strings ("abc")
//   - Empty strings ("")
func IsPositiveInteger(s string) bool {
	// Must not be empty
	if s == "" {
		return false
	}

	// Must not have leading zeros (except "0" itself, but that's not positive)
	if len(s) > 1 && s[0] == '0' {
		return false
	}

	// Must be numeric and > 0
	num, err := strconv.ParseInt(s, 10, 64)
	return err == nil && num > 0
}
