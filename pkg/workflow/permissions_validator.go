package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var permissionsValidatorLog = logger.New("workflow:permissions_validator")

// GitHubToolsetPermissions maps GitHub MCP toolsets to their required permissions
type GitHubToolsetPermissions struct {
	ReadPermissions  []PermissionScope
	WritePermissions []PermissionScope
}

// toolsetPermissionsMap defines the mapping of GitHub MCP toolsets to required permissions
var toolsetPermissionsMap = map[string]GitHubToolsetPermissions{
	"context": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
	"repos": {
		ReadPermissions:  []PermissionScope{PermissionContents},
		WritePermissions: []PermissionScope{PermissionContents},
	},
	"issues": {
		ReadPermissions:  []PermissionScope{PermissionIssues},
		WritePermissions: []PermissionScope{PermissionIssues},
	},
	"pull_requests": {
		ReadPermissions:  []PermissionScope{PermissionPullRequests},
		WritePermissions: []PermissionScope{PermissionPullRequests},
	},
	"actions": {
		ReadPermissions:  []PermissionScope{PermissionActions},
		WritePermissions: []PermissionScope{},
	},
	"code_security": {
		ReadPermissions:  []PermissionScope{PermissionSecurityEvents},
		WritePermissions: []PermissionScope{PermissionSecurityEvents},
	},
	"dependabot": {
		ReadPermissions:  []PermissionScope{PermissionSecurityEvents},
		WritePermissions: []PermissionScope{},
	},
	"discussions": {
		ReadPermissions:  []PermissionScope{PermissionDiscussions},
		WritePermissions: []PermissionScope{PermissionDiscussions},
	},
	"experiments": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
	"gists": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
	"labels": {
		ReadPermissions:  []PermissionScope{PermissionIssues},
		WritePermissions: []PermissionScope{PermissionIssues},
	},
	"notifications": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
	"orgs": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
	"projects": {
		ReadPermissions:  []PermissionScope{PermissionRepositoryProj},
		WritePermissions: []PermissionScope{PermissionRepositoryProj},
	},
	"secret_protection": {
		ReadPermissions:  []PermissionScope{PermissionSecurityEvents},
		WritePermissions: []PermissionScope{},
	},
	"security_advisories": {
		ReadPermissions:  []PermissionScope{PermissionSecurityEvents},
		WritePermissions: []PermissionScope{PermissionSecurityEvents},
	},
	"stargazers": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
	"users": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
	"search": {
		ReadPermissions:  []PermissionScope{},
		WritePermissions: []PermissionScope{},
	},
}

// PermissionsValidationResult contains the result of permissions validation
type PermissionsValidationResult struct {
	MissingPermissions     map[PermissionScope]PermissionLevel // Permissions required but not granted
	ExcessPermissions      map[PermissionScope]PermissionLevel // Permissions granted but not needed
	ReadOnlyMode           bool                                // Whether the GitHub MCP is in read-only mode
	HasValidationIssues    bool                                // Whether there are any validation issues
	MissingToolsetDetails  map[string][]PermissionScope        // Maps toolset name to missing permissions
	ExcessPermissionScopes []PermissionScope                   // List of permission scopes that are over-provisioned
}

// ValidatePermissions validates that permissions match the required GitHub MCP toolsets
func ValidatePermissions(permissions *Permissions, githubTool any) *PermissionsValidationResult {
	permissionsValidatorLog.Print("Starting permissions validation")

	result := &PermissionsValidationResult{
		MissingPermissions:    make(map[PermissionScope]PermissionLevel),
		ExcessPermissions:     make(map[PermissionScope]PermissionLevel),
		MissingToolsetDetails: make(map[string][]PermissionScope),
	}

	// If GitHub tool is not configured, no validation needed
	if githubTool == nil {
		permissionsValidatorLog.Print("No GitHub tool configured, skipping validation")
		return result
	}

	// Extract toolsets from GitHub tool configuration
	toolsetsStr := getGitHubToolsets(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	result.ReadOnlyMode = readOnly

	permissionsValidatorLog.Printf("Validating toolsets: %s, read-only: %v", toolsetsStr, readOnly)

	// Parse toolsets
	toolsets := ParseGitHubToolsets(toolsetsStr)
	if len(toolsets) == 0 {
		permissionsValidatorLog.Print("No toolsets to validate")
		return result
	}

	// Collect required permissions for all toolsets
	requiredPermissions := collectRequiredPermissions(toolsets, readOnly)
	permissionsValidatorLog.Printf("Required permissions: %v", requiredPermissions)

	// Check for missing permissions
	checkMissingPermissions(permissions, requiredPermissions, toolsets, result)

	// Check for excess permissions (only for specific toolsets, not "all")
	// Note: "default" is expanded to specific toolsets, so excess checking should still happen
	shouldCheckExcess := !containsAllToolset(toolsetsStr)
	if shouldCheckExcess {
		checkExcessPermissions(permissions, requiredPermissions, result)
	}

	result.HasValidationIssues = len(result.MissingPermissions) > 0 || len(result.ExcessPermissions) > 0

	return result
}

// containsAllToolset checks if the original toolsets string contains "all"
func containsAllToolset(toolsetsStr string) bool {
	toolsets := strings.Split(toolsetsStr, ",")
	for _, t := range toolsets {
		if strings.TrimSpace(t) == "all" {
			return true
		}
	}
	return false
}

// collectRequiredPermissions collects all required permissions for the given toolsets
func collectRequiredPermissions(toolsets []string, readOnly bool) map[PermissionScope]PermissionLevel {
	required := make(map[PermissionScope]PermissionLevel)

	for _, toolset := range toolsets {
		perms, exists := toolsetPermissionsMap[toolset]
		if !exists {
			permissionsValidatorLog.Printf("Unknown toolset: %s", toolset)
			continue
		}

		// Add read permissions
		for _, scope := range perms.ReadPermissions {
			// Always require at least read access
			if existing, found := required[scope]; !found || existing == PermissionNone {
				required[scope] = PermissionRead
			}
		}

		// Add write permissions only if not in read-only mode
		if !readOnly {
			for _, scope := range perms.WritePermissions {
				required[scope] = PermissionWrite
			}
		}
	}

	return required
}

// checkMissingPermissions checks if all required permissions are granted
func checkMissingPermissions(permissions *Permissions, required map[PermissionScope]PermissionLevel, toolsets []string, result *PermissionsValidationResult) {
	for scope, requiredLevel := range required {
		grantedLevel, granted := permissions.Get(scope)

		missing := false
		if !granted {
			missing = true
		} else if requiredLevel == PermissionWrite && grantedLevel != PermissionWrite {
			missing = true
		}

		if missing {
			result.MissingPermissions[scope] = requiredLevel

			// Track which toolsets require this permission
			for _, toolset := range toolsets {
				perms, exists := toolsetPermissionsMap[toolset]
				if !exists {
					continue
				}

				requiresScope := false
				for _, readScope := range perms.ReadPermissions {
					if readScope == scope {
						requiresScope = true
						break
					}
				}
				if !requiresScope {
					for _, writeScope := range perms.WritePermissions {
						if writeScope == scope {
							requiresScope = true
							break
						}
					}
				}

				if requiresScope {
					result.MissingToolsetDetails[toolset] = append(result.MissingToolsetDetails[toolset], scope)
				}
			}
		}
	}
}

// checkExcessPermissions checks if any granted permissions are not required
func checkExcessPermissions(permissions *Permissions, required map[PermissionScope]PermissionLevel, result *PermissionsValidationResult) {
	// Get all permission scopes to check
	allScopes := GetAllPermissionScopes()

	for _, scope := range allScopes {
		grantedLevel, granted := permissions.Get(scope)
		if !granted {
			continue
		}

		// Skip if permission is none
		if grantedLevel == PermissionNone {
			continue
		}

		requiredLevel, needed := required[scope]

		// If permission is granted but not required at all
		if !needed {
			result.ExcessPermissions[scope] = grantedLevel
			result.ExcessPermissionScopes = append(result.ExcessPermissionScopes, scope)
			continue
		}

		// If write permission is granted but only read is required
		if grantedLevel == PermissionWrite && requiredLevel == PermissionRead {
			result.ExcessPermissions[scope] = PermissionWrite
			result.ExcessPermissionScopes = append(result.ExcessPermissionScopes, scope)
		}
	}
}

// FormatValidationMessage formats the validation result into a human-readable message
func FormatValidationMessage(result *PermissionsValidationResult, strict bool) string {
	if !result.HasValidationIssues {
		return ""
	}

	var messages []string

	// Format missing permissions
	if len(result.MissingPermissions) > 0 {
		msg := formatMissingPermissionsMessage(result)
		messages = append(messages, msg)
	}

	// Format excess permissions
	if len(result.ExcessPermissions) > 0 {
		msg := formatExcessPermissionsMessage(result, strict)
		messages = append(messages, msg)
	}

	return strings.Join(messages, "\n\n")
}

// formatMissingPermissionsMessage formats the missing permissions error message
func formatMissingPermissionsMessage(result *PermissionsValidationResult) string {
	var scopes []string
	for scope := range result.MissingPermissions {
		scopes = append(scopes, string(scope))
	}
	sort.Strings(scopes)

	var lines []string
	lines = append(lines, "ERROR: Missing required permissions for GitHub MCP toolsets:")

	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.MissingPermissions[scope]
		lines = append(lines, fmt.Sprintf("  - %s: %s", scope, level))
	}

	// Add toolset details
	if len(result.MissingToolsetDetails) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Required by toolsets:")

		var toolsetNames []string
		for toolset := range result.MissingToolsetDetails {
			toolsetNames = append(toolsetNames, toolset)
		}
		sort.Strings(toolsetNames)

		for _, toolset := range toolsetNames {
			scopes := result.MissingToolsetDetails[toolset]
			scopeStrs := make([]string, len(scopes))
			for i, s := range scopes {
				scopeStrs[i] = string(s)
			}
			lines = append(lines, fmt.Sprintf("  - %s: needs %s", toolset, strings.Join(scopeStrs, ", ")))
		}
	}

	lines = append(lines, "")
	lines = append(lines, "Suggested fix: Add the following to your workflow frontmatter:")
	lines = append(lines, "permissions:")
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.MissingPermissions[scope]
		lines = append(lines, fmt.Sprintf("  %s: %s", scope, level))
	}

	return strings.Join(lines, "\n")
}

// formatExcessPermissionsMessage formats the excess permissions warning/error message
func formatExcessPermissionsMessage(result *PermissionsValidationResult, strict bool) string {
	var scopes []string
	for scope := range result.ExcessPermissions {
		scopes = append(scopes, string(scope))
	}
	sort.Strings(scopes)

	var lines []string
	prefix := "WARNING"
	if strict {
		prefix = "ERROR"
	}

	lines = append(lines, fmt.Sprintf("%s: Over-provisioned permissions detected for GitHub MCP toolsets:", prefix))

	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.ExcessPermissions[scope]
		lines = append(lines, fmt.Sprintf("  - %s: %s (not required by configured toolsets)", scope, level))
	}

	lines = append(lines, "")
	lines = append(lines, "Principle of least privilege: Only grant permissions that are needed.")
	lines = append(lines, "Consider removing these permissions or adjusting your toolsets configuration.")

	return strings.Join(lines, "\n")
}
