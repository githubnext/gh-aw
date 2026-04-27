//go:build !integration

package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeExtensionIfOutdated_DevBuild(t *testing.T) {
	// Save original version and restore after test
	originalVersion := GetVersion()
	defer SetVersionInfo(originalVersion)

	// Set a dev version – upgrade check must be skipped for dev builds because
	// workflow.IsReleasedVersion returns false for non-release builds.
	SetVersionInfo("dev")

	// Verify the function exits before making any API calls.
	// If it did make API calls we'd see a network error in test environments,
	// but the function must return (false, "", nil) immediately.
	upgraded, installPath, err := upgradeExtensionIfOutdated(false, false)
	require.NoError(t, err, "Should not return error for dev builds")
	assert.False(t, upgraded, "Should not report upgrade for dev builds")
	assert.Empty(t, installPath, "installPath should be empty for dev builds")
}

func TestUpgradeExtensionIfOutdated_SilentFailureOnAPIError(t *testing.T) {
	// When the GitHub API is unreachable the function must fail silently and
	// must NOT report an upgrade so that the rest of the upgrade command
	// continues unaffected.

	originalVersion := GetVersion()
	defer SetVersionInfo(originalVersion)

	// Use a release version so the API call is attempted
	SetVersionInfo("v0.1.0")

	upgraded, installPath, err := upgradeExtensionIfOutdated(false, false)
	require.NoError(t, err, "Should fail silently on API errors")
	assert.False(t, upgraded, "Should not report upgrade when API is unreachable")
	assert.Empty(t, installPath, "installPath should be empty when API is unreachable")
}

func TestFirstAttemptWriter_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only behavior")
	}
	var buf bytes.Buffer
	dst := &bytes.Buffer{}
	w := firstAttemptWriter(dst, &buf)
	// On Linux the writer should be the buffer, not dst.
	assert.Equal(t, &buf, w, "firstAttemptWriter should return the buffer on Linux")
}

func TestFirstAttemptWriter_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only behavior")
	}
	var buf bytes.Buffer
	dst := &bytes.Buffer{}
	w := firstAttemptWriter(dst, &buf)
	// On Windows the writer should be the buffer (rename+retry workaround).
	assert.Equal(t, &buf, w, "firstAttemptWriter should return the buffer on Windows")
}

func TestFirstAttemptWriter_NonLinuxNonWindows(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		t.Skip("Non-Linux/non-Windows behavior only")
	}
	var buf bytes.Buffer
	dst := &bytes.Buffer{}
	w := firstAttemptWriter(dst, &buf)
	// On other platforms the writer should be dst.
	assert.Equal(t, dst, w, "firstAttemptWriter should return dst on non-Linux/non-Windows")
}

func TestNeedsRenameWorkaround(t *testing.T) {
	result := needsRenameWorkaround()
	expected := runtime.GOOS == "linux" || runtime.GOOS == "windows"
	assert.Equal(t, expected, result, "needsRenameWorkaround should return true only on Linux and Windows")
}

func TestRenamePathForUpgrade(t *testing.T) {
	// Create a temporary file to act as the "executable".
	dir := t.TempDir()
	exe := filepath.Join(dir, "gh-aw")
	require.NoError(t, os.WriteFile(exe, []byte("binary"), 0o755), "Should create temp executable")

	installPath, backupPath, err := renamePathForUpgrade(exe)
	require.NoError(t, err, "renamePathForUpgrade should succeed")
	assert.Equal(t, exe, installPath, "installPath should equal the original exe path")
	assert.NotEmpty(t, backupPath, "backupPath should be non-empty")
	assert.Contains(t, backupPath, ".bak", "backupPath should have .bak suffix")

	// The original path should no longer exist.
	_, statErr := os.Stat(exe)
	assert.True(t, os.IsNotExist(statErr), "Original executable should have been renamed away")

	// The backup should exist at the returned path.
	_, statErr = os.Stat(backupPath)
	assert.NoError(t, statErr, "Backup file should exist")
}

func TestRenamePathForUpgrade_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "nonexistent")

	_, _, err := renamePathForUpgrade(exe)
	assert.Error(t, err, "renamePathForUpgrade should fail for non-existent file")
}

func TestRestoreExecutableBackup_NoNewBinary(t *testing.T) {
	// Simulate: backup exists, new binary was NOT written (upgrade failed).
	dir := t.TempDir()
	installPath := filepath.Join(dir, "gh-aw")
	backup := installPath + ".99999.bak"

	require.NoError(t, os.WriteFile(backup, []byte("old binary"), 0o755), "Should create backup")

	restoreExecutableBackup(installPath, backup)

	// Backup should be renamed back to installPath.
	_, statErr := os.Stat(installPath)
	require.NoError(t, statErr, "Original executable should be restored")

	// Backup file should be gone.
	_, statErr = os.Stat(backup)
	assert.True(t, os.IsNotExist(statErr), "Backup file should have been renamed away")
}

func TestRestoreExecutableBackup_NewBinaryPresent(t *testing.T) {
	// Simulate: both backup and new binary exist (upgrade partially succeeded).
	dir := t.TempDir()
	installPath := filepath.Join(dir, "gh-aw")
	backup := installPath + ".99999.bak"

	require.NoError(t, os.WriteFile(installPath, []byte("new binary"), 0o755), "Should create new binary")
	require.NoError(t, os.WriteFile(backup, []byte("old binary"), 0o755), "Should create backup")

	restoreExecutableBackup(installPath, backup)

	// New binary should still be present.
	_, statErr := os.Stat(installPath)
	require.NoError(t, statErr, "New binary should remain intact")

	// Backup should be cleaned up.
	_, statErr = os.Stat(backup)
	assert.True(t, os.IsNotExist(statErr), "Backup file should be cleaned up")
}

func TestCleanupExecutableBackup(t *testing.T) {
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "gh-aw.99999.bak")

	require.NoError(t, os.WriteFile(backupPath, []byte("old binary"), 0o755), "Should create backup")

	cleanupExecutableBackup(backupPath)

	// Backup file should be removed.
	_, statErr := os.Stat(backupPath)
	assert.True(t, os.IsNotExist(statErr), "Backup file should be removed after cleanup")
}

func TestCleanupExecutableBackup_NoBackup(t *testing.T) {
	// Should not fail if backup doesn't exist.
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "gh-aw.99999.bak")

	// No panic or error expected.
	cleanupExecutableBackup(backupPath)
}

func TestIsWindowsLockError(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		err      error
		expected bool
	}{
		{
			name:     "access denied in output",
			output:   "gh: Access is denied.\n",
			err:      nil,
			expected: true,
		},
		{
			name:     "sharing violation in output",
			output:   "The process cannot access the file because it is being used by another process.",
			err:      nil,
			expected: true,
		},
		{
			name:     "access denied in error",
			output:   "",
			err:      errors.New("exit status 1: Access is denied"),
			expected: true,
		},
		{
			name:     "unrelated error",
			output:   "gh: 401 Unauthorized",
			err:      errors.New("exit status 1"),
			expected: false,
		},
		{
			name:     "empty output and nil error",
			output:   "",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWindowsLockError(tt.output, tt.err)
			assert.Equal(t, tt.expected, result, "isWindowsLockError result mismatch")
		})
	}
}

func TestExtensionUpgradeArgs(t *testing.T) {
	args := extensionUpgradeArgs()
	assert.Equal(t, []string{"extension", "upgrade", "github/gh-aw", "--force"}, args, "upgrade command must force upgrades for pinned extensions")
}
