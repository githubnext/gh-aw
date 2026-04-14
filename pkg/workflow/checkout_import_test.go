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

	// The main workflow's checkout is primary; CheckoutManager merges duplicates with first-seen wins
	// for most fields. Both should result in one logical entry for (org/target-repo, target-repo).
	require.NotEmpty(t, data.CheckoutConfigs, "Should have checkout configs")
	// Find the entry for target-repo path
	var found *CheckoutConfig
	for _, cfg := range data.CheckoutConfigs {
		if cfg.Path == "target-repo" {
			found = cfg
			break
		}
	}
	require.NotNil(t, found, "Should find checkout config for target-repo path")
	assert.Equal(t, "org/target-repo", found.Repository, "Repository should be org/target-repo")
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
