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
	assert.Nil(t, workflowData.SafeOutputs, "SafeOutputs should be nil")
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
