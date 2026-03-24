//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	goyaml "github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileVulnerabilityAlertsPermissionFiltered compiles the canonical
// test-vulnerability-alerts-permission.md workflow file and verifies that
// the GitHub App-only `vulnerability-alerts` scope does NOT appear in any
// job-level permissions block of the compiled lock file.
//
// GitHub Actions rejects workflows that declare App-only permissions at the
// job level (e.g. vulnerability-alerts, members, administration). These scopes
// must only appear as `permission-*` inputs to actions/create-github-app-token.
func TestCompileVulnerabilityAlertsPermissionFiltered(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Copy the canonical workflow file into the test's .github/workflows dir
	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-vulnerability-alerts-permission.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-vulnerability-alerts-permission.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "Failed to read source workflow file %s", srcPath)
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "Failed to write workflow to test dir")

	// Compile the workflow using the pre-built binary
	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI compile command failed:\n%s", string(output))

	// Read the compiled lock file
	lockFilePath := filepath.Join(setup.workflowsDir, "test-vulnerability-alerts-permission.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read lock file")
	lockContentStr := string(lockContent)

	// The App token minting step must receive `permission-vulnerability-alerts: read` as input.
	assert.Contains(t, lockContentStr, "permission-vulnerability-alerts: read",
		"App token minting step should include permission-vulnerability-alerts: read")
	assert.Contains(t, lockContentStr, "id: github-mcp-app-token",
		"GitHub App token minting step should be generated")

	// Critically: vulnerability-alerts must NOT appear inside any job-level permissions block.
	// Parse the YAML and walk every job's permissions map to check.
	var workflow map[string]any
	require.NoError(t, goyaml.Unmarshal(lockContent, &workflow),
		"Lock file should be valid YAML")

	jobs, ok := workflow["jobs"].(map[string]any)
	require.True(t, ok, "Lock file should have a jobs section")

	for jobName, jobConfig := range jobs {
		jobMap, ok := jobConfig.(map[string]any)
		if !ok {
			continue
		}
		perms, hasPerms := jobMap["permissions"]
		if !hasPerms {
			continue
		}
		permsMap, ok := perms.(map[string]any)
		if !ok {
			// Shorthand permissions (read-all / write-all / none) — nothing to check
			continue
		}
		assert.NotContains(t, permsMap, "vulnerability-alerts",
			"Job %q must not have vulnerability-alerts in job-level permissions block "+
				"(it is a GitHub App-only scope and not a valid GitHub Actions permission)", jobName)
	}

	// Extra belt-and-suspenders: the string "vulnerability-alerts: read" must not appear
	// anywhere other than inside the App token step inputs.
	// We verify by counting occurrences: exactly one occurrence for the App token step.
	occurrences := strings.Count(lockContentStr, "vulnerability-alerts: read")
	// The permission-vulnerability-alerts: read line contains "vulnerability-alerts: read"
	// as a substring, so we count that and only that occurrence.
	appTokenOccurrences := strings.Count(lockContentStr, "permission-vulnerability-alerts: read")
	assert.Equal(t, appTokenOccurrences, occurrences,
		"vulnerability-alerts: read should appear only inside the App token step inputs, not elsewhere in the lock file\nLock file:\n%s",
		lockContentStr)
}
