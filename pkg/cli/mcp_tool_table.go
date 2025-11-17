package cli

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// MCPToolTableOptions configures how the MCP tool table is rendered
type MCPToolTableOptions struct {
	// TruncateLength is the maximum length for tool descriptions before truncation
	// A value of 0 means no truncation
	TruncateLength int
	// ShowSummary controls whether to display the summary line at the bottom
	ShowSummary bool
	// SummaryFormat is the format string for the summary (default: "📊 Summary: %d allowed, %d not allowed out of %d total tools\n")
	SummaryFormat string
	// ShowVerboseHint controls whether to show the "Run with --verbose" hint in non-verbose mode
	ShowVerboseHint bool
}

// DefaultMCPToolTableOptions returns the default options for rendering MCP tool tables
func DefaultMCPToolTableOptions() MCPToolTableOptions {
	return MCPToolTableOptions{
		TruncateLength:  60,
		ShowSummary:     true,
		SummaryFormat:   "\n📊 Summary: %d allowed, %d not allowed out of %d total tools\n",
		ShowVerboseHint: false,
	}
}

// renderMCPToolTable renders an MCP tool table with configurable options
// This is the shared rendering logic used by both mcp list-tools and mcp inspect commands
func renderMCPToolTable(info *parser.MCPServerInfo, opts MCPToolTableOptions) string {
	if len(info.Tools) == 0 {
		return ""
	}

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

	// Build table headers and rows
	headers := []string{"Tool Name", "Allow", "Description"}
	rows := make([][]string, 0, len(info.Tools))

	for _, tool := range info.Tools {
		description := tool.Description

		// Apply truncation if requested
		if opts.TruncateLength > 0 && len(description) > opts.TruncateLength {
			// Leave room for "..."
			truncateAt := opts.TruncateLength - 3
			if truncateAt > 0 {
				description = description[:truncateAt] + "..."
			}
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

	// Render the table
	table := console.RenderTable(console.TableConfig{
		Headers: headers,
		Rows:    rows,
	})

	result := table

	// Add summary if requested
	if opts.ShowSummary {
		allowedCount := 0
		for _, tool := range info.Tools {
			if len(info.Config.Allowed) == 0 || hasWildcard || allowedMap[tool.Name] {
				allowedCount++
			}
		}

		summaryFormat := opts.SummaryFormat
		if summaryFormat == "" {
			summaryFormat = "\n📊 Summary: %d allowed, %d not allowed out of %d total tools\n"
		}

		result += fmt.Sprintf(summaryFormat,
			allowedCount, len(info.Tools)-allowedCount, len(info.Tools))
	}

	// Add verbose hint if requested
	if opts.ShowVerboseHint {
		result += "\nRun with --verbose for detailed information\n"
	}

	return result
}
