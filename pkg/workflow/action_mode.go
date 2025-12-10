package workflow

import (
	"os"
	"strings"
)

// ActionMode defines how JavaScript is embedded in workflow steps
type ActionMode string

const (
	// ActionModeInline embeds JavaScript inline using actions/github-script (current behavior)
	ActionModeInline ActionMode = "inline"

	// ActionModeDev references custom actions using local paths (development mode)
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
	return m == ActionModeInline || m == ActionModeDev || m == ActionModeRelease
}

// DetectActionMode determines the appropriate action mode based on environment
// Returns ActionModeRelease if running from main branch or release tag,
// ActionModeDev for PR/local development, or ActionModeInline as fallback.
// Can be overridden with GH_AW_ACTION_MODE environment variable.
func DetectActionMode() ActionMode {
	// Check for explicit override via environment variable
	if envMode := os.Getenv("GH_AW_ACTION_MODE"); envMode != "" {
		mode := ActionMode(envMode)
		if mode.IsValid() {
			return mode
		}
	}

	// Check GitHub Actions context
	githubRef := os.Getenv("GITHUB_REF")
	githubEventName := os.Getenv("GITHUB_EVENT_NAME")

	// Release mode conditions:
	// 1. Running on a release branch (refs/heads/release*)
	// 2. Running on a release tag (refs/tags/*)
	// 3. Running on a release event
	if strings.HasPrefix(githubRef, "refs/heads/release") ||
		strings.HasPrefix(githubRef, "refs/tags/") ||
		githubEventName == "release" {
		return ActionModeRelease
	}

	// Dev mode conditions:
	// 1. Running on a PR (refs/pull/*)
	// 2. Running locally (no GITHUB_REF)
	// 3. Running on any other branch (including main)
	if strings.HasPrefix(githubRef, "refs/pull/") ||
		githubRef == "" ||
		strings.HasPrefix(githubRef, "refs/heads/") {
		return ActionModeDev
	}

	// Fallback to inline mode for backwards compatibility
	return ActionModeInline
}
