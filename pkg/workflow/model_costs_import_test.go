//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportedModelsPricingMergesAcrossSharedWorkflows(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))
	workflowsDir := makeTestWorkflowsDir(t)

	sharedBase := `---
models:
  providers:
    anthropic:
      models:
        claude-opus:
          cost:
            input: "1e-5"
            output: "3e-5"
        claude-haiku:
          cost:
            input: "2e-6"
    openai:
      models:
        gpt-4:
          cost:
            input: "5e-6"
---
# Shared base
`
	writeWorkflowFile(t, workflowsDir, "shared-base.md", sharedBase)

	sharedOverlay := `---
models:
  providers:
    anthropic:
      models:
        claude-opus:
          cost:
            input: "9e-6"
        custom-anthropic:
          cost:
            input: "1e-6"
---
# Shared overlay
`
	writeWorkflowFile(t, workflowsDir, "shared-overlay.md", sharedOverlay)

	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-base.md
  - ./shared-overlay.md
---
# Main workflow
`
	writeWorkflowFile(t, workflowsDir, "main.md", mainWorkflow)

	withWorkingDirectory(t, workflowsDir, func() {
		workflowData, err := compiler.ParseWorkflowFile("main.md")
		require.NoError(t, err)
		require.NotNil(t, workflowData.ModelCosts)

		providers, ok := workflowData.ModelCosts["providers"].(map[string]any)
		require.True(t, ok)
		anthropic := providers["anthropic"].(map[string]any)
		anthropicModels := anthropic["models"].(map[string]any)

		assert.Equal(t, map[string]any{"cost": map[string]any{"input": "9e-6"}}, anthropicModels["claude-opus"])
		assert.Equal(t, map[string]any{"cost": map[string]any{"input": "2e-6"}}, anthropicModels["claude-haiku"])
		assert.Equal(t, map[string]any{"cost": map[string]any{"input": "1e-6"}}, anthropicModels["custom-anthropic"])

		openai := providers["openai"].(map[string]any)
		openaiModels := openai["models"].(map[string]any)
		assert.Equal(t, map[string]any{"cost": map[string]any{"input": "5e-6"}}, openaiModels["gpt-4"])
	})
}

func TestMainWorkflowModelsPricingOverridesImportedSharedWorkflow(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))
	workflowsDir := makeTestWorkflowsDir(t)

	sharedWorkflow := `---
models:
  providers:
    anthropic:
      models:
        claude-opus:
          cost:
            input: "1e-5"
            output: "3e-5"
        claude-haiku:
          cost:
            input: "2e-6"
---
# Shared workflow
`
	writeWorkflowFile(t, workflowsDir, "shared.md", sharedWorkflow)

	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared.md
models:
  providers:
    anthropic:
      models:
        claude-opus:
          cost:
            input: "7e-6"
            output: "2e-5"
    openai:
      models:
        gpt-5:
          cost:
            input: "4e-6"
---
# Main workflow
`
	writeWorkflowFile(t, workflowsDir, "main.md", mainWorkflow)

	withWorkingDirectory(t, workflowsDir, func() {
		workflowData, err := compiler.ParseWorkflowFile("main.md")
		require.NoError(t, err)
		require.NotNil(t, workflowData.ModelCosts)

		providers, ok := workflowData.ModelCosts["providers"].(map[string]any)
		require.True(t, ok)
		anthropic := providers["anthropic"].(map[string]any)
		anthropicModels := anthropic["models"].(map[string]any)

		assert.Equal(t, map[string]any{"cost": map[string]any{"input": "7e-6", "output": "2e-5"}}, anthropicModels["claude-opus"])
		assert.Equal(t, map[string]any{"cost": map[string]any{"input": "2e-6"}}, anthropicModels["claude-haiku"])

		openai := providers["openai"].(map[string]any)
		openaiModels := openai["models"].(map[string]any)
		assert.Equal(t, map[string]any{"cost": map[string]any{"input": "4e-6"}}, openaiModels["gpt-5"])
	})
}

func makeTestWorkflowsDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))
	return workflowsDir
}

func writeWorkflowFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644))
}

func withWorkingDirectory(t *testing.T, dir string, fn func()) {
	t.Helper()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() {
		require.NoError(t, os.Chdir(oldDir))
	}()
	fn()
}
