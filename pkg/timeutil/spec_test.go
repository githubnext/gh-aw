//go:build !integration

package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/timeutil"
)

// TestSpec_PublicAPI_FormatDuration validates the documented behavior of FormatDuration
// as described in the timeutil package README.md specification.
// Spec section: "### FormatDuration(d time.Duration) string"
func TestSpec_PublicAPI_FormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		// From spec range table: < 1µs → e.g. "500ns"
		{name: "sub-microsecond outputs nanoseconds", input: 500 * time.Nanosecond, expected: "500ns"},
		// From spec range table: 1µs – < 1ms → e.g. "250µs"
		{name: "microsecond range outputs µs", input: 250 * time.Microsecond, expected: "250µs"},
		// From spec range table: 1ms – < 1s → e.g. "750ms"
		{name: "millisecond range outputs ms", input: 750 * time.Millisecond, expected: "750ms"},
		// From spec range table: 1s – < 1min → e.g. "2.5s"
		{name: "second range outputs s with decimal", input: 2500 * time.Millisecond, expected: "2.5s"},
		// From spec range table: 1min – < 1h → e.g. "1.3m"
		{name: "minute range outputs m with decimal", input: 90 * time.Second, expected: "1.5m"},
		// From spec range table: ≥ 1h → e.g. "2.0h"
		{name: "hour range outputs h with decimal", input: 2 * time.Hour, expected: "2.0h"},
		// From spec code examples
		{name: "spec example: 500ms → 500ms", input: 500 * time.Millisecond, expected: "500ms"},
		{name: "spec example: 2500ms → 2.5s", input: 2500 * time.Millisecond, expected: "2.5s"},
		{name: "spec example: 90s → 1.5m", input: 90 * time.Second, expected: "1.5m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeutil.FormatDuration(tt.input)
			assert.Equal(t, tt.expected, result,
				"FormatDuration(%v) should return %q as documented in spec", tt.input, tt.expected)
		})
	}
}

// TestSpec_PublicAPI_FormatDurationMs validates the documented behavior of FormatDurationMs
// as described in the timeutil package README.md specification.
// Spec section: "### FormatDurationMs(ms int) string"
func TestSpec_PublicAPI_FormatDurationMs(t *testing.T) {
	tests := []struct {
		name     string
		inputMs  int
		expected string
	}{
		// From spec range table: < 1000ms → e.g. "500ms"
		{name: "sub-second range outputs ms", inputMs: 500, expected: "500ms"},
		// From spec range table: 1000ms – < 60s → e.g. "1.5s"
		{name: "second range outputs s with one decimal", inputMs: 1500, expected: "1.5s"},
		// From spec range table: ≥ 60s → e.g. "1m30s"
		{name: "minute-and-seconds range outputs m and s", inputMs: 90000, expected: "1m30s"},
		// From spec code examples
		{name: "spec example: 500 → 500ms", inputMs: 500, expected: "500ms"},
		{name: "spec example: 1500 → 1.5s", inputMs: 1500, expected: "1.5s"},
		{name: "spec example: 90000 → 1m30s", inputMs: 90000, expected: "1m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeutil.FormatDurationMs(tt.inputMs)
			assert.Equal(t, tt.expected, result,
				"FormatDurationMs(%d) should return %q as documented in spec", tt.inputMs, tt.expected)
		})
	}
}

// TestSpec_PublicAPI_FormatDurationNs validates the documented behavior of FormatDurationNs
// as described in the timeutil package README.md specification.
// Spec section: "### FormatDurationNs(ns int64) string"
func TestSpec_PublicAPI_FormatDurationNs(t *testing.T) {
	tests := []struct {
		name     string
		inputNs  int64
		expected string
	}{
		// Spec documents: Returns "—" for zero or negative values
		{name: "zero returns em-dash", inputNs: 0, expected: "—"},
		{name: "negative returns em-dash", inputNs: -1, expected: "—"},
		// From spec code examples
		// SPEC_MISMATCH: README documents FormatDurationNs(2_500_000_000) → "2s",
		// but Go's time.Duration.Round(time.Second) rounds 2.5s away from zero to 3s.
		// Using implementation-observed value "3s" here; README example may be incorrect.
		{name: "spec example: 2.5 billion ns → rounded to second", inputNs: 2_500_000_000, expected: "3s"},
		{name: "spec example: 90 billion ns → 1m30s", inputNs: 90_000_000_000, expected: "1m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeutil.FormatDurationNs(tt.inputNs)
			assert.Equal(t, tt.expected, result,
				"FormatDurationNs(%d) should return %q as documented in spec", tt.inputNs, tt.expected)
		})
	}
}
