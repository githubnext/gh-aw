package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestActionFolderFeature_Integration tests the action-folder feature end-to-end
func TestActionFolderFeature_Integration(t *testing.T) {
	tests := []struct {
		name               string
		workflowContent    string
		actionMode         workflow.ActionMode
		expectedFolders    []string
		shouldHaveCheckout bool
	}{
		{
			name: "claude engine auto-adds .claude folder",
			workflowContent: `---
name: Test Claude Engine
engine: claude
on: issues
---
Test workflow with Claude engine`,
			actionMode:         workflow.ActionModeDev,
			expectedFolders:    []string{"actions", ".claude"},
			shouldHaveCheckout: true,
		},
		{
			name: "codex engine auto-adds .codex folder",
			workflowContent: `---
name: Test Codex Engine
engine: codex
on: issues
---
Test workflow with Codex engine`,
			actionMode:         workflow.ActionModeDev,
			expectedFolders:    []string{"actions", ".codex"},
			shouldHaveCheckout: true,
		},
		{
			name: "copilot engine does not add additional folder",
			workflowContent: `---
name: Test Copilot Engine
engine: copilot
on: issues
---
Test workflow with Copilot engine`,
			actionMode:         workflow.ActionModeDev,
			expectedFolders:    []string{"actions"},
			shouldHaveCheckout: true,
		},
		{
			name: "single custom folder in dev mode",
			workflowContent: `---
name: Test Action Folder
engine: copilot
on: issues
features:
  action-folder: custom-actions
---
Test workflow`,
			actionMode:         workflow.ActionModeDev,
			expectedFolders:    []string{"actions", "custom-actions"},
			shouldHaveCheckout: true,
		},
		{
			name: "multiple comma-separated folders in dev mode",
			workflowContent: `---
name: Test Multiple Folders
engine: copilot
on: issues
features:
  action-folder: "folder1, folder2, folder3"
---
Test workflow`,
			actionMode:         workflow.ActionModeDev,
			expectedFolders:    []string{"actions", "folder1", "folder2", "folder3"},
			shouldHaveCheckout: true,
		},
		{
			name: "no action-folder specified",
			workflowContent: `---
name: Test No Custom Folder
engine: copilot
on: issues
---
Test workflow`,
			actionMode:         workflow.ActionModeDev,
			expectedFolders:    []string{"actions"},
			shouldHaveCheckout: true,
		},
		{
			name: "action-folder in release mode (no checkout)",
			workflowContent: `---
name: Test Release Mode
engine: copilot
on: issues
features:
  action-folder: custom-actions
---
Test workflow`,
			actionMode:         workflow.ActionModeRelease,
			shouldHaveCheckout: false,
		},
		{
			name: "action-folder with action-tag (no checkout)",
			workflowContent: `---
name: Test With Action Tag
engine: copilot
on: issues
features:
  action-tag: a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
  action-folder: custom-actions
---
Test workflow`,
			actionMode:         workflow.ActionModeDev,
			shouldHaveCheckout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary workflow file
			tmpDir := t.TempDir()
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			err := os.WriteFile(workflowPath, []byte(tt.workflowContent), 0644)
			require.NoError(t, err, "Should create test workflow file")

			// Create the lock file path
			lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"

			// Compile the workflow
			compiler := workflow.NewCompiler(false, "", "test-version")
			compiler.SetActionMode(tt.actionMode)

			err = compiler.CompileWorkflow(workflowPath)
			require.NoError(t, err, "Should compile workflow successfully")

			// Read the generated lock file
			yamlContent, err := os.ReadFile(lockFilePath)
			require.NoError(t, err, "Should read generated lock file")

			yamlStr := string(yamlContent)

			// Check if checkout step exists
			hasCheckout := strings.Contains(yamlStr, "Checkout actions folder")

			if tt.shouldHaveCheckout {
				assert.True(t, hasCheckout, "Workflow should contain checkout actions folder step")
				assert.Contains(t, yamlStr, "sparse-checkout:", "Should have sparse-checkout configuration")

				// Verify each expected folder is in the sparse-checkout
				for _, folder := range tt.expectedFolders {
					assert.Contains(t, yamlStr, folder, "Sparse-checkout should include folder: %s", folder)
				}
			} else {
				assert.False(t, hasCheckout, "Workflow should NOT contain checkout actions folder step")
			}
		})
	}
}

// TestActionFolderFeature_ArrayFormat tests the action-folder feature with array format
func TestActionFolderFeature_ArrayFormat(t *testing.T) {
	workflowContent := `---
name: Test Array Format
engine: copilot
on: issues
features:
  action-folder:
    - folder1
    - folder2
    - .github/custom
---
Test workflow with array format`

	// Create a temporary workflow file
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Should create test workflow file")

	// Create the lock file path
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test-version")
	compiler.SetActionMode(workflow.ActionModeDev)

	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "Should compile workflow successfully")

	// Read the generated lock file
	yamlContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Should read generated lock file")

	yamlStr := string(yamlContent)

	// Verify all folders are present
	expectedFolders := []string{"actions", "folder1", "folder2", ".github/custom"}
	for _, folder := range expectedFolders {
		assert.Contains(t, yamlStr, folder, "Should include folder: %s", folder)
	}
}
