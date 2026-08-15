//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestActivationJobNoCheckoutStep tests that the activation job uses GitHub API
// instead of checking out the repository for the timestamp check
func TestActivationJobNoCheckoutStep(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
	}{
		{
			name: "basic workflow has no checkout in activation",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
engine: claude
strict: false
---`,
		},
		{
			name: "workflow without contents permission has no checkout in activation",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  issues: read
engine: claude
strict: false
---`,
		},
		{
			name: "workflow with reaction has no checkout in activation",
			frontmatter: `---
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  issues: read
engine: claude
strict: false
---`,
		},
		{
			// Top-level workflow permissions cannot grant write scopes directly (enforced by
			// validateDangerousPermissions), but safe-outputs such as create-pull-request require
			// contents: write in their own downstream job. This case verifies that even when the
			// workflow needs write-capable safe-outputs, the activation job itself still only
			// performs the sparse .github checkout - it never checks out the full repository.
			name: "workflow with write-capable safe-outputs still has no full checkout in activation",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
engine: claude
strict: false
safe-outputs:
  create-pull-request:
---`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "activation-checkout-test")

			testContent := tt.frontmatter + "\n\n# Test Workflow\n\nTest workflow content."
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

			compiler := NewCompiler(WithVersion("dev"))
			// Use dev mode to use local action paths
			compiler.SetActionMode(ActionModeDev)

			// Compile the workflow
			require.NoError(t, compiler.CompileWorkflow(testFile), "Failed to compile workflow")

			// Calculate the lock file path
			lockFile := stringutil.MarkdownToLockFile(testFile)

			// Read the generated lock file
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Failed to read lock file")

			lockContentStr := string(lockContent)

			// Verify activation job exists
			require.Contains(t, lockContentStr, "activation:", "Expected activation job to be present")

			// Extract the activation job section using the shared job-boundary helper
			activationJobSection := extractJobSection(lockContentStr, "activation")
			require.NotEmpty(t, activationJobSection, "Activation job section should not be empty")

			// The activation job always sparse-checks-out .github/.agents (and, in dev mode,
			// actions/setup) so it can load helper scripts and engine config - this is
			// unaffected by the workflow's permissions or triggers (including contents: write,
			// via write-capable safe-outputs). What must never happen is a full checkout of
			// the repository, or a checkout of .github/workflows for timestamp checking -
			// that always uses the GitHub API instead.

			// Verify the activation checkout is the sparse .github/.agents checkout
			assert.Contains(t, activationJobSection, "name: Checkout .github and .agents folders", "Should use the sparse .github/.agents checkout step")
			assert.Contains(t, activationJobSection, "sparse-checkout: |", "Sparse checkout should be configured")
			assert.Contains(t, activationJobSection, "sparse-checkout-cone-mode: true", "Sparse checkout cone mode should be enabled")

			// Verify it does NOT perform a full repository checkout
			assert.NotContains(t, activationJobSection, "name: Checkout repository", "Should not have a full repository checkout step")

			// Verify it does NOT checkout .github/workflows for timestamp checking
			assert.NotContains(t, activationJobSection, "Checkout workflows", "Should not have 'Checkout workflows' step - uses GitHub API for timestamp checking")

			// Verify timestamp check step is present
			assert.Contains(t, activationJobSection, "Check workflow lock file", "Should contain timestamp check step")

			// Verify scripts are loaded via require() (not inlined)
			assert.Contains(t, activationJobSection, "require(", "Should load scripts via require()")
		})
	}
}
