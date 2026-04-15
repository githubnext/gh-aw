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
	compiler := NewCompiler(WithVersion("1.0.0"))

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
	compiler := NewCompiler(WithVersion("1.0.0"))

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

// TestTopLevelGitHubAppToolsGitHubTokenSkip tests that the fallback is NOT applied
// to tools.github when a custom github-token is already configured for the MCP server.
func TestTopLevelGitHubAppToolsGitHubTokenSkip(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	// Use ParseWorkflowFile directly with inline frontmatter
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Workflow with top-level github-app but tools.github uses an explicit github-token
	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
tools:
  github:
    mode: remote
    toolsets: [default]
    github-token: ${{ secrets.CUSTOM_PAT }}
engine: copilot
---

# Workflow With Explicit GitHub Token for MCP

When tools.github.github-token is set, the top-level github-app fallback should NOT override it.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	// The top-level github-app should be resolved at the top level
	require.NotNil(t, data.TopLevelGitHubApp, "TopLevelGitHubApp should be populated")

	// But it must NOT be injected into tools.github because github-token takes precedence
	require.NotNil(t, data.ParsedTools, "ParsedTools should be populated")
	require.NotNil(t, data.ParsedTools.GitHub, "ParsedTools.GitHub should be populated")
	assert.Equal(t, "${{ secrets.CUSTOM_PAT }}", data.ParsedTools.GitHub.GitHubToken,
		"tools.github.github-token should be preserved")
	assert.Nil(t, data.ParsedTools.GitHub.GitHubApp,
		"tools.github.github-app should NOT be set when github-token is configured")
}

// TestTopLevelGitHubAppToolsGitHubFalseSkip tests that the fallback is NOT applied
// to tools.github when github is explicitly disabled (github: false).
func TestTopLevelGitHubAppToolsGitHubFalseSkip(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Workflow with top-level github-app but tools.github explicitly disabled
	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
tools:
  github: false
engine: copilot
---

# Workflow With GitHub Tool Explicitly Disabled

When tools.github is set to false, the top-level github-app fallback should NOT re-enable it.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	// The top-level github-app should be resolved at the top level
	require.NotNil(t, data.TopLevelGitHubApp, "TopLevelGitHubApp should be populated")

	// tools.github should remain disabled — applyDefaultTools removes the key when false.
	// After compilation, ParsedTools.GitHub should be nil (no GitHub MCP tool enabled).
	assert.Nil(t, data.ParsedTools.GitHub,
		"ParsedTools.GitHub should be nil when tools.github: false — fallback must not re-enable it")
}

// workflow is propagated to the activation job (reactions/status comments).
func TestTopLevelGitHubAppImportActivation(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

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

// TestTopLevelGitHubAppActivationOverride tests that an explicit on.github-app configuration
// takes precedence over the top-level github-app fallback.
func TestTopLevelGitHubAppActivationOverride(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on:
  issues:
    types: [opened]
  reaction: eyes
  github-app:
    app-id: ${{ vars.ACTIVATION_APP_ID }}
    private-key: ${{ secrets.ACTIVATION_APP_KEY }}
permissions:
  contents: read
github-app:
  app-id: ${{ vars.TOP_LEVEL_APP_ID }}
  private-key: ${{ secrets.TOP_LEVEL_APP_KEY }}
safe-outputs:
  create-issue:
engine: copilot
---

# Workflow With Explicit on.github-app Override

When on.github-app is explicitly set, it takes precedence over the top-level github-app.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.NotNil(t, data.TopLevelGitHubApp, "TopLevelGitHubApp should be populated")
	assert.Equal(t, "${{ vars.TOP_LEVEL_APP_ID }}", data.TopLevelGitHubApp.AppID,
		"TopLevelGitHubApp should hold the top-level app")

	// on.github-app should win over the top-level fallback
	require.NotNil(t, data.ActivationGitHubApp, "ActivationGitHubApp should be populated")
	assert.Equal(t, "${{ vars.ACTIVATION_APP_ID }}", data.ActivationGitHubApp.AppID,
		"ActivationGitHubApp should use the section-specific on.github-app, not the top-level fallback")
}

// TestTopLevelGitHubAppActivationTokenSkip tests that the top-level github-app fallback
// is NOT applied to activation when on.github-token is explicitly configured, because
// at runtime app tokens take precedence over tokens and injecting the fallback would flip
// the user's intended auth precedence.
func TestTopLevelGitHubAppActivationTokenSkip(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on:
  issues:
    types: [opened]
  reaction: eyes
  github-token: ${{ secrets.CUSTOM_ACTIVATION_TOKEN }}
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  create-issue:
engine: copilot
---

# Workflow With on.github-token — Fallback Must Not Apply

When on.github-token is set, the top-level github-app must NOT be applied to activation.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.NotNil(t, data.TopLevelGitHubApp, "TopLevelGitHubApp should be populated")
	assert.Equal(t, "${{ secrets.CUSTOM_ACTIVATION_TOKEN }}", data.ActivationGitHubToken,
		"ActivationGitHubToken should be preserved")
	assert.Nil(t, data.ActivationGitHubApp,
		"ActivationGitHubApp must be nil when on.github-token is set — fallback must not override it")
}

// TestTopLevelGitHubAppSafeOutputsFallback tests that the top-level github-app is applied
// to safe-outputs when no section-specific github-app is configured.
func TestTopLevelGitHubAppSafeOutputsFallback(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  create-issue:
engine: copilot
---

# Top-level github-app fallback for safe-outputs.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.NotNil(t, data.SafeOutputs, "SafeOutputs should be populated")
	require.NotNil(t, data.SafeOutputs.GitHubApp,
		"SafeOutputs.GitHubApp should be populated from top-level fallback")
	assert.Equal(t, "${{ vars.APP_ID }}", data.SafeOutputs.GitHubApp.AppID,
		"SafeOutputs should use the top-level github-app fallback")
}

// TestTopLevelGitHubAppSafeOutputsOverride tests that a section-specific safe-outputs.github-app
// takes precedence over the top-level github-app fallback.
func TestTopLevelGitHubAppSafeOutputsOverride(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.TOP_LEVEL_APP_ID }}
  private-key: ${{ secrets.TOP_LEVEL_APP_KEY }}
safe-outputs:
  github-app:
    app-id: ${{ vars.SAFE_OUTPUTS_APP_ID }}
    private-key: ${{ secrets.SAFE_OUTPUTS_APP_KEY }}
  create-issue:
engine: copilot
---

# Section-specific safe-outputs.github-app overrides top-level fallback.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.NotNil(t, data.SafeOutputs, "SafeOutputs should be populated")
	require.NotNil(t, data.SafeOutputs.GitHubApp, "SafeOutputs.GitHubApp should be populated")
	assert.Equal(t, "${{ vars.SAFE_OUTPUTS_APP_ID }}", data.SafeOutputs.GitHubApp.AppID,
		"SafeOutputs.GitHubApp should use the section-specific app, not the top-level fallback")
}

// TestTopLevelGitHubAppSafeOutputsTokenSkip tests that the top-level github-app fallback
// is NOT applied to safe-outputs when safe-outputs.github-token is explicitly set.
func TestTopLevelGitHubAppSafeOutputsTokenSkip(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  github-token: ${{ secrets.CUSTOM_SAFE_OUTPUTS_TOKEN }}
  create-issue:
engine: copilot
---

# safe-outputs.github-token is set — top-level github-app must not override it.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.NotNil(t, data.SafeOutputs, "SafeOutputs should be populated")
	assert.Equal(t, "${{ secrets.CUSTOM_SAFE_OUTPUTS_TOKEN }}", data.SafeOutputs.GitHubToken,
		"SafeOutputs.GitHubToken should be preserved")
	assert.Nil(t, data.SafeOutputs.GitHubApp,
		"SafeOutputs.GitHubApp must be nil when safe-outputs.github-token is configured")
}

// TestTopLevelGitHubAppCheckoutFallback tests that the top-level github-app is applied
// to a checkout entry that has no explicit auth (no github-app, no github-token).
func TestTopLevelGitHubAppCheckoutFallback(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
checkout:
  repository: myorg/private-repo
  path: private
safe-outputs:
  create-issue:
engine: copilot
---

# Top-level github-app fallback for checkout.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.Len(t, data.CheckoutConfigs, 1, "Should have one checkout config")
	require.NotNil(t, data.CheckoutConfigs[0].GitHubApp,
		"CheckoutConfig.GitHubApp should be populated from top-level fallback")
	assert.Equal(t, "${{ vars.APP_ID }}", data.CheckoutConfigs[0].GitHubApp.AppID,
		"Checkout should use the top-level github-app fallback")
}

// TestTopLevelGitHubAppCheckoutOverride tests that a section-specific checkout.github-app
// takes precedence over the top-level github-app fallback.
func TestTopLevelGitHubAppCheckoutOverride(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.TOP_LEVEL_APP_ID }}
  private-key: ${{ secrets.TOP_LEVEL_APP_KEY }}
checkout:
  - repository: myorg/private-repo
    path: private
    github-app:
      app-id: ${{ vars.CHECKOUT_APP_ID }}
      private-key: ${{ secrets.CHECKOUT_APP_KEY }}
safe-outputs:
  create-issue:
engine: copilot
---

# checkout.github-app overrides the top-level github-app fallback.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.Len(t, data.CheckoutConfigs, 1, "Should have one checkout config")
	require.NotNil(t, data.CheckoutConfigs[0].GitHubApp, "CheckoutConfig.GitHubApp should be populated")
	assert.Equal(t, "${{ vars.CHECKOUT_APP_ID }}", data.CheckoutConfigs[0].GitHubApp.AppID,
		"Checkout should use its own section-specific github-app, not the top-level fallback")
}

// TestTopLevelGitHubAppCheckoutTokenSkip tests that the top-level github-app fallback is NOT
// applied to a checkout entry that has an explicit github-token set.
func TestTopLevelGitHubAppCheckoutTokenSkip(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
checkout:
  - repository: myorg/private-repo
    path: private
    github-token: ${{ secrets.CHECKOUT_PAT }}
safe-outputs:
  create-issue:
engine: copilot
---

# checkout.github-token is set — top-level github-app must not override it.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.Len(t, data.CheckoutConfigs, 1, "Should have one checkout config")
	assert.Equal(t, "${{ secrets.CHECKOUT_PAT }}", data.CheckoutConfigs[0].GitHubToken,
		"CheckoutConfig.GitHubToken should be preserved")
	assert.Nil(t, data.CheckoutConfigs[0].GitHubApp,
		"CheckoutConfig.GitHubApp must be nil when checkout.github-token is configured")
}

// TestTopLevelGitHubAppToolsFallback tests that the top-level github-app is applied
// to tools.github when no section-specific auth is configured.
func TestTopLevelGitHubAppToolsFallback(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
engine: copilot
---

# Top-level github-app fallback for tools.github.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.NotNil(t, data.ParsedTools, "ParsedTools should be populated")
	require.NotNil(t, data.ParsedTools.GitHub, "ParsedTools.GitHub should be populated")
	require.NotNil(t, data.ParsedTools.GitHub.GitHubApp,
		"ParsedTools.GitHub.GitHubApp should be populated from top-level fallback")
	assert.Equal(t, "${{ vars.APP_ID }}", data.ParsedTools.GitHub.GitHubApp.AppID,
		"tools.github should use the top-level github-app fallback")
}

// TestTopLevelGitHubAppToolsOverride tests that a section-specific tools.github.github-app
// takes precedence over the top-level github-app fallback.
func TestTopLevelGitHubAppToolsOverride(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
github-app:
  app-id: ${{ vars.TOP_LEVEL_APP_ID }}
  private-key: ${{ secrets.TOP_LEVEL_APP_KEY }}
tools:
  github:
    mode: remote
    toolsets: [default]
    github-app:
      app-id: ${{ vars.MCP_APP_ID }}
      private-key: ${{ secrets.MCP_APP_KEY }}
safe-outputs:
  create-issue:
engine: copilot
---

# tools.github.github-app overrides the top-level github-app fallback.
`
	mdPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(workflowContent), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workflowsDir))
	defer func() { _ = os.Chdir(origDir) }()

	data, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err)

	require.NotNil(t, data.ParsedTools, "ParsedTools should be populated")
	require.NotNil(t, data.ParsedTools.GitHub, "ParsedTools.GitHub should be populated")
	require.NotNil(t, data.ParsedTools.GitHub.GitHubApp, "ParsedTools.GitHub.GitHubApp should be populated")
	assert.Equal(t, "${{ vars.MCP_APP_ID }}", data.ParsedTools.GitHub.GitHubApp.AppID,
		"tools.github should use its own section-specific github-app, not the top-level fallback")
}
