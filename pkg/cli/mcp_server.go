package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// NewMCPServerCommand creates the mcp-server command
func NewMCPServerCommand() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "mcp-server",
		Short: "Run an MCP (Model Context Protocol) server exposing gh-aw commands as tools",
		Long: `Run an MCP server that exposes gh-aw CLI commands as MCP tools.

This command starts an MCP server that wraps the gh-aw CLI, spawning subprocess
calls for each tool invocation. This design ensures that GitHub tokens and other
secrets are not shared with the MCP server process itself.

The server provides the following tools:
  - status   - Show status of agentic workflow files
  - compile  - Compile markdown workflow files to YAML
  - logs     - Download and analyze workflow logs
  - audit    - Investigate a workflow run and generate a report

By default, the server uses stdio transport. Use the --port flag to run
an HTTP server with SSE (Server-Sent Events) transport instead.

Examples:
  gh aw mcp-server              # Run with stdio transport
  gh aw mcp-server --port 8080  # Run HTTP server on port 8080`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runMCPServer(port); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run HTTP server on (uses stdio if not specified)")

	return cmd
}

// runMCPServer starts the MCP server on stdio or HTTP transport
func runMCPServer(port int) error {
	// Validate that the CLI and secrets are properly configured
	if err := validateMCPServerConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create the server configuration
	server := createMCPServer()

	if port > 0 {
		// Run HTTP server with SSE transport
		return runHTTPServer(server, port)
	}

	// Run stdio transport
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// validateMCPServerConfiguration validates that the CLI is properly configured
// by running the status command as a test
func validateMCPServerConfiguration() error {
	// Allow skipping validation via environment variable (useful for tests)
	if os.Getenv("GH_AW_SKIP_MCP_VALIDATION") == "1" {
		return nil
	}

	// Try to run the status command to verify CLI is working
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "aw", "status")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check for common error cases
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("status command timed out - this may indicate a configuration issue")
		}

		// If the command failed, provide helpful error message
		return fmt.Errorf("failed to run status command: %w\nOutput: %s\n\nPlease ensure:\n  - gh CLI is installed and in PATH\n  - gh aw extension is installed (run: gh extension install githubnext/gh-aw)\n  - You are in a git repository with .github/workflows directory", err, string(output))
	}

	// Status command succeeded - configuration is valid
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✅ Configuration validated successfully"))
	return nil
}

// createMCPServer creates and configures the MCP server with all tools
func createMCPServer() *mcp.Server {
	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gh-aw",
		Version: GetVersion(),
	}, nil)

	// Add status tool
	type statusArgs struct {
		Pattern string `json:"pattern,omitempty" jsonschema:"Optional pattern to filter workflows by name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "status",
		Description: "Show status of agentic workflow files and workflows",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		cmdArgs := []string{"aw", "status"}
		if args.Pattern != "" {
			cmdArgs = append(cmdArgs, args.Pattern)
		}

		// Execute the CLI command
		cmd := exec.CommandContext(ctx, "gh", cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(output)},
			},
		}, nil, nil
	})

	// Add compile tool
	type compileArgs struct {
		Workflows []string `json:"workflows,omitempty" jsonschema:"Workflow files to compile (empty for all)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "compile",
		Description: "Compile markdown workflow files to YAML workflows",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args compileArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		// Always validate (validation is enabled by default)
		cmdArgs := []string{"aw", "compile"}
		cmdArgs = append(cmdArgs, args.Workflows...)

		// Execute the CLI command
		cmd := exec.CommandContext(ctx, "gh", cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(output)},
			},
		}, nil, nil
	})

	// Add logs tool
	type logsArgs struct {
		WorkflowName string `json:"workflow_name,omitempty" jsonschema:"Name of the workflow to download logs for (empty for all)"`
		Count        int    `json:"count,omitempty" jsonschema:"Number of workflow runs to download"`
		StartDate    string `json:"start_date,omitempty" jsonschema:"Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		EndDate      string `json:"end_date,omitempty" jsonschema:"Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		Engine       string `json:"engine,omitempty" jsonschema:"Filter logs by agentic engine type (claude, codex, copilot)"`
		Branch       string `json:"branch,omitempty" jsonschema:"Filter runs by branch name"`
		AfterRunID   int64  `json:"after_run_id,omitempty" jsonschema:"Filter runs with database ID after this value (exclusive)"`
		BeforeRunID  int64  `json:"before_run_id,omitempty" jsonschema:"Filter runs with database ID before this value (exclusive)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "logs",
		Description: "Download and analyze workflow logs",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		// Force output directory to /tmp/gh-aw/aw-mcp/logs for MCP server
		cmdArgs := []string{"aw", "logs", "-o", "/tmp/gh-aw/aw-mcp/logs"}
		if args.WorkflowName != "" {
			cmdArgs = append(cmdArgs, args.WorkflowName)
		}
		if args.Count > 0 {
			cmdArgs = append(cmdArgs, "-c", strconv.Itoa(args.Count))
		}
		if args.StartDate != "" {
			cmdArgs = append(cmdArgs, "--start-date", args.StartDate)
		}
		if args.EndDate != "" {
			cmdArgs = append(cmdArgs, "--end-date", args.EndDate)
		}
		if args.Engine != "" {
			cmdArgs = append(cmdArgs, "--engine", args.Engine)
		}
		if args.Branch != "" {
			cmdArgs = append(cmdArgs, "--branch", args.Branch)
		}
		if args.AfterRunID > 0 {
			cmdArgs = append(cmdArgs, "--after-run-id", strconv.FormatInt(args.AfterRunID, 10))
		}
		if args.BeforeRunID > 0 {
			cmdArgs = append(cmdArgs, "--before-run-id", strconv.FormatInt(args.BeforeRunID, 10))
		}

		// Execute the CLI command
		cmd := exec.CommandContext(ctx, "gh", cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(output)},
			},
		}, nil, nil
	})

	// Add audit tool
	type auditArgs struct {
		RunID int64 `json:"run_id" jsonschema:"GitHub Actions workflow run ID to audit"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "audit",
		Description: "Investigate a workflow run and generate a concise report",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args auditArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		// Force output directory to /tmp/gh-aw/aw-mcp/logs for MCP server (same as logs)
		cmdArgs := []string{"aw", "audit", strconv.FormatInt(args.RunID, 10), "-o", "/tmp/gh-aw/aw-mcp/logs"}

		// Execute the CLI command
		cmd := exec.CommandContext(ctx, "gh", cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(output)},
			},
		}, nil, nil
	})

	return server
}

// runHTTPServer runs the MCP server with HTTP/SSE transport
func runHTTPServer(server *mcp.Server, port int) error {
	// Create SSE handler
	handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server {
		return server
	})

	// Create HTTP server
	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP server on http://localhost%s", addr)))

	// Run the HTTP server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}
