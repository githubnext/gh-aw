package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// NewMCPBridgeCommand creates the mcp bridge command
func NewMCPBridgeCommand() *cobra.Command {
	var port int
	var command string
	var args []string

	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "Bridge a stdio MCP server to HTTP transport",
		Long: `Bridge converts a stdio-based MCP server to HTTP transport with SSE (Server-Sent Events).

This enables better process isolation by allowing HTTP clients to connect to 
stdio MCP servers without spawning subprocesses themselves. The bridge:

1. Starts the stdio MCP server once as a subprocess
2. Creates a proxy that bridges HTTP requests to the stdio server
3. Exposes an HTTP endpoint with SSE transport
4. Manages the lifecycle of the stdio server process

This is useful for:
- Running stdio MCP servers in containerized environments
- Providing network-accessible MCP endpoints
- Improving security through process isolation
- Enabling multiple HTTP clients to connect to a single stdio server

Examples:
  # Bridge a simple stdio server
  gh aw mcp bridge --command "npx" --args "@my/mcp-server" --port 8080

  # Bridge with multiple arguments
  gh aw mcp bridge --command "python" --args "server.py,--verbose" --port 3000

  # Bridge a custom server
  gh aw mcp bridge --command "./my-server" --port 9000`,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			if command == "" {
				return fmt.Errorf("--command is required")
			}
			if port <= 0 {
				return fmt.Errorf("--port must be a positive number")
			}
			return runMCPBridge(port, command, args)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run HTTP server on (required)")
	cmd.Flags().StringVarP(&command, "command", "c", "", "Command to run the stdio MCP server (required)")
	cmd.Flags().StringSliceVarP(&args, "args", "a", []string{}, "Arguments for the stdio MCP server command (comma-separated)")

	cmd.MarkFlagRequired("port")
	cmd.MarkFlagRequired("command")

	return cmd
}

// runMCPBridge starts the stdio-to-HTTP bridge
func runMCPBridge(port int, command string, args []string) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting stdio-to-HTTP MCP bridge on port %d", port)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Stdio server: %s %v", command, args)))

	// Validate that the command exists
	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("command not found: %s (%w)", command, err)
	}

	// Create context for the stdio server process
	ctx := context.Background()

	// Start the stdio MCP server as a subprocess
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()

	// Create MCP client to connect to the stdio server
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-bridge-client",
		Version: "1.0.0",
	}, nil)

	// Create transport for the stdio server
	transport := &mcp.CommandTransport{Command: cmd}

	// Connect to the stdio server
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Connecting to stdio MCP server..."))
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to stdio server: %w", err)
	}
	defer session.Close()

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully connected to stdio MCP server"))

	// Create a proxy MCP server that forwards requests to the stdio server
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Creating HTTP bridge proxy..."))
	proxyServer, err := createProxyMCPServer(session)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Proxy server created successfully"))

	// Create HTTP handler for SSE transport
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return proxyServer
	}, nil)

	// Start the HTTP server
	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("MCP bridge server listening on http://localhost%s", addr)))

	// Run the HTTP server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}

// createProxyMCPServer creates an MCP server that proxies requests to a client session
func createProxyMCPServer(session *mcp.ClientSession) (*mcp.Server, error) {
	// Create a new MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-bridge-proxy",
		Version: "1.0.0",
	}, nil)

	// Discover and register all tools from the stdio server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List all tools from the stdio server
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools from stdio server: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Discovered %d tools from stdio server", len(toolsResult.Tools))))

	// Add each tool to the proxy server with a forwarding handler
	for _, tool := range toolsResult.Tools {
		// Capture tool name for closure
		toolName := tool.Name

		// Add the tool with a handler that forwards to the stdio server
		server.AddTool(tool, func(ctx context.Context, req *mcp.ServerRequest[*mcp.CallToolParamsRaw]) (*mcp.CallToolResult, error) {
			// Forward the tool call to the stdio server
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      toolName,
				Arguments: req.Params.Arguments,
			})
			return result, err
		})
	}

	// Similarly discover and register prompts
	promptsResult, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err == nil && len(promptsResult.Prompts) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Discovered %d prompts from stdio server", len(promptsResult.Prompts))))

		for _, prompt := range promptsResult.Prompts {
			// Add the prompt with a handler that forwards to the stdio server
			server.AddPrompt(prompt, func(ctx context.Context, req *mcp.ServerRequest[*mcp.GetPromptParams]) (*mcp.GetPromptResult, error) {
				// Forward the prompt request to the stdio server
				result, err := session.GetPrompt(ctx, req.Params)
				if err != nil {
					return nil, err
				}
				return result, nil
			})
		}
	}

	// Similarly discover and register resources
	resourcesResult, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err == nil && len(resourcesResult.Resources) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Discovered %d resources from stdio server", len(resourcesResult.Resources))))

		for _, resource := range resourcesResult.Resources {
			// Add the resource with a handler that forwards to the stdio server
			server.AddResource(resource, func(ctx context.Context, req *mcp.ServerRequest[*mcp.ReadResourceParams]) (*mcp.ReadResourceResult, error) {
				// Forward the resource read request to the stdio server
				result, err := session.ReadResource(ctx, req.Params)
				if err != nil {
					return nil, err
				}
				return result, nil
			})
		}
	}

	return server, nil
}
