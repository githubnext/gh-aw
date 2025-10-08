package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// NewMCPServerCommand creates the mcp-server command
func NewMCPServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp-server",
		Short: "Run an MCP (Model Context Protocol) server exposing gh-aw commands as tools",
		Long: `Run an MCP server that exposes gh-aw CLI commands as MCP tools.

This command starts a stdio-based MCP server that wraps the gh-aw CLI,
spawning subprocess calls for each tool invocation. This design ensures
that GitHub tokens and other secrets are not shared with the MCP server
process itself.

The server provides the following tools:
  - status   - Show status of agentic workflow files
  - compile  - Compile markdown workflow files to YAML
  - logs     - Download and analyze workflow logs
  - audit    - Investigate a workflow run and generate a report

The server uses stdio transport for communication and is designed to be used
by AI agents or other MCP clients.

Example:
  gh aw mcp-server`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runMCPServer(); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	return cmd
}

// runMCPServer starts the MCP server on stdio transport
func runMCPServer() error {
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
		Engine    string   `json:"engine,omitempty" jsonschema:"Override AI engine (claude, codex, copilot)"`
		Validate  bool     `json:"validate" jsonschema:"Enable GitHub Actions workflow schema validation"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "compile",
		Description: "Compile markdown workflow files to YAML workflows",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args compileArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		cmdArgs := []string{"aw", "compile"}
		if args.Engine != "" {
			cmdArgs = append(cmdArgs, "--engine", args.Engine)
		}
		if !args.Validate {
			cmdArgs = append(cmdArgs, "--validate=false")
		}
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
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "logs",
		Description: "Download and analyze workflow logs",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		// Force output directory to /tmp/aw-mcp/logs for MCP server
		cmdArgs := []string{"aw", "logs", "-o", "/tmp/aw-mcp/logs"}
		if args.WorkflowName != "" {
			cmdArgs = append(cmdArgs, args.WorkflowName)
		}
		if args.Count > 0 {
			cmdArgs = append(cmdArgs, "-c", strconv.Itoa(args.Count))
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
		RunID     int64  `json:"run_id" jsonschema:"GitHub Actions workflow run ID to audit"`
		OutputDir string `json:"output_dir,omitempty" jsonschema:"Output directory for audit report"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "audit",
		Description: "Investigate a workflow run and generate a concise report",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args auditArgs) (*mcp.CallToolResult, any, error) {
		// Build command arguments
		cmdArgs := []string{"aw", "audit", strconv.FormatInt(args.RunID, 10)}
		if args.OutputDir != "" {
			cmdArgs = append(cmdArgs, "-o", args.OutputDir)
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

	// Run the server on stdio transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("MCP server failed: %w", err)
	}

	return nil
}
