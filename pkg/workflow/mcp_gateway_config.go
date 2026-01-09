package workflow

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
