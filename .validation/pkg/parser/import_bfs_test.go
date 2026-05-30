//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNestedImportEntries_LenientArrayParsing(t *testing.T) {
	frontmatter := map[string]any{
		"imports": []any{
			"valid-a.md",
			map[string]any{"path": "valid-b.md", "inputs": map[string]any{"env": "prod"}},
			map[string]any{"path": 123},
			map[string]any{"inputs": map[string]any{"env": "ignored"}},
			42,
		},
	}

	entries := parseNestedImportEntries(frontmatter)
	require.Len(t, entries, 2)
	require.Equal(t, "valid-a.md", entries[0].path)
	require.Nil(t, entries[0].inputs)
	require.Equal(t, "valid-b.md", entries[1].path)
	require.Equal(t, map[string]any{"env": "prod"}, entries[1].inputs)
}
