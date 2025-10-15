package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// generateMembershipCheck generates steps for the check_membership job that only sets outputs
func (c *Compiler) generateMembershipCheck(data *WorkflowData, steps []string) []string {
	if data.Command != "" {
		steps = append(steps, "      - name: Check team membership for command workflow\n")
	} else {
		steps = append(steps, "      - name: Check team membership for workflow\n")
	}
	steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckMembershipStepID))
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables for permission check
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_REQUIRED_ROLES: %s\n", strings.Join(data.Roles, ",")))

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Generate the JavaScript code for the membership check (output-only version)
	scriptContent := c.generateMembershipCheckScript(data.Roles)
	scriptLines := strings.Split(scriptContent, "\n")
	for _, line := range scriptLines {
		if strings.TrimSpace(line) != "" {
			steps = append(steps, fmt.Sprintf("            %s\n", line))
		}
	}
	return steps
}

// generateMembershipCheckScript generates JavaScript code to check user permissions (output-only version)
func (c *Compiler) generateMembershipCheckScript(requiredPermissions []string) string {
	// If "all" is specified, no checks needed (this shouldn't happen since needsRoleCheck would return false)
	if len(requiredPermissions) == 1 && requiredPermissions[0] == "all" {
		return `
core.setOutput("is_team_member", "true");
core.setOutput("result", "roles_all");
console.log("Permission check skipped - 'roles: all' specified");`
	}

	// Use the embedded check_membership.cjs script
	// The GITHUB_AW_REQUIRED_ROLES environment variable is set via the env field
	return checkMembershipScript
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

	// Command workflows always need permission checks
	if data.Command != "" {
		return true
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
			hasWorkflowDispatch := false

			for eventName := range onMap {
				// Skip command events as they are handled separately
				// Skip stop-after and reaction as they are not event types
				if eventName == "command" || eventName == "stop-after" || eventName == "reaction" {
					continue
				}

				// Track if workflow_dispatch is present
				if eventName == "workflow_dispatch" {
					hasWorkflowDispatch = true
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

			// Special handling for workflow_dispatch:
			// workflow_dispatch can be triggered by users with "write" access,
			// so it's only considered "safe" if "write" is in the allowed roles
			if hasWorkflowDispatch && !hasUnsafeEvents {
				// Check if "write" is in the allowed roles
				hasWriteRole := false
				for _, role := range data.Roles {
					if role == "write" {
						hasWriteRole = true
						break
					}
				}
				// If write is not in the allowed roles, workflow_dispatch needs permission checks
				if !hasWriteRole {
					return false
				}
			}

			return eventCount > 0 && !hasUnsafeEvents
		}
	}

	// If no "on" section or it's a string, check for default command trigger
	// For command workflows, they are not considered "safe only"
	return false
}
