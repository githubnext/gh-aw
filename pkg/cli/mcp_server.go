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

// redirectStdoutToString captures stdout during function execution and returns it as a string
func redirectStdoutToString(fn func() error) (string, error) {
	// Save original stdout
	originalStdout := os.Stdout

	// Create a pipe to capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipe: %w", err)
	}

	// Redirect stdout to the pipe
	os.Stdout = w

	// Execute the function
	fnErr := fn()

	// Close the write end and restore stdout
	w.Close()
	os.Stdout = originalStdout

	// Read the captured output
	output, readErr := io.ReadAll(r)
	r.Close()

	if readErr != nil {
		return "", fmt.Errorf("failed to read captured output: %w", readErr)
	}

	// Return the captured output and any error from the function
	if fnErr != nil {
		return string(output), fnErr
	}

	return string(output), nil
}

// createMCPServer creates and configures the MCP server with specified CLI tools
func createMCPServer(verbose bool, allowedTools []string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "github_agentic_workflows",
		Version: GetVersion(),
	}, nil)

	// Helper function to check if a tool is allowed
	isToolAllowed := func(toolName string) bool {
		if len(allowedTools) == 0 {
			return true // If no filter specified, allow all tools
		}
		for _, allowed := range allowedTools {
			if allowed == toolName {
				return true
			}
		}
		return false
	}

	// Add compile tool
	if isToolAllowed("compile") {
		type compileArgs struct {
			Files    []string `json:"files,omitempty"`
			Verbose  bool     `json:"verbose,omitempty"`
			Validate bool     `json:"validate,omitempty"`
			Purge    bool     `json:"purge,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "compile",
			Description: "Compile markdown workflow files to YAML",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args compileArgs) (*mcp.CallToolResult, any, error) {
			if verbose || args.Verbose {
				fmt.Fprintf(os.Stderr, "🔧 Compiling workflows...\n")
			}

			// Capture the compilation output using the helper function
			output, err := redirectStdoutToString(func() error {
				return CompileWorkflows(args.Files, args.Verbose || verbose, "", args.Validate, false, "", false, false, args.Purge)
			})

			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error compiling workflows: %v", err)}},
				}, nil, nil
			}

			// Provide the actual output or a default message based on args
			if output == "" {
				message := "Successfully compiled all workflow files"
				if len(args.Files) > 0 {
					message = fmt.Sprintf("Successfully compiled %d workflow file(s)", len(args.Files))
				}
				output = message
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: output}},
			}, nil, nil
		})
	}

	// Add logs tool
	if isToolAllowed("logs") {
		type logsArgs struct {
			Workflow string `json:"workflow,omitempty"`
			Count    int    `json:"count,omitempty"`
			Engine   string `json:"engine,omitempty"`
			Verbose  bool   `json:"verbose,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "logs",
			Description: "Download and analyze agentic workflow logs",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args logsArgs) (*mcp.CallToolResult, any, error) {
			if verbose || args.Verbose {
				fmt.Fprintf(os.Stderr, "📊 Downloading workflow logs...\n")
			}

			count := args.Count
			if count <= 0 {
				count = 30
			}

			// Capture the logs output using the helper function
			output, err := redirectStdoutToString(func() error {
				return DownloadWorkflowLogs(args.Workflow, count, "", "", "", args.Engine, "", 0, 0, args.Verbose || verbose, false, false)
			})

			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error downloading logs: %v", err)}},
				}, nil, nil
			}

			// Provide the actual output or a default message
			if output == "" {
				output = "Successfully downloaded and analyzed workflow logs"
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: output}},
			}, nil, nil
		})
	}

	// Add mcp inspect tool
	if isToolAllowed("mcp_inspect") {
		type mcpInspectArgs struct {
			Workflow string `json:"workflow,omitempty"`
			Server   string `json:"server,omitempty"`
			Tool     string `json:"tool,omitempty"`
			Verbose  bool   `json:"verbose,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "mcp_inspect",
			Description: "Inspect MCP servers and list available tools",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpInspectArgs) (*mcp.CallToolResult, any, error) {
			if verbose || args.Verbose {
				fmt.Fprintf(os.Stderr, "🔍 Inspecting MCP servers...\n")
			}

			// Capture the inspection output using the helper function
			output, err := redirectStdoutToString(func() error {
				return InspectWorkflowMCP(args.Workflow, args.Server, args.Tool, args.Verbose || verbose)
			})

			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error inspecting MCP servers: %v", err)}},
				}, nil, nil
			}

			// Provide the actual output or a default message
			if output == "" {
				output = "MCP server inspection completed"
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: output}},
			}, nil, nil
		})
	}

	// Add mcp list tool
	if isToolAllowed("mcp_list") {
		type mcpListArgs struct {
			Workflow string `json:"workflow,omitempty"`
			Verbose  bool   `json:"verbose,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "mcp_list",
			Description: "List MCP servers defined in agentic workflows",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpListArgs) (*mcp.CallToolResult, any, error) {
			if verbose || args.Verbose {
				fmt.Fprintf(os.Stderr, "📋 Listing MCP servers...\n")
			}

			// Capture the list output using the helper function
			output, err := redirectStdoutToString(func() error {
				return ListWorkflowMCP(args.Workflow, args.Verbose || verbose)
			})

			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error listing MCP servers: %v", err)}},
				}, nil, nil
			}

			// Provide the actual output or a default message
			if output == "" {
				output = "MCP server listing completed"
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: output}},
			}, nil, nil
		})
	}

	// Add mcp add tool
	if isToolAllowed("mcp_add") {
		type mcpAddArgs struct {
			Workflow  string `json:"workflow"`
			Server    string `json:"server"`
			Registry  string `json:"registry,omitempty"`
			Transport string `json:"transport,omitempty"`
			ToolID    string `json:"tool_id,omitempty"`
			Verbose   bool   `json:"verbose,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "mcp_add",
			Description: "Add an MCP tool to an agentic workflow",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpAddArgs) (*mcp.CallToolResult, any, error) {
			if verbose || args.Verbose {
				fmt.Fprintf(os.Stderr, "➕ Adding MCP tool to workflow...\n")
			}

			if args.Workflow == "" || args.Server == "" {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: "Both workflow and server are required"}},
				}, nil, nil
			}

			// Capture the mcp_add output using the helper function
			output, err := redirectStdoutToString(func() error {
				return AddMCPTool(args.Workflow, args.Server, args.Registry, args.Transport, args.ToolID, args.Verbose || verbose)
			})

			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error adding MCP tool: %v", err)}},
				}, nil, nil
			}

			// Provide the actual output or a default message
			if output == "" {
				output = fmt.Sprintf("Successfully added MCP tool '%s' to workflow '%s'", args.Server, args.Workflow)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: output}},
			}, nil, nil
		})
	}

	// Add run tool
	if isToolAllowed("run") {
		type runArgs struct {
			Workflows []string `json:"workflows"`
			Repeat    int      `json:"repeat,omitempty"`
			Enable    bool     `json:"enable,omitempty"`
			Verbose   bool     `json:"verbose,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "run",
			Description: "Run agentic workflows on GitHub Actions",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args runArgs) (*mcp.CallToolResult, any, error) {
			if verbose || args.Verbose {
				fmt.Fprintf(os.Stderr, "🚀 Running workflows...\n")
			}

			if len(args.Workflows) == 0 {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: "At least one workflow is required"}},
				}, nil, nil
			}

			// Capture the run output using the helper function
			output, err := redirectStdoutToString(func() error {
				return RunWorkflowsOnGitHub(args.Workflows, args.Repeat, args.Enable, args.Verbose || verbose)
			})

			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error running workflows: %v", err)}},
				}, nil, nil
			}

			// Provide the actual output or generate a default message
			if output == "" {
				message := fmt.Sprintf("Successfully ran %d workflow(s)", len(args.Workflows))
				if len(args.Workflows) == 1 {
					message = fmt.Sprintf("Successfully ran workflow: %s", args.Workflows[0])
				}
				output = message
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: output}},
			}, nil, nil
		})
	}

	// Add enable tool
	if isToolAllowed("enable") {
		type enableArgs struct {
			Pattern string `json:"pattern,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "enable",
			Description: "Enable natural language action workflows",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args enableArgs) (*mcp.CallToolResult, any, error) {
			if verbose {
				fmt.Fprintf(os.Stderr, "✅ Enabling workflows...\n")
			}

			err := EnableWorkflows(args.Pattern)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error enabling workflows: %v", err)}},
				}, nil, nil
			}

			message := "Successfully enabled all workflows"
			if args.Pattern != "" {
				message = fmt.Sprintf("Successfully enabled workflows matching pattern: %s", args.Pattern)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: message}},
			}, nil, nil
		})
	}

	// Add disable tool
	if isToolAllowed("disable") {
		type disableArgs struct {
			Pattern string `json:"pattern,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "disable",
			Description: "Disable natural language action workflows",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args disableArgs) (*mcp.CallToolResult, any, error) {
			if verbose {
				fmt.Fprintf(os.Stderr, "❌ Disabling workflows...\n")
			}

			err := DisableWorkflows(args.Pattern)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error disabling workflows: %v", err)}},
				}, nil, nil
			}

			message := "Successfully disabled all workflows"
			if args.Pattern != "" {
				message = fmt.Sprintf("Successfully disabled workflows matching pattern: %s", args.Pattern)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: message}},
			}, nil, nil
		})
	}

	// Add status tool
	if isToolAllowed("status") {
		type statusArgs struct {
			Pattern string `json:"pattern,omitempty"`
			Verbose bool   `json:"verbose,omitempty"`
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "status",
			Description: "Show status of natural language action files and workflows",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, any, error) {
			if verbose || args.Verbose {
				fmt.Fprintf(os.Stderr, "📊 Checking workflow status...\n")
			}

			// Capture the status output using the helper function
			output, statusErr := redirectStdoutToString(func() error {
				return StatusWorkflows(args.Pattern, args.Verbose || verbose)
			})

			if statusErr != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error checking status: %v", statusErr)}},
				}, nil, nil
			}

			// Provide a default message if no output was captured
			if output == "" {
				output = "Status check completed successfully"
			}

			if args.Pattern != "" {
				output = fmt.Sprintf("Status for pattern '%s':\n%s", args.Pattern, output)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: output}},
			}, nil, nil
		})
	}

	return server
}

// NewMCPServerSubcommand creates the mcp server subcommand
func NewMCPServerSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch an MCP server exposing CLI tools",
		Long: `Launch a Model Context Protocol server that exposes various gh-aw CLI commands as MCP tools.

This command starts an MCP server that can be used by AI assistants and other MCP clients
to interact with GitHub Agentic Workflows functionality. The server exposes the following tools:

  compile      - Compile markdown workflow files to YAML
  logs         - Download and analyze agentic workflow logs  
  mcp_inspect  - Inspect MCP servers and list available tools
  mcp_list     - List MCP servers defined in agentic workflows
  mcp_add      - Add MCP tools to agentic workflows
  run          - Run agentic workflows on GitHub Actions
  enable       - Enable workflows
  disable      - Disable workflows
  status       - Show status of natural language action files and workflows

The server uses stdio transport by default, making it suitable for use with various MCP clients.

Examples:
  gh aw mcp serve                     # Start MCP server on stdio
  gh aw mcp serve -v                  # Start with verbose logging
  gh aw mcp serve --allowed-tools compile,logs  # Only expose compile and logs tools
  gh aw mcp serve --allowed-tools run,enable,disable  # Only expose workflow management tools`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			allowedTools, _ := cmd.Flags().GetStringSlice("allowed-tools")

			// Inherit verbose from parent commands
			if !verbose {
				if cmd.Parent() != nil {
					if parentVerbose, _ := cmd.Parent().PersistentFlags().GetBool("verbose"); parentVerbose {
						verbose = true
					}
					if cmd.Parent().Parent() != nil {
						if rootVerbose, _ := cmd.Parent().Parent().PersistentFlags().GetBool("verbose"); rootVerbose {
							verbose = true
						}
					}
				}
			}

			return runMCPServer(verbose, allowedTools)
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output with detailed logging")
	cmd.Flags().StringSlice("allowed-tools", []string{}, "Comma-separated list of tools to enable (compile,logs,mcp_inspect,mcp_list,mcp_add,run,enable,disable,status). If not specified, all tools are enabled.")

	return cmd
}

// runMCPServer starts and runs the MCP server
func runMCPServer(verbose bool, allowedTools []string) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting GitHub Agentic Workflows MCP server..."))
		if len(allowedTools) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Enabled tools: %v", allowedTools)))
		}
	}

	// Create the MCP server with specified tools
	server := createMCPServer(verbose, allowedTools)

	// Create stdio transport
	transport := &mcp.StdioTransport{}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("MCP server ready on stdio"))
	}

	// Start the server (this blocks until the client disconnects)
	ctx := context.Background()
	if err := server.Run(ctx, transport); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("MCP server stopped"))
	}

	return nil
}
