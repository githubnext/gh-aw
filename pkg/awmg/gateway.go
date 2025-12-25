package awmg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var gatewayLog = logger.New("awmg:gateway")

// version is set by the main package.
var version = "dev"

// SetVersionInfo sets the version information for the awmg package.
func SetVersionInfo(v string) {
	version = v
}

// GetVersion returns the current version.
func GetVersion() string {
	return version
}

// MCPGatewayConfig represents the configuration for the MCP gateway.
type MCPGatewayConfig struct {
	MCPServers map[string]parser.MCPServerConfig `json:"mcpServers"`
	Gateway    GatewaySettings                   `json:"gateway,omitempty"`
}

// GatewaySettings represents gateway-specific settings.
type GatewaySettings struct {
	Port   int    `json:"port,omitempty"`
	APIKey string `json:"apiKey,omitempty"`
}

// MCPGatewayServer manages multiple MCP sessions and exposes them via HTTP
type MCPGatewayServer struct {
	config   *MCPGatewayConfig
	sessions map[string]*mcp.ClientSession
	mu       sync.RWMutex
	logDir   string
}

// NewMCPGatewayCommand creates the mcp-gateway command
func NewMCPGatewayCommand() *cobra.Command {
	var configFiles []string
	var port int
	var logDir string

	cmd := &cobra.Command{
		Use:   "mcp-gateway",
		Short: "Run an MCP gateway proxy that aggregates multiple MCP servers",
		Long: `Run an MCP gateway that acts as a proxy to multiple MCP servers.

The gateway:
- Integrates by default with the sandbox.mcp extension point
- Imports Claude/Copilot/Codex MCP server JSON configuration
- Starts each MCP server and mounts an MCP client on each
- Mounts an HTTP MCP server that acts as a gateway to the MCP clients
- Supports most MCP gestures through the go-MCP SDK
- Provides extensive logging to file in the MCP log folder

Configuration can be provided via:
1. --config flag(s) pointing to JSON config file(s) (can be specified multiple times)
2. stdin (reads JSON configuration from standard input)

Multiple config files are merged in order, with later files overriding earlier ones.

Configuration format:
{
  "mcpServers": {
    "server-name": {
      "command": "command",
      "args": ["arg1", "arg2"],
      "env": {"KEY": "value"}
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "optional-key"
  }
}

Examples:
  awmg --config config.json                    # From single file
  awmg --config base.json --config override.json # From multiple files (merged)
  awmg --port 8080                             # From stdin
  echo '{"mcpServers":{...}}' | awmg           # Pipe config
  awmg --config config.json --log-dir /tmp/logs # Custom log dir`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPGateway(configFiles, port, logDir)
		},
	}

	cmd.Flags().StringArrayVarP(&configFiles, "config", "c", []string{}, "Path to MCP gateway configuration JSON file (can be specified multiple times)")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run HTTP gateway on")
	cmd.Flags().StringVar(&logDir, "log-dir", "/tmp/gh-aw/mcp-logs", "Directory for MCP gateway logs")

	return cmd
}

// runMCPGateway starts the MCP gateway server
func runMCPGateway(configFiles []string, port int, logDir string) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP gateway (port: %d, logDir: %s, configFiles: %v)", port, logDir, configFiles)))
	gatewayLog.Printf("Starting MCP gateway on port %d", port)

	// Read configuration
	config, originalConfigPath, err := readGatewayConfig(configFiles)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read configuration: %v", err)))
		return fmt.Errorf("failed to read gateway configuration: %w", err)
	}

	// Override port if specified in command line
	if port > 0 {
		config.Gateway.Port = port
	} else if config.Gateway.Port == 0 {
		config.Gateway.Port = 8080 // Default port
	}

	// Create log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create gateway server
	gateway := &MCPGatewayServer{
		config:   config,
		sessions: make(map[string]*mcp.ClientSession),
		logDir:   logDir,
	}

	// Initialize MCP sessions for each server
	if err := gateway.initializeSessions(); err != nil {
		return fmt.Errorf("failed to initialize MCP sessions: %w", err)
	}

	// Rewrite the MCP config file to point servers to the gateway
	if originalConfigPath != "" {
		if err := rewriteMCPConfigForGateway(originalConfigPath, config); err != nil {
			gatewayLog.Printf("Warning: Failed to rewrite MCP config: %v", err)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to rewrite MCP config: %v", err)))
			// Don't fail - gateway can still run
		}
	} else {
		gatewayLog.Print("Skipping config rewrite (config was read from stdin)")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping config rewrite (config was read from stdin)"))
	}

	// Start HTTP server
	return gateway.startHTTPServer()
}

// readGatewayConfig reads the gateway configuration from files or stdin
// Returns the config, the path to the first config file (for rewriting), and any error
func readGatewayConfig(configFiles []string) (*MCPGatewayConfig, string, error) {
	var configs []*MCPGatewayConfig
	var originalConfigPath string

	if len(configFiles) > 0 {
		// Read from file(s)
		for i, configFile := range configFiles {
			gatewayLog.Printf("Reading configuration from file: %s", configFile)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Reading configuration from file: %s", configFile)))

			// Store the first config file path for rewriting
			if i == 0 {
				originalConfigPath = configFile
			}

			// Check if file exists
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Configuration file not found: %s", configFile)))
				gatewayLog.Printf("Configuration file not found: %s", configFile)
				return nil, "", fmt.Errorf("configuration file not found: %s", configFile)
			}

			data, err := os.ReadFile(configFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read config file: %v", err)))
				return nil, "", fmt.Errorf("failed to read config file: %w", err)
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Read %d bytes from file", len(data))))
			gatewayLog.Printf("Read %d bytes from file", len(data))

			// Validate we have data
			if len(data) == 0 {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: Configuration data is empty"))
				gatewayLog.Print("Configuration data is empty")
				return nil, "", fmt.Errorf("configuration data is empty")
			}

			config, err := parseGatewayConfig(data)
			if err != nil {
				return nil, "", err
			}

			configs = append(configs, config)
		}
	} else {
		// Read from stdin
		gatewayLog.Print("Reading configuration from stdin")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Reading configuration from stdin..."))
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read from stdin: %v", err)))
			return nil, "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Read %d bytes from stdin", len(data))))
		gatewayLog.Printf("Read %d bytes from stdin", len(data))

		if len(data) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: No configuration data received from stdin"))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please provide configuration via --config flag or pipe JSON to stdin"))
			gatewayLog.Print("No data received from stdin")
			return nil, "", fmt.Errorf("no configuration data received from stdin")
		}

		// Validate we have data
		if len(data) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: Configuration data is empty"))
			gatewayLog.Print("Configuration data is empty")
			return nil, "", fmt.Errorf("configuration data is empty")
		}

		config, err := parseGatewayConfig(data)
		if err != nil {
			return nil, "", err
		}

		configs = append(configs, config)
		// No config file path when reading from stdin
		originalConfigPath = ""
	}

	// Merge all configs
	if len(configs) == 0 {
		return nil, "", fmt.Errorf("no configuration loaded")
	}

	mergedConfig := configs[0]
	for i := 1; i < len(configs); i++ {
		gatewayLog.Printf("Merging configuration %d of %d", i+1, len(configs))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Merging configuration %d of %d", i+1, len(configs))))
		mergedConfig = mergeConfigs(mergedConfig, configs[i])
	}

	gatewayLog.Printf("Successfully merged %d configuration(s)", len(configs))
	if len(configs) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully merged %d configurations", len(configs))))
	}

	gatewayLog.Printf("Loaded configuration with %d MCP servers", len(mergedConfig.MCPServers))
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully loaded configuration with %d MCP servers", len(mergedConfig.MCPServers))))

	// Validate we have at least one server configured
	if len(mergedConfig.MCPServers) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: No MCP servers configured in configuration"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Configuration must include at least one MCP server in 'mcpServers' section"))
		gatewayLog.Print("No MCP servers configured")
		return nil, "", fmt.Errorf("no MCP servers configured in configuration")
	}

	// Log server names for debugging
	serverNames := make([]string, 0, len(mergedConfig.MCPServers))
	for name := range mergedConfig.MCPServers {
		serverNames = append(serverNames, name)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("MCP servers configured: %v", serverNames)))
	gatewayLog.Printf("MCP servers configured: %v", serverNames)

	return mergedConfig, originalConfigPath, nil
}

// parseGatewayConfig parses raw JSON data into a gateway config
func parseGatewayConfig(data []byte) (*MCPGatewayConfig, error) {
	gatewayLog.Printf("Parsing %d bytes of configuration data", len(data))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Parsing %d bytes of configuration data", len(data))))

	var config MCPGatewayConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to parse JSON: %v", err)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Data received (first 500 chars): %s", string(data[:min(500, len(data))]))))
		gatewayLog.Printf("Failed to parse JSON: %v", err)
		return nil, fmt.Errorf("failed to parse configuration JSON: %w", err)
	}

	gatewayLog.Printf("Successfully parsed JSON configuration")

	// Filter out internal workflow MCP servers (safeinputs and safeoutputs)
	// These are used internally by the workflow and should not be proxied by the gateway
	filteredServers := make(map[string]parser.MCPServerConfig)
	for name, serverConfig := range config.MCPServers {
		if name == "safeinputs" || name == "safeoutputs" {
			gatewayLog.Printf("Filtering out internal workflow server: %s", name)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Filtering out internal workflow server: %s", name)))
			continue
		}
		filteredServers[name] = serverConfig
	}
	config.MCPServers = filteredServers

	return &config, nil
}

// mergeConfigs merges two gateway configurations, with the second overriding the first
func mergeConfigs(base, override *MCPGatewayConfig) *MCPGatewayConfig {
	result := &MCPGatewayConfig{
		MCPServers: make(map[string]parser.MCPServerConfig),
		Gateway:    base.Gateway,
	}

	// Copy all servers from base
	for name, config := range base.MCPServers {
		result.MCPServers[name] = config
	}

	// Override/add servers from override config
	for name, config := range override.MCPServers {
		gatewayLog.Printf("Merging server config for: %s", name)
		result.MCPServers[name] = config
	}

	// Override gateway settings if provided
	if override.Gateway.Port != 0 {
		result.Gateway.Port = override.Gateway.Port
		gatewayLog.Printf("Override gateway port: %d", override.Gateway.Port)
	}
	if override.Gateway.APIKey != "" {
		result.Gateway.APIKey = override.Gateway.APIKey
		gatewayLog.Printf("Override gateway API key (length: %d)", len(override.Gateway.APIKey))
	}

	return result
}

// rewriteMCPConfigForGateway rewrites the MCP config file to point all servers to the gateway
func rewriteMCPConfigForGateway(configPath string, config *MCPGatewayConfig) error {
	gatewayLog.Printf("Rewriting MCP config file: %s", configPath)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Rewriting MCP config file: %s", configPath)))

	// Read the original config file to preserve non-proxied servers
	gatewayLog.Printf("Reading original config from %s", configPath)
	originalConfigData, err := os.ReadFile(configPath)
	if err != nil {
		gatewayLog.Printf("Failed to read original config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read original config: %v", err)))
		return fmt.Errorf("failed to read original config: %w", err)
	}

	var originalConfig map[string]any
	if err := json.Unmarshal(originalConfigData, &originalConfig); err != nil {
		gatewayLog.Printf("Failed to parse original config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to parse original config: %v", err)))
		return fmt.Errorf("failed to parse original config: %w", err)
	}

	port := config.Gateway.Port
	if port == 0 {
		port = 8080
	}
	// Use host.docker.internal instead of localhost to allow Docker containers to reach the gateway
	gatewayURL := fmt.Sprintf("http://host.docker.internal:%d", port)

	gatewayLog.Printf("Gateway URL: %s", gatewayURL)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Gateway URL: %s", gatewayURL)))

	// Get original mcpServers to preserve non-proxied servers
	var originalMCPServers map[string]any
	if servers, ok := originalConfig["mcpServers"].(map[string]any); ok {
		originalMCPServers = servers
	} else {
		originalMCPServers = make(map[string]any)
	}

	// Create merged config with rewritten proxied servers and preserved non-proxied servers
	rewrittenConfig := make(map[string]any)
	mcpServers := make(map[string]any)

	// First, copy all servers from original (preserves non-proxied servers like safeinputs/safeoutputs)
	for serverName, serverConfig := range originalMCPServers {
		mcpServers[serverName] = serverConfig
	}

	gatewayLog.Printf("Transforming %d proxied servers to point to gateway", len(config.MCPServers))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Transforming %d proxied servers to point to gateway", len(config.MCPServers))))

	// Then, overwrite with gateway URLs for proxied servers only
	for serverName := range config.MCPServers {
		serverURL := fmt.Sprintf("%s/mcp/%s", gatewayURL, serverName)

		gatewayLog.Printf("Rewriting server '%s' to use gateway URL: %s", serverName, serverURL)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  %s -> %s", serverName, serverURL)))

		serverConfig := map[string]any{
			"type":  "http",
			"url":   serverURL,
			"tools": []string{"*"},
		}

		// Add authentication header if API key is configured
		if config.Gateway.APIKey != "" {
			gatewayLog.Printf("Adding authorization header for server '%s'", serverName)
			serverConfig["headers"] = map[string]any{
				"Authorization": fmt.Sprintf("Bearer %s", config.Gateway.APIKey),
			}
		}

		mcpServers[serverName] = serverConfig
	}

	rewrittenConfig["mcpServers"] = mcpServers

	// Do NOT include gateway section in rewritten config (per requirement)
	gatewayLog.Print("Gateway section removed from rewritten config")

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(rewrittenConfig, "", "  ")
	if err != nil {
		gatewayLog.Printf("Failed to marshal rewritten config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to marshal rewritten config: %v", err)))
		return fmt.Errorf("failed to marshal rewritten config: %w", err)
	}

	gatewayLog.Printf("Writing %d bytes to config file", len(data))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Writing %d bytes to config file", len(data))))

	// Write back to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		gatewayLog.Printf("Failed to write rewritten config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to write rewritten config: %v", err)))
		return fmt.Errorf("failed to write rewritten config: %w", err)
	}

	gatewayLog.Printf("Successfully rewrote MCP config file")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully rewrote MCP config: %s", configPath)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  %d proxied servers now point to gateway at %s", len(config.MCPServers), gatewayURL)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  %d total servers in config", len(mcpServers))))

	return nil
}

// initializeSessions creates MCP sessions for all configured servers
func (g *MCPGatewayServer) initializeSessions() error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Initializing %d MCP sessions", len(g.config.MCPServers))))
	gatewayLog.Printf("Initializing %d MCP sessions", len(g.config.MCPServers))

	// This should never happen as we validate in readGatewayConfig, but double-check
	if len(g.config.MCPServers) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: No MCP servers to initialize"))
		gatewayLog.Print("No MCP servers to initialize")
		return fmt.Errorf("no MCP servers configured")
	}

	successCount := 0
	for serverName, serverConfig := range g.config.MCPServers {
		gatewayLog.Printf("Initializing session for server: %s", serverName)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Initializing session for server: %s (command: %s, args: %v)", serverName, serverConfig.Command, serverConfig.Args)))

		session, err := g.createMCPSession(serverName, serverConfig)
		if err != nil {
			gatewayLog.Printf("Failed to initialize session for %s: %v", serverName, err)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to initialize session for %s: %v", serverName, err)))
			return fmt.Errorf("failed to create session for server %s: %w", serverName, err)
		}

		g.mu.Lock()
		g.sessions[serverName] = session
		g.mu.Unlock()

		successCount++
		gatewayLog.Printf("Successfully initialized session for %s (%d/%d)", serverName, successCount, len(g.config.MCPServers))
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully initialized session for %s (%d/%d)", serverName, successCount, len(g.config.MCPServers))))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("All %d MCP sessions initialized successfully", len(g.config.MCPServers))))
	gatewayLog.Printf("All %d MCP sessions initialized successfully", len(g.config.MCPServers))
	return nil
}

// createMCPSession creates an MCP session for a single server configuration
func (g *MCPGatewayServer) createMCPSession(serverName string, config parser.MCPServerConfig) (*mcp.ClientSession, error) {
	// Create log file for this server (flat directory structure)
	logFile := filepath.Join(g.logDir, fmt.Sprintf("%s.log", serverName))
	gatewayLog.Printf("Creating log file for %s: %s", serverName, logFile)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating log file for %s: %s", serverName, logFile)))

	logFd, err := os.Create(logFile)
	if err != nil {
		gatewayLog.Printf("Failed to create log file for %s: %v", serverName, err)
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFd.Close()

	gatewayLog.Printf("Log file created successfully for %s", serverName)

	// Handle different server types
	if config.URL != "" {
		// Streamable HTTP transport using the go-sdk StreamableClientTransport
		gatewayLog.Printf("Creating streamable HTTP client for %s at %s", serverName, config.URL)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using streamable HTTP transport: %s", config.URL)))

		// Create streamable client transport
		transport := &mcp.StreamableClientTransport{
			Endpoint: config.URL,
		}

		gatewayLog.Printf("Creating MCP client for %s", serverName)
		client := mcp.NewClient(&mcp.Implementation{
			Name:    fmt.Sprintf("gateway-client-%s", serverName),
			Version: GetVersion(),
		}, nil)

		gatewayLog.Printf("Connecting to MCP server %s with 30s timeout", serverName)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Connecting to %s...", serverName)))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			gatewayLog.Printf("Failed to connect to HTTP server %s: %v", serverName, err)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Connection failed for %s: %v", serverName, err)))
			return nil, fmt.Errorf("failed to connect to HTTP server: %w", err)
		}

		gatewayLog.Printf("Successfully connected to MCP server %s via streamable HTTP", serverName)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Connected to %s successfully via streamable HTTP", serverName)))
		return session, nil
	} else if config.Command != "" {
		// Command transport (subprocess with stdio)
		gatewayLog.Printf("Creating command client for %s with command: %s %v", serverName, config.Command, config.Args)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using command transport: %s %v", config.Command, config.Args)))

		// Create command with environment variables
		cmd := exec.Command(config.Command, config.Args...)
		if len(config.Env) > 0 {
			gatewayLog.Printf("Setting %d environment variables for %s", len(config.Env), serverName)
			cmd.Env = os.Environ()
			for k, v := range config.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
				gatewayLog.Printf("Env var for %s: %s=%s", serverName, k, v)
			}
		}

		// Create command transport
		gatewayLog.Printf("Creating CommandTransport for %s", serverName)
		transport := &mcp.CommandTransport{
			Command: cmd,
		}

		gatewayLog.Printf("Creating MCP client for %s", serverName)
		client := mcp.NewClient(&mcp.Implementation{
			Name:    fmt.Sprintf("gateway-client-%s", serverName),
			Version: GetVersion(),
		}, nil)

		gatewayLog.Printf("Connecting to MCP server %s with 30s timeout", serverName)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Connecting to %s...", serverName)))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			gatewayLog.Printf("Failed to connect to command server %s: %v", serverName, err)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Connection failed for %s: %v", serverName, err)))
			return nil, fmt.Errorf("failed to connect to command server: %w", err)
		}

		gatewayLog.Printf("Successfully connected to MCP server %s", serverName)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Connected to %s successfully", serverName)))
		return session, nil
	} else if config.Container != "" {
		// Docker container (not yet implemented)
		gatewayLog.Printf("Docker container requested for %s but not yet implemented", serverName)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Docker container support not available for %s", serverName)))
		return nil, fmt.Errorf("docker container support not yet implemented")
	}

	gatewayLog.Printf("Invalid server configuration for %s: no command, url, or container specified", serverName)
	fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Invalid configuration for %s: must specify command, url, or container", serverName)))
	return nil, fmt.Errorf("invalid server configuration: must specify command, url, or container")
}

// startHTTPServer starts the HTTP server for the gateway
func (g *MCPGatewayServer) startHTTPServer() error {
	port := g.config.Gateway.Port
	gatewayLog.Printf("Starting HTTP server on port %d", port)

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// MCP endpoint for each server
	for serverName := range g.config.MCPServers {
		serverNameCopy := serverName // Capture for closure
		path := fmt.Sprintf("/mcp/%s", serverName)
		gatewayLog.Printf("Registering endpoint: %s", path)

		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			g.handleMCPRequest(w, r, serverNameCopy)
		})
	}

	// List servers endpoint
	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		g.handleListServers(w, r)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	fmt.Fprintf(os.Stderr, "%s\n", console.FormatSuccessMessage(fmt.Sprintf("MCP gateway listening on http://localhost:%d", port)))
	gatewayLog.Printf("HTTP server ready on port %d", port)

	return server.ListenAndServe()
}

// handleMCPRequest handles an MCP protocol request for a specific server
func (g *MCPGatewayServer) handleMCPRequest(w http.ResponseWriter, r *http.Request, serverName string) {
	gatewayLog.Printf("Handling MCP request for server: %s", serverName)

	// Check API key if configured
	if g.config.Gateway.APIKey != "" {
		authHeader := r.Header.Get("Authorization")
		expectedAuth := fmt.Sprintf("Bearer %s", g.config.Gateway.APIKey)
		if authHeader != expectedAuth {
			gatewayLog.Printf("Unauthorized request for %s", serverName)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Get the session
	g.mu.RLock()
	session, exists := g.sessions[serverName]
	g.mu.RUnlock()

	if !exists {
		gatewayLog.Printf("Server not found: %s", serverName)
		http.Error(w, fmt.Sprintf("Server not found: %s", serverName), http.StatusNotFound)
		return
	}

	// Parse request body
	var reqBody map[string]any
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		gatewayLog.Printf("Failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	method, _ := reqBody["method"].(string)
	gatewayLog.Printf("MCP method: %s for server: %s", method, serverName)

	// Handle different MCP methods
	var response any
	var err error

	switch method {
	case "initialize":
		response, err = g.handleInitialize(session)
	case "tools/list":
		response, err = g.handleListTools(session)
	case "tools/call":
		response, err = g.handleCallTool(session, reqBody)
	case "resources/list":
		response, err = g.handleListResources(session)
	case "prompts/list":
		response, err = g.handleListPrompts(session)
	default:
		gatewayLog.Printf("Unsupported method: %s", method)
		http.Error(w, fmt.Sprintf("Unsupported method: %s", method), http.StatusBadRequest)
		return
	}

	if err != nil {
		gatewayLog.Printf("Error handling %s: %v", method, err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		gatewayLog.Printf("Failed to encode JSON response: %v", err)
	}
}

// handleInitialize handles the initialize method
func (g *MCPGatewayServer) handleInitialize(session *mcp.ClientSession) (any, error) {
	// Return server capabilities
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
			"prompts":   map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "mcp-gateway",
			"version": GetVersion(),
		},
	}, nil
}

// handleListTools handles the tools/list method
func (g *MCPGatewayServer) handleListTools(session *mcp.ClientSession) (any, error) {
	ctx := context.Background()
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return map[string]any{
		"tools": result.Tools,
	}, nil
}

// handleCallTool handles the tools/call method
func (g *MCPGatewayServer) handleCallTool(session *mcp.ClientSession, reqBody map[string]any) (any, error) {
	params, ok := reqBody["params"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid params")
	}

	name, _ := params["name"].(string)
	arguments := params["arguments"]

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	return map[string]any{
		"content": result.Content,
	}, nil
}

// handleListResources handles the resources/list method
func (g *MCPGatewayServer) handleListResources(session *mcp.ClientSession) (any, error) {
	ctx := context.Background()
	result, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return map[string]any{
		"resources": result.Resources,
	}, nil
}

// handleListPrompts handles the prompts/list method
func (g *MCPGatewayServer) handleListPrompts(session *mcp.ClientSession) (any, error) {
	ctx := context.Background()
	result, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return map[string]any{
		"prompts": result.Prompts,
	}, nil
}

// handleListServers handles the /servers endpoint
func (g *MCPGatewayServer) handleListServers(w http.ResponseWriter, r *http.Request) {
	gatewayLog.Print("Handling list servers request")

	g.mu.RLock()
	servers := make([]string, 0, len(g.sessions))
	for name := range g.sessions {
		servers = append(servers, name)
	}
	g.mu.RUnlock()

	response := map[string]any{
		"servers": servers,
		"count":   len(servers),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		gatewayLog.Printf("Failed to encode JSON response: %v", err)
	}
}
