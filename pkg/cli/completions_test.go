package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkflowDescription(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	tests := []struct {
		name        string
		content     string
		wantDesc    string
		description string
	}{
		{
			name: "workflow with description",
			content: `---
description: Daily CI workflow for testing
engine: copilot
---

# Test workflow
`,
			wantDesc:    "Daily CI workflow for testing",
			description: "Should extract description field",
		},
		{
			name: "workflow with long description",
			content: `---
description: This is a very long description that exceeds the sixty character limit and should be truncated
engine: copilot
---

# Test workflow
`,
			wantDesc:    "This is a very long description that exceeds the sixty ch...",
			description: "Should truncate description to 60 characters",
		},
		{
			name: "workflow with name but no description",
			content: `---
name: Production deployment workflow
engine: copilot
---

# Test workflow
`,
			wantDesc:    "Production deployment workflow",
			description: "Should fallback to name field",
		},
		{
			name: "workflow with neither description nor name",
			content: `---
engine: copilot
---

# Test workflow
`,
			wantDesc:    "",
			description: "Should return empty string",
		},
		{
			name: "workflow without frontmatter",
			content: `# Test workflow

This is a test workflow without frontmatter.
`,
			wantDesc:    "",
			description: "Should return empty string for no frontmatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test workflow file
			testFile := filepath.Join(workflowsDir, "test-workflow.md")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			// Test getWorkflowDescription
			desc := getWorkflowDescription(testFile)
			assert.Equal(t, tt.wantDesc, desc, tt.description)

			// Clean up
			require.NoError(t, os.Remove(testFile))
		})
	}
}

func TestCompleteWorkflowNamesWithDescriptions(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Create test workflow files with descriptions
	workflows := map[string]string{
		"test-workflow.md": `---
description: Daily CI workflow for testing
engine: copilot
---

Test workflow content
`,
		"ci-doctor.md": `---
description: Automated CI health checks
engine: copilot
---

CI doctor workflow
`,
		"weekly-research.md": `---
name: Research workflow
engine: copilot
---

Weekly research workflow
`,
		"no-desc.md": `---
engine: copilot
---

No description workflow
`,
	}

	for filename, content := range workflows {
		f, err := os.Create(filepath.Join(workflowsDir, filename))
		require.NoError(t, err)
		_, err = f.WriteString(content)
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
		wantCount  int
		wantNames  []string
		checkDescs bool
	}{
		{
			name:       "empty prefix returns all workflows with descriptions",
			toComplete: "",
			wantCount:  4,
			wantNames:  []string{"test-workflow", "ci-doctor", "weekly-research", "no-desc"},
			checkDescs: true,
		},
		{
			name:       "c prefix returns ci-doctor",
			toComplete: "c",
			wantCount:  1,
			wantNames:  []string{"ci-doctor"},
			checkDescs: true,
		},
		{
			name:       "test prefix returns test-workflow",
			toComplete: "test",
			wantCount:  1,
			wantNames:  []string{"test-workflow"},
			checkDescs: true,
		},
		{
			name:       "x prefix returns nothing",
			toComplete: "x",
			wantCount:  0,
			wantNames:  []string{},
			checkDescs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := CompleteWorkflowNames(cmd, nil, tt.toComplete)
			assert.Len(t, completions, tt.wantCount)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

			if tt.checkDescs && tt.wantCount > 0 {
				// Verify that completions are in the correct format
				for _, completion := range completions {
					// Check if it has a tab-separated description or is just a name
					parts := strings.Split(completion, "\t")
					assert.True(t, len(parts) == 1 || len(parts) == 2,
						"Completion should be 'name' or 'name\\tdescription', got: %s", completion)

					// Verify the name part matches expected names
					name := parts[0]
					found := false
					for _, expectedName := range tt.wantNames {
						if name == expectedName {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected name %s to be in %v", name, tt.wantNames)
				}
			}
		})
	}
}

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
