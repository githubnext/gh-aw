package workflow

const (
	// DefaultMCPGatewayPort is the default port for the MCP gateway (gh-aw-mcpg)
	// Port 80 is used for host.docker.internal access from AWF containers
	DefaultMCPGatewayPort = 80

	// DefaultMCPGatewayImage is the Docker image for gh-aw-mcpg
	// IMPORTANT: Always use a pinned version, NEVER use "latest"
	DefaultMCPGatewayImage = "ghcr.io/githubnext/gh-aw-mcpg"

	// DefaultMCPGatewayVersion is the pinned version of gh-aw-mcpg
	// MUST be updated when releasing new gh-aw-mcpg versions
	DefaultMCPGatewayVersion = "v0.1.0"

	// DefaultGatewaySessionToken is the default Bearer token for MCP client authentication
	DefaultGatewaySessionToken = "awf-session"

	// MCPGatewayLogsFolder is the folder where gateway logs are stored
	MCPGatewayLogsFolder = "/tmp/gh-aw/mcp-gateway-logs"

	// MCPGatewayContainerName is the name of the gh-aw-mcpg Docker container
	MCPGatewayContainerName = "gh-aw-mcpg"

	// MCPGatewayContainerPort is the internal port in the gh-aw-mcpg container
	MCPGatewayContainerPort = 8000
)
