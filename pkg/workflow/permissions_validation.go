package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var permissionsValidationLog = logger.New("workflow:permissions_validation")

// validatePermissionScopes validates that all permission scopes are valid for GitHub Actions
// This catches common mistakes like using 'repository-projects' which exists in the schema
// but doesn't actually work in GitHub Actions workflows
func validatePermissionScopes(frontmatter map[string]any) error {
	permissionsValue, exists := frontmatter["permissions"]
	if !exists {
		// No permissions specified is fine
		permissionsValidationLog.Print("No permissions specified, validation skipped")
		return nil
	}

	// Handle shorthand permissions (read-all, write-all, read, write, none)
	if strValue, ok := permissionsValue.(string); ok {
		permissionsValidationLog.Printf("Shorthand permission detected: %s, validation skipped", strValue)
		return nil
	}

	// Handle map format
	permsMap, ok := permissionsValue.(map[string]any)
	if !ok {
		permissionsValidationLog.Print("Permissions not in expected format, validation skipped")
		return nil
	}

	// List of invalid permission scopes with explanations
	invalidScopes := map[string]string{
		"repository-projects": "The 'repository-projects' permission is for classic GitHub Projects (which have been sunset) and only works with the REST API, not with GitHub Actions workflows. For Projects v2, use a Personal Access Token (PAT) or GitHub App token instead of GITHUB_TOKEN. See: https://github.com/orgs/community/discussions/54538",
	}

	// Check each permission scope
	var errors []string
	for scope := range permsMap {
		if explanation, isInvalid := invalidScopes[scope]; isInvalid {
			permissionsValidationLog.Printf("Invalid permission scope detected: %s", scope)
			errors = append(errors, fmt.Sprintf("Invalid permission scope '%s': %s", scope, explanation))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "\n\n"))
	}

	permissionsValidationLog.Print("Permission scopes validation passed")
	return nil
}
