package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var dangerousPermissionsLog = logger.New("workflow:dangerous_permissions_validation")

// validateDangerousPermissions validates that write permissions are not used unless
// the dangerous-permissions-write feature flag is enabled.
//
// This validation applies to:
// - Top-level workflow permissions
//
// This validation does NOT apply to:
// - Custom jobs (jobs defined in the jobs: section)
// - Safe outputs jobs (jobs defined in safe-outputs.job section)
//
// Returns an error if write permissions are found without the feature flag enabled.
func validateDangerousPermissions(workflowData *WorkflowData) error {
	dangerousPermissionsLog.Print("Starting dangerous permissions validation")

	// Check if the feature flag is enabled
	featureEnabled := isFeatureEnabled(constants.DangerousPermissionsWriteFeatureFlag, workflowData)
	if featureEnabled {
		dangerousPermissionsLog.Print("dangerous-permissions-write feature flag is enabled, allowing write permissions")
		return nil
	}

	// Parse the top-level workflow permissions
	if workflowData.Permissions == "" {
		dangerousPermissionsLog.Print("No permissions defined, validation passed")
		return nil
	}

	permissions := NewPermissionsParser(workflowData.Permissions).ToPermissions()
	if permissions == nil {
		dangerousPermissionsLog.Print("Could not parse permissions, validation passed")
		return nil
	}

	// Check for write permissions
	writePermissions := findWritePermissions(permissions)
	if len(writePermissions) > 0 {
		dangerousPermissionsLog.Printf("Found %d write permissions without feature flag", len(writePermissions))
		return formatDangerousPermissionsError(writePermissions)
	}

	dangerousPermissionsLog.Print("No write permissions found, validation passed")
	return nil
}

// findWritePermissions returns a list of permission scopes that have write access
func findWritePermissions(permissions *Permissions) []PermissionScope {
	if permissions == nil {
		return nil
	}

	var writePerms []PermissionScope

	// Check all permission scopes
	for _, scope := range GetAllPermissionScopes() {
		level, exists := permissions.Get(scope)
		if exists && level == PermissionWrite {
			writePerms = append(writePerms, scope)
		}
	}

	return writePerms
}

// formatDangerousPermissionsError formats an error message for write permissions violations
func formatDangerousPermissionsError(writePermissions []PermissionScope) error {
	var lines []string
	lines = append(lines, "🔒 Write permissions require extra safety configuration")
	lines = append(lines, "")
	lines = append(lines, "💡 What we found:")
	for _, scope := range writePermissions {
		lines = append(lines, fmt.Sprintf("  - %s: write", scope))
	}
	lines = append(lines, "")
	lines = append(lines, "GitHub Agentic Workflows uses read-only permissions by default to keep your")
	lines = append(lines, "repositories safe. Write operations are handled through validated 'safe-outputs'.")
	lines = append(lines, "")
	lines = append(lines, "✅ How to fix:")
	lines = append(lines, "")
	lines = append(lines, "Option 1: Use safe-outputs (recommended)")
	lines = append(lines, "Replace write permissions with safe-outputs for controlled operations:")
	lines = append(lines, "")
	lines = append(lines, "permissions:")
	for _, scope := range writePermissions {
		lines = append(lines, fmt.Sprintf("  %s: read", scope))
	}
	lines = append(lines, "safe-outputs:")
	lines = append(lines, "  create-issue:    # For creating issues")
	lines = append(lines, "  add-comment:     # For adding comments")
	lines = append(lines, "")
	lines = append(lines, "Option 2: Enable write permissions feature")
	lines = append(lines, "If you need direct write access, enable the feature flag:")
	lines = append(lines, "")
	lines = append(lines, "features:")
	lines = append(lines, "  dangerous-permissions-write: true")
	lines = append(lines, "")
	lines = append(lines, "📚 Learn more:")
	lines = append(lines, "  Safe outputs: https://githubnext.github.io/gh-aw/reference/safe-outputs/")
	lines = append(lines, "  Security: https://githubnext.github.io/gh-aw/guides/security/")

	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}
