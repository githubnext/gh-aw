package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidEngineNames(t *testing.T) {
	engines := ValidEngineNames()

	// Verify the list is not empty
	assert.NotEmpty(t, engines, "Engine names list should not be empty")

	// Verify expected engines are present
	expectedEngines := []string{"copilot", "claude", "codex", "custom"}
	for _, expected := range expectedEngines {
		assert.Contains(t, engines, expected, "Expected engine '%s' to be in the list", expected)
	}
}

func TestCompleteEngineNames(t *testing.T) {
	cmd := &cobra.Command{}

	tests := []struct {
		name       string
		toComplete string
		wantLen    int
	}{
		{
			name:       "empty prefix returns all engines",
			toComplete: "",
			wantLen:    4, // copilot, claude, codex, custom
		},
		{
			name:       "c prefix returns claude, codex, copilot, custom",
			toComplete: "c",
			wantLen:    4,
		},
		{
			name:       "co prefix returns copilot, codex",
			toComplete: "co",
			wantLen:    2,
		},
		{
			name:       "cop prefix returns copilot",
			toComplete: "cop",
			wantLen:    1,
		},
		{
			name:       "x prefix returns nothing",
			toComplete: "x",
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := CompleteEngineNames(cmd, nil, tt.toComplete)
			assert.Len(t, completions, tt.wantLen)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func TestCompleteWorkflowNames(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Create test workflow files
	testWorkflows := []string{"test-workflow.md", "ci-doctor.md", "weekly-research.md"}
	for _, wf := range testWorkflows {
		f, err := os.Create(filepath.Join(workflowsDir, wf))
		require.NoError(t, err)
		f.Close()
	}

	// Change to the temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	cmd := &cobra.Command{}

	tests := []struct {
		name       string
		toComplete string
		wantLen    int
	}{
		{
			name:       "empty prefix returns all workflows",
			toComplete: "",
			wantLen:    3,
		},
		{
			name:       "c prefix returns ci-doctor",
			toComplete: "c",
			wantLen:    1,
		},
		{
			name:       "test prefix returns test-workflow",
			toComplete: "test",
			wantLen:    1,
		},
		{
			name:       "x prefix returns nothing",
			toComplete: "x",
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := CompleteWorkflowNames(cmd, nil, tt.toComplete)
			assert.Len(t, completions, tt.wantLen)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func TestCompleteWorkflowNamesNoWorkflowsDir(t *testing.T) {
	// Create a temporary directory without .github/workflows
	tmpDir := t.TempDir()

	// Change to the temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	cmd := &cobra.Command{}

	completions, directive := CompleteWorkflowNames(cmd, nil, "")
	assert.Empty(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompleteDirectories(t *testing.T) {
	cmd := &cobra.Command{}

	completions, directive := CompleteDirectories(cmd, nil, "")
	assert.Nil(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveFilterDirs, directive)
}

func TestRegisterEngineFlagCompletion(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("engine", "e", "", "AI engine")

	// This should not panic
	RegisterEngineFlagCompletion(cmd)
}

func TestRegisterDirFlagCompletion(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("dir", "d", "", "Directory")

	// This should not panic
	RegisterDirFlagCompletion(cmd, "dir")
}
