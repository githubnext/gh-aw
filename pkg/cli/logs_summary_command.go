// Package cli provides command-line interface functionality for gh-aw.
// This file (logs_summary_command.go) contains the CLI command definition for the logs summary subcommand.
//
// Key responsibilities:
//   - Defining the Cobra command structure for gh aw logs summary
//   - Generating markdown-formatted summaries suitable for GitHub Actions Step Summary
//   - Reusing the existing logs downloading and parsing infrastructure
//   - Outputting GitHub-flavored markdown to stdout
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var logsSummaryCommandLog = logger.New("cli:logs_summary_command")

// NewLogsSummaryCommand creates the logs summary subcommand
func NewLogsSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary [workflow]",
		Short: "Generate markdown summary of workflow logs for GitHub Actions Step Summary",
		Long: `Generate a markdown-formatted summary of workflow execution logs.

This command downloads workflow run logs and generates a markdown report suitable
for GitHub Actions Step Summary ($GITHUB_STEP_SUMMARY). It provides an overview
of workflow executions including metrics, errors, warnings, and firewall analysis.

The output is written to stdout in GitHub-flavored markdown format, making it easy
to pipe to $GITHUB_STEP_SUMMARY for CI/CD reporting.

` + WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` logs summary                    # Summary for all workflows
  ` + string(constants.CLIExtensionPrefix) + ` logs summary weekly-research    # Summary for specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` logs summary -c 5               # Summary of last 5 runs
  ` + string(constants.CLIExtensionPrefix) + ` logs summary --firewall         # Summary for firewall-enabled runs
  ` + string(constants.CLIExtensionPrefix) + ` logs summary >> $GITHUB_STEP_SUMMARY  # Append to step summary

Usage in GitHub Actions:
  - name: Firewall summary
    if: always()
    run: gh aw logs summary --firewall >> $GITHUB_STEP_SUMMARY`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logsSummaryCommandLog.Printf("Starting logs summary command: args=%d", len(args))

			// Parse flags
			workflowName := ""
			if len(args) > 0 {
				workflowName = args[0]
			}

			count, _ := cmd.Flags().GetInt("count")
			startDate, _ := cmd.Flags().GetString("start-date")
			endDate, _ := cmd.Flags().GetString("end-date")
			engine, _ := cmd.Flags().GetString("engine")
			ref, _ := cmd.Flags().GetString("ref")
			beforeRunID, _ := cmd.Flags().GetInt64("before-run-id")
			afterRunID, _ := cmd.Flags().GetInt64("after-run-id")
			firewallOnly, _ := cmd.Flags().GetBool("firewall")
			noFirewall, _ := cmd.Flags().GetBool("no-firewall")
			repoOverride, _ := cmd.Flags().GetString("repo")
			verbose, _ := cmd.Flags().GetBool("verbose")

			logsSummaryCommandLog.Printf("Executing logs summary: workflow=%s, count=%d, engine=%s", workflowName, count, engine)

			return RunLogsSummary(cmd.Context(), LogsSummaryConfig{
				WorkflowName: workflowName,
				Count:        count,
				StartDate:    startDate,
				EndDate:      endDate,
				Engine:       engine,
				Ref:          ref,
				BeforeRunID:  beforeRunID,
				AfterRunID:   afterRunID,
				FirewallOnly: firewallOnly,
				NoFirewall:   noFirewall,
				RepoOverride: repoOverride,
				Verbose:      verbose,
			})
		},
	}

	// Add flags
	cmd.Flags().IntP("count", "c", 10, "Maximum number of matching workflow runs to return")
	cmd.Flags().String("start-date", "", "Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	cmd.Flags().String("end-date", "", "Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	addEngineFilterFlag(cmd)
	cmd.Flags().String("ref", "", "Filter runs by branch or tag name (e.g., main, v1.0.0)")
	cmd.Flags().Int64("before-run-id", 0, "Filter runs with database ID before this value (exclusive)")
	cmd.Flags().Int64("after-run-id", 0, "Filter runs with database ID after this value (exclusive)")
	cmd.Flags().Bool("firewall", false, "Filter to only runs with firewall enabled")
	cmd.Flags().Bool("no-firewall", false, "Filter to only runs without firewall enabled")
	addRepoFlag(cmd)
	cmd.MarkFlagsMutuallyExclusive("firewall", "no-firewall")

	// Register completions
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterEngineFlagCompletion(cmd)

	return cmd
}

// LogsSummaryConfig contains configuration for the logs summary command
type LogsSummaryConfig struct {
	WorkflowName string
	Count        int
	StartDate    string
	EndDate      string
	Engine       string
	Ref          string
	BeforeRunID  int64
	AfterRunID   int64
	FirewallOnly bool
	NoFirewall   bool
	RepoOverride string
	Verbose      bool
}

// RunLogsSummary executes the logs summary command logic
func RunLogsSummary(ctx context.Context, config LogsSummaryConfig) error {
	logsSummaryCommandLog.Printf("Running logs summary with config: %+v", config)

	// Create a temporary directory for logs download
	tmpDir, err := os.MkdirTemp("", "gh-aw-logs-summary-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	logsSummaryCommandLog.Printf("Created temporary directory: %s", tmpDir)

	// Download logs using the existing infrastructure
	// The summary.json file will be written to tmpDir
	err = DownloadWorkflowLogs(
		ctx,
		config.WorkflowName,
		config.Count,
		config.StartDate,
		config.EndDate,
		tmpDir,                 // outputDir
		config.Engine,          // engine
		config.Ref,             // ref
		config.BeforeRunID,     // beforeRunID
		config.AfterRunID,      // afterRunID
		config.RepoOverride,    // repoOverride
		false,                  // verbose - suppress output
		false,                  // toolGraph
		true,                   // noStaged - exclude staged runs from summary
		config.FirewallOnly,    // firewallOnly
		config.NoFirewall,      // noFirewall
		false,                  // parse
		false,                  // jsonOutput
		0,                      // timeout
		false,                  // campaignOnly
		"summary.json",         // summaryFile
	)

	if err != nil {
		return fmt.Errorf("failed to download logs: %w", err)
	}

	// Load the summary.json file
	summaryPath := filepath.Join(tmpDir, "summary.json")
	logsData, err := loadLogsDataFromFile(summaryPath)
	if err != nil {
		return fmt.Errorf("failed to load summary data: %w", err)
	}

	// Generate markdown from logs data
	markdown := generateMarkdownFromLogsData(logsData, config)
	
	// Output to stdout (not stderr) so it can be piped to $GITHUB_STEP_SUMMARY
	fmt.Print(markdown)

	logsSummaryCommandLog.Print("Logs summary completed successfully")
	return nil
}

// loadLogsDataFromFile loads LogsData from a summary.json file
func loadLogsDataFromFile(path string) (LogsData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LogsData{}, fmt.Errorf("failed to read summary file: %w", err)
	}

	var logsData LogsData
	if err := json.Unmarshal(data, &logsData); err != nil {
		return LogsData{}, fmt.Errorf("failed to parse summary JSON: %w", err)
	}

	return logsData, nil
}

// generateMarkdownFromLogsData generates a GitHub-flavored markdown summary from LogsData
func generateMarkdownFromLogsData(logsData LogsData, config LogsSummaryConfig) string {
	var md strings.Builder

	// Header
	md.WriteString("# Workflow Execution Summary\n\n")

	if config.WorkflowName != "" {
		md.WriteString(fmt.Sprintf("**Workflow:** %s\n\n", config.WorkflowName))
	}

	// Filters section
	filters := []string{}
	if config.Engine != "" {
		filters = append(filters, fmt.Sprintf("Engine: `%s`", config.Engine))
	}
	if config.Ref != "" {
		filters = append(filters, fmt.Sprintf("Branch/Tag: `%s`", config.Ref))
	}
	if config.FirewallOnly {
		filters = append(filters, "Firewall: enabled")
	}
	if config.NoFirewall {
		filters = append(filters, "Firewall: disabled")
	}
	if len(filters) > 0 {
		md.WriteString("**Filters:** ")
		md.WriteString(strings.Join(filters, " | "))
		md.WriteString("\n\n")
	}

	// Summary metrics
	md.WriteString("## Summary\n\n")
	md.WriteString("| Metric | Value |\n")
	md.WriteString("|--------|-------|\n")
	md.WriteString(fmt.Sprintf("| Total Runs | %d |\n", logsData.Summary.TotalRuns))
	md.WriteString(fmt.Sprintf("| Total Duration | %s |\n", logsData.Summary.TotalDuration))
	md.WriteString(fmt.Sprintf("| Total Tokens | %d |\n", logsData.Summary.TotalTokens))
	md.WriteString(fmt.Sprintf("| Total Cost | $%.4f |\n", logsData.Summary.TotalCost))
	md.WriteString(fmt.Sprintf("| Total Turns | %d |\n", logsData.Summary.TotalTurns))
	md.WriteString(fmt.Sprintf("| Total Errors | %d |\n", logsData.Summary.TotalErrors))
	md.WriteString(fmt.Sprintf("| Total Warnings | %d |\n", logsData.Summary.TotalWarnings))
	md.WriteString(fmt.Sprintf("| Missing Tools | %d |\n", logsData.Summary.TotalMissingTools))
	md.WriteString("\n")

	// Firewall Analysis (if present)
	if logsData.FirewallLog != nil && logsData.FirewallLog.TotalRequests > 0 {
		md.WriteString("## 🔥 Firewall Analysis\n\n")
		md.WriteString(fmt.Sprintf("**Total Requests:** %d\n\n", logsData.FirewallLog.TotalRequests))
		md.WriteString(fmt.Sprintf("- ✅ Allowed: %d\n", logsData.FirewallLog.AllowedRequests))
		md.WriteString(fmt.Sprintf("- ❌ Denied: %d\n\n", logsData.FirewallLog.DeniedRequests))

		// Top requested domains
		if len(logsData.FirewallLog.RequestsByDomain) > 0 {
			md.WriteString("### Top Domains\n\n")
			md.WriteString("| Domain | Allowed | Denied | Total |\n")
			md.WriteString("|--------|---------|--------|-------|\n")

			// Sort domains by total requests
			type domainStat struct {
				domain  string
				allowed int
				denied  int
				total   int
			}
			var domains []domainStat
			for domain, stats := range logsData.FirewallLog.RequestsByDomain {
				domains = append(domains, domainStat{
					domain:  domain,
					allowed: stats.Allowed,
					denied:  stats.Denied,
					total:   stats.Allowed + stats.Denied,
				})
			}
			// Sort by total descending
			for i := 0; i < len(domains); i++ {
				for j := i + 1; j < len(domains); j++ {
					if domains[j].total > domains[i].total {
						domains[i], domains[j] = domains[j], domains[i]
					}
				}
			}

			// Show top 10
			for i, stat := range domains {
				if i >= 10 {
					break
				}
				md.WriteString(fmt.Sprintf("| %s | %d | %d | %d |\n", 
					stat.domain, stat.allowed, stat.denied, stat.total))
			}
			md.WriteString("\n")
		}
	}

	// Errors and Warnings (if present)
	if len(logsData.ErrorsAndWarnings) > 0 {
		md.WriteString("## ⚠️ Errors and Warnings\n\n")
		
		// Separate errors and warnings
		var errors []ErrorSummary
		var warnings []ErrorSummary
		for _, item := range logsData.ErrorsAndWarnings {
			if item.Type == "Error" {
				errors = append(errors, item)
			} else {
				warnings = append(warnings, item)
			}
		}

		if len(errors) > 0 {
			md.WriteString("### Errors\n\n")
			md.WriteString("| Message | Count | Engine |\n")
			md.WriteString("|---------|-------|--------|\n")
			for _, err := range errors {
				// Truncate message if too long
				msg := err.Message
				if len(msg) > 80 {
					msg = msg[:77] + "..."
				}
				md.WriteString(fmt.Sprintf("| %s | %d | %s |\n", msg, err.Count, err.Engine))
			}
			md.WriteString("\n")
		}

		if len(warnings) > 0 {
			md.WriteString("### Warnings\n\n")
			md.WriteString("| Message | Count | Engine |\n")
			md.WriteString("|---------|-------|--------|\n")
			for _, warn := range warnings {
				// Truncate message if too long
				msg := warn.Message
				if len(msg) > 80 {
					msg = msg[:77] + "..."
				}
				md.WriteString(fmt.Sprintf("| %s | %d | %s |\n", msg, warn.Count, warn.Engine))
			}
			md.WriteString("\n")
		}
	}

	// Missing Tools (if present)
	if len(logsData.MissingTools) > 0 {
		md.WriteString("## 🛠️ Missing Tools\n\n")
		md.WriteString("| Tool | Count | Workflows |\n")
		md.WriteString("|------|-------|----------|\n")
		for _, tool := range logsData.MissingTools {
			md.WriteString(fmt.Sprintf("| %s | %d | %s |\n", 
				tool.Tool, tool.Count, tool.WorkflowsDisplay))
		}
		md.WriteString("\n")
	}

	// MCP Failures (if present)
	if len(logsData.MCPFailures) > 0 {
		md.WriteString("## ⚠️ MCP Server Failures\n\n")
		md.WriteString("| Server | Count | Workflows |\n")
		md.WriteString("|--------|-------|----------|\n")
		for _, failure := range logsData.MCPFailures {
			md.WriteString(fmt.Sprintf("| %s | %d | %s |\n", 
				failure.ServerName, failure.Count, failure.WorkflowsDisplay))
		}
		md.WriteString("\n")
	}

	// Footer
	md.WriteString("---\n")
	md.WriteString(fmt.Sprintf("_Generated by [GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)_\n"))

	return md.String()
}
