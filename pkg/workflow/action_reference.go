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

// ResolveSetupActionReference resolves the actions/setup action reference based on action mode and version.
// This is a standalone helper function that can be used by both Compiler methods and standalone
// workflow generators (like maintenance workflow) that don't have access to WorkflowData.
//
// Parameters:
//   - actionMode: The action mode (dev or release)
//   - version: The version string to use for release mode
//
// Returns:
//   - For dev mode: "./actions/setup" (local path)
//   - For release mode: "githubnext/gh-aw/actions/setup@<version>" (remote reference)
//   - Falls back to local path if version is invalid in release mode
func ResolveSetupActionReference(actionMode ActionMode, version string) string {
	localPath := "./actions/setup"

	// Dev mode - return local path
	if actionMode == ActionModeDev {
		actionRefLog.Printf("Dev mode: using local action path: %s", localPath)
		return localPath
	}

	// Release mode - convert to remote reference
	if actionMode == ActionModeRelease {
		actionPath := strings.TrimPrefix(localPath, "./")

		// Check if version is valid for release mode
		if version == "" || version == "dev" {
			actionRefLog.Print("WARNING: No release tag available in binary version (version is 'dev' or empty), falling back to local path")
			return localPath
		}

		// Construct the remote reference with tag: githubnext/gh-aw/actions/setup@tag
		// The SHA will be resolved later by action pinning infrastructure
		remoteRef := fmt.Sprintf("%s/%s@%s", GitHubOrgRepo, actionPath, version)
		actionRefLog.Printf("Release mode: using remote action reference: %s (SHA will be resolved via action pins)", remoteRef)
		return remoteRef
	}

	// Unknown mode - default to local path
	actionRefLog.Printf("WARNING: Unknown action mode %s, defaulting to local path", actionMode)
	return localPath
}

// resolveActionReference converts a local action path to the appropriate reference
// based on the current action mode (dev vs release).
// If action-tag is specified in features, it overrides the mode check and enables release mode behavior.
// For dev mode: returns the local path as-is (e.g., "./actions/create-issue")
// For release mode: converts to SHA-pinned remote reference (e.g., "githubnext/gh-aw/actions/create-issue@SHA # tag")
func (c *Compiler) resolveActionReference(localActionPath string, data *WorkflowData) string {
	// Check if action-tag is specified in features - if so, override mode and use release behavior
	hasActionTag := false
	if data != nil && data.Features != nil {
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				hasActionTag = true
				actionRefLog.Printf("action-tag feature detected: %s - using release mode behavior", actionTagStr)
			}
		}
	}

	// For ./actions/setup without action-tag override, use the shared helper
	if localActionPath == "./actions/setup" && !hasActionTag {
		return ResolveSetupActionReference(c.actionMode, c.version)
	}

	// Use release mode if either actionMode is release OR action-tag is specified
	if c.actionMode == ActionModeRelease || hasActionTag {
		// Convert to SHA-pinned remote reference for release
		remoteRef := c.convertToRemoteActionRef(localActionPath, data)
		if remoteRef == "" {
			actionRefLog.Printf("WARNING: Could not resolve remote reference for %s", localActionPath)
			return ""
		}
		if hasActionTag {
			actionRefLog.Printf("action-tag override: using remote action reference: %s", remoteRef)
		} else {
			actionRefLog.Printf("Release mode: using remote action reference: %s", remoteRef)
		}
		return remoteRef
	}

	// Dev mode - return local path
	if c.actionMode == ActionModeDev {
		actionRefLog.Printf("Dev mode: using local action path: %s", localActionPath)
		return localActionPath
	}

	// Default to dev mode for unknown modes
	actionRefLog.Printf("WARNING: Unknown action mode %s, defaulting to dev mode", c.actionMode)
	return localActionPath
}

// convertToRemoteActionRef converts a local action path to a tag-based remote reference
// that will be resolved to a SHA later in the release pipeline using action pins.
// Uses the action-tag from WorkflowData.Features if specified (for testing), otherwise uses the version stored in the compiler binary.
// Example: "./actions/create-issue" -> "githubnext/gh-aw/actions/create-issue@v1.0.0"
func (c *Compiler) convertToRemoteActionRef(localPath string, data *WorkflowData) string {
	// Strip the leading "./" if present
	actionPath := strings.TrimPrefix(localPath, "./")

	// Use action-tag from WorkflowData.Features if specified, otherwise fall back to compiler version
	var tag string
	if data != nil && data.Features != nil {
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				tag = actionTagStr
				actionRefLog.Printf("Using action-tag from features: %s", tag)
			}
		}
	}

	if tag == "" {
		tag = c.version
		if tag == "" || tag == "dev" {
			actionRefLog.Print("WARNING: No release tag available in binary version (version is 'dev' or empty)")
			return ""
		}
		actionRefLog.Printf("Using tag from binary version: %s", tag)
	}

	// Construct the remote reference with tag: githubnext/gh-aw/actions/name@tag
	// The SHA will be resolved later by action pinning infrastructure
	remoteRef := fmt.Sprintf("%s/%s@%s", GitHubOrgRepo, actionPath, tag)
	actionRefLog.Printf("Remote reference: %s (SHA will be resolved via action pins)", remoteRef)

	return remoteRef
}
