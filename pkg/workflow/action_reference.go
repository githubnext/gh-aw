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

// convertToRemoteActionRef converts a local action path to a SHA-pinned remote reference
// with optional tag comment.
// Example: "./actions/create-issue" -> "githubnext/gh-aw/actions/create-issue@abc123... # v1.0.0"
func convertToRemoteActionRef(localPath string) string {
	// Strip the leading "./" if present
	actionPath := strings.TrimPrefix(localPath, "./")

	// Determine the commit SHA to use
	sha := GetCurrentCommitSHA()
	if sha == "" {
		actionRefLog.Print("WARNING: Could not determine current commit SHA")
		return ""
	}

	// Construct the remote reference: githubnext/gh-aw/actions/name@SHA
	remoteRef := fmt.Sprintf("%s/%s@%s", GitHubOrgRepo, actionPath, sha)

	// Add tag comment if available
	if tag := GetCurrentGitTag(); tag != "" {
		remoteRef = fmt.Sprintf("%s # %s", remoteRef, tag)
		actionRefLog.Printf("Added tag comment to reference: %s", tag)
	}

	return remoteRef
}
