package cli

import (
	"context"
	"encoding/json"
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
  - status   - Show status of agentic workflow files
  - compile  - Compile markdown workflow files to YAML
  - logs     - Download and analyze workflow logs
  - audit    - Investigate a workflow run and generate a report

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

	// Note: Output schema not set for status tool because it returns an array directly,
	// but MCP requires output schemas to be objects. The helper GenerateOutputSchema
	// is available for future use if the status command is modified to return an object.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "status",
		Description: "Show status of agentic workflow files and workflows",
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

		// Return text content (output schema not supported for array return types)
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
		// Always validate (validation is enabled by default)
		cmdArgs := []string{"compile"}
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
		Count        int    `json:"count,omitempty" jsonschema:"Number of workflow runs to download"`
		StartDate    string `json:"start_date,omitempty" jsonschema:"Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		EndDate      string `json:"end_date,omitempty" jsonschema:"Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		Engine       string `json:"engine,omitempty" jsonschema:"Filter logs by agentic engine type (claude, codex, copilot)"`
		Branch       string `json:"branch,omitempty" jsonschema:"Filter runs by branch name"`
		AfterRunID   int64  `json:"after_run_id,omitempty" jsonschema:"Filter runs with database ID after this value (exclusive)"`
		BeforeRunID  int64  `json:"before_run_id,omitempty" jsonschema:"Filter runs with database ID before this value (exclusive)"`
		JqFilter     string `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}

	// Generate output schema for logs tool
	logsOutputSchema, schemaErr := GenerateOutputSchema[LogsData]()
	if schemaErr != nil {
		// Log error but don't fail server startup
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate output schema for logs tool: %v", schemaErr)))
		logsOutputSchema = nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:         "logs",
		Description:  "Download and analyze workflow logs",
		OutputSchema: logsOutputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, *LogsData, error) {
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

		// Parse the JSON output into structured data
		var logsData LogsData
		if parseErr := json.Unmarshal([]byte(outputStr), &logsData); parseErr != nil {
			// If parsing fails, return text content only (graceful degradation)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: outputStr},
				},
			}, nil, nil
		}

		// Return both text and structured content
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, &logsData, nil
	})

	// Add audit tool
	type auditArgs struct {
		RunID    int64  `json:"run_id" jsonschema:"GitHub Actions workflow run ID to audit"`
		JqFilter string `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}

	// Generate output schema for audit tool
	auditOutputSchema, schemaErr := GenerateOutputSchema[AuditData]()
	if schemaErr != nil {
		// Log error but don't fail server startup
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate output schema for audit tool: %v", schemaErr)))
		auditOutputSchema = nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:         "audit",
		Description:  "Investigate a workflow run and generate a concise report",
		OutputSchema: auditOutputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args auditArgs) (*mcp.CallToolResult, *AuditData, error) {
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

		// Parse the JSON output into structured data
		var auditData AuditData
		if parseErr := json.Unmarshal([]byte(outputStr), &auditData); parseErr != nil {
			// If parsing fails, return text content only (graceful degradation)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: outputStr},
				},
			}, nil, nil
		}

		// Return both text and structured content
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, &auditData, nil
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
