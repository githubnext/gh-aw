//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckoutImportFromSharedWorkflow tests that a checkout block defined in a shared
// workflow is inherited by the importing workflow.
func TestCheckoutImportFromSharedWorkflow(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Shared workflow that declares a checkout block for a side repository
	sharedWorkflow := `---
checkout:
  - repository: org/target-repo
    ref: master
    path: target-repo
    current: true
---

# Shared side-repo checkout configuration

This shared workflow centralizes the checkout block for SideRepoOps workflows.
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-checkout.md"), []byte(sharedWorkflow), 0644))

	// Main workflow that imports the shared workflow without its own checkout block
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-checkout.md
---

# Main Workflow

This workflow inherits the checkout configuration from the shared workflow.
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.Len(t, data.CheckoutConfigs, 1, "Should have one checkout config from the shared workflow")
	cfg := data.CheckoutConfigs[0]
	assert.Equal(t, "org/target-repo", cfg.Repository, "Repository should come from the shared workflow")
	assert.Equal(t, "master", cfg.Ref, "Ref should come from the shared workflow")
	assert.Equal(t, "target-repo", cfg.Path, "Path should come from the shared workflow")
	assert.True(t, cfg.Current, "Current should be true from the shared workflow")
}

// TestCheckoutImportMainWorkflowTakesPrecedence tests that the main workflow's checkout
// takes precedence over an imported checkout for the same (repository, path) key.
func TestCheckoutImportMainWorkflowTakesPrecedence(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	sharedWorkflow := `---
checkout:
  - repository: org/target-repo
    ref: main
    path: target-repo
---

# Shared Checkout
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-checkout.md"), []byte(sharedWorkflow), 0644))

	// Main workflow overrides the checkout for the same path
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-checkout.md
checkout:
  - repository: org/target-repo
    ref: feature-branch
    path: target-repo
---

# Main Workflow

This workflow overrides the checkout from the shared workflow.
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	// data.CheckoutConfigs holds the raw (pre-dedup) slice: main entry first, then imported.
	// Deduplication and merge-precedence are enforced by NewCheckoutManager.
	require.NotEmpty(t, data.CheckoutConfigs, "Should have checkout configs")

	cm := NewCheckoutManager(data.CheckoutConfigs)
	// After deduplication there should be exactly one resolved entry for (org/target-repo, target-repo).
	require.Len(t, cm.ordered, 1, "Duplicate (repository, path) entries should be merged into one")
	entry := cm.ordered[0]
	assert.Equal(t, "org/target-repo", entry.key.repository, "Repository should be org/target-repo")
	assert.Equal(t, "target-repo", entry.key.path, "Path should be target-repo")
	assert.Equal(t, "feature-branch", entry.ref, "Main workflow's ref should take precedence over imported ref")
}

// TestCheckoutImportDisabledByMainWorkflow tests that checkout: false in the main workflow
// suppresses imported checkout configs.
func TestCheckoutImportDisabledByMainWorkflow(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	sharedWorkflow := `---
checkout:
  - repository: org/target-repo
    ref: main
    path: target-repo
---

# Shared Checkout
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-checkout.md"), []byte(sharedWorkflow), 0644))

	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-checkout.md
checkout: false
---

# Main Workflow

This workflow disables checkout entirely.
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	assert.True(t, data.CheckoutDisabled, "Checkout should be disabled")
	assert.Empty(t, data.CheckoutConfigs, "No checkout configs should be merged when checkout is disabled")
}

// TestCheckoutImportMultipleImports tests that checkout configs from multiple shared
// workflows are all merged into the importing workflow.
func TestCheckoutImportMultipleImports(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	shared1 := `---
checkout:
  - repository: org/repo-a
    path: repo-a
---

# Shared Checkout A
`
	shared2 := `---
checkout:
  - repository: org/repo-b
    path: repo-b
---

# Shared Checkout B
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-a.md"), []byte(shared1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-b.md"), []byte(shared2), 0644))

	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-a.md
  - ./shared-b.md
---

# Main Workflow
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.Len(t, data.CheckoutConfigs, 2, "Should have checkout configs from both shared workflows")

	repos := make(map[string]bool)
	for _, cfg := range data.CheckoutConfigs {
		repos[cfg.Repository] = true
	}
	assert.True(t, repos["org/repo-a"], "Should include checkout for org/repo-a")
	assert.True(t, repos["org/repo-b"], "Should include checkout for org/repo-b")
}

// TestCheckoutImportAuthPrecedence tests that the main workflow's auth method is preserved
// when an imported shared workflow defines conflicting auth for the same (repository, path).
// A main workflow github-token must not be overridden by an imported github-app, and vice versa.
func TestCheckoutImportAuthPrecedence(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Shared workflow has a github-app for the same repository/path
	sharedWorkflow := `---
checkout:
  - repository: org/target-repo
    ref: main
    path: target-repo
    github-app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Shared Checkout with App Auth
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-checkout.md"), []byte(sharedWorkflow), 0644))

	// Main workflow uses a plain token for the same repository/path
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-checkout.md
checkout:
  - repository: org/target-repo
    ref: feature-branch
    path: target-repo
    github-token: ${{ secrets.MY_PAT }}
---

# Main Workflow
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainWorkflow), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	cm := NewCheckoutManager(data.CheckoutConfigs)
	require.Len(t, cm.ordered, 1, "Should have one merged entry for the duplicate (repository, path)")
	entry := cm.ordered[0]
	assert.Equal(t, "${{ secrets.MY_PAT }}", entry.token, "Main workflow's github-token must be preserved")
	assert.Nil(t, entry.githubApp, "Imported github-app must not override main workflow's github-token")
	assert.Equal(t, "feature-branch", entry.ref, "Main workflow's ref should take precedence")
	assert.False(t, cm.HasAppAuth(), "Checkout manager should report no app auth (main token takes precedence)")
}
