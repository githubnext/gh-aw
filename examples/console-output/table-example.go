// Package main demonstrates table rendering with Lipgloss
//
// This example shows how to create styled tables with headers,
// zebra striping, and total rows using the console package.
//
// Run: go run examples/console-output/table-example.go
package main

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
)

func main() {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Table Rendering Examples"))
	fmt.Fprintln(os.Stderr, "")

	// Example 1: Basic table
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 1: Basic Workflow Status Table"))
	basicTable := console.TableConfig{
		Title: "Workflow Status",
		Headers: []string{"Name", "Status", "Duration", "Conclusion"},
		Rows: [][]string{
			{"issue-triage", "completed", "2m 30s", "success"},
			{"pr-review", "completed", "1m 45s", "success"},
			{"code-scan", "running", "1m 15s", "-"},
			{"deploy", "queued", "-", "-"},
		},
	}
	fmt.Fprint(os.Stderr, console.RenderTable(basicTable))
	fmt.Fprintln(os.Stderr, "")

	// Example 2: Table with total row
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 2: Cost Analysis with Totals"))
	costTable := console.TableConfig{
		Title: "Workflow Cost Analysis",
		Headers: []string{"Workflow", "Runs", "Avg Duration", "Total Cost"},
		Rows: [][]string{
			{"issue-triage", "25", "2m 15s", "$0.50"},
			{"pr-review", "15", "3m 30s", "$0.75"},
			{"code-scan", "40", "1m 45s", "$0.40"},
			{"deploy", "10", "5m 00s", "$1.25"},
		},
		ShowTotal: true,
		TotalRow: []string{"Total", "90", "-", "$2.90"},
	}
	fmt.Fprint(os.Stderr, console.RenderTable(costTable))
	fmt.Fprintln(os.Stderr, "")

	// Example 3: MCP Server status table
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 3: MCP Server Status"))
	mcpTable := console.TableConfig{
		Headers: []string{"Server", "Type", "Status", "Tools Available"},
		Rows: [][]string{
			{"github", "remote", "connected", "15"},
			{"filesystem", "stdio", "connected", "8"},
			{"playwright", "stdio", "disconnected", "-"},
			{"serena", "remote", "connected", "5"},
		},
	}
	fmt.Fprint(os.Stderr, console.RenderTable(mcpTable))
	fmt.Fprintln(os.Stderr, "")

	// Example 4: Job execution details
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 4: Job Execution Details"))
	jobTable := console.TableConfig{
		Title: "Workflow Run #12345",
		Headers: []string{"Job", "Status", "Started", "Duration", "Conclusion"},
		Rows: [][]string{
			{"setup", "completed", "10:00:00", "15s", "success"},
			{"build", "completed", "10:00:15", "2m 30s", "success"},
			{"test", "completed", "10:02:45", "1m 45s", "success"},
			{"deploy", "completed", "10:04:30", "3m 15s", "success"},
		},
		ShowTotal: true,
		TotalRow: []string{"Total", "-", "-", "7m 45s", "-"},
	}
	fmt.Fprint(os.Stderr, console.RenderTable(jobTable))
	fmt.Fprintln(os.Stderr, "")

	// Example 5: JSON output for machine consumption
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 5: JSON Table Output"))
	jsonTable := console.TableConfig{
		Headers: []string{"Name", "Status", "Cost"},
		Rows: [][]string{
			{"workflow-1", "success", "$0.50"},
			{"workflow-2", "failed", "$0.25"},
		},
	}
	
	jsonOutput, err := console.RenderTableAsJSON(jsonTable)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to render JSON: %v", err)))
		os.Exit(1)
	}
	
	// Note: JSON output goes to stdout, not stderr
	fmt.Println(jsonOutput)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("JSON output written to stdout"))
	fmt.Fprintln(os.Stderr, "")

	// Example 6: Large dataset with zebra striping
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 6: Large Dataset (Zebra Striping)"))
	largeTable := console.TableConfig{
		Title: "Recent Workflow Runs",
		Headers: []string{"Run ID", "Workflow", "Status", "Duration"},
		Rows: [][]string{
			{"12345", "issue-triage", "success", "2m 30s"},
			{"12344", "pr-review", "success", "3m 15s"},
			{"12343", "code-scan", "failed", "1m 45s"},
			{"12342", "deploy", "success", "5m 00s"},
			{"12341", "issue-triage", "success", "2m 20s"},
			{"12340", "pr-review", "success", "3m 30s"},
			{"12339", "code-scan", "success", "1m 50s"},
			{"12338", "deploy", "cancelled", "0m 30s"},
		},
	}
	fmt.Fprint(os.Stderr, console.RenderTable(largeTable))

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Table examples completed"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Tables automatically adapt to TTY vs non-TTY output"))
}
