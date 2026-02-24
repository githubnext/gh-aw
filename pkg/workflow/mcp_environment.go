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
//   - Handling safe-outputs and safe-inputs environment variables
//   - Processing Playwright domain secrets
//   - Extracting secrets from HTTP MCP server headers
//   - Managing agentic-workflows GITHUB_TOKEN
//
// Environment variable categories:
//   - GitHub MCP: GITHUB_MCP_SERVER_TOKEN, GITHUB_MCP_LOCKDOWN
//   - Safe Outputs: GH_AW_SAFE_OUTPUTS_*, GH_AW_ASSETS_*
//   - Safe Inputs: GH_AW_SAFE_INPUTS_PORT, GH_AW_SAFE_INPUTS_API_KEY
//   - Serena: GH_AW_SERENA_PORT (local mode only)
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
//   - safe_inputs.go: Safe inputs configuration
//
// Example usage:
//
//	envVars := collectMCPEnvironmentVariables(tools, mcpTools, workflowData, hasAgenticWorkflows)
//	// Returns map[string]string with all required environment variables
package workflow

import (
	"maps"

	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpEnvironmentLog = logger.New("workflow:mcp_environment")

// collectMCPEnvironmentVariables collects all MCP-related environment variables
// from the workflow configuration to be passed to both Start MCP gateway and MCP Gateway steps
func collectMCPEnvironmentVariables(tools map[string]any, mcpTools []string, workflowData *WorkflowData, hasAgenticWorkflows bool) map[string]string {
	envVars := make(map[string]string)

	// Check for GitHub MCP server token
	hasGitHub := slices.Contains(mcpTools, "github")
	if hasGitHub {
		githubTool := tools["github"]

		// Check if GitHub App is configured for token minting
		appConfigured := hasGitHubApp(githubTool)

		// If GitHub App is configured, use the app token (overrides other tokens)
		if appConfigured {
			mcpEnvironmentLog.Print("Using GitHub App token for GitHub MCP server (overrides custom and default tokens)")
			envVars["GITHUB_MCP_SERVER_TOKEN"] = "${{ steps.github-mcp-app-token.outputs.token }}"
		} else {
			// Otherwise, use custom token or default fallback
			customGitHubToken := getGitHubToken(githubTool)
			effectiveToken := getEffectiveGitHubToken(customGitHubToken)
			envVars["GITHUB_MCP_SERVER_TOKEN"] = effectiveToken
		}

		// Add lockdown value if it's determined from step output.
		// Skip when a GitHub App is configured — in that case, the determine-automatic-lockdown
		// step is not generated, so there is no step output to reference.
		// Security: Pass step output through environment variable to prevent template injection
		// Convert "true"/"false" to "1"/"0" at the source to avoid shell conversion in templates
		if !hasGitHubLockdownExplicitlySet(githubTool) && !appConfigured {
			envVars["GITHUB_MCP_LOCKDOWN"] = "${{ steps.determine-automatic-lockdown.outputs.lockdown == 'true' && '1' || '0' }}"
		}
	}

	// Check for safe-outputs env vars
	hasSafeOutputs := slices.Contains(mcpTools, "safe-outputs")
	if hasSafeOutputs {
		envVars["GH_AW_SAFE_OUTPUTS"] = "${{ env.GH_AW_SAFE_OUTPUTS }}"
		// Only add upload-assets env vars if upload-assets is configured
		if workflowData.SafeOutputs.UploadAssets != nil {
			envVars["GH_AW_ASSETS_BRANCH"] = "${{ env.GH_AW_ASSETS_BRANCH }}"
			envVars["GH_AW_ASSETS_MAX_SIZE_KB"] = "${{ env.GH_AW_ASSETS_MAX_SIZE_KB }}"
			envVars["GH_AW_ASSETS_ALLOWED_EXTS"] = "${{ env.GH_AW_ASSETS_ALLOWED_EXTS }}"
		}
	}

	// Check for safe-inputs env vars
	// Only add env vars if safe-inputs is actually enabled (has tools configured)
	// This prevents referencing step outputs that don't exist when safe-inputs isn't used
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		// Add server configuration env vars from step outputs
		envVars["GH_AW_SAFE_INPUTS_PORT"] = "${{ steps.safe-inputs-start.outputs.port }}"
		envVars["GH_AW_SAFE_INPUTS_API_KEY"] = "${{ steps.safe-inputs-start.outputs.api_key }}"

		// Add tool-specific env vars (secrets passthrough)
		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		maps.Copy(envVars, safeInputsSecrets)
	}

	// Check for safe-outputs env vars
	// Only add env vars if safe-outputs is actually enabled
	// This prevents referencing step outputs that don't exist when safe-outputs isn't used
	if workflowData != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		// Add server configuration env vars from step outputs
		envVars["GH_AW_SAFE_OUTPUTS_PORT"] = "${{ steps.safe-outputs-start.outputs.port }}"
		envVars["GH_AW_SAFE_OUTPUTS_API_KEY"] = "${{ steps.safe-outputs-start.outputs.api_key }}"
	}

	// Check if serena is in local mode and add its environment variables
	if workflowData != nil && isSerenaInLocalMode(workflowData.ParsedTools) {
		envVars["GH_AW_SERENA_PORT"] = "${{ steps.serena-config.outputs.serena_port }}"
	}

	// Check for agentic-workflows GITHUB_TOKEN
	if hasAgenticWorkflows {
		envVars["GITHUB_TOKEN"] = "${{ secrets.GITHUB_TOKEN }}"
	}

	// Check for Playwright domain secrets
	hasPlaywright := slices.Contains(mcpTools, "playwright")
	if hasPlaywright {
		// Extract all expressions from playwright custom args using ExpressionExtractor
		if playwrightTool, ok := tools["playwright"]; ok {
			playwrightConfig := parsePlaywrightTool(playwrightTool)
			customArgs := getPlaywrightCustomArgs(playwrightConfig)
			playwrightArgSecrets := extractExpressionsFromPlaywrightArgs(customArgs)
			maps.Copy(envVars, playwrightArgSecrets)
		}
	}

	// Check for HTTP MCP servers with secrets in headers (e.g., Tavily)
	// These need to be available as environment variables when the MCP gateway starts
	for toolName, toolValue := range tools {
		// Skip standard tools that are handled above
		if toolName == "github" || toolName == "playwright" || toolName == "serena" ||
			toolName == "cache-memory" || toolName == "agentic-workflows" ||
			toolName == "safe-outputs" || toolName == "safe-inputs" {
			continue
		}

		// Check if this is an MCP tool
		if toolConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, _ := hasMCPConfig(toolConfig); !hasMcp {
				continue
			}

			// Get MCP config and check if it's an HTTP type
			mcpConfig, err := getMCPConfig(toolConfig, toolName)
			if err != nil {
				mcpEnvironmentLog.Printf("Failed to parse MCP config for tool %s: %v", toolName, err)
				continue
			}

			// Extract secrets from headers for HTTP MCP servers
			if mcpConfig.Type == "http" && len(mcpConfig.Headers) > 0 {
				headerSecrets := ExtractSecretsFromMap(mcpConfig.Headers)
				mcpEnvironmentLog.Printf("Extracted %d secrets from HTTP MCP server '%s'", len(headerSecrets), toolName)
				maps.Copy(envVars, headerSecrets)
			}

			// Also extract secrets from env section if present
			if len(mcpConfig.Env) > 0 {
				envSecrets := ExtractSecretsFromMap(mcpConfig.Env)
				mcpEnvironmentLog.Printf("Extracted %d secrets from env section of MCP server '%s'", len(envSecrets), toolName)
				maps.Copy(envVars, envSecrets)
			}
		}
	}

	// Extract environment variables from plugin MCP configurations
	// Plugins can define MCP servers with environment variables that need to be available during gateway setup
	// We need to pass ALL env vars (not just secrets) since plugins may need configuration values
	if workflowData != nil && workflowData.PluginInfo != nil && len(workflowData.PluginInfo.MCPConfigs) > 0 {
		mcpEnvironmentLog.Printf("Extracting environment variables from %d plugin MCP configurations", len(workflowData.PluginInfo.MCPConfigs))
		for pluginID, mcpConfig := range workflowData.PluginInfo.MCPConfigs {
			if mcpConfig != nil && len(mcpConfig.Env) > 0 {
				mcpEnvironmentLog.Printf("Adding %d environment variables from plugin '%s' MCP configuration", len(mcpConfig.Env), pluginID)
				// Add ALL environment variables from plugin MCP config (not just secrets)
				maps.Copy(envVars, mcpConfig.Env)
			}
		}
	}

	return envVars
}
