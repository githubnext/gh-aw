package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewMCPCommand creates the main mcp command with subcommands
func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol (MCP) server management and inspection",
		Long: `Model Context Protocol (MCP) server management and inspection.

MCP enables AI workflows to connect to external tools and data sources through
standardized servers. This command provides tools for inspecting and managing
MCP server configurations in your agentic workflows.

Available subcommands:
  inspect  - Inspect MCP servers and list available tools, resources, and roots`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewMCPInspectSubcommand())

	return cmd
}

// NewMCPInspectSubcommand creates the mcp inspect subcommand
// This is the former mcp-inspect command now nested under mcp
func NewMCPInspectSubcommand() *cobra.Command {
	var serverFilter string
	var toolFilter string
	var spawnInspector bool

	cmd := &cobra.Command{
		Use:   "inspect [workflow-file]",
		Short: "Inspect MCP servers and list available tools, resources, and roots",
		Long: `Inspect MCP servers used by a workflow and display available tools, resources, and roots.

This command starts each MCP server configured in the workflow, queries its capabilities,
and displays the results in a formatted table. It supports stdio, Docker, and HTTP MCP servers.

Examples:
  gh aw mcp inspect                    # List workflows with MCP servers
  gh aw mcp inspect weekly-research    # Inspect MCP servers in weekly-research.md  
  gh aw mcp inspect repomind --server repo-mind  # Inspect only the repo-mind server
  gh aw mcp inspect weekly-research --server github --tool create_issue  # Show details for a specific tool
  gh aw mcp inspect weekly-research -v # Verbose output with detailed connection info
  gh aw mcp inspect weekly-research --inspector  # Launch @modelcontextprotocol/inspector

The command will:
- Parse the workflow file to extract MCP server configurations
- Start each MCP server (stdio, docker, http)
- Query available tools, resources, and roots
- Validate required secrets are available  
- Display results in formatted tables with error details`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var workflowFile string
			if len(args) > 0 {
				workflowFile = args[0]
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			// Check for verbose flag from parent commands (root and mcp)
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

			// Validate that tool flag requires server flag
			if toolFilter != "" && serverFilter == "" {
				return fmt.Errorf("--tool flag requires --server flag to be specified")
			}

			// Handle spawn inspector flag
			if spawnInspector {
				return spawnMCPInspector(workflowFile, serverFilter, verbose)
			}

			return InspectWorkflowMCP(workflowFile, serverFilter, toolFilter, verbose)
		},
	}

	cmd.Flags().StringVar(&serverFilter, "server", "", "Filter to inspect only the specified MCP server")
	cmd.Flags().StringVar(&toolFilter, "tool", "", "Show detailed information about a specific tool (requires --server)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output with detailed connection information")
	cmd.Flags().BoolVar(&spawnInspector, "inspector", false, "Launch the official @modelcontextprotocol/inspector tool")

	return cmd
}
