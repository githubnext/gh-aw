package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var updateExtensionCheckLog = logger.New("cli:update_extension_check")

// upgradeExtensionIfOutdated checks if a newer version of the gh-aw extension is available
// and, if so, upgrades it automatically.
//
// Returns:
//   - upgraded: true if an upgrade was performed.
//   - installPath: on Linux, the resolved path where the new binary was installed
//     (captured before any rename so the caller can relaunch the new binary from
//     the correct path even after os.Executable() starts returning a "(deleted)"
//     suffix). Empty string on non-Linux systems or when the path cannot be
//     determined.
//   - err: non-nil if the upgrade failed.
//
// When upgraded is true the CURRENTLY RUNNING PROCESS still has the old version
// baked in. The caller should re-launch the freshly-installed binary (at
// installPath) so that subsequent work (e.g. lock-file compilation) uses the
// correct new version string.
func upgradeExtensionIfOutdated(verbose bool) (bool, string, error) {
	currentVersion := GetVersion()
	updateExtensionCheckLog.Printf("Checking if extension needs upgrade (current: %s)", currentVersion)

	// Skip for non-release versions (dev builds)
	if !workflow.IsReleasedVersion(currentVersion) {
		updateExtensionCheckLog.Print("Not a released version, skipping upgrade check")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping extension upgrade check (development build)"))
		}
		return false, "", nil
	}

	// Query GitHub API for latest release
	latestVersion, err := getLatestRelease()
	if err != nil {
		// Fail silently - don't block the upgrade command if we can't reach GitHub
		updateExtensionCheckLog.Printf("Failed to check for latest release (silently ignoring): %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not check for extension updates: %v", err)))
		}
		return false, "", nil
	}

	if latestVersion == "" {
		updateExtensionCheckLog.Print("Could not determine latest version, skipping upgrade")
		return false, "", nil
	}

	updateExtensionCheckLog.Printf("Latest version: %s", latestVersion)

	// Ensure both versions have the 'v' prefix required by the semver package.
	currentSV := "v" + strings.TrimPrefix(currentVersion, "v")
	latestSV := "v" + strings.TrimPrefix(latestVersion, "v")

	// Already on the latest (or newer) version – use proper semver comparison so
	// that e.g. "0.10.0" is correctly treated as newer than "0.9.0".
	if semver.IsValid(currentSV) && semver.IsValid(latestSV) {
		if semver.Compare(currentSV, latestSV) >= 0 {
			updateExtensionCheckLog.Print("Extension is already up to date")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension is up to date"))
			}
			return false, "", nil
		}
	} else {
		// Versions are not valid semver; skip unreliable string comparison and
		// proceed with the upgrade to avoid incorrectly treating an outdated
		// version as up to date (lexicographic comparison breaks for e.g. "0.9.0" vs "0.10.0").
		updateExtensionCheckLog.Printf("Non-semver versions detected (current=%q, latest=%q); proceeding with upgrade", currentVersion, latestVersion)
	}

	// A newer version is available – upgrade automatically
	updateExtensionCheckLog.Printf("Upgrading extension from %s to %s", currentVersion, latestVersion)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Upgrading gh-aw extension from %s to %s...", currentVersion, latestVersion)))

	// First attempt: run the upgrade without touching the filesystem.
	// On most systems (and on Linux when there is no in-use binary conflict)
	// this will succeed.  On Linux with WSL the kernel may return ETXTBSY when
	// gh tries to open the currently-executing binary for writing; in that case
	// we fall through to the rename+retry path below.
	//
	// On Linux we buffer the first attempt's output rather than printing it
	// directly, so that the ETXTBSY error message is suppressed when the
	// rename+retry path succeeds and the user is not shown a confusing failure.
	var firstAttemptBuf bytes.Buffer
	firstAttemptOut := firstAttemptWriter(os.Stderr, &firstAttemptBuf)
	firstCmd := exec.Command("gh", "extension", "upgrade", "github/gh-aw")
	firstCmd.Stdout = firstAttemptOut
	firstCmd.Stderr = firstAttemptOut
	firstErr := firstCmd.Run()
	if firstErr == nil {
		// First attempt succeeded without any file manipulation.
		if runtime.GOOS == "linux" {
			// Replay the buffered output that was not shown during the attempt.
			_, _ = io.Copy(os.Stderr, &firstAttemptBuf)
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension upgraded to "+latestVersion))
		return true, "", nil
	}

	// First attempt failed.
	if runtime.GOOS != "linux" {
		// On non-Linux systems there is nothing more to try.
		return false, "", fmt.Errorf("failed to upgrade gh-aw extension: %w", firstErr)
	}

	// On Linux the failure is likely ETXTBSY.  Log the first attempt's output
	// at debug level and attempt the rename+retry workaround.
	updateExtensionCheckLog.Printf("First upgrade attempt failed (likely ETXTBSY); retrying with rename workaround. First attempt output: %s", firstAttemptBuf.String())

	// Resolve the current executable path before renaming; after the rename
	// os.Executable() returns a "(deleted)"-suffixed path on Linux.
	var installPath string
	if exe, exeErr := os.Executable(); exeErr == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(exe); resolveErr == nil {
			exe = resolved
		}
		if path, renameErr := renamePathForUpgrade(exe); renameErr != nil {
			// Rename failed; the retry will likely fail again with ETXTBSY.
			updateExtensionCheckLog.Printf("Could not rename executable for retry (upgrade will likely fail with ETXTBSY): %v", renameErr)
		} else {
			installPath = path
		}
	}

	retryCmd := exec.Command("gh", "extension", "upgrade", "github/gh-aw")
	retryCmd.Stdout = os.Stderr
	retryCmd.Stderr = os.Stderr
	if retryErr := retryCmd.Run(); retryErr != nil {
		// Retry also failed. Restore the backup so the user still has gh-aw.
		if installPath != "" {
			restoreExecutableBackup(installPath)
		}
		return false, "", fmt.Errorf("failed to upgrade gh-aw extension: %w", retryErr)
	}

	// Retry succeeded. Clean up the backup.
	if installPath != "" {
		cleanupExecutableBackup(installPath)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension upgraded to "+latestVersion))
	return true, installPath, nil
}

// firstAttemptWriter returns a writer that buffers output on Linux (so that
// ETXTBSY error messages from a failed first upgrade attempt can be suppressed
// when the rename+retry workaround succeeds) and writes directly to dst on
// other platforms.
func firstAttemptWriter(dst io.Writer, buf *bytes.Buffer) io.Writer {
	if runtime.GOOS == "linux" {
		return buf
	}
	return dst
}

// renamePathForUpgrade renames the binary at exe to exe+".bak", freeing the
// original path for the new binary to be written by gh extension upgrade.
// Returns exe (the install path) so the caller can relaunch the new binary and
// restore the backup if the upgrade fails.
func renamePathForUpgrade(exe string) (string, error) {
	backup := exe + ".bak"
	if err := os.Rename(exe, backup); err != nil {
		return "", fmt.Errorf("could not rename %s → %s: %w", exe, backup, err)
	}
	updateExtensionCheckLog.Printf("Renamed %s → %s to avoid ETXTBSY on Linux", exe, backup)
	return exe, nil
}

// restoreExecutableBackup renames the exe+".bak" backup back to exe.
// Called when the upgrade command failed and the new binary was not written.
func restoreExecutableBackup(installPath string) {
	backup := installPath + ".bak"
	if _, statErr := os.Stat(installPath); os.IsNotExist(statErr) {
		// New binary was not installed; restore the backup.
		if renErr := os.Rename(backup, installPath); renErr != nil {
			updateExtensionCheckLog.Printf("could not restore backup %s → %s: %v", backup, installPath, renErr)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to restore gh-aw backup after upgrade failure. Manually rename %s to %s to recover.", backup, installPath)))
		} else {
			updateExtensionCheckLog.Printf("Restored backup %s → %s after failed upgrade", backup, installPath)
		}
	} else {
		// New binary is present (upgrade partially succeeded); just clean up.
		_ = os.Remove(backup)
	}
}

// cleanupExecutableBackup removes the exe+".bak" backup after a successful upgrade.
func cleanupExecutableBackup(installPath string) {
	backup := installPath + ".bak"
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		updateExtensionCheckLog.Printf("Could not remove backup %s: %v", backup, err)
	}
}
