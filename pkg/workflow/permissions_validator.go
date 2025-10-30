package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var permissionsValidatorLog = logger.New("workflow:permissions_validator")

//go:embed data/github_toolsets_permissions.json
var githubToolsetsPermissionsJSON []byte

// GitHubToolsetPermissions maps GitHub MCP toolsets to their required permissions
type GitHubToolsetPermissions struct {
	ReadPermissions  []PermissionScope
	WritePermissions []PermissionScope
	Tools            []string // List of tools in this toolset (for verification)
}

// GitHubToolsetsData represents the structure of the embedded JSON file
type GitHubToolsetsData struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Toolsets    map[string]struct {
		Description      string   `json:"description"`
		ReadPermissions  []string `json:"read_permissions"`
		WritePermissions []string `json:"write_permissions"`
		Tools            []string `json:"tools"`
	} `json:"toolsets"`
}

// toolsetPermissionsMap defines the mapping of GitHub MCP toolsets to required permissions
// This is loaded from the embedded JSON file at initialization
var toolsetPermissionsMap map[string]GitHubToolsetPermissions

// init loads the GitHub toolsets and permissions from the embedded JSON
func init() {
	permissionsValidatorLog.Print("Loading GitHub toolsets permissions from embedded JSON")

	var data GitHubToolsetsData
	if err := json.Unmarshal(githubToolsetsPermissionsJSON, &data); err != nil {
		panic(fmt.Sprintf("failed to load GitHub toolsets permissions from JSON: %v", err))
	}

	// Convert JSON data to internal format
	toolsetPermissionsMap = make(map[string]GitHubToolsetPermissions)
	for toolsetName, toolsetData := range data.Toolsets {
		// Convert string permission names to PermissionScope types
		readPerms := make([]PermissionScope, len(toolsetData.ReadPermissions))
		for i, perm := range toolsetData.ReadPermissions {
			readPerms[i] = PermissionScope(perm)
		}

		writePerms := make([]PermissionScope, len(toolsetData.WritePermissions))
		for i, perm := range toolsetData.WritePermissions {
			writePerms[i] = PermissionScope(perm)
		}

		toolsetPermissionsMap[toolsetName] = GitHubToolsetPermissions{
			ReadPermissions:  readPerms,
			WritePermissions: writePerms,
			Tools:            toolsetData.Tools,
		}
	}

	permissionsValidatorLog.Printf("Loaded %d GitHub toolsets from JSON", len(toolsetPermissionsMap))
}

// GetToolsetsData returns the parsed GitHub toolsets data (for use by workflows)
func GetToolsetsData() GitHubToolsetsData {
	var data GitHubToolsetsData
	if err := json.Unmarshal(githubToolsetsPermissionsJSON, &data); err != nil {
		// This should never happen as we validate in init
		panic(fmt.Sprintf("failed to parse GitHub toolsets data: %v", err))
	}
	return data
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
	
	// Build permission list with toolset details inline
	var permLines []string
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.MissingPermissions[scope]
		
		// Find which toolsets need this permission
		var requiredBy []string
		if len(result.MissingToolsetDetails) > 0 {
			for toolset, toolsetScopes := range result.MissingToolsetDetails {
				for _, ts := range toolsetScopes {
					if ts == scope {
						requiredBy = append(requiredBy, toolset)
						break
					}
				}
			}
		}
		
		// Format: "- scope: level (required by toolset1, toolset2)"
		if len(requiredBy) > 0 {
			sort.Strings(requiredBy)
			permLines = append(permLines, fmt.Sprintf("  - %s: %s (required by %s)", scope, level, strings.Join(requiredBy, ", ")))
		} else {
			permLines = append(permLines, fmt.Sprintf("  - %s: %s", scope, level))
		}
	}

	lines = append(lines, "Missing required permissions for github toolsets:")
	lines = append(lines, permLines...)
	lines = append(lines, "")
	lines = append(lines, "Add to your workflow frontmatter:")
	lines = append(lines, "permissions:")
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.MissingPermissions[scope]
		lines = append(lines, fmt.Sprintf("  %s: %s", scope, level))
	}

	return strings.Join(lines, "\n")
}

// formatExcessPermissionsMessage formats the excess permissions warning message
func formatExcessPermissionsMessage(result *PermissionsValidationResult, strict bool) string {
	var scopes []string
	for scope := range result.ExcessPermissions {
		scopes = append(scopes, string(scope))
	}
	sort.Strings(scopes)

	var lines []string
	lines = append(lines, "Over-provisioned permissions detected for github toolsets:")

	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.ExcessPermissions[scope]
		lines = append(lines, fmt.Sprintf("  - %s: %s (not required)", scope, level))
	}

	lines = append(lines, "")
	lines = append(lines, "Only grant permissions that are needed.")

	return strings.Join(lines, "\n")
}
