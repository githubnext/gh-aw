package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var customMCPLog = logger.New("workflow:compiler-custom-mcp")

// generateCustomMCPServerAppTokenMintingSteps generates GitHub App token-mint steps
// for every custom MCP server that has a github-app block configured.
//
// For each such server, a step is emitted with:
//   - id:  <server-name>-mcp-app-token
//   - uses: actions/create-github-app-token
//   - with: app-id, private-key, owner, repositories, permission-* fields
//
// The minted token is later referenced as an env var in the MCP gateway container via
// customMCPServerAppTokenEnvVar(serverName) → MCP_<SERVER_NAME>_APP_TOKEN.
func (c *Compiler) generateCustomMCPServerAppTokenMintingSteps(yaml *strings.Builder, tools map[string]any, data *WorkflowData) {
	serverNames := sortedCustomMCPServerNames(tools)

	var permissions *Permissions
	if data.Permissions != "" {
		permissions = NewPermissionsParser(data.Permissions).ToPermissions()
	} else {
		permissions = NewPermissions()
	}

	for _, serverName := range serverNames {
		toolValue := tools[serverName]
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}
		app := parseCustomMCPGitHubApp(toolConfig)
		if app == nil {
			continue
		}

		customMCPLog.Printf("Generating GitHub App token mint step for custom MCP server: server=%s app-id=%s", serverName, app.AppID)

		steps := c.buildGitHubAppTokenMintStep(app, permissions, "")
		stepID := customMCPServerAppTokenStepID(serverName)

		for _, step := range steps {
			// Rewrite the default step ID (safe-outputs-app-token) to the server-specific ID
			modified := strings.ReplaceAll(step, "id: safe-outputs-app-token", "id: "+stepID)
			yaml.WriteString(modified)
		}
	}
}

// generateCustomMCPServerAppTokenInvalidationSteps generates GitHub App token-invalidation
// steps for every custom MCP server that has a github-app block configured.
//
// These steps always run (even on failure) to ensure tokens are revoked promptly.
func (c *Compiler) generateCustomMCPServerAppTokenInvalidationSteps(yaml *strings.Builder, tools map[string]any) {
	serverNames := sortedCustomMCPServerNames(tools)

	for _, serverName := range serverNames {
		toolValue := tools[serverName]
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}
		app := parseCustomMCPGitHubApp(toolConfig)
		if app == nil {
			continue
		}

		customMCPLog.Printf("Generating GitHub App token invalidation step for custom MCP server: server=%s", serverName)

		steps := c.buildGitHubAppTokenInvalidationStep()
		stepID := customMCPServerAppTokenStepID(serverName)

		for _, step := range steps {
			modified := strings.ReplaceAll(
				step,
				"steps.safe-outputs-app-token.outputs.token",
				fmt.Sprintf("steps.%s.outputs.token", stepID),
			)
			yaml.WriteString(modified)
		}
	}
}

// hasCustomMCPServerOIDCAuth returns true if any custom MCP server in tools has
// auth.type == "github-oidc" configured.
func hasCustomMCPServerOIDCAuth(tools map[string]any) bool {
	for _, toolValue := range tools {
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}
		hasMCP, mcpType := hasMCPConfig(toolConfig)
		if !hasMCP || mcpType != "http" {
			continue
		}
		if authRaw, exists := toolConfig["auth"]; exists {
			if authMap, ok := authRaw.(map[string]any); ok {
				if authType, ok := authMap["type"].(string); ok && authType == "github-oidc" {
					return true
				}
			}
		}
	}
	return false
}

// sortedCustomMCPServerNames returns the names of custom MCP server tools sorted alphabetically.
// Built-in tool names (github, playwright, etc.) are excluded.
func sortedCustomMCPServerNames(tools map[string]any) []string {
	builtin := map[string]bool{
		"github":            true,
		"playwright":        true,
		"cache-memory":      true,
		"repo-memory":       true,
		"agentic-workflows": true,
		"safe-outputs":      true,
		"mcp-scripts":       true,
		"bash":              true,
		"web-fetch":         true,
		"web-search":        true,
		"edit":              true,
		"qmd":               true,
		"timeout":           true,
		"startup-timeout":   true,
	}

	var names []string
	for name, toolValue := range tools {
		if builtin[name] {
			continue
		}
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}
		if hasMCP, _ := hasMCPConfig(toolConfig); !hasMCP {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
