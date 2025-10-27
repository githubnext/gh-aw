package cli

import (
	"context"
	"fmt"
	"log"
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
	var cmdPath string

	cmd := &cobra.Command{
		Use:   "mcp-server",
		Short: "Run an MCP (Model Context Protocol) server exposing gh-aw commands as tools",
		Long: `Run an MCP server that exposes gh-aw CLI commands as MCP tools.

This command starts an MCP server that wraps the gh-aw CLI, spawning subprocess
calls for each tool invocation. This design ensures that GitHub tokens and other
secrets are not shared with the MCP server process itself.

The server provides the following tools:
  - status      - Show status of agentic workflow files
  - compile     - Compile markdown workflow files to YAML
  - logs        - Download and analyze workflow logs
  - audit       - Investigate a workflow run and generate a report
  - mcp-inspect - Inspect MCP servers in workflows and list available tools

By default, the server uses stdio transport. Use the --port flag to run
an HTTP server with SSE (Server-Sent Events) transport instead.

Examples:
  gh aw mcp-server              # Run with stdio transport
  gh aw mcp-server --port 8080  # Run HTTP server on port 8080
  gh aw mcp-server --cmd ./gh-aw # Use custom gh-aw binary`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runMCPServer(port, cmdPath); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run HTTP server on (uses stdio if not specified)")
	cmd.Flags().StringVar(&cmdPath, "cmd", "", "Path to gh-aw command to use (defaults to 'gh aw')")

	return cmd
}

// runMCPServer starts the MCP server on stdio or HTTP transport
func runMCPServer(port int, cmdPath string) error {
	// Validate that the CLI and secrets are properly configured
	// Note: Validation failures are logged as warnings but don't prevent server startup
	// This allows the server to start in test environments or non-repository directories
	if err := validateMCPServerConfiguration(cmdPath); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Configuration validation warning: %v", err)))
	}

	// Create the server configuration
	server := createMCPServer(cmdPath)

	if port > 0 {
		// Run HTTP server with SSE transport
		return runHTTPServer(server, port)
	}

	// Run stdio transport
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// validateMCPServerConfiguration validates that the CLI is properly configured
// by running the status command as a test
func validateMCPServerConfiguration(cmdPath string) error {
	// Try to run the status command to verify CLI is working
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if cmdPath != "" {
		// Use custom command path
		cmd = exec.CommandContext(ctx, cmdPath, "status")
	} else {
		// Use default gh aw command
		cmd = exec.CommandContext(ctx, "gh", "aw", "status")
	}
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check for common error cases
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("status command timed out - this may indicate a configuration issue")
		}

		// If the command failed, provide helpful error message
		if cmdPath != "" {
			return fmt.Errorf("failed to run status command with custom command '%s': %w\nOutput: %s\n\nPlease ensure:\n  - The command path is correct and executable\n  - You are in a git repository with .github/workflows directory", cmdPath, err, string(output))
		}
		return fmt.Errorf("failed to run status command: %w\nOutput: %s\n\nPlease ensure:\n  - gh CLI is installed and in PATH\n  - gh aw extension is installed (run: gh extension install githubnext/gh-aw)\n  - You are in a git repository with .github/workflows directory", err, string(output))
	}

	// Status command succeeded - configuration is valid
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✅ Configuration validated successfully"))
	return nil
}

// createMCPServer creates and configures the MCP server with all tools
func createMCPServer(cmdPath string) *mcp.Server {
	// Helper function to execute command with proper path
	execCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		if cmdPath != "" {
			// Use custom command path
			return exec.CommandContext(ctx, cmdPath, args...)
		}
		// Use default gh aw command
		return exec.CommandContext(ctx, "gh", append([]string{"aw"}, args...)...)
	}

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gh-aw",
		Version: GetVersion(),
	}, nil)

	// Add status tool
	type statusArgs struct {
		Pattern  string `json:"pattern,omitempty" jsonschema:"Optional pattern to filter workflows by name"`
		JqFilter string `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "status",
		Description: `Show status of agentic workflow files and workflows.

Returns a JSON array where each element has the following structure:
- workflow: Name of the workflow file
- agent: AI engine used (e.g., "copilot", "claude", "codex")
- compiled: Whether the workflow is compiled ("Yes", "No", or "N/A")
- status: GitHub workflow status ("active", "disabled", "Unknown")
- time_remaining: Time remaining until workflow deadline (if applicable)

Note: Output can be filtered using the jq parameter.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments - always use JSON for MCP
		cmdArgs := []string{"status", "--json"}
		if args.Pattern != "" {
			cmdArgs = append(cmdArgs, args.Pattern)
		}

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))},
				},
			}, nil, nil
		}

		// Apply jq filter if provided
		outputStr := string(output)
		if args.JqFilter != "" {
			filteredOutput, jqErr := ApplyJqFilter(outputStr, args.JqFilter)
			if jqErr != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error applying jq filter: %v", jqErr)},
					},
				}, nil, nil
			}
			outputStr = filteredOutput
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
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
		// Always validate workflows during compilation
		cmdArgs := []string{"compile", "--validate"}
		cmdArgs = append(cmdArgs, args.Workflows...)

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
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
		Count        int    `json:"count,omitempty" jsonschema:"Number of workflow runs to download (default: 100)"`
		StartDate    string `json:"start_date,omitempty" jsonschema:"Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		EndDate      string `json:"end_date,omitempty" jsonschema:"Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		Engine       string `json:"engine,omitempty" jsonschema:"Filter logs by agentic engine type (claude, codex, copilot)"`
		Branch       string `json:"branch,omitempty" jsonschema:"Filter runs by branch name"`
		AfterRunID   int64  `json:"after_run_id,omitempty" jsonschema:"Filter runs with database ID after this value (exclusive)"`
		BeforeRunID  int64  `json:"before_run_id,omitempty" jsonschema:"Filter runs with database ID before this value (exclusive)"`
		Timeout      int    `json:"timeout,omitempty" jsonschema:"Maximum time in seconds to spend downloading logs (default: 50 for MCP server)"`
		JqFilter     string `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
		MaxTokens    int    `json:"max_tokens,omitempty" jsonschema:"Maximum number of tokens in output before triggering guardrail (default: 12000)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name: "logs",
		Description: `Download and analyze workflow logs.

Returns JSON with workflow run data and metrics. If the command times out before fetching all available logs, 
a "continuation" field will be present in the response with updated parameters to continue fetching more data.
Check for the presence of the continuation field to determine if there are more logs available.

The continuation field includes all necessary parameters (before_run_id, etc.) to resume fetching from where 
the previous request stopped due to timeout.

⚠️  Output Size Guardrail: If the output exceeds the token limit (default: 12000 tokens), the tool will 
return a schema description and suggested jq filters instead of the full output. Use the 'jq' parameter 
to filter the output to a manageable size, or adjust the 'max_tokens' parameter. Common filters include:
  - .summary (get only summary statistics)
  - .runs[:5] (get first 5 runs)
  - .runs | map(select(.conclusion == "failure")) (get only failed runs)`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		// Force output directory to /tmp/gh-aw/aw-mcp/logs for MCP server
		cmdArgs := []string{"logs", "-o", "/tmp/gh-aw/aw-mcp/logs"}
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

		// Set timeout to 50 seconds for MCP server if not explicitly specified
		timeoutValue := args.Timeout
		if timeoutValue == 0 {
			timeoutValue = 50
		}
		cmdArgs = append(cmdArgs, "--timeout", strconv.Itoa(timeoutValue))

		// Always use --json mode in MCP server
		cmdArgs = append(cmdArgs, "--json")

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))},
				},
			}, nil, nil
		}

		// Apply jq filter if provided
		outputStr := string(output)
		if args.JqFilter != "" {
			filteredOutput, err := ApplyJqFilter(outputStr, args.JqFilter)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error applying jq filter: %v", err)},
					},
				}, nil, nil
			}
			outputStr = filteredOutput
		}

		// Check output size and apply guardrail if needed
		finalOutput, _ := checkLogsOutputSize(outputStr, args.MaxTokens)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: finalOutput},
			},
		}, nil, nil
	})

	// Add audit tool
	type auditArgs struct {
		RunID    int64  `json:"run_id" jsonschema:"GitHub Actions workflow run ID to audit"`
		JqFilter string `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "audit",
		Description: `Investigate a workflow run and generate a concise report.

Returns JSON with the following structure:
- overview: Basic run information (run_id, workflow_name, status, conclusion, created_at, started_at, updated_at, duration, event, branch, url)
- metrics: Execution metrics (token_usage, estimated_cost, turns, error_count, warning_count)
- jobs: List of job details (name, status, conclusion, duration)
- downloaded_files: List of artifact files (path, size, size_formatted, description, is_directory)
- missing_tools: Tools that were requested but not available (tool, reason, alternatives, timestamp, workflow_name, run_id)
- mcp_failures: MCP server failures (server_name, status, timestamp, workflow_name, run_id)
- errors: Error details (file, line, type, message)
- warnings: Warning details (file, line, type, message)
- tool_usage: Tool usage statistics (name, call_count, max_output_size, max_duration)

Note: Output can be filtered using the jq parameter.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args auditArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		// Force output directory to /tmp/gh-aw/aw-mcp/logs for MCP server (same as logs)
		// Use --json flag to output structured JSON for MCP consumption
		cmdArgs := []string{"audit", strconv.FormatInt(args.RunID, 10), "-o", "/tmp/gh-aw/aw-mcp/logs", "--json"}

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))},
				},
			}, nil, nil
		}

		// Apply jq filter if provided
		outputStr := string(output)
		if args.JqFilter != "" {
			filteredOutput, jqErr := ApplyJqFilter(outputStr, args.JqFilter)
			if jqErr != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error applying jq filter: %v", jqErr)},
					},
				}, nil, nil
			}
			outputStr = filteredOutput
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})

	// Add mcp-inspect tool
	type mcpInspectArgs struct {
		WorkflowFile string `json:"workflow_file,omitempty" jsonschema:"Workflow file to inspect MCP servers from (empty to list all workflows with MCP servers)"`
		Server       string `json:"server,omitempty" jsonschema:"Filter to inspect only the specified MCP server"`
		Tool         string `json:"tool,omitempty" jsonschema:"Show detailed information about a specific tool (requires server parameter)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "mcp-inspect",
		Description: `Inspect MCP servers used by a workflow and list available tools, resources, and roots.

This tool starts each MCP server configured in the workflow, queries its capabilities,
and displays the results. It supports stdio, Docker, and HTTP MCP servers.

Secret checking is enabled by default to validate GitHub Actions secrets availability.
If GitHub token is not available or has no permissions, secret checking is silently skipped.

When called without workflow_file, lists all workflows that contain MCP server configurations.
When called with workflow_file, inspects the MCP servers in that specific workflow.

Use the server parameter to filter to a specific MCP server.
Use the tool parameter (requires server) to show detailed information about a specific tool.

Returns formatted text output showing:
- Available MCP servers in the workflow
- Tools, resources, and roots exposed by each server
- Secret availability status (if GitHub token is available)
- Detailed tool information when tool parameter is specified`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpInspectArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		cmdArgs := []string{"mcp", "inspect"}

		if args.WorkflowFile != "" {
			cmdArgs = append(cmdArgs, args.WorkflowFile)
		}

		if args.Server != "" {
			cmdArgs = append(cmdArgs, "--server", args.Server)
		}

		if args.Tool != "" {
			cmdArgs = append(cmdArgs, "--tool", args.Tool)
		}

		// Always enable secret checking (will be silently ignored if GitHub token is not available)
		cmdArgs = append(cmdArgs, "--check-secrets")

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
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

		// Log request details.
		log.Printf("[REQUEST] %s | %s | %s %s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details.
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration)
	})
}

// runHTTPServer runs the MCP server with HTTP/SSE transport
func runHTTPServer(server *mcp.Server, port int) error {
	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	handlerWithLogging := loggingHandler(handler)

	// Create HTTP server
	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handlerWithLogging,
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP server on http://localhost%s", addr)))

	// Run the HTTP server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}
