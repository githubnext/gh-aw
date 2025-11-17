package mcp

import (
	"github.com/spf13/cobra"
)

func NewCmdMCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP servers and tools",
		Long: `
Available subcommands:
  add        Add an MCP tool to an agentic workflow
  list       List MCP servers defined in agentic workflows
  listtools  List available tools for a specific MCP server
  inspect    Inspect MCP servers and list available tools, resources, and roots
`,
	}

	// subcommands will be added here later
	return cmd
}
