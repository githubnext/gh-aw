package cli

import (
	"context"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerAddTool registers the add tool with the MCP server.
func registerAddTool(server *mcp.Server, execCmd execCmdFunc) {
	type addArgs struct {
		Workflows []string `json:"workflows" jsonschema:"Workflows to add (e.g., 'owner/repo/workflow-name' or 'owner/repo/workflow-name@version')"`
		Number    int      `json:"number,omitempty" jsonschema:"Create multiple numbered copies (corresponds to -c flag, default: 1)"`
		Name      string   `json:"name,omitempty" jsonschema:"Specify name for the added workflow - without .md extension (corresponds to -n flag)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "add",
		Annotations: &mcp.ToolAnnotations{
			OpenWorldHint: boolPtr(true),
		},
		Description: "Add workflows from remote repositories to .github/workflows",
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

		// Validate required arguments
		if len(args.Workflows) == 0 {
			return nil, nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: "missing required parameter: at least one workflow specification is required",
				Data:    nil,
			}
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

// registerUpdateTool registers the update tool with the MCP server.
func registerUpdateTool(server *mcp.Server, execCmd execCmdFunc) {
	type updateArgs struct {
		Workflows []string `json:"workflows,omitempty" jsonschema:"Workflow IDs to update (empty for all workflows)"`
		Major     bool     `json:"major,omitempty" jsonschema:"Allow major version updates when updating tagged releases"`
		Force     bool     `json:"force,omitempty" jsonschema:"Force update even if no changes detected"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "update",
		Annotations: &mcp.ToolAnnotations{
			OpenWorldHint: boolPtr(true),
		},
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

// registerFixTool registers the fix tool with the MCP server.
func registerFixTool(server *mcp.Server, execCmd execCmdFunc) {
	type fixArgs struct {
		Workflows    []string `json:"workflows,omitempty" jsonschema:"Workflow IDs to fix (empty for all workflows)"`
		Write        bool     `json:"write,omitempty" jsonschema:"Write changes to files (default is dry-run)"`
		ListCodemods bool     `json:"list_codemods,omitempty" jsonschema:"List all available codemods and exit"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "fix",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint:  true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
		Description: `Apply automatic codemod-style fixes to agentic workflow files.

This command applies a registry of codemods that automatically update deprecated fields
and migrate to new syntax. Codemods preserve formatting and comments as much as possible.

Available codemods:
• timeout-minutes-migration: Replaces 'timeout_minutes' with 'timeout-minutes'
• network-firewall-migration: Removes deprecated 'network.firewall' field
• sandbox-agent-false-removal: Removes 'sandbox.agent: false' (firewall now mandatory)
• safe-inputs-mode-removal: Removes deprecated 'safe-inputs.mode' field

If no workflows are specified, all Markdown files in .github/workflows will be processed.

The command will:
1. Scan workflow files for deprecated fields
2. Apply relevant codemods to fix issues
3. Report what was changed in each file
4. Write updated files back to disk (with write flag)

Returns formatted text output showing:
- List of workflow files processed
- Which codemods were applied to each file
- Summary of fixes applied`,
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

		// Add workflow IDs if specified
		cmdArgs = append(cmdArgs, args.Workflows...)

		// Add optional flags
		if args.Write {
			cmdArgs = append(cmdArgs, "--write")
		}
		if args.ListCodemods {
			cmdArgs = append(cmdArgs, "--list-codemods")
		}

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
