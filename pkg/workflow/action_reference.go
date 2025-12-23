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
func (c *Compiler) resolveActionReference(localActionPath string, data *WorkflowData) string {
	switch c.actionMode {
	case ActionModeDev:
		// Return local path as-is for development
		actionRefLog.Printf("Dev mode: using local action path: %s", localActionPath)
		return localActionPath

	case ActionModeRelease:
		// Convert to SHA-pinned remote reference for release
		remoteRef := c.convertToRemoteActionRef(localActionPath)
		if remoteRef == "" {
			actionRefLog.Printf("WARNING: Could not resolve remote reference for %s", localActionPath)
			return ""
		}
		actionRefLog.Printf("Release mode: using remote action reference: %s", remoteRef)
		return remoteRef

	default:
		// Default to dev mode for unknown modes
		actionRefLog.Printf("WARNING: Unknown action mode %s, defaulting to dev mode", c.actionMode)
		return localActionPath
	}
}

// convertToRemoteActionRef converts a local action path to a tag-based remote reference
// that will be resolved to a SHA later in the release pipeline using action pins.
// Uses the version stored in the compiler binary instead of querying git.
// Example: "./actions/create-issue" -> "githubnext/gh-aw/actions/create-issue@v1.0.0"
func (c *Compiler) convertToRemoteActionRef(localPath string) string {
	// Strip the leading "./" if present
	actionPath := strings.TrimPrefix(localPath, "./")

	// Use the version from the compiler binary
	tag := c.version
	if tag == "" || tag == "dev" {
		actionRefLog.Print("WARNING: No release tag available in binary version (version is 'dev' or empty)")
		return ""
	}

	// Construct the remote reference with tag: githubnext/gh-aw/actions/name@tag
	// The SHA will be resolved later by action pinning infrastructure
	remoteRef := fmt.Sprintf("%s/%s@%s", GitHubOrgRepo, actionPath, tag)
	actionRefLog.Printf("Using tag from binary version: %s", tag)
	actionRefLog.Printf("Remote reference: %s (SHA will be resolved via action pins)", remoteRef)

	return remoteRef
}
