package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var taintLog = logger.New("cli:taint")

// NewTaintCommand creates the taint command
func NewTaintCommand() *cobra.Command {
	taintCmd := &cobra.Command{
		Use:   "taint",
		Short: "Perform taint flow analysis on agentic workflows",
		Long: `Perform taint flow analysis across all GitHub Actions workflows.

This command analyzes data flow from agentic workflows (considered tainted sources)
through the system, identifying paths where tainted data flows to third-party inputs
or other potentially unsafe destinations.

The analysis:
- Treats all agentic workflows as taint sources
- Identifies third-party inputs (web searches, external APIs, GitHub issues)
- Traces data flow through workflow steps and jobs
- Generates a Mermaid graph visualization
- Highlights potentially unsafe paths
- Provides recommendations for safeguards

Examples:
  ` + constants.CLIExtensionPrefix + ` taint                    # Analyze all workflows
  ` + constants.CLIExtensionPrefix + ` taint -o taint-report.md # Save report to file
  ` + constants.CLIExtensionPrefix + ` taint --mermaid-only      # Output only Mermaid graph
  ` + constants.CLIExtensionPrefix + ` taint -v                  # Verbose output`,
		Run: func(cmd *cobra.Command, args []string) {
			outputFile, _ := cmd.Flags().GetString("output")
			mermaidOnly, _ := cmd.Flags().GetBool("mermaid-only")
			verbose, _ := cmd.Flags().GetBool("verbose")
			workflowsDir, _ := cmd.Flags().GetString("workflows-dir")

			if err := RunTaintAnalysis(outputFile, mermaidOnly, verbose, workflowsDir); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	// Add flags to taint command
	taintCmd.Flags().StringP("output", "o", "", "Output file for the taint analysis report (default: stdout)")
	taintCmd.Flags().Bool("mermaid-only", false, "Output only the Mermaid graph without analysis report")
	taintCmd.Flags().String("workflows-dir", "", "Relative directory containing workflows (default: .github/workflows)")

	return taintCmd
}

// RunTaintAnalysis performs taint flow analysis on all workflows
func RunTaintAnalysis(outputFile string, mermaidOnly bool, verbose bool, workflowsDir string) error {
	taintLog.Printf("Starting taint analysis: output=%s, mermaidOnly=%v, verbose=%v", outputFile, mermaidOnly, verbose)

	// Determine workflows directory
	if workflowsDir == "" {
		workflowsDir = filepath.Join(constants.GetWorkflowDir())
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Construct absolute path to workflows directory
	workflowsDirPath := filepath.Join(cwd, workflowsDir)
	taintLog.Printf("Workflows directory: %s", workflowsDirPath)

	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDirPath); os.IsNotExist(err) {
		return fmt.Errorf("workflows directory not found: %s", workflowsDirPath)
	}

	// Load and analyze all workflows
	analysis, err := performTaintAnalysis(workflowsDirPath, verbose)
	if err != nil {
		return fmt.Errorf("taint analysis failed: %w", err)
	}

	// Generate output
	var output string
	if mermaidOnly {
		output = generateMermaidGraph(analysis)
	} else {
		output = generateFullReport(analysis)
	}

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Taint analysis report written to: %s", outputFile)))
	} else {
		fmt.Print(output)
	}

	return nil
}
