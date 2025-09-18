package workflow

import "github.com/githubnext/gh-aw/pkg/constants"

// generateRoleCheckScript generates JavaScript code to check user permissions
func (c *Compiler) generateRoleCheckScript(requiredPermissions []string) string {
	// If "all" is specified, no checks needed (this shouldn't happen since needsRoleCheck would return false)
	if len(requiredPermissions) == 1 && requiredPermissions[0] == "all" {
		return `
core.setOutput("is_team_member", "true");
console.log("Permission check skipped - 'roles: all' specified");`
	}

	// Use the embedded check_permissions.cjs script
	// The GITHUB_AW_REQUIRED_ROLES environment variable is set via the env field
	return checkPermissionsScript
}

// extractRoles extracts the 'roles' field from frontmatter to determine permission requirements
func (c *Compiler) extractRoles(frontmatter map[string]any) []string {
	if rolesValue, exists := frontmatter["roles"]; exists {
		switch v := rolesValue.(type) {
		case string:
			if v == "all" {
				// Special case: "all" means no restrictions
				return []string{"all"}
			}
			// Single permission level as string
			return []string{v}
		case []any:
			// Array of permission levels
			var permissions []string
			for _, item := range v {
				if str, ok := item.(string); ok {
					permissions = append(permissions, str)
				}
			}
			return permissions
		case []string:
			// Already a string slice
			return v
		}
	}
	// Default: require admin or maintainer permissions
	return []string{"admin", "maintainer"}
}

// needsRoleCheck determines if the workflow needs permission checks with full context
func (c *Compiler) needsRoleCheck(data *WorkflowData, frontmatter map[string]any) bool {
	// If user explicitly specified "roles: all", no permission checks needed
	if len(data.Roles) == 1 && data.Roles[0] == "all" {
		return false
	}

	// Check if the workflow uses only safe events (only if frontmatter is available)
	if frontmatter != nil && c.hasSafeEventsOnly(data, frontmatter) {
		return false
	}

	// Permission checks are needed by default for non-safe events
	return true
}

// hasSafeEventsOnly checks if the workflow uses only safe events that don't require permission checks
func (c *Compiler) hasSafeEventsOnly(data *WorkflowData, frontmatter map[string]any) bool {
	// If user explicitly specified "roles: all", skip permission checks
	if len(data.Roles) == 1 && data.Roles[0] == "all" {
		return true
	}

	// Parse the "on" section to determine events
	if onValue, exists := frontmatter["on"]; exists {
		if onMap, ok := onValue.(map[string]any); ok {
			// Check if only safe events are present
			hasUnsafeEvents := false

			for eventName := range onMap {
				// Skip command events as they are handled separately
				// Skip stop-after and reaction as they are not event types
				if eventName == "command" || eventName == "stop-after" || eventName == "reaction" {
					continue
				}

				// Check if this event is in the safe list
				isSafe := false
				for _, safeEvent := range constants.SafeWorkflowEvents {
					if eventName == safeEvent {
						isSafe = true
						break
					}
				}
				if !isSafe {
					hasUnsafeEvents = true
					break
				}
			}

			// If there are events and none are unsafe, then it's safe
			eventCount := len(onMap)
			// Subtract non-event entries
			if _, hasCommand := onMap["command"]; hasCommand {
				eventCount--
			}
			if _, hasStopAfter := onMap["stop-after"]; hasStopAfter {
				eventCount--
			}
			if _, hasReaction := onMap["reaction"]; hasReaction {
				eventCount--
			}

			return eventCount > 0 && !hasUnsafeEvents
		}
	}

	// If no "on" section or it's a string, check for default command trigger
	// For command workflows, they are not considered "safe only"
	return false
}
