//go:build !integration

package jsonutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalCompactNoHTMLEscape(t *testing.T) {
	input := map[string]string{
		"expr": "${{ env.MCP_ENV == 'staging' && env.MCP_URL_STAGING || env.MCP_URL_PROD }}",
	}

	result, err := MarshalCompactNoHTMLEscape(input)
	require.NoError(t, err, "marshal should succeed")

	assert.Contains(t, result, "&&", "expected expression operators to be preserved")
	assert.NotContains(t, result, "\\u0026", "expected '&' to not be HTML-escaped")
	assert.NotContains(t, result, "\n", "expected compact JSON without trailing newline")
}
