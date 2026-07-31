package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var updateExtensionCheckLog = logger.New("cli:update_extension_check")

// maxBackupCleanupAttempts is the number of times cleanupStaleWindowsBackups
// retries removing a stale .bak file before giving up.
const maxBackupCleanupAttempts = 3

// backupCleanupRetryDelay is the pause between successive cleanup attempts.
// The delay allows transient locks (e.g. Windows Defender scanning) to clear.
const backupCleanupRetryDelay = 300 * time.Millisecond

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
func upgradeExtensionIfOutdated(ctx context.Context, verbose bool, includePrereleases bool) (bool, string, error) {
	currentVersion := GetVersion()
	updateExtensionCheckLog.Printf("Checking if extension needs upgrade (current: %s)", currentVersion)

	latestVersion, skip, err := checkExtensionNeedsUpgrade(ctx, currentVersion, verbose, includePrereleases)
	if err != nil || skip {
		return false, "", err
	}

	// A newer version is available – upgrade automatically
	updateExtensionCheckLog.Printf("Upgrading extension from %s to %s", currentVersion, latestVersion)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Upgrading gh-aw extension from %s to %s...", renderReleaseVersion(currentVersion), renderReleaseVersion(latestVersion))))

	// On platforms without binary locking (e.g. macOS), skip the standard
	// upgrade path for prereleases because gh extension upgrade --force resolves
	// via /releases/latest which excludes prereleases.
	if includePrereleases && !needsRenameWorkaround() {
		return tryPrereleaseDirectInstall(latestVersion)
	}

	done, firstAttemptBuf, firstErr, terminalErr := performFirstUpgradeAttempt(latestVersion)
	if terminalErr != nil {
		return false, "", terminalErr
	}
	if done {
		return true, "", nil
	}
	if !needsRenameWorkaround() {
		return false, "", fmt.Errorf("failed to upgrade gh-aw extension: %w", firstErr)
	}

	updateExtensionCheckLog.Printf("First upgrade attempt failed (likely locked binary); retrying with rename workaround. First attempt output: %s", firstAttemptBuf.String())
	installPath, backupPath := resolveExecutableForRename()
	resultPath, retryErr := performRenameRetryInstall(latestVersion, installPath, backupPath, firstAttemptBuf, firstErr)
	if retryErr != nil {
		return false, "", retryErr
	}
	return true, resultPath, nil
}

// checkExtensionNeedsUpgrade determines whether the extension should be upgraded.
// Returns (latestVersion, skip, err): when skip=true the caller should return immediately.
func checkExtensionNeedsUpgrade(ctx context.Context, currentVersion string, verbose bool, includePrereleases bool) (string, bool, error) {
	// Skip for non-release versions (dev builds)
	if !workflow.IsReleasedVersion(currentVersion) {
		updateExtensionCheckLog.Print("Not a released version, skipping upgrade check")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping extension upgrade check (development build)"))
		}
		return "", true, nil
	}

	// Query GitHub API for latest release
	latestVersion, err := getLatestRelease(ctx, includePrereleases)
	if err != nil {
		// Fail silently - don't block the upgrade command if we can't reach GitHub
		updateExtensionCheckLog.Printf("Failed to check for latest release (silently ignoring): %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not check for extension updates: %v", err)))
		}
		return "", true, nil
	}

	if latestVersion == "" {
		updateExtensionCheckLog.Print("Could not determine latest version, skipping upgrade")
		return "", true, nil
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
			if notice := prereleaseChannelNotice(currentVersion, latestVersion, includePrereleases); len(notice) > 0 {
				for _, line := range notice {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(line))
				}
			} else if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension is up to date"))
			}
			return "", true, nil
		}
	} else {
		// Versions are not valid semver; proceed with upgrade to avoid treating
		// an outdated version as up to date (lexicographic comparison is unreliable).
		updateExtensionCheckLog.Printf("Non-semver versions detected (current=%q, latest=%q); proceeding with upgrade", currentVersion, latestVersion)
	}

	return latestVersion, false, nil
}

// tryPrereleaseDirectInstall installs an exact prerelease version via pin on
// platforms that do not lock the running binary (e.g. macOS).
func tryPrereleaseDirectInstall(latestVersion string) (bool, string, error) {
	updateExtensionCheckLog.Printf("Prerelease upgrade on macOS: skipping gh extension upgrade (uses /releases/latest, ignores prereleases), using pin-based install for %s", latestVersion)
	removeCmd := ghCmdForExtension("extension", "remove", extensionRepo)
	removeCmd.Stdout = os.Stderr
	removeCmd.Stderr = os.Stderr
	if removeErr := removeCmd.Run(); removeErr != nil {
		updateExtensionCheckLog.Printf("Could not remove extension before pin-based install (continuing anyway): %v", removeErr)
	}
	pinCmd := ghCmdForExtension("extension", "install", extensionRepo, "--pin", latestVersion)
	pinCmd.Stdout = os.Stderr
	pinCmd.Stderr = os.Stderr
	if pinErr := pinCmd.Run(); pinErr != nil {
		return false, "", fmt.Errorf("failed to install gh-aw extension at version %s: %w", renderReleaseVersion(latestVersion), pinErr)
	}
	_, versionErr := verifyInstalledVersion(latestVersion)
	if versionErr != nil {
		return false, "", fmt.Errorf("failed to verify gh-aw extension version after upgrade: %w", versionErr)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension upgraded to "+renderReleaseVersion(latestVersion)))
	return true, "", nil
}

// performFirstUpgradeAttempt runs the standard gh extension upgrade command.
// Returns (done, buf, firstErr, terminalErr):
//   - done=true: upgrade succeeded; caller should return true, "", nil
//   - terminalErr!=nil: non-retryable failure; caller should return false, "", terminalErr
//   - otherwise: first attempt failed; caller should check needsRenameWorkaround
func performFirstUpgradeAttempt(latestVersion string) (done bool, buf bytes.Buffer, firstErr error, terminalErr error) {
	firstAttemptOut := firstAttemptWriter(os.Stderr, &buf)
	firstCmd := ghCmdForExtension(extensionUpgradeArgs()...)
	firstCmd.Stdout = firstAttemptOut
	firstCmd.Stderr = firstAttemptOut
	firstErr = firstCmd.Run()
	if firstErr != nil {
		return false, buf, firstErr, nil
	}
	// First attempt succeeded without any file manipulation.
	if needsRenameWorkaround() {
		// Replay the buffered output that was not shown during the attempt.
		_, _ = io.Copy(os.Stderr, &buf)
	}
	installedVersion, versionErr := installedExtensionVersion()
	if versionErr != nil {
		return false, buf, nil, fmt.Errorf("failed to verify gh-aw extension version after upgrade: %w", versionErr)
	}
	if normalizeVersion(installedVersion) != normalizeVersion(latestVersion) {
		updateExtensionCheckLog.Printf("First upgrade attempt reported success but installed version is %s (expected %s)", renderReleaseVersion(installedVersion), renderReleaseVersion(latestVersion))
		mismatchErr := fmt.Errorf("failed to upgrade gh-aw extension: expected %s, got %s", renderReleaseVersion(latestVersion), renderReleaseVersion(installedVersion))
		if !needsRenameWorkaround() {
			return false, buf, nil, mismatchErr
		}
		firstErr = fmt.Errorf("gh extension upgrade reported success without installing target version: %w", mismatchErr)
		return false, buf, firstErr, nil
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension upgraded to "+renderReleaseVersion(latestVersion)))
	return true, buf, nil, nil
}

// resolveExecutableForRename resolves the current executable path and renames
// it to a backup location to free the path for the new binary.
// Returns (installPath, backupPath); either may be empty on failure.
func resolveExecutableForRename() (installPath, backupPath string) {
	exe, exeErr := os.Executable()
	if exeErr != nil {
		return
	}
	if resolved, resolveErr := filepath.EvalSymlinks(exe); resolveErr == nil {
		exe = resolved
	}
	iPath, bPath, renameErr := renamePathForUpgrade(exe)
	if renameErr != nil {
		// Rename failed; the retry will likely fail again.
		updateExtensionCheckLog.Printf("Could not rename executable for retry (upgrade will likely fail): %v", renameErr)
		return
	}
	installPath = iPath
	backupPath = bPath
	if runtime.GOOS != "windows" {
		return
	}
	// On Windows move the backup outside the extension directory so that
	// gh extension remove can succeed.
	extDir := filepath.Dir(backupPath)
	moved := false
	// Attempt 1: OS temp directory
	tmpBackup := filepath.Join(os.TempDir(), filepath.Base(backupPath))
	if moveErr := os.Rename(backupPath, tmpBackup); moveErr == nil {
		updateExtensionCheckLog.Printf("Moved Windows backup %s -> %s to free extension directory for removal", backupPath, tmpBackup)
		backupPath = tmpBackup
		moved = true
	} else {
		updateExtensionCheckLog.Printf("Could not move backup to %s (cross-drive?): %v; trying same-drive fallback", tmpBackup, moveErr)
	}
	// Attempt 2: parent of the extension directory (same drive as backup)
	if !moved {
		sameDriveDir := filepath.Dir(extDir)
		sameDriveTmp := filepath.Join(sameDriveDir, filepath.Base(backupPath))
		if moveErr2 := os.Rename(backupPath, sameDriveTmp); moveErr2 == nil {
			updateExtensionCheckLog.Printf("Moved Windows backup %s -> %s (same-drive fallback) to free extension directory for removal", backupPath, sameDriveTmp)
			backupPath = sameDriveTmp
		} else {
			updateExtensionCheckLog.Printf("Could not move backup out of extension directory (gh extension remove may fail): %v", moveErr2)
		}
	}
	// Remove stale .bak files left by previous attempts; these may be briefly
	// locked by Windows Defender.
	cleanupStaleWindowsBackups(extDir, backupPath)
	return
}

// performRenameRetryInstall retries the extension install via remove+pin after
// the initial upgrade attempt failed due to a locked binary.
// Returns (installPath, error).
func performRenameRetryInstall(latestVersion, installPath, backupPath string, firstAttemptBuf bytes.Buffer, _ error) (string, error) {
	// Remove the extension first so that "gh extension install" does not take
	// the "already installed" path and skip the actual write.
	removeCmd := ghCmdForExtension("extension", "remove", extensionRepo)
	removeCmd.Stdout = os.Stderr
	removeCmd.Stderr = os.Stderr
	if removeErr := removeCmd.Run(); removeErr == nil {
		// Extension directory has been deleted.
		backupPath = ""
	} else {
		updateExtensionCheckLog.Printf("Could not remove extension before reinstall (will attempt install anyway): %v", removeErr)
	}

	retryCmd := ghCmdForExtension("extension", "install", extensionRepo, "--pin", latestVersion)
	retryCmd.Stdout = os.Stderr
	retryCmd.Stderr = os.Stderr
	if retryErr := retryCmd.Run(); retryErr != nil {
		// Retry also failed. Restore the backup so the user still has gh-aw.
		if backupPath != "" {
			restoreExecutableBackup(installPath, backupPath)
		}
		if runtime.GOOS == "windows" && isWindowsLockError(firstAttemptBuf.String(), retryErr) {
			printWindowsLockUpgradeHelp(latestVersion)
		}
		return "", fmt.Errorf("failed to upgrade gh-aw extension: %w", retryErr)
	}

	_, versionErr := verifyInstalledVersion(latestVersion)
	if versionErr != nil {
		return "", fmt.Errorf("failed to verify gh-aw extension version after upgrade: %w", versionErr)
	}

	// Verification passed; clean up the backup.
	if backupPath != "" {
		cleanupExecutableBackup(backupPath)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension upgraded to "+renderReleaseVersion(latestVersion)))
	return installPath, nil
}

// printWindowsLockUpgradeHelp prints guidance for manual upgrade when the
// running binary is locked on Windows.
func printWindowsLockUpgradeHelp(latestVersion string) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("On Windows, gh-aw cannot self-upgrade while it is running."))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please upgrade manually by running one of the following:"))
	fmt.Fprintln(os.Stderr, "  "+extensionUpgradeHelpCommand(latestVersion))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("If that does not work, try reinstalling:"))
	fmt.Fprintln(os.Stderr, "  gh extension remove gh-aw")
	fmt.Fprintln(os.Stderr, "  "+extensionInstallHelpCommand(latestVersion))
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

// cleanupStaleWindowsBackups attempts to remove any .bak files left in extDir
// by previous upgrade attempts — either by our own code or the gh CLI's own
// rename mechanism.  The file at ownBackup (our currently-active backup for
// this upgrade attempt) is excluded so we do not remove our own relocated file.
//
// Retries with short delays to handle transient locks from antivirus scanners
// (e.g. Windows Defender) that may briefly hold the file open after a process
// exits.  The function is best-effort: if a file cannot be removed it is
// logged and skipped; gh extension remove may still fail but the caller's
// existing error-handling path covers that case.
func cleanupStaleWindowsBackups(extDir string, ownBackup string) {
	entries, err := os.ReadDir(extDir)
	if err != nil {
		updateExtensionCheckLog.Printf("Could not read extension directory for stale .bak cleanup: %v", err)
		return
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}
		bakFile := filepath.Join(extDir, entry.Name())
		if bakFile == ownBackup {
			continue // do not remove our own active backup
		}
		for attempt := range maxBackupCleanupAttempts {
			if removeErr := os.Remove(bakFile); removeErr == nil {
				updateExtensionCheckLog.Printf("Removed stale .bak file: %s", bakFile)
				break
			} else if attempt < maxBackupCleanupAttempts-1 {
				updateExtensionCheckLog.Printf("Could not remove stale .bak file %s (attempt %d/%d, retrying in %v): %v",
					bakFile, attempt+1, maxBackupCleanupAttempts, backupCleanupRetryDelay, removeErr)
				time.Sleep(backupCleanupRetryDelay)
			} else {
				updateExtensionCheckLog.Printf("Could not remove stale .bak file %s after %d attempts (gh extension remove may fail): %v",
					bakFile, maxBackupCleanupAttempts, removeErr)
			}
		}
	}
}

// isWindowsLockError reports whether the output or error from an upgrade
// attempt indicate a Windows file-locking issue (the running-binary-lock
// symptom).  Only when a lock error is detected should the Windows-specific
// self-upgrade guidance be shown; other failures should propagate the
// underlying error message instead.
func isWindowsLockError(output string, err error) bool {
	lockMsgs := []string{
		"Access is denied",
		"The process cannot access the file",
		// The gh CLI prints this when it finds a stale .bak file it cannot
		// remove, which is a symptom of the same locked-binary problem.
		"failed to remove previous extension update state",
	}
	for _, msg := range lockMsgs {
		if strings.Contains(output, msg) {
			return true
		}
		//nolint:errstringmatch // gh extension upgrade reports Windows locked-binary failures only via stderr text fragments.
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
	return []string{"extension", "upgrade", extensionRepo, "--force"}
}

// ghCmdForExtension creates an exec.Cmd for a gh CLI invocation that must
// target github.com.  gh-aw is only published to github.com; in mixed-host
// environments where GH_HOST points at a GHE instance, the default gh
// commands would hit the wrong host and report "already up to date" or fail.
// Pinning GH_HOST=github.com in the child process environment prevents that.
func ghCmdForExtension(args ...string) *exec.Cmd {
	cmd := exec.Command("gh", args...)
	// Inherit the full environment so that PATH, HOME, etc. remain intact,
	// then override (or add) GH_HOST to ensure github.com is always used.
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GH_HOST=") {
			env = append(env, e)
		}
	}
	env = append(env, "GH_HOST=github.com")
	cmd.Env = env
	return cmd
}

func prereleaseChannelNotice(currentVersion, latestStable string, includePrereleases bool) []string {
	if includePrereleases || latestStable == "" || !isPrereleaseVersion(currentVersion) {
		return nil
	}
	return []string{
		fmt.Sprintf("Current gh-aw version %s is newer than the latest stable release %s.", renderReleaseVersion(currentVersion), renderReleaseVersion(latestStable)),
		"Run `gh aw upgrade --pre-releases` to check for newer pre-releases.",
	}
}

func renderReleaseVersion(version string) string {
	if isPrereleaseVersion(version) {
		return version + " (pre-release)"
	}
	return version
}

func extensionUpgradeHelpCommand(targetVersion string) string {
	if isPrereleaseVersion(targetVersion) {
		return "gh extension install " + extensionRepo + " --force --pin " + targetVersion
	}
	return "gh " + strings.Join(extensionUpgradeArgs(), " ")
}

func extensionInstallHelpCommand(targetVersion string) string {
	if isPrereleaseVersion(targetVersion) {
		return "gh extension install " + extensionRepo + " --force --pin " + targetVersion
	}
	return "gh extension install " + extensionRepo
}

func installedExtensionVersion() (string, error) {
	var out bytes.Buffer
	cmd := ghCmdForExtension("aw", "version")
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to query installed gh-aw version: %w (output: %s)", err, summarizeCommandOutput(out.String()))
	}
	version, err := parseInstalledVersionOutput(out.String())
	if err != nil {
		return "", err
	}
	return version, nil
}

func verifyInstalledVersion(targetVersion string) (string, error) {
	installedVersion, err := installedExtensionVersion()
	if err != nil {
		return "", err
	}
	if normalizeVersion(installedVersion) != normalizeVersion(targetVersion) {
		return installedVersion, fmt.Errorf("failed to upgrade gh-aw extension: expected %s, got %s", renderReleaseVersion(targetVersion), renderReleaseVersion(installedVersion))
	}
	return installedVersion, nil
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

// parseInstalledVersionOutput scans the output of `gh aw version` for the
// first token that is a valid semantic version (with or without a leading "v").
// It returns the version normalized to include a "v" prefix.
func parseInstalledVersionOutput(output string) (string, error) {
	for token := range strings.FieldsSeq(output) {
		if semverutil.IsValid(token) {
			return semverutil.EnsureVPrefix(token), nil
		}
	}
	return "", fmt.Errorf("could not parse installed gh-aw version from output: %s", summarizeCommandOutput(output))
}

func summarizeCommandOutput(output string) string {
	const maxOutputLen = 300
	trimmed := strings.TrimSpace(output)
	if len(trimmed) <= maxOutputLen {
		return trimmed
	}
	return trimmed[:maxOutputLen] + "…"
}

func isPrereleaseVersion(version string) bool {
	version = "v" + strings.TrimPrefix(version, "v")
	return semver.IsValid(version) && semver.Prerelease(version) != ""
}

// extensionRepo is the GitHub repo slug used in all gh-extension CLI invocations.
const extensionRepo = "github/gh-aw"
