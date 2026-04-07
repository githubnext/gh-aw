//go:build integration

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeUpdateWorkflowWithSecret is a minimal workflow that includes a custom job
// with a non-GITHUB_TOKEN secret in its environment.  The secret reference will
// appear in the compiled YAML body and be detected by CollectSecretReferences.
const safeUpdateWorkflowWithSecret = `---
name: Safe Update Secret Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
jobs:
  secret-job:
    runs-on: ubuntu-latest
    needs: [activation]
    env:
      MY_API_SECRET: ${{ secrets.MY_API_SECRET }}
    steps:
      - run: echo "hello"
---

# Safe Update Secret Test

Test workflow that uses a custom secret in a custom job.
`

// safeUpdateWorkflowWithCustomAction is a minimal workflow that includes a custom
// job using a non-actions/* action reference.  The uses: line will appear in the
// compiled YAML body and be detected by CollectActionReferences.
const safeUpdateWorkflowWithCustomAction = `---
name: Safe Update Action Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
jobs:
  action-job:
    runs-on: ubuntu-latest
    needs: [activation]
    steps:
      - uses: my-org/custom-action@v1
---

# Safe Update Action Test

Test workflow that uses a custom action in a custom job.
`

// safeUpdateWorkflowBasic is a minimal workflow that uses only GITHUB_TOKEN and
// actions/* actions.  Safe update mode should allow it on a first compile.
const safeUpdateWorkflowBasic = `---
name: Safe Update Basic Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
---

# Safe Update Basic Test

Test workflow that uses only GITHUB_TOKEN.
`

// manifestWithAPISecret is a minimal lock file content containing a gh-aw-manifest
// that pre-approves secrets.MY_API_SECRET.  Writing this to the lock file path
// before compilation simulates a workflow that was previously compiled and approved.
func manifestLockFileWithSecret(secretName string) string {
	return fmt.Sprintf(
		"# gh-aw-metadata: {\"schema_version\":\"v3\",\"frontmatter_hash\":\"abc\",\"agent_id\":\"copilot\"}\n"+
			"# gh-aw-manifest: {\"version\":1,\"secrets\":[\"secrets.%s\",\"secrets.GITHUB_TOKEN\"],\"actions\":[]}\n"+
			"name: placeholder\n",
		secretName,
	)
}

// TestSafeUpdateRejectsNewSecretOnFirstCompile verifies that --safe-update rejects
// a first compilation that introduces a non-GITHUB_TOKEN secret when no lock file
// (and therefore no prior manifest) exists yet.
func TestSafeUpdateRejectsNewSecretOnFirstCompile(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-secret.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithSecret), 0o644),
		"should write workflow file")

	// Use release mode so the compiler reads the lock file from the filesystem
	// (not from git HEAD), allowing us to control the manifest via file I/O.
	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	assert.Error(t, err, "compile should fail in safe update mode when a new secret is introduced")
	assert.Contains(t, outputStr, "safe update mode", "error should mention safe update mode")
	assert.Contains(t, outputStr, "MY_API_SECRET", "error should name the offending secret")
	t.Logf("Safe update correctly rejected new secret.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateRejectsNewCustomActionOnFirstCompile verifies that --safe-update
// rejects a first compilation that introduces a non-actions/* action reference when
// no prior manifest exists.
func TestSafeUpdateRejectsNewCustomActionOnFirstCompile(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-action.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithCustomAction), 0o644),
		"should write workflow file")

	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	assert.Error(t, err, "compile should fail in safe update mode when a new custom action is introduced")
	assert.Contains(t, outputStr, "safe update mode", "error should mention safe update mode")
	assert.Contains(t, outputStr, "my-org/custom-action", "error should name the offending action")
	t.Logf("Safe update correctly rejected new custom action.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateAllowsKnownSecretWithPriorManifest verifies that --safe-update
// allows a compilation when the secret is already recorded in the prior manifest
// embedded in the existing lock file.
func TestSafeUpdateAllowsKnownSecretWithPriorManifest(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-known-secret.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithSecret), 0o644),
		"should write workflow file")

	// Pre-create a lock file whose gh-aw-manifest already records MY_API_SECRET.
	// In release mode the compiler reads the manifest from the filesystem, so this
	// simulates a workflow that was previously compiled and approved.
	lockFilePath := filepath.Join(setup.workflowsDir, "safe-update-known-secret.lock.yml")
	require.NoError(t, os.WriteFile(lockFilePath, []byte(manifestLockFileWithSecret("MY_API_SECRET")), 0o644),
		"should write pre-existing lock file with manifest")

	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	assert.NoError(t, err, "compile should succeed when the secret is in the prior manifest\nOutput:\n%s", outputStr)
	t.Logf("Safe update correctly allowed known secret.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateAllowsGitHubTokenOnFirstCompile verifies that --safe-update allows
// a first compilation that only uses GITHUB_TOKEN (always-permitted) with no prior
// lock file present.
func TestSafeUpdateAllowsGitHubTokenOnFirstCompile(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-basic.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowBasic), 0o644),
		"should write workflow file")

	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	assert.NoError(t, err, "compile should succeed when only GITHUB_TOKEN is used\nOutput:\n%s", outputStr)

	// Also verify the lock file was created.
	lockFilePath := filepath.Join(setup.workflowsDir, "safe-update-basic.lock.yml")
	_, statErr := os.Stat(lockFilePath)
	assert.NoError(t, statErr, "lock file should be created on successful safe-update compilation")

	// Verify the manifest is embedded in the lock file.
	lockContent, readErr := os.ReadFile(lockFilePath)
	require.NoError(t, readErr, "should read lock file")
	assert.Contains(t, string(lockContent), "gh-aw-manifest:", "lock file should contain a gh-aw-manifest header")
	assert.NotContains(t, string(lockContent), "MY_API_SECRET", "lock file manifest should not contain unapproved secrets")
	t.Logf("Safe update correctly allowed GITHUB_TOKEN-only workflow.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateNoFlagAllowsNewSecret verifies that without --safe-update the
// compiler accepts a new secret freely (safe update enforcement is disabled by default).
func TestSafeUpdateNoFlagAllowsNewSecret(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "no-safe-update.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithSecret), 0o644),
		"should write workflow file")

	// No --safe-update flag; compilation should succeed even though a new secret is present.
	cmd := exec.Command(setup.binaryPath, "compile", workflowPath)
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	assert.NoError(t, err, "compile without --safe-update should succeed regardless of new secrets\nOutput:\n%s", outputStr)
	assert.False(t, strings.Contains(outputStr, "safe update mode"),
		"output should not mention safe update mode when flag is not set")
	t.Logf("Compilation without safe update flag succeeded as expected.\nOutput:\n%s", outputStr)
}
