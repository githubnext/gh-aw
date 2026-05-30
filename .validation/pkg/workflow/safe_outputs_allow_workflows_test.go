//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllowWorkflowsPermission tests that allow-workflows: true on create-pull-request
// adds workflows: write to the computed safe-output permissions.
func TestAllowWorkflowsPermission(t *testing.T) {
	tests := []struct {
		name           string
		safeOutputs    *SafeOutputsConfig
		expectWorkflow bool
	}{
		{
			name: "create-pull-request with allow-workflows true adds workflows write",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowWorkflows: true,
				},
			},
			expectWorkflow: true,
		},
		{
			name: "create-pull-request without allow-workflows does not add workflows write",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectWorkflow: false,
		},
		{
			name: "push-to-pull-request-branch with allow-workflows true adds workflows write",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					AllowWorkflows: true,
				},
			},
			expectWorkflow: true,
		},
		{
			name: "push-to-pull-request-branch without allow-workflows does not add workflows write",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			expectWorkflow: false,
		},
		{
			name: "both handlers with allow-workflows true",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowWorkflows: true,
				},
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					AllowWorkflows: true,
				},
			},
			expectWorkflow: true,
		},
		{
			name: "staged create-pull-request with allow-workflows does not add workflows write",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true},
					AllowWorkflows:       true,
				},
			},
			expectWorkflow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permissions := ComputePermissionsForSafeOutputs(tt.safeOutputs)
			require.NotNil(t, permissions, "Permissions should not be nil")

			level, ok := permissions.GetExplicit(PermissionWorkflows)
			if tt.expectWorkflow {
				assert.True(t, ok, "workflows permission should be present")
				assert.Equal(t, PermissionWrite, level, "workflows should be write")
			} else {
				assert.False(t, ok, "workflows permission should not be present")
			}
		})
	}
}

// TestAllowWorkflowsValidationRequiresGitHubApp tests that allow-workflows: true
// without a GitHub App configuration produces a validation error.
func TestAllowWorkflowsValidationRequiresGitHubApp(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expectError bool
	}{
		{
			name:        "nil safe outputs - no error",
			safeOutputs: nil,
			expectError: false,
		},
		{
			name: "allow-workflows without github-app - error",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowWorkflows: true,
				},
			},
			expectError: true,
		},
		{
			name: "allow-workflows with github-app - no error",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowWorkflows: true,
				},
				GitHubApp: &GitHubAppConfig{
					AppID:      "${{ vars.APP_ID }}",
					PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
				},
			},
			expectError: false,
		},
		{
			name: "push-to-pr-branch allow-workflows without github-app - error",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					AllowWorkflows: true,
				},
			},
			expectError: true,
		},
		{
			name: "push-to-pr-branch allow-workflows with github-app - no error",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					AllowWorkflows: true,
				},
				GitHubApp: &GitHubAppConfig{
					AppID:      "${{ vars.APP_ID }}",
					PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
				},
			},
			expectError: false,
		},
		{
			name: "no allow-workflows - no error even without github-app",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectError: false,
		},
		{
			name: "allow-workflows with empty github-app config - error",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowWorkflows: true,
				},
				GitHubApp: &GitHubAppConfig{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsAllowWorkflows(tt.safeOutputs)
			if tt.expectError {
				require.Error(t, err, "Expected validation error")
				assert.Contains(t, err.Error(), "allow-workflows", "Error should mention allow-workflows")
				assert.Contains(t, err.Error(), "requires a GitHub App", "Error should mention GitHub App requirement")
				assert.Contains(t, err.Error(), "github-app:", "Error should include configuration example")
			} else {
				assert.NoError(t, err, "Expected no validation error")
			}
		})
	}
}

// TestAllowWorkflowsParsing tests that allow-workflows is correctly parsed from frontmatter.
func TestAllowWorkflowsParsing(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
permissions:
  contents: read
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-pull-request:
    allow-workflows: true
    allowed-files:
      - ".github/workflows/*.lock.yml"
---

# Test Workflow

Test workflow with allow-workflows on create-pull-request.
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.CreatePullRequests, "CreatePullRequests should not be nil")

	assert.True(t, workflowData.SafeOutputs.CreatePullRequests.AllowWorkflows, "AllowWorkflows should be true")
	assert.Equal(t, []string{".github/workflows/*.lock.yml"}, workflowData.SafeOutputs.CreatePullRequests.AllowedFiles, "AllowedFiles should be parsed")
}

// TestAllowWorkflowsParsingPushToPullRequestBranch tests parsing for push-to-pull-request-branch.
func TestAllowWorkflowsParsingPushToPullRequestBranch(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: pull_request
permissions:
  contents: read
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  push-to-pull-request-branch:
    allow-workflows: true
    allowed-files:
      - ".github/workflows/*.lock.yml"
---

# Test Workflow

Test workflow with allow-workflows on push-to-pull-request-branch.
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.PushToPullRequestBranch, "PushToPullRequestBranch should not be nil")

	assert.True(t, workflowData.SafeOutputs.PushToPullRequestBranch.AllowWorkflows, "AllowWorkflows should be true")
}

// TestAllowWorkflowsAppTokenPermission tests that when allow-workflows is true
// and a GitHub App is configured, the compiled output includes permission-workflows: write.
func TestAllowWorkflowsAppTokenPermission(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
permissions:
  contents: read
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-pull-request:
    allow-workflows: true
    allowed-files:
      - ".github/workflows/*.lock.yml"
---

# Test Workflow

Test workflow checking permission-workflows: write in GitHub App token.
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "main", testFile)
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	stepsStr := strings.Join(job.Steps, "")
	assert.Contains(t, stepsStr, "permission-workflows: write", "GitHub App token should include workflows write permission")
}

// TestAllowWorkflowsCompileErrorWithoutGitHubApp tests that compiling a workflow
// with allow-workflows: true but no GitHub App produces a compile error.
func TestAllowWorkflowsCompileErrorWithoutGitHubApp(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
permissions:
  contents: read
safe-outputs:
  create-pull-request:
    allow-workflows: true
    allowed-files:
      - ".github/workflows/*.lock.yml"
---

# Test Workflow

Test workflow with allow-workflows but no GitHub App.
`

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))
	mdPath := filepath.Join(workflowsDir, "test.md")
	err := os.WriteFile(mdPath, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err = compiler.CompileWorkflow(mdPath)
	require.Error(t, err, "Compilation should fail without GitHub App")
	assert.Contains(t, err.Error(), "allow-workflows", "Error should mention allow-workflows")
}
