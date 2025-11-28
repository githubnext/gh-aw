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
		Short: "Manage MCP (Model Context Protocol) servers in agentic workflows",
		Long: `Manage Model Context Protocol (MCP) servers in your agentic workflows.

MCP enables AI workflows to connect to external tools and data sources through
standardized servers. Use these commands to discover, add, and inspect MCP server
configurations in your workflow files.

SUBCOMMANDS:
  list        Discover which workflows have MCP servers configured
  list-tools  See what tools are available from a specific MCP server
  inspect     Start MCP servers and query their capabilities in detail
  add         Add a new MCP server from the registry to a workflow

EXAMPLES:
  # See all workflows with MCP servers
  gh aw mcp list

  # Add the GitHub MCP server to a workflow
  gh aw mcp add my-workflow github

  # List tools available from the playwright MCP server
  gh aw mcp list-tools playwright my-workflow

  # Inspect all MCP servers in a workflow with detailed output
  gh aw mcp inspect my-workflow -v

COMMON WORKFLOWS:
  1. Discover: gh aw mcp list → Find workflows with MCP configurations
  2. Explore:  gh aw mcp inspect <workflow> → See available tools
  3. Add:      gh aw mcp add <workflow> <server> → Add new MCP server`,
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
