package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var actionRefLog = logger.New("workflow:action_reference")

const (
	// GitHubOrgRepo is the organization and repository name for custom action references
	GitHubOrgRepo = "github/gh-aw"

	// GitHubActionsOrgRepo is the organization and repository name for the external gh-aw-actions repository
	GitHubActionsOrgRepo = "github/gh-aw-actions"
)

// ResolveSetupActionReference resolves the actions/setup action reference based on action mode and version.
// This is a standalone helper function that can be used by both Compiler methods and standalone
// workflow generators (like maintenance workflow) that don't have access to WorkflowData.
//
// Parameters:
//   - actionMode: The action mode (dev, release, or action)
//   - version: The version string to use for release/action mode
//   - actionTag: Optional override tag/SHA (takes precedence over version when in release mode)
//   - resolver: Optional ActionSHAResolver for dynamic SHA resolution (can be nil for standalone use)
//
// Returns:
//   - For dev mode: "./actions/setup" (local path)
//   - For release mode with resolver: "github/gh-aw/actions/setup@<sha> # <version>" (SHA-pinned)
//   - For release mode without resolver: "github/gh-aw/actions/setup@<version>" (tag-based, SHA resolved later)
//   - For action mode with resolver: "github/gh-aw-actions/setup@<sha> # <version>" (SHA-pinned)
//   - For action mode without resolver: "github/gh-aw-actions/setup@<version>" (tag-based, SHA resolved later)
//   - Falls back to local path if version is invalid in release/action mode
func ResolveSetupActionReference(actionMode ActionMode, version string, actionTag string, resolver ActionSHAResolver) string {
	return resolveSetupActionRef(actionMode, version, actionTag, resolver, "")
}

// resolveSetupActionRef is the internal implementation of ResolveSetupActionReference
// that accepts an optional actionsOrgRepo override. When actionsOrgRepo is empty,
// GitHubActionsOrgRepo is used.
func resolveSetupActionRef(actionMode ActionMode, version string, actionTag string, resolver ActionSHAResolver, actionsOrgRepo string) string {
	if actionsOrgRepo == "" {
		actionsOrgRepo = GitHubActionsOrgRepo
	}

	localPath := "./actions/setup"

	// Dev mode - return local path
	if actionMode == ActionModeDev {
		actionRefLog.Printf("Dev mode: using local action path: %s", localPath)
		return localPath
	}

	// Action mode - use external gh-aw-actions repository with SHA pinning if possible
	if actionMode == ActionModeAction {
		// Use actionTag if provided, otherwise fall back to version
		tag := actionTag
		if tag == "" {
			tag = version
		}

		// Check if tag is valid for action mode
		if tag == "" || tag == "dev" {
			actionRefLog.Print("WARNING: No release tag available in binary version (version is 'dev' or empty), falling back to local path")
			return localPath
		}

		// Construct the remote reference: <actionsOrgRepo>/setup@tag
		actionRepo := actionsOrgRepo + "/setup"
		remoteRef := fmt.Sprintf("%s@%s", actionRepo, tag)

		// If a resolver is available, try to resolve the SHA
		if resolver != nil {
			sha, err := resolver.ResolveSHA(actionRepo, tag)
			if err == nil && sha != "" {
				pinnedRef := formatActionReference(actionRepo, sha, tag)
				actionRefLog.Printf("Action mode: resolved %s to SHA-pinned reference: %s", remoteRef, pinnedRef)
				return pinnedRef
			}
			if err != nil {
				actionRefLog.Printf("Failed to resolve SHA for %s@%s: %v", actionRepo, tag, err)
			}
		}

		// If no resolver or SHA resolution failed, return tag-based reference
		actionRefLog.Printf("Action mode: using tag-based external actions repo reference: %s (SHA will be resolved later)", remoteRef)
		return remoteRef
	}

	// Release mode - convert to remote reference
	if actionMode == ActionModeRelease {
		actionPath := strings.TrimPrefix(localPath, "./")

		// Use actionTag if provided, otherwise fall back to version
		tag := actionTag
		if tag == "" {
			tag = version
		}

		// Check if tag is valid for release mode
		if tag == "" || tag == "dev" {
			actionRefLog.Print("WARNING: No release tag available in binary version (version is 'dev' or empty), falling back to local path")
			return localPath
		}

		// Construct the remote reference with tag: github/gh-aw/actions/setup@tag
		actionRepo := fmt.Sprintf("%s/%s", GitHubOrgRepo, actionPath)
		remoteRef := fmt.Sprintf("%s@%s", actionRepo, tag)

		// If a resolver is available, try to resolve the SHA
		if resolver != nil {
			sha, err := resolver.ResolveSHA(actionRepo, tag)
			if err == nil && sha != "" {
				pinnedRef := formatActionReference(actionRepo, sha, tag)
				actionRefLog.Printf("Release mode: resolved %s to SHA-pinned reference: %s", remoteRef, pinnedRef)
				return pinnedRef
			}
			if err != nil {
				actionRefLog.Printf("Failed to resolve SHA for %s@%s: %v", actionRepo, tag, err)
			}
		}

		// If no resolver or SHA resolution failed, return tag-based reference
		// This is for backward compatibility with standalone workflow generators
		actionRefLog.Printf("Release mode: using tag-based remote action reference: %s (SHA will be resolved later)", remoteRef)
		return remoteRef
	}

	// Unknown mode - default to local path
	actionRefLog.Printf("WARNING: Unknown action mode %s, defaulting to local path", actionMode)
	return localPath
}

// resolveActionReference converts a local action path to the appropriate reference
// based on the current action mode (dev vs release vs action).
// If action-tag is specified in features, it overrides the mode check and enables action mode behavior
// (using the github/gh-aw-actions external repository).
// For dev mode: returns the local path as-is (e.g., "./actions/create-issue")
// For release mode: converts to SHA-pinned remote reference (e.g., "github/gh-aw/actions/create-issue@SHA # tag")
// For action mode: converts to SHA-pinned reference in external repo if possible (e.g., "github/gh-aw-actions/create-issue@SHA # version")
func (c *Compiler) resolveActionReference(localActionPath string, data *WorkflowData) string {
	// Check if action-tag is specified in features - if so, override mode and use action mode behavior
	hasActionTag := false
	var frontmatterActionTag string
	if data != nil && data.Features != nil {
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				hasActionTag = true
				frontmatterActionTag = actionTagStr
				actionRefLog.Printf("action-tag feature detected: %s - using action mode behavior", actionTagStr)
			}
		}
	}

	// For ./actions/setup, check for compiler-level actionTag override first
	if localActionPath == "./actions/setup" {
		// Use compiler actionTag if available, otherwise check features
		var resolver ActionSHAResolver
		if data != nil && data.ActionResolver != nil {
			resolver = data.ActionResolver
		}
		if c.actionTag != "" {
			return resolveSetupActionRef(c.actionMode, c.version, c.actionTag, resolver, c.effectiveActionsRepo())
		}
		if !hasActionTag {
			return resolveSetupActionRef(c.actionMode, c.version, "", resolver, c.effectiveActionsRepo())
		}
		// hasActionTag is true and no compiler actionTag: use action mode with the frontmatter tag
		return resolveSetupActionRef(ActionModeAction, c.version, frontmatterActionTag, resolver, c.effectiveActionsRepo())
	}

	// Action mode - use external gh-aw-actions repository
	if c.actionMode == ActionModeAction || hasActionTag {
		return c.convertToExternalActionsRef(localActionPath, data)
	}

	// Use release mode
	if c.actionMode == ActionModeRelease {
		// Convert to tag-based remote reference for release
		remoteRef := c.convertToRemoteActionRef(localActionPath, data)
		if remoteRef == "" {
			actionRefLog.Printf("WARNING: Could not resolve remote reference for %s", localActionPath)
			return ""
		}

		// Now resolve the tag to a SHA using action pins
		// Extract repo and version from the remote reference (format: "repo/path@version")
		actionRepo := extractActionRepo(remoteRef)
		version := extractActionVersion(remoteRef)

		if actionRepo != "" && version != "" {
			// Resolve the SHA using action pins
			pinnedRef, err := getActionPinWithData(actionRepo, version, data)
			if err != nil {
				// In strict mode, getActionPinWithData returns an error
				actionRefLog.Printf("Failed to pin action %s@%s: %v", actionRepo, version, err)
				return ""
			}
			if pinnedRef != "" {
				// Successfully resolved to SHA
				actionRefLog.Printf("Release mode: resolved %s to SHA-pinned reference: %s", remoteRef, pinnedRef)
				return pinnedRef
			}
		}

		// If we couldn't resolve to SHA, return the tag-based reference
		// This happens in non-strict mode when no pin is available
		actionRefLog.Printf("Release mode: using tag-based remote action reference: %s", remoteRef)
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
// If compiler has actionTag set, it takes priority over both.
// Example: "./actions/create-issue" -> "github/gh-aw/actions/create-issue@v1.0.0"
func (c *Compiler) convertToRemoteActionRef(localPath string, data *WorkflowData) string {
	// Strip the leading "./" if present
	actionPath := strings.TrimPrefix(localPath, "./")

	// Priority order for tag selection:
	// 1. Compiler actionTag (from --action-tag flag)
	// 2. WorkflowData.Features["action-tag"] (from frontmatter)
	// 3. Compiler version
	var tag string

	// Check compiler actionTag first (highest priority)
	if c.actionTag != "" {
		tag = c.actionTag
		actionRefLog.Printf("Using action-tag from compiler: %s", tag)
	} else if data != nil && data.Features != nil {
		// Check WorkflowData.Features for action-tag
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				tag = actionTagStr
				actionRefLog.Printf("Using action-tag from features: %s", tag)
			}
		}
	}

	// Fall back to compiler version if no tag specified
	if tag == "" {
		tag = c.version
		if tag == "" || tag == "dev" {
			actionRefLog.Print("WARNING: No release tag available in binary version (version is 'dev' or empty)")
			return ""
		}
		actionRefLog.Printf("Using tag from binary version: %s", tag)
	}

	// Construct the remote reference with tag: github/gh-aw/actions/name@tag
	// The SHA will be resolved later by action pinning infrastructure
	remoteRef := fmt.Sprintf("%s/%s@%s", GitHubOrgRepo, actionPath, tag)
	actionRefLog.Printf("Remote reference: %s (SHA will be resolved via action pins)", remoteRef)

	return remoteRef
}

// convertToExternalActionsRef converts a local action path to a SHA-pinned (if possible) reference
// in the external github/gh-aw-actions repository.
// Example: "./actions/create-issue" -> "github/gh-aw-actions/create-issue@<sha> # v1.0.0"
//
// If SHA resolution fails (no resolver or pin not available), falls back to version-tagged reference:
// Example: "./actions/create-issue" -> "github/gh-aw-actions/create-issue@v1.0.0"
func (c *Compiler) convertToExternalActionsRef(localPath string, data *WorkflowData) string {
	// Strip the leading "./" prefix
	actionPath := strings.TrimPrefix(localPath, "./")

	// Strip the "actions/" prefix to get just the action name
	// e.g., "actions/create-issue" -> "create-issue"
	actionName := strings.TrimPrefix(actionPath, "actions/")

	// Determine tag: use compiler actionTag or version
	tag := c.actionTag
	if tag == "" {
		if data != nil && data.Features != nil {
			if actionTagVal, exists := data.Features["action-tag"]; exists {
				if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
					tag = actionTagStr
				}
			}
		}
	}
	if tag == "" {
		tag = c.version
		if tag == "" || tag == "dev" {
			actionRefLog.Print("WARNING: No release tag available in binary version (version is 'dev' or empty)")
			return ""
		}
	}

	// Construct the external actions reference: <actionsRepo>/action-name@tag
	actionRepo := fmt.Sprintf("%s/%s", c.effectiveActionsRepo(), actionName)
	remoteRef := fmt.Sprintf("%s@%s", actionRepo, tag)

	// Try to resolve the SHA using action pins
	if data != nil {
		pinnedRef, err := getActionPinWithData(actionRepo, tag, data)
		if err != nil {
			// Log and fall through to tag-based reference (action mode is not strict)
			actionRefLog.Printf("Failed to pin action %s@%s: %v, falling back to tag-based reference", actionRepo, tag, err)
		} else if pinnedRef != "" {
			actionRefLog.Printf("Action mode: resolved %s to SHA-pinned reference: %s", remoteRef, pinnedRef)
			return pinnedRef
		}
	}

	// If SHA resolution unavailable or pin not found, return tag-based reference
	actionRefLog.Printf("Action mode: using tag-based external actions repo reference: %s (SHA will be resolved later)", remoteRef)
	return remoteRef
}
