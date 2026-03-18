//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTopLevelGitHubAppImport tests that a top-level github-app can be imported
// from a shared agent workflow and propagated as a fallback for all nested operations.
func TestTopLevelGitHubAppImport(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	// Create a temporary directory simulating .github/workflows layout
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Shared workflow that declares a top-level github-app
	sharedWorkflow := `---
github-app:
  app-id: ${{ vars.SHARED_APP_ID }}
  private-key: ${{ secrets.SHARED_APP_SECRET }}
safe-outputs:
  create-issue:
---

# Shared GitHub App Configuration

This shared workflow provides a top-level github-app for all operations.
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-app.md"), []byte(sharedWorkflow), 0644))

	// Main workflow that imports the shared workflow but does NOT set its own github-app
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-app.md
safe-outputs:
  create-issue:
---

# Main Workflow

This workflow imports the top-level github-app from the shared workflow.
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	// Change to workflows directory so relative imports resolve correctly
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	// The top-level github-app from the shared workflow should be resolved
	require.NotNil(t, data.TopLevelGitHubApp, "TopLevelGitHubApp should be populated from import")
	assert.Equal(t, "${{ vars.SHARED_APP_ID }}", data.TopLevelGitHubApp.AppID,
		"TopLevelGitHubApp.AppID should come from the shared workflow")
	assert.Equal(t, "${{ secrets.SHARED_APP_SECRET }}", data.TopLevelGitHubApp.PrivateKey,
		"TopLevelGitHubApp.PrivateKey should come from the shared workflow")

	// The fallback should also propagate to safe-outputs (since safe-outputs has no explicit github-app)
	require.NotNil(t, data.SafeOutputs, "SafeOutputs should be populated")
	require.NotNil(t, data.SafeOutputs.GitHubApp,
		"SafeOutputs.GitHubApp should be populated from the imported top-level github-app")
	assert.Equal(t, "${{ vars.SHARED_APP_ID }}", data.SafeOutputs.GitHubApp.AppID,
		"SafeOutputs should use the imported top-level github-app")
}

// TestTopLevelGitHubAppImportOverride tests that the current workflow's own top-level
// github-app takes precedence over one imported from a shared workflow.
func TestTopLevelGitHubAppImportOverride(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Shared workflow with a top-level github-app
	sharedWorkflow := `---
github-app:
  app-id: ${{ vars.SHARED_APP_ID }}
  private-key: ${{ secrets.SHARED_APP_SECRET }}
safe-outputs:
  create-issue:
---

# Shared GitHub App Configuration
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-app.md"), []byte(sharedWorkflow), 0644))

	// Main workflow that has its OWN top-level github-app (should win)
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-app.md
github-app:
  app-id: ${{ vars.LOCAL_APP_ID }}
  private-key: ${{ secrets.LOCAL_APP_SECRET }}
safe-outputs:
  create-issue:
---

# Main Workflow

This workflow's own top-level github-app takes precedence over the imported one.
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	// The current workflow's top-level github-app should win
	require.NotNil(t, data.TopLevelGitHubApp, "TopLevelGitHubApp should be populated")
	assert.Equal(t, "${{ vars.LOCAL_APP_ID }}", data.TopLevelGitHubApp.AppID,
		"Current workflow's github-app should take precedence over the imported one")
	assert.Equal(t, "${{ secrets.LOCAL_APP_SECRET }}", data.TopLevelGitHubApp.PrivateKey,
		"Current workflow's github-app should take precedence over the imported one")
}

// TestTopLevelGitHubAppImportActivation tests that a top-level github-app imported from a shared
// workflow is propagated to the activation job (reactions/status comments).
func TestTopLevelGitHubAppImportActivation(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	sharedWorkflow := `---
github-app:
  app-id: ${{ vars.SHARED_APP_ID }}
  private-key: ${{ secrets.SHARED_APP_SECRET }}
safe-outputs:
  create-issue:
---

# Shared GitHub App Configuration
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-app.md"), []byte(sharedWorkflow), 0644))

	// Workflow with a reaction trigger – no explicit on.github-app
	mainWorkflow := `---
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  contents: read
imports:
  - ./shared-app.md
safe-outputs:
  create-issue:
---

# Main Workflow With Reaction
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	// The imported top-level github-app should propagate to activation
	require.NotNil(t, data.ActivationGitHubApp,
		"ActivationGitHubApp should be populated from the imported top-level github-app")
	assert.Equal(t, "${{ vars.SHARED_APP_ID }}", data.ActivationGitHubApp.AppID,
		"Activation should use the imported top-level github-app")
}
