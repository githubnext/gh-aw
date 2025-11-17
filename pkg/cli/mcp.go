package cli

import (
	"github.com/spf13/cobra"
)

// NewMCPCommand creates the main mcp command with subcommands
func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP (Model Context Protocol) servers",
		Long: `Model Context Protocol (MCP) server management and inspection.

MCP enables AI workflows to connect to external tools and data sources through
standardized servers. This command provides tools for inspecting and managing
MCP server configurations in your agentic workflows.

Available subcommands:
  add         Add an MCP tool to an agentic workflow
  list        List MCP servers defined in agentic workflows
  list-tools  List available tools for a specific MCP server
  inspect     Inspect MCP servers and list available tools, resources, and roots`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewMCPAddSubcommand())
	cmd.AddCommand(NewMCPListSubcommand())
	cmd.AddCommand(NewMCPListToolsSubcommand())
	cmd.AddCommand(NewMCPInspectSubcommand())

	return cmd
}
