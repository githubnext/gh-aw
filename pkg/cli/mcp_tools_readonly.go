package cli

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerStatusTool registers the status tool with the MCP server.
// The status tool is read-only and idempotent.
func registerStatusTool(server *mcp.Server) {
	type statusArgs struct {
		Pattern string `json:"pattern,omitempty" jsonschema:"Optional pattern to filter workflows by name"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "status",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(false),
		},
		Description: `Show status of agentic workflow files and workflows.

Returns a JSON array where each element has the following structure:
- workflow: Name of the workflow file
- agent: AI engine used (e.g., "copilot", "claude", "codex")
- compiled: Whether the workflow is compiled ("Yes", "No", or "N/A")
- status: GitHub workflow status ("active", "disabled", "Unknown")
- time_remaining: Time remaining until workflow deadline (if applicable)`,
		Icons: []mcp.Icon{
			{Source: "📊"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, any, error) {
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

		mcpLog.Printf("Executing status tool: pattern=%s", args.Pattern)

		// Call GetWorkflowStatuses directly instead of spawning subprocess
		statuses, err := GetWorkflowStatuses(args.Pattern, "", "", "")
		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to get workflow statuses",
				Data:    mcpErrorData(map[string]any{"error": err.Error()}),
			}
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(statuses)
		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to marshal workflow statuses",
				Data:    mcpErrorData(map[string]any{"error": err.Error()}),
			}
		}

		outputStr := string(jsonBytes)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})
}

// registerCompileTool registers the compile tool with the MCP server.
// Returns an error if schema generation fails, which causes the server to stop registering tools.
func registerCompileTool(server *mcp.Server, execCmd execCmdFunc) error {
	type compileArgs struct {
		Workflows  []string `json:"workflows,omitempty" jsonschema:"Workflow files to compile (empty for all)"`
		Strict     bool     `json:"strict,omitempty" jsonschema:"Override frontmatter to enforce strict mode validation for all workflows. Note: Workflows default to strict mode unless frontmatter sets strict: false"`
		Zizmor     bool     `json:"zizmor,omitempty" jsonschema:"Run zizmor security scanner on generated .lock.yml files"`
		Poutine    bool     `json:"poutine,omitempty" jsonschema:"Run poutine security scanner on generated .lock.yml files"`
		Actionlint bool     `json:"actionlint,omitempty" jsonschema:"Run actionlint linter on generated .lock.yml files"`
		Fix        bool     `json:"fix,omitempty" jsonschema:"Apply automatic codemod fixes to workflows before compiling"`
	}

	// Generate schema with elicitation defaults
	compileSchema, err := GenerateSchema[compileArgs]()
	if err != nil {
		mcpLog.Printf("Failed to generate compile tool schema: %v", err)
		return err
	}
	// Add elicitation default: strict defaults to true (most common case)
	if err := AddSchemaDefault(compileSchema, "strict", true); err != nil {
		mcpLog.Printf("Failed to add default for strict: %v", err)
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "compile",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint:  true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
		Description: `Compile Markdown workflows to GitHub Actions YAML with optional static analysis tools.

⚠️  IMPORTANT: Any change to .github/workflows/*.md files MUST be compiled using this tool.
This tool generates .lock.yml files from .md workflow files. The .lock.yml files are what GitHub Actions
actually executes, so failing to compile after modifying a .md file means your changes won't take effect.

Workflows use strict mode validation by default (unless frontmatter sets strict: false).
Strict mode enforces: action pinning to SHAs, explicit network config, safe-outputs for write operations,
and refuses write permissions and deprecated fields. Use the strict parameter to override frontmatter settings.

Returns JSON array with validation results for each workflow:
- workflow: Name of the workflow file
- valid: Boolean indicating if compilation was successful
- errors: Array of error objects with type, message, and optional line number
- warnings: Array of warning objects
- compiled_file: Path to the generated .lock.yml file`,
		InputSchema: compileSchema,
		Icons: []mcp.Icon{
			{Source: "🔨"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args compileArgs) (*mcp.CallToolResult, any, error) {
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

		// Check if any static analysis tools are requested that require Docker images
		if args.Zizmor || args.Poutine || args.Actionlint {
			// Check if Docker images are available; if not, start downloading and return retry message
			if err := CheckAndPrepareDockerImages(ctx, args.Zizmor, args.Poutine, args.Actionlint); err != nil {
				return nil, nil, &jsonrpc.Error{
					Code:    jsonrpc.CodeInternalError,
					Message: "docker images not ready",
					Data:    mcpErrorData(err.Error()),
				}
			}

			// Check for cancellation after Docker image preparation
			select {
			case <-ctx.Done():
				return nil, nil, &jsonrpc.Error{
					Code:    jsonrpc.CodeInternalError,
					Message: "request cancelled",
					Data:    mcpErrorData(ctx.Err().Error()),
				}
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

		mcpLog.Printf("Executing compile tool: workflows=%v, strict=%v, fix=%v, zizmor=%v, poutine=%v, actionlint=%v",
			args.Workflows, args.Strict, args.Fix, args.Zizmor, args.Poutine, args.Actionlint)

		// Execute the CLI command
		// Use separate stdout/stderr capture instead of CombinedOutput because:
		// - Stdout contains JSON output (--json flag)
		// - Stderr contains console messages that shouldn't be mixed with JSON
		cmd := execCmd(ctx, cmdArgs...)
		stdout, err := cmd.Output()

		// The compile command always outputs JSON to stdout when --json flag is used, even on error.
		// We should return the JSON output to the LLM so it can see validation errors.
		// Only return an MCP error if we cannot get any output at all.
		outputStr := string(stdout)

		// If the command failed but we have output, it's likely compilation errors
		// which are included in the JSON output. Return the output, not an MCP error.
		if err != nil {
			mcpLog.Printf("Compile command exited with error: %v (output length: %d)", err, len(outputStr))
			// If we have no output, this is a real execution failure
			if len(outputStr) == 0 {
				// Try to get stderr for error details
				var stderr string
				if exitErr, ok := err.(*exec.ExitError); ok {
					stderr = string(exitErr.Stderr)
				}
				return nil, nil, &jsonrpc.Error{
					Code:    jsonrpc.CodeInternalError,
					Message: "failed to compile workflows",
					Data:    mcpErrorData(map[string]any{"error": err.Error(), "stderr": stderr}),
				}
			}
			// Otherwise, we have output (likely validation errors in JSON), so continue
			// and return it to the LLM
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})

	return nil
}

// registerMCPInspectTool registers the mcp-inspect tool with the MCP server.
func registerMCPInspectTool(server *mcp.Server, execCmd execCmdFunc) {
	type mcpInspectArgs struct {
		WorkflowFile string `json:"workflow_file,omitempty" jsonschema:"Workflow file to inspect MCP servers from (empty to list all workflows with MCP servers)"`
		Server       string `json:"server,omitempty" jsonschema:"Filter to inspect only the specified MCP server"`
		Tool         string `json:"tool,omitempty" jsonschema:"Show detailed information about a specific tool (requires server parameter)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "mcp-inspect",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(true),
		},
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
		Icons: []mcp.Icon{
			{Source: "🔎"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpInspectArgs) (*mcp.CallToolResult, any, error) {
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
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to inspect MCP servers",
				Data:    mcpErrorData(map[string]any{"error": err.Error(), "output": string(output)}),
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(output)},
			},
		}, nil, nil
	})
}
