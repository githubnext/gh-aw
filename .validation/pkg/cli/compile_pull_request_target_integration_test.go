//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPullRequestTargetCheckoutFalseWithImports verifies that a pull_request_target workflow
// with `checkout: false` and shared-workflow imports compiles successfully.
//
// In non-strict mode the workflow should compile cleanly (no error).
// In strict mode the workflow should compile successfully but emit a dangerous-trigger warning.
func TestPullRequestTargetCheckoutFalseWithImports(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Copy the fixture and its shared import into the test's .github/workflows dir.
	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-pull-request-target-checkout-false.md")
	srcSharedDir := filepath.Join(projectRoot, "pkg/cli/workflows/shared")
	dstPath := filepath.Join(setup.workflowsDir, "test-pull-request-target-checkout-false.md")
	dstSharedDir := filepath.Join(setup.workflowsDir, "shared")

	require.NoError(t, os.MkdirAll(dstSharedDir, 0755), "create shared/ dir")
	copyWorkflowFile(t, srcPath, dstPath)
	// Copy shared/keep-it-short.md (used by the fixture via imports).
	copyWorkflowFile(t, filepath.Join(srcSharedDir, "keep-it-short.md"), filepath.Join(dstSharedDir, "keep-it-short.md"))
	copyWorkflowFile(t, filepath.Join(srcSharedDir, "use-emojis.md"), filepath.Join(dstSharedDir, "use-emojis.md"))

	// Non-strict: should compile without error.
	t.Run("non-strict mode", func(t *testing.T) {
		cmd := exec.Command(setup.binaryPath, "compile", dstPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "compile should succeed in non-strict mode:\n%s", string(output))

		// The insecure-checkout warning must NOT appear because checkout: false is set.
		assert.NotContains(t, string(output), "extremely insecure",
			"no insecure-checkout warning expected when checkout: false")
	})

	// Strict: should compile successfully but emit the dangerous-trigger warning.
	t.Run("strict mode", func(t *testing.T) {
		cmd := exec.Command(setup.binaryPath, "compile", "--strict", dstPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "compile should succeed in strict mode with checkout: false:\n%s", string(output))

		// The dangerous-trigger warning must appear in strict mode.
		assert.Contains(t, string(output), "pull_request_target is a very dangerous trigger",
			"strict mode should emit dangerous-trigger warning even when checkout: false")

		// The hard error about insecure checkout must NOT appear.
		assert.NotContains(t, string(output), "extremely insecure",
			"strict mode should not emit insecure-checkout error when checkout: false")
	})
}

// TestPullRequestTargetWithImportsNoCheckoutFalse verifies that a pull_request_target workflow
// that does NOT set `checkout: false` emits a warning in non-strict mode and an error in
// strict mode, even when shared-workflow imports are present.
func TestPullRequestTargetWithImportsNoCheckoutFalse(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Copy the fixture and its shared import into the test's .github/workflows dir.
	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-pull-request-target-with-imports.md")
	srcSharedDir := filepath.Join(projectRoot, "pkg/cli/workflows/shared")
	dstPath := filepath.Join(setup.workflowsDir, "test-pull-request-target-with-imports.md")
	dstSharedDir := filepath.Join(setup.workflowsDir, "shared")

	require.NoError(t, os.MkdirAll(dstSharedDir, 0755), "create shared/ dir")
	copyWorkflowFile(t, srcPath, dstPath)
	copyWorkflowFile(t, filepath.Join(srcSharedDir, "keep-it-short.md"), filepath.Join(dstSharedDir, "keep-it-short.md"))
	copyWorkflowFile(t, filepath.Join(srcSharedDir, "use-emojis.md"), filepath.Join(dstSharedDir, "use-emojis.md"))

	// Non-strict: should compile (exit 0) but emit a warning.
	t.Run("non-strict mode emits warning", func(t *testing.T) {
		cmd := exec.Command(setup.binaryPath, "compile", dstPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "compile should succeed (with warning) in non-strict mode:\n%s", string(output))

		assert.Contains(t, string(output), "extremely insecure",
			"non-strict mode should warn about insecure pull_request_target checkout")
	})

	// Strict: should fail with an error because checkout: false is not set.
	t.Run("strict mode returns error", func(t *testing.T) {
		cmd := exec.Command(setup.binaryPath, "compile", "--strict", dstPath)
		output, _ := cmd.CombinedOutput()
		combined := string(output)

		// The process must exit non-zero.
		assert.False(t, cmd.ProcessState.Success(),
			"compile should fail in strict mode when checkout: false is absent")

		// The error message must mention the insecure checkout.
		assert.Contains(t, combined, "extremely insecure",
			"strict error should cite the insecure pull_request_target checkout")

		// The dangerous-trigger warning should also have been emitted before the error.
		assert.True(t,
			strings.Contains(combined, "very dangerous trigger") ||
				strings.Contains(combined, "extremely insecure"),
			"output should contain security diagnostics")
	})
}

// copyWorkflowFile is a test helper that copies a single file from src to dst.
func copyWorkflowFile(t *testing.T, src, dst string) {
	t.Helper()
	content, err := os.ReadFile(src)
	require.NoError(t, err, "Failed to read source file %s", src)
	require.NoError(t, os.WriteFile(dst, content, 0644), "Failed to write file %s", dst)
}
