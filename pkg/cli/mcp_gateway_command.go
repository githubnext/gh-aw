package cli

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
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var gatewayLog = logger.New("cli:mcp_gateway")

// MCPGatewayConfig represents the configuration for the MCP gateway
type MCPGatewayConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	Gateway    GatewaySettings            `json:"gateway,omitempty"`
}

// MCPServerConfig represents configuration for a single MCP server
type MCPServerConfig struct {
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Container string            `json:"container,omitempty"`
}

// GatewaySettings represents gateway-specific settings
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
	cmd.Flags().StringVar(&logDir, "log-dir", "/tmp/gh-aw/mcp-gateway-logs", "Directory for MCP gateway logs")

	return cmd
}

// runMCPGateway starts the MCP gateway server
func runMCPGateway(configFiles []string, port int, logDir string) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP gateway (port: %d, logDir: %s, configFiles: %v)", port, logDir, configFiles)))
	gatewayLog.Printf("Starting MCP gateway on port %d", port)

	// Read configuration
	config, err := readGatewayConfig(configFiles)
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

	// Start HTTP server
	return gateway.startHTTPServer()
}

// readGatewayConfig reads the gateway configuration from files or stdin
func readGatewayConfig(configFiles []string) (*MCPGatewayConfig, error) {
	var configs []*MCPGatewayConfig
	
	if len(configFiles) > 0 {
		// Read from file(s)
		for _, configFile := range configFiles {
			gatewayLog.Printf("Reading configuration from file: %s", configFile)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Reading configuration from file: %s", configFile)))
			
			// Check if file exists
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Configuration file not found: %s", configFile)))
				gatewayLog.Printf("Configuration file not found: %s", configFile)
				return nil, fmt.Errorf("configuration file not found: %s", configFile)
			}
			
			data, err := os.ReadFile(configFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read config file: %v", err)))
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Read %d bytes from file", len(data))))
			gatewayLog.Printf("Read %d bytes from file", len(data))
			
			// Validate we have data
			if len(data) == 0 {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: Configuration data is empty"))
				gatewayLog.Print("Configuration data is empty")
				return nil, fmt.Errorf("configuration data is empty")
			}
			
			config, err := parseGatewayConfig(data)
			if err != nil {
				return nil, err
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
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Read %d bytes from stdin", len(data))))
		gatewayLog.Printf("Read %d bytes from stdin", len(data))
		
		if len(data) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: No configuration data received from stdin"))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please provide configuration via --config flag or pipe JSON to stdin"))
			gatewayLog.Print("No data received from stdin")
			return nil, fmt.Errorf("no configuration data received from stdin")
		}
		
		// Validate we have data
		if len(data) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: Configuration data is empty"))
			gatewayLog.Print("Configuration data is empty")
			return nil, fmt.Errorf("configuration data is empty")
		}
		
		config, err := parseGatewayConfig(data)
		if err != nil {
			return nil, err
		}
		
		configs = append(configs, config)
	}
	
	// Merge all configs
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configuration loaded")
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
		return nil, fmt.Errorf("no MCP servers configured in configuration")
	}
	
	// Log server names for debugging
	serverNames := make([]string, 0, len(mergedConfig.MCPServers))
	for name := range mergedConfig.MCPServers {
		serverNames = append(serverNames, name)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("MCP servers configured: %v", serverNames)))
	gatewayLog.Printf("MCP servers configured: %v", serverNames)
	
	return mergedConfig, nil
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
	filteredServers := make(map[string]MCPServerConfig)
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
		MCPServers: make(map[string]MCPServerConfig),
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
func (g *MCPGatewayServer) createMCPSession(serverName string, config MCPServerConfig) (*mcp.ClientSession, error) {
	// Create log file for this server
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
		// HTTP transport (not yet fully supported in go-sdk for SSE)
		gatewayLog.Printf("Attempting HTTP client for %s at %s", serverName, config.URL)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("HTTP transport not yet supported for %s", serverName)))
		return nil, fmt.Errorf("HTTP transport not yet fully implemented in MCP gateway")
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
	json.NewEncoder(w).Encode(response)
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
	json.NewEncoder(w).Encode(response)
}
