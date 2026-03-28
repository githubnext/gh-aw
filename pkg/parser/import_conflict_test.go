//go:build !integration

package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportConflict_SameFileConflictingWith tests that importing the same file twice
// with different 'with' values produces a clear error.
func TestImportConflict_SameFileConflictingWith(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-conflict-*")

	// Shared workflow
	sharedPath := filepath.Join(tempDir, "shared.md")
	sharedContent := `---
import-schema:
  region:
    type: string
    required: true
---
# Shared
Deploy to ${{ github.aw.import-inputs.region }}.
`
	require.NoError(t, os.WriteFile(sharedPath, []byte(sharedContent), 0644))

	// Main workflow imports shared.md twice with conflicting 'with' values
	mainPath := filepath.Join(tempDir, "main.md")
	mainContent := `---
on: issues
engine: copilot
imports:
  - uses: shared.md
    with:
      region: us-east-1
  - uses: shared.md
    with:
      region: eu-west-1
permissions:
  contents: read
  issues: read
---
# Main
`
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

	frontmatter := map[string]any{
		"on":     "issues",
		"engine": "copilot",
		"imports": []any{
			map[string]any{
				"uses": "shared.md",
				"with": map[string]any{"region": "us-east-1"},
			},
			map[string]any{
				"uses": "shared.md",
				"with": map[string]any{"region": "eu-west-1"},
			},
		},
	}

	_, err := parser.ProcessImportsFromFrontmatterWithSource(frontmatter, tempDir, nil, mainPath, mainContent)
	require.Error(t, err, "Importing the same file with conflicting 'with' values should error")
	assert.Contains(t, err.Error(), "import conflict", "Error should mention import conflict")
	assert.Contains(t, err.Error(), "shared.md", "Error should mention the conflicting file")
}

// TestImportConflict_SameFileIdenticalWith tests that importing the same file twice
// with identical 'with' values is allowed (the second import is a no-op).
func TestImportConflict_SameFileIdenticalWith(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-dup-ok-*")

	sharedPath := filepath.Join(tempDir, "shared.md")
	sharedContent := `---
import-schema:
  region:
    type: string
    required: true
---
# Shared
Deploy to ${{ github.aw.import-inputs.region }}.
`
	require.NoError(t, os.WriteFile(sharedPath, []byte(sharedContent), 0644))

	mainPath := filepath.Join(tempDir, "main.md")
	mainContent := `---
on: issues
engine: copilot
imports:
  - uses: shared.md
    with:
      region: us-east-1
  - uses: shared.md
    with:
      region: us-east-1
permissions:
  contents: read
  issues: read
---
# Main
`
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

	frontmatter := map[string]any{
		"on":     "issues",
		"engine": "copilot",
		"imports": []any{
			map[string]any{
				"uses": "shared.md",
				"with": map[string]any{"region": "us-east-1"},
			},
			map[string]any{
				"uses": "shared.md",
				"with": map[string]any{"region": "us-east-1"},
			},
		},
	}

	_, err := parser.ProcessImportsFromFrontmatterWithSource(frontmatter, tempDir, nil, mainPath, mainContent)
	assert.NoError(t, err, "Importing the same file twice with identical 'with' values should be allowed")
}

// TestImportConflict_SameFileTwiceNoWith tests that importing the same file (no 'with')
// twice is silently deduplicated (no error).
func TestImportConflict_SameFileTwiceNoWith(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-dup-noWith-*")

	sharedPath := filepath.Join(tempDir, "shared.md")
	sharedContent := `---
mcp-servers:
  my-tool:
    url: "https://example.com/tool"
    allowed: ["*"]
---
# Shared Tool
`
	require.NoError(t, os.WriteFile(sharedPath, []byte(sharedContent), 0644))

	mainPath := filepath.Join(tempDir, "main.md")
	mainContent := `---
on: issues
engine: copilot
imports:
  - shared.md
  - shared.md
permissions:
  contents: read
  issues: read
---
# Main
`
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

	frontmatter := map[string]any{
		"on":      "issues",
		"engine":  "copilot",
		"imports": []any{"shared.md", "shared.md"},
	}

	_, err := parser.ProcessImportsFromFrontmatterWithSource(frontmatter, tempDir, nil, mainPath, mainContent)
	assert.NoError(t, err, "Importing the same file (no 'with') twice should be silently deduplicated")
}

// TestImportConflict_NestedConflict tests that a conflict detected via nested imports
// (two different parent files both importing the same leaf with different 'with') is also caught.
func TestImportConflict_NestedConflict(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-nested-conflict-*")

	// leaf.md: parameterized shared workflow
	leafPath := filepath.Join(tempDir, "leaf.md")
	leafContent := `---
import-schema:
  env:
    type: string
    required: true
mcp-servers:
  leaf-tool:
    url: "https://example.com/leaf"
    allowed: ["*"]
---
# Leaf - ${{ github.aw.import-inputs.env }}
`
	require.NoError(t, os.WriteFile(leafPath, []byte(leafContent), 0644))

	// parent-a.md: imports leaf.md with env=staging
	parentAPath := filepath.Join(tempDir, "parent-a.md")
	parentAContent := `---
imports:
  - uses: leaf.md
    with:
      env: staging
---
# Parent A
`
	require.NoError(t, os.WriteFile(parentAPath, []byte(parentAContent), 0644))

	// parent-b.md: imports leaf.md with env=production (CONFLICT)
	parentBPath := filepath.Join(tempDir, "parent-b.md")
	parentBContent := `---
imports:
  - uses: leaf.md
    with:
      env: production
---
# Parent B
`
	require.NoError(t, os.WriteFile(parentBPath, []byte(parentBContent), 0644))

	// main.md: imports both parent-a and parent-b, which transitively causes a conflict on leaf.md
	mainPath := filepath.Join(tempDir, "main.md")
	mainContent := `---
on: issues
engine: copilot
imports:
  - parent-a.md
  - parent-b.md
permissions:
  contents: read
  issues: read
---
# Main
`
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

	frontmatter := map[string]any{
		"on":      "issues",
		"engine":  "copilot",
		"imports": []any{"parent-a.md", "parent-b.md"},
	}

	_, err := parser.ProcessImportsFromFrontmatterWithSource(frontmatter, tempDir, nil, mainPath, mainContent)
	require.Error(t, err, "Nested conflict on leaf.md should produce an error")
	assert.True(t,
		strings.Contains(err.Error(), "import conflict") || strings.Contains(err.Error(), "leaf.md"),
		"Error should mention the conflict: %v", err)
}
