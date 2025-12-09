package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionRefLog = logger.New("workflow:action_reference")

const (
	// GitHubOrgRepo is the organization and repository name for custom action references
	GitHubOrgRepo = "githubnext/gh-aw"
)

// resolveActionReference converts a local action path to the appropriate reference
// based on the current action mode (dev vs release).
// For dev mode: returns the local path as-is (e.g., "./actions/create-issue")
// For release mode: converts to SHA-pinned remote reference (e.g., "githubnext/gh-aw/actions/create-issue@SHA # tag")
// For inline mode: returns empty string to fallback to inline mode
func (c *Compiler) resolveActionReference(localActionPath string, data *WorkflowData) string {
	switch c.actionMode {
	case ActionModeDev:
		// Return local path as-is for development
		actionRefLog.Printf("Dev mode: using local action path: %s", localActionPath)
		return localActionPath

	case ActionModeRelease:
		// Convert to SHA-pinned remote reference for release
		remoteRef := convertToRemoteActionRef(localActionPath)
		if remoteRef == "" {
			actionRefLog.Printf("WARNING: Could not resolve remote reference for %s", localActionPath)
			return ""
		}
		actionRefLog.Printf("Release mode: using remote action reference: %s", remoteRef)
		return remoteRef

	case ActionModeInline:
		// Return empty to fallback to inline mode
		actionRefLog.Print("Inline mode: returning empty to use inline JavaScript")
		return ""

	default:
		actionRefLog.Printf("WARNING: Unknown action mode %s, returning empty", c.actionMode)
		return ""
	}
}

// convertToRemoteActionRef converts a local action path to a tag-based remote reference
// that will be resolved to a SHA later in the release pipeline using action pins.
// Example: "./actions/create-issue" -> "githubnext/gh-aw/actions/create-issue@v1.0.0"
func convertToRemoteActionRef(localPath string) string {
	// Strip the leading "./" if present
	actionPath := strings.TrimPrefix(localPath, "./")

	// Get the current release tag
	tag := GetCurrentGitTag()
	if tag == "" {
		actionRefLog.Print("WARNING: No git tag available for release mode")
		return ""
	}

	// Construct the remote reference with tag: githubnext/gh-aw/actions/name@tag
	// The SHA will be resolved later by action pinning infrastructure
	remoteRef := fmt.Sprintf("%s/%s@%s", GitHubOrgRepo, actionPath, tag)
	actionRefLog.Printf("Using tag-based reference: %s (SHA will be resolved via action pins)", remoteRef)

	return remoteRef
}
