package workflow

const (
	// DefaultMCPGatewayHostPort is the default port exposed on the host for the MCP gateway
	DefaultMCPGatewayHostPort = 80

	// DefaultMCPGatewayContainerPort is the default port the MCP gateway listens on inside the container
	DefaultMCPGatewayContainerPort = 8000

	// DefaultMCPGatewayPort is the default port for the MCP gateway (host-side)
	// This constant is kept for backwards compatibility with existing configurations
	DefaultMCPGatewayPort = DefaultMCPGatewayHostPort
)
