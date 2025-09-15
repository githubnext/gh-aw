package cli

import (
	"github.com/spf13/cobra"
)

// NewMCPCommand creates the mcp command with subcommands
func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol (MCP) server management",
		Long: `Manage Model Context Protocol (MCP) servers used by agentic workflows.
		
This command provides subcommands for inspecting, configuring, and launching MCP servers.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewMCPInspectSubCommand())

	return cmd
}