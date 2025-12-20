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
	var configFile string
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
1. --config flag pointing to a JSON config file
2. stdin (reads JSON configuration from standard input)

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
  awmg --config config.json                    # From file
  awmg --port 8080                             # From stdin
  echo '{"mcpServers":{...}}' | awmg           # Pipe config
  awmg --config config.json --log-dir /tmp/logs # Custom log dir`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPGateway(configFile, port, logDir)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to MCP gateway configuration JSON file")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run HTTP gateway on")
	cmd.Flags().StringVar(&logDir, "log-dir", "/tmp/gh-aw/mcp-gateway-logs", "Directory for MCP gateway logs")

	return cmd
}

// runMCPGateway starts the MCP gateway server
func runMCPGateway(configFile string, port int, logDir string) error {
	gatewayLog.Printf("Starting MCP gateway on port %d", port)

	// Read configuration
	config, err := readGatewayConfig(configFile)
	if err != nil {
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

// readGatewayConfig reads the gateway configuration from file or stdin
func readGatewayConfig(configFile string) (*MCPGatewayConfig, error) {
	var data []byte
	var err error

	if configFile != "" {
		// Read from file
		gatewayLog.Printf("Reading configuration from file: %s", configFile)
		data, err = os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		// Read from stdin
		gatewayLog.Print("Reading configuration from stdin")
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
	}

	var config MCPGatewayConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration JSON: %w", err)
	}

	gatewayLog.Printf("Loaded configuration with %d MCP servers", len(config.MCPServers))
	return &config, nil
}

// initializeSessions creates MCP sessions for all configured servers
func (g *MCPGatewayServer) initializeSessions() error {
	gatewayLog.Printf("Initializing %d MCP sessions", len(g.config.MCPServers))

	for serverName, serverConfig := range g.config.MCPServers {
		gatewayLog.Printf("Initializing session for server: %s", serverName)

		session, err := g.createMCPSession(serverName, serverConfig)
		if err != nil {
			gatewayLog.Printf("Failed to initialize session for %s: %v", serverName, err)
			return fmt.Errorf("failed to create session for server %s: %w", serverName, err)
		}

		g.mu.Lock()
		g.sessions[serverName] = session
		g.mu.Unlock()

		gatewayLog.Printf("Successfully initialized session for %s", serverName)
	}

	return nil
}

// createMCPSession creates an MCP session for a single server configuration
func (g *MCPGatewayServer) createMCPSession(serverName string, config MCPServerConfig) (*mcp.ClientSession, error) {
	// Create log file for this server
	logFile := filepath.Join(g.logDir, fmt.Sprintf("%s.log", serverName))
	logFd, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFd.Close()

	gatewayLog.Printf("Created log file for %s: %s", serverName, logFile)

	// Handle different server types
	if config.URL != "" {
		// HTTP transport (not yet fully supported in go-sdk for SSE)
		gatewayLog.Printf("Creating HTTP client for %s at %s", serverName, config.URL)
		return nil, fmt.Errorf("HTTP transport not yet fully implemented in MCP gateway")
	} else if config.Command != "" {
		// Command transport (subprocess with stdio)
		gatewayLog.Printf("Creating command client for %s with command: %s %v", serverName, config.Command, config.Args)

		// Create command with environment variables
		cmd := exec.Command(config.Command, config.Args...)
		if len(config.Env) > 0 {
			cmd.Env = os.Environ()
			for k, v := range config.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}

		// Create command transport
		transport := &mcp.CommandTransport{
			Command: cmd,
		}

		client := mcp.NewClient(&mcp.Implementation{
			Name:    fmt.Sprintf("gateway-client-%s", serverName),
			Version: GetVersion(),
		}, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to command server: %w", err)
		}

		return session, nil
	} else if config.Container != "" {
		// Docker container (not yet implemented)
		return nil, fmt.Errorf("docker container support not yet implemented")
	}

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
