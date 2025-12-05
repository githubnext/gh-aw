package gateway

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

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var log = logger.New("pkg:gateway")

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	Type      string            `json:"type,omitempty"`
	Command   string            `json:"command,omitempty"`
	Container string            `json:"container,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
}

// GatewayConfig represents the configuration for the MCP gateway
type GatewayConfig struct {
	MCPServers       map[string]MCPServerConfig `json:"mcpServers"`
	Port             int                        `json:"port"`
	APIKey           string                     `json:"apiKey,omitempty"`
	SafeInputsConfig string                     `json:"-"` // Path to safe-inputs tools.json file
}

// Gateway represents an MCP gateway that proxies to multiple MCP servers
type Gateway struct {
	config   GatewayConfig
	clients  map[string]*mcp.Client
	sessions map[string]*mcp.ClientSession
	mu       sync.RWMutex
}

// NewGateway creates a new MCP gateway from configuration
func NewGateway(config GatewayConfig) (*Gateway, error) {
	serverCount := len(config.MCPServers)
	if config.SafeInputsConfig != "" {
		serverCount++
	}

	log.Printf("Creating new gateway with %d MCP servers + safe-inputs on port %d", len(config.MCPServers), config.Port)

	if config.Port == 0 {
		return nil, fmt.Errorf("gateway port must be specified")
	}

	if len(config.MCPServers) == 0 && config.SafeInputsConfig == "" {
		return nil, fmt.Errorf("no MCP servers or safe-inputs configured")
	}

	return &Gateway{
		config:   config,
		clients:  make(map[string]*mcp.Client),
		sessions: make(map[string]*mcp.ClientSession),
	}, nil
}

// Start starts the gateway HTTP server and connects to all configured MCP servers
func (g *Gateway) Start(ctx context.Context) error {
	log.Printf("Starting gateway on port %d", g.config.Port)

	// Connect to all MCP servers (if configured)
	if len(g.config.MCPServers) > 0 {
		if err := g.connectToServers(ctx); err != nil {
			return fmt.Errorf("failed to connect to MCP servers: %w", err)
		}
	}

	// Connect to safe-inputs server (if configured)
	if g.config.SafeInputsConfig != "" {
		if err := g.connectToSafeInputsServer(ctx); err != nil {
			return fmt.Errorf("failed to connect to safe-inputs server: %w", err)
		}
	}

	// List all available tools from all servers
	tools, err := g.listAllTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	log.Printf("Found %d tools across all servers", len(tools))

	// Create the MCP server that will handle incoming requests
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-gateway",
		Version: "1.0.0",
	}, nil)

	// Register tool handlers that proxy to backend servers
	g.registerToolHandlers(server, tools)

	// Create HTTP handler
	var handler http.Handler = mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	// Wrap with authentication middleware if API key is configured
	if g.config.APIKey != "" {
		log.Print("API key authentication enabled")
		handler = g.authMiddleware(handler)
	}

	// Start HTTP server
	addr := fmt.Sprintf(":%d", g.config.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Printf("Gateway HTTP server listening on %s", addr)
	fmt.Fprintf(os.Stderr, "MCP Gateway listening on http://localhost%s\n", addr)

	// Run the HTTP server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}

// connectToServers connects to all configured MCP servers
func (g *Gateway) connectToServers(ctx context.Context) error {
	log.Printf("Connecting to %d MCP servers", len(g.config.MCPServers))

	var errs []error

	for name, config := range g.config.MCPServers {
		log.Printf("Connecting to MCP server: %s (type: %s)", name, config.Type)

		client, session, err := g.createClient(ctx, name, config)
		if err != nil {
			log.Printf("Failed to connect to server %s: %v", name, err)
			errs = append(errs, fmt.Errorf("server %s: %w", name, err))
			continue
		}

		g.mu.Lock()
		g.clients[name] = client
		g.sessions[name] = session
		g.mu.Unlock()

		log.Printf("Successfully connected to server: %s", name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to connect to some servers: %v", errs)
	}

	log.Print("Successfully connected to all MCP servers")
	return nil
}

// connectToSafeInputsServer connects to the safe-inputs MCP server
func (g *Gateway) connectToSafeInputsServer(ctx context.Context) error {
	log.Printf("Connecting to safe-inputs server with config: %s", g.config.SafeInputsConfig)

	// Create a temporary directory for the safe-inputs server scripts
	tmpDir, err := os.MkdirTemp("", "mcp-gateway-safeinputs-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	log.Printf("Created temp directory for safe-inputs: %s", tmpDir)

	// Get the config file directory to resolve relative handler paths
	configDir := "."
	if absPath, err := os.Getwd(); err == nil {
		configDir = absPath
	}
	if g.config.SafeInputsConfig != "" {
		if absConfigPath, err := filepath.Abs(g.config.SafeInputsConfig); err == nil {
			configDir = filepath.Dir(absConfigPath)
		}
	}
	log.Printf("Config directory for handler resolution: %s", configDir)

	// Copy the tools.json to the temp directory, but first update handler paths to be absolute
	toolsConfigPath := filepath.Join(tmpDir, "tools.json")
	if err := g.prepareToolsConfig(configDir, toolsConfigPath); err != nil {
		return fmt.Errorf("failed to prepare tools config: %w", err)
	}

	// Write the required JavaScript modules to temp directory
	scripts := map[string]string{
		"mcp_server_core.cjs":            workflow.GetMCPServerCoreScript(),
		"read_buffer.cjs":                workflow.GetReadBufferScript(),
		"safe_inputs_mcp_server.cjs":    workflow.GetSafeInputsMCPServerScript(),
		"safe_inputs_config_loader.cjs": workflow.GetSafeInputsConfigLoaderScript(),
		"safe_inputs_tool_factory.cjs":  workflow.GetSafeInputsToolFactoryScript(),
		"mcp_handler_shell.cjs":         workflow.GetMCPHandlerShellScript(),
		"mcp_handler_python.cjs":        workflow.GetMCPHandlerPythonScript(),
	}

	for filename, content := range scripts {
		scriptPath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		log.Printf("Wrote script: %s", filename)
	}

	// Create the server config
	serverConfig := MCPServerConfig{
		Command: "node",
		Args:    []string{filepath.Join(tmpDir, "safe_inputs_mcp_server.cjs"), toolsConfigPath},
		Env:     make(map[string]string),
	}

	// Connect to the safe-inputs server
	client, session, err := g.createClient(ctx, "safeinputs", serverConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to safe-inputs server: %w", err)
	}

	g.mu.Lock()
	g.clients["safeinputs"] = client
	g.sessions["safeinputs"] = session
	g.mu.Unlock()

	log.Print("Successfully connected to safe-inputs server")
	return nil
}

// prepareToolsConfig reads the tools.json config and updates handler paths to be absolute
func (g *Gateway) prepareToolsConfig(configDir, outputPath string) error {
	// Read the original config
	data, err := os.ReadFile(g.config.SafeInputsConfig)
	if err != nil {
		return fmt.Errorf("failed to read tools config: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse tools config: %w", err)
	}

	// Update handler paths to be absolute
	if tools, ok := config["tools"].([]any); ok {
		for _, toolAny := range tools {
			if tool, ok := toolAny.(map[string]any); ok {
				if handler, ok := tool["handler"].(string); ok && handler != "" {
					// Convert relative paths to absolute
					if !filepath.IsAbs(handler) {
						absHandler := filepath.Join(configDir, handler)
						tool["handler"] = absHandler
						log.Printf("Updated handler path: %s -> %s", handler, absHandler)
					}
				}
			}
		}
	}

	// Write the updated config
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(outputPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated config: %w", err)
	}

	return nil
}

// createClient creates and connects an MCP client based on the server configuration
func (g *Gateway) createClient(ctx context.Context, name string, config MCPServerConfig) (*mcp.Client, *mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    fmt.Sprintf("gateway-client-%s", name),
		Version: "1.0.0",
	}, nil)

	var transport mcp.Transport
	var err error

	// Determine transport type based on configuration
	if config.URL != "" {
		// HTTP MCP server
		log.Printf("Creating HTTP transport for %s: %s", name, config.URL)
		transport, err = g.createHTTPTransport(config)
	} else if config.Command != "" {
		// Stdio MCP server
		log.Printf("Creating stdio transport for %s: command=%s", name, config.Command)
		transport, err = g.createStdioTransport(config)
	} else if config.Container != "" {
		// Docker container MCP server
		log.Printf("Creating Docker transport for %s: container=%s", name, config.Container)
		transport, err = g.createDockerTransport(config)
	} else {
		return nil, nil, fmt.Errorf("invalid server configuration: must specify url, command, or container")
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create transport: %w", err)
	}

	// Connect to the MCP server
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}

	log.Printf("Client connected to %s, session established", name)

	return client, session, nil
}

// createHTTPTransport creates an HTTP transport for an MCP server
func (g *Gateway) createHTTPTransport(config MCPServerConfig) (mcp.Transport, error) {
	return &mcp.SSEClientTransport{
		Endpoint: config.URL,
	}, nil
}

// createStdioTransport creates a stdio transport for an MCP server
func (g *Gateway) createStdioTransport(config MCPServerConfig) (mcp.Transport, error) {
	cmd := exec.Command(config.Command, config.Args...)

	// Set environment variables
	if len(config.Env) > 0 {
		cmd.Env = os.Environ()
		for key, value := range config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return &mcp.CommandTransport{
		Command: cmd,
	}, nil
}

// createDockerTransport creates a Docker transport for an MCP server
func (g *Gateway) createDockerTransport(config MCPServerConfig) (mcp.Transport, error) {
	// Build docker run command
	args := []string{"run", "--rm", "-i"}

	// Add environment variables
	for key, value := range config.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add container image
	args = append(args, config.Container)

	// Add container args
	args = append(args, config.Args...)

	cmd := exec.Command("docker", args...)

	return &mcp.CommandTransport{
		Command: cmd,
	}, nil
}

// toolMapping represents a mapping from a tool name to its server
type toolMapping struct {
	serverName string
	tool       *mcp.Tool
}

// listAllTools lists all tools from all connected servers
func (g *Gateway) listAllTools(ctx context.Context) (map[string]toolMapping, error) {
	log.Print("Listing all tools from connected servers")

	g.mu.RLock()
	defer g.mu.RUnlock()

	tools := make(map[string]toolMapping)

	for name, session := range g.sessions {
		log.Printf("Listing tools from server: %s", name)

		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			log.Printf("Failed to list tools from server %s: %v", name, err)
			continue
		}

		log.Printf("Server %s has %d tools", name, len(result.Tools))

		for _, tool := range result.Tools {
			// Handle tool name collisions by prefixing with server name
			toolName := tool.Name
			if _, exists := tools[toolName]; exists {
				log.Printf("Tool name collision detected: %s (from server %s)", toolName, name)
				toolName = fmt.Sprintf("%s.%s", name, tool.Name)
			}

			tools[toolName] = toolMapping{
				serverName: name,
				tool:       tool,
			}
			log.Printf("Registered tool: %s from server %s", toolName, name)
		}
	}

	return tools, nil
}

// registerToolHandlers registers tool handlers that proxy requests to backend servers
func (g *Gateway) registerToolHandlers(server *mcp.Server, tools map[string]toolMapping) {
	log.Printf("Registering %d gateway tool handlers", len(tools))

	for toolName, mapping := range tools {
		// Capture variables for closure
		serverName := mapping.serverName
		tool := mapping.tool

		log.Printf("Registering tool handler: %s -> server %s", toolName, serverName)

		// Create a handler that proxies to the backend server
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
			log.Printf("Proxying tool call: %s to server %s", toolName, serverName)

			g.mu.RLock()
			session, exists := g.sessions[serverName]
			g.mu.RUnlock()

			if !exists {
				log.Printf("Server not found: %s", serverName)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error: server %s not found", serverName)},
					},
				}, nil, nil
			}

			// Convert args back to map for CallTool
			var argsMap map[string]any
			if args != nil {
				// If args is already a map, use it directly
				if m, ok := args.(map[string]any); ok {
					argsMap = m
				} else {
					// Otherwise, marshal and unmarshal to convert
					jsonBytes, err := json.Marshal(args)
					if err != nil {
						log.Printf("Failed to marshal args: %v", err)
						return &mcp.CallToolResult{
							Content: []mcp.Content{
								&mcp.TextContent{Text: fmt.Sprintf("Error marshaling args: %v", err)},
							},
						}, nil, nil
					}
					if err := json.Unmarshal(jsonBytes, &argsMap); err != nil {
						log.Printf("Failed to unmarshal args: %v", err)
						return &mcp.CallToolResult{
							Content: []mcp.Content{
								&mcp.TextContent{Text: fmt.Sprintf("Error unmarshaling args: %v", err)},
							},
						}, nil, nil
					}
				}
			}

			// Call the tool on the backend server
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tool.Name, // Use original tool name without prefix
				Arguments: argsMap,
			})

			if err != nil {
				log.Printf("Tool call failed: %v", err)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error calling tool: %v", err)},
					},
				}, nil, nil
			}

			log.Printf("Tool call succeeded: %s", toolName)
			return result, nil, nil
		})
	}

	log.Print("All tool handlers registered")
}

// authMiddleware wraps an HTTP handler with API key authentication
func (g *Gateway) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for API key in Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Print("Authentication failed: missing Authorization header")
			http.Error(w, "Unauthorized: missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Support both "Bearer <token>" and plain token formats
		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		// Validate API key
		if token != g.config.APIKey {
			log.Print("Authentication failed: invalid API key")
			http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
			return
		}

		log.Print("Authentication successful")
		next.ServeHTTP(w, r)
	})
}

// Close closes all client connections
func (g *Gateway) Close() error {
	log.Print("Closing gateway and all client connections")

	g.mu.Lock()
	defer g.mu.Unlock()

	var errs []error
	for name, session := range g.sessions {
		log.Printf("Closing connection to server: %s", name)
		if err := session.Close(); err != nil {
			log.Printf("Error closing session for server %s: %v", name, err)
			errs = append(errs, fmt.Errorf("server %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}

	return nil
}

// LoadConfigFromJSON loads gateway configuration from a JSON file
func LoadConfigFromJSON(filename string) (GatewayConfig, error) {
	log.Printf("Loading gateway configuration from: %s", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return GatewayConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GatewayConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return GatewayConfig{}, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	log.Printf("Loaded configuration with %d MCP servers", len(config.MCPServers))
	return config, nil
}

// LoadConfigFromReader loads gateway configuration from an io.Reader
func LoadConfigFromReader(r io.Reader) (GatewayConfig, error) {
	log.Print("Loading gateway configuration from reader")

	data, err := io.ReadAll(r)
	if err != nil {
		return GatewayConfig{}, fmt.Errorf("failed to read config: %w", err)
	}

	var config GatewayConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return GatewayConfig{}, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	log.Printf("Loaded configuration with %d MCP servers", len(config.MCPServers))
	return config, nil
}
