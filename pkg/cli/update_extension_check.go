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

// upgradeExtensionIfOutdated checks if a newer version of the gh-aw extension is available
// and, if so, upgrades it automatically. Returns true if an upgrade was performed.
//
// When true is returned the CURRENTLY RUNNING PROCESS still has the old version baked in.
// The caller should re-launch the freshly-installed binary so that subsequent work
// (e.g. lock-file compilation) uses the correct new version string.
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
			return false, nil
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

	cmd := exec.Command("gh", "extension", "upgrade", "github/gh-aw")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to upgrade gh-aw extension: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ gh-aw extension upgraded to "+latestVersion))
	return true, nil
}
