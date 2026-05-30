//go:build integration

package cli

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpgradeCommand_OnExistingRepository verifies that the upgrade command runs
// successfully against the actual project repository. It uses --no-fix to skip
// codemods, action updates, and compilation, and --skip-extension-upgrade to
// avoid a network call for the extension upgrade check.
func TestUpgradeCommand_OnExistingRepository(t *testing.T) {
	cmd := exec.Command(globalBinaryPath, "upgrade", "--no-fix", "--skip-extension-upgrade")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	t.Logf("Upgrade output: %s", outputStr)

	require.NoError(t, err, "upgrade command should succeed on existing repository, output: %s", outputStr)
	assert.Contains(t, outputStr, "Upgrade complete", "Should report upgrade complete")
}
