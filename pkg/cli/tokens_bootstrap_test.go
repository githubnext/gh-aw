//go:build !integration

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectRequiredSecretsFromWorkflows_WithTestFixtures(t *testing.T) {
	// Create a temporary directory with test workflow files
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err)

	// Create test workflow files with different engines
	testWorkflows := map[string]string{
		"copilot-workflow.md": `---
engine: copilot
on: push
---
# Copilot Workflow
This workflow uses Copilot.`,
		"claude-workflow.md": `---
engine: claude
on: pull_request
---
# Claude Workflow
This workflow uses Claude.`,
		"codex-workflow.md": `---
engine: codex
on: workflow_dispatch
---
# Codex Workflow
This workflow uses Codex.`,
	}

	for filename, content := range testWorkflows {
		path := filepath.Join(workflowsDir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Save original directory and change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Run("discovers secrets from test workflows", func(t *testing.T) {
		requirements, err := getSecretRequirements("")

		require.NoError(t, err, "Should successfully collect secrets from workflows")
		require.NotEmpty(t, requirements, "Should find at least one required secret")

		// Should include system secrets
		hasSystemSecret := false
		for _, req := range requirements {
			if req.Name == "GH_AW_GITHUB_TOKEN" {
				hasSystemSecret = true
				assert.False(t, req.IsEngineSecret, "System secret should not be marked as engine secret")
				break
			}
		}
		assert.True(t, hasSystemSecret, "Should include system secret GH_AW_GITHUB_TOKEN")

		// Should include engine secrets for copilot, claude, and codex
		engineSecrets := make(map[string]bool)
		for _, req := range requirements {
			if req.IsEngineSecret {
				engineSecrets[req.Name] = true
				assert.NotEmpty(t, req.EngineName, "Engine secret should have engine name")
				assert.False(t, req.Optional, "Engine secret %s should be required", req.Name)
			}
		}

		assert.True(t, engineSecrets["COPILOT_GITHUB_TOKEN"], "Should include COPILOT_GITHUB_TOKEN")
		assert.True(t, engineSecrets["ANTHROPIC_API_KEY"], "Should include ANTHROPIC_API_KEY")
		assert.True(t, engineSecrets["OPENAI_API_KEY"], "Should include OPENAI_API_KEY")
	})

	t.Run("filters by engine", func(t *testing.T) {
		requirements, err := getSecretRequirements("copilot")

		require.NoError(t, err, "Should successfully collect secrets for copilot engine")
		require.NotEmpty(t, requirements, "Should find required secrets")

		// Should only include copilot-related secrets (and system secrets)
		hasOnlyCopilot := true
		for _, req := range requirements {
			if req.IsEngineSecret && req.Name != "COPILOT_GITHUB_TOKEN" {
				hasOnlyCopilot = false
				break
			}
		}
		assert.True(t, hasOnlyCopilot, "Should only include Copilot secrets when filtering by copilot")
	})

	t.Run("handles no matching workflows gracefully", func(t *testing.T) {
		requirements, err := getSecretRequirements("nonexistent-engine")

		require.NoError(t, err, "Should not error when no workflows match filter")

		// Should still return system secrets even if no engine workflows match
		hasSystemSecret := false
		for _, req := range requirements {
			if req.Name == "GH_AW_GITHUB_TOKEN" {
				hasSystemSecret = true
				break
			}
		}
		assert.True(t, hasSystemSecret, "Should include system secrets even with no matching engine workflows")

		// Should not include any engine secrets
		for _, req := range requirements {
			if req.IsEngineSecret {
				t.Errorf("Should not include engine secret %s when no workflows match", req.Name)
			}
		}
	})
}

func TestCollectRequiredSecretsFromWorkflows_NoWorkflowsDir(t *testing.T) {
	// Save original working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	// Create a temporary directory without .github/workflows
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("fails when no engine specified", func(t *testing.T) {
		_, err = getSecretRequirements("")
		require.Error(t, err, "Should error when no workflows directory exists and no engine specified")
		assert.Contains(t, err.Error(), "failed to discover workflows", "Error should indicate workflow discovery failed")
	})

	t.Run("succeeds when engine specified", func(t *testing.T) {
		requirements, err := getSecretRequirements("copilot")
		require.NoError(t, err, "Should not error when engine is explicitly specified")
		require.NotEmpty(t, requirements, "Should return secrets for specified engine")

		// Should include system secrets
		hasSystemSecret := false
		hasCopilotSecret := false
		for _, req := range requirements {
			if req.Name == "GH_AW_GITHUB_TOKEN" {
				hasSystemSecret = true
			}
			if req.Name == "COPILOT_GITHUB_TOKEN" {
				hasCopilotSecret = true
			}
		}
		assert.True(t, hasSystemSecret, "Should include system secret")
		assert.True(t, hasCopilotSecret, "Should include Copilot secret when engine=copilot")
	})
}

func TestCollectRequiredSecretsFromWorkflows_EmptyWorkflowsDir(t *testing.T) {
	// Save original working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	// Create a temporary directory with empty .github/workflows
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("fails when no engine specified", func(t *testing.T) {
		_, err = getSecretRequirements("")
		require.Error(t, err, "Should error when no workflow files found and no engine specified")
		assert.Contains(t, err.Error(), "no workflow files found", "Error should indicate no workflow files")
	})

	t.Run("succeeds when engine specified", func(t *testing.T) {
		requirements, err := getSecretRequirements("claude")
		require.NoError(t, err, "Should not error when engine is explicitly specified")
		require.NotEmpty(t, requirements, "Should return secrets for specified engine")

		// Should include system secrets
		hasSystemSecret := false
		hasClaudeSecret := false
		for _, req := range requirements {
			if req.Name == "GH_AW_GITHUB_TOKEN" {
				hasSystemSecret = true
			}
			if req.Name == "ANTHROPIC_API_KEY" {
				hasClaudeSecret = true
			}
		}
		assert.True(t, hasSystemSecret, "Should include system secret")
		assert.True(t, hasClaudeSecret, "Should include Claude API key when engine=claude")
	})
}

func TestCollectRequiredSecretsFromWorkflows_Deduplication(t *testing.T) {
	// Create a temporary directory with multiple workflows using the same engine
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err)

	// Create multiple workflows that use the same engine
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(workflowsDir, fmt.Sprintf("copilot-workflow-%d.md", i))
		content := `---
engine: copilot
on: push
---
# Copilot Workflow
This workflow uses Copilot.`
		err := os.WriteFile(filename, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Save original directory and change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	requirements, err := getSecretRequirements("")
	require.NoError(t, err)

	// Count occurrences of each secret name
	secretCounts := make(map[string]int)
	for _, req := range requirements {
		secretCounts[req.Name]++
	}

	// Verify no duplicates
	for secretName, count := range secretCounts {
		assert.Equal(t, 1, count, "Secret %s should appear exactly once, found %d times", secretName, count)
	}

	// Should have system secrets + copilot token
	// (System secrets include required and optional ones)
	assert.GreaterOrEqual(t, len(secretCounts), 2, "Should have at least 2 unique secrets")
	assert.Contains(t, secretCounts, "GH_AW_GITHUB_TOKEN", "Should include system token")
	assert.Contains(t, secretCounts, "COPILOT_GITHUB_TOKEN", "Should include Copilot token")
}
func TestExtractEnginesFromWorkflows(t *testing.T) {
	// Create a temporary directory for test workflow files
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err)

	t.Run("extracts multiple unique engines", func(t *testing.T) {
		// Create test workflow files
		testWorkflows := map[string]string{
			"copilot-workflow.md": `---
engine: copilot
on: push
---
# Copilot Workflow`,
			"claude-workflow.md": `---
engine: claude
on: pull_request
---
# Claude Workflow`,
			"codex-workflow.md": `---
engine: codex
on: workflow_dispatch
---
# Codex Workflow`,
		}

		var workflowFiles []string
		for filename, content := range testWorkflows {
			path := filepath.Join(workflowsDir, filename)
			err := os.WriteFile(path, []byte(content), 0644)
			require.NoError(t, err)
			workflowFiles = append(workflowFiles, path)
		}

		engines := extractEnginesFromWorkflows(workflowFiles)

		require.Len(t, engines, 3, "Should extract 3 unique engines")

		// Convert to map for easy checking
		engineMap := make(map[string]bool)
		for _, engine := range engines {
			engineMap[engine] = true
		}

		assert.True(t, engineMap["copilot"], "Should include copilot engine")
		assert.True(t, engineMap["claude"], "Should include claude engine")
		assert.True(t, engineMap["codex"], "Should include codex engine")
	})

	t.Run("deduplicates duplicate engines", func(t *testing.T) {
		// Create multiple workflows with the same engine
		var workflowFiles []string
		for i := 1; i <= 3; i++ {
			path := filepath.Join(workflowsDir, fmt.Sprintf("copilot-%d.md", i))
			content := `---
engine: copilot
on: push
---
# Copilot Workflow`
			err := os.WriteFile(path, []byte(content), 0644)
			require.NoError(t, err)
			workflowFiles = append(workflowFiles, path)
		}

		engines := extractEnginesFromWorkflows(workflowFiles)

		assert.Len(t, engines, 1, "Should deduplicate to 1 unique engine")
		assert.Equal(t, "copilot", engines[0], "Should be copilot engine")
	})

	t.Run("handles empty workflow list", func(t *testing.T) {
		engines := extractEnginesFromWorkflows([]string{})

		assert.Empty(t, engines, "Should return empty slice for empty input")
	})

	t.Run("filters subset of workflows", func(t *testing.T) {
		// Create workflows with different engines
		copilotPath := filepath.Join(workflowsDir, "copilot-only.md")
		err := os.WriteFile(copilotPath, []byte(`---
engine: copilot
on: push
---
# Copilot`), 0644)
		require.NoError(t, err)

		claudePath := filepath.Join(workflowsDir, "claude-only.md")
		err = os.WriteFile(claudePath, []byte(`---
engine: claude
on: push
---
# Claude`), 0644)
		require.NoError(t, err)

		// Extract engines from only the copilot workflow
		engines := extractEnginesFromWorkflows([]string{copilotPath})

		assert.Len(t, engines, 1, "Should extract only 1 engine")
		assert.Equal(t, "copilot", engines[0], "Should be copilot engine")

		// Extract engines from only the claude workflow
		engines = extractEnginesFromWorkflows([]string{claudePath})

		assert.Len(t, engines, 1, "Should extract only 1 engine")
		assert.Equal(t, "claude", engines[0], "Should be claude engine")
	})
}
