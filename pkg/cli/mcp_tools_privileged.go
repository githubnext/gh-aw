package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// appendRepoFlagFromEnv appends "--repo <owner/repo>" to args when GITHUB_REPOSITORY
// is set in the environment. This allows gh CLI subcommands to identify the repository
// without falling back to git-based detection, which fails in sandboxed environments
// (e.g., MCP server containers where git is not installed).
// GITHUB_REPOSITORY is forwarded to the agentic-workflows MCP server container via
// env_vars in the MCP configuration and inherited by spawned subprocesses.
func appendRepoFlagFromEnv(args []string) []string {
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		return append(args, "--repo", repo)
	}
	return args
}

// The logs tool requires write+ access and checks actor permissions.
// Returns an error if schema generation fails.
func registerLogsTool(server *mcp.Server, execCmd execCmdFunc, actor string, validateActor bool) error {
	type logsArgs struct {
		WorkflowName      string   `json:"workflow_name,omitempty" jsonschema:"Name of the workflow to download logs for (empty for all)"`
		Count             int      `json:"count,omitempty" jsonschema:"Number of workflow runs to download (default: 100)"`
		StartDate         string   `json:"start_date,omitempty" jsonschema:"Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		EndDate           string   `json:"end_date,omitempty" jsonschema:"Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)"`
		Engine            string   `json:"engine,omitempty" jsonschema:"Filter logs by agentic engine type (claude, codex, copilot)"`
		Firewall          bool     `json:"firewall,omitempty" jsonschema:"Filter to only runs with firewall enabled"`
		NoFirewall        bool     `json:"no_firewall,omitempty" jsonschema:"Filter to only runs without firewall enabled"`
		FilteredIntegrity bool     `json:"filtered_integrity,omitempty" jsonschema:"Filter to only runs that contain DIFC integrity-filtered events in gateway logs"`
		Branch            string   `json:"branch,omitempty" jsonschema:"Filter runs by branch name"`
		AfterRunID        int64    `json:"after_run_id,omitempty" jsonschema:"Filter runs with database ID after this value (exclusive)"`
		BeforeRunID       int64    `json:"before_run_id,omitempty" jsonschema:"Filter runs with database ID before this value (exclusive)"`
		Timeout           int      `json:"timeout,omitempty" jsonschema:"Maximum time in minutes to spend downloading logs (default: 1 for MCP server)"`
		MaxTokens         int      `json:"max_tokens,omitempty" jsonschema:"Deprecated: accepted for backward compatibility but ignored. Output is always written to a file."`
		Artifacts         []string `json:"artifacts,omitempty" jsonschema:"Artifact sets to download (default: all). Valid sets: all, activation, agent, detection, firewall, github-api, mcp"`
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
	if err := AddSchemaDefault(logsSchema, "timeout", 1); err != nil {
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

In the normal case, returns a file path to a JSON file with workflow run data and metrics.
The data is written to a file to avoid large inline payloads. Use the returned file_path
to read the full data. In rare error cases (e.g., invalid workflow name), a JSON error
response is returned inline instead.

If the command times out before fetching all available logs, a "continuation" field will be present
in the JSON data with updated parameters to continue fetching more data.
Check for the presence of the continuation field to determine if there are more logs available.

The continuation field includes all necessary parameters (before_run_id, etc.) to resume fetching
from where the previous request stopped due to timeout.`,
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
			return nil, nil, newMCPError(jsonrpc.CodeInternalError, "request cancelled", ctx.Err().Error())
		default:
		}

		// Validate firewall parameters
		if args.Firewall && args.NoFirewall {
			return nil, nil, newMCPError(jsonrpc.CodeInvalidParams, "conflicting parameters: cannot specify both 'firewall' and 'no_firewall'", nil)
		}

		// Validate workflow name before executing command
		if err := validateMCPWorkflowName(args.WorkflowName); err != nil {
			mcpLog.Printf("Workflow name validation failed, returning empty result: %v", err)
			// Return an empty structured result instead of an MCP protocol error so
			// callers can always expect consistent JSON from this tool.
			// Use explicit empty slices so JSON marshaling produces "runs":[], etc.,
			// rather than null (nil slices), and set TotalDuration to match the normal
			// zero-duration formatting.
			emptyData := LogsData{
				Runs:     []RunData{},
				Episodes: []EpisodeData{},
				Edges:    []EpisodeEdge{},
				Message:  err.Error(),
			}
			emptyData.Summary.TotalDuration = "0ns"
			jsonBytes, jsonErr := json.Marshal(emptyData)
			if jsonErr != nil {
				return nil, nil, newMCPError(jsonrpc.CodeInvalidParams, err.Error(), nil)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
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
		if args.FilteredIntegrity {
			cmdArgs = append(cmdArgs, "--filtered-integrity")
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
		if len(args.Artifacts) > 0 {
			cmdArgs = append(cmdArgs, "--artifacts", strings.Join(args.Artifacts, ","))
		}

		cmdArgs = appendRepoFlagFromEnv(cmdArgs)

		// Set timeout to 1 minute for MCP server if not explicitly specified
		timeoutValue := args.Timeout
		if timeoutValue == 0 {
			timeoutValue = 1
		}
		cmdArgs = append(cmdArgs, "--timeout", strconv.Itoa(timeoutValue))

		// Always use --json mode in MCP server
		cmdArgs = append(cmdArgs, "--json")

		// Log the command being executed for debugging
		mcpLog.Printf("Executing logs tool: workflow=%s, count=%d, firewall=%v, no_firewall=%v, filtered_integrity=%v, timeout=%d, command_args=%v",
			args.WorkflowName, args.Count, args.Firewall, args.NoFirewall, args.FilteredIntegrity, timeoutValue, cmdArgs)

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

			// Extract the user-facing message from stderr, filtering out debug log lines
			// (e.g. "workflow:script_registry Creating new script registry +151ns")
			// to avoid leaking internal diagnostic output in the MCP error response.
			mainMsg := extractLastConsoleMessage(stderr)
			if mainMsg == "" {
				mainMsg = err.Error()
			}
			return nil, nil, newMCPError(jsonrpc.CodeInternalError, "failed to download workflow logs: "+mainMsg, errorData)
		}

		// Always write output to a file and return schema + file path
		finalOutput := buildLogsFileResponse(outputStr)

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
		RunIDOrURL   string   `json:"run_id_or_url,omitempty"   jsonschema:"Deprecated: use run_ids_or_urls instead. Single GitHub Actions workflow run ID or URL."`
		RunIDsOrURLs []string `json:"run_ids_or_urls,omitempty" jsonschema:"One or more workflow run IDs or URLs. Single item: detailed audit report. Multiple items: diff mode with first as base (see tool description for accepted formats)."`
		Artifacts    []string `json:"artifacts,omitempty"        jsonschema:"Artifact sets to download (default: all). Valid sets: all, activation, agent, detection, firewall, github-api, mcp"`
		MaxTokens    int      `json:"max_tokens,omitempty"       jsonschema:"Deprecated: accepted for backward compatibility but ignored."`
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
		Description: `Investigate one or more workflow runs and generate a concise report.

When a single run is provided, generates a detailed audit report.
When two or more runs are provided, the first is the base (reference) run and
the remaining runs are compared against it (diff mode), showing changes in
firewall domains, MCP tool usage, and run metrics.

Each run accepts:
- Numeric run ID: 1234567890
- Run URL: https://github.com/owner/repo/actions/runs/1234567890
- Job URL: https://github.com/owner/repo/actions/runs/1234567890/job/9876543210
- Job URL with step: https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1

When a job URL is provided (single-run mode only):
- If a step number is included (#step:7:1), extracts that specific step's output
- If no step number, finds and extracts the first failing step's output
- Saves job logs and step-specific logs to the output directory

Single-run returns JSON with:
- overview: Basic run information (run_id, workflow_name, status, conclusion, created_at, started_at, updated_at, duration, event, branch, url, logs_path)
- metrics: Execution metrics (token_usage, estimated_cost, turns, error_count, warning_count)
- jobs: List of job details (name, status, conclusion, duration)
- downloaded_files: List of artifact files (path, size, size_formatted, description, is_directory)
- missing_tools: Tools that were requested but not available (tool, reason, alternatives, timestamp, workflow_name, run_id)
- mcp_failures: MCP server failures (server_name, status, timestamp, workflow_name, run_id)
- errors: Error details (file, line, type, message)
- warnings: Warning details (file, line, type, message)
- tool_usage: Tool usage statistics (name, call_count, max_output_size, max_duration)
- firewall_analysis: Network firewall analysis if available (total_requests, allowed_requests, blocked_requests, allowed_domains, blocked_domains)

Multi-run diff returns JSON describing changes between the base and each comparison run.`,
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
			return nil, nil, newMCPError(jsonrpc.CodeInternalError, "request cancelled", ctx.Err().Error())
		default:
		}

		// Resolve the list of run IDs/URLs to pass to the audit command.
		// run_ids_or_urls takes precedence; fall back to the deprecated run_id_or_url field.
		runItems := args.RunIDsOrURLs
		if len(runItems) == 0 && args.RunIDOrURL != "" {
			runItems = []string{args.RunIDOrURL}
		}
		if len(runItems) == 0 {
			return nil, nil, newMCPError(jsonrpc.CodeInvalidParams, "at least one run ID or URL must be provided via run_ids_or_urls or run_id_or_url", nil)
		}

		// Build command arguments.
		// Force output directory to /tmp/gh-aw/aw-mcp/logs for MCP server (same as logs).
		// Use --json flag to output structured JSON for MCP consumption.
		// Pass all run IDs/URLs directly - the audit command handles single vs. diff mode.
		cmdArgs := []string{"audit"}
		cmdArgs = append(cmdArgs, runItems...)
		cmdArgs = append(cmdArgs, "-o", "/tmp/gh-aw/aw-mcp/logs", "--json")
		if len(args.Artifacts) > 0 {
			cmdArgs = append(cmdArgs, "--artifacts", strings.Join(args.Artifacts, ","))
		}

		cmdArgs = appendRepoFlagFromEnv(cmdArgs)

		// Execute the CLI command.
		// Use separate stdout/stderr capture instead of CombinedOutput because:
		// - Stdout contains JSON output (--json flag)
		// - Stderr contains console messages and debug logs that shouldn't be mixed with JSON
		cmd := execCmd(ctx, cmdArgs...)
		stdout, err := cmd.Output()

		// The audit command outputs JSON to stdout when --json flag is used.
		// If the command fails, we need to provide detailed error information.
		outputStr := string(stdout)

		if err != nil {
			// Try to get stderr for message extraction
			var stderr string
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				stderr = string(exitErr.Stderr)
			}

			mcpLog.Printf("Audit command exited with error: %v (stdout length: %d, stderr length: %d)",
				err, len(outputStr), len(stderr))

			// Extract the user-facing message from stderr, filtering out debug log lines
			// (e.g. "workflow:script_registry Creating new script registry +151ns")
			// to avoid leaking internal diagnostic output in the MCP error response.
			mainMsg := extractLastConsoleMessage(stderr)
			if mainMsg == "" {
				mainMsg = err.Error()
			}

			// Return a JSON error envelope instead of an MCP protocol error so
			// callers always receive consistent JSON and the run IDs are always present.
			// IsError must be false so that callers (e.g. mcp_cli_bridge) treat this as
			// a graceful not-found / failure response rather than a fatal protocol error.
			errorMsg := "failed to audit workflow run: " + mainMsg
			if len(runItems) > 1 {
				errorMsg = "failed to audit workflow runs: " + mainMsg
			}
			errorEnvelope := map[string]any{
				"error":           errorMsg,
				"run_ids_or_urls": runItems,
				"suggestions": []string{
					"Verify the run ID is correct",
					"Use the 'logs' tool to list recent run IDs",
				},
			}
			jsonBytes, jsonErr := json.Marshal(errorEnvelope)
			if jsonErr != nil {
				return nil, nil, newMCPError(jsonrpc.CodeInternalError, errorMsg, nil)
			}
			return &mcp.CallToolResult{
				IsError: false,
				Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})

	return nil
}

// registerAuditDiffTool registers the audit-diff tool with the MCP server.
// It exposes the `gh aw audit diff` subcommand for comparing two workflow runs.
func registerAuditDiffTool(server *mcp.Server, execCmd execCmdFunc, actor string, validateActor bool) error {
	type auditDiffArgs struct {
		BaseRunID     string   `json:"base_run_id"     jsonschema:"Numeric ID of the base (reference) workflow run"`
		CompareRunIDs []string `json:"compare_run_ids" jsonschema:"One or more numeric IDs of the comparison runs"`
		Artifacts     []string `json:"artifacts,omitempty" jsonschema:"Artifact sets to download (default: all). Valid sets: all, activation, agent, detection, firewall, github-api, mcp"`
	}

	schema, err := GenerateSchema[auditDiffArgs]()
	if err != nil {
		mcpLog.Printf("Failed to generate audit-diff tool schema: %v", err)
		return err
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "audit-diff",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(true),
		},
		Description: `Compare behavior between a base workflow run and one or more comparison runs.

Downloads artifacts for all referenced runs (using locally cached data when available),
then produces a diff showing:
- New or removed domains in firewall logs
- Domain allow/deny status changes
- Anomaly flags (new denied domains, previously-denied now allowed)
- MCP tool invocation changes (new/removed tools, call/error count diffs)
- Run metrics comparison (token usage, duration, turns)

Returns JSON describing the differences between the base run and each comparison run.`,
		InputSchema: schema,
		Icons: []mcp.Icon{
			{Source: "🔍"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args auditDiffArgs) (*mcp.CallToolResult, any, error) {
		if err := checkActorPermission(ctx, actor, validateActor, "audit-diff"); err != nil {
			return nil, nil, err
		}

		select {
		case <-ctx.Done():
			return nil, nil, newMCPError(jsonrpc.CodeInternalError, "request cancelled", ctx.Err().Error())
		default:
		}

		if args.BaseRunID == "" {
			return nil, nil, newMCPError(jsonrpc.CodeInvalidParams, "base_run_id is required", nil)
		}
		if len(args.CompareRunIDs) == 0 {
			return nil, nil, newMCPError(jsonrpc.CodeInvalidParams, "compare_run_ids must contain at least one run ID", nil)
		}

		// Build: gh aw audit diff <base> <compare...> -o ... --json [--artifacts ...]
		cmdArgs := []string{"audit", "diff", args.BaseRunID}
		cmdArgs = append(cmdArgs, args.CompareRunIDs...)
		cmdArgs = append(cmdArgs, "-o", "/tmp/gh-aw/aw-mcp/logs", "--json")
		if len(args.Artifacts) > 0 {
			cmdArgs = append(cmdArgs, "--artifacts", strings.Join(args.Artifacts, ","))
		}

		cmd := execCmd(ctx, cmdArgs...)
		stdout, err := cmd.Output()
		outputStr := string(stdout)

		if err != nil {
			var stderr string
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				stderr = string(exitErr.Stderr)
			}
			mcpLog.Printf("Audit-diff command failed: %v (stdout: %d bytes, stderr: %d bytes)", err, len(outputStr), len(stderr))
			mainMsg := extractLastConsoleMessage(stderr)
			if mainMsg == "" {
				mainMsg = err.Error()
			}
			errorEnvelope := map[string]any{
				"error":        "failed to diff workflow runs: " + mainMsg,
				"base_run_id":  args.BaseRunID,
				"compare_runs": args.CompareRunIDs,
				"suggestions": []string{
					"Verify the run IDs are correct",
					"Use the 'logs' tool to list recent run IDs",
				},
			}
			jsonBytes, jsonErr := json.Marshal(errorEnvelope)
			if jsonErr != nil {
				return nil, nil, newMCPError(jsonrpc.CodeInternalError, "failed to diff workflow runs: "+mainMsg, nil)
			}
			return &mcp.CallToolResult{
				IsError: false,
				Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: outputStr}},
		}, nil, nil
	})

	return nil
}

// filtering out debug log lines (e.g. "workflow:script_registry Creating... +151ns").
// Console messages are identified by their prefix symbols (✗, ✓, ℹ, ⚠, etc.).
// Falls back to the last non-empty line if no console message is found.
func extractLastConsoleMessage(stderr string) string {
	// Console message prefixes used by the console package
	consoleSymbols := []string{"✗ ", "✓ ", "ℹ ", "⚠ ", "⚡ ", "🔨 ", "❓ ", "🔍 "}

	var lastConsole string
	var lastLine string

	for line := range strings.SplitSeq(stderr, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lastLine = trimmed
		for _, sym := range consoleSymbols {
			if strings.HasPrefix(trimmed, sym) {
				lastConsole = trimmed
				break
			}
		}
	}

	if lastConsole != "" {
		return lastConsole
	}
	return lastLine
}
