//go:build !integration

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequiredSecretsForWorkflow(t *testing.T) {
	// Create a temporary directory with test workflow files
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Should create workflows directory")

	tests := []struct {
		name               string
		workflowContent    string
		expectedSecretName string
		expectNil          bool
	}{
		{
			name: "copilot engine workflow",
			workflowContent: `---
engine: copilot
on: push
---
# Copilot Workflow`,
			expectedSecretName: "COPILOT_GITHUB_TOKEN",
			expectNil:          false,
		},
		{
			name: "claude engine workflow",
			workflowContent: `---
engine: claude
on: push
---
# Claude Workflow`,
			expectedSecretName: "ANTHROPIC_API_KEY",
			expectNil:          false,
		},
		{
			name: "workflow without engine",
			workflowContent: `---
on: push
---
# No Engine Workflow`,
			expectedSecretName: "COPILOT_GITHUB_TOKEN", // Defaults to copilot
			expectNil:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create workflow file with unique name
			workflowPath := filepath.Join(workflowsDir, tt.name+".md")
			err := os.WriteFile(workflowPath, []byte(tt.workflowContent), 0644)
			require.NoError(t, err, "Should write workflow file")
			defer os.Remove(workflowPath)

			// Test the function
			secrets := getSecretRequirementsForWorkflow(workflowPath)

			if tt.expectNil {
				assert.Nil(t, secrets, "Should return nil for workflow without engine")
			} else {
				require.NotNil(t, secrets, "Should return secrets")
				require.NotEmpty(t, secrets, "Should have at least one secret")

				// Check that the expected secret is present
				found := false
				for _, secret := range secrets {
					if secret.Name == tt.expectedSecretName {
						found = true
						assert.True(t, secret.IsEngineSecret, "Should be marked as engine secret")
						break
					}
				}
				assert.True(t, found, "Should include expected secret %s", tt.expectedSecretName)
			}
		})
	}
}

func TestGetRequiredSecretsForWorkflows(t *testing.T) {
	t.Run("collects secrets from multiple workflows", func(t *testing.T) {
		// Create a temporary directory with test workflow files
		tempDir := t.TempDir()
		workflowsDir := filepath.Join(tempDir, ".github", "workflows")
		err := os.MkdirAll(workflowsDir, 0755)
		require.NoError(t, err, "Should create workflows directory")

		// Create test workflow files with different engines
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
			require.NoError(t, err, "Should write workflow file %s", filename)
			workflowFiles = append(workflowFiles, path)
		}

		// Test the function
		secrets := getSecretsRequirementsForWorkflows(workflowFiles)

		require.NotEmpty(t, secrets, "Should return secrets")

		// Convert to map for easy checking
		secretMap := make(map[string]SecretRequirement)
		for _, secret := range secrets {
			secretMap[secret.Name] = secret
		}

		// Check engine secrets
		assert.Contains(t, secretMap, "COPILOT_GITHUB_TOKEN", "Should include Copilot secret")
		assert.Contains(t, secretMap, "ANTHROPIC_API_KEY", "Should include Claude secret")
		assert.Contains(t, secretMap, "OPENAI_API_KEY", "Should include Codex secret")

		// Check system secrets are included
		for _, sys := range constants.SystemSecrets {
			assert.Contains(t, secretMap, sys.Name, "Should include system secret %s", sys.Name)
		}

		// Verify no duplicates
		assert.Len(t, secrets, len(secretMap), "Should not have duplicate secrets")
	})

	t.Run("handles empty workflow list", func(t *testing.T) {
		secrets := getSecretsRequirementsForWorkflows([]string{})

		require.NotEmpty(t, secrets, "Should return at least system secrets")

		// Should only have system secrets
		assert.Len(t, secrets, len(constants.SystemSecrets), "Should only have system secrets")

		// Verify all are system secrets
		for _, secret := range secrets {
			assert.False(t, secret.IsEngineSecret, "All secrets should be system secrets")
		}
	})

	t.Run("deduplicates secrets from workflows with same engine", func(t *testing.T) {
		// Create a temporary directory with test workflow files
		tempDir := t.TempDir()
		workflowsDir := filepath.Join(tempDir, ".github", "workflows")
		err := os.MkdirAll(workflowsDir, 0755)
		require.NoError(t, err, "Should create workflows directory")

		// Create multiple workflows using the same engine
		var workflowFiles []string
		for i := 1; i <= 3; i++ {
			content := `---
engine: copilot
on: push
---
# Copilot Workflow`
			path := filepath.Join(workflowsDir, fmt.Sprintf("copilot-workflow-%d.md", i))
			err := os.WriteFile(path, []byte(content), 0644)
			require.NoError(t, err, "Should write workflow file")
			workflowFiles = append(workflowFiles, path)
		}

		secrets := getSecretsRequirementsForWorkflows(workflowFiles)

		// Count Copilot tokens
		copilotCount := 0
		for _, secret := range secrets {
			if secret.Name == "COPILOT_GITHUB_TOKEN" {
				copilotCount++
			}
		}

		assert.Equal(t, 1, copilotCount, "Should have exactly one Copilot token despite multiple workflows")
	})

	t.Run("skips workflows without engines", func(t *testing.T) {
		// Create a temporary directory with test workflow files
		tempDir := t.TempDir()
		workflowsDir := filepath.Join(tempDir, ".github", "workflows")
		err := os.MkdirAll(workflowsDir, 0755)
		require.NoError(t, err, "Should create workflows directory")

		// Create one workflow with engine and one without
		workflowWithEngine := filepath.Join(workflowsDir, "with-engine.md")
		err = os.WriteFile(workflowWithEngine, []byte(`---
engine: copilot
on: push
---
# With Engine`), 0644)
		require.NoError(t, err, "Should write workflow file")

		workflowWithoutEngine := filepath.Join(workflowsDir, "without-engine.md")
		err = os.WriteFile(workflowWithoutEngine, []byte(`---
on: push
---
# Without Engine`), 0644)
		require.NoError(t, err, "Should write workflow file")

		secrets := getSecretsRequirementsForWorkflows([]string{workflowWithEngine, workflowWithoutEngine})

		// Count engine secrets (should only have Copilot)
		engineSecretCount := 0
		for _, secret := range secrets {
			if secret.IsEngineSecret {
				engineSecretCount++
				assert.Equal(t, "COPILOT_GITHUB_TOKEN", secret.Name, "Should only have Copilot secret")
			}
		}

		assert.Equal(t, 1, engineSecretCount, "Should have exactly one engine secret")
	})
}
