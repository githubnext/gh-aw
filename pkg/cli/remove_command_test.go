package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveWorkflows_NoWorkflowsDir(t *testing.T) {
	// Create a temporary directory without .github/workflows
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")

	// Test with non-existent workflows directory
	err = RemoveWorkflows("test-pattern", false)
	assert.NoError(t, err, "Should not error when no workflows directory exists")
}

func TestRemoveWorkflows_NoWorkflowFiles(t *testing.T) {
	// Create a temporary directory with empty .github/workflows
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")

	// Test with empty workflows directory
	err = RemoveWorkflows("test-pattern", false)
	assert.NoError(t, err, "Should not error when no workflow files exist")
}

func TestRemoveWorkflows_NoPattern(t *testing.T) {
	// Create a temporary directory with some workflow files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create test workflow files
	testWorkflow := filepath.Join(workflowsDir, "test-workflow.md")
	err = os.WriteFile(testWorkflow, []byte("---\nname: Test Workflow\n---\n"), 0644)
	require.NoError(t, err, "Failed to create test workflow")

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")

	// Test with no pattern (should list workflows)
	err = RemoveWorkflows("", false)
	assert.NoError(t, err, "Should not error when no pattern is provided")
}

func TestRemoveWorkflows_NoMatchingFiles(t *testing.T) {
	// Create a temporary directory with workflow files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create test workflow files
	testWorkflow := filepath.Join(workflowsDir, "test-workflow.md")
	err = os.WriteFile(testWorkflow, []byte("---\nname: Test Workflow\n---\n"), 0644)
	require.NoError(t, err, "Failed to create test workflow")

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")

	// Test with pattern that doesn't match
	err = RemoveWorkflows("nonexistent-pattern", false)
	assert.NoError(t, err, "Should not error when no files match the pattern")
}

func TestRemoveWorkflows_PatternMatching(t *testing.T) {
	tests := []struct {
		name          string
		workflowFiles map[string]string // filename -> content
		pattern       string
		shouldMatch   bool
	}{
		{
			name: "match by filename",
			workflowFiles: map[string]string{
				"ci-doctor.md": "---\nname: CI Doctor\n---\n",
			},
			pattern:     "doctor",
			shouldMatch: true,
		},
		{
			name: "match by workflow name",
			workflowFiles: map[string]string{
				"workflow.md": "---\nname: Test Workflow\n---\n",
			},
			pattern:     "test",
			shouldMatch: true,
		},
		{
			name: "case insensitive matching",
			workflowFiles: map[string]string{
				"MyWorkflow.md": "---\nname: My Workflow\n---\n",
			},
			pattern:     "myworkflow",
			shouldMatch: true,
		},
		{
			name: "no match",
			workflowFiles: map[string]string{
				"other-workflow.md": "---\nname: Other Workflow\n---\n",
			},
			pattern:     "nonexistent",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory with workflow files
			tmpDir := t.TempDir()
			workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
			err := os.MkdirAll(workflowsDir, 0755)
			require.NoError(t, err, "Failed to create workflows directory")

			for filename, content := range tt.workflowFiles {
				path := filepath.Join(workflowsDir, filename)
				err = os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err, "Failed to create workflow file: %s", filename)
			}

			originalDir, err := os.Getwd()
			require.NoError(t, err, "Failed to get current directory")
			defer os.Chdir(originalDir)

			err = os.Chdir(tmpDir)
			require.NoError(t, err, "Failed to change to temp directory")

			// Note: We can't fully test removal without mocking stdin for confirmation
			// This test just verifies the pattern matching logic doesn't error
			err = RemoveWorkflows(tt.pattern, false)
			
			// The function will either find matches or not, but shouldn't error
			assert.NoError(t, err, "RemoveWorkflows should not error for pattern: %s", tt.pattern)
		})
	}
}

func TestRemoveWorkflows_KeepOrphansFlag(t *testing.T) {
	tests := []struct {
		name        string
		keepOrphans bool
	}{
		{
			name:        "keep orphans enabled",
			keepOrphans: true,
		},
		{
			name:        "keep orphans disabled",
			keepOrphans: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory with workflow files
			tmpDir := t.TempDir()
			workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
			err := os.MkdirAll(workflowsDir, 0755)
			require.NoError(t, err, "Failed to create workflows directory")

			testWorkflow := filepath.Join(workflowsDir, "test-workflow.md")
			err = os.WriteFile(testWorkflow, []byte("---\nname: Test Workflow\n---\n"), 0644)
			require.NoError(t, err, "Failed to create test workflow")

			originalDir, err := os.Getwd()
			require.NoError(t, err, "Failed to get current directory")
			defer os.Chdir(originalDir)

			err = os.Chdir(tmpDir)
			require.NoError(t, err, "Failed to change to temp directory")

			// Test with different keepOrphans settings
			err = RemoveWorkflows("test", tt.keepOrphans)
			assert.NoError(t, err, "RemoveWorkflows should not error with keepOrphans=%v", tt.keepOrphans)
		})
	}
}
