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
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpLog = logger.New("mcp:server")

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
  - audit       - Investigate a workflow run and generate a report
  - mcp-inspect - Inspect MCP servers in workflows and list available tools
  - add         - Add workflows from remote repositories to .github/workflows
  - update      - Update workflows from their source repositories

By default, the server uses stdio transport. Use the --port flag to run
an HTTP server with SSE (Server-Sent Events) transport instead.

Examples:
  gh aw mcp-server              # Run with stdio transport
  gh aw mcp-server --port 8080  # Run HTTP server on port 8080
  gh aw mcp-server --cmd ./gh-aw # Use custom gh-aw binary`,
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
		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
				},
			}, nil, nil
		default:
		}

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
		Workflows  []string `json:"workflows,omitempty" jsonschema:"Workflow files to compile (empty for all)"`
		Strict     bool     `json:"strict,omitempty" jsonschema:"Override frontmatter to enforce strict mode validation for all workflows. Note: Workflows default to strict mode unless frontmatter sets strict: false"`
		Zizmor     bool     `json:"zizmor,omitempty" jsonschema:"Run zizmor security scanner on generated .lock.yml files"`
		Poutine    bool     `json:"poutine,omitempty" jsonschema:"Run poutine security scanner on generated .lock.yml files"`
		Actionlint bool     `json:"actionlint,omitempty" jsonschema:"Run actionlint linter on generated .lock.yml files"`
		Fix        bool     `json:"fix,omitempty" jsonschema:"Apply automatic codemod fixes to workflows before compiling"`
		JqFilter   string   `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name: "compile",
		Description: `Compile Markdown workflows to GitHub Actions YAML with optional static analysis tools.

Workflows use strict mode validation by default (unless frontmatter sets strict: false).
Strict mode enforces: action pinning to SHAs, explicit network config, safe-outputs for write operations,
and refuses write permissions and deprecated fields. Use the strict parameter to override frontmatter settings.

Returns JSON array with validation results for each workflow:
- workflow: Name of the workflow file
- valid: Boolean indicating if compilation was successful
- errors: Array of error objects with type, message, and optional line number
- warnings: Array of warning objects
- compiled_file: Path to the generated .lock.yml file

Note: Output can be filtered using the jq parameter.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args compileArgs) (*mcp.CallToolResult, any, error) {
		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
				},
			}, nil, nil
		default:
		}

		// Check if any static analysis tools are requested that require Docker images
		if args.Zizmor || args.Poutine || args.Actionlint {
			// Check if Docker images are available; if not, start downloading and return retry message
			if err := CheckAndPrepareDockerImages(args.Zizmor, args.Poutine, args.Actionlint); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: err.Error()},
					},
				}, nil, nil
			}

			// Check for cancellation after Docker image preparation
			select {
			case <-ctx.Done():
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
					},
				}, nil, nil
			default:
			}
		}

		// Build command arguments
		// Always validate workflows during compilation and use JSON output for MCP
		cmdArgs := []string{"compile", "--validate", "--json"}

		// Add fix flag if requested
		if args.Fix {
			cmdArgs = append(cmdArgs, "--fix")
		}

		// Add strict flag if requested
		if args.Strict {
			cmdArgs = append(cmdArgs, "--strict")
		}

		// Add static analysis flags if requested
		if args.Zizmor {
			cmdArgs = append(cmdArgs, "--zizmor")
		}
		if args.Poutine {
			cmdArgs = append(cmdArgs, "--poutine")
		}
		if args.Actionlint {
			cmdArgs = append(cmdArgs, "--actionlint")
		}

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

	// Add logs tool
	type logsArgs struct {
		WorkflowName string `json:"workflow_name,omitempty" jsonschema:"Name of the workflow to download logs for (empty for all)"`
		Count        int    `json:"count,omitempty" jsonschema:"Number of workflow runs to download (default: 100)"`
		StartDate    string `json:"start_date,omitempty" jsonschema:"Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		EndDate      string `json:"end_date,omitempty" jsonschema:"Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		Engine       string `json:"engine,omitempty" jsonschema:"Filter logs by agentic engine type (claude, codex, copilot)"`
		Firewall     bool   `json:"firewall,omitempty" jsonschema:"Filter to only runs with firewall enabled"`
		NoFirewall   bool   `json:"no_firewall,omitempty" jsonschema:"Filter to only runs without firewall enabled"`
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
		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
				},
			}, nil, nil
		default:
		}

		// Validate firewall parameters
		if args.Firewall && args.NoFirewall {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Error: cannot specify both 'firewall' and 'no_firewall' parameters"},
				},
			}, nil, nil
		}

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
		if args.Firewall {
			cmdArgs = append(cmdArgs, "--firewall")
		}
		if args.NoFirewall {
			cmdArgs = append(cmdArgs, "--no-firewall")
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
		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
				},
			}, nil, nil
		default:
		}

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
		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
				},
			}, nil, nil
		default:
		}

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

	// Add add tool
	type addArgs struct {
		Workflows []string `json:"workflows" jsonschema:"Workflows to add (e.g., 'owner/repo/workflow-name' or 'owner/repo/workflow-name@version')"`
		Number    int      `json:"number,omitempty" jsonschema:"Create multiple numbered copies (corresponds to -c flag, default: 1)"`
		Name      string   `json:"name,omitempty" jsonschema:"Specify name for the added workflow - without .md extension (corresponds to -n flag)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add",
		Description: "Add workflows from remote repositories to .github/workflows",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args addArgs) (*mcp.CallToolResult, any, error) {
		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
				},
			}, nil, nil
		default:
		}

		// Validate required arguments
		if len(args.Workflows) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Error: at least one workflow specification is required"},
				},
			}, nil, nil
		}

		// Build command arguments
		cmdArgs := []string{"add"}

		// Add workflows
		cmdArgs = append(cmdArgs, args.Workflows...)

		// Add optional flags
		if args.Number > 0 {
			cmdArgs = append(cmdArgs, "-c", strconv.Itoa(args.Number))
		}
		if args.Name != "" {
			cmdArgs = append(cmdArgs, "-n", args.Name)
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

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(output)},
			},
		}, nil, nil
	})

	// Add update tool
	type updateArgs struct {
		Workflows []string `json:"workflows,omitempty" jsonschema:"Workflow IDs to update (empty for all workflows)"`
		Major     bool     `json:"major,omitempty" jsonschema:"Allow major version updates when updating tagged releases"`
		Force     bool     `json:"force,omitempty" jsonschema:"Force update even if no changes detected"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "update",
		Description: `Update workflows from their source repositories and check for gh-aw updates.

The command:
1. Checks if a newer version of gh-aw is available
2. Updates workflows using the 'source' field in the workflow frontmatter
3. Compiles each workflow immediately after update

For workflow updates, it fetches the latest version based on the current ref:
- If the ref is a tag, it updates to the latest release (use major flag for major version updates)
- If the ref is a branch, it fetches the latest commit from that branch
- Otherwise, it fetches the latest commit from the default branch

Returns formatted text output showing:
- Extension update status
- Updated workflows with their new versions
- Compilation status for each updated workflow`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args updateArgs) (*mcp.CallToolResult, any, error) {
		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", ctx.Err())},
				},
			}, nil, nil
		default:
		}

		// Build command arguments
		cmdArgs := []string{"update"}

		// Add workflow IDs if specified
		cmdArgs = append(cmdArgs, args.Workflows...)

		// Add optional flags
		if args.Major {
			cmdArgs = append(cmdArgs, "--major")
		}
		if args.Force {
			cmdArgs = append(cmdArgs, "--force")
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
