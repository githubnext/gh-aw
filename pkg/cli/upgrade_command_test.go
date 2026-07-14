//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeCommandHelpTextConsistency(t *testing.T) {
	cmd := NewUpgradeCommand()
	require.NotNil(t, cmd, "upgrade command should be created")

	assert.Contains(t, cmd.Long, "Upgrade the repository to the latest version of agentic workflows.", "long description should use correct grammar")

	approveFlag := cmd.Flags().Lookup("approve")
	require.NotNil(t, approveFlag, "--approve flag should exist")
	assert.Contains(t, approveFlag.Usage, "When strict mode is active", "--approve description should match compile semantics")

	preReleasesFlag := cmd.Flags().Lookup("pre-releases")
	require.NotNil(t, preReleasesFlag, "--pre-releases flag should exist")
	assert.Contains(t, preReleasesFlag.Usage, "Include pre-release versions", "--pre-releases description should mention pre-release upgrades")
	assert.Contains(t, preReleasesFlag.Usage, "installed by exact tag", "--pre-releases description should explain prerelease pinning")
	assert.Contains(t, cmd.Example, "stable releases are the default", "help text should distinguish stable releases from prereleases")

	disableCodemodFlag := cmd.Flags().Lookup("disable-codemod")
	require.NotNil(t, disableCodemodFlag, "--disable-codemod flag should exist")
	assert.Equal(t, "stringSlice", disableCodemodFlag.Value.Type())
	assert.Contains(t, disableCodemodFlag.Usage, "Disable specific codemod IDs", "--disable-codemod usage should describe codemod exclusion")
}

func TestUpgradeCommandNewFlags(t *testing.T) {
	cmd := NewUpgradeCommand()
	require.NotNil(t, cmd, "upgrade command should be created")

	// F1: --engine flag
	engineFlag := cmd.Flags().Lookup("engine")
	require.NotNil(t, engineFlag, "--engine/-e flag should exist on upgrade command")
	assert.Equal(t, "e", engineFlag.Shorthand, "--engine flag should have -e shorthand")
	assert.Contains(t, engineFlag.Usage, "Override AI engine", "--engine description should describe engine override")
	assert.Contains(t, cmd.Example, "--engine", "upgrade examples should show --engine usage")

	// F4: --repo flag
	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag, "--repo/-r flag should exist on upgrade command")
	assert.Equal(t, "r", repoFlag.Shorthand, "--repo flag should have -r shorthand")
	assert.Contains(t, repoFlag.Usage, "Target repository", "--repo description should describe target repository")
	assert.Contains(t, cmd.Example, "--repo", "upgrade examples should show --repo usage")
}

func TestUpgradeCommandRepoOrgMutualExclusion(t *testing.T) {
	cmd := NewUpgradeCommand()
	cmd.SetArgs([]string{"--repo", "owner/repo", "--org", "my-org"})
	err := cmd.Execute()
	require.Error(t, err, "should error when both --repo and --org are specified")
	assert.Contains(t, err.Error(), "--repo", "error should mention --repo flag")
	assert.Contains(t, err.Error(), "--org", "error should mention --org flag")
}
