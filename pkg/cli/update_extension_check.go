package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var updateExtensionCheckLog = logger.New("cli:update_extension_check")

// ensureLatestExtensionVersion checks if the current release matches the latest release
// and issues a warning if an update is needed. This function fails silently if the
// release URL is not available or blocked.
func ensureLatestExtensionVersion(verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Checking for gh-aw extension updates..."))
	}

	// Get current version
	currentVersion := GetVersion()
	updateExtensionCheckLog.Printf("Current version: %s", currentVersion)

	// Skip check for non-release versions (dev builds)
	if !workflow.IsReleasedVersion(currentVersion) {
		updateExtensionCheckLog.Print("Not a released version, skipping update check")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping version check (development build)"))
		}
		return nil
	}

	// Query GitHub API for latest release
	latestVersion, err := getLatestRelease()
	if err != nil {
		// Fail silently - don't block upgrade if we can't check for updates
		updateExtensionCheckLog.Printf("Failed to check for updates (silently ignoring): %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not check for updates: %v", err)))
		}
		return nil
	}

	if latestVersion == "" {
		updateExtensionCheckLog.Print("Could not determine latest version")
		return nil
	}

	updateExtensionCheckLog.Printf("Latest version: %s", latestVersion)

	// Normalize versions for comparison (remove 'v' prefix)
	currentVersionNormalized := strings.TrimPrefix(currentVersion, "v")
	latestVersionNormalized := strings.TrimPrefix(latestVersion, "v")

	// Compare versions
	if currentVersionNormalized == latestVersionNormalized {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension is up to date"))
		}
		updateExtensionCheckLog.Print("Extension is up to date")
		return nil
	}

	// Check if we're on a newer version (development/prerelease)
	if currentVersionNormalized > latestVersionNormalized {
		updateExtensionCheckLog.Printf("Current version (%s) appears newer than latest release (%s)", currentVersion, latestVersion)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Running a development or pre-release version"))
		}
		return nil
	}

	// A newer version is available - display warning message (not error)
	updateExtensionCheckLog.Printf("Newer version available: %s (current: %s)", latestVersion, currentVersion)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("A newer version of gh-aw is available: %s (current: %s)", latestVersion, currentVersion)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Consider upgrading with: gh extension upgrade github/gh-aw"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// upgradeExtensionIfOutdated checks if a newer version of the gh-aw extension is available
// and, if so, upgrades it automatically. Returns true if an upgrade was performed.
//
// When true is returned the CURRENTLY RUNNING PROCESS still has the old version baked in.
// The caller must stop all further work that would embed version strings (e.g. lock-file
// compilation) and ask the user to re-run the command so the freshly-installed binary is
// used instead.
func upgradeExtensionIfOutdated(verbose bool) (bool, error) {
	currentVersion := GetVersion()
	updateExtensionCheckLog.Printf("Checking if extension needs upgrade (current: %s)", currentVersion)

	// Skip for non-release versions (dev builds)
	if !workflow.IsReleasedVersion(currentVersion) {
		updateExtensionCheckLog.Print("Not a released version, skipping upgrade check")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping extension upgrade check (development build)"))
		}
		return false, nil
	}

	// Query GitHub API for latest release
	latestVersion, err := getLatestRelease()
	if err != nil {
		// Fail silently - don't block the upgrade command if we can't reach GitHub
		updateExtensionCheckLog.Printf("Failed to check for latest release (silently ignoring): %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not check for extension updates: %v", err)))
		}
		return false, nil
	}

	if latestVersion == "" {
		updateExtensionCheckLog.Print("Could not determine latest version, skipping upgrade")
		return false, nil
	}

	updateExtensionCheckLog.Printf("Latest version: %s", latestVersion)

	// Ensure both versions have the 'v' prefix required by the semver package.
	currentSV := currentVersion
	if !strings.HasPrefix(currentSV, "v") {
		currentSV = "v" + currentSV
	}
	latestSV := latestVersion
	if !strings.HasPrefix(latestSV, "v") {
		latestSV = "v" + latestSV
	}

	// Already on the latest (or newer) version – use proper semver comparison so
	// that e.g. "0.10.0" is correctly treated as newer than "0.9.0".
	if semver.IsValid(currentSV) && semver.IsValid(latestSV) {
		if semver.Compare(currentSV, latestSV) >= 0 {
			updateExtensionCheckLog.Print("Extension is already up to date")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension is up to date"))
			}
			return false, nil
		}
	} else {
		// Fall back to normalised string comparison when versions are not valid semver.
		currentNorm := strings.TrimPrefix(currentVersion, "v")
		latestNorm := strings.TrimPrefix(latestVersion, "v")
		if currentNorm >= latestNorm {
			updateExtensionCheckLog.Print("Extension is already up to date (string comparison fallback)")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension is up to date"))
			}
			return false, nil
		}
	}

	// A newer version is available – upgrade automatically
	updateExtensionCheckLog.Printf("Upgrading extension from %s to %s", currentVersion, latestVersion)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Upgrading gh-aw extension from %s to %s...", currentVersion, latestVersion)))

	cmd := exec.Command("gh", "extension", "upgrade", "github/gh-aw")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to upgrade gh-aw extension: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension upgraded to "+latestVersion))
	return true, nil
}
