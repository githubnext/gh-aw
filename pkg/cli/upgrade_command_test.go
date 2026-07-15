//go:build !integration

package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upgradeValidateEngineStub is a no-op engine validator for upgrade command tests.
func upgradeValidateEngineStub(engine string) error { return nil }

func TestUpgradeCommandHelpTextConsistency(t *testing.T) {
	cmd := NewUpgradeCommand(upgradeValidateEngineStub)
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
	cmd := NewUpgradeCommand(upgradeValidateEngineStub)
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

func TestUpgradeCommandEngineValidationRunsEarly(t *testing.T) {
	validated := false
	validate := func(engine string) error {
		validated = true
		assert.Equal(t, "bad-engine", engine)
		return assert.AnError
	}
	cmd := NewUpgradeCommand(validate)
	cmd.SetArgs([]string{"--engine", "bad-engine"})
	err := cmd.Execute()
	require.Error(t, err, "invalid engine should fail early")
	assert.True(t, validated, "engine validator should have been called")
}

func TestUpgradeCommandRepoOrgMutualExclusion(t *testing.T) {
	cmd := NewUpgradeCommand(upgradeValidateEngineStub)
	cmd.SetArgs([]string{"--repo", "owner/repo", "--org", "my-org"})
	err := cmd.Execute()
	require.Error(t, err, "should error when both --repo and --org are specified")
	assert.Contains(t, err.Error(), "--repo", "error should mention --repo flag")
	assert.Contains(t, err.Error(), "--org", "error should mention --org flag")
}

func TestUpgradeCommandFlagRegistration(t *testing.T) {
	cmd := NewUpgradeCommand(upgradeValidateEngineStub)
	require.NotNil(t, cmd, "upgrade command should be created")

	// --repo alone should be accepted at flag parse time (RunE will proceed)
	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag, "--repo flag should be registered")

	// --org alone should be accepted at flag parse time (RunE will proceed)
	orgFlag := cmd.Flags().Lookup("org")
	require.NotNil(t, orgFlag, "--org flag should be registered")

	// --engine alone should be accepted
	engineFlag := cmd.Flags().Lookup("engine")
	require.NotNil(t, engineFlag, "--engine flag should be registered")

	// --repos without --org should fail at RunE
	cmd2 := NewUpgradeCommand(upgradeValidateEngineStub)
	cmd2.SetArgs([]string{"--repos", "foo-*"})
	err := cmd2.Execute()
	require.Error(t, err, "should error when --repos is specified without --org")
	assert.Contains(t, err.Error(), "--repos", "error should mention --repos flag")
}

// TestUpgradeCommandRepoDispatchNoPR verifies that plain `upgrade --repo`
// dispatches to the target-repo runner without requesting PR creation.
func TestUpgradeCommandRepoDispatchNoPR(t *testing.T) {
	origFn := runUpgradeForTargetRepoFn
	defer func() { runUpgradeForTargetRepoFn = origFn }()

	var capturedCreatePR bool
	runUpgradeForTargetRepoFn = func(_ context.Context, repo string, _ upgradeOptions, createPR bool, _ bool) error {
		capturedCreatePR = createPR
		return nil
	}

	cmd := NewUpgradeCommand(upgradeValidateEngineStub)
	cmd.SetArgs([]string{"--repo", "owner/repo"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.False(t, capturedCreatePR, "plain --repo must not request PR creation")
}

// TestUpgradeCommandRepoDispatchWithPR verifies that `upgrade --repo --create-pull-request`
// dispatches to the target-repo runner with PR creation requested.
func TestUpgradeCommandRepoDispatchWithPR(t *testing.T) {
	origFn := runUpgradeForTargetRepoFn
	defer func() { runUpgradeForTargetRepoFn = origFn }()

	var capturedCreatePR bool
	runUpgradeForTargetRepoFn = func(_ context.Context, repo string, _ upgradeOptions, createPR bool, _ bool) error {
		capturedCreatePR = createPR
		return nil
	}

	cmd := NewUpgradeCommand(upgradeValidateEngineStub)
	cmd.SetArgs([]string{"--repo", "owner/repo", "--create-pull-request"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.True(t, capturedCreatePR, "--repo --create-pull-request must request PR creation")
}
