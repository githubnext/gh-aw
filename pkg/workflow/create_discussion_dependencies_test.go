//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDiscussionsConfigDefaultExpiration(t *testing.T) {
	tests := []struct {
		name            string
		config          map[string]any
		expectedExpires int
	}{
		{
			name: "No expires field - should default to 7 days (168 hours)",
			config: map[string]any{
				"create-discussion": map[string]any{
					"category": "general",
				},
			},
			expectedExpires: 168, // 7 days = 168 hours
		},
		{
			name: "Explicit expires integer - should use provided value",
			config: map[string]any{
				"create-discussion": map[string]any{
					"category": "general",
					"expires":  14, // 14 days
				},
			},
			expectedExpires: 336, // 14 days = 336 hours
		},
		{
			name: "Explicit expires string format - should use provided value",
			config: map[string]any{
				"create-discussion": map[string]any{
					"category": "general",
					"expires":  "7d",
				},
			},
			expectedExpires: 168, // 7 days = 168 hours
		},
		{
			name: "Explicit expires zero - should use default",
			config: map[string]any{
				"create-discussion": map[string]any{
					"category": "general",
					"expires":  0,
				},
			},
			expectedExpires: 168, // Should default to 7 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			result := compiler.parseDiscussionsConfig(tt.config)

			require.NotNil(t, result, "parseDiscussionsConfig should return a config")
			assert.Equal(t, tt.expectedExpires, result.Expires, "Expires value should match expected")
		})
	}
}
