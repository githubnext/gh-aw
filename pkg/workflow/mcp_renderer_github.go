package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
)

// RenderGitHubMCP generates the GitHub MCP server configuration
// Supports both local (Docker) and remote (hosted) modes
func (r *MCPConfigRendererUnified) RenderGitHubMCP(yaml *strings.Builder, githubTool any, workflowData *WorkflowData) {
	githubType := getGitHubType(githubTool)
	readOnly := getGitHubReadOnly(githubTool)

	// Get explicit lockdown value (only used when lockdown is explicitly configured)
	lockdown := getGitHubLockdown(githubTool)

	// Guard policies from step: automatically applied for public repositories when no explicit
	// guard policy is configured and no GitHub App token is in use.
	// The determine-automatic-lockdown step outputs min_integrity and repos for public repos.
	explicitGuardPolicies := getGitHubGuardPolicies(githubTool)
	shouldUseStepOutputForGuardPolicy := len(explicitGuardPolicies) == 0 && !hasGitHubApp(githubTool)

	toolsets := getGitHubToolsets(githubTool)

	mcpRendererLog.Printf("Rendering GitHub MCP: type=%s, read_only=%t, lockdown=%t (explicit=%t), guard_from_step=%t, toolsets=%v, format=%s",
		githubType, readOnly, lockdown, hasGitHubLockdownExplicitlySet(githubTool), shouldUseStepOutputForGuardPolicy, toolsets, r.options.Format)

	if r.options.Format == "toml" {
		r.renderGitHubTOML(yaml, githubTool, workflowData)
		return
	}

	yaml.WriteString("              \"github\": {\n")

	// Check if remote mode is enabled (type: remote)
	if githubType == "remote" {
		// Determine authorization value based on engine requirements
		// Copilot uses MCP passthrough syntax: "Bearer \${GITHUB_PERSONAL_ACCESS_TOKEN}"
		// Other engines use shell variable: "Bearer $GITHUB_MCP_SERVER_TOKEN"
		authValue := "Bearer $GITHUB_MCP_SERVER_TOKEN"
		if r.options.IncludeCopilotFields {
			authValue = "Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}"
		}

		RenderGitHubMCPRemoteConfig(yaml, GitHubMCPRemoteOptions{
			ReadOnly:              readOnly,
			Lockdown:              lockdown,
			LockdownFromStep:      false,
			GuardPoliciesFromStep: shouldUseStepOutputForGuardPolicy,
			Toolsets:              toolsets,
			AuthorizationValue:    authValue,
			IncludeToolsField:     r.options.IncludeCopilotFields,
			AllowedTools:          getGitHubAllowedTools(githubTool),
			IncludeEnvSection:     r.options.IncludeCopilotFields,
			GuardPolicies:         explicitGuardPolicies,
		})
	} else {
		// Local mode - use Docker-based GitHub MCP server (default)
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)
		mounts := getGitHubMounts(githubTool)

		RenderGitHubMCPDockerConfig(yaml, GitHubMCPDockerOptions{
			ReadOnly:              readOnly,
			Lockdown:              lockdown,
			LockdownFromStep:      false,
			GuardPoliciesFromStep: shouldUseStepOutputForGuardPolicy,
			Toolsets:              toolsets,
			DockerImageVersion:    githubDockerImageVersion,
			CustomArgs:            customArgs,
			Mounts:                mounts,
			IncludeTypeField:      r.options.IncludeCopilotFields,
			AllowedTools:          getGitHubAllowedTools(githubTool),
			EffectiveToken:        "", // Token passed via env
			GuardPolicies:         explicitGuardPolicies,
		})
	}

	if r.options.IsLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderGitHubTOML generates GitHub MCP configuration in TOML format (for Codex engine)
func (r *MCPConfigRendererUnified) renderGitHubTOML(yaml *strings.Builder, githubTool any, workflowData *WorkflowData) {
	githubType := getGitHubType(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	lockdown := getGitHubLockdown(githubTool)
	toolsets := getGitHubToolsets(githubTool)

	mcpRendererLog.Printf("Rendering GitHub MCP TOML: type=%s, read_only=%t, lockdown=%t, toolsets=%s", githubType, readOnly, lockdown, toolsets)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.github]\n")

	// Add user_agent field defaulting to workflow identifier
	userAgent := "github-agentic-workflow"
	if workflowData != nil {
		// Check if user_agent is configured in engine config first
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.UserAgent != "" {
			userAgent = workflowData.EngineConfig.UserAgent
		} else if workflowData.Name != "" {
			// Fall back to sanitizing workflow name to identifier
			userAgent = SanitizeIdentifier(workflowData.Name)
		}
	}
	yaml.WriteString("          user_agent = \"" + userAgent + "\"\n")

	// Use tools.startup-timeout if specified, otherwise default to DefaultMCPStartupTimeout
	startupTimeout := int(constants.DefaultMCPStartupTimeout / time.Second)
	if workflowData != nil && workflowData.ToolsStartupTimeout > 0 {
		startupTimeout = workflowData.ToolsStartupTimeout
	}
	fmt.Fprintf(yaml, "          startup_timeout_sec = %d\n", startupTimeout)

	// Use tools.timeout if specified, otherwise default to DefaultToolTimeout
	toolTimeout := int(constants.DefaultToolTimeout / time.Second)
	if workflowData != nil && workflowData.ToolsTimeout > 0 {
		toolTimeout = workflowData.ToolsTimeout
	}
	fmt.Fprintf(yaml, "          tool_timeout_sec = %d\n", toolTimeout)

	// Check if remote mode is enabled
	if githubType == "remote" {
		// Remote mode - use hosted GitHub MCP server with streamable HTTP
		// Use readonly endpoint if read-only mode is enabled
		if readOnly {
			yaml.WriteString("          url = \"https://api.githubcopilot.com/mcp-readonly/\"\n")
		} else {
			yaml.WriteString("          url = \"https://api.githubcopilot.com/mcp/\"\n")
		}

		// Use bearer_token_env_var for authentication
		yaml.WriteString("          bearer_token_env_var = \"GH_AW_GITHUB_TOKEN\"\n")
	} else {
		// Local mode - use Docker-based GitHub MCP server with MCP Gateway spec format
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)
		mounts := getGitHubMounts(githubTool)

		// MCP Gateway spec fields for containerized stdio servers
		yaml.WriteString("          container = \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"\n")

		// Append custom args if present (these are Docker runtime args, go before container image)
		if len(customArgs) > 0 {
			yaml.WriteString("          args = [\n")
			for _, arg := range customArgs {
				yaml.WriteString("            " + strconv.Quote(arg) + ",\n")
			}
			yaml.WriteString("          ]\n")
		}

		// Add volume mounts if present
		if len(mounts) > 0 {
			yaml.WriteString("          mounts = [")
			for i, mount := range mounts {
				if i > 0 {
					yaml.WriteString(", ")
				}
				yaml.WriteString(strconv.Quote(mount))
			}
			yaml.WriteString("]\n")
		}

		// Build environment variables
		envVars := make(map[string]string)
		envVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = "$GH_AW_GITHUB_TOKEN"
		// GitHub host for enterprise deployments (format: https://hostname, e.g. https://myorg.ghe.com).
		// GITHUB_SERVER_URL is set by GitHub Actions as a full URL (https://hostname, no trailing slash),
		// which matches the format expected by github-mcp-server for GITHUB_HOST.
		envVars["GITHUB_HOST"] = "$GITHUB_SERVER_URL"

		if readOnly {
			envVars["GITHUB_READ_ONLY"] = "1"
		}

		if lockdown {
			envVars["GITHUB_LOCKDOWN_MODE"] = "1"
		}

		envVars["GITHUB_TOOLSETS"] = toolsets

		// Write environment variables in sorted order for deterministic output
		envKeys := sortedMapKeys(envVars)

		yaml.WriteString("          env = { ")
		for i, key := range envKeys {
			if i > 0 {
				yaml.WriteString(", ")
			}
			fmt.Fprintf(yaml, "\"%s\" = \"%s\"", key, envVars[key])
		}
		yaml.WriteString(" }\n")

		// Use env_vars array to reference environment variables
		yaml.WriteString("          env_vars = [")
		for i, key := range envKeys {
			if i > 0 {
				yaml.WriteString(", ")
			}
			fmt.Fprintf(yaml, "\"%s\"", key)
		}
		yaml.WriteString("]\n")
	}
}

// RenderGitHubMCPDockerConfig renders the GitHub MCP server configuration for Docker (local mode).
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypointArgs, and env fields.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - options: GitHub MCP Docker rendering options
func RenderGitHubMCPDockerConfig(yaml *strings.Builder, options GitHubMCPDockerOptions) {
	mcpRendererLog.Printf("Rendering GitHub MCP Docker config: image=%s, read_only=%t, lockdown=%t", options.DockerImageVersion, options.ReadOnly, options.Lockdown)

	// Add type field if needed (Copilot requires this, Claude doesn't)
	// Per MCP Gateway Specification v1.0.0 section 4.1.2, use "stdio" for containerized servers
	if options.IncludeTypeField {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	// MCP Gateway spec fields for containerized stdio servers
	yaml.WriteString("                \"container\": \"ghcr.io/github/github-mcp-server:" + options.DockerImageVersion + "\",\n")

	// Append custom args if present (these are Docker runtime args, go before container image)
	if len(options.CustomArgs) > 0 {
		yaml.WriteString("                \"args\": [\n")
		for _, arg := range options.CustomArgs {
			quotedArg, _ := json.Marshal(arg)
			yaml.WriteString("                  " + string(quotedArg) + ",\n")
		}
		yaml.WriteString("                ],\n")
	}

	// Add volume mounts if present
	if len(options.Mounts) > 0 {
		yaml.WriteString("                \"mounts\": [\n")
		for i, mount := range options.Mounts {
			quotedMount, _ := json.Marshal(mount)
			yaml.WriteString("                  " + string(quotedMount))
			if i < len(options.Mounts)-1 {
				yaml.WriteString(",")
			}
			yaml.WriteString("\n")
		}
		yaml.WriteString("                ],\n")
	}

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot (see convert_gateway_config_copilot.sh). This keeps the gateway
	// config compatible with the schema which doesn't have the tools field.

	// Add env section for GitHub MCP server environment variables
	yaml.WriteString("                \"env\": {\n")

	// Build environment variables map
	envVars := make(map[string]string)

	// GitHub token (always required)
	if options.IncludeTypeField {
		// Copilot engine: use escaped variable for Copilot CLI to interpolate
		envVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = "\\${GITHUB_MCP_SERVER_TOKEN}"
		// GitHub host for enterprise deployments (format: https://hostname, e.g. https://myorg.ghe.com).
		// GITHUB_SERVER_URL is set by GitHub Actions as a full URL (https://hostname, no trailing slash),
		// which matches the format expected by github-mcp-server for GITHUB_HOST.
		// Copilot CLI interpolation syntax used here.
		envVars["GITHUB_HOST"] = "\\${GITHUB_SERVER_URL}"
	} else {
		// Non-Copilot engines (Claude/Custom): use plain shell variable
		envVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = "$GITHUB_MCP_SERVER_TOKEN"
		// GitHub host for enterprise deployments (format: https://hostname, e.g. https://myorg.ghe.com).
		// GITHUB_SERVER_URL is set by GitHub Actions as a full URL (https://hostname, no trailing slash),
		// which matches the format expected by github-mcp-server for GITHUB_HOST.
		envVars["GITHUB_HOST"] = "$GITHUB_SERVER_URL"
	}

	// Read-only mode
	if options.ReadOnly {
		envVars["GITHUB_READ_ONLY"] = "1"
	}

	// GitHub lockdown mode (only when explicitly configured)
	if options.Lockdown {
		// Use explicit lockdown value from configuration
		envVars["GITHUB_LOCKDOWN_MODE"] = "1"
	}

	// Toolsets (always configured, defaults to "default")
	envVars["GITHUB_TOOLSETS"] = options.Toolsets

	// Write environment variables in sorted order for deterministic output
	envKeys := sortedMapKeys(envVars)

	for i, key := range envKeys {
		isLast := i == len(envKeys)-1
		comma := ""
		if !isLast {
			comma = ","
		}
		fmt.Fprintf(yaml, "                  \"%s\": \"%s\"%s\n", key, envVars[key], comma)
	}

	// Close env section, with trailing comma if guard-policies follows
	hasGuardPolicies := len(options.GuardPolicies) > 0 || options.GuardPoliciesFromStep
	if hasGuardPolicies {
		yaml.WriteString("                },\n")
		if options.GuardPoliciesFromStep {
			// Render guard-policies with env var refs resolved at runtime from step outputs
			// GITHUB_MCP_GUARD_MIN_INTEGRITY and GITHUB_MCP_GUARD_REPOS are set in Start MCP
			// Gateway step from the determine-automatic-lockdown step outputs. They are
			// non-empty only for public repositories.
			renderGuardPoliciesJSON(yaml, map[string]any{
				"allow-only": map[string]any{
					"min-integrity": "$GITHUB_MCP_GUARD_MIN_INTEGRITY",
					"repos":         "$GITHUB_MCP_GUARD_REPOS",
				},
			}, "                ")
		} else {
			renderGuardPoliciesJSON(yaml, options.GuardPolicies, "                ")
		}
	} else {
		yaml.WriteString("                }\n")
	}
}

// RenderGitHubMCPRemoteConfig renders the GitHub MCP server configuration for remote (hosted) mode.
// This shared function extracts the duplicate pattern from Claude and Copilot engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - options: GitHub MCP remote rendering options
func RenderGitHubMCPRemoteConfig(yaml *strings.Builder, options GitHubMCPRemoteOptions) {
	mcpRendererLog.Printf("Rendering GitHub MCP remote config: read_only=%t, lockdown=%t, toolsets=%s", options.ReadOnly, options.Lockdown, options.Toolsets)

	// Remote mode - use hosted GitHub MCP server
	yaml.WriteString("                \"type\": \"http\",\n")
	yaml.WriteString("                \"url\": \"https://api.githubcopilot.com/mcp/\",\n")
	yaml.WriteString("                \"headers\": {\n")

	// Collect headers in a map
	headers := make(map[string]string)
	headers["Authorization"] = options.AuthorizationValue

	// Add X-MCP-Readonly header if read-only mode is enabled
	if options.ReadOnly {
		headers["X-MCP-Readonly"] = "true"
	}

	// Add X-MCP-Lockdown header only when explicitly configured
	if options.Lockdown {
		// Use explicit lockdown value from configuration
		headers["X-MCP-Lockdown"] = "true"
	}

	// Add X-MCP-Toolsets header if toolsets are configured
	if options.Toolsets != "" {
		headers["X-MCP-Toolsets"] = options.Toolsets
	}

	// Write headers using helper
	writeHeadersToYAML(yaml, headers, "                  ")

	// Determine if guard-policies section follows (explicit or from step)
	hasGuardPolicies := len(options.GuardPolicies) > 0 || options.GuardPoliciesFromStep

	// Close headers section
	if options.IncludeToolsField || options.IncludeEnvSection || hasGuardPolicies {
		yaml.WriteString("                },\n")
	} else {
		yaml.WriteString("                }\n")
	}

	// Add tools field if requested (Copilot needs it, Claude doesn't)
	// Note: This is added here when IncludeToolsField is true, but in some cases
	// the converter script also adds it back (see convert_gateway_config_copilot.sh).
	if options.IncludeToolsField && len(options.AllowedTools) > 0 {
		yaml.WriteString("                \"tools\": [\n")
		for i, tool := range options.AllowedTools {
			yaml.WriteString("                  \"")
			yaml.WriteString(tool)
			yaml.WriteString("\"")
			if i < len(options.AllowedTools)-1 {
				yaml.WriteString(",")
			}
			yaml.WriteString("\n")
		}
		if options.IncludeEnvSection || hasGuardPolicies {
			yaml.WriteString("                ],\n")
		} else {
			yaml.WriteString("                ]\n")
		}
	}

	// Add env section if needed (Copilot uses this, Claude doesn't)
	if options.IncludeEnvSection {
		yaml.WriteString("                \"env\": {\n")
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"\\${GITHUB_MCP_SERVER_TOKEN}\",\n")
		// GitHub host for enterprise deployments (format: https://hostname, e.g. https://myorg.ghe.com).
		// GITHUB_SERVER_URL is set by GitHub Actions as a full URL (https://hostname, no trailing slash),
		// which matches the format expected by github-mcp-server for GITHUB_HOST.
		yaml.WriteString("                  \"GITHUB_HOST\": \"\\${GITHUB_SERVER_URL}\"\n")
		// Close env section, with trailing comma if guard-policies follows
		if hasGuardPolicies {
			yaml.WriteString("                },\n")
		} else {
			yaml.WriteString("                }\n")
		}
	}

	// Add guard-policies if configured or from step
	if options.GuardPoliciesFromStep {
		// Render guard-policies with env var refs resolved at runtime from step outputs
		// GITHUB_MCP_GUARD_MIN_INTEGRITY and GITHUB_MCP_GUARD_REPOS are set in Start MCP
		// Gateway step from the determine-automatic-lockdown step outputs. They are
		// non-empty only for public repositories.
		renderGuardPoliciesJSON(yaml, map[string]any{
			"allow-only": map[string]any{
				"min-integrity": "$GITHUB_MCP_GUARD_MIN_INTEGRITY",
				"repos":         "$GITHUB_MCP_GUARD_REPOS",
			},
		}, "                ")
	} else if len(options.GuardPolicies) > 0 {
		renderGuardPoliciesJSON(yaml, options.GuardPolicies, "                ")
	}
}
