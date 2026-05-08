//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWorkflowYAML(t *testing.T) {
	t.Run("reads valid workflow yaml from absolute path", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "workflow.yml")
		content := "on:\n  workflow_call: {}\n"
		require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0644), "Should write workflow file")

		workflow, err := readWorkflowYAML(workflowPath)
		require.NoError(t, err, "Should read workflow YAML without error")
		require.NotNil(t, workflow, "Should return parsed workflow map")

		onSection, ok := workflow["on"].(map[string]any)
		require.True(t, ok, "Parsed workflow should contain on map")
		_, hasWorkflowCall := onSection["workflow_call"]
		assert.True(t, hasWorkflowCall, "Parsed workflow should include workflow_call trigger")
	})

	t.Run("rejects relative path", func(t *testing.T) {
		workflow, err := readWorkflowYAML("workflow.yml")
		assert.Nil(t, workflow, "Relative paths should not return workflow data")
		require.Error(t, err, "Relative paths should return an error")
		assert.Equal(t, "workflow path must be absolute: workflow.yml", err.Error(), "Should return path validation error")
	})

	t.Run("returns parse error for invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "invalid.yml")
		invalid := "on:\n  workflow_call: [\n"
		require.NoError(t, os.WriteFile(workflowPath, []byte(invalid), 0644), "Should write invalid workflow file")

		workflow, err := readWorkflowYAML(workflowPath)
		assert.Nil(t, workflow, "Invalid YAML should not return workflow data")
		require.Error(t, err, "Invalid YAML should return an error")
		assert.Contains(t, err.Error(), "failed to parse workflow file", "Should wrap parse error consistently")
	})
}
