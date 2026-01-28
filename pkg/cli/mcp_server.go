package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpLog = logger.New("mcp:server")

// mcpErrorData marshals data to JSON for use in jsonrpc.Error.Data field.
// Returns nil if marshaling fails to avoid errors in error handling.
func mcpErrorData(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		// Log the error but return nil to avoid breaking error handling
		mcpLog.Printf("Failed to marshal error data: %v", err)
		return nil
	}
	return data
}

// NewMCPServerCommand creates the mcp-server command
func NewMCPServerCommand() *cobra.Command {
	var port int
	var cmdPath string

	cmd := &cobra.Command{
		Use:   "mcp-server",
		Short: "Run an MCP (Model Context Protocol) server exposing gh aw commands as tools",
		Long: `Run an MCP server that exposes gh aw CLI commands as MCP tools.

This command starts an MCP server that wraps the gh aw CLI, spawning subprocess
calls for each tool invocation. This design ensures that GitHub tokens and other
secrets are not shared with the MCP server process itself.

The server provides the following tools:
  - status      - Show status of agentic workflow files
  - compile     - Compile Markdown workflows to GitHub Actions YAML
  - logs        - Download and analyze workflow logs
  - audit       - Investigate a workflow run, job, or step and generate a report
  - mcp-inspect - Inspect MCP servers in workflows and list available tools
  - add         - Add workflows from remote repositories to .github/workflows
  - update      - Update workflows from their source repositories
  - fix         - Apply automatic codemod-style fixes to workflow files

By default, the server uses stdio transport. Use the --port flag to run
an HTTP server with SSE (Server-Sent Events) transport instead.

Examples:
  gh aw mcp-server                    # Run with stdio transport (default for MCP clients)
  gh aw mcp-server --port 8080        # Run HTTP server on port 8080 (for web-based clients)
  gh aw mcp-server --cmd ./gh-aw      # Use custom gh-aw binary path
  DEBUG=mcp:* gh aw mcp-server        # Run with verbose logging for debugging`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer(port, cmdPath)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run HTTP server on (uses stdio if not specified)")
	cmd.Flags().StringVar(&cmdPath, "cmd", "", "Path to gh aw command to use (defaults to 'gh aw')")

	return cmd
}

// runMCPServer starts the MCP server on stdio or HTTP transport
func runMCPServer(port int, cmdPath string) error {
	if port > 0 {
		mcpLog.Printf("Starting MCP server on HTTP port %d", port)
	} else {
		mcpLog.Print("Starting MCP server with stdio transport")
	}

	// Validate that the CLI and secrets are properly configured
	// Note: Validation failures are logged as warnings but don't prevent server startup
	// This allows the server to start in test environments or non-repository directories
	if err := validateMCPServerConfiguration(cmdPath); err != nil {
		mcpLog.Printf("Configuration validation warning: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Configuration validation warning: %v", err)))
	}

	// Create the server configuration
	server := createMCPServer(cmdPath)

	if port > 0 {
		// Run HTTP server with SSE transport
		return runHTTPServer(server, port)
	}

	// Run stdio transport
	mcpLog.Print("MCP server ready on stdio")
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// createMCPServer creates and configures the MCP server with all tools
func createMCPServer(cmdPath string) *mcp.Server {
	// Helper function to execute command with proper path
	execCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		if cmdPath != "" {
			// Use custom command path
			return exec.CommandContext(ctx, cmdPath, args...)
		}
		// Use default gh aw command with proper token handling
		return workflow.ExecGHContext(ctx, append([]string{"aw"}, args...)...)
	}

	// Create MCP server with capabilities and logging
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gh-aw",
		Version: GetVersion(),
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{
				ListChanged: false, // Tools are static, no notifications needed
			},
		},
		Logger: logger.NewSlogLoggerWithHandler(mcpLog),
	})

	// Register all tools using extracted functions
	addStatusTool(server, execCmd)
	addCompileTool(server, execCmd)
	addLogsTool(server, execCmd)
	addAuditTool(server, execCmd)
	addMCPInspectTool(server, execCmd)
	addAddTool(server, execCmd)
	addUpdateTool(server, execCmd)
	addFixTool(server, execCmd)

	return server
}

// sanitizeForLog removes newline and carriage return characters from user input
// to prevent log injection attacks where malicious users could forge log entries.
func sanitizeForLog(input string) string {
	// Remove both \n and \r to prevent log injection
	sanitized := strings.ReplaceAll(input, "\n", "")
	sanitized = strings.ReplaceAll(sanitized, "\r", "")
	return sanitized
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Sanitize user-controlled input before logging to prevent log injection
		sanitizedPath := sanitizeForLog(r.URL.Path)

		// Log request details.
		log.Printf("[REQUEST] %s | %s | %s %s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			sanitizedPath)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details.
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			sanitizedPath,
			wrapped.statusCode,
			duration)
	})
}

// runHTTPServer runs the MCP server with HTTP/SSE transport
func runHTTPServer(server *mcp.Server, port int) error {
	mcpLog.Printf("Creating HTTP server on port %d", port)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 2 * time.Hour, // Close idle sessions after 2 hours
		Logger:         logger.NewSlogLoggerWithHandler(mcpLog),
	})

	handlerWithLogging := loggingHandler(handler)

	// Create HTTP server
	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handlerWithLogging,
		ReadHeaderTimeout: MCPServerHTTPTimeout,
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP server on http://localhost%s", addr)))
	mcpLog.Printf("HTTP server listening on %s", addr)

	// Run the HTTP server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		mcpLog.Printf("HTTP server failed: %v", err)
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}
