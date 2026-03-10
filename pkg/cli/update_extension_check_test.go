//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureLatestExtensionVersion_DevBuild(t *testing.T) {
	// Save original version and restore after test
	originalVersion := GetVersion()
	defer SetVersionInfo(originalVersion)

	// Set a dev version
	SetVersionInfo("dev")

	// Should return nil without error for dev builds
	err := ensureLatestExtensionVersion(false)
	require.NoError(t, err, "Should not return error for dev builds")
}

func TestEnsureLatestExtensionVersion_SilentFailure(t *testing.T) {
	// This test verifies that network/API errors are handled silently
	// The actual API call will fail in the test environment but should not return an error

	// Save original version and restore after test
	originalVersion := GetVersion()
	defer SetVersionInfo(originalVersion)

	// Set a valid release version
	SetVersionInfo("v0.1.0")

	// Should return nil even if API call fails (fails silently)
	err := ensureLatestExtensionVersion(false)
	require.NoError(t, err, "Should fail silently on API errors")
}

func TestUpgradeExtensionIfOutdated_DevBuild(t *testing.T) {
	// Save original version and restore after test
	originalVersion := GetVersion()
	defer SetVersionInfo(originalVersion)

	// Set a dev version – upgrade check must be skipped for dev builds because
	// workflow.IsReleasedVersion returns false for non-release builds.
	SetVersionInfo("dev")

	// Verify the function exits before making any API calls.
	// If it did make API calls we'd see a network error in test environments,
	// but the function must return (false, nil) immediately.
	upgraded, err := upgradeExtensionIfOutdated(false)
	require.NoError(t, err, "Should not return error for dev builds")
	assert.False(t, upgraded, "Should not report upgrade for dev builds")
}

func TestUpgradeExtensionIfOutdated_SilentFailureOnAPIError(t *testing.T) {
	// When the GitHub API is unreachable the function must fail silently and
	// must NOT report an upgrade so that the rest of the upgrade command
	// continues unaffected.

	originalVersion := GetVersion()
	defer SetVersionInfo(originalVersion)

	// Use a release version so the API call is attempted
	SetVersionInfo("v0.1.0")

	upgraded, err := upgradeExtensionIfOutdated(false)
	require.NoError(t, err, "Should fail silently on API errors")
	assert.False(t, upgraded, "Should not report upgrade when API is unreachable")
}
