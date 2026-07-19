// Package workflow provides environment variable management for MCP server execution.
//
// # MCP Environment Variables
//
// This file is responsible for collecting and managing all environment variables
// required by MCP servers during workflow execution. Environment variables are
// used to pass configuration, authentication tokens, and runtime settings to
// MCP servers running in the gateway.
//
// Key responsibilities:
//   - Collecting MCP-related environment variables from workflow configuration
//   - Managing GitHub MCP server tokens (custom, default, and GitHub App tokens)
//   - Handling safe-outputs and mcp-scripts environment variables
//   - Processing Playwright domain secrets
//   - Extracting secrets from HTTP MCP server headers
//   - Managing agentic-workflows GITHUB_TOKEN
//
// Environment variable categories:
//   - GitHub MCP: GITHUB_MCP_SERVER_TOKEN, GITHUB_MCP_GUARD_MIN_INTEGRITY, GITHUB_MCP_GUARD_REPOS
//   - Safe Outputs: GH_AW_SAFE_OUTPUTS_*, GH_AW_ASSETS_*
//   - MCP Scripts: GH_AW_MCP_SCRIPTS_PORT, GH_AW_MCP_SCRIPTS_API_KEY
//   - Serena: removed (use shared/mcp/serena.md instead)
//   - Playwright: Secrets from custom args expressions
//   - HTTP MCP: Custom secrets from headers and env sections
//
// Token precedence for GitHub MCP:
//  1. GitHub App token (if app configuration exists)
//  2. Custom github-token from tool configuration
//  3. Top-level github-token from frontmatter
//  4. Default GITHUB_TOKEN secret
//
// The environment variables collected here are passed to both the
// "Start MCP gateway" step and the "MCP Gateway" step to ensure
// MCP servers have access to necessary configuration and secrets.
//
// Related files:
//   - mcp_setup_generator.go: Uses collected env vars in gateway setup
//   - mcp_github_config.go: GitHub-specific token and configuration
//   - safe_outputs.go: Safe outputs configuration
//   - mcp_scripts.go: MCP Scripts configuration
//
// Example usage:
//
//	envVars := collectMCPEnvironmentVariables(tools, mcpTools, workflowData, hasAgenticWorkflows)
//	// Returns map[string]string with all required environment variables
package workflow

import (
	"fmt"
	"maps"

	"slices"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var mcpEnvironmentLog = logger.New("workflow:mcp_environment")

// collectMCPEnvironmentVariables collects all MCP-related environment variables
// from the workflow configuration to be passed to both Start MCP gateway and MCP Gateway steps
func collectMCPEnvironmentVariables(tools map[string]any, mcpTools []string, workflowData *WorkflowData, hasAgenticWorkflows bool) map[string]string {
	envVars := make(map[string]string)

	collectGitHubMCPEnv(envVars, tools, mcpTools)
	collectSafeOutputsMCPEnv(envVars, mcpTools, workflowData)
	collectMCPScriptsEnv(envVars, workflowData)

	// Check for agentic-workflows GITHUB_TOKEN
	if hasAgenticWorkflows {
		envVars["GITHUB_TOKEN"] = "${{ secrets.GITHUB_TOKEN }}"
	}

	collectPlaywrightMCPEnv(envVars, tools, mcpTools)
	collectHTTPMCPServerEnv(envVars, tools)
	collectCodexMCPEnv(envVars, workflowData)

	return envVars
}

func collectGitHubMCPEnv(envVars map[string]string, tools map[string]any, mcpTools []string) {
	if !slices.Contains(mcpTools, "github") {
		return
	}
	toolConfig, _ := tools["github"].(map[string]any)
	if hasGitHubApp(toolConfig) {
		envVars["GITHUB_MCP_SERVER_TOKEN"] = githubMCPAppTokenExpression(toolConfig)
	} else {
		customGitHubToken := getGitHubToken(toolConfig)
		envVars["GITHUB_MCP_SERVER_TOKEN"] = getEffectiveGitHubToken(customGitHubToken)
	}
	if len(getGitHubGuardPolicies(toolConfig)) == 0 {
		envVars["GITHUB_MCP_GUARD_MIN_INTEGRITY"] = "${{ steps.determine-automatic-lockdown.outputs.min_integrity }}"
		envVars["GITHUB_MCP_GUARD_REPOS"] = "${{ steps.determine-automatic-lockdown.outputs.repos }}"
	}
}

func githubMCPAppTokenExpression(toolConfig map[string]any) string {
	mcpEnvironmentLog.Print("Using GitHub App token from agent job step for GitHub MCP server (overrides custom and default tokens)")
	tokenExpression := "${{ steps.github-mcp-app-token.outputs.token }}"
	if appMap, ok := toolConfig["github-app"].(map[string]any); ok {
		if appConfig := parseAppConfig(appMap); appConfig.shouldIgnoreMissingKey() {
			customGitHubToken := getGitHubToken(toolConfig)
			tokenExpression = combineTokenExpressions(tokenExpression, getEffectiveGitHubToken(customGitHubToken))
		}
	}
	return tokenExpression
}

func collectSafeOutputsMCPEnv(envVars map[string]string, mcpTools []string, workflowData *WorkflowData) {
	if !slices.Contains(mcpTools, "safe-outputs") {
		return
	}
	envVars["GH_AW_SAFE_OUTPUTS"] = "${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}"
	envVars["GH_AW_SAFE_OUTPUTS_CONFIG_PATH"] = "${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS_CONFIG_PATH }}"
	envVars["GH_AW_SAFE_OUTPUTS_TOOLS_PATH"] = "${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS_TOOLS_PATH }}"
	envVars[compilerenv.PolicyAllowCreatePullRequest] = fmt.Sprintf("${{ vars.%s || 'true' }}", compilerenv.PolicyAllowCreatePullRequest)
	if _, ok := envVars["GITHUB_TOKEN"]; !ok {
		envVars["GITHUB_TOKEN"] = "${{ secrets.GITHUB_TOKEN }}"
	}
	if workflowData.SafeOutputs.UploadAssets != nil {
		envVars["GH_AW_ASSETS_BRANCH"] = "${{ env.GH_AW_ASSETS_BRANCH }}"
		envVars["GH_AW_ASSETS_MAX_SIZE_KB"] = "${{ env.GH_AW_ASSETS_MAX_SIZE_KB }}"
		envVars["GH_AW_ASSETS_ALLOWED_EXTS"] = "${{ env.GH_AW_ASSETS_ALLOWED_EXTS }}"
	}
}

func collectMCPScriptsEnv(envVars map[string]string, workflowData *WorkflowData) {
	if !IsMCPScriptsEnabled(workflowData.MCPScripts) {
		return
	}
	envVars["GH_AW_MCP_SCRIPTS_PORT"] = "${{ steps.mcp-scripts-start.outputs.port }}"
	envVars["GH_AW_MCP_SCRIPTS_API_KEY"] = "${{ steps.mcp-scripts-start.outputs.api_key }}"
	maps.Copy(envVars, collectMCPScriptsSecrets(workflowData.MCPScripts))
}

func collectPlaywrightMCPEnv(envVars map[string]string, tools map[string]any, mcpTools []string) {
	if !slices.Contains(mcpTools, "playwright") {
		return
	}
	if playwrightTool, ok := tools["playwright"]; ok {
		playwrightConfig := parsePlaywrightTool(playwrightTool)
		customArgs := getPlaywrightCustomArgs(playwrightConfig)
		maps.Copy(envVars, extractExpressionsFromPlaywrightArgs(customArgs))
	}
}

func collectHTTPMCPServerEnv(envVars map[string]string, tools map[string]any) {
	for toolName, toolValue := range tools {
		if isStandardMCPEnvironmentTool(toolName) {
			continue
		}
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}
		collectCustomMCPServerEnv(envVars, toolName, toolConfig)
	}
}

func isStandardMCPEnvironmentTool(toolName string) bool {
	return toolName == "github" || toolName == "playwright" ||
		toolName == "cache-memory" || toolName == "agentic-workflows" ||
		toolName == "safe-outputs" || toolName == "mcp-scripts"
}

func collectCustomMCPServerEnv(envVars map[string]string, toolName string, toolConfig map[string]any) {
	if hasMcp, _ := hasMCPConfig(toolConfig); !hasMcp {
		return
	}
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		mcpEnvironmentLog.Printf("Failed to parse MCP config for tool %s: %v", toolName, err)
		return
	}
	if mcpConfig.Type == "http" && len(mcpConfig.Headers) > 0 {
		headerSecrets := ExtractSecretsFromMap(mcpConfig.Headers)
		mcpEnvironmentLog.Printf("Extracted %d secrets from HTTP MCP server '%s'", len(headerSecrets), toolName)
		maps.Copy(envVars, headerSecrets)
	}
	collectCustomMCPEnvSection(envVars, toolName, mcpConfig)
}

func collectCustomMCPEnvSection(envVars map[string]string, toolName string, mcpConfig *parser.RegistryMCPServerConfig) {
	if len(mcpConfig.Env) == 0 {
		return
	}
	envSecrets := ExtractSecretsFromMap(mcpConfig.Env)
	mcpEnvironmentLog.Printf("Extracted %d secrets from env section of MCP server '%s'", len(envSecrets), toolName)
	maps.Copy(envVars, envSecrets)
	envExprs := ExtractEnvExpressionsFromMap(mcpConfig.Env)
	mcpEnvironmentLog.Printf("Extracted %d env expressions from env section of MCP server '%s'", len(envExprs), toolName)
	maps.Copy(envVars, envExprs)
}

func collectCodexMCPEnv(envVars map[string]string, workflowData *WorkflowData) {
	if workflowData != nil && workflowData.AI == string(constants.CodexEngine) {
		envVars["CODEX_HOME"] = constants.TmpMcpConfigDir
	}
}

// hasGitHubOIDCAuthInTools checks if any HTTP MCP server in the tools configuration
// uses auth.type: "github-oidc". This is used to determine whether the OIDC env vars
// (ACTIONS_ID_TOKEN_REQUEST_URL, ACTIONS_ID_TOKEN_REQUEST_TOKEN) need to be forwarded
// to the MCP gateway container.
func hasGitHubOIDCAuthInTools(tools map[string]any) bool {
	for toolName, toolValue := range tools {
		// Skip standard tools that don't support auth config
		if toolName == "github" || toolName == "playwright" ||
			toolName == "cache-memory" || toolName == "agentic-workflows" ||
			toolName == "safe-outputs" || toolName == "mcp-scripts" {
			continue
		}

		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}

		hasMcp, _ := hasMCPConfig(toolConfig)
		if !hasMcp {
			continue
		}

		mcpConfig, err := getMCPConfig(toolConfig, toolName)
		if err != nil {
			continue
		}

		if mcpConfig.Type == "http" && mcpConfig.Auth != nil && mcpConfig.Auth.Type == "github-oidc" {
			mcpEnvironmentLog.Printf("Found github-oidc auth on HTTP MCP server '%s'", toolName)
			return true
		}
	}
	return false
}
