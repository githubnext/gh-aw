package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var mcpListToolsLog = logger.New("cli:mcp_list_tools")

const (
	// maxDescriptionLength is the maximum length for tool descriptions before truncation
	maxDescriptionLength = 60
)

// ListToolsForMCP lists available tools for a specific MCP server
func ListToolsForMCP(workflowFile string, mcpServerName string, verbose bool) error {
	mcpListToolsLog.Printf("Listing tools for MCP server: %s, workflow: %s", mcpServerName, workflowFile)
	workflowsDir := getWorkflowsDir()

	// If no workflow file specified, search for workflows containing the MCP server
	if workflowFile == "" {
		mcpListToolsLog.Printf("No workflow file specified, searching in: %s", workflowsDir)
		return findWorkflowsWithMCPServer(workflowsDir, mcpServerName, verbose)
	}

	// Resolve the workflow file path
	workflowPath, err := ResolveWorkflowPath(workflowFile)
	if err != nil {
		return err
	}

	// Convert to absolute path if needed
	if !filepath.IsAbs(workflowPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		workflowPath = filepath.Join(cwd, workflowPath)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Looking for MCP server '%s' in: %s", mcpServerName, workflowPath)))
	}

	// Parse the workflow file and extract MCP configurations
	_, mcpConfigs, err := loadWorkflowMCPConfigs(workflowPath, mcpServerName)
	if err != nil {
		return err
	}

	// Find the specific MCP server
	var targetConfig *parser.MCPServerConfig
	for _, config := range mcpConfigs {
		if strings.EqualFold(config.Name, mcpServerName) {
			targetConfig = &config
			break
		}
	}

	if targetConfig == nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("MCP server '%s' not found in workflow '%s'", mcpServerName, filepath.Base(workflowPath))))

		// Show available servers
		if len(mcpConfigs) > 0 {
			fmt.Fprintf(os.Stderr, "Available MCP servers: ")
			serverNames := make([]string, len(mcpConfigs))
			for i, config := range mcpConfigs {
				serverNames[i] = config.Name
			}
			fmt.Fprintf(os.Stderr, "%s\n", strings.Join(serverNames, ", "))
		}
		return nil
	}

	// Connect to the MCP server and get its tools
	fmt.Printf("%s %s (%s)\n",
		console.FormatInfoMessage("📡 Connecting to MCP server:"),
		targetConfig.Name,
		targetConfig.Type)

	info, err := connectToMCPServer(*targetConfig, verbose)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server '%s': %w", mcpServerName, err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully connected to MCP server"))
	}

	// Display the tools
	displayToolsList(info, verbose)

	return nil
}

// findWorkflowsWithMCPServer searches for workflows containing a specific MCP server
func findWorkflowsWithMCPServer(workflowsDir string, mcpServerName string, verbose bool) error {
	// Scan workflows for MCP configurations, filtering by server name
	results, err := ScanWorkflowsForMCP(workflowsDir, mcpServerName, verbose)
	if err != nil {
		return err
	}

	var matchingWorkflows []string

	for _, result := range results {
		// Check if this workflow contains the target MCP server
		for _, config := range result.MCPConfigs {
			if strings.EqualFold(config.Name, mcpServerName) {
				matchingWorkflows = append(matchingWorkflows, result.BaseName)
				break
			}
		}
	}

	if len(matchingWorkflows) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("MCP server '%s' not found in any workflow", mcpServerName)))
		return nil
	}

	// Display matching workflows and suggest using one
	fmt.Fprintf(os.Stderr, "Found MCP server '%s' in %d workflow(s): %s\n",
		mcpServerName, len(matchingWorkflows), strings.Join(matchingWorkflows, ", "))
	fmt.Fprintf(os.Stderr, "\nRun 'gh aw mcp list-tools %s <workflow-name>' to list tools for a specific workflow\n", mcpServerName)

	return nil
}

// displayToolsList shows the tools available from the MCP server in a formatted table
func displayToolsList(info *parser.MCPServerInfo, verbose bool) {
	if len(info.Tools) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No tools available from this MCP server"))
		return
	}

	fmt.Printf("\n%s\n", console.FormatInfoMessage(fmt.Sprintf("🛠️  Available Tools (%d total)", len(info.Tools))))

	// Configure options based on verbose flag
	opts := MCPToolTableOptions{
		ShowSummary: true,
	}

	if verbose {
		// In verbose mode, show full descriptions without truncation
		opts.TruncateLength = 0
		opts.ShowVerboseHint = false
	} else {
		// In non-verbose mode, truncate descriptions to keep tools on single lines
		opts.TruncateLength = maxDescriptionLength
		opts.ShowVerboseHint = true
	}

	// Render the table using the shared helper
	table := renderMCPToolTable(info, opts)
	fmt.Print(table)
}

// NewMCPListToolsSubcommand creates the mcp list-tools subcommand
func NewMCPListToolsSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-tools <mcp-server> [workflow-id-or-file]",
		Short: "List all tools available from a specific MCP server",
		Long: `List available tools from a specific MCP server.

This command connects to the specified MCP server and displays all available tools.
If no workflow is specified, it searches for workflows containing that MCP server.

ARGUMENTS:
  mcp-server             Required. Name of the MCP server (e.g., "github", "playwright")
  workflow-id-or-file    Optional. Can be:
                         - A workflow ID (e.g., "weekly-research")
                         - A file path (e.g., "weekly-research.md")

EXAMPLES:
  # Find which workflows use the 'github' MCP server
  gh aw mcp list-tools github

  # List tools from 'github' server in a specific workflow
  gh aw mcp list-tools github weekly-research

  # List tools from 'playwright' server with full descriptions
  gh aw mcp list-tools playwright my-workflow -v

  # List tools from custom MCP server
  gh aw mcp list-tools notion weekly-research

OUTPUT:
  Without workflow - Shows workflows containing the MCP server:
    Found MCP server 'github' in 2 workflow(s): weekly-research, issue-triage

  With workflow - Shows tools table:
    🛠️  Available Tools (15 total)
    ┌──────────────────┬─────────────────────────────────┬─────────┐
    │ Tool             │ Description                     │ Allowed │
    ├──────────────────┼─────────────────────────────────┼─────────┤
    │ create_issue     │ Create a GitHub issue           │ ✓       │
    │ list_issues      │ List issues in a repository     │ ✓       │
    │ add_comment      │ Add a comment to an issue or PR │ ✗       │
    └──────────────────┴─────────────────────────────────┴─────────┘

  With --verbose - Shows full tool descriptions without truncation`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mcpServerName := args[0]
			var workflowFile string
			if len(args) > 1 {
				workflowFile = args[1]
			}

			verbose, _ := cmd.Flags().GetBool("verbose")

			return ListToolsForMCP(workflowFile, mcpServerName, verbose)
		},
	}

	return cmd
}
