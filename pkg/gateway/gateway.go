package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/githubnext/gh-aw/pkg/logger"
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
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	Port       int                        `json:"port"`
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
	log.Printf("Creating new gateway with %d MCP servers on port %d", len(config.MCPServers), config.Port)

	if config.Port == 0 {
		return nil, fmt.Errorf("gateway port must be specified")
	}

	if len(config.MCPServers) == 0 {
		return nil, fmt.Errorf("no MCP servers configured")
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

	// Connect to all MCP servers
	if err := g.connectToServers(ctx); err != nil {
		return fmt.Errorf("failed to connect to MCP servers: %w", err)
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
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

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
