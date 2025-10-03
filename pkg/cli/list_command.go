package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// GitHubWorkflow represents a GitHub Actions workflow from the API
// GitHubWorkflowsResponse represents the GitHub API response for workflows
// Note: The API returns an array directly, not wrapped in a workflows field

// ListEnginesAndOtherInformation lists available workflow components
func ListEnginesAndOtherInformation(verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Searching for available workflow components..."))
	}

	// List available agentic engines
	if err := listAgenticEngines(verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to list agentic engines: %v", err)))
	}

	// Provide information about workflow repositories
	fmt.Println("\nTo add workflows to your project:")
	fmt.Println("=================================")
	fmt.Println("Use the 'add' command with repository/workflow specifications:")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add owner/repo/workflow-name")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add owner/repo/workflow-name@version")
	fmt.Println("\nExample:")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add githubnext/agentics/ci-doctor")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add githubnext/agentics/daily-plan@main")
	return nil
}

// listAgenticEngines lists all available agentic engines with their characteristics
func listAgenticEngines(verbose bool) error {
	// Create an engine registry directly to access the engines
	registry := workflow.GetGlobalEngineRegistry()

	// Get all supported engines from the registry
	engines := registry.GetSupportedEngines()

	if len(engines) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No agentic engines available."))
		return nil
	}

	// Build table configuration
	var headers []string
	if verbose {
		headers = []string{"ID", "Display Name", "Status", "MCP", "HTTP Transport", "Description"}
	} else {
		headers = []string{"ID", "Display Name", "Status", "MCP", "HTTP Transport"}
	}

	var rows [][]string

	for _, engineID := range engines {
		engine, err := registry.GetEngine(engineID)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to get engine '%s': %v", engineID, err)))
			}
			continue
		}

		// Determine status
		status := "Stable"
		if engine.IsExperimental() {
			status = "Experimental"
		}

		// MCP support
		mcpSupport := "No"
		if engine.SupportsToolsAllowlist() {
			mcpSupport = "Yes"
		}

		// HTTP transport support
		httpTransport := "No"
		if engine.SupportsHTTPTransport() {
			httpTransport = "Yes"
		}

		// Build row data
		var row []string
		if verbose {
			row = []string{
				engine.GetID(),
				engine.GetDisplayName(),
				status,
				mcpSupport,
				httpTransport,
				engine.GetDescription(),
			}
		} else {
			row = []string{
				engine.GetID(),
				engine.GetDisplayName(),
				status,
				mcpSupport,
				httpTransport,
			}
		}
		rows = append(rows, row)
	}

	// Render the table
	tableConfig := console.TableConfig{
		Title:   "Available Agentic Engines",
		Headers: headers,
		Rows:    rows,
	}
	fmt.Fprint(os.Stderr, console.RenderTable(tableConfig))

	fmt.Fprintln(os.Stderr, "")
	return nil
}
