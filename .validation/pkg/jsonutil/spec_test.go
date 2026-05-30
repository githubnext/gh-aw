//go:build !integration

package jsonutil_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/jsonutil"
)

// TestSpec_PublicAPI_MarshalCompactNoHTMLEscape validates the documented behavior of
// MarshalCompactNoHTMLEscape as described in the jsonutil README.md.
//
// Specification:
// - Marshals v to compact JSON without HTML escaping.
// - Characters like '&', '<', '>' are preserved as-is (not encoded to \u0026, \u003c, \u003e).
// - Trailing newline emitted by json.Encoder is trimmed so the result matches json.Marshal style.
func TestSpec_PublicAPI_MarshalCompactNoHTMLEscape(t *testing.T) {
	t.Run("preserves expression operators (& and |)", func(t *testing.T) {
		input := map[string]string{
			"expr": "${{ env.MCP_ENV == 'staging' && env.MCP_URL_STAGING || env.MCP_URL_PROD }}",
		}

		result, err := jsonutil.MarshalCompactNoHTMLEscape(input)
		require.NoError(t, err, "marshal should succeed")

		assert.Contains(t, result, "&&", "expected && to be preserved")
		assert.NotContains(t, result, `\u0026`, "expected & not to be HTML-escaped")
	})

	t.Run("output is compact (no trailing newline)", func(t *testing.T) {
		input := map[string]string{"key": "value"}

		result, err := jsonutil.MarshalCompactNoHTMLEscape(input)
		require.NoError(t, err, "marshal should succeed")

		assert.False(t, strings.HasSuffix(result, "\n"), "result must not have trailing newline")
	})

	t.Run("preserves angle brackets without HTML escaping", func(t *testing.T) {
		input := map[string]string{"x": "<tag>"}

		result, err := jsonutil.MarshalCompactNoHTMLEscape(input)
		require.NoError(t, err, "marshal should succeed")

		assert.Contains(t, result, "<tag>", "expected <tag> to be preserved")
		assert.NotContains(t, result, `\u003c`, "expected < not to be HTML-escaped")
		assert.NotContains(t, result, `\u003e`, "expected > not to be HTML-escaped")
	})
}
