// Package main demonstrates basic console output formatting
//
// This example shows how to use the console package formatters
// for different message types with proper stderr usage.
//
// Run: go run examples/console-output/simple-output.go
package main

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
)

func main() {
	// Success messages - Use for completed operations
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow compiled successfully"))
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All tests passed"))

	// Info messages - Use for general information
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Processing 5 workflow files..."))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found 12 workflows in repository"))

	// Warning messages - Use for non-critical issues
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow file has uncommitted changes"))
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("MCP server connection timeout, retrying..."))

	// Error messages - Use for failures and critical issues
	fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Failed to compile workflow: syntax error in frontmatter"))
	fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Network connection failed"))

	// Location messages - Use for directory/file paths
	fmt.Fprintln(os.Stderr, console.FormatLocationMessage("Output saved to .github/aw/logs"))
	fmt.Fprintln(os.Stderr, console.FormatLocationMessage("Workflow directory: .github/workflows"))

	// Command messages - Use for command execution
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage("Running gh aw compile"))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage("Executing workflow validation"))

	// Progress messages - Use for ongoing activities
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Compiling workflows..."))
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Downloading artifacts..."))

	// Prompt messages - Use for user input requests
	fmt.Fprintln(os.Stderr, console.FormatPromptMessage("Enter workflow name:"))
	fmt.Fprintln(os.Stderr, console.FormatPromptMessage("Would you like to continue?"))

	// Count messages - Use for numeric status
	fmt.Fprintln(os.Stderr, console.FormatCountMessage("Found 25 workflow runs"))
	fmt.Fprintln(os.Stderr, console.FormatCountMessage("Processed 100 of 150 files"))

	// Verbose messages - Use for debugging output
	fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Debug: Token count = 1234"))
	fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Trace: Network latency = 45ms"))

	// List formatting
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Available Workflows:"))
	fmt.Fprintln(os.Stderr, console.FormatListItem("issue-triage.md"))
	fmt.Fprintln(os.Stderr, console.FormatListItem("pr-review.md"))
	fmt.Fprintln(os.Stderr, console.FormatListItem("code-scan.md"))

	// Error with suggestions
	fmt.Fprintln(os.Stderr, "")
	suggestions := []string{
		"Run `gh aw compile` to validate the workflow",
		"Check the workflow frontmatter for syntax errors",
		"Ensure all required fields are present",
		"Review the documentation at https://github.com/githubnext/gh-aw",
	}
	fmt.Fprintln(os.Stderr, console.FormatErrorWithSuggestions(
		"Workflow validation failed",
		suggestions,
	))

	// Note: All output goes to stderr (os.Stderr)
	// JSON or machine-readable output would go to stdout (os.Stdout)
}
