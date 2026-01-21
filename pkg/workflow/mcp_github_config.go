package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var githubConfigLog = logger.New("workflow:mcp_github_config")

// hasGitHubTool checks if the GitHub tool is configured (using ParsedTools)
func hasGitHubTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.GitHub != nil
}

// getGitHubType extracts the mode from GitHub tool configuration (local or remote)
func getGitHubType(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if modeSetting, exists := toolConfig["mode"]; exists {
			if stringValue, ok := modeSetting.(string); ok {
				return stringValue
			}
		}
	}
	return "local" // default to local (Docker)
}

// getGitHubToken extracts the custom github-token from GitHub tool configuration
func getGitHubToken(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if tokenSetting, exists := toolConfig["github-token"]; exists {
			if stringValue, ok := tokenSetting.(string); ok {
				return stringValue
			}
		}
	}
	return ""
}

// getGitHubReadOnly checks if read-only mode is enabled for GitHub tool
// Defaults to true for security
func getGitHubReadOnly(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if readOnlySetting, exists := toolConfig["read-only"]; exists {
			if boolValue, ok := readOnlySetting.(bool); ok {
				return boolValue
			}
		}
	}
	return true // default to read-only for security
}

// getGitHubLockdown checks if lockdown mode is enabled for GitHub tool
// Defaults to false (lockdown disabled)
func getGitHubLockdown(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if lockdownSetting, exists := toolConfig["lockdown"]; exists {
			if boolValue, ok := lockdownSetting.(bool); ok {
				return boolValue
			}
		}
	}
	return false // default to lockdown disabled
}

// hasGitHubLockdownExplicitlySet checks if lockdown field is explicitly set in GitHub tool config
func hasGitHubLockdownExplicitlySet(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		_, exists := toolConfig["lockdown"]
		return exists
	}
	return false
}

// getGitHubToolsets extracts the toolsets configuration from GitHub tool
// Expands "default" to individual toolsets for action-friendly compatibility
func getGitHubToolsets(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if toolsetsSetting, exists := toolConfig["toolsets"]; exists {
			// Handle array format only
			switch v := toolsetsSetting.(type) {
			case []any:
				// Convert array to comma-separated string
				toolsets := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						toolsets = append(toolsets, str)
					}
				}
				toolsetsStr := strings.Join(toolsets, ",")
				// Expand "default" to individual toolsets for action-friendly compatibility
				return expandDefaultToolset(toolsetsStr)
			case []string:
				toolsetsStr := strings.Join(v, ",")
				// Expand "default" to individual toolsets for action-friendly compatibility
				return expandDefaultToolset(toolsetsStr)
			}
		}
	}
	// default to action-friendly toolsets (excludes "users" which GitHub Actions tokens don't support)
	return strings.Join(ActionFriendlyGitHubToolsets, ",")
}

// expandDefaultToolset expands "default" and "action-friendly" keywords to individual toolsets.
// This ensures that "default" and "action-friendly" in the source expand to action-friendly toolsets
// (excluding "users" which GitHub Actions tokens don't support).
func expandDefaultToolset(toolsetsStr string) string {
	if toolsetsStr == "" {
		return strings.Join(ActionFriendlyGitHubToolsets, ",")
	}

	// Split by comma and check if "default" or "action-friendly" is present
	toolsets := strings.Split(toolsetsStr, ",")
	var result []string
	seenToolsets := make(map[string]bool)

	for _, toolset := range toolsets {
		toolset = strings.TrimSpace(toolset)
		if toolset == "" {
			continue
		}

		if toolset == "default" || toolset == "action-friendly" {
			// Expand "default" or "action-friendly" to action-friendly toolsets (excludes "users")
			for _, dt := range ActionFriendlyGitHubToolsets {
				if !seenToolsets[dt] {
					result = append(result, dt)
					seenToolsets[dt] = true
				}
			}
		} else {
			// Keep other toolsets as-is (including "all", individual toolsets, etc.)
			if !seenToolsets[toolset] {
				result = append(result, toolset)
				seenToolsets[toolset] = true
			}
		}
	}

	return strings.Join(result, ",")
}

// getGitHubAllowedTools extracts the allowed tools list from GitHub tool configuration
// Returns the list of allowed tools, or nil if no allowed list is specified (which means all tools are allowed)
func getGitHubAllowedTools(githubTool any) []string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if allowedSetting, exists := toolConfig["allowed"]; exists {
			// Handle array format
			switch v := allowedSetting.(type) {
			case []any:
				// Convert array to string slice
				tools := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						tools = append(tools, str)
					}
				}
				return tools
			case []string:
				return v
			}
		}
	}
	return nil
}

func getGitHubDockerImageVersion(githubTool any) string {
	githubDockerImageVersion := string(constants.DefaultGitHubMCPServerVersion) // Default Docker image version
	// Extract version setting from tool properties
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["version"]; exists {
			// Handle different version types
			switch v := versionSetting.(type) {
			case string:
				githubDockerImageVersion = v
			case int:
				githubDockerImageVersion = fmt.Sprintf("%d", v)
			case int64:
				githubDockerImageVersion = fmt.Sprintf("%d", v)
			case uint64:
				githubDockerImageVersion = fmt.Sprintf("%d", v)
			case float64:
				// Use %g to avoid trailing zeros and scientific notation for simple numbers
				githubDockerImageVersion = fmt.Sprintf("%g", v)
			}
		}
	}
	return githubDockerImageVersion
}

// generateGitHubMCPLockdownDetectionStep generates a step to determine automatic lockdown mode
// for GitHub MCP server based on repository visibility. This step is added when:
// - GitHub tool is enabled AND
// - lockdown field is not explicitly specified in the workflow configuration
// The step includes a runtime condition that only executes if GH_AW_GITHUB_MCP_SERVER_TOKEN is defined
func (c *Compiler) generateGitHubMCPLockdownDetectionStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if GitHub tool is present
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub || githubTool == false {
		return
	}

	// Check if lockdown is already explicitly set
	if hasGitHubLockdownExplicitlySet(githubTool) {
		githubConfigLog.Print("Lockdown explicitly set in workflow, skipping automatic lockdown determination")
		return
	}

	githubConfigLog.Print("Generating automatic lockdown determination step for GitHub MCP server")

	// Resolve the latest version of actions/github-script
	actionRepo := "actions/github-script"
	actionVersion := string(constants.DefaultGitHubScriptVersion)
	pinnedAction, err := GetActionPinWithData(actionRepo, actionVersion, data)
	if err != nil {
		githubConfigLog.Printf("Failed to resolve %s@%s: %v", actionRepo, actionVersion, err)
		// In strict mode, this error would have been returned by GetActionPinWithData
		// In normal mode, we fall back to using the version tag without pinning
		pinnedAction = fmt.Sprintf("%s@%s", actionRepo, actionVersion)
	}

	// Generate the step using the determine_automatic_lockdown.cjs action
	// The step only runs if GH_AW_GITHUB_MCP_SERVER_TOKEN secret is defined
	yaml.WriteString("      - name: Determine automatic lockdown mode for GitHub MCP server\n")
	yaml.WriteString("        id: determine-automatic-lockdown\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          TOKEN_CHECK: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}\n")
	yaml.WriteString("        if: env.TOKEN_CHECK != ''\n")
	fmt.Fprintf(yaml, "        uses: %s\n", pinnedAction)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const determineAutomaticLockdown = require('/opt/gh-aw/actions/determine_automatic_lockdown.cjs');\n")
	yaml.WriteString("            await determineAutomaticLockdown(github, context, core);\n")
}
