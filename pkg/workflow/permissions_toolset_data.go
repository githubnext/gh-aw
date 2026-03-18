package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
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
