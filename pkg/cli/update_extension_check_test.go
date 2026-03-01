//go:build !integration

package cli

import (
	"testing"

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
