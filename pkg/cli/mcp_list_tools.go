package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

const (
	// maxDescriptionLength is the maximum length for tool descriptions before truncation
	maxDescriptionLength = 60
	// truncationLength is the length at which to truncate descriptions (leaving room for "...")
	truncationLength = 57
)

// ListToolsForMCP lists available tools for a specific MCP server
func ListToolsForMCP(workflowFile string, mcpServerName string, verbose bool) error {
	workflowsDir := getWorkflowsDir()

	// If no workflow file specified, search for workflows containing the MCP server
	if workflowFile == "" {
		return findWorkflowsWithMCPServer(workflowsDir, mcpServerName, verbose)
	}

	// Load the workflow file with MCP configurations
	workflowInfo, err := loadWorkflowWithMCP(workflowFile, mcpServerName)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Looking for MCP server '%s' in: %s", mcpServerName, workflowInfo.FilePath)))
	}

	// Find the specific MCP server
	var targetConfig *parser.MCPServerConfig
	for _, config := range workflowInfo.MCPConfigs {
		if strings.EqualFold(config.Name, mcpServerName) {
			targetConfig = &config
			break
		}
	}

	if targetConfig == nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("MCP server '%s' not found in workflow '%s'", mcpServerName, filepath.Base(workflowInfo.FilePath))))

		// Show available servers
		if len(workflowInfo.MCPConfigs) > 0 {
			fmt.Fprintf(os.Stderr, "Available MCP servers: ")
			serverNames := make([]string, len(workflowInfo.MCPConfigs))
			for i, config := range workflowInfo.MCPConfigs {
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
	// Scan workflows directory for workflows with the specific MCP server
	workflowInfos, err := scanWorkflowsDirectory(workflowsDir, mcpServerName, verbose)
	if err != nil {
		return err
	}

	var matchingWorkflows []string

	// Filter workflows that contain the target MCP server
	for _, info := range workflowInfos {
		for _, config := range info.MCPConfigs {
			if strings.EqualFold(config.Name, mcpServerName) {
				matchingWorkflows = append(matchingWorkflows, info.Name)
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

	// Create a map for quick lookup of allowed tools from workflow configuration
	allowedMap := make(map[string]bool)

	// Check for wildcard "*" which means all tools are allowed
	hasWildcard := false
	for _, allowed := range info.Config.Allowed {
		if allowed == "*" {
			hasWildcard = true
		}
		allowedMap[allowed] = true
	}

	if verbose {
		// Detailed table with full descriptions
		headers := []string{"Tool Name", "Allow", "Description"}
		rows := make([][]string, 0, len(info.Tools))

		for _, tool := range info.Tools {
			// In verbose mode, show full descriptions without truncation
			description := tool.Description

			// Determine status
			status := "🚫"
			if len(info.Config.Allowed) == 0 || hasWildcard {
				// If no allowed list is specified or "*" wildcard is present, assume all tools are allowed
				status = "✅"
			} else if allowedMap[tool.Name] {
				status = "✅"
			}

			rows = append(rows, []string{tool.Name, status, description})
		}

		table := console.RenderTable(console.TableConfig{
			Headers: headers,
			Rows:    rows,
		})
		fmt.Print(table)

		// Display summary
		allowedCount := 0
		for _, tool := range info.Tools {
			if len(info.Config.Allowed) == 0 || hasWildcard || allowedMap[tool.Name] {
				allowedCount++
			}
		}
		fmt.Printf("\n📊 Summary: %d allowed, %d not allowed out of %d total tools\n",
			allowedCount, len(info.Tools)-allowedCount, len(info.Tools))
	} else {
		// Compact table with truncated descriptions for single-line display
		headers := []string{"Tool Name", "Allow", "Description"}
		rows := make([][]string, 0, len(info.Tools))

		for _, tool := range info.Tools {
			// In non-verbose mode, truncate descriptions to keep tools on single lines
			description := tool.Description
			if len(description) > maxDescriptionLength {
				description = description[:truncationLength] + "..."
			}

			// Determine status
			status := "🚫"
			if len(info.Config.Allowed) == 0 || hasWildcard {
				// If no allowed list is specified or "*" wildcard is present, assume all tools are allowed
				status = "✅"
			} else if allowedMap[tool.Name] {
				status = "✅"
			}

			rows = append(rows, []string{tool.Name, status, description})
		}

		table := console.RenderTable(console.TableConfig{
			Headers: headers,
			Rows:    rows,
		})
		fmt.Print(table)
		fmt.Printf("\nRun with --verbose for detailed information\n")
	}
}

// NewMCPListToolsSubcommand creates the mcp list-tools subcommand
func NewMCPListToolsSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-tools <mcp_server> [workflow-file]",
		Short: "List available tools for a specific MCP server",
		Long: `List available tools for a specific MCP server.

This command connects to the specified MCP server and displays all available tools.
It reuses the same infrastructure as 'mcp inspect' to establish connections and
query server capabilities.

Examples:
  gh aw mcp list-tools github                    # Find workflows with 'github' MCP server
  gh aw mcp list-tools github weekly-research    # List tools for 'github' server in weekly-research.md
  gh aw mcp list-tools safe-outputs issue-triage # List tools for 'safe-outputs' server in issue-triage.md
  gh aw mcp list-tools playwright test-workflow -v  # Verbose output with tool descriptions

The command will:
- Parse the workflow to find the specified MCP server configuration
- Connect to the MCP server using the same logic as 'mcp inspect'
- Display available tools with their descriptions and allowance status`,
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
