//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInlineImports_FrontmatterField verifies that inline-imports: true activates
// compile-time inlining of imports (without inputs) and the main workflow markdown.
func TestInlineImports_FrontmatterField(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a shared import file with markdown content
	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755))
	sharedFile := filepath.Join(sharedDir, "common.md")
	sharedContent := `---
tools:
  bash: true
---

# Shared Instructions

Always follow best practices.
`
	require.NoError(t, os.WriteFile(sharedFile, []byte(sharedContent), 0o644))

	// Create the main workflow file with inline-imports: true
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "test-workflow.md")
	workflowContent := `---
name: inline-imports-test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
inline-imports: true
imports:
  - shared/common.md
---

# Main Workflow

This is the main workflow content.
`
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0o644))

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowFile(workflowFile)
	require.NoError(t, err, "should parse workflow file")
	require.NotNil(t, wd)

	// ParsedFrontmatter should have InlineImports = true
	require.NotNil(t, wd.ParsedFrontmatter, "ParsedFrontmatter should not be nil")
	assert.True(t, wd.ParsedFrontmatter.InlineImports, "InlineImports should be true")

	// Compile and get YAML
	yamlContent, err := compiler.CompileToYAML(wd, workflowFile)
	require.NoError(t, err, "should compile workflow")
	require.NotEmpty(t, yamlContent, "YAML should not be empty")

	// With inline-imports: true, the import should be inlined (no runtime-import macros)
	assert.NotContains(t, yamlContent, "{{#runtime-import", "should not generate any runtime-import macros")

	// The shared content should be inlined in the prompt
	assert.Contains(t, yamlContent, "Shared Instructions", "shared import content should be inlined")
	assert.Contains(t, yamlContent, "Always follow best practices", "shared import content should be inlined")

	// The main workflow content should also be inlined (no runtime-import for main file)
	assert.Contains(t, yamlContent, "Main Workflow", "main workflow content should be inlined")
	assert.Contains(t, yamlContent, "This is the main workflow content", "main workflow content should be inlined")
}

// TestInlineImports_Disabled verifies that without inline-imports, runtime-import macros are used.
func TestInlineImports_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755))
	sharedFile := filepath.Join(sharedDir, "common.md")
	sharedContent := `---
tools:
  bash: true
---

# Shared Instructions

Always follow best practices.
`
	require.NoError(t, os.WriteFile(sharedFile, []byte(sharedContent), 0o644))

	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "test-workflow.md")
	workflowContent := `---
name: no-inline-imports-test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
imports:
  - shared/common.md
---

# Main Workflow

This is the main workflow content.
`
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0o644))

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowFile(workflowFile)
	require.NoError(t, err, "should parse workflow file")
	require.NotNil(t, wd)

	require.NotNil(t, wd.ParsedFrontmatter, "ParsedFrontmatter should be populated")
	assert.False(t, wd.ParsedFrontmatter.InlineImports, "InlineImports should be false by default")

	yamlContent, err := compiler.CompileToYAML(wd, workflowFile)
	require.NoError(t, err, "should compile workflow")

	// Without inline-imports, the import should use runtime-import macro (with full path from workspace root)
	assert.Contains(t, yamlContent, "{{#runtime-import .github/workflows/shared/common.md}}", "should generate runtime-import macro for import")

	// The main workflow markdown should also use a runtime-import macro
	assert.Contains(t, yamlContent, "{{#runtime-import .github/workflows/test-workflow.md}}", "should generate runtime-import macro for main workflow")
}

// TestInlineImports_HashChangesWithBody verifies that the frontmatter hash includes
// the entire markdown body when inline-imports: true.
func TestInlineImports_HashChangesWithBody(t *testing.T) {
	tmpDir := t.TempDir()

	content1 := `---
name: test
on:
  workflow_dispatch:
inline-imports: true
engine: copilot
---

# Original body
`
	content2 := `---
name: test
on:
  workflow_dispatch:
inline-imports: true
engine: copilot
---

# Modified body - different
`
	// Normal mode (no inline-imports) - body changes should not affect hash
	contentNormal1 := `---
name: test
on:
  workflow_dispatch:
engine: copilot
---

# Body variant A
`
	contentNormal2 := `---
name: test
on:
  workflow_dispatch:
engine: copilot
---

# Body variant B - same hash expected
`

	file1 := filepath.Join(tmpDir, "test1.md")
	file2 := filepath.Join(tmpDir, "test2.md")
	fileN1 := filepath.Join(tmpDir, "normal1.md")
	fileN2 := filepath.Join(tmpDir, "normal2.md")
	require.NoError(t, os.WriteFile(file1, []byte(content1), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte(content2), 0o644))
	require.NoError(t, os.WriteFile(fileN1, []byte(contentNormal1), 0o644))
	require.NoError(t, os.WriteFile(fileN2, []byte(contentNormal2), 0o644))

	cache := parser.NewImportCache(tmpDir)

	hash1, err := parser.ComputeFrontmatterHashFromFile(file1, cache)
	require.NoError(t, err)
	hash2, err := parser.ComputeFrontmatterHashFromFile(file2, cache)
	require.NoError(t, err)
	hashN1, err := parser.ComputeFrontmatterHashFromFile(fileN1, cache)
	require.NoError(t, err)
	hashN2, err := parser.ComputeFrontmatterHashFromFile(fileN2, cache)
	require.NoError(t, err)

	// With inline-imports: true, different body content should produce different hashes
	assert.NotEqual(t, hash1, hash2,
		"with inline-imports: true, different body content should produce different hashes")

	// Without inline-imports, body-only changes produce the same hash
	// (only env./vars. expressions from body are included)
	assert.Equal(t, hashN1, hashN2,
		"without inline-imports, body-only changes should not affect hash")

	// inline-imports mode should also produce a different hash than normal mode
	// (frontmatter text differs, so hash differs regardless of body treatment)
	assert.NotEqual(t, hash1, hashN1,
		"inline-imports and normal mode should produce different hashes (different frontmatter)")
}

// TestInlineImports_FrontmatterHashInline_SameBodySameHash verifies determinism.
func TestInlineImports_FrontmatterHashInline_SameBodySameHash(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
name: test
on:
  workflow_dispatch:
inline-imports: true
engine: copilot
---

# Same body content
`
	file1 := filepath.Join(tmpDir, "a.md")
	file2 := filepath.Join(tmpDir, "b.md")
	require.NoError(t, os.WriteFile(file1, []byte(content), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte(content), 0o644))

	cache := parser.NewImportCache(tmpDir)
	hash1, err := parser.ComputeFrontmatterHashFromFile(file1, cache)
	require.NoError(t, err)
	hash2, err := parser.ComputeFrontmatterHashFromFile(file2, cache)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "same content should produce the same hash")
}

// TestInlineImports_InlinePromptActivated verifies that inline-imports also activates inline prompt mode.
func TestInlineImports_InlinePromptActivated(t *testing.T) {
	tmpDir := t.TempDir()

	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "inline-test.md")
	workflowContent := `---
name: inline-test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
inline-imports: true
---

# My Workflow

Do something useful.
`
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0o644))

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowFile(workflowFile)
	require.NoError(t, err)

	yamlContent, err := compiler.CompileToYAML(wd, workflowFile)
	require.NoError(t, err)

	// When inline-imports is true, the main markdown body is also inlined (no runtime-import for main file)
	assert.NotContains(t, yamlContent, "{{#runtime-import", "should not generate any runtime-import macros")
	// Main workflow content should be inlined
	assert.Contains(t, yamlContent, "My Workflow", "main workflow content should be inlined")
	assert.Contains(t, yamlContent, "Do something useful", "main workflow body should be inlined")
}
