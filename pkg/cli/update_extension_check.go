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
func upgradeExtensionIfOutdated(verbose bool, includePrereleases bool) (bool, string, error) {
	currentVersion := GetVersion()
	updateExtensionCheckLog.Printf("Checking if extension needs upgrade (current: %s)", currentVersion)

	latestVersion, ok, err := upgradeExtensionIfOutdatedLatest(currentVersion, verbose, includePrereleases)
	if err != nil || !ok {
		return false, "", err
	}
	if upgradeExtensionIfOutdatedAlreadyCurrent(currentVersion, latestVersion, verbose, includePrereleases) {
		return false, "", nil
	}

	upgradeExtensionIfOutdatedAnnounce(currentVersion, latestVersion)
	if includePrereleases && !needsRenameWorkaround() {
		return upgradeExtensionIfOutdatedInstallPinned(latestVersion)
	}

	firstErr, firstAttemptBuf, upgraded, err := upgradeExtensionIfOutdatedFirstAttempt(latestVersion)
	if err != nil || upgraded {
		return upgraded, "", err
	}
	if !needsRenameWorkaround() {
		return false, "", fmt.Errorf("failed to upgrade gh-aw extension: %w", firstErr)
	}
	return upgradeExtensionIfOutdatedRetryWithRename(latestVersion, firstAttemptBuf)
}

func upgradeExtensionIfOutdatedLatest(currentVersion string, verbose bool, includePrereleases bool) (string, bool, error) {
	if !workflow.IsReleasedVersion(currentVersion) {
		updateExtensionCheckLog.Print("Not a released version, skipping upgrade check")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping extension upgrade check (development build)"))
		}
		return "", false, nil
	}
	latestVersion, err := getLatestRelease(includePrereleases)
	if err != nil {
		updateExtensionCheckLog.Printf("Failed to check for latest release (silently ignoring): %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not check for extension updates: %v", err)))
		}
		return "", false, nil
	}
	if latestVersion == "" {
		updateExtensionCheckLog.Print("Could not determine latest version, skipping upgrade")
		return "", false, nil
	}
	updateExtensionCheckLog.Printf("Latest version: %s", latestVersion)
	return latestVersion, true, nil
}

func upgradeExtensionIfOutdatedAlreadyCurrent(currentVersion, latestVersion string, verbose bool, includePrereleases bool) bool {
	currentSV := "v" + strings.TrimPrefix(currentVersion, "v")
	latestSV := "v" + strings.TrimPrefix(latestVersion, "v")
	if !semver.IsValid(currentSV) || !semver.IsValid(latestSV) {
		updateExtensionCheckLog.Printf("Non-semver versions detected (current=%q, latest=%q); proceeding with upgrade", currentVersion, latestVersion)
		return false
	}
	if semver.Compare(currentSV, latestSV) < 0 {
		return false
	}
	updateExtensionCheckLog.Print("Extension is already up to date")
	if notice := prereleaseChannelNotice(currentVersion, latestVersion, includePrereleases); len(notice) > 0 {
		for _, line := range notice {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(line))
		}
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension is up to date"))
	}
	return true
}

func upgradeExtensionIfOutdatedAnnounce(currentVersion, latestVersion string) {
	updateExtensionCheckLog.Printf("Upgrading extension from %s to %s", currentVersion, latestVersion)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Upgrading gh-aw extension from %s to %s...", renderReleaseVersion(currentVersion), renderReleaseVersion(latestVersion))))
}

func upgradeExtensionIfOutdatedInstallPinned(latestVersion string) (bool, string, error) {
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

func upgradeExtensionIfOutdatedFirstAttempt(latestVersion string) (error, bytes.Buffer, bool, error) {
	var firstAttemptBuf bytes.Buffer
	firstAttemptOut := firstAttemptWriter(os.Stderr, &firstAttemptBuf)
	firstCmd := ghCmdForExtension(extensionUpgradeArgs()...)
	firstCmd.Stdout = firstAttemptOut
	firstCmd.Stderr = firstAttemptOut
	firstErr := firstCmd.Run()
	if firstErr != nil {
		return firstErr, firstAttemptBuf, false, nil
	}
	if needsRenameWorkaround() {
		_, _ = io.Copy(os.Stderr, &firstAttemptBuf)
	}
	installedVersion, versionErr := installedExtensionVersion()
	if versionErr != nil {
		return nil, firstAttemptBuf, false, fmt.Errorf("failed to verify gh-aw extension version after upgrade: %w", versionErr)
	}
	if normalizeVersion(installedVersion) != normalizeVersion(latestVersion) {
		updateExtensionCheckLog.Printf("First upgrade attempt reported success but installed version is %s (expected %s)", renderReleaseVersion(installedVersion), renderReleaseVersion(latestVersion))
		mismatchErr := fmt.Errorf("failed to upgrade gh-aw extension: expected %s, got %s", renderReleaseVersion(latestVersion), renderReleaseVersion(installedVersion))
		if !needsRenameWorkaround() {
			return nil, firstAttemptBuf, false, mismatchErr
		}
		firstErr = fmt.Errorf("gh extension upgrade reported success without installing target version: %w", mismatchErr)
		return firstErr, firstAttemptBuf, false, nil
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension upgraded to "+renderReleaseVersion(latestVersion)))
	return nil, firstAttemptBuf, true, nil
}

func upgradeExtensionIfOutdatedRetryWithRename(latestVersion string, firstAttemptBuf bytes.Buffer) (bool, string, error) {
	updateExtensionCheckLog.Printf("First upgrade attempt failed (likely locked binary); retrying with rename workaround. First attempt output: %s", firstAttemptBuf.String())
	installPath, backupPath := upgradeExtensionIfOutdatedRenameCurrentExecutable()
	if err := upgradeExtensionIfOutdatedRemoveForRetry(&backupPath); err != nil {
		updateExtensionCheckLog.Print(err)
	}
	if err := upgradeExtensionIfOutdatedRunRetry(latestVersion, installPath, backupPath, firstAttemptBuf); err != nil {
		return false, "", err
	}
	_, versionErr := verifyInstalledVersion(latestVersion)
	if versionErr != nil {
		return false, "", fmt.Errorf("failed to verify gh-aw extension version after upgrade: %w", versionErr)
	}
	if backupPath != "" {
		cleanupExecutableBackup(backupPath)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension upgraded to "+renderReleaseVersion(latestVersion)))
	return true, installPath, nil
}

func upgradeExtensionIfOutdatedRenameCurrentExecutable() (string, string) {
	var installPath string
	var backupPath string
	if exe, exeErr := os.Executable(); exeErr == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(exe); resolveErr == nil {
			exe = resolved
		}
		if iPath, bPath, renameErr := renamePathForUpgrade(exe); renameErr != nil {
			updateExtensionCheckLog.Printf("Could not rename executable for retry (upgrade will likely fail): %v", renameErr)
		} else {
			installPath = iPath
			backupPath = bPath
			backupPath = upgradeExtensionIfOutdatedMoveWindowsBackup(backupPath)
		}
	}
	return installPath, backupPath
}

func upgradeExtensionIfOutdatedMoveWindowsBackup(backupPath string) string {
	if runtime.GOOS != "windows" || backupPath == "" {
		return backupPath
	}
	extDir := filepath.Dir(backupPath)
	backupPath = upgradeExtensionIfOutdatedMoveWindowsBackupOut(backupPath, extDir)
	cleanupStaleWindowsBackups(extDir, backupPath)
	return backupPath
}

func upgradeExtensionIfOutdatedMoveWindowsBackupOut(backupPath, extDir string) string {
	tmpBackup := filepath.Join(os.TempDir(), filepath.Base(backupPath))
	if moveErr := os.Rename(backupPath, tmpBackup); moveErr == nil {
		updateExtensionCheckLog.Printf("Moved Windows backup %s -> %s to free extension directory for removal", backupPath, tmpBackup)
		return tmpBackup
	}
	sameDriveDir := filepath.Dir(extDir)
	sameDriveTmp := filepath.Join(sameDriveDir, filepath.Base(backupPath))
	if moveErr2 := os.Rename(backupPath, sameDriveTmp); moveErr2 == nil {
		updateExtensionCheckLog.Printf("Moved Windows backup %s -> %s (same-drive fallback) to free extension directory for removal", backupPath, sameDriveTmp)
		return sameDriveTmp
	}
	updateExtensionCheckLog.Printf("Could not move backup out of extension directory (gh extension remove may fail)")
	return backupPath
}

func upgradeExtensionIfOutdatedRemoveForRetry(backupPath *string) error {
	removeCmd := ghCmdForExtension("extension", "remove", extensionRepo)
	removeCmd.Stdout = os.Stderr
	removeCmd.Stderr = os.Stderr
	if removeErr := removeCmd.Run(); removeErr != nil {
		return fmt.Errorf("Could not remove extension before reinstall (will attempt install anyway): %w", removeErr)
	}
	*backupPath = ""
	return nil
}

func upgradeExtensionIfOutdatedRunRetry(latestVersion, installPath, backupPath string, firstAttemptBuf bytes.Buffer) error {
	retryCmd := ghCmdForExtension("extension", "install", extensionRepo, "--pin", latestVersion)
	retryCmd.Stdout = os.Stderr
	retryCmd.Stderr = os.Stderr
	if retryErr := retryCmd.Run(); retryErr != nil {
		if backupPath != "" {
			restoreExecutableBackup(installPath, backupPath)
		}
		if runtime.GOOS == "windows" && isWindowsLockError(firstAttemptBuf.String(), retryErr) {
			upgradeExtensionIfOutdatedPrintWindowsHelp(latestVersion)
		}
		return fmt.Errorf("failed to upgrade gh-aw extension: %w", retryErr)
	}
	return nil
}

func upgradeExtensionIfOutdatedPrintWindowsHelp(latestVersion string) {
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
