//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shouldSkipFirewallWorkflow returns true if the workflow filename contains ".firewall"
// and the firewall feature is not enabled. This helper should be used in tests that
// iterate over actual workflow files in .github/workflows to skip firewall workflows
// when the GH_AW_FEATURES environment variable doesn't include "firewall".
func shouldSkipFirewallWorkflow(workflowName string) bool {
	return strings.Contains(workflowName, ".firewall") && !isFeatureEnabled(constants.FeatureFlag("firewall"), nil)
}

// setupWorkflowDir creates a temporary directory with the workflows sub-directory,
// changes the working directory to it, and returns the path to the workflows directory.
func setupWorkflowDir(t *testing.T) string {
	t.Helper()
	tempDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tempDir, constants.GetWorkflowDir())
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "failed to create workflows directory")
	t.Chdir(tempDir)
	return workflowsDir
}

func TestNormalizeWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain name",
			input:    "weekly-research",
			expected: "weekly-research",
		},
		{
			name:     "name with .md extension",
			input:    "weekly-research.md",
			expected: "weekly-research",
		},
		{
			name:     "name with .lock.yml extension",
			input:    "weekly-research.lock.yml",
			expected: "weekly-research",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "name with multiple dots",
			input:    "daily-test.coverage.md",
			expected: "daily-test.coverage",
		},
		{
			name:     "name ending with partial extension",
			input:    "workflow.lock",
			expected: "workflow.lock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringutil.NormalizeWorkflowName(tt.input)
			assert.Equal(t, tt.expected, result, "NormalizeWorkflowName(%q) should return %q", tt.input, tt.expected)
		})
	}
}

func TestResolveWorkflowName(t *testing.T) {
	workflowsDir := setupWorkflowDir(t)

	// Create sample workflow files
	testWorkflows := map[string]string{
		"weekly-research": "Weekly Research",
		"daily-plan":      "Daily Plan",
		"issue-triage":    "Issue Triage",
	}
	for workflowID, workflowName := range testWorkflows {
		mdFile := filepath.Join(workflowsDir, workflowID+".md")
		lockFile := filepath.Join(workflowsDir, workflowID+".lock.yml")

		require.NoError(t, os.WriteFile(mdFile, []byte("# "+workflowID+"\nSome content"), 0644), "failed to write workflow markdown file")

		lockContent := "name: \"" + workflowName + "\"\non: push\n"
		require.NoError(t, os.WriteFile(lockFile, []byte(lockContent), 0644), "failed to write workflow lock file")
	}

	tests := []struct {
		name                 string
		workflowInput        string
		expectedWorkflowName string
		expectError          bool
	}{
		{
			name:                 "valid workflow ID",
			workflowInput:        "weekly-research",
			expectedWorkflowName: "Weekly Research",
			expectError:          false,
		},
		{
			name:                 "valid workflow ID with .md extension",
			workflowInput:        "daily-plan.md",
			expectedWorkflowName: "Daily Plan",
			expectError:          false,
		},
		{
			name:                 "valid workflow ID with .lock.yml extension",
			workflowInput:        "issue-triage.lock.yml",
			expectedWorkflowName: "Issue Triage",
			expectError:          false,
		},
		{
			name:                 "empty workflow ID",
			workflowInput:        "",
			expectedWorkflowName: "",
			expectError:          false,
		},
		{
			name:          "non-existent workflow ID",
			workflowInput: "non-existent",
			expectError:   true,
		},
		{
			name:          "non-existent workflow ID with extension",
			workflowInput: "non-existent.md",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveWorkflowName(tt.workflowInput)

			if tt.expectError {
				assert.Error(t, err, "ResolveWorkflowName(%q) should return an error", tt.workflowInput)
			} else {
				require.NoError(t, err, "ResolveWorkflowName(%q) should not return an error", tt.workflowInput)
				assert.Equal(t, tt.expectedWorkflowName, result, "ResolveWorkflowName(%q) should return correct workflow name", tt.workflowInput)
			}
		})
	}
}

func TestResolveWorkflowName_MissingLockFile(t *testing.T) {
	workflowsDir := setupWorkflowDir(t)

	// Create only the .md file, but not the .lock.yml file
	mdFile := filepath.Join(workflowsDir, "incomplete-workflow.md")
	require.NoError(t, os.WriteFile(mdFile, []byte("# Incomplete Workflow\nSome content"), 0644), "failed to write workflow markdown file")

	// Test that it returns an error when lock file is missing
	_, err := ResolveWorkflowName("incomplete-workflow")
	require.Error(t, err, "ResolveWorkflowName should return an error when lock file is missing")
	assert.Contains(t, err.Error(), "Run 'gh aw compile'", "error should mention compilation when lock file is missing")
}

func TestResolveWorkflowName_InvalidYAML(t *testing.T) {
	workflowsDir := setupWorkflowDir(t)

	// Create workflow with invalid YAML
	mdFile := filepath.Join(workflowsDir, "invalid-yaml.md")
	lockFile := filepath.Join(workflowsDir, "invalid-yaml.lock.yml")

	require.NoError(t, os.WriteFile(mdFile, []byte("# Invalid YAML\nSome content"), 0644), "failed to write workflow markdown file")
	require.NoError(t, os.WriteFile(lockFile, []byte("invalid: yaml: content: ["), 0644), "failed to write invalid YAML lock file")

	// Test that it returns an error when YAML is invalid
	_, err := ResolveWorkflowName("invalid-yaml")
	require.Error(t, err, "ResolveWorkflowName should return an error when YAML is invalid")
	assert.Contains(t, err.Error(), "failed to parse YAML", "error should mention YAML parsing failure")
}

func TestResolveWorkflowName_MissingNameField(t *testing.T) {
	workflowsDir := setupWorkflowDir(t)

	// Create workflow with valid YAML but missing name field
	mdFile := filepath.Join(workflowsDir, "no-name.md")
	lockFile := filepath.Join(workflowsDir, "no-name.lock.yml")

	require.NoError(t, os.WriteFile(mdFile, []byte("# No Name\nSome content"), 0644), "failed to write workflow markdown file")
	require.NoError(t, os.WriteFile(lockFile, []byte("on: push\njobs: {}\n"), 0644), "failed to write lock file without name field")

	// Test that it returns an error when name field is missing
	_, err := ResolveWorkflowName("no-name")
	require.Error(t, err, "ResolveWorkflowName should return an error when name field is missing")
	assert.Contains(t, err.Error(), "workflow name not found", "error should mention missing workflow name")
}

func TestResolveWorkflowName_ExistingAgenticWorkflow(t *testing.T) {
	// The test is run from the project root where go.mod is located.
	// If running from a subdirectory, walk up to find the project root.
	currentDir, err := os.Getwd()
	require.NoError(t, err, "cannot determine current directory")

	projectRoot := currentDir
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(projectRoot, constants.GetWorkflowDir())); err == nil {
				break // Found project root
			}
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Skipf("Cannot find project root with go.mod and .github/workflows")
		}
		projectRoot = parent
	}

	t.Chdir(projectRoot)

	// Test with known existing workflows - we'll read the actual name from lock files
	knownWorkflows := []string{"weekly-research", "daily-plan", "issue-triage"}

	for _, workflow := range knownWorkflows {
		t.Run("existing_"+workflow, func(t *testing.T) {
			workflowsDir := ".github/workflows"
			mdFile := filepath.Join(workflowsDir, workflow+".md")
			lockFile := filepath.Join(workflowsDir, workflow+".lock.yml")

			// Skip .firewall workflows unless the firewall feature is enabled
			if shouldSkipFirewallWorkflow(workflow) {
				t.Skipf("Skipping firewall workflow %s (feature not enabled)", workflow)
			}

			// Check if both files exist
			if _, err := os.Stat(mdFile); err != nil {
				t.Skipf("Workflow %s.md not found, skipping", workflow)
			}
			if _, err := os.Stat(lockFile); err != nil {
				t.Skipf("Workflow %s.lock.yml not found, skipping", workflow)
			}

			// Test resolving the workflow
			result, err := ResolveWorkflowName(workflow)
			require.NoError(t, err, "ResolveWorkflowName should resolve existing workflow %q without error", workflow)

			// The result should be the actual workflow name from the YAML, not the filename
			assert.NotEmpty(t, result, "ResolveWorkflowName should return a non-empty name for existing workflow %q", workflow)
			assert.NotEqual(t, workflow+".lock.yml", result, "ResolveWorkflowName should return the YAML name, not the lock filename, for %q", workflow)

			// Test with different input formats - should all return the same workflow name
			result2, err := ResolveWorkflowName(workflow + ".md")
			require.NoError(t, err, "ResolveWorkflowName should resolve %q with .md extension without error", workflow)
			assert.Equal(t, result, result2, "ResolveWorkflowName should return the same name for %q with .md extension", workflow)

			result3, err := ResolveWorkflowName(workflow + ".lock.yml")
			require.NoError(t, err, "ResolveWorkflowName should resolve %q with .lock.yml extension without error", workflow)
			assert.Equal(t, result, result3, "ResolveWorkflowName should return the same name for %q with .lock.yml extension", workflow)
		})
	}
}

func TestShouldSkipFirewallWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		featureValue string
		shouldSkip   bool
	}{
		{
			name:         "regular workflow without firewall feature",
			workflowName: "weekly-research",
			featureValue: "",
			shouldSkip:   false,
		},
		{
			name:         "firewall workflow without firewall feature",
			workflowName: "dev.firewall",
			featureValue: "",
			shouldSkip:   true,
		},
		{
			name:         "firewall workflow with firewall feature enabled",
			workflowName: "dev.firewall",
			featureValue: "firewall",
			shouldSkip:   false,
		},
		{
			name:         "firewall workflow with multiple features including firewall",
			workflowName: "test.firewall.workflow",
			featureValue: "feature1,firewall,feature2",
			shouldSkip:   false,
		},
		{
			name:         "regular workflow with firewall feature enabled",
			workflowName: "daily-plan",
			featureValue: "firewall",
			shouldSkip:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the environment variable
			if tt.featureValue != "" {
				t.Setenv("GH_AW_FEATURES", tt.featureValue)
			}

			result := shouldSkipFirewallWorkflow(tt.workflowName)
			assert.Equal(t, tt.shouldSkip, result,
				"shouldSkipFirewallWorkflow(%q) with GH_AW_FEATURES=%q should return %v",
				tt.workflowName, tt.featureValue, tt.shouldSkip)
		})
	}
}

func TestFindWorkflowName(t *testing.T) {
	workflowsDir := setupWorkflowDir(t)

	// Create sample workflow files
	testWorkflows := map[string]string{
		"ci-failure-doctor": "CI Failure Doctor",
		"weekly-research":   "Weekly Research",
		"daily-plan":        "Daily Plan",
	}
	for workflowID, displayName := range testWorkflows {
		mdFile := filepath.Join(workflowsDir, workflowID+".md")
		lockFile := filepath.Join(workflowsDir, workflowID+".lock.yml")

		require.NoError(t, os.WriteFile(mdFile, []byte("# "+workflowID+"\nSome content"), 0644), "failed to write workflow markdown file")

		lockContent := "name: \"" + displayName + "\"\non: push\n"
		require.NoError(t, os.WriteFile(lockFile, []byte(lockContent), 0644), "failed to write workflow lock file")
	}

	tests := []struct {
		name         string
		input        string
		expectedName string
		expectError  bool
	}{
		{
			name:         "exact workflow ID match",
			input:        "ci-failure-doctor",
			expectedName: "CI Failure Doctor",
			expectError:  false,
		},
		{
			name:         "case-insensitive workflow ID match",
			input:        "CI-FAILURE-DOCTOR",
			expectedName: "CI Failure Doctor",
			expectError:  false,
		},
		{
			name:         "mixed case workflow ID match",
			input:        "Ci-Failure-Doctor",
			expectedName: "CI Failure Doctor",
			expectError:  false,
		},
		{
			name:         "exact display name match",
			input:        "CI Failure Doctor",
			expectedName: "CI Failure Doctor",
			expectError:  false,
		},
		{
			name:         "case-insensitive display name match",
			input:        "ci failure doctor",
			expectedName: "CI Failure Doctor",
			expectError:  false,
		},
		{
			name:         "workflow ID with .md extension",
			input:        "weekly-research.md",
			expectedName: "Weekly Research",
			expectError:  false,
		},
		{
			name:         "workflow ID with .lock.yml extension",
			input:        "daily-plan.lock.yml",
			expectedName: "Daily Plan",
			expectError:  false,
		},
		{
			name:        "non-existent workflow",
			input:       "non-existent-workflow",
			expectError: true,
		},
		{
			name:         "empty input",
			input:        "",
			expectedName: "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindWorkflowName(tt.input)

			if tt.expectError {
				assert.Error(t, err, "FindWorkflowName(%q) should return an error", tt.input)
			} else {
				require.NoError(t, err, "FindWorkflowName(%q) should not return an error", tt.input)
				assert.Equal(t, tt.expectedName, result, "FindWorkflowName(%q) should return correct workflow name", tt.input)
			}
		})
	}
}

func TestGetAllWorkflows(t *testing.T) {
	workflowsDir := setupWorkflowDir(t)

	// Create sample workflow files
	testWorkflows := map[string]string{
		"workflow-one":   "Workflow One",
		"workflow-two":   "Workflow Two",
		"workflow-three": "Workflow Three",
	}
	for workflowID, displayName := range testWorkflows {
		lockFile := filepath.Join(workflowsDir, workflowID+".lock.yml")
		lockContent := "name: \"" + displayName + "\"\non: push\n"
		require.NoError(t, os.WriteFile(lockFile, []byte(lockContent), 0644), "failed to write workflow lock file")
	}

	// Get all workflows
	workflows, err := GetAllWorkflows()
	require.NoError(t, err, "GetAllWorkflows should not return an error")

	// Check count
	assert.Len(t, workflows, len(testWorkflows), "GetAllWorkflows should return the expected number of workflows")

	// Check that all workflows are present
	workflowMap := make(map[string]string)
	for _, wf := range workflows {
		workflowMap[wf.WorkflowID] = wf.DisplayName
	}

	for workflowID, expectedDisplayName := range testWorkflows {
		actualDisplayName, exists := workflowMap[workflowID]
		assert.True(t, exists, "workflow ID %q should be present in results", workflowID)
		assert.Equal(t, expectedDisplayName, actualDisplayName, "workflow ID %q should have correct display name", workflowID)
	}
}

func TestGetWorkflowLockFileName(t *testing.T) {
	workflowsDir := setupWorkflowDir(t)

	// Create sample workflow files
	testWorkflows := map[string]string{
		"smoke-copilot": "Smoke Copilot",
		"weekly-plan":   "Weekly Plan",
	}
	for workflowID, displayName := range testWorkflows {
		mdFile := filepath.Join(workflowsDir, workflowID+".md")
		lockFile := filepath.Join(workflowsDir, workflowID+".lock.yml")

		require.NoError(t, os.WriteFile(mdFile, []byte("# "+workflowID+"\nSome content"), 0644), "failed to write workflow markdown file")

		lockContent := "name: \"" + displayName + "\"\non: push\n"
		require.NoError(t, os.WriteFile(lockFile, []byte(lockContent), 0644), "failed to write workflow lock file")
	}

	tests := []struct {
		name         string
		input        string
		expectedFile string
		expectError  bool
	}{
		{
			name:         "workflow ID",
			input:        "smoke-copilot",
			expectedFile: "smoke-copilot.lock.yml",
		},
		{
			name:         "workflow ID with .md extension",
			input:        "smoke-copilot.md",
			expectedFile: "smoke-copilot.lock.yml",
		},
		{
			name:         "workflow ID with .lock.yml extension",
			input:        "smoke-copilot.lock.yml",
			expectedFile: "smoke-copilot.lock.yml",
		},
		{
			name:         "display name exact match",
			input:        "Smoke Copilot",
			expectedFile: "smoke-copilot.lock.yml",
		},
		{
			name:         "display name case-insensitive match",
			input:        "smoke copilot",
			expectedFile: "smoke-copilot.lock.yml",
		},
		{
			name:        "non-existent workflow",
			input:       "non-existent",
			expectError: true,
		},
		{
			name:         "empty input",
			input:        "",
			expectedFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetWorkflowLockFileName(tt.input)

			if tt.expectError {
				assert.Error(t, err, "GetWorkflowLockFileName(%q) should return an error", tt.input)
				return
			}

			require.NoError(t, err, "GetWorkflowLockFileName(%q) should not return an error", tt.input)
			assert.Equal(t, tt.expectedFile, result, "GetWorkflowLockFileName(%q) should return the correct lock file name", tt.input)
		})
	}
}
