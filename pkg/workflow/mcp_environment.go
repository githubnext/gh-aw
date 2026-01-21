package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpEnvironmentLog = logger.New("workflow:mcp_environment")

// collectMCPEnvironmentVariables collects all MCP-related environment variables
// from the workflow configuration to be passed to both Start MCP gateway and MCP Gateway steps
func collectMCPEnvironmentVariables(tools map[string]any, mcpTools []string, workflowData *WorkflowData, hasAgenticWorkflows bool) map[string]string {
	envVars := make(map[string]string)

	// Check for GitHub MCP server token
	hasGitHub := false
	for _, toolName := range mcpTools {
		if toolName == "github" {
			hasGitHub = true
			break
		}
	}
	if hasGitHub {
		githubTool := tools["github"]
		customGitHubToken := getGitHubToken(githubTool)
		effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
		envVars["GITHUB_MCP_SERVER_TOKEN"] = effectiveToken

		// Add lockdown value if it's determined from step output
		// Security: Pass step output through environment variable to prevent template injection
		// Convert "true"/"false" to "1"/"0" at the source to avoid shell conversion in templates
		if !hasGitHubLockdownExplicitlySet(githubTool) {
			envVars["GITHUB_MCP_LOCKDOWN"] = "${{ steps.determine-automatic-lockdown.outputs.lockdown == 'true' && '1' || '0' }}"
		}
	}

	// Check for safe-outputs env vars
	hasSafeOutputs := false
	for _, toolName := range mcpTools {
		if toolName == "safe-outputs" {
			hasSafeOutputs = true
			break
		}
	}
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
		for envVarName, secretExpr := range safeInputsSecrets {
			envVars[envVarName] = secretExpr
		}
	}

	// Check for safe-outputs env vars
	// Only add env vars if safe-outputs is actually enabled
	// This prevents referencing step outputs that don't exist when safe-outputs isn't used
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
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
	hasPlaywright := false
	for _, toolName := range mcpTools {
		if toolName == "playwright" {
			hasPlaywright = true
			break
		}
	}
	if hasPlaywright {
		// Extract all expressions from playwright arguments using ExpressionExtractor
		if playwrightTool, ok := tools["playwright"]; ok {
			playwrightConfig := parsePlaywrightTool(playwrightTool)
			allowedDomains := generatePlaywrightAllowedDomains(playwrightConfig)
			customArgs := getPlaywrightCustomArgs(playwrightConfig)
			playwrightAllowedDomainsSecrets := extractExpressionsFromPlaywrightArgs(allowedDomains, customArgs)
			for envVarName, originalExpr := range playwrightAllowedDomainsSecrets {
				envVars[envVarName] = originalExpr
			}
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
				for envVarName, secretExpr := range headerSecrets {
					envVars[envVarName] = secretExpr
				}
			}

			// Also extract secrets from env section if present
			if len(mcpConfig.Env) > 0 {
				envSecrets := ExtractSecretsFromMap(mcpConfig.Env)
				mcpEnvironmentLog.Printf("Extracted %d secrets from env section of MCP server '%s'", len(envSecrets), toolName)
				for envVarName, secretExpr := range envSecrets {
					envVars[envVarName] = secretExpr
				}
			}
		}
	}

	return envVars
}
