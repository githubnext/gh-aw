//go:build !integration

package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersionValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  any
		expected string
	}{
		// String versions
		{
			name:     "string version",
			version:  "v1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "numeric string",
			version:  "123",
			expected: "123",
		},
		{
			name:     "empty string",
			version:  "",
			expected: "",
		},
		// Integer versions
		{
			name:     "int version",
			version:  42,
			expected: "42",
		},
		{
			name:     "int64 version",
			version:  int64(100),
			expected: "100",
		},
		{
			name:     "uint64 version",
			version:  uint64(999),
			expected: "999",
		},
		// Float versions
		{
			name:     "float64 simple",
			version:  float64(1.5),
			expected: "1.5",
		},
		{
			name:     "float64 whole number",
			version:  float64(2.0),
			expected: "2",
		},
		{
			name:     "float64 with precision",
			version:  float64(1.234),
			expected: "1.234",
		},
		// Unsupported types
		{
			name:     "nil",
			version:  nil,
			expected: "",
		},
		{
			name:     "bool",
			version:  true,
			expected: "",
		},
		{
			name:     "slice",
			version:  []string{"1", "2"},
			expected: "",
		},
		{
			name:     "map",
			version:  map[string]string{"version": "1.0"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ParseVersionValue(tt.version)
			assert.Equal(t, tt.expected, result, "ParseVersionValue(%v) should return normalized string representation", tt.version)
		})
	}
}
