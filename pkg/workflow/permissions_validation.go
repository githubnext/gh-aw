package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

var permissionsValidationLog = newValidationLogger("permissions")

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
	permissionsValidationLog.Print("Loading GitHub toolsets permissions from embedded JSON")

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

	permissionsValidationLog.Printf("Loaded %d GitHub toolsets from JSON", len(toolsetPermissionsMap))
}

// ValidatableTool represents a tool configuration that can be validated for permissions
// This interface abstracts the tool configuration structure to enable type-safe permission validation
type ValidatableTool interface {
	// GetToolsets returns the comma-separated list of toolsets configured for this tool
	GetToolsets() string
	// IsReadOnly returns whether the tool is configured in read-only mode
	IsReadOnly() bool
}

// GetToolsets implements ValidatableTool for GitHubToolConfig
func (g *GitHubToolConfig) GetToolsets() string {
	if g == nil {
		// Should not happen - ValidatePermissions checks for nil before calling this
		return ""
	}
	// Convert toolset array to comma-separated string
	// If empty, expandDefaultToolset will apply defaults
	toolsetsStr := strings.Join(g.Toolset.ToStringSlice(), ",")
	return expandDefaultToolset(toolsetsStr)
}

// IsReadOnly implements ValidatableTool for GitHubToolConfig.
// The GitHub MCP server always operates in read-only mode.
func (g *GitHubToolConfig) IsReadOnly() bool {
	return true
}

// PermissionsValidationResult contains the result of permissions validation
type PermissionsValidationResult struct {
	MissingPermissions    map[PermissionScope]PermissionLevel // Permissions required but not granted
	ReadOnlyMode          bool                                // Whether the GitHub MCP is in read-only mode
	HasValidationIssues   bool                                // Whether there are any validation issues
	MissingToolsetDetails map[string][]PermissionScope        // Maps toolset name to missing permissions
}

// ValidatePermissions validates that workflow permissions match the required GitHub MCP toolsets
// This is the general-purpose permission validator used during workflow compilation to check
// that the declared permissions are sufficient for the GitHub MCP toolsets being used.
//
// Parameters:
//   - permissions: The workflow's declared permissions
//   - githubTool: The GitHub tool configuration implementing ValidatableTool interface
//
// Returns:
//   - A validation result indicating any missing permissions and which toolsets require them
//
// Use ValidatePermissions (this function) for general permission validation against GitHub MCP toolsets.
// Use ValidateIncludedPermissions (in imports.go) when validating permissions from included/imported workflow files.
func ValidatePermissions(permissions *Permissions, githubTool ValidatableTool) *PermissionsValidationResult {
	permissionsValidationLog.Print("Starting permissions validation")

	result := &PermissionsValidationResult{
		MissingPermissions:    make(map[PermissionScope]PermissionLevel),
		MissingToolsetDetails: make(map[string][]PermissionScope),
	}

	// If GitHub tool is not configured, no validation needed
	// Check both for nil interface and nil concrete type
	if githubTool == nil {
		permissionsValidationLog.Print("No GitHub tool configured (nil interface), skipping validation")
		return result
	}

	// Check if concrete type is nil (interface wrapping nil pointer)
	if config, ok := githubTool.(*GitHubToolConfig); ok && config == nil {
		permissionsValidationLog.Print("No GitHub tool configured (nil concrete type), skipping validation")
		return result
	}

	// Extract toolsets from GitHub tool configuration
	toolsetsStr := githubTool.GetToolsets()
	readOnly := githubTool.IsReadOnly()
	result.ReadOnlyMode = readOnly

	permissionsValidationLog.Printf("Validating toolsets: %s, read-only: %v", toolsetsStr, readOnly)

	// Parse toolsets
	toolsets := ParseGitHubToolsets(toolsetsStr)
	if len(toolsets) == 0 {
		permissionsValidationLog.Print("No toolsets to validate")
		return result
	}

	// Collect required permissions for all toolsets
	requiredPermissions := collectRequiredPermissions(toolsets, readOnly)
	permissionsValidationLog.Printf("Required permissions: %v", requiredPermissions)

	// Check for missing permissions
	checkMissingPermissions(permissions, requiredPermissions, toolsets, result)

	result.HasValidationIssues = len(result.MissingPermissions) > 0
	permissionsValidationLog.Printf("Validation complete: hasIssues=%v, missingCount=%d", result.HasValidationIssues, len(result.MissingPermissions))

	return result
}

// collectRequiredPermissions collects all required permissions for the given toolsets
func collectRequiredPermissions(toolsets []string, readOnly bool) map[PermissionScope]PermissionLevel {
	permissionsValidationLog.Printf("Collecting required permissions for %d toolsets, read_only=%t", len(toolsets), readOnly)
	required := make(map[PermissionScope]PermissionLevel)

	for _, toolset := range toolsets {
		perms, exists := toolsetPermissionsMap[toolset]
		if !exists {
			permissionsValidationLog.Printf("Unknown toolset: %s", toolset)
			continue
		}

		// Add read permissions only (write tools are not considered for permission requirements)
		for _, scope := range perms.ReadPermissions {
			// Always require at least read access
			if existing, found := required[scope]; !found || existing == PermissionNone {
				required[scope] = PermissionRead
			}
		}
	}

	return required
}

// checkMissingPermissions checks if all required permissions are granted
func checkMissingPermissions(permissions *Permissions, required map[PermissionScope]PermissionLevel, toolsets []string, result *PermissionsValidationResult) {
	permissionsValidationLog.Printf("Checking missing permissions: required_count=%d, toolsets=%v", len(required), toolsets)
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

				requiresScope := slices.Contains(perms.ReadPermissions, scope)
				if !requiresScope {
					if slices.Contains(perms.WritePermissions, scope) {
						requiresScope = true
					}
				}

				if requiresScope {
					result.MissingToolsetDetails[toolset] = append(result.MissingToolsetDetails[toolset], scope)
				}
			}
		}
	}
}

// FormatValidationMessage formats the validation result into a human-readable message
func FormatValidationMessage(result *PermissionsValidationResult, strict bool) string {
	if !result.HasValidationIssues {
		return ""
	}

	// Format missing permissions
	if len(result.MissingPermissions) > 0 {
		return formatMissingPermissionsMessage(result)
	}

	return ""
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
				if slices.Contains(toolsetScopes, scope) {
					requiredBy = append(requiredBy, toolset)
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

	lines = append(lines, "Missing required permissions for GitHub toolsets:")
	lines = append(lines, permLines...)
	lines = append(lines, "")
	lines = append(lines, "To fix this, you can either:")
	lines = append(lines, "")
	lines = append(lines, "Option 1: Add missing permissions to your workflow frontmatter:")
	lines = append(lines, "permissions:")
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.MissingPermissions[scope]
		lines = append(lines, fmt.Sprintf("  %s: %s", scope, level))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("See: %s", constants.DocsPermissionsURL))

	// Add suggestion to reduce toolsets if we have toolset details
	if len(result.MissingToolsetDetails) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Option 2: Reduce the required toolsets in your workflow:")
		lines = append(lines, "Remove or adjust toolsets that require these permissions:")

		// Get unique toolsets from MissingToolsetDetails
		toolsetsMap := make(map[string]bool)
		for toolset := range result.MissingToolsetDetails {
			toolsetsMap[toolset] = true
		}
		var toolsetsList []string
		for toolset := range toolsetsMap {
			toolsetsList = append(toolsetsList, toolset)
		}
		sort.Strings(toolsetsList)

		for _, toolset := range toolsetsList {
			lines = append(lines, "  - "+toolset)
		}
	}

	return strings.Join(lines, "\n")
}

// ValidateIncludedPermissions validates that the main workflow permissions satisfy the imported
// workflow requirements. This function is specifically used when merging included/imported workflow
// files to ensure the main workflow has sufficient permissions to support all imported files.
//
// Use ValidatePermissions (in permissions_validator.go) for general permission validation against
// GitHub MCP toolsets. Use ValidateIncludedPermissions (this function) when validating permissions
// from included/imported workflow files.
func (c *Compiler) ValidateIncludedPermissions(topPermissionsYAML string, importedPermissionsJSON string) error {
	permissionsValidationLog.Print("Validating included workflow permissions")

	// If no imported permissions, no validation needed
	if importedPermissionsJSON == "" || importedPermissionsJSON == "{}" {
		permissionsValidationLog.Print("No included workflow permissions to validate")
		return nil
	}

	// Parse top-level permissions
	var topPerms *Permissions
	if topPermissionsYAML != "" {
		topPerms = NewPermissionsParser(topPermissionsYAML).ToPermissions()
	} else {
		topPerms = NewPermissions()
	}

	// Track missing permissions
	missingPermissions := make(map[PermissionScope]PermissionLevel)
	insufficientPermissions := make(map[PermissionScope]struct {
		required PermissionLevel
		current  PermissionLevel
	})

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(importedPermissionsJSON, "\n")
	permissionsValidationLog.Printf("Processing %d permission definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to permissions map
		var importedPermsMap map[string]any
		if err := json.Unmarshal([]byte(line), &importedPermsMap); err != nil {
			permissionsValidationLog.Printf("Skipping malformed permission entry: %q (error: %v)", line, err)
			continue
		}

		// Check each permission from the imported map
		for scopeStr, levelValue := range importedPermsMap {
			scope := PermissionScope(scopeStr)

			// Parse the level - it might be a string or already unmarshaled
			var requiredLevel PermissionLevel
			if levelStr, ok := levelValue.(string); ok {
				requiredLevel = PermissionLevel(levelStr)
			} else {
				// Skip invalid level values
				continue
			}

			// Get current level for this scope
			currentLevel, exists := topPerms.Get(scope)

			// Validate that the main workflow has sufficient permissions
			if !exists || currentLevel == PermissionNone {
				// Permission is missing entirely
				missingPermissions[scope] = requiredLevel
				permissionsValidationLog.Printf("Missing permission: %s: %s", scope, requiredLevel)
			} else if !isPermissionSufficient(currentLevel, requiredLevel) {
				// Permission exists but is insufficient
				insufficientPermissions[scope] = struct {
					required PermissionLevel
					current  PermissionLevel
				}{requiredLevel, currentLevel}
				permissionsValidationLog.Printf("Insufficient permission: %s: has %s, needs %s", scope, currentLevel, requiredLevel)
			}
		}
	}

	// If there are missing or insufficient permissions, return an error
	if len(missingPermissions) > 0 || len(insufficientPermissions) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("ERROR: Imported workflows require permissions that are not granted in the main workflow.\n\n")
		errorMsg.WriteString("The permission set must be explicitly declared in the main workflow.\n\n")

		if len(missingPermissions) > 0 {
			errorMsg.WriteString("Missing permissions:\n")
			// Sort for consistent output
			var scopes []PermissionScope
			for scope := range missingPermissions {
				scopes = append(scopes, scope)
			}
			SortPermissionScopes(scopes)
			for _, scope := range scopes {
				level := missingPermissions[scope]
				fmt.Fprintf(&errorMsg, "  - %s: %s\n", scope, level)
			}
			errorMsg.WriteString("\n")
		}

		if len(insufficientPermissions) > 0 {
			errorMsg.WriteString("Insufficient permissions:\n")
			// Sort for consistent output
			var scopes []PermissionScope
			for scope := range insufficientPermissions {
				scopes = append(scopes, scope)
			}
			SortPermissionScopes(scopes)
			for _, scope := range scopes {
				info := insufficientPermissions[scope]
				fmt.Fprintf(&errorMsg, "  - %s: has %s, requires %s\n", scope, info.current, info.required)
			}
			errorMsg.WriteString("\n")
		}

		errorMsg.WriteString("Suggested fix: Add the required permissions to your main workflow frontmatter:\n")
		errorMsg.WriteString("permissions:\n")

		// Combine all required permissions for the suggestion
		allRequired := make(map[PermissionScope]PermissionLevel)
		maps.Copy(allRequired, missingPermissions)
		for scope, info := range insufficientPermissions {
			allRequired[scope] = info.required
		}

		var scopes []PermissionScope
		for scope := range allRequired {
			scopes = append(scopes, scope)
		}
		SortPermissionScopes(scopes)
		for _, scope := range scopes {
			level := allRequired[scope]
			fmt.Fprintf(&errorMsg, "  %s: %s\n", scope, level)
		}

		return fmt.Errorf("%s", errorMsg.String())
	}

	permissionsValidationLog.Print("All included workflow permissions are satisfied by main workflow")
	return nil
}

// isPermissionSufficient checks if the current permission level is sufficient for the required level.
// write > read > none
func isPermissionSufficient(current, required PermissionLevel) bool {
	if current == required {
		return true
	}
	// write satisfies read requirement
	if current == PermissionWrite && required == PermissionRead {
		return true
	}
	return false
}
