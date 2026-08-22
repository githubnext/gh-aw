// Package stringutil provides utility functions for working with strings.
package stringutil

import (
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
