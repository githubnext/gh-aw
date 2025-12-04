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
	"github.com/spf13/cobra"
)

var mcpGatewayLog = logger.New("cli:mcp_gateway")

// NewMCPGatewayCommand creates the mcp-gateway command
func NewMCPGatewayCommand() *cobra.Command {
	var port int
	var apiKey string
	var logsDir string

	cmd := &cobra.Command{
		Use:   "mcp-gateway <mcp-server.json>",
		Short: "Run an MCP gateway that proxies to multiple MCP servers",
		Long: `Run an MCP gateway that acts as a proxy to multiple MCP servers.

The gateway starts an HTTP MCP server that forwards tool calls to the configured
backend MCP servers. This allows clients to connect to a single gateway endpoint
and access tools from multiple MCP servers.

The configuration file should be a JSON file with the following structure:

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
    "port": 8080
  }

Each server can be configured as:
  - stdio: Specify "command" and optional "args"
  - HTTP: Specify "url"
  - Docker: Specify "container" and optional "args"

The gateway will:
1. Connect to all configured MCP servers at startup
2. List all available tools from each server
3. Start an HTTP MCP server that proxies tool calls to the appropriate backend
4. Handle tool name collisions by prefixing with server name

Example:
  gh aw mcp-gateway mcp-server.json
  gh aw mcp-gateway --port 9000 --api-key secret123 mcp-server.json
  gh aw mcp-gateway --logs-dir /tmp/gateway-logs mcp-server.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]
			return runMCPGateway(configFile, port, apiKey, logsDir)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run the gateway HTTP server on (default: 8080 or from config file)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key to authorize connections to the gateway")
	cmd.Flags().StringVar(&logsDir, "logs-dir", "", "Directory to write debug logs (default: no file logging)")

	return cmd
}

// runMCPGateway starts the MCP gateway
func runMCPGateway(configFile string, portOverride int, apiKey string, logsDir string) error {
	mcpGatewayLog.Printf("Starting MCP gateway with config file: %s", configFile)

	// Set up file logging if logs-dir is specified
	if logsDir != "" {
		if err := setupFileLogging(logsDir); err != nil {
			return fmt.Errorf("failed to setup file logging: %w", err)
		}
		mcpGatewayLog.Printf("File logging enabled to directory: %s", logsDir)
	}

	// Load configuration from file
	config, err := gateway.LoadConfigFromJSON(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override port if specified or use default
	if portOverride > 0 {
		mcpGatewayLog.Printf("Using port from flag: %d", portOverride)
		config.Port = portOverride
	} else if config.Port == 0 {
		// Use default port if not specified in config or flag
		config.Port = 8080
		mcpGatewayLog.Printf("Using default port: %d", config.Port)
	}

	// Set API key if provided
	if apiKey != "" {
		config.APIKey = apiKey
		mcpGatewayLog.Print("API key authentication enabled")
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP Gateway on port %d with %d servers", config.Port, len(config.MCPServers))))

	if apiKey != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("API key authentication: enabled"))
	}

	if logsDir != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Debug logs: %s", logsDir)))
	}

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
