package workflow

import (
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionModeLog = logger.New("workflow:action_mode")

// ActionMode defines how JavaScript is embedded in workflow steps
type ActionMode string

const (
	// ActionModeDev references custom actions using local paths (development mode, default)
	ActionModeDev ActionMode = "dev"

	// ActionModeRelease references custom actions using SHA-pinned remote paths (release mode)
	ActionModeRelease ActionMode = "release"
)

// String returns the string representation of the action mode
func (m ActionMode) String() string {
	return string(m)
}

// IsValid checks if the action mode is valid
func (m ActionMode) IsValid() bool {
	return m == ActionModeDev || m == ActionModeRelease
}

// IsDev returns true if the action mode is development mode
func (m ActionMode) IsDev() bool {
	return m == ActionModeDev
}

// IsRelease returns true if the action mode is release mode
func (m ActionMode) IsRelease() bool {
	return m == ActionModeRelease
}

// UsesExternalActions returns true (always true since inline mode was removed)
func (m ActionMode) UsesExternalActions() bool {
	return true
}

// DetectActionMode determines the appropriate action mode based on environment and version.
// Returns ActionModeRelease if the binary version is a release tag,
// otherwise returns ActionModeDev as the default.
// Can be overridden with GH_AW_ACTION_MODE environment variable.
// If version parameter is provided, it will be used to determine release mode.
// Never uses dirty SHA - only clean version tags or dev mode with local paths.
func DetectActionMode(version string) ActionMode {
	actionModeLog.Printf("Detecting action mode: version=%s", version)

	// Check for explicit override via environment variable
	if envMode := os.Getenv("GH_AW_ACTION_MODE"); envMode != "" {
		mode := ActionMode(envMode)
		if mode.IsValid() {
			actionModeLog.Printf("Using action mode from environment override: %s", mode)
			return mode
		}
		actionModeLog.Printf("Invalid action mode in environment: %s, falling back to auto-detection", envMode)
	}

	// Check if version indicates a release build (not "dev" and not empty)
	if version != "" && version != "dev" {
		// Version is a release tag, use release mode
		actionModeLog.Printf("Detected release mode from binary version: %s", version)
		return ActionModeRelease
	}

	// Check GitHub Actions context for additional hints
	githubRef := os.Getenv("GITHUB_REF")
	githubEventName := os.Getenv("GITHUB_EVENT_NAME")
	actionModeLog.Printf("GitHub context: ref=%s, event=%s", githubRef, githubEventName)

	// Release mode conditions from GitHub Actions context:
	// 1. Running on a release branch (refs/heads/release*)
	// 2. Running on a release tag (refs/tags/*)
	// 3. Running on a release event
	if strings.HasPrefix(githubRef, "refs/heads/release") ||
		strings.HasPrefix(githubRef, "refs/tags/") ||
		githubEventName == "release" {
		actionModeLog.Printf("Detected release mode from GitHub context: ref=%s, event=%s", githubRef, githubEventName)
		return ActionModeRelease
	}

	// Default to dev mode for all other cases:
	// 1. Running on a PR (refs/pull/*)
	// 2. Running locally (no GITHUB_REF)
	// 3. Running on any other branch (including main)
	// 4. Version is "dev" or empty
	actionModeLog.Printf("Detected dev mode (default): version=%s, ref=%s", version, githubRef)
	return ActionModeDev
}
