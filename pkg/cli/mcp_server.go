package cli

import (
	"context"
	"fmt"
	"io"
	"os"

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

This command starts a stdio-based MCP server that provides the following tools:
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

	// Redirect all logging to /dev/null to avoid interfering with stdio protocol
	// Store original stderr for restoration if needed
	originalStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/null: %w", err)
	}
	defer devNull.Close()

	// Redirect stderr to /dev/null
	os.Stderr = devNull

	// Also redirect the console output by setting a discard writer
	// This ensures no console formatting output interferes with MCP protocol
	defer func() {
		os.Stderr = originalStderr
	}()

	// Add status tool
	type statusArgs struct {
		Pattern string `json:"pattern,omitempty" jsonschema:"Optional pattern to filter workflows by name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "status",
		Description: "Show status of agentic workflow files and workflows",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, any, error) {
		// Capture stdout to return as tool result
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run status command
		err := StatusWorkflows(args.Pattern, false)

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		output, _ := io.ReadAll(r)

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)},
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
		// Capture stdout to return as tool result
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		config := CompileConfig{
			MarkdownFiles:    args.Workflows,
			Verbose:          false,
			EngineOverride:   args.Engine,
			Validate:         args.Validate,
			Watch:            false,
			WorkflowDir:      "",
			SkipInstructions: false,
			NoEmit:           false,
			Purge:            false,
			TrialMode:        false,
			Strict:           false,
		}

		_, err := CompileWorkflows(config)

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		output, _ := io.ReadAll(r)

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
		OutputDir    string `json:"output_dir,omitempty" jsonschema:"Output directory for logs"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "logs",
		Description: "Download and analyze workflow logs",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, any, error) {
		// Capture stdout to return as tool result
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputDir := args.OutputDir
		if outputDir == "" {
			outputDir = "./logs"
		}

		count := args.Count
		if count == 0 {
			count = 5
		}

		err := DownloadWorkflowLogs(args.WorkflowName, count, "", "", outputDir, "", "", 0, 0, false, false, false)

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		output, _ := io.ReadAll(r)

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
		// Capture stdout to return as tool result
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputDir := args.OutputDir
		if outputDir == "" {
			outputDir = "./logs"
		}

		err := AuditWorkflowRun(args.RunID, outputDir, false)

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		output, _ := io.ReadAll(r)

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
