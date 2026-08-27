//go:build !integration

package jsonutil_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/jsonutil"
)

// TestSpec_PublicAPI_MarshalCompactNoHTMLEscape validates the documented behavior
// of MarshalCompactNoHTMLEscape as described in the package README.md.
func TestSpec_PublicAPI_MarshalCompactNoHTMLEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name: "documented expression preserves html-sensitive characters",
			input: map[string]string{
				"expr": "${{ env.MCP_ENV == 'staging' && env.MCP_URL_STAGING || env.MCP_URL_PROD }}",
			},
			expected: `{"expr":"${{ env.MCP_ENV == 'staging' && env.MCP_URL_STAGING || env.MCP_URL_PROD }}"}`,
		},
		{
			name:     "compact output omits trailing newline for simple string",
			input:    map[string]string{"message": "hello"},
			expected: `{"message":"hello"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonutil.MarshalCompactNoHTMLEscape(tt.input)
			require.NoError(t, err, "unexpected error for: %s", tt.name)
			assert.Equal(t, tt.expected, result, "result mismatch for: %s", tt.name)
			assert.NotContains(t, result, "\n", "result should be compact for: %s", tt.name)
		})
	}
}

// TestSpec_PublicAPI_MarshalCompactNoHTMLEscape_Error validates the documented
// error-returning behavior when the input cannot be marshaled by encoding/json.
func TestSpec_PublicAPI_MarshalCompactNoHTMLEscape_Error(t *testing.T) {
	input := map[string]any{"bad": math.Inf(1)}

	result, err := jsonutil.MarshalCompactNoHTMLEscape(input)
	assert.Error(t, err, "should return error for unsupported JSON values")
	assert.Empty(t, result, "result should be empty when marshaling fails")
}
