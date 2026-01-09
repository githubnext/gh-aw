package workflow

const (
	// DefaultMCPGatewayPort is the default port for the MCP gateway
	// Using port 80 so AWF containers can connect via host.docker.internal (default HTTP port)
	DefaultMCPGatewayPort = 80
)
