package inventory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple workflow file",
			path:     "my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "workflow with path",
			path:     ".github/workflows/my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "lock file",
			path:     "my-workflow.lock.yml",
			expected: "my-workflow",
		},
		{
			name:     "lock file with path",
			path:     ".github/workflows/my-workflow.lock.yml",
			expected: "my-workflow",
		},
		{
			name:     "campaign workflow",
			path:     "security.campaign.md",
			expected: "security",
		},
		{
			name:     "campaign lock file",
			path:     "security.campaign.lock.yml",
			expected: "security",
		},
		{
			name:     "generated campaign orchestrator",
			path:     "security.campaign.g.md",
			expected: "security",
		},
		{
			name:     "workflow with multiple dots",
			path:     "test.lock.yml",
			expected: "test",
		},
		{
			name:     "workflow with dashes",
			path:     "my-test-workflow.md",
			expected: "my-test-workflow",
		},
		{
			name:     "workflow with underscores",
			path:     "my_test_workflow.md",
			expected: "my_test_workflow",
		},
		{
			name:     "just filename no extension",
			path:     "workflow",
			expected: "workflow",
		},
		{
			name:     "absolute path",
			path:     "/home/user/.github/workflows/deploy.md",
			expected: "deploy",
		},
		{
			name:     "campaign with path",
			path:     ".github/workflows/update-deps.campaign.md",
			expected: "update-deps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractWorkflowName(tt.path)
			assert.Equal(t, tt.expected, result, "ExtractWorkflowName(%q) should return %q", tt.path, tt.expected)
		})
	}
}

func TestNormalizeWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain workflow name",
			input:    "my-workflow",
			expected: "my-workflow",
		},
		{
			name:     "workflow name with .md",
			input:    "my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with workflow file",
			input:    ".github/workflows/my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "lock file reference",
			input:    "my-workflow.lock.yml",
			expected: "my-workflow",
		},
		{
			name:     "campaign workflow",
			input:    "security.campaign.md",
			expected: "security",
		},
		{
			name:     "relative path",
			input:    "workflows/deploy.md",
			expected: "deploy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWorkflowName(tt.input)
			assert.Equal(t, tt.expected, result, "NormalizeWorkflowName(%q) should return %q", tt.input, tt.expected)
		})
	}
}

func TestGetWorkflowPath(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		workflowsDir string
		expected     string
	}{
		{
			name:         "default directory",
			workflowName: "my-workflow",
			workflowsDir: "",
			expected:     ".github/workflows/my-workflow.md",
		},
		{
			name:         "custom directory",
			workflowName: "my-workflow",
			workflowsDir: "/custom/path",
			expected:     "/custom/path/my-workflow.md",
		},
		{
			name:         "workflow name with .md extension",
			workflowName: "my-workflow.md",
			workflowsDir: "",
			expected:     ".github/workflows/my-workflow.md",
		},
		{
			name:         "workflow name with path",
			workflowName: ".github/workflows/my-workflow.md",
			workflowsDir: "",
			expected:     ".github/workflows/my-workflow.md",
		},
		{
			name:         "relative custom directory",
			workflowName: "deploy",
			workflowsDir: "workflows",
			expected:     "workflows/deploy.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorkflowPath(tt.workflowName, tt.workflowsDir)
			assert.Equal(t, tt.expected, result, "GetWorkflowPath(%q, %q) should return %q", tt.workflowName, tt.workflowsDir, tt.expected)
		})
	}
}

func TestGetLockFilePath(t *testing.T) {
	tests := []struct {
		name         string
		workflowPath string
		workflowsDir string
		expected     string
	}{
		{
			name:         "regular workflow",
			workflowPath: "my-workflow.md",
			workflowsDir: "",
			expected:     ".github/workflows/my-workflow.lock.yml",
		},
		{
			name:         "regular workflow with path",
			workflowPath: ".github/workflows/my-workflow.md",
			workflowsDir: "",
			expected:     ".github/workflows/my-workflow.lock.yml",
		},
		{
			name:         "campaign workflow",
			workflowPath: "security.campaign.md",
			workflowsDir: "",
			expected:     ".github/workflows/security.campaign.lock.yml",
		},
		{
			name:         "campaign workflow with path",
			workflowPath: ".github/workflows/security.campaign.md",
			workflowsDir: "",
			expected:     ".github/workflows/security.campaign.lock.yml",
		},
		{
			name:         "generated campaign orchestrator",
			workflowPath: "security.campaign.g.md",
			workflowsDir: "",
			expected:     ".github/workflows/security.campaign.lock.yml",
		},
		{
			name:         "generated campaign orchestrator with path",
			workflowPath: ".github/workflows/update-deps.campaign.g.md",
			workflowsDir: "",
			expected:     ".github/workflows/update-deps.campaign.lock.yml",
		},
		{
			name:         "custom workflows directory",
			workflowPath: "deploy.md",
			workflowsDir: "/custom/workflows",
			expected:     "/custom/workflows/deploy.lock.yml",
		},
		{
			name:         "workflow name only",
			workflowPath: "test-workflow.md",
			workflowsDir: "workflows",
			expected:     "workflows/test-workflow.lock.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLockFilePath(tt.workflowPath, tt.workflowsDir)
			assert.Equal(t, tt.expected, result, "GetLockFilePath(%q, %q) should return %q", tt.workflowPath, tt.workflowsDir, tt.expected)
		})
	}
}

func TestIsWorkflowFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "regular workflow file",
			filename: "my-workflow.md",
			expected: true,
		},
		{
			name:     "README.md uppercase",
			filename: "README.md",
			expected: false,
		},
		{
			name:     "readme.md lowercase",
			filename: "readme.md",
			expected: false,
		},
		{
			name:     "ReadMe.md mixed case",
			filename: "ReadMe.md",
			expected: false,
		},
		{
			name:     "README with prefix",
			filename: "README-workflow.md",
			expected: true,
		},
		{
			name:     "README with suffix",
			filename: "workflow-README.md",
			expected: true,
		},
		{
			name:     "path with README.md",
			filename: ".github/workflows/README.md",
			expected: false,
		},
		{
			name:     "path with workflow",
			filename: ".github/workflows/deploy.md",
			expected: true,
		},
		{
			name:     "campaign file",
			filename: "security.campaign.md",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWorkflowFile(tt.filename)
			assert.Equal(t, tt.expected, result, "isWorkflowFile(%q) should return %v", tt.filename, tt.expected)
		})
	}
}

func TestListWorkflowFiles(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create test workflow files
	testFiles := map[string]string{
		"workflow1.md":            "---\nengine: copilot\n---\n# Workflow 1",
		"workflow2.md":            "---\nengine: claude\n---\n# Workflow 2",
		"README.md":               "# Workflows Documentation",
		"security.campaign.md":    "---\nengine: copilot\n---\n# Security Campaign",
		"update-deps.campaign.md": "---\nengine: copilot\n---\n# Update Dependencies",
		"security.campaign.g.md":  "---\nengine: copilot\n---\n# Generated Security",
		"deploy-prod.md":          "---\nengine: codex\n---\n# Deploy Production",
		"test-workflow-README.md": "---\nengine: copilot\n---\n# Test with README suffix",
		"README-test-workflow.md": "---\nengine: copilot\n---\n# README prefix workflow",
	}

	for filename, content := range testFiles {
		path := filepath.Join(workflowsDir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err, "Failed to write test file %s", filename)
	}

	tests := []struct {
		name             string
		includeCampaigns bool
		includeGenerated bool
		expectedCount    int
		expectedNames    []string
		excludedNames    []string
	}{
		{
			name:             "regular workflows only",
			includeCampaigns: false,
			includeGenerated: false,
			expectedCount:    5,
			expectedNames:    []string{"workflow1", "workflow2", "deploy-prod", "test-workflow-README", "README-test-workflow"},
			excludedNames:    []string{"security", "update-deps"},
		},
		{
			name:             "include campaigns",
			includeCampaigns: true,
			includeGenerated: false,
			expectedCount:    7,
			expectedNames:    []string{"workflow1", "workflow2", "security", "update-deps", "deploy-prod"},
			excludedNames:    []string{},
		},
		{
			name:             "include generated",
			includeCampaigns: false,
			includeGenerated: true,
			expectedCount:    6,
			expectedNames:    []string{"workflow1", "workflow2", "security", "deploy-prod"},
			excludedNames:    []string{"update-deps"},
		},
		{
			name:             "include everything",
			includeCampaigns: true,
			includeGenerated: true,
			expectedCount:    8,
			expectedNames:    []string{"workflow1", "workflow2", "security", "update-deps", "deploy-prod"},
			excludedNames:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflows, err := ListWorkflowFiles(workflowsDir, tt.includeCampaigns, tt.includeGenerated)
			require.NoError(t, err, "ListWorkflowFiles should not return error")

			assert.Len(t, workflows, tt.expectedCount, "Should find %d workflows", tt.expectedCount)

			// Extract workflow names for easier checking
			var foundNames []string
			for _, wf := range workflows {
				foundNames = append(foundNames, wf.Name)
			}

			// Check expected names are present
			for _, expectedName := range tt.expectedNames {
				assert.Contains(t, foundNames, expectedName, "Should include workflow %s", expectedName)
			}

			// Check excluded names are not present
			for _, excludedName := range tt.excludedNames {
				assert.NotContains(t, foundNames, excludedName, "Should not include workflow %s", excludedName)
			}

			// Verify README.md is never included
			assert.NotContains(t, foundNames, "README", "Should never include README.md")
		})
	}
}

func TestListWorkflowFiles_NonExistentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")

	workflows, err := ListWorkflowFiles(nonExistentDir, false, false)
	require.Error(t, err, "Should return error for non-existent directory")
	assert.Nil(t, workflows, "Should return nil workflows on error")
}

func TestWorkflowFile_Fields(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create test files
	testFile := filepath.Join(workflowsDir, "test.md")
	err = os.WriteFile(testFile, []byte("---\nengine: copilot\n---\n# Test"), 0644)
	require.NoError(t, err, "Failed to write test file")

	campaignFile := filepath.Join(workflowsDir, "campaign.campaign.md")
	err = os.WriteFile(campaignFile, []byte("---\nengine: copilot\n---\n# Campaign"), 0644)
	require.NoError(t, err, "Failed to write campaign file")

	workflows, err := ListWorkflowFiles(workflowsDir, true, false)
	require.NoError(t, err, "ListWorkflowFiles should not return error")
	require.Len(t, workflows, 2, "Should find 2 workflows")

	// Check regular workflow
	regularWorkflow := findWorkflowByName(workflows, "test")
	require.NotNil(t, regularWorkflow, "Should find regular workflow")
	assert.Equal(t, "test", regularWorkflow.Name)
	assert.Equal(t, testFile, regularWorkflow.Path)
	assert.Equal(t, WorkflowTypeRegular, regularWorkflow.Type)
	assert.Equal(t, filepath.Join(workflowsDir, "test.lock.yml"), regularWorkflow.LockPath)

	// Check campaign workflow
	campaignWorkflow := findWorkflowByName(workflows, "campaign")
	require.NotNil(t, campaignWorkflow, "Should find campaign workflow")
	assert.Equal(t, "campaign", campaignWorkflow.Name)
	assert.Equal(t, campaignFile, campaignWorkflow.Path)
	assert.Equal(t, WorkflowTypeCampaign, campaignWorkflow.Type)
	assert.Equal(t, filepath.Join(workflowsDir, "campaign.campaign.lock.yml"), campaignWorkflow.LockPath)
}

// Helper function to find workflow by name in slice
func findWorkflowByName(workflows []WorkflowFile, name string) *WorkflowFile {
	for i := range workflows {
		if workflows[i].Name == name {
			return &workflows[i]
		}
	}
	return nil
}
