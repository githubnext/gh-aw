package cli

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerLogsTool registers the logs tool with the MCP server.
// The logs tool requires write+ access and checks actor permissions.
// Returns an error if schema generation fails.
func registerLogsTool(server *mcp.Server, execCmd execCmdFunc, actor string, validateActor bool) error {
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
		MaxTokens    int    `json:"max_tokens,omitempty" jsonschema:"Maximum number of tokens in output before triggering guardrail (default: 12000)"`
	}

	// Generate schema with elicitation defaults
	logsSchema, err := GenerateSchema[logsArgs]()
	if err != nil {
		mcpLog.Printf("Failed to generate logs tool schema: %v", err)
		return err
	}
	// Add elicitation defaults for common parameters
	if err := AddSchemaDefault(logsSchema, "count", 100); err != nil {
		mcpLog.Printf("Failed to add default for count: %v", err)
	}
	if err := AddSchemaDefault(logsSchema, "timeout", 50); err != nil {
		mcpLog.Printf("Failed to add default for timeout: %v", err)
	}
	if err := AddSchemaDefault(logsSchema, "max_tokens", 12000); err != nil {
		mcpLog.Printf("Failed to add default for max_tokens: %v", err)
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "logs",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(true),
		},
		Description: `Download and analyze workflow logs.

Returns JSON with workflow run data and metrics. If the command times out before fetching all available logs, 
a "continuation" field will be present in the response with updated parameters to continue fetching more data.
Check for the presence of the continuation field to determine if there are more logs available.

The continuation field includes all necessary parameters (before_run_id, etc.) to resume fetching from where 
the previous request stopped due to timeout.

⚠️  Output Size Guardrail: If the output exceeds the token limit (default: 12000 tokens), the tool will 
return a schema description instead of the full output. Adjust the 'max_tokens' parameter to control this behavior.`,
		InputSchema: logsSchema,
		Icons: []mcp.Icon{
			{Source: "📜"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, any, error) {
		// Check actor permissions first
		if err := checkActorPermission(ctx, actor, validateActor, "logs"); err != nil {
			return nil, nil, err
		}

		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "request cancelled",
				Data:    mcpErrorData(ctx.Err().Error()),
			}
		default:
		}

		// Validate firewall parameters
		if args.Firewall && args.NoFirewall {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: "conflicting parameters: cannot specify both 'firewall' and 'no_firewall'",
				Data:    nil,
			}
		}

		// Validate workflow name before executing command
		if err := validateWorkflowName(args.WorkflowName); err != nil {
			mcpLog.Printf("Workflow name validation failed: %v", err)
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: err.Error(),
				Data: mcpErrorData(map[string]any{
					"workflow_name": args.WorkflowName,
					"error_type":    "workflow_not_found",
				}),
			}
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

		// Log the command being executed for debugging
		mcpLog.Printf("Executing logs tool: workflow=%s, count=%d, firewall=%v, no_firewall=%v, timeout=%d, command_args=%v",
			args.WorkflowName, args.Count, args.Firewall, args.NoFirewall, timeoutValue, cmdArgs)

		// Execute the CLI command
		// Use separate stdout/stderr capture instead of CombinedOutput because:
		// - Stdout contains JSON output (--json flag)
		// - Stderr contains console messages and error details
		cmd := execCmd(ctx, cmdArgs...)
		stdout, err := cmd.Output()

		// The logs command outputs JSON to stdout when --json flag is used.
		// If the command fails, we need to provide detailed error information.
		outputStr := string(stdout)

		if err != nil {
			// Try to get stderr and exit code for detailed error reporting
			var stderr string
			var exitCode int
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				stderr = string(exitErr.Stderr)
				exitCode = exitErr.ExitCode()
			}

			mcpLog.Printf("Logs command exited with error: %v (stdout length: %d, stderr length: %d, exit_code: %d)",
				err, len(outputStr), len(stderr), exitCode)

			// Build detailed error data
			errorData := map[string]any{
				"error":     err.Error(),
				"command":   strings.Join(cmdArgs, " "),
				"exit_code": exitCode,
				"stdout":    outputStr,
				"stderr":    stderr,
				"timeout":   timeoutValue,
				"workflow":  args.WorkflowName,
			}

			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to download workflow logs: " + err.Error(),
				Data:    mcpErrorData(errorData),
			}
		}

		// Check output size and apply guardrail if needed
		finalOutput, _ := checkLogsOutputSize(outputStr, args.MaxTokens)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: finalOutput},
			},
		}, nil, nil
	})

	return nil
}

// registerAuditTool registers the audit tool with the MCP server.
// The audit tool requires write+ access and checks actor permissions.
// Returns an error if schema generation fails.
func registerAuditTool(server *mcp.Server, execCmd execCmdFunc, actor string, validateActor bool) error {
	type auditArgs struct {
		RunIDOrURL string `json:"run_id_or_url" jsonschema:"GitHub Actions workflow run ID or URL. Accepts: numeric run ID (e.g., 1234567890), run URL (https://github.com/owner/repo/actions/runs/1234567890), job URL (https://github.com/owner/repo/actions/runs/1234567890/job/9876543210), or job URL with step (https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1)"`
	}

	// Generate schema for audit tool
	auditSchema, err := GenerateSchema[auditArgs]()
	if err != nil {
		mcpLog.Printf("Failed to generate audit tool schema: %v", err)
		return err
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "audit",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(true),
		},
		Description: `Investigate a workflow run, job, or specific step and generate a concise report.

Accepts multiple input formats:
- Numeric run ID: 1234567890
- Run URL: https://github.com/owner/repo/actions/runs/1234567890
- Job URL: https://github.com/owner/repo/actions/runs/1234567890/job/9876543210
- Job URL with step: https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1

When a job URL is provided:
- If a step number is included (#step:7:1), extracts that specific step's output
- If no step number, finds and extracts the first failing step's output
- Saves job logs and step-specific logs to the output directory

Returns JSON with the following structure:
- overview: Basic run information (run_id, workflow_name, status, conclusion, created_at, started_at, updated_at, duration, event, branch, url, logs_path)
- metrics: Execution metrics (token_usage, estimated_cost, turns, error_count, warning_count)
- jobs: List of job details (name, status, conclusion, duration)
- downloaded_files: List of artifact files (path, size, size_formatted, description, is_directory)
- missing_tools: Tools that were requested but not available (tool, reason, alternatives, timestamp, workflow_name, run_id)
- mcp_failures: MCP server failures (server_name, status, timestamp, workflow_name, run_id)
- errors: Error details (file, line, type, message)
- warnings: Warning details (file, line, type, message)
- tool_usage: Tool usage statistics (name, call_count, max_output_size, max_duration)
- firewall_analysis: Network firewall analysis if available (total_requests, allowed_requests, blocked_requests, allowed_domains, blocked_domains)`,
		InputSchema: auditSchema,
		Icons: []mcp.Icon{
			{Source: "🔍"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args auditArgs) (*mcp.CallToolResult, any, error) {
		// Check actor permissions first
		if err := checkActorPermission(ctx, actor, validateActor, "audit"); err != nil {
			return nil, nil, err
		}

		// Check for cancellation before starting
		select {
		case <-ctx.Done():
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "request cancelled",
				Data:    mcpErrorData(ctx.Err().Error()),
			}
		default:
		}

		// Build command arguments
		// Force output directory to /tmp/gh-aw/aw-mcp/logs for MCP server (same as logs)
		// Use --json flag to output structured JSON for MCP consumption
		// Pass the run ID or URL directly - the audit command will parse it
		cmdArgs := []string{"audit", args.RunIDOrURL, "-o", "/tmp/gh-aw/aw-mcp/logs", "--json"}

		// Execute the CLI command
		// Use separate stdout/stderr capture instead of CombinedOutput because:
		// - Stdout contains JSON output (--json flag)
		// - Stderr contains console messages and debug logs that shouldn't be mixed with JSON
		cmd := execCmd(ctx, cmdArgs...)
		stdout, err := cmd.Output()

		// The audit command outputs JSON to stdout when --json flag is used.
		// If the command fails, we need to provide detailed error information.
		outputStr := string(stdout)

		if err != nil {
			// Try to get stderr and exit code for detailed error reporting
			var stderr string
			var exitCode int
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				stderr = string(exitErr.Stderr)
				exitCode = exitErr.ExitCode()
			}

			mcpLog.Printf("Audit command exited with error: %v (stdout length: %d, stderr length: %d, exit_code: %d)",
				err, len(outputStr), len(stderr), exitCode)

			// Build detailed error data
			errorData := map[string]any{
				"error":         err.Error(),
				"exit_code":     exitCode,
				"stdout":        outputStr,
				"stderr":        stderr,
				"run_id_or_url": args.RunIDOrURL,
			}

			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to audit workflow run: " + err.Error(),
				Data:    mcpErrorData(errorData),
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})

	return nil
}
