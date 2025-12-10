package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/gateway"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var mcpGatewayLog = logger.New("cli:mcp_gateway")

// NewMCPGatewayCommand creates the mcp-gateway command
func NewMCPGatewayCommand() *cobra.Command {
	var port int
	var apiKey string
	var logsDir string
	var mcpsConfig string
	var scriptsConfig string

	cmd := &cobra.Command{
		Use:   "mcp-gateway",
		Short: "Run an MCP gateway that proxies to multiple MCP servers",
		Long: `Run an MCP gateway that acts as a proxy to multiple MCP servers.

The gateway starts an HTTP MCP server that forwards tool calls to the configured
backend MCP servers. This allows clients to connect to a single gateway endpoint
and access tools from multiple MCP servers and/or safe-inputs tools.

Configuration can be provided in two ways:

1. MCP Servers (--mcps): JSON file defining backend MCP servers to proxy
2. Safe-Inputs (--scripts): JSON file defining custom tools with JS/Python/Shell handlers

MCP servers configuration (--mcps) should be a JSON file with the following structure:

  {
    "mcpServers": {
      "server1": {
        "command": "node",
        "args": ["path/to/server.js"]
      },
      "server2": {
        "url": "http://localhost:3000"
      },
      "server3": {
        "container": "my-mcp-server:latest",
        "args": ["--option", "value"]
      }
    },
    "port": 8088
  }

Each server can be configured as:
  - stdio: Specify "command" and optional "args"
  - HTTP: Specify "url"
  - Docker: Specify "container" and optional "args"

Safe-inputs configuration (--scripts) should be a JSON file with the following structure:

  {
    "tools": [
      {
        "name": "my_tool",
        "description": "A custom tool",
        "inputSchema": {
          "type": "object",
          "properties": {
            "param": {"type": "string", "description": "A parameter"}
          }
        },
        "handler": "my_tool.cjs"
      }
    ]
  }

Handlers can be .cjs (Node.js), .py (Python 3.13), or .sh (Shell script).

The gateway will:
1. Connect to all configured MCP servers at startup (if --mcps is provided)
2. Load safe-inputs tools from handlers (if --scripts is provided)
3. List all available tools from each server
4. Start an HTTP MCP server that proxies tool calls to the appropriate backend
5. Handle tool name collisions by prefixing with server name

Example:
  gh aw mcp-gateway --mcps mcp-servers.json
  gh aw mcp-gateway --scripts tools.json
  gh aw mcp-gateway --mcps mcp-servers.json --scripts tools.json
  gh aw mcp-gateway --port 9000 --api-key secret123 --mcps mcp-servers.json
  gh aw mcp-gateway --logs-dir /tmp/gateway-logs --scripts tools.json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate that at least one config is provided
			if mcpsConfig == "" && scriptsConfig == "" {
				return fmt.Errorf("at least one of --mcps or --scripts must be provided")
			}
			return runMCPGateway(mcpsConfig, scriptsConfig, port, apiKey, logsDir)
		},
	}

	cmd.Flags().StringVar(&mcpsConfig, "mcps", "", "Path to MCP servers configuration file (mcpServers.json)")
	cmd.Flags().StringVar(&scriptsConfig, "scripts", "", "Path to safe-inputs tools configuration file (tools.json)")
	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run the gateway HTTP server on (default: 8088 or from config file)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key to authorize connections to the gateway")
	cmd.Flags().StringVar(&logsDir, "logs-dir", "", "Directory to write debug logs (default: no file logging)")

	return cmd
}

// runMCPGateway starts the MCP gateway
func runMCPGateway(mcpsConfigFile string, scriptsConfigFile string, portOverride int, apiKey string, logsDir string) error {
	mcpGatewayLog.Printf("Starting MCP gateway with mcps: %s, scripts: %s", mcpsConfigFile, scriptsConfigFile)

	// Set up file logging if logs-dir is specified
	if logsDir != "" {
		if err := setupFileLogging(logsDir); err != nil {
			return fmt.Errorf("failed to setup file logging: %w", err)
		}
		mcpGatewayLog.Printf("File logging enabled to directory: %s", logsDir)
	}

	var config gateway.GatewayConfig

	// Load MCP servers configuration if provided
	if mcpsConfigFile != "" {
		mcpGatewayLog.Printf("Loading MCP servers configuration from: %s", mcpsConfigFile)
		mcpConfig, err := gateway.LoadConfigFromJSON(mcpsConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load MCP servers configuration: %w", err)
		}
		config = mcpConfig
	} else {
		// Initialize empty config
		config = gateway.GatewayConfig{
			MCPServers: make(map[string]parser.MCPServerConfig),
		}
	}

	// Load safe-inputs configuration if provided
	if scriptsConfigFile != "" {
		mcpGatewayLog.Printf("Loading safe-inputs configuration from: %s", scriptsConfigFile)
		config.SafeInputsConfig = scriptsConfigFile
	}

	// Override port if specified or use default
	if portOverride > 0 {
		mcpGatewayLog.Printf("Using port from flag: %d", portOverride)
		config.Port = portOverride
	} else if config.Port == 0 {
		// Use default port if not specified in config or flag
		config.Port = 8088
		mcpGatewayLog.Printf("Using default port: %d", config.Port)
	}

	// Set API key if provided
	if apiKey != "" {
		config.APIKey = apiKey
		mcpGatewayLog.Print("API key authentication enabled")
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP Gateway on port %d", config.Port)))

	if apiKey != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("API key authentication: enabled"))
	}

	if logsDir != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Debug logs: %s", logsDir)))
	}

	// Display MCP servers
	if len(config.MCPServers) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("MCP Servers: %d", len(config.MCPServers))))
		for name, serverConfig := range config.MCPServers {
			var serverType string
			if serverConfig.URL != "" {
				serverType = fmt.Sprintf("HTTP (%s)", serverConfig.URL)
			} else if serverConfig.Command != "" {
				serverType = fmt.Sprintf("stdio (%s)", serverConfig.Command)
			} else if serverConfig.Container != "" {
				serverType = fmt.Sprintf("Docker (%s)", serverConfig.Container)
			}
			mcpGatewayLog.Printf("Server %s: %s", name, serverType)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  - %s: %s", name, serverType)))
		}
	}

	// Display safe-inputs info
	if config.SafeInputsConfig != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Safe-Inputs: %s", config.SafeInputsConfig)))
	}

	// Create gateway
	gw, err := gateway.NewGateway(config)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start gateway in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := gw.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		mcpGatewayLog.Printf("Received signal: %v", sig)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Received signal %v, shutting down...", sig)))
		cancel()
		if err := gw.Close(); err != nil {
			mcpGatewayLog.Printf("Error closing gateway: %v", err)
			return fmt.Errorf("error during shutdown: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Gateway shut down successfully"))
		return nil
	case err := <-errChan:
		mcpGatewayLog.Printf("Gateway error: %v", err)
		return err
	}
}

// setupFileLogging sets up logging to files in the specified directory
func setupFileLogging(logsDir string) error {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Note: The logger package uses DEBUG environment variable for filtering
	// File logging would need to be implemented in the logger package itself
	// For now, we just ensure the directory exists
	mcpGatewayLog.Printf("Logs directory created/verified: %s", logsDir)

	return nil
}
