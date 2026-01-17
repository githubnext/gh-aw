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

// formatDangerousPermissionsError formats an empathetic error message for write permissions violations
func formatDangerousPermissionsError(writePermissions []PermissionScope) error {
	var lines []string
	lines = append(lines, "🔒 Write permissions detected in your workflow.")
	lines = append(lines, "")
	lines = append(lines, "Why this matters: For security, workflows use read-only permissions by default.")
	lines = append(lines, "Write permissions can modify repository contents and settings, which requires")
	lines = append(lines, "explicit opt-in through a feature flag.")
	lines = append(lines, "")
	lines = append(lines, "Found write permissions:")
	for _, scope := range writePermissions {
		lines = append(lines, fmt.Sprintf("  • %s: write", scope))
	}
	lines = append(lines, "")
	lines = append(lines, "You have two options:")
	lines = append(lines, "")
	lines = append(lines, "1. Change to read-only (recommended):")
	lines = append(lines, "   permissions:")
	for _, scope := range writePermissions {
		lines = append(lines, fmt.Sprintf("     %s: read", scope))
	}
	lines = append(lines, "")
	lines = append(lines, "2. Enable write permissions feature flag:")
	lines = append(lines, "   features:")
	lines = append(lines, "     dangerous-permissions-write: true")
	lines = append(lines, "")
	lines = append(lines, "Learn more: https://githubnext.github.io/gh-aw/reference/permissions/")

	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}
