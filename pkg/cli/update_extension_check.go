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
//   - installPath: on Linux or Windows, the resolved path where the new binary
//     was installed (captured before any rename so the caller can relaunch the
//     new binary from the correct path; on Linux os.Executable() may return a
//     "(deleted)"-suffixed path after the rename). Empty string on other systems
//     or when the path cannot be determined.
//   - err: non-nil if the upgrade failed.
//
// When upgraded is true the CURRENTLY RUNNING PROCESS still has the old version
// baked in. The caller should re-launch the freshly-installed binary (at
// installPath) so that subsequent work (e.g. lock-file compilation) uses the
// correct new version string.
func upgradeExtensionIfOutdated(verbose bool, includePrereleases bool) (bool, string, error) {
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
	latestVersion, err := getLatestRelease(includePrereleases)
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
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension is up to date"))
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
	// On most systems this will succeed.  On Linux with WSL the kernel may
	// return ETXTBSY when gh tries to open the currently-executing binary for
	// writing; on Windows the OS returns "Access is denied" for the same
	// reason.  In both cases we fall through to the rename+retry path below.
	//
	// On Linux and Windows we buffer the first attempt's output rather than
	// printing it directly, so that the error message is suppressed when the
	// rename+retry path succeeds and the user is not shown a confusing failure.
	var firstAttemptBuf bytes.Buffer
	firstAttemptOut := firstAttemptWriter(os.Stderr, &firstAttemptBuf)
	firstCmd := exec.Command("gh", extensionUpgradeArgs()...)
	firstCmd.Stdout = firstAttemptOut
	firstCmd.Stderr = firstAttemptOut
	firstErr := firstCmd.Run()
	if firstErr == nil {
		// First attempt succeeded without any file manipulation.
		if needsRenameWorkaround() {
			// Replay the buffered output that was not shown during the attempt.
			_, _ = io.Copy(os.Stderr, &firstAttemptBuf)
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension upgraded to "+latestVersion))
		return true, "", nil
	}

	// First attempt failed.
	if !needsRenameWorkaround() {
		// On platforms other than Linux and Windows there is nothing more to try.
		return false, "", fmt.Errorf("failed to upgrade gh-aw extension: %w", firstErr)
	}

	// On Linux the failure is likely ETXTBSY; on Windows it is likely
	// "Access is denied". Both arise because the OS prevents overwriting a
	// running binary. Attempt the rename+retry workaround: rename the
	// currently-running binary away to free up its path, then retry the
	// upgrade so that gh can write the new binary at the original location.
	updateExtensionCheckLog.Printf("First upgrade attempt failed (likely locked binary); retrying with rename workaround. First attempt output: %s", firstAttemptBuf.String())

	// Resolve the current executable path before renaming; after the rename
	// os.Executable() returns a "(deleted)"-suffixed path on Linux.
	var installPath string
	var backupPath string
	if exe, exeErr := os.Executable(); exeErr == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(exe); resolveErr == nil {
			exe = resolved
		}
		if iPath, bPath, renameErr := renamePathForUpgrade(exe); renameErr != nil {
			// Rename failed; the retry will likely fail again.
			updateExtensionCheckLog.Printf("Could not rename executable for retry (upgrade will likely fail): %v", renameErr)
		} else {
			installPath = iPath
			backupPath = bPath
		}
	}

	retryCmd := exec.Command("gh", extensionUpgradeArgs()...)
	retryCmd.Stdout = os.Stderr
	retryCmd.Stderr = os.Stderr
	if retryErr := retryCmd.Run(); retryErr != nil {
		// Retry also failed. Restore the backup so the user still has gh-aw.
		if backupPath != "" {
			restoreExecutableBackup(installPath, backupPath)
		}
		if runtime.GOOS == "windows" && isWindowsLockError(firstAttemptBuf.String(), retryErr) {
			// On Windows, self-upgrade may not be possible while the binary is
			// running. Guide the user to upgrade manually from a separate shell.
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("On Windows, gh-aw cannot self-upgrade while it is running."))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please upgrade manually by running one of the following:"))
			fmt.Fprintln(os.Stderr, "  gh extension upgrade gh-aw")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("If that does not work, try reinstalling:"))
			fmt.Fprintln(os.Stderr, "  gh extension remove gh-aw")
			fmt.Fprintln(os.Stderr, "  gh extension install github/gh-aw")
		}
		return false, "", fmt.Errorf("failed to upgrade gh-aw extension: %w", retryErr)
	}

	// Retry succeeded. Clean up the backup.
	if backupPath != "" {
		cleanupExecutableBackup(backupPath)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension upgraded to "+latestVersion))
	return true, installPath, nil
}

// needsRenameWorkaround reports whether the current platform requires the
// rename+retry workaround when upgrading the running binary.
//
// On Linux, overwriting a running binary returns ETXTBSY.
// On Windows, the same operation returns "Access is denied".
// Both errors are resolved by renaming the current binary away first.
func needsRenameWorkaround() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "windows"
}

// firstAttemptWriter returns a writer that buffers output on platforms that
// use the rename+retry workaround (Linux and Windows), so that error messages
// from a failed first upgrade attempt are suppressed when the retry succeeds.
// On other platforms it writes directly to dst.
func firstAttemptWriter(dst io.Writer, buf *bytes.Buffer) io.Writer {
	if needsRenameWorkaround() {
		return buf
	}
	return dst
}

// renamePathForUpgrade renames the binary at exe to a PID-qualified backup
// path (exe+".<pid>.bak"), freeing the original path for the new binary to be
// written by gh extension upgrade.  Using a PID-qualified name ensures each
// invocation gets a unique backup so that a failed cleanup (e.g. Windows cannot
// remove a running binary) does not cause the destination to already exist on
// a subsequent upgrade attempt.
// Returns the install path (exe) and the backup path so the caller can
// relaunch the new binary and restore or clean up the backup.
func renamePathForUpgrade(exe string) (string, string, error) {
	backup := fmt.Sprintf("%s.%d.bak", exe, os.Getpid())
	if err := os.Rename(exe, backup); err != nil {
		return "", "", fmt.Errorf("could not rename %s → %s: %w", exe, backup, err)
	}
	updateExtensionCheckLog.Printf("Renamed %s → %s to free path for upgrade", exe, backup)
	return exe, backup, nil
}

// restoreExecutableBackup renames backupPath back to installPath.
// Called when the upgrade command failed and the new binary was not written.
func restoreExecutableBackup(installPath, backupPath string) {
	if _, statErr := os.Stat(installPath); os.IsNotExist(statErr) {
		// New binary was not installed; restore the backup.
		if renErr := os.Rename(backupPath, installPath); renErr != nil {
			updateExtensionCheckLog.Printf("could not restore backup %s → %s: %v", backupPath, installPath, renErr)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to restore gh-aw backup after upgrade failure. Manually rename %s to %s to recover.", backupPath, installPath)))
		} else {
			updateExtensionCheckLog.Printf("Restored backup %s → %s after failed upgrade", backupPath, installPath)
		}
	} else {
		// New binary is present (upgrade partially succeeded); just clean up.
		_ = os.Remove(backupPath)
	}
}

// cleanupExecutableBackup removes backupPath after a successful upgrade.
func cleanupExecutableBackup(backupPath string) {
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		updateExtensionCheckLog.Printf("Could not remove backup %s: %v", backupPath, err)
	}
}

// isWindowsLockError reports whether the output or error from an upgrade
// attempt indicate a Windows file-locking issue (the running-binary-lock
// symptom).  Only when a lock error is detected should the Windows-specific
// self-upgrade guidance be shown; other failures should propagate the
// underlying error message instead.
func isWindowsLockError(output string, err error) bool {
	lockMsgs := []string{"Access is denied", "The process cannot access the file"}
	for _, msg := range lockMsgs {
		if strings.Contains(output, msg) {
			return true
		}
		if err != nil && strings.Contains(err.Error(), msg) {
			return true
		}
	}
	return false
}

// extensionUpgradeArgs returns the gh extension upgrade invocation used by
// self-upgrade checks.
//
// --force is required so pinned installs (e.g. `gh extension install ... --pin`)
// can be upgraded in-place.
func extensionUpgradeArgs() []string {
	return []string{"extension", "upgrade", "github/gh-aw", "--force"}
}
