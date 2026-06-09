//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestParseImportSpecsFromArray_InvalidIfType(t *testing.T) {
	_, err := parseImportSpecsFromArray([]any{
		map[string]any{
			"uses": "shared/workflow.md",
			"if":   true,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "import 'if' must be a string")
}

func TestProcessImportsFromFrontmatter_NestedImportInheritsIfCondition(t *testing.T) {
	tmpDir := t.TempDir()
	sharedDir := filepath.Join(tmpDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "leaf.md"), []byte(`---
steps:
  - name: Leaf
    run: echo leaf
---
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "parent.md"), []byte(`---
imports:
  - uses: shared/leaf.md
---
`), 0o644))

	mainContent := `---
imports:
  - uses: shared/parent.md
    if: "experiments.variant == 'a'"
---
`
	result, err := ExtractFrontmatterFromContent(mainContent)
	require.NoError(t, err)

	importsResult, err := ProcessImportsFromFrontmatterWithSource(result.Frontmatter, tmpDir, nil, "", "")
	require.NoError(t, err)
	assert.Contains(t, importsResult.MergedSteps, "needs.activation.outputs.variant == 'a'")
}

func TestProcessImportsFromFrontmatter_DuplicateImportWithDifferentIfErrors(t *testing.T) {
	tmpDir := t.TempDir()
	sharedDir := filepath.Join(tmpDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "a.md"), []byte(`---
steps:
  - name: A
    run: echo a
---
`), 0o644))

	mainContent := `---
imports:
  - uses: shared/a.md
    if: "experiments.variant == 'a'"
  - uses: shared/a.md
    if: "experiments.variant == 'b'"
---
`
	result, err := ExtractFrontmatterFromContent(mainContent)
	require.NoError(t, err)

	_, err = ProcessImportsFromFrontmatterWithSource(result.Frontmatter, tmpDir, nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different 'if' conditions")
}
