package cli

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var mcpToolsLog = logger.New("mcp:server_tools")

// execCmdFunc is a function type for executing commands
// This allows us to inject different command execution strategies for testing
type execCmdFunc func(ctx context.Context, args ...string) *exec.Cmd

// addStatusTool registers the status tool with the MCP server
func addStatusTool(server *mcp.Server, execCmd execCmdFunc) {
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

		// Build command arguments - always use JSON for MCP
		cmdArgs := []string{"status", "--json"}
		if args.Pattern != "" {
			cmdArgs = append(cmdArgs, args.Pattern)
		}

		mcpToolsLog.Printf("Executing status tool: pattern=%s, jqFilter=%s", args.Pattern, args.JqFilter)
		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to execute status command",
				Data:    mcpErrorData(map[string]any{"error": err.Error(), "output": string(output)}),
			}
		}

		// Apply jq filter if provided
		outputStr := string(output)
		if args.JqFilter != "" {
			filteredOutput, jqErr := ApplyJqFilter(outputStr, args.JqFilter)
			if jqErr != nil {
				return nil, nil, &jsonrpc.Error{
					Code:    jsonrpc.CodeInvalidParams,
					Message: "invalid jq filter expression",
					Data:    mcpErrorData(map[string]any{"error": jqErr.Error(), "filter": args.JqFilter}),
				}
			}
			outputStr = filteredOutput
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})
}

// addCompileTool registers the compile tool with the MCP server
func addCompileTool(server *mcp.Server, execCmd execCmdFunc) {
	type compileArgs struct {
		Workflows  []string `json:"workflows,omitempty" jsonschema:"Workflow files to compile (empty for all)"`
		Strict     bool     `json:"strict,omitempty" jsonschema:"Override frontmatter to enforce strict mode validation for all workflows. Note: Workflows default to strict mode unless frontmatter sets strict: false"`
		Zizmor     bool     `json:"zizmor,omitempty" jsonschema:"Run zizmor security scanner on generated .lock.yml files"`
		Poutine    bool     `json:"poutine,omitempty" jsonschema:"Run poutine security scanner on generated .lock.yml files"`
		Actionlint bool     `json:"actionlint,omitempty" jsonschema:"Run actionlint linter on generated .lock.yml files"`
		Fix        bool     `json:"fix,omitempty" jsonschema:"Apply automatic codemod fixes to workflows before compiling"`
		JqFilter   string   `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}

	// Generate schema with elicitation defaults
	compileSchema, err := GenerateOutputSchema[compileArgs]()
	if err != nil {
		mcpToolsLog.Printf("Failed to generate compile tool schema: %v", err)
		return
	}
	// Add elicitation default: strict defaults to true (most common case)
	if err := AddSchemaDefault(compileSchema, "strict", true); err != nil {
		mcpToolsLog.Printf("Failed to add default for strict: %v", err)
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "compile",
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
- compiled_file: Path to the generated .lock.yml file

Static analysis tools (zizmor, poutine, actionlint) can be enabled to check security and best practices.
These tools analyze the generated .lock.yml files and report any issues found.

Example: compile({"workflows": ["my-workflow.md"], "strict": true, "zizmor": true})`,
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

		// Build command arguments - always use JSON for MCP
		cmdArgs := []string{"compile", "--json"}

		// Add individual workflows if specified
		for _, w := range args.Workflows {
			cmdArgs = append(cmdArgs, w)
		}

		// Add flags based on arguments
		if args.Strict {
			cmdArgs = append(cmdArgs, "--strict")
		}
		if args.Zizmor {
			cmdArgs = append(cmdArgs, "--zizmor")
		}
		if args.Poutine {
			cmdArgs = append(cmdArgs, "--poutine")
		}
		if args.Actionlint {
			cmdArgs = append(cmdArgs, "--actionlint")
		}
		if args.Fix {
			cmdArgs = append(cmdArgs, "--fix")
		}

		mcpToolsLog.Printf("Executing compile tool: workflows=%v, strict=%v", args.Workflows, args.Strict)

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			// Compile command may return error exit code but still produce valid JSON output
			// containing error details, so we don't immediately fail here
			mcpToolsLog.Printf("Compile command returned error: %v", err)
		}

		// Apply jq filter if provided
		outputStr := string(output)
		if args.JqFilter != "" {
			filteredOutput, jqErr := ApplyJqFilter(outputStr, args.JqFilter)
			if jqErr != nil {
				return nil, nil, &jsonrpc.Error{
					Code:    jsonrpc.CodeInvalidParams,
					Message: "invalid jq filter expression",
					Data:    mcpErrorData(map[string]any{"error": jqErr.Error(), "filter": args.JqFilter}),
				}
			}
			outputStr = filteredOutput
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})
}

// addLogsTool registers the logs tool with the MCP server
func addLogsTool(server *mcp.Server, execCmd execCmdFunc) {
	type logsArgs struct {
		WorkflowID string `json:"workflow_id" jsonschema:"Required: workflow file name (e.g., 'my-workflow.md') or numeric run ID"`
		JqFilter   string `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "logs",
		Description: `Download and analyze logs from a workflow run.

This tool downloads logs from GitHub Actions for a specific workflow run and generates
structured JSON output with metrics, analysis, and detailed job information.

The workflow_id parameter can be:
- A workflow filename (e.g., "my-workflow.md") to download logs from the most recent run
- A numeric run ID (e.g., "1234567890") to download logs from a specific run

Returns JSON object with:
- run_id: The GitHub Actions run ID
- run_url: URL to view the run on GitHub
- workflow_name: Name of the workflow
- status: Status of the run (success, failure, cancelled, etc.)
- conclusion: Final conclusion of the run
- jobs: Array of job objects with detailed logs and metrics
- metrics: Overall metrics for the run (duration, log sizes, etc.)
- analysis: Analysis of errors, warnings, and patterns in logs

Example: logs({"workflow_id": "my-workflow.md"})`,
		Icons: []mcp.Icon{
			{Source: "📜"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, any, error) {
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

		if args.WorkflowID == "" {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: "workflow_id is required",
				Data:    mcpErrorData("workflow_id must be specified"),
			}
		}

		// Build command arguments - always use JSON for MCP
		cmdArgs := []string{"logs", args.WorkflowID, "--json"}

		mcpToolsLog.Printf("Executing logs tool: workflow_id=%s", args.WorkflowID)

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to download logs",
				Data:    mcpErrorData(map[string]any{"error": err.Error(), "output": string(output)}),
			}
		}

		// Apply jq filter if provided
		outputStr := string(output)
		if args.JqFilter != "" {
			filteredOutput, jqErr := ApplyJqFilter(outputStr, args.JqFilter)
			if jqErr != nil {
				return nil, nil, &jsonrpc.Error{
					Code:    jsonrpc.CodeInvalidParams,
					Message: "invalid jq filter expression",
					Data:    mcpErrorData(map[string]any{"error": jqErr.Error(), "filter": args.JqFilter}),
				}
			}
			outputStr = filteredOutput
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})
}

// addAuditTool registers the audit tool with the MCP server
func addAuditTool(server *mcp.Server, execCmd execCmdFunc) {
	type auditArgs struct {
		RunID    string `json:"run_id" jsonschema:"Required: workflow run ID to audit"`
		JqFilter string `json:"jq,omitempty" jsonschema:"Optional jq filter to apply to JSON output"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "audit",
		Description: `Investigate a workflow run and generate a detailed audit report.

This tool performs a comprehensive audit of a workflow run, analyzing:
- Workflow configuration and setup
- Job execution and dependencies
- Step timing and resource usage
- Errors, warnings, and failures
- Security and best practices compliance
- Performance metrics and bottlenecks

Returns JSON object with detailed audit information including:
- run_metadata: Basic information about the run
- workflow_config: Configuration and frontmatter
- jobs: Detailed analysis of each job
- timing_analysis: Performance breakdown
- error_analysis: Root cause analysis of failures
- recommendations: Suggested improvements

Example: audit({"run_id": "1234567890"})`,
		Icons: []mcp.Icon{
			{Source: "🔍"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args auditArgs) (*mcp.CallToolResult, any, error) {
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

		if args.RunID == "" {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: "run_id is required",
				Data:    mcpErrorData("run_id must be specified"),
			}
		}

		// Build command arguments - always use JSON for MCP
		cmdArgs := []string{"audit", args.RunID, "--json"}

		mcpToolsLog.Printf("Executing audit tool: run_id=%s", args.RunID)

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to audit run",
				Data:    mcpErrorData(map[string]any{"error": err.Error(), "output": string(output)}),
			}
		}

		// Apply jq filter if provided
		outputStr := string(output)
		if args.JqFilter != "" {
			filteredOutput, jqErr := ApplyJqFilter(outputStr, args.JqFilter)
			if jqErr != nil {
				return nil, nil, &jsonrpc.Error{
					Code:    jsonrpc.CodeInvalidParams,
					Message: "invalid jq filter expression",
					Data:    mcpErrorData(map[string]any{"error": jqErr.Error(), "filter": args.JqFilter}),
				}
			}
			outputStr = filteredOutput
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputStr},
			},
		}, nil, nil
	})
}

// addMCPInspectTool registers the mcp-inspect tool with the MCP server
func addMCPInspectTool(server *mcp.Server, execCmd execCmdFunc) {
	type mcpInspectArgs struct {
		WorkflowID string          `json:"workflow_id,omitempty" jsonschema:"Workflow file name to inspect MCP servers for (e.g., 'my-workflow.md')"`
		ListAll    bool            `json:"list_all,omitempty" jsonschema:"List all workflows that use MCP servers"`
		Format     string          `json:"format,omitempty" jsonschema:"Output format: 'json' (default) or 'text'"`
		Raw        json.RawMessage `json:"raw,omitempty" jsonschema:"Additional raw arguments (advanced)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "mcp-inspect",
		Description: `Inspect MCP servers configured in workflows and list available tools.

This tool examines workflows to identify MCP servers and their configurations.
It can show which tools are available from each server and how they're configured.

Usage modes:
1. Inspect a specific workflow: mcp_inspect({"workflow_id": "my-workflow.md"})
2. List all workflows with MCP servers: mcp_inspect({"list_all": true})

Returns JSON array with MCP server information including:
- workflow: Name of the workflow file
- servers: Array of MCP server objects
- Each server object contains: name, version, tools, configuration

Example: mcp_inspect({"workflow_id": "my-workflow.md", "format": "json"})`,
		Icons: []mcp.Icon{
			{Source: "🔌"},
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

		// Default to JSON format for MCP
		if args.Format == "" || args.Format == "json" {
			cmdArgs = append(cmdArgs, "--json")
		}

		if args.ListAll {
			// List all workflows with MCP servers
			cmdArgs = []string{"mcp", "list", "--json"}
		} else if args.WorkflowID != "" {
			cmdArgs = append(cmdArgs, args.WorkflowID)
		} else {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: "either workflow_id or list_all must be specified",
				Data:    mcpErrorData("specify workflow_id or set list_all to true"),
			}
		}

		mcpToolsLog.Printf("Executing mcp-inspect tool: workflow_id=%s, list_all=%v", args.WorkflowID, args.ListAll)

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

// addAddTool registers the add tool with the MCP server
func addAddTool(server *mcp.Server, execCmd execCmdFunc) {
	type addArgs struct {
		WorkflowSpecs []string `json:"workflow_specs" jsonschema:"Required: workflow specifications to add (e.g., ['org/repo/workflow-name'])"`
		Engine        string   `json:"engine,omitempty" jsonschema:"Override AI engine for workflows (e.g., 'copilot', 'claude')"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "add",
		Description: `Add workflows from remote repositories to .github/workflows.

This tool downloads workflow files from other repositories and adds them to the current
repository's .github/workflows directory. It handles compilation and setup automatically.

Workflow specifications should be in the format: owner/repo/workflow-name
- Example: "githubnext/agentics/weekly-research"

The tool will:
1. Download the workflow file from the specified repository
2. Compile it to generate .lock.yml file
3. Add necessary configuration files (if needed)
4. Report success or any errors encountered

Returns text output with:
- Status of each workflow added
- Compilation results
- Any errors or warnings

Example: add({"workflow_specs": ["githubnext/agentics/daily-standup"]})`,
		Icons: []mcp.Icon{
			{Source: "➕"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args addArgs) (*mcp.CallToolResult, any, error) {
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

		if len(args.WorkflowSpecs) == 0 {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: "workflow_specs is required",
				Data:    mcpErrorData("at least one workflow specification must be provided"),
			}
		}

		// Build command arguments
		cmdArgs := []string{"add"}
		cmdArgs = append(cmdArgs, args.WorkflowSpecs...)

		if args.Engine != "" {
			cmdArgs = append(cmdArgs, "--engine", args.Engine)
		}

		mcpToolsLog.Printf("Executing add tool: workflows=%v, engine=%s", args.WorkflowSpecs, args.Engine)

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to add workflows",
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

// addUpdateTool registers the update tool with the MCP server
func addUpdateTool(server *mcp.Server, execCmd execCmdFunc) {
	type updateArgs struct {
		Workflows []string `json:"workflows,omitempty" jsonschema:"Workflow files to update (empty for all)"`
		Force     bool     `json:"force,omitempty" jsonschema:"Force update even if workflows have uncommitted changes"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "update",
		Description: `Update workflows from their source repositories.

This tool checks for updates to workflows that were previously added from remote repositories.
It downloads the latest version from the source and updates the local copy.

The tool will:
1. Identify workflows that have remote sources
2. Check if updates are available
3. Download and apply updates
4. Recompile workflows
5. Report what was updated

Returns text output with:
- List of workflows checked
- Update status for each workflow
- Compilation results after updates
- Any errors or warnings

Example: update({"workflows": ["my-workflow.md"]})
Example: update({}) to update all workflows`,
		Icons: []mcp.Icon{
			{Source: "🔄"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args updateArgs) (*mcp.CallToolResult, any, error) {
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
		cmdArgs := []string{"update"}

		for _, w := range args.Workflows {
			cmdArgs = append(cmdArgs, w)
		}

		if args.Force {
			cmdArgs = append(cmdArgs, "--force")
		}

		mcpToolsLog.Printf("Executing update tool: workflows=%v, force=%v", args.Workflows, args.Force)

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to update workflows",
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

// addFixTool registers the fix tool with the MCP server
func addFixTool(server *mcp.Server, execCmd execCmdFunc) {
	type fixArgs struct {
		Workflows []string `json:"workflows,omitempty" jsonschema:"Workflow files to fix (empty for all)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "fix",
		Description: `Apply automatic codemod-style fixes to workflow files.

This tool automatically fixes common issues in workflow markdown files:
- Updates deprecated syntax to current format
- Fixes formatting issues
- Applies best practices
- Corrects common mistakes

The tool will:
1. Analyze workflow files for fixable issues
2. Apply automatic fixes
3. Report what was fixed
4. Preserve manual formatting where appropriate

Returns text output with:
- List of workflows checked
- Fixes applied to each workflow
- Any issues that require manual intervention
- Compilation status after fixes

Example: fix({"workflows": ["my-workflow.md"]})
Example: fix({}) to fix all workflows`,
		Icons: []mcp.Icon{
			{Source: "🔧"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args fixArgs) (*mcp.CallToolResult, any, error) {
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
		cmdArgs := []string{"fix"}

		for _, w := range args.Workflows {
			cmdArgs = append(cmdArgs, w)
		}

		mcpToolsLog.Printf("Executing fix tool: workflows=%v", args.Workflows)

		// Execute the CLI command
		cmd := execCmd(ctx, cmdArgs...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInternalError,
				Message: "failed to fix workflows",
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
