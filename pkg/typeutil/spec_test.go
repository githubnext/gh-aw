//go:build !integration

package typeutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_ParseIntValue validates the documented behavior of
// ParseIntValue as described in the package README.md.
//
// Specification: Strictly parses numeric types (int, int64, uint64, float64)
// to int. Returns (value, true) on success and (0, false) for any unrecognized
// or non-numeric type.
func TestSpec_PublicAPI_ParseIntValue(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		wantValue int
		wantOK    bool
	}{
		{
			name:      "int input returns (value, true)",
			input:     42,
			wantValue: 42,
			wantOK:    true,
		},
		{
			name:      "int64 input returns (value, true)",
			input:     int64(99),
			wantValue: 99,
			wantOK:    true,
		},
		{
			name:      "uint64 input returns (value, true)",
			input:     uint64(7),
			wantValue: 7,
			wantOK:    true,
		},
		{
			name:      "float64 integer value returns (value, true)",
			input:     float64(10),
			wantValue: 10,
			wantOK:    true,
		},
		{
			name:      "string input returns (0, false)",
			input:     "42",
			wantValue: 0,
			wantOK:    false,
		},
		{
			name:      "nil input returns (0, false)",
			input:     nil,
			wantValue: 0,
			wantOK:    false,
		},
		{
			name:      "bool input returns (0, false)",
			input:     true,
			wantValue: 0,
			wantOK:    false,
		},
		{
			name:      "zero int returns (0, true)",
			input:     0,
			wantValue: 0,
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOK := ParseIntValue(tt.input)
			assert.Equal(t, tt.wantValue, gotValue,
				"ParseIntValue(%v) value mismatch", tt.input)
			assert.Equal(t, tt.wantOK, gotOK,
				"ParseIntValue(%v) ok flag mismatch", tt.input)
		})
	}
}

// TestSpec_PublicAPI_ParseBool validates the documented behavior of ParseBool
// as described in the package README.md.
//
// Specification: Extracts a boolean value from a map[string]any by key.
// Returns false if the map is nil, the key is absent, or the value is not a bool.
func TestSpec_PublicAPI_ParseBool(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected bool
	}{
		{
			name:     "true bool value returns true",
			m:        map[string]any{"enabled": true},
			key:      "enabled",
			expected: true,
		},
		{
			name:     "false bool value returns false",
			m:        map[string]any{"enabled": false},
			key:      "enabled",
			expected: false,
		},
		{
			name:     "nil map returns false",
			m:        nil,
			key:      "enabled",
			expected: false,
		},
		{
			name:     "absent key returns false",
			m:        map[string]any{"other": true},
			key:      "enabled",
			expected: false,
		},
		{
			name:     "non-bool value returns false",
			m:        map[string]any{"enabled": "yes"},
			key:      "enabled",
			expected: false,
		},
		{
			name:     "integer value returns false",
			m:        map[string]any{"enabled": 1},
			key:      "enabled",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBool(tt.m, tt.key)
			assert.Equal(t, tt.expected, result,
				"ParseBool(map, %q) should match documented behavior", tt.key)
		})
	}
}

// TestSpec_SafeOverflow_SafeUint64ToInt validates the documented behavior of
// SafeUint64ToInt as described in the package README.md.
//
// Specification: Converts uint64 to int, returning 0 if the value would
// overflow int.
func TestSpec_SafeOverflow_SafeUint64ToInt(t *testing.T) {
	t.Run("normal value converts correctly", func(t *testing.T) {
		result := SafeUint64ToInt(uint64(100))
		assert.Equal(t, 100, result,
			"SafeUint64ToInt(100) should return 100")
	})

	t.Run("zero converts to zero", func(t *testing.T) {
		result := SafeUint64ToInt(uint64(0))
		assert.Equal(t, 0, result,
			"SafeUint64ToInt(0) should return 0")
	})

	t.Run("overflow value returns 0 (documented defensive behavior)", func(t *testing.T) {
		// uint64 max overflows int on all supported platforms
		result := SafeUint64ToInt(math.MaxUint64)
		assert.Equal(t, 0, result,
			"SafeUint64ToInt(MaxUint64) should return 0 to prevent overflow panic")
	})
}

// TestSpec_SafeOverflow_SafeUintToInt validates the documented behavior of
// SafeUintToInt as described in the package README.md.
//
// Specification: Converts uint to int, returning 0 if the value would overflow
// int. Thin wrapper around SafeUint64ToInt.
func TestSpec_SafeOverflow_SafeUintToInt(t *testing.T) {
	t.Run("normal value converts correctly", func(t *testing.T) {
		result := SafeUintToInt(uint(42))
		assert.Equal(t, 42, result,
			"SafeUintToInt(42) should return 42")
	})

	t.Run("zero converts to zero", func(t *testing.T) {
		result := SafeUintToInt(uint(0))
		assert.Equal(t, 0, result,
			"SafeUintToInt(0) should return 0")
	})
}

// TestSpec_PublicAPI_ConvertToInt validates the documented behavior of
// ConvertToInt as described in the package README.md.
//
// Specification: Leniently converts any value to int, returning 0 on failure.
// Also handles string inputs via strconv.Atoi, making it suitable for
// heterogeneous sources.
func TestSpec_PublicAPI_ConvertToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{
			name:     "int input returns value",
			input:    55,
			expected: 55,
		},
		{
			name:     "int64 input returns value",
			input:    int64(77),
			expected: 77,
		},
		{
			name:     "float64 input returns int value",
			input:    float64(3),
			expected: 3,
		},
		{
			name:     "numeric string returns parsed value (documented behavior)",
			input:    "42",
			expected: 42,
		},
		{
			name:     "non-numeric string returns 0",
			input:    "not-a-number",
			expected: 0,
		},
		{
			name:     "nil returns 0",
			input:    nil,
			expected: 0,
		},
		{
			name:     "bool returns 0",
			input:    true,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToInt(tt.input)
			assert.Equal(t, tt.expected, result,
				"ConvertToInt(%v) should match documented behavior", tt.input)
		})
	}
}

// TestSpec_PublicAPI_ConvertToFloat validates the documented behavior of
// ConvertToFloat as described in the package README.md.
//
// Specification: Safely converts any value (float64, int, int64, string) to
// float64, returning 0 on failure.
func TestSpec_PublicAPI_ConvertToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{
			name:     "float64 input returns value",
			input:    float64(3.14),
			expected: 3.14,
		},
		{
			name:     "int input returns float value",
			input:    10,
			expected: 10.0,
		},
		{
			name:     "int64 input returns float value",
			input:    int64(20),
			expected: 20.0,
		},
		{
			name:     "numeric string returns parsed value",
			input:    "2.5",
			expected: 2.5,
		},
		{
			name:     "non-numeric string returns 0",
			input:    "not-a-float",
			expected: 0,
		},
		{
			name:     "nil returns 0",
			input:    nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToFloat(tt.input)
			assert.InDelta(t, tt.expected, result, 1e-9,
				"ConvertToFloat(%v) should match documented behavior", tt.input)
		})
	}
}
