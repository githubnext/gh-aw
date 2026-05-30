//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileClaudeWIFAnthropic verifies that a workflow using Anthropic
// Workload Identity Federation (engine.auth.type=github-oidc with
// provider=anthropic) compiles successfully without requiring ANTHROPIC_API_KEY,
// and that the WIF fields are correctly emitted as env vars in the lock file.
//
// This is acceptance criterion 6 from issue #35937:
// "Integration test: Claude WIF workflow compiles without requiring ANTHROPIC_API_KEY secret"
func TestCompileClaudeWIFAnthropic(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Copy the canonical Anthropic WIF workflow fixture into the test's .github/workflows dir
	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-claude-wif-anthropic.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-claude-wif-anthropic.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "Failed to read source workflow file %s", srcPath)
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "Failed to write workflow to test dir")

	// Compile the workflow - it must succeed (exit 0) without ANTHROPIC_API_KEY.
	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Claude WIF Anthropic workflow must compile without error:\n%s", string(output))

	// Verify the lock file was created and contains the expected WIF env vars.
	lockFilePath := filepath.Join(setup.workflowsDir, "test-claude-wif-anthropic.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Expected lock file %s to be created", lockFilePath)
	lockStr := string(lockContent)

	// All five WIF fields from the fixture must be emitted as env vars in the compiled lock
	// file. Checking for "KEY: value" pairs ensures both the key and the value round-trip
	// correctly through the schema → parser → compiler pipeline.
	assert.Contains(t, lockStr, "AWF_AUTH_PROVIDER: anthropic", "lock file should contain AWF_AUTH_PROVIDER=anthropic")
	assert.Contains(t, lockStr, "AWF_AUTH_ANTHROPIC_FEDERATION_RULE_ID: fdrl_test", "lock file should contain AWF_AUTH_ANTHROPIC_FEDERATION_RULE_ID=fdrl_test")
	assert.Contains(t, lockStr, "AWF_AUTH_ANTHROPIC_ORGANIZATION_ID: org_test", "lock file should contain AWF_AUTH_ANTHROPIC_ORGANIZATION_ID=org_test")
	assert.Contains(t, lockStr, "AWF_AUTH_ANTHROPIC_SERVICE_ACCOUNT_ID: svac_test", "lock file should contain AWF_AUTH_ANTHROPIC_SERVICE_ACCOUNT_ID=svac_test")
	assert.Contains(t, lockStr, "AWF_AUTH_ANTHROPIC_WORKSPACE_ID: ws_test", "lock file should contain AWF_AUTH_ANTHROPIC_WORKSPACE_ID=ws_test")

	t.Logf("Anthropic WIF workflow compiled successfully to %s", lockFilePath)
}
