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

// TestSafeOutputsAppConfiguration tests that app configuration is correctly parsed
func TestSafeOutputsAppConfiguration(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
safe-outputs:
  create-issue:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    repositories:
      - "repo1"
      - "repo2"
---

# Test Workflow

Test workflow with app configuration.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.GitHubApp, "App configuration should be parsed")

	// Verify app configuration
	assert.Equal(t, "${{ vars.APP_ID }}", workflowData.SafeOutputs.GitHubApp.AppID)
	assert.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", workflowData.SafeOutputs.GitHubApp.PrivateKey)
	assert.Equal(t, []string{"repo1", "repo2"}, workflowData.SafeOutputs.GitHubApp.Repositories)
}

// TestSafeOutputsAppConfigurationMinimal tests minimal app configuration without repositories
func TestSafeOutputsAppConfigurationMinimal(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
safe-outputs:
  create-issue:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow

Test workflow with minimal app configuration.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.GitHubApp, "App configuration should be parsed")

	// Verify app configuration
	assert.Equal(t, "${{ vars.APP_ID }}", workflowData.SafeOutputs.GitHubApp.AppID)
	assert.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", workflowData.SafeOutputs.GitHubApp.PrivateKey)
	assert.Empty(t, workflowData.SafeOutputs.GitHubApp.Repositories)
}

func TestSafeOutputsAppIgnoreIfMissing(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
safe-outputs:
  add-comment:
  github-app:
    app-id: ${{ secrets.GH_AW_APP_ID }}
    private-key: ${{ secrets.GH_AW_APP_PRIVATE_KEY }}
    ignore-if-missing: true
---

# Test Workflow
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.GitHubApp, "GitHub app configuration should be parsed")
	assert.True(t, workflowData.SafeOutputs.GitHubApp.IgnoreIfMissing)

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "main", testFile)
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	stepsStr := strings.Join(job.Steps, "")
	assert.Contains(t, stepsStr, "if: ${{ secrets.GH_AW_APP_ID != '' && secrets.GH_AW_APP_PRIVATE_KEY != '' }}")
	assert.NotContains(t, stepsStr, "GH_AW_APP_CLIENT_ID:")
	assert.NotContains(t, stepsStr, "GH_AW_APP_PRIVATE_KEY:")
	assert.Contains(t, stepsStr, "github-token: ${{ steps.safe-outputs-app-token.outputs.token || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}")
}

func TestSafeOutputsAppIgnoreIfMissingInvalidType(t *testing.T) {
	app := parseAppConfig(map[string]any{
		"client-id":         "${{ vars.APP_ID }}",
		"private-key":       "${{ secrets.APP_PRIVATE_KEY }}",
		"ignore-if-missing": "not-a-bool",
	})

	require.NotNil(t, app)
	assert.False(t, app.IgnoreIfMissing)
	assert.False(t, app.shouldIgnoreMissingKey())
}

func TestBuildIgnoreIfMissingCondition(t *testing.T) {
	tests := []struct {
		name       string
		appID      string
		privateKey string
		expected   string
	}{
		{
			name:       "wrapped expressions",
			appID:      "${{ secrets.GH_AW_APP_ID }}",
			privateKey: "${{ secrets.GH_AW_APP_PRIVATE_KEY }}",
			expected:   "${{ secrets.GH_AW_APP_ID != '' && secrets.GH_AW_APP_PRIVATE_KEY != '' }}",
		},
		{
			name:       "literal values",
			appID:      "  id value  ",
			privateKey: "key'value",
			expected:   "${{ 'id value' != '' && 'key''value' != '' }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &GitHubAppConfig{
				AppID:      tt.appID,
				PrivateKey: tt.privateKey,
			}
			assert.Equal(t, tt.expected, buildIgnoreIfMissingCondition(app))
		})
	}
}

// TestSafeOutputsAppWithoutSafeOutputs tests that app without safe outputs doesn't break
func TestSafeOutputsAppWithoutSafeOutputs(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
permissions:
  contents: read
---

# Test Workflow

Test workflow without safe outputs.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	// create-issue is auto-injected even when no safe-outputs section is configured
	if workflowData.SafeOutputs == nil {
		t.Fatal("Expected SafeOutputs to be non-nil after auto-injection of create-issue")
	}
	if workflowData.SafeOutputs.CreateIssues == nil || !workflowData.SafeOutputs.AutoInjectedCreateIssue {
		t.Error("Expected create-issue to be auto-injected when no safe-outputs configured")
	}
}

// TestSafeOutputsAppTokenDiscussionsPermission tests that discussions permission is included
// in the GitHub App token minting step when create-discussion is configured.
//
// actions/create-github-app-token v3+ declares "permission-discussions" as a valid input.
// When any permission-* input is specified, the action scopes the token to ONLY those permissions,
// so omitting permission-discussions would exclude discussions access from the minted token.
func TestSafeOutputsAppTokenDiscussionsPermission(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
safe-outputs:
  create-discussion:
    category: "general"
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow

Test workflow with discussions permission.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.CreateDiscussions, "CreateDiscussions should not be nil")

	// Build the consolidated safe_outputs job
	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "main", testFile)
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	// Convert steps to string for easier assertion
	stepsStr := strings.Join(job.Steps, "")

	// permission-discussions must be present because when any permission-* input is set,
	// actions/create-github-app-token scopes the token to only those permissions.
	assert.Contains(t, stepsStr, "permission-discussions: write", "GitHub App token should include discussions write permission")
	// Other explicitly supported permission inputs should still be present
	assert.Contains(t, stepsStr, "permission-contents: read", "GitHub App token should include contents read permission")
	assert.Contains(t, stepsStr, "permission-issues: write", "GitHub App token should include issues write permission (create-discussion falls back to issue)")
}

// TestSafeOutputsAppTokenUpdateProjectIssuesReadPermission tests that issues read permission
// is included in the GitHub App token minting step when update-project is configured.
func TestSafeOutputsAppTokenUpdateProjectIssuesReadPermission(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
safe-outputs:
  update-project:
    project: "https://github.com/orgs/my-org/projects/1"
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow

Test workflow with update-project permissions.
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.UpdateProjects, "UpdateProjects should not be nil")

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "main", testFile)
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	stepsStr := strings.Join(job.Steps, "")

	assert.Contains(t, stepsStr, "permission-organization-projects: write", "GitHub App token should include organization projects write permission")
	assert.Contains(t, stepsStr, "permission-issues: read", "GitHub App token should include issues read permission for issue-backed project items")
	assert.Contains(t, stepsStr, "permission-contents: read", "GitHub App token should include contents read permission")
}

// TestSafeOutputsAppTokenCreateProjectWithItemURLIssuesReadPermission tests that issues read permission
// is included in the GitHub App token minting step when create-project is configured with item_url.
func TestSafeOutputsAppTokenCreateProjectWithItemURLIssuesReadPermission(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on: issues
safe-outputs:
  create-project:
    target-owner: "my-org"
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow

Test workflow with create-project item_url permissions.
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.CreateProjects, "CreateProjects should not be nil")

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "main", testFile)
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	stepsStr := strings.Join(job.Steps, "")

	assert.Contains(t, stepsStr, "permission-organization-projects: write", "GitHub App token should include organization projects write permission")
	assert.Contains(t, stepsStr, "permission-issues: read", "GitHub App token should include issues read permission for issue-backed project items")
	assert.Contains(t, stepsStr, "permission-contents: read", "GitHub App token should include contents read permission")
}

// TestSafeOutputsAppTokenAddCommentAddLabelsIssuesWrite is a regression test for the issue
// where safe_outputs App token permissions were capped at the workflow-level permissions block
// instead of being derived from the configured safe-output handlers.
//
// Repro: workflow declares `permissions: { issues: read }` (required by agent-no-write rule),
// but configures add-comment (issues: true) and add-labels — both needing issues: write.
// The compiled App token MUST emit `permission-issues: write`, not `read`.
func TestSafeOutputsAppTokenAddCommentAddLabelsIssuesWrite(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    owner: my-org
  add-comment:
    max: 1
    issues: true
    pull-requests: false
    discussions: false
  add-labels:
    max: 4
    allowed: [routed]
---
Test workflow
`

	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.md"
	require.NoError(t, os.WriteFile(testFile, []byte(markdown), 0644), "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.GitHubApp, "GitHubApp should not be nil")

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", testFile)
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	stepsStr := strings.Join(job.Steps, "")

	// The workflow declares `issues: read` but the handlers require `issues: write`.
	// The App token permissions MUST come from handler-computed scope, NOT the
	// workflow-level `permissions:` block.
	assert.Contains(t, stepsStr, "permission-issues: write",
		"App token must use handler-computed issues:write, not workflow-level issues:read")
	assert.Contains(t, stepsStr, "permission-pull-requests: write",
		"App token must include pull-requests:write from add-labels handler")
	assert.Contains(t, stepsStr, "permission-contents: read",
		"App token must include contents:read")

	// The job-level permissions YAML must also reflect the handler-computed scope.
	assert.Contains(t, job.Permissions, "issues: write",
		"Job-level permissions must be handler-computed (issues:write)")
}

// TestSafeOutputsAppTokenUpdateProjectDoesNotDowngradeIssuesWrite is a regression test for the
// add-comment + add-labels + update-project co-presence case reported after github/gh-aw#30437.
// update-project must not downgrade issues permission from write to read in the minted GitHub App token.
func TestSafeOutputsAppTokenUpdateProjectDoesNotDowngradeIssuesWrite(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	markdown := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    owner: my-org
  add-comment:
    max: 1
    issues: true
    pull-requests: false
    discussions: false
  add-labels:
    max: 4
    allowed: [routed]
  update-project:
    max: 1
    project: https://github.com/orgs/my-org/projects/1
---
Test workflow
`

	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.md"
	require.NoError(t, os.WriteFile(testFile, []byte(markdown), 0644), "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.GitHubApp, "GitHubApp should not be nil")

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", testFile)
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	stepsStr := strings.Join(job.Steps, "")

	assert.Contains(t, stepsStr, "permission-issues: write",
		"App token must preserve issues:write required by add-comment/add-labels when update-project is present")
	assert.Contains(t, stepsStr, "permission-organization-projects: write",
		"App token must include organization-projects:write for update-project")
	assert.Contains(t, job.Permissions, "issues: write",
		"Job-level permissions must preserve handler-computed issues:write")
}

// TestSafeOutputsAppTokenPermissionsOverride tests that safe-outputs.github-app.permissions:
// overrides take effect in the minted token. Users can supply GitHub App-only scopes
// (e.g. members: read) not expressible via standard safe-output handler declarations.
func TestSafeOutputsAppTokenPermissionsOverride(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
				Permissions: map[string]string{
					"members": "read",
				},
			},
			CreateIssues: &CreateIssuesConfig{TitlePrefix: "[Test] "},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", "test.md")
	require.NoError(t, err, "Failed to build safe_outputs job")
	require.NotNil(t, job, "Job should not be nil")

	stepsStr := strings.Join(job.Steps, "")

	// The override must add permission-members: read to the minted token.
	assert.Contains(t, stepsStr, "permission-members: read",
		"App token must include members:read from github-app.permissions override")
}
