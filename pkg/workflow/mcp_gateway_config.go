package workflow

import (
	"encoding/json"
	"fmt"
)

// MCPGatewayConfigTypes defines the Go types for MCP gateway configuration
// that comply with the MCP Gateway Specification v1.0.0
// See: docs/src/content/docs/reference/mcp-gateway.md

// GatewayMCPServerConfig represents a single MCP server configuration for the gateway
// Compliant with MCP Gateway Specification v1.0.0 section 4.1.2
type GatewayMCPServerConfig struct {
	// Container image for the MCP server (required for stdio servers)
	Container string `json:"container,omitempty"`

	// Optional entrypoint override for container (equivalent to docker run --entrypoint)
	Entrypoint string `json:"entrypoint,omitempty"`

	// Arguments passed to container entrypoint (container only)
	EntrypointArgs []string `json:"entrypointArgs,omitempty"`

	// Docker runtime arguments (passed before container image in docker run command)
	// These are NOT part of the MCP gateway spec, but are needed for the gateway implementation
	Args []string `json:"args,omitempty"`

	// Volume mounts for container (format: "source:dest:mode" where mode is "ro" or "rw")
	Mounts []string `json:"mounts,omitempty"`

	// Environment variables for the server process
	Env map[string]string `json:"env,omitempty"`

	// Transport type: "stdio" or "http" (default: "stdio")
	Type string `json:"type,omitempty"`

	// HTTP endpoint URL for HTTP servers (required for HTTP servers)
	URL string `json:"url,omitempty"`

	// HTTP headers for HTTP servers
	Headers map[string]string `json:"headers,omitempty"`

	// Tools field for Copilot compatibility (list of allowed tools or ["*"] for all)
	// This is NOT part of the MCP gateway spec, but required by GitHub Copilot CLI
	Tools []string `json:"tools,omitempty"`
}

// GatewayConfig represents the gateway section configuration
// Compliant with MCP Gateway Specification v1.0.0 section 4.1.3
type GatewayConfig struct {
	// HTTP server port (default: 8080)
	Port int `json:"port,omitempty"`

	// API key for authentication
	APIKey string `json:"apiKey,omitempty"`

	// Gateway domain (localhost or host.docker.internal)
	Domain string `json:"domain,omitempty"`

	// Server startup timeout in seconds (default: 30)
	StartupTimeout int `json:"startupTimeout,omitempty"`

	// Tool invocation timeout in seconds (default: 60)
	ToolTimeout int `json:"toolTimeout,omitempty"`
}

// GatewayMCPRootConfig represents the root MCP configuration structure
// Compliant with MCP Gateway Specification v1.0.0 section 4.1.1
type GatewayMCPRootConfig struct {
	// Map of server name to server configuration
	MCPServers map[string]*GatewayMCPServerConfig `json:"mcpServers"`

	// Optional gateway configuration
	Gateway *GatewayConfig `json:"gateway,omitempty"`
}

// BuildGatewayMCPServerConfig builds a GatewayMCPServerConfig from GitHubMCPDockerOptions
// This converts the current rendering options into the spec-compliant format
func BuildGatewayMCPServerConfig(options GitHubMCPDockerOptions) *GatewayMCPServerConfig {
	config := &GatewayMCPServerConfig{
		Container: "ghcr.io/github/github-mcp-server:" + options.DockerImageVersion,
		Type:      "stdio",
	}

	// Add Docker runtime args if present
	if len(options.CustomArgs) > 0 {
		config.Args = options.CustomArgs
	}

	// Build environment variables
	config.Env = make(map[string]string)

	// GitHub token (always required)
	if options.IncludeTypeField {
		// Copilot engine: use escaped variable for Copilot CLI to interpolate
		config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "\\${GITHUB_MCP_SERVER_TOKEN}"
	} else {
		// Non-Copilot engines (Claude/Custom): use plain shell variable
		config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "$GITHUB_MCP_SERVER_TOKEN"
	}

	// Read-only mode
	if options.ReadOnly {
		config.Env["GITHUB_READ_ONLY"] = "1"
	}

	// Lockdown mode
	if options.LockdownFromStep {
		// Security: Use environment variable instead of template expression to prevent template injection
		// The GITHUB_MCP_LOCKDOWN env var is set in Setup MCPs step from step output
		// Value is already converted to "1" or "0" in the environment variable
		config.Env["GITHUB_LOCKDOWN_MODE"] = "$GITHUB_MCP_LOCKDOWN"
	} else if options.Lockdown {
		// Use explicit lockdown value from configuration
		config.Env["GITHUB_LOCKDOWN_MODE"] = "1"
	}

	// Toolsets (always configured, defaults to "default")
	config.Env["GITHUB_TOOLSETS"] = options.Toolsets

	// Add tools field if needed (Copilot uses this, Claude doesn't)
	if len(options.AllowedTools) > 0 {
		config.Tools = options.AllowedTools
	} else if options.IncludeTypeField {
		// Copilot always includes tools field, even if empty (uses wildcard)
		config.Tools = []string{"*"}
	}

	return config
}

// BuildGatewayMCPServerConfigFromRemote builds a GatewayMCPServerConfig from GitHubMCPRemoteOptions
func BuildGatewayMCPServerConfigFromRemote(options GitHubMCPRemoteOptions) *GatewayMCPServerConfig {
	config := &GatewayMCPServerConfig{
		Type: "http",
		URL:  "https://api.githubcopilot.com/mcp/",
	}

	// Collect headers
	config.Headers = make(map[string]string)
	config.Headers["Authorization"] = options.AuthorizationValue

	// Add X-MCP-Readonly header if read-only mode is enabled
	if options.ReadOnly {
		config.Headers["X-MCP-Readonly"] = "true"
	}

	// Add X-MCP-Lockdown header if lockdown mode is enabled
	if options.LockdownFromStep {
		// Security: Use environment variable instead of template expression to prevent template injection
		// The GITHUB_MCP_LOCKDOWN env var contains "1" or "0", convert to "true" or "false" for header
		config.Headers["X-MCP-Lockdown"] = "$([ \"$GITHUB_MCP_LOCKDOWN\" = \"1\" ] && echo true || echo false)"
	} else if options.Lockdown {
		// Use explicit lockdown value from configuration
		config.Headers["X-MCP-Lockdown"] = "true"
	}

	// Add X-MCP-Toolsets header if toolsets are configured
	if options.Toolsets != "" {
		config.Headers["X-MCP-Toolsets"] = options.Toolsets
	}

	// Add tools field if needed (Copilot uses this, Claude doesn't)
	if options.IncludeToolsField {
		if len(options.AllowedTools) > 0 {
			config.Tools = options.AllowedTools
		} else {
			config.Tools = []string{"*"}
		}
	}

	// Add env section if needed (Copilot uses this, Claude doesn't)
	if options.IncludeEnvSection {
		config.Env = make(map[string]string)
		config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "\\${GITHUB_MCP_SERVER_TOKEN}"
	}

	return config
}

// BuildGatewayConfigFromRuntime builds a GatewayConfig from MCPGatewayRuntimeConfig
func BuildGatewayConfigFromRuntime(runtime *MCPGatewayRuntimeConfig) *GatewayConfig {
	if runtime == nil {
		return nil
	}

	return &GatewayConfig{
		Port:   runtime.Port,
		APIKey: runtime.APIKey,
		Domain: runtime.Domain,
	}
}

// RenderGatewayMCPConfigJSON renders a GatewayMCPRootConfig as JSON to a string builder
// This uses JSON marshaling instead of manual string building for spec compliance
func RenderGatewayMCPConfigJSON(config *GatewayMCPRootConfig) (string, error) {
	data, err := json.MarshalIndent(config, "          ", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP gateway config: %w", err)
	}
	
	// Adjust indentation to match the context where it will be used (cat > file << EOF)
	// The JSON is already indented with "          " (10 spaces) as base
	return string(data), nil
}
