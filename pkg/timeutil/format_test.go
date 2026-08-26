//go:build !integration

package timeutil

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	t.Parallel()
	// assert.Equal is intentional for this file's table-driven tests:
	// each subtest has one assertion, and using assert consistently keeps
	// the style aligned across all duration-format tests here.
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		// Nanosecond range
		{
			name:     "nanoseconds",
			duration: 500 * time.Nanosecond,
			expected: "500ns",
		},
		{
			name:     "999 nanoseconds",
			duration: 999 * time.Nanosecond,
			expected: "999ns",
		},
		// Microsecond range
		{
			name:     "microseconds",
			duration: 500 * time.Microsecond,
			expected: "500µs",
		},
		{
			name:     "999 microseconds",
			duration: 999 * time.Microsecond,
			expected: "999µs",
		},
		// Millisecond range
		{
			name:     "milliseconds",
			duration: 250 * time.Millisecond,
			expected: "250ms",
		},
		{
			name:     "999 milliseconds",
			duration: 999 * time.Millisecond,
			expected: "999ms",
		},
		// Second range
		{
			name:     "seconds",
			duration: 5 * time.Second,
			expected: "5.0s",
		},
		{
			name:     "seconds with decimal",
			duration: 5500 * time.Millisecond,
			expected: "5.5s",
		},
		{
			name:     "59 seconds",
			duration: 59 * time.Second,
			expected: "59.0s",
		},
		// Minute range
		{
			name:     "1 minute",
			duration: time.Minute,
			expected: "1.0m",
		},
		{
			name:     "minutes with decimal",
			duration: 90 * time.Second,
			expected: "1.5m",
		},
		{
			name:     "59 minutes",
			duration: 59 * time.Minute,
			expected: "59.0m",
		},
		// Hour range
		{
			name:     "1 hour",
			duration: time.Hour,
			expected: "1.0h",
		},
		{
			name:     "hours with decimal",
			duration: 90 * time.Minute,
			expected: "1.5h",
		},
		{
			name:     "multiple hours",
			duration: 5*time.Hour + 30*time.Minute,
			expected: "5.5h",
		},
		// Edge cases
		{
			name:     "zero duration",
			duration: 0,
			expected: "0ns",
		},
		{
			name:     "1 nanosecond",
			duration: 1 * time.Nanosecond,
			expected: "1ns",
		},
		{
			name:     "just under microsecond",
			duration: 999 * time.Nanosecond,
			expected: "999ns",
		},
		{
			name:     "exactly 1 microsecond",
			duration: 1 * time.Microsecond,
			expected: "1µs",
		},
		{
			name:     "just under millisecond",
			duration: 999 * time.Microsecond,
			expected: "999µs",
		},
		{
			name:     "exactly 1 millisecond",
			duration: 1 * time.Millisecond,
			expected: "1ms",
		},
		{
			name:     "just under second",
			duration: 999 * time.Millisecond,
			expected: "999ms",
		},
		{
			name:     "exactly 1 second",
			duration: 1 * time.Second,
			expected: "1.0s",
		},
		{
			name:     "just under minute rolls over",
			duration: 59*time.Second + 999*time.Millisecond,
			expected: "1.0m",
		},
		{
			name:     "exactly 1 minute",
			duration: 1 * time.Minute,
			expected: "1.0m",
		},
		{
			name:     "just under hour rolls over",
			duration: 59*time.Minute + 59*time.Second,
			expected: "1.0h",
		},
		{
			name:     "exactly 1 hour",
			duration: 1 * time.Hour,
			expected: "1.0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result, "FormatDuration(%v) mismatch", tt.duration)
		})
	}
}

func TestFormatDurationMs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ms       int
		expected string
	}{
		// Millisecond range
		{
			name:     "zero milliseconds",
			ms:       0,
			expected: "0ms",
		},
		{
			name:     "sub-second milliseconds",
			ms:       500,
			expected: "500ms",
		},
		{
			name:     "just under one second",
			ms:       999,
			expected: "999ms",
		},
		// Second range
		{
			name:     "exactly one second",
			ms:       1000,
			expected: "1.0s",
		},
		{
			name:     "seconds with decimal",
			ms:       1500,
			expected: "1.5s",
		},
		{
			name: "just under one minute rolls over to minutes",
			ms:   59999,
			// 59_999 ms rounds to 60.0s, which rolls over into the minute
			// range instead of rendering the "60.0s" artifact.
			expected: "1.0m",
		},
		// Minute range
		{
			name:     "exactly one minute",
			ms:       60000,
			expected: "1.0m",
		},
		{
			name:     "one minute thirty seconds",
			ms:       90000,
			expected: "1.5m",
		},
		{
			name:     "multi-minute composition",
			ms:       125000,
			expected: "2.1m",
		},
		{
			name:     "multi-hour value rolls over to hours",
			ms:       3_600_000,
			expected: "1.0h",
		},
		// Negative input is passed through the millisecond branch as-is.
		{
			name:     "negative milliseconds",
			ms:       -500,
			expected: "-500ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatDurationMs(tt.ms)
			assert.Equal(t, tt.expected, result, "FormatDurationMs(%d) mismatch", tt.ms)
		})
	}
}

// TestFormatDurationMsOverflowGuard verifies that millisecond values beyond
// time.Duration's representable nanosecond range are formatted directly in
// hours instead of silently overflowing. Skipped on platforms where int is
// narrower than 64 bits, since the tested values cannot be represented there.
func TestFormatDurationMsOverflowGuard(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("requires a 64-bit int to represent the tested values")
	}

	var beyondRange int64 = 9_223_372_036_855
	atBoundary := math.MaxInt64 / int64(time.Millisecond)

	tests := []struct {
		name     string
		ms       int64
		expected string
	}{
		{
			name:     "value beyond time.Duration range formats in hours",
			ms:       beyondRange,
			expected: "2562047.8h",
		},
		{
			name:     "value at time.Duration boundary uses standard path",
			ms:       atBoundary,
			expected: "2562047.8h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatDurationMs(int(tt.ms))
			assert.Equal(t, tt.expected, result, "FormatDurationMs(%d) mismatch", tt.ms)
		})
	}
}

func TestFormatDurationNs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ns       int64
		expected string
	}{
		// Zero / negative guard
		{
			name:     "zero returns em dash",
			ns:       0,
			expected: "—",
		},
		{
			name:     "negative returns em dash",
			ns:       -1,
			expected: "—",
		},
		{
			name:     "large negative returns em dash",
			ns:       -90_000_000_000,
			expected: "—",
		},
		// Rounding boundaries
		{
			name:     "just under half a second stays in milliseconds",
			ns:       499_999_999,
			expected: "499ms",
		},
		{
			name:     "exactly half a second stays in milliseconds",
			ns:       500_000_000,
			expected: "500ms",
		},
		{
			name:     "one and a half seconds",
			ns:       1_500_000_000,
			expected: "1.5s",
		},
		{
			name:     "just under one and a half seconds",
			ns:       1_499_999_999,
			expected: "1.5s",
		},
		// Composition
		{
			name:     "two seconds",
			ns:       2_000_000_000,
			expected: "2.0s",
		},
		{
			name:     "one minute thirty seconds",
			ns:       90_000_000_000,
			expected: "1.5m",
		},
		{
			name:     "multi-hour duration",
			ns:       7_265_000_000_000,
			expected: "2.0h",
		},
		{
			name:     "multi-hour duration with sub-second rounding",
			ns:       3_600_500_000_000,
			expected: "1.0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatDurationNs(tt.ns)
			assert.Equal(t, tt.expected, result, "FormatDurationNs(%d) mismatch", tt.ns)
		})
	}
}
