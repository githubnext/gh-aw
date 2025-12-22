package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var mcpListLog = logger.New("cli:mcp_list")

// ListWorkflowMCP lists MCP servers defined in a workflow
func ListWorkflowMCP(workflowFile string, verbose bool) error {
	mcpListLog.Printf("Listing MCP servers: workflow=%s, verbose=%t", workflowFile, verbose)
	// Determine the workflow directory and file
	workflowsDir := ".github/workflows"
	var workflowPath string

	if workflowFile != "" {
		// Resolve the workflow file path
		var err error
		workflowPath, err = ResolveWorkflowPath(workflowFile)
		if err != nil {
			mcpListLog.Printf("Failed to resolve workflow path: %v", err)
			return err
		}
		mcpListLog.Printf("Resolved workflow path: %s", workflowPath)
	} else {
		// No specific workflow file provided, list all workflows with MCP servers
		mcpListLog.Print("No workflow file specified, listing all workflows with MCP servers")
		return listWorkflowsWithMCPServers(workflowsDir, verbose)
	}

	// Parse the specific workflow file and extract MCP configurations
	_, mcpConfigs, err := loadWorkflowMCPConfigs(workflowPath, "")
	if err != nil {
		mcpListLog.Printf("Failed to load MCP configs from workflow: %v", err)
		return err
	}

	mcpListLog.Printf("Found %d MCP servers in workflow", len(mcpConfigs))
	if len(mcpConfigs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No MCP servers found in workflow"))
		return nil
	}

	// Display the MCP servers
	if verbose {
		// Create detailed table for verbose mode
		headers := []string{"Name", "Type", "Command/URL", "Args", "Allowed Tools", "Env Vars"}
		rows := make([][]string, 0, len(mcpConfigs))

		for _, config := range mcpConfigs {
			commandOrURL := ""
			if config.Command != "" {
				commandOrURL = config.Command
			} else if config.URL != "" {
				commandOrURL = config.URL
			} else if config.Container != "" {
				commandOrURL = config.Container
			}

			args := ""
			if len(config.Args) > 0 {
				args = strings.Join(config.Args, " ")
				// Truncate if too long
				if len(args) > 30 {
					args = args[:27] + "..."
				}
			}

			allowedTools := ""
			if len(config.Allowed) > 0 {
				allowedTools = strings.Join(config.Allowed, ", ")
				// Truncate if too long
				if len(allowedTools) > 30 {
					allowedTools = allowedTools[:27] + "..."
				}
			}

			envVars := ""
			if len(config.Env) > 0 {
				envVars = fmt.Sprintf("%d defined", len(config.Env))
			}

			rows = append(rows, []string{
				config.Name,
				config.Type,
				commandOrURL,
				args,
				allowedTools,
				envVars,
			})
		}

		tableConfig := console.TableConfig{
			Title:   fmt.Sprintf("MCP servers in %s", filepath.Base(workflowPath)),
			Headers: headers,
			Rows:    rows,
		}
		fmt.Print(console.RenderTable(tableConfig))
	} else {
		// Simple table for basic mode
		headers := []string{"Name", "Type"}
		rows := make([][]string, 0, len(mcpConfigs))

		for _, config := range mcpConfigs {
			rows = append(rows, []string{config.Name, config.Type})
		}

		tableConfig := console.TableConfig{
			Title:   fmt.Sprintf("MCP servers in %s", filepath.Base(workflowPath)),
			Headers: headers,
			Rows:    rows,
		}
		fmt.Fprint(os.Stderr, console.RenderTable(tableConfig))
	}

	if !verbose {
		fmt.Fprintf(os.Stderr, "\nRun 'gh aw mcp list %s --verbose' for detailed information\n", workflowFile)
	}

	return nil
}

// listWorkflowsWithMCPServers shows available workflow files that contain MCP configurations
func listWorkflowsWithMCPServers(workflowsDir string, verbose bool) error {
	// Scan workflows for MCP configurations
	results, err := ScanWorkflowsForMCP(workflowsDir, "", verbose)
	if err != nil {
		return err
	}

	var workflowData []struct {
		name        string
		serverCount int
		serverNames []string
	}
	var totalMCPCount int

	for _, result := range results {
		serverNames := make([]string, len(result.MCPConfigs))
		for i, config := range result.MCPConfigs {
			serverNames[i] = config.Name
		}

		workflowData = append(workflowData, struct {
			name        string
			serverCount int
			serverNames []string
		}{
			name:        result.BaseName,
			serverCount: len(result.MCPConfigs),
			serverNames: serverNames,
		})
		totalMCPCount += len(result.MCPConfigs)
	}

	if len(workflowData) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflows with MCP servers found"))
		return nil
	}

	// Display results in table format
	if verbose {
		// Detailed table with server names
		headers := []string{"Workflow", "Server Count", "MCP Servers"}
		rows := make([][]string, 0, len(workflowData))

		for _, workflow := range workflowData {
			serverList := strings.Join(workflow.serverNames, ", ")
			// Truncate if too long
			if len(serverList) > 50 {
				serverList = serverList[:47] + "..."
			}

			rows = append(rows, []string{
				workflow.name,
				fmt.Sprintf("%d", workflow.serverCount),
				serverList,
			})
		}

		tableConfig := console.TableConfig{
			Headers: headers,
			Rows:    rows,
		}
		fmt.Fprint(os.Stderr, console.RenderTable(tableConfig))
	} else {
		// Simple table with just workflow names and counts
		headers := []string{"Workflow", "Server Count"}
		rows := make([][]string, 0, len(workflowData))

		for _, workflow := range workflowData {
			rows = append(rows, []string{
				workflow.name,
				fmt.Sprintf("%d", workflow.serverCount),
			})
		}

		tableConfig := console.TableConfig{
			Headers: headers,
			Rows:    rows,
		}
		fmt.Fprint(os.Stderr, console.RenderTable(tableConfig))
	}

	if !verbose {
		fmt.Fprintf(os.Stderr, "\nRun 'gh aw mcp list --verbose' for detailed information\n")
	}
	fmt.Fprintf(os.Stderr, "Run 'gh aw mcp list <workflow-name>' to list MCP servers in a specific workflow\n")

	return nil
}

// NewMCPListSubcommand creates the mcp list subcommand
func NewMCPListSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [workflow]",
		Short: "List MCP servers defined in agentic workflows",
		Long: `List MCP servers defined in agentic workflows.

When no workflow ID/file is specified, lists all workflows that contain MCP server configurations.
When a workflow ID/file is specified, lists the MCP servers configured in that specific workflow.

The workflow-id-or-file can be:
- A workflow ID (basename without .md extension, e.g., "weekly-research")
- A file path (e.g., "weekly-research.md" or ".github/workflows/weekly-research.md")

Examples:
  gh aw mcp list                     # List all workflows with MCP servers
  gh aw mcp list weekly-research     # List MCP servers in weekly-research.md
  gh aw mcp list weekly-research -v  # List with detailed information
  gh aw mcp list --verbose           # List all workflows with detailed MCP server info

The command will:
- Parse workflow frontmatter to extract MCP server configurations
- Display server names and types
- In verbose mode, show detailed configuration including commands, URLs, and allowed tools`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var workflowFile string
			if len(args) > 0 {
				workflowFile = args[0]
			}

			verbose, _ := cmd.Flags().GetBool("verbose")

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

			return ListWorkflowMCP(workflowFile, verbose)
		},
	}

	// Register completions for mcp list command
	cmd.ValidArgsFunction = CompleteWorkflowNames

	return cmd
}
