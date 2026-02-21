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

// TestInlinedImports_FrontmatterField verifies that inlined-imports: true activates
// compile-time inlining of imports (without inputs) and the main workflow markdown.
func TestInlinedImports_FrontmatterField(t *testing.T) {
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

	// Create the main workflow file with inlined-imports: true
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "test-workflow.md")
	workflowContent := `---
name: inlined-imports-test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
inlined-imports: true
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

	// WorkflowData.InlinedImports should be true (parsed into the workspace data)
	assert.True(t, wd.InlinedImports, "WorkflowData.InlinedImports should be true")

	// ParsedFrontmatter should also have InlinedImports = true
	require.NotNil(t, wd.ParsedFrontmatter, "ParsedFrontmatter should not be nil")
	assert.True(t, wd.ParsedFrontmatter.InlinedImports, "InlinedImports should be true")

	// Compile and get YAML
	yamlContent, err := compiler.CompileToYAML(wd, workflowFile)
	require.NoError(t, err, "should compile workflow")
	require.NotEmpty(t, yamlContent, "YAML should not be empty")

	// With inlined-imports: true, the import should be inlined (no runtime-import macros)
	assert.NotContains(t, yamlContent, "{{#runtime-import", "should not generate any runtime-import macros")

	// The shared content should be inlined in the prompt
	assert.Contains(t, yamlContent, "Shared Instructions", "shared import content should be inlined")
	assert.Contains(t, yamlContent, "Always follow best practices", "shared import content should be inlined")

	// The main workflow content should also be inlined (no runtime-import for main file)
	assert.Contains(t, yamlContent, "Main Workflow", "main workflow content should be inlined")
	assert.Contains(t, yamlContent, "This is the main workflow content", "main workflow content should be inlined")
}

// TestInlinedImports_Disabled verifies that without inlined-imports, runtime-import macros are used.
func TestInlinedImports_Disabled(t *testing.T) {
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
name: no-inlined-imports-test
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
	assert.False(t, wd.ParsedFrontmatter.InlinedImports, "InlinedImports should be false by default")

	yamlContent, err := compiler.CompileToYAML(wd, workflowFile)
	require.NoError(t, err, "should compile workflow")

	// Without inlined-imports, the import should use runtime-import macro (with full path from workspace root)
	assert.Contains(t, yamlContent, "{{#runtime-import .github/workflows/shared/common.md}}", "should generate runtime-import macro for import")

	// The main workflow markdown should also use a runtime-import macro
	assert.Contains(t, yamlContent, "{{#runtime-import .github/workflows/test-workflow.md}}", "should generate runtime-import macro for main workflow")
}

// TestInlinedImports_HashChangesWithBody verifies that the frontmatter hash includes
// the entire markdown body when inlined-imports: true.
func TestInlinedImports_HashChangesWithBody(t *testing.T) {
	tmpDir := t.TempDir()

	content1 := `---
name: test
on:
  workflow_dispatch:
inlined-imports: true
engine: copilot
---

# Original body
`
	content2 := `---
name: test
on:
  workflow_dispatch:
inlined-imports: true
engine: copilot
---

# Modified body - different
`
	// Normal mode (no inlined-imports) - body changes should not affect hash
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

	// With inlined-imports: true, different body content should produce different hashes
	assert.NotEqual(t, hash1, hash2,
		"with inlined-imports: true, different body content should produce different hashes")

	// Without inlined-imports, body-only changes produce the same hash
	// (only env./vars. expressions from body are included)
	assert.Equal(t, hashN1, hashN2,
		"without inlined-imports, body-only changes should not affect hash")

	// inlined-imports mode should also produce a different hash than normal mode
	// (frontmatter text differs, so hash differs regardless of body treatment)
	assert.NotEqual(t, hash1, hashN1,
		"inlined-imports and normal mode should produce different hashes (different frontmatter)")
}

// TestInlinedImports_FrontmatterHashInline_SameBodySameHash verifies determinism.
func TestInlinedImports_FrontmatterHashInline_SameBodySameHash(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
name: test
on:
  workflow_dispatch:
inlined-imports: true
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

// TestInlinedImports_InlinePromptActivated verifies that inlined-imports also activates inline prompt mode.
func TestInlinedImports_InlinePromptActivated(t *testing.T) {
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
inlined-imports: true
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

	// When inlined-imports is true, the main markdown body is also inlined (no runtime-import for main file)
	assert.NotContains(t, yamlContent, "{{#runtime-import", "should not generate any runtime-import macros")
	// Main workflow content should be inlined
	assert.Contains(t, yamlContent, "My Workflow", "main workflow content should be inlined")
	assert.Contains(t, yamlContent, "Do something useful", "main workflow body should be inlined")
}

// TestInlinedImports_AgentFileError verifies that when inlined-imports: true and a custom agent
// file is imported, ParseWorkflowFile returns a compilation error.
// Agent files require runtime access and will not be resolved without sources.
func TestInlinedImports_AgentFileError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the .github/agents directory and agent file
	agentsDir := filepath.Join(tmpDir, ".github", "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	agentFile := filepath.Join(agentsDir, "my-agent.md")
	require.NoError(t, os.WriteFile(agentFile, []byte("# Agent\nDo things.\n"), 0o644))

	// Create the workflow file with inlined-imports: true importing the agent file
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "test-workflow.md")
	workflowContent := `---
name: inlined-agent-test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
inlined-imports: true
imports:
  - ../../.github/agents/my-agent.md
---

# Main Workflow

Do something.
`
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0o644))

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	_, err := compiler.ParseWorkflowFile(workflowFile)
	require.Error(t, err, "should return an error when inlined-imports is used with an agent file")
	assert.Contains(t, err.Error(), "inlined-imports cannot be used with agent file imports",
		"error message should explain the conflict")
	assert.Contains(t, err.Error(), "my-agent.md",
		"error message should include the agent file path")
}

// TestInlinedImports_AgentFileCleared verifies that buildInitialWorkflowData clears the AgentFile
// field when inlined-imports is true. Note: ParseWorkflowFile will error before this state is used.
func TestInlinedImports_AgentFileCleared(t *testing.T) {
	compiler := NewCompiler()

	frontmatterResult := &parser.FrontmatterResult{
		Frontmatter: map[string]any{
			"name":            "agent-test",
			"engine":          "copilot",
			"inlined-imports": true,
		},
		FrontmatterLines: []string{
			"name: agent-test",
			"engine: copilot",
			"inlined-imports: true",
		},
	}

	toolsResult := &toolsProcessingResult{
		workflowName:         "agent-test",
		frontmatterName:      "agent-test",
		parsedFrontmatter:    &FrontmatterConfig{Name: "agent-test", Engine: "copilot", InlinedImports: true},
		tools:                map[string]any{},
		importPaths:          []string{".github/agents/my-agent.md"},
		mainWorkflowMarkdown: "# Main",
	}

	engineSetup := &engineSetupResult{
		engineSetting: "copilot",
		engineConfig:  &EngineConfig{ID: "copilot"},
		sandboxConfig: &SandboxConfig{},
	}

	importsResult := &parser.ImportsResult{
		AgentFile:       ".github/agents/my-agent.md",
		AgentImportSpec: ".github/agents/my-agent.md",
	}

	wd := compiler.buildInitialWorkflowData(frontmatterResult, toolsResult, engineSetup, importsResult)

	// InlinedImports should be true in WorkflowData
	assert.True(t, wd.InlinedImports, "InlinedImports should be true in WorkflowData")

	// AgentFile should be cleared (content inlined via ImportPaths instead)
	assert.Empty(t, wd.AgentFile, "AgentFile should be cleared when inlined-imports is true")
	assert.Empty(t, wd.AgentImportSpec, "AgentImportSpec should be cleared when inlined-imports is true")
}
