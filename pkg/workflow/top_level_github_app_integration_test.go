//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTopLevelGitHubAppFallback tests that a top-level github-app field in the frontmatter
// is used as a fallback for all nested github-app token minting operations when no
// section-specific github-app is configured.
func TestTopLevelGitHubAppFallback(t *testing.T) {
	tmpDir := testutil.TempDir(t, "top-level-github-app-test")

	t.Run("fallback applied to safe-outputs when no section-specific github-app", func(t *testing.T) {
		content := `---
name: Top Level GitHub App Safe Outputs Fallback
on:
  issues:
    types: [opened]
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

Test workflow verifying top-level github-app fallback for safe-outputs.
`
		mdPath := filepath.Join(tmpDir, "test-safe-outputs-fallback.md")
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Workflow with top-level github-app should compile successfully")

		lockPath := filepath.Join(tmpDir, "test-safe-outputs-fallback.lock.yml")
		compiledBytes, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		compiled := string(compiledBytes)

		// The safe-outputs job should use the top-level github-app for token minting
		assert.Contains(t, compiled, "id: safe-outputs-app-token",
			"Safe outputs job should generate a token minting step")
		assert.Contains(t, compiled, "app-id: ${{ vars.APP_ID }}",
			"Token minting step should use the top-level APP_ID")
		assert.Contains(t, compiled, "private-key: ${{ secrets.APP_PRIVATE_KEY }}",
			"Token minting step should use the top-level APP_PRIVATE_KEY")
	})

	t.Run("fallback applied to activation when no on.github-app", func(t *testing.T) {
		content := `---
name: Top Level GitHub App Activation Fallback
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

Test workflow verifying top-level github-app fallback for activation.
`
		mdPath := filepath.Join(tmpDir, "test-activation-fallback.md")
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Workflow with top-level github-app should compile successfully")

		lockPath := filepath.Join(tmpDir, "test-activation-fallback.lock.yml")
		compiledBytes, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		compiled := string(compiledBytes)

		// The activation job should use the top-level github-app for token minting
		assert.Contains(t, compiled, "id: activation-app-token",
			"Activation job should generate a token minting step using top-level github-app")
		assert.Contains(t, compiled, "app-id: ${{ vars.APP_ID }}",
			"Token minting step should use the top-level APP_ID")
		// The reaction step should use the minted app token
		assert.Contains(t, compiled, "github-token: ${{ steps.activation-app-token.outputs.token }}",
			"Activation step should use the minted app token")
	})

	t.Run("fallback applied to checkout when no checkout.github-app", func(t *testing.T) {
		content := `---
name: Top Level GitHub App Checkout Fallback
on:
  issues:
    types: [opened]
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
    title-prefix: "[automated] "
engine: copilot
---

Test workflow verifying top-level github-app fallback for checkout.
`
		mdPath := filepath.Join(tmpDir, "test-checkout-fallback.md")
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Workflow with top-level github-app should compile successfully")

		lockPath := filepath.Join(tmpDir, "test-checkout-fallback.lock.yml")
		compiledBytes, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		compiled := string(compiledBytes)

		// The checkout should use the top-level github-app for token minting
		assert.Contains(t, compiled, "id: checkout-app-token-0",
			"Checkout should generate a token minting step using top-level github-app")
		assert.Contains(t, compiled, "app-id: ${{ vars.APP_ID }}",
			"Token minting step should use the top-level APP_ID")
	})

	t.Run("fallback applied to tools.github when no tools.github.github-app", func(t *testing.T) {
		content := `---
name: Top Level GitHub App MCP Fallback
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

Test workflow verifying top-level github-app fallback for tools.github.
`
		mdPath := filepath.Join(tmpDir, "test-mcp-fallback.md")
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Workflow with top-level github-app should compile successfully")

		lockPath := filepath.Join(tmpDir, "test-mcp-fallback.lock.yml")
		compiledBytes, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		compiled := string(compiledBytes)

		// The agent job should use the top-level github-app for GitHub MCP token minting
		assert.Contains(t, compiled, "id: github-mcp-app-token",
			"Agent job should generate a GitHub MCP token minting step using top-level github-app")
		assert.Contains(t, compiled, "app-id: ${{ vars.APP_ID }}",
			"Token minting step should use the top-level APP_ID")
	})

	t.Run("fallback applied to APM dependencies when no dependencies.github-app", func(t *testing.T) {
		content := `---
name: Top Level GitHub App APM Dependencies Fallback
on:
  issues:
    types: [opened]
permissions:
  contents: read
github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
dependencies:
  packages:
    - myorg/private-skill
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

Test workflow verifying top-level github-app fallback for APM dependencies.
`
		mdPath := filepath.Join(tmpDir, "test-dependencies-fallback.md")
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Workflow with top-level github-app should compile successfully")

		lockPath := filepath.Join(tmpDir, "test-dependencies-fallback.lock.yml")
		compiledBytes, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		compiled := string(compiledBytes)

		// The activation job should have an APM app token minting step using the top-level github-app
		assert.Contains(t, compiled, "id: apm-app-token",
			"Activation job should generate an APM app token minting step using top-level github-app")
		assert.Contains(t, compiled, "app-id: ${{ vars.APP_ID }}",
			"APM token minting step should use the top-level APP_ID")
	})

	t.Run("section-specific github-app takes precedence over top-level", func(t *testing.T) {
		content := `---
name: Section Specific GitHub App Precedence
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
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  github-app:
    app-id: ${{ vars.SAFE_OUTPUTS_APP_ID }}
    private-key: ${{ secrets.SAFE_OUTPUTS_APP_KEY }}
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

Test workflow verifying section-specific github-app takes precedence over top-level fallback.
`
		mdPath := filepath.Join(tmpDir, "test-section-precedence.md")
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Workflow with section-specific github-app configs should compile successfully")

		lockPath := filepath.Join(tmpDir, "test-section-precedence.lock.yml")
		compiledBytes, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		compiled := string(compiledBytes)

		// The safe-outputs job should use SAFE_OUTPUTS_APP_ID (section-specific), not APP_ID (top-level)
		assert.Contains(t, compiled, "app-id: ${{ vars.SAFE_OUTPUTS_APP_ID }}",
			"Safe outputs job should use section-specific SAFE_OUTPUTS_APP_ID")
		assert.Contains(t, compiled, "app-id: ${{ vars.ACTIVATION_APP_ID }}",
			"Activation job should use section-specific ACTIVATION_APP_ID")
		// The top-level APP_ID should NOT appear anywhere because it's overridden by section-specific
		// configs in both on.github-app and safe-outputs.github-app. SAFE_OUTPUTS_APP_ID and
		// ACTIVATION_APP_ID are distinct values from APP_ID, so their presence does not conflict
		// with this assertion.
		assert.NotContains(t, compiled, "app-id: ${{ vars.APP_ID }}",
			"Top-level APP_ID should NOT be used when section-specific configs are present")
	})

	t.Run("no fallback applied when no top-level github-app", func(t *testing.T) {
		content := `---
name: No Top Level GitHub App
on:
  issues:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

Test workflow verifying no token minting when no github-app is configured.
`
		mdPath := filepath.Join(tmpDir, "test-no-fallback.md")
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Workflow without github-app should compile successfully")

		lockPath := filepath.Join(tmpDir, "test-no-fallback.lock.yml")
		compiledBytes, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		compiled := string(compiledBytes)

		// No token minting steps should be generated
		assert.NotContains(t, compiled, "id: safe-outputs-app-token",
			"No safe-outputs token minting step should be generated without github-app")
		assert.NotContains(t, compiled, "id: activation-app-token",
			"No activation token minting step should be generated without github-app")
	})
}

// TestTopLevelGitHubAppWorkflowFiles verifies that the sample workflow files in
// pkg/cli/workflows compile successfully and produce the expected token minting steps.
func TestTopLevelGitHubAppWorkflowFiles(t *testing.T) {
	tmpDir := testutil.TempDir(t, "top-level-github-app-workflow-files-test")

	tests := []struct {
		name           string
		workflowFile   string
		expectContains []string
	}{
		{
			name:         "safe-outputs fallback workflow file",
			workflowFile: "../cli/workflows/test-top-level-github-app-safe-outputs.md",
			expectContains: []string{
				"id: safe-outputs-app-token",
				"app-id: ${{ vars.APP_ID }}",
				"private-key: ${{ secrets.APP_PRIVATE_KEY }}",
			},
		},
		{
			name:         "activation fallback workflow file",
			workflowFile: "../cli/workflows/test-top-level-github-app-activation.md",
			expectContains: []string{
				"id: activation-app-token",
				"app-id: ${{ vars.APP_ID }}",
				"github-token: ${{ steps.activation-app-token.outputs.token }}",
			},
		},
		{
			name:         "checkout fallback workflow file",
			workflowFile: "../cli/workflows/test-top-level-github-app-checkout.md",
			expectContains: []string{
				"id: checkout-app-token-0",
				"app-id: ${{ vars.APP_ID }}",
			},
		},
		{
			name:         "dependencies fallback workflow file",
			workflowFile: "../cli/workflows/test-top-level-github-app-dependencies.md",
			expectContains: []string{
				"id: apm-app-token",
				"app-id: ${{ vars.APP_ID }}",
			},
		},
		{
			name:         "section-specific override workflow file",
			workflowFile: "../cli/workflows/test-top-level-github-app-override.md",
			expectContains: []string{
				"app-id: ${{ vars.SAFE_OUTPUTS_APP_ID }}",
				"app-id: ${{ vars.ACTIVATION_APP_ID }}",
			},
		},
		{
			name:         "tools.github MCP fallback workflow file",
			workflowFile: "../cli/workflows/test-top-level-github-app-mcp.md",
			expectContains: []string{
				"id: github-mcp-app-token",
				"app-id: ${{ vars.APP_ID }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := os.ReadFile(tt.workflowFile)
			require.NoError(t, err, "Failed to read workflow file %s", tt.workflowFile)

			baseName := filepath.Base(tt.workflowFile)
			mdDst := filepath.Join(tmpDir, baseName)
			require.NoError(t, os.WriteFile(mdDst, src, 0600))

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(mdDst)
			require.NoError(t, err, "Workflow file %s should compile successfully", tt.workflowFile)

			lockName := strings.TrimSuffix(baseName, ".md") + ".lock.yml"
			lockPath := filepath.Join(tmpDir, lockName)
			compiledBytes, err := os.ReadFile(lockPath)
			require.NoError(t, err)
			compiled := string(compiledBytes)

			for _, expected := range tt.expectContains {
				assert.Contains(t, compiled, expected,
					"Compiled workflow should contain %q", expected)
			}
		})
	}
}
