package cli

import (
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var mcpCommandLog = logger.New("cli:mcp")

// NewMCPCommand creates the main mcp command with subcommands
func NewMCPCommand() *cobra.Command {
	mcpCommandLog.Print("Creating MCP command with subcommands")
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP (Model Context Protocol) servers",
		Long: `Model Context Protocol (MCP) server management and inspection.

MCP enables AI workflows to connect to external tools and data sources through
standardized servers. This command provides tools for inspecting and managing
MCP server configurations in your agentic workflows.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewMCPAddSubcommand())
	cmd.AddCommand(NewMCPListSubcommand())
	cmd.AddCommand(NewMCPListToolsSubcommand())
	cmd.AddCommand(NewMCPInspectSubcommand())

	return cmd
}
