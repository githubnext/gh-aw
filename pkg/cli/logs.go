package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/timeutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
)

var logsLog = logger.New("cli:logs")

const (
	// defaultAgentStdioLogPath is the default log file path for agent stdout/stderr
	defaultAgentStdioLogPath = "/tmp/gh-aw/agent-stdio.log"
	// runSummaryFileName is the name of the summary file created in each run folder
	runSummaryFileName = "run_summary.json"
)

// WorkflowRun represents a GitHub Actions workflow run with metrics
type WorkflowRun struct {
	DatabaseID       int64     `json:"databaseId"`
	Number           int       `json:"number"`
	URL              string    `json:"url"`
	Status           string    `json:"status"`
	Conclusion       string    `json:"conclusion"`
	WorkflowName     string    `json:"workflowName"`
	WorkflowPath     string    `json:"workflowPath,omitempty"` // Workflow file path (e.g., .github/workflows/copilot-swe-agent.yml)
	CreatedAt        time.Time `json:"createdAt"`
	StartedAt        time.Time `json:"startedAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Event            string    `json:"event"`
	HeadBranch       string    `json:"headBranch"`
	HeadSha          string    `json:"headSha"`
	DisplayTitle     string    `json:"displayTitle"`
	Duration         time.Duration
	TokenUsage       int
	EstimatedCost    float64
	Turns            int
	ErrorCount       int
	WarningCount     int
	MissingToolCount int
	LogsPath         string
}

// LogMetrics represents extracted metrics from log files
// This is now an alias to the shared type in workflow package
type LogMetrics = workflow.LogMetrics

// ProcessedRun represents a workflow run with its associated analysis
type ProcessedRun struct {
	Run              WorkflowRun
	AccessAnalysis   *DomainAnalysis
	FirewallAnalysis *FirewallAnalysis
	MissingTools     []MissingToolReport
	MCPFailures      []MCPFailureReport
	JobDetails       []JobInfoWithDuration
}

// MissingToolReport represents a missing tool reported by an agentic workflow
type MissingToolReport struct {
	Tool         string `json:"tool"`
	Reason       string `json:"reason"`
	Alternatives string `json:"alternatives,omitempty"`
	Timestamp    string `json:"timestamp"`
	WorkflowName string `json:"workflow_name,omitempty"` // Added for tracking which workflow reported this
	RunID        int64  `json:"run_id,omitempty"`        // Added for tracking which run reported this
}

// MCPFailureReport represents an MCP server failure detected in a workflow run
type MCPFailureReport struct {
	ServerName   string `json:"server_name"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp,omitempty"`
	WorkflowName string `json:"workflow_name,omitempty"`
	RunID        int64  `json:"run_id,omitempty"`
}

// MissingToolSummary aggregates missing tool reports across runs
type MissingToolSummary struct {
	Tool               string   `json:"tool" console:"header:Tool"`
	Count              int      `json:"count" console:"header:Occurrences"`
	Workflows          []string `json:"workflows" console:"-"`                     // List of workflow names that reported this tool
	WorkflowsDisplay   string   `json:"-" console:"header:Workflows,maxlen:40"`    // Formatted display of workflows
	FirstReason        string   `json:"first_reason" console:"-"`                  // Reason from the first occurrence
	FirstReasonDisplay string   `json:"-" console:"header:First Reason,maxlen:50"` // Formatted display of first reason
	RunIDs             []int64  `json:"run_ids" console:"-"`                       // List of run IDs where this tool was reported
}

// ErrNoArtifacts indicates that a workflow run has no artifacts
var ErrNoArtifacts = errors.New("no artifacts found for this run")

// RunSummary represents a complete summary of a workflow run's artifacts and metrics.
// This file is written to each run folder as "run_summary.json" to cache processing results
// and avoid re-downloading and re-processing already analyzed runs.
//
// Key features:
// - Acts as a marker that a run has been fully processed
// - Stores all extracted metrics and analysis results
// - Includes CLI version for cache invalidation when the tool is updated
// - Enables fast reloading of run data without re-parsing logs
//
// Cache invalidation:
// - If the CLI version in the summary doesn't match the current version, the run is reprocessed
// - This ensures that bug fixes and improvements in log parsing are automatically applied
type RunSummary struct {
	CLIVersion       string                `json:"cli_version"`       // CLI version used to process this run
	RunID            int64                 `json:"run_id"`            // Workflow run database ID
	ProcessedAt      time.Time             `json:"processed_at"`      // When this summary was created
	Run              WorkflowRun           `json:"run"`               // Full workflow run metadata
	Metrics          LogMetrics            `json:"metrics"`           // Extracted log metrics
	AccessAnalysis   *DomainAnalysis       `json:"access_analysis"`   // Network access analysis
	FirewallAnalysis *FirewallAnalysis     `json:"firewall_analysis"` // Firewall log analysis
	MissingTools     []MissingToolReport   `json:"missing_tools"`     // Missing tool reports
	MCPFailures      []MCPFailureReport    `json:"mcp_failures"`      // MCP server failures
	ArtifactsList    []string              `json:"artifacts_list"`    // List of downloaded artifact files
	JobDetails       []JobInfoWithDuration `json:"job_details"`       // Job execution details
}

// fetchJobStatuses gets job information for a workflow run and counts failed jobs
func fetchJobStatuses(runID int64, verbose bool) (int, error) {
	logsLog.Printf("Fetching job statuses: runID=%d", runID)
	args := []string{"api", fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/jobs", runID), "--jq", ".jobs[] | {name: .name, status: .status, conclusion: .conclusion}"}

	if verbose {
		fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Fetching job statuses for run %d", runID)))
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Failed to fetch job statuses for run %d: %v", runID, err)))
		}
		// Don't fail the entire operation if we can't get job info
		return 0, nil
	}

	// Parse each line as a separate JSON object
	failedJobs := 0
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var job JobInfo
		if err := json.Unmarshal([]byte(line), &job); err != nil {
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Failed to parse job info: %s", line)))
			}
			continue
		}

		// Count jobs with failure conclusions as errors
		if isFailureConclusion(job.Conclusion) {
			failedJobs++
			logsLog.Printf("Found failed job: name=%s, conclusion=%s", job.Name, job.Conclusion)
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Found failed job '%s' with conclusion '%s'", job.Name, job.Conclusion)))
			}
		}
	}

	logsLog.Printf("Job status check complete: failedJobs=%d", failedJobs)
	return failedJobs, nil
}

// fetchJobDetails gets detailed job information including durations for a workflow run
func fetchJobDetails(runID int64, verbose bool) ([]JobInfoWithDuration, error) {
	args := []string{"api", fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/jobs", runID), "--jq", ".jobs[] | {name: .name, status: .status, conclusion: .conclusion, started_at: .started_at, completed_at: .completed_at}"}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching job details for run %d", runID)))
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to fetch job details for run %d: %v", runID, err)))
		}
		// Don't fail the entire operation if we can't get job info
		return nil, nil
	}

	var jobs []JobInfoWithDuration
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var job JobInfo
		if err := json.Unmarshal([]byte(line), &job); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to parse job info: %s", line)))
			}
			continue
		}

		jobWithDuration := JobInfoWithDuration{
			JobInfo: job,
		}

		// Calculate duration if both timestamps are available
		if !job.StartedAt.IsZero() && !job.CompletedAt.IsZero() {
			jobWithDuration.Duration = job.CompletedAt.Sub(job.StartedAt)
		}

		jobs = append(jobs, jobWithDuration)
	}

	return jobs, nil
}

// DownloadResult represents the result of downloading artifacts for a single run
type DownloadResult struct {
	Run              WorkflowRun
	Metrics          LogMetrics
	AccessAnalysis   *DomainAnalysis
	FirewallAnalysis *FirewallAnalysis
	MissingTools     []MissingToolReport
	MCPFailures      []MCPFailureReport
	Error            error
	Skipped          bool
	LogsPath         string
}

// JobInfo represents basic information about a workflow job
type JobInfo struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// isFailureConclusion returns true if the conclusion represents a failure state
// (timed_out, failure, or cancelled) that should be counted as an error
func isFailureConclusion(conclusion string) bool {
	return conclusion == "timed_out" || conclusion == "failure" || conclusion == "cancelled"
}

// JobInfoWithDuration extends JobInfo with calculated duration
type JobInfoWithDuration struct {
	JobInfo
	Duration time.Duration
}

// AwInfo represents the structure of aw_info.json files
// AwInfoSteps represents the steps information in aw_info.json files
type AwInfoSteps struct {
	Firewall string `json:"firewall,omitempty"` // Firewall type (e.g., "squid") or empty if no firewall
}

type AwInfo struct {
	EngineID     string      `json:"engine_id"`
	EngineName   string      `json:"engine_name"`
	Model        string      `json:"model"`
	Version      string      `json:"version"`
	WorkflowName string      `json:"workflow_name"`
	Staged       bool        `json:"staged"`
	Steps        AwInfoSteps `json:"steps,omitempty"` // Steps metadata
	CreatedAt    string      `json:"created_at"`
	// Additional fields that might be present
	RunID      any    `json:"run_id,omitempty"`
	RunNumber  any    `json:"run_number,omitempty"`
	Repository string `json:"repository,omitempty"`
}

// Constants for the iterative algorithm
const (
	// MaxIterations limits how many batches we fetch to prevent infinite loops
	MaxIterations = 20
	// BatchSize is the number of runs to fetch in each iteration
	BatchSize = 100
	// BatchSizeForAllWorkflows is the larger batch size when searching for agentic workflows
	// There can be a really large number of workflow runs in a repository, so
	// we are generous in the batch size when used without qualification.
	BatchSizeForAllWorkflows = 250
	// MaxConcurrentDownloads limits the number of parallel artifact downloads
	MaxConcurrentDownloads = 10
)

// NewLogsCommand creates the logs command
func NewLogsCommand() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs [workflow-id]",
		Short: "Download and analyze agentic workflow logs with aggregated metrics",
		Long: `Download workflow run logs and artifacts from GitHub Actions for agentic workflows.

This command fetches workflow runs, downloads their artifacts, and extracts them into
organized folders named by run ID. It also provides an overview table with aggregate
metrics including duration, token usage, and cost information.

Downloaded artifacts include:
- aw_info.json: Engine configuration and workflow metadata
- safe_output.jsonl: Agent's final output content (available when non-empty)
- agent_output/: Agent logs directory (if the workflow produced logs)
- agent-stdio.log: Agent standard output/error logs
- aw.patch: Git patch of changes made during execution
- workflow-logs/: GitHub Actions workflow run logs (job logs organized in subdirectory)

The workflow-id is the basename of the markdown file without the .md extension.
For example, for 'weekly-research.md', use 'weekly-research' as the workflow ID.

Examples:
  ` + constants.CLIExtensionPrefix + ` logs                           # Download logs for all workflows
  ` + constants.CLIExtensionPrefix + ` logs weekly-research           # Download logs for specific workflow
  ` + constants.CLIExtensionPrefix + ` logs -c 10                     # Download last 10 matching runs
  ` + constants.CLIExtensionPrefix + ` logs --start-date 2024-01-01   # Download all runs after date
  ` + constants.CLIExtensionPrefix + ` logs --end-date 2024-01-31     # Download all runs before date
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1w          # Download all runs from last week
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1w -c 5     # Download all runs from last week, show up to 5
  ` + constants.CLIExtensionPrefix + ` logs --end-date -1d            # Download all runs until yesterday
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1mo         # Download all runs from last month
  ` + constants.CLIExtensionPrefix + ` logs --engine claude           # Filter logs by claude engine
  ` + constants.CLIExtensionPrefix + ` logs --engine codex            # Filter logs by codex engine
  ` + constants.CLIExtensionPrefix + ` logs --engine copilot          # Filter logs by copilot engine
  ` + constants.CLIExtensionPrefix + ` logs --firewall                # Filter logs with firewall enabled
  ` + constants.CLIExtensionPrefix + ` logs --no-firewall             # Filter logs without firewall
  ` + constants.CLIExtensionPrefix + ` logs -o ./my-logs              # Custom output directory
  ` + constants.CLIExtensionPrefix + ` logs --branch main             # Filter logs by branch name
  ` + constants.CLIExtensionPrefix + ` logs --branch feature-xyz      # Filter logs by feature branch
  ` + constants.CLIExtensionPrefix + ` logs --after-run-id 1000       # Filter runs after run ID 1000
  ` + constants.CLIExtensionPrefix + ` logs --before-run-id 2000      # Filter runs before run ID 2000
  ` + constants.CLIExtensionPrefix + ` logs --after-run-id 1000 --before-run-id 2000  # Filter runs in range
  ` + constants.CLIExtensionPrefix + ` logs --tool-graph              # Generate Mermaid tool sequence graph
  ` + constants.CLIExtensionPrefix + ` logs --parse                   # Parse logs and generate markdown reports
  ` + constants.CLIExtensionPrefix + ` logs --json                    # Output metrics in JSON format
  ` + constants.CLIExtensionPrefix + ` logs --parse --json            # Generate both markdown and JSON`,
		Run: func(cmd *cobra.Command, args []string) {
			var workflowName string
			if len(args) > 0 && args[0] != "" {
				// Convert workflow ID to GitHub Actions workflow name
				// First try to resolve as a workflow ID
				resolvedName, err := workflow.ResolveWorkflowName(args[0])
				if err != nil {
					// If that fails, check if it's already a GitHub Actions workflow name
					// by checking if any .lock.yml files have this as their name
					agenticWorkflowNames, nameErr := getAgenticWorkflowNames(false)
					if nameErr == nil && contains(agenticWorkflowNames, args[0]) {
						// It's already a valid GitHub Actions workflow name
						workflowName = args[0]
					} else {
						// Neither workflow ID nor valid GitHub Actions workflow name
						suggestions := []string{
							fmt.Sprintf("Run '%s status' to see all available workflows", constants.CLIExtensionPrefix),
							"Check for typos in the workflow name",
							"Use the workflow ID (e.g., 'test-claude') or GitHub Actions workflow name (e.g., 'Test Claude')",
						}
						fmt.Fprintln(os.Stderr, console.FormatErrorWithSuggestions(
							fmt.Sprintf("workflow '%s' not found", args[0]),
							suggestions,
						))
						os.Exit(1)
					}
				} else {
					workflowName = resolvedName
				}
			}

			count, _ := cmd.Flags().GetInt("count")
			startDate, _ := cmd.Flags().GetString("start-date")
			endDate, _ := cmd.Flags().GetString("end-date")
			outputDir, _ := cmd.Flags().GetString("output")
			engine, _ := cmd.Flags().GetString("engine")
			branch, _ := cmd.Flags().GetString("branch")
			beforeRunID, _ := cmd.Flags().GetInt64("before-run-id")
			afterRunID, _ := cmd.Flags().GetInt64("after-run-id")
			verbose, _ := cmd.Flags().GetBool("verbose")
			toolGraph, _ := cmd.Flags().GetBool("tool-graph")
			noStaged, _ := cmd.Flags().GetBool("no-staged")
			firewallOnly, _ := cmd.Flags().GetBool("firewall")
			noFirewall, _ := cmd.Flags().GetBool("no-firewall")
			parse, _ := cmd.Flags().GetBool("parse")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			timeout, _ := cmd.Flags().GetInt("timeout")

			// Resolve relative dates to absolute dates for GitHub CLI
			now := time.Now()
			if startDate != "" {
				resolvedStartDate, err := workflow.ResolveRelativeDate(startDate, now)
				if err != nil {
					fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
						Type:    "error",
						Message: fmt.Sprintf("invalid start-date format '%s': %v", startDate, err),
					}))
					os.Exit(1)
				}
				startDate = resolvedStartDate
			}
			if endDate != "" {
				resolvedEndDate, err := workflow.ResolveRelativeDate(endDate, now)
				if err != nil {
					fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
						Type:    "error",
						Message: fmt.Sprintf("invalid end-date format '%s': %v", endDate, err),
					}))
					os.Exit(1)
				}
				endDate = resolvedEndDate
			}

			// Validate engine parameter using the engine registry
			if engine != "" {
				registry := workflow.GetGlobalEngineRegistry()
				if !registry.IsValidEngine(engine) {
					supportedEngines := registry.GetSupportedEngines()
					fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
						Type:    "error",
						Message: fmt.Sprintf("invalid engine value '%s'. Must be one of: %s", engine, strings.Join(supportedEngines, ", ")),
					}))
					os.Exit(1)
				}
			}

			// Validate firewall parameters
			if firewallOnly && noFirewall {
				fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
					Type:    "error",
					Message: "cannot specify both --firewall and --no-firewall flags",
				}))
				os.Exit(1)
			}

			if err := DownloadWorkflowLogs(workflowName, count, startDate, endDate, outputDir, engine, branch, beforeRunID, afterRunID, verbose, toolGraph, noStaged, firewallOnly, noFirewall, parse, jsonOutput, timeout); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
					Type:    "error",
					Message: err.Error(),
				}))
				os.Exit(1)
			}
		},
	}

	// Add flags to logs command
	logsCmd.Flags().IntP("count", "c", 100, "Maximum number of matching workflow runs to return (after applying filters)")
	logsCmd.Flags().String("start-date", "", "Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	logsCmd.Flags().String("end-date", "", "Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	logsCmd.Flags().StringP("output", "o", "./logs", "Output directory for downloaded logs and artifacts")
	logsCmd.Flags().String("engine", "", "Filter logs by agentic engine type (claude, codex, copilot)")
	logsCmd.Flags().String("branch", "", "Filter runs by branch name (e.g., main, feature-branch)")
	logsCmd.Flags().Int64("before-run-id", 0, "Filter runs with database ID before this value (exclusive)")
	logsCmd.Flags().Int64("after-run-id", 0, "Filter runs with database ID after this value (exclusive)")
	logsCmd.Flags().Bool("tool-graph", false, "Generate Mermaid tool sequence graph from agent logs")
	logsCmd.Flags().Bool("no-staged", false, "Filter out staged workflow runs (exclude runs with staged: true in aw_info.json)")
	logsCmd.Flags().Bool("firewall", false, "Filter to only runs with firewall enabled")
	logsCmd.Flags().Bool("no-firewall", false, "Filter to only runs without firewall enabled")
	logsCmd.Flags().Bool("parse", false, "Run JavaScript parsers on agent logs and firewall logs, writing markdown to log.md and firewall.md")
	logsCmd.Flags().Bool("json", false, "Output logs data as JSON instead of formatted console tables")
	logsCmd.Flags().Int("timeout", 0, "Maximum time in seconds to spend downloading logs (0 = no timeout)")

	return logsCmd
}

// DownloadWorkflowLogs downloads and analyzes workflow logs with metrics
func DownloadWorkflowLogs(workflowName string, count int, startDate, endDate, outputDir, engine, branch string, beforeRunID, afterRunID int64, verbose bool, toolGraph bool, noStaged bool, firewallOnly bool, noFirewall bool, parse bool, jsonOutput bool, timeout int) error {
	logsLog.Printf("Starting workflow log download: workflow=%s, count=%d, startDate=%s, endDate=%s, outputDir=%s", workflowName, count, startDate, endDate, outputDir)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}

	// Start timeout timer if specified
	var startTime time.Time
	var timeoutReached bool
	if timeout > 0 {
		startTime = time.Now()
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout set to %d seconds", timeout)))
		}
	}

	var processedRuns []ProcessedRun
	var beforeDate string
	iteration := 0

	// Determine if we should fetch all runs (when date filters are specified) or limit by count
	// When date filters are specified, we fetch all runs within that range and apply count to final output
	// When no date filters, we fetch up to 'count' runs with artifacts (old behavior for backward compatibility)
	fetchAllInRange := startDate != "" || endDate != ""

	// Iterative algorithm: keep fetching runs until we have enough or exhaust available runs
	for iteration < MaxIterations {
		// Check timeout if specified
		if timeout > 0 {
			elapsed := time.Since(startTime).Seconds()
			if elapsed >= float64(timeout) {
				timeoutReached = true
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Timeout reached after %.1f seconds, stopping download", elapsed)))
				}
				break
			}
		}

		// Stop if we've collected enough processed runs
		if len(processedRuns) >= count {
			break
		}

		iteration++

		if verbose && iteration > 1 {
			if fetchAllInRange {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Fetching more runs in date range...", iteration)))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Need %d more runs with artifacts, fetching more...", iteration, count-len(processedRuns))))
			}
		}

		// Fetch a batch of runs
		batchSize := BatchSize
		if workflowName == "" {
			// When searching for all agentic workflows, use a larger batch size
			// since there may be many CI runs interspersed with agentic runs
			batchSize = BatchSizeForAllWorkflows
		}

		// When not fetching all in range, optimize batch size based on how many we still need
		if !fetchAllInRange && count-len(processedRuns) < batchSize {
			// If we need fewer runs than the batch size, request exactly what we need
			// but add some buffer since many runs might not have artifacts
			needed := count - len(processedRuns)
			batchSize = needed * 3 // Request 3x what we need to account for runs without artifacts
			if workflowName == "" && batchSize < BatchSizeForAllWorkflows {
				// For all-workflows search, maintain a minimum batch size
				batchSize = BatchSizeForAllWorkflows
			}
			if batchSize > BatchSizeForAllWorkflows {
				batchSize = BatchSizeForAllWorkflows
			}
		}

		runs, totalFetched, err := listWorkflowRunsWithPagination(workflowName, batchSize, startDate, endDate, beforeDate, branch, beforeRunID, afterRunID, len(processedRuns), count, verbose)
		if err != nil {
			return err
		}

		if len(runs) == 0 {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No more workflow runs found, stopping iteration"))
			}
			break
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow runs in batch %d", len(runs), iteration)))
		}

		// Process each run in this batch
		batchProcessed := 0
		// Always limit downloads to remaining count needed to avoid downloading extras
		maxDownloads := count - len(processedRuns)
		if maxDownloads <= 0 {
			// Already have enough processed runs, stop downloading
			break
		}
		downloadResults := downloadRunArtifactsConcurrent(runs, outputDir, verbose, maxDownloads)

		for _, result := range downloadResults {
			if result.Skipped {
				if verbose {
					if result.Error != nil {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", result.Run.DatabaseID, result.Error)))
					}
				}
				continue
			}

			if result.Error != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", result.Run.DatabaseID, result.Error)))
				continue
			}

			// Parse aw_info.json once for all filters that need it (optimization)
			var awInfo *AwInfo
			var awInfoErr error
			awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")

			// Only parse if we need it for any filter
			if engine != "" || noStaged || firewallOnly || noFirewall {
				awInfo, awInfoErr = parseAwInfo(awInfoPath, verbose)
			}

			// Apply engine filtering if specified
			if engine != "" {
				// Check if the run's engine matches the filter
				detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)

				var engineMatches bool
				if detectedEngine != nil {
					// Get the engine ID to compare with the filter
					registry := workflow.GetGlobalEngineRegistry()
					for _, supportedEngine := range constants.AgenticEngines {
						if testEngine, err := registry.GetEngine(supportedEngine); err == nil && testEngine == detectedEngine {
							engineMatches = (supportedEngine == engine)
							break
						}
					}
				}

				if !engineMatches {
					if verbose {
						engineName := "unknown"
						if detectedEngine != nil {
							// Try to get a readable name for the detected engine
							registry := workflow.GetGlobalEngineRegistry()
							for _, supportedEngine := range constants.AgenticEngines {
								if testEngine, err := registry.GetEngine(supportedEngine); err == nil && testEngine == detectedEngine {
									engineName = supportedEngine
									break
								}
							}
						}
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: engine '%s' does not match filter '%s'", result.Run.DatabaseID, engineName, engine)))
					}
					continue
				}
			}

			// Apply staged filtering if --no-staged flag is specified
			if noStaged {
				var isStaged bool
				if awInfoErr == nil && awInfo != nil {
					isStaged = awInfo.Staged
				}

				if isStaged {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow is staged (filtered out by --no-staged)", result.Run.DatabaseID)))
					}
					continue
				}
			}

			// Apply firewall filtering if --firewall or --no-firewall flag is specified
			if firewallOnly || noFirewall {
				var hasFirewall bool
				if awInfoErr == nil && awInfo != nil {
					// Firewall is enabled if steps.firewall is non-empty (e.g., "squid")
					hasFirewall = awInfo.Steps.Firewall != ""
				}

				// Check if the run matches the filter
				if firewallOnly && !hasFirewall {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow does not use firewall (filtered by --firewall)", result.Run.DatabaseID)))
					}
					continue
				}
				if noFirewall && hasFirewall {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow uses firewall (filtered by --no-firewall)", result.Run.DatabaseID)))
					}
					continue
				}
			}

			// Update run with metrics and path
			run := result.Run
			run.TokenUsage = result.Metrics.TokenUsage
			run.EstimatedCost = result.Metrics.EstimatedCost
			run.Turns = result.Metrics.Turns
			run.ErrorCount = workflow.CountErrors(result.Metrics.Errors)
			run.WarningCount = workflow.CountWarnings(result.Metrics.Errors)
			run.LogsPath = result.LogsPath

			// Add failed jobs to error count
			if failedJobCount, err := fetchJobStatuses(run.DatabaseID, verbose); err == nil {
				run.ErrorCount += failedJobCount
				if verbose && failedJobCount > 0 {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Added %d failed jobs to error count for run %d", failedJobCount, run.DatabaseID)))
				}
			}

			// Store access analysis for later display (we'll access it via the result)
			// No need to modify the WorkflowRun struct for this

			// Always use GitHub API timestamps for duration calculation
			if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
				run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			}

			processedRun := ProcessedRun{
				Run:              run,
				AccessAnalysis:   result.AccessAnalysis,
				FirewallAnalysis: result.FirewallAnalysis,
				MissingTools:     result.MissingTools,
				MCPFailures:      result.MCPFailures,
			}
			processedRuns = append(processedRuns, processedRun)
			batchProcessed++

			// If --parse flag is set, parse the agent log and write to log.md
			if parse {
				// Get the engine from aw_info.json
				awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
				detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)

				if err := parseAgentLog(result.LogsPath, detectedEngine, verbose); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse log for run %d: %v", run.DatabaseID, err)))
				} else {
					// Always show success message for parsing, not just in verbose mode
					logMdPath := filepath.Join(result.LogsPath, "log.md")
					if _, err := os.Stat(logMdPath); err == nil {
						fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", run.DatabaseID, logMdPath)))
					}
				}

				// Also parse firewall logs if they exist
				if err := parseFirewallLogs(result.LogsPath, verbose); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", run.DatabaseID, err)))
				} else {
					// Show success message if firewall.md was created
					firewallMdPath := filepath.Join(result.LogsPath, "firewall.md")
					if _, err := os.Stat(firewallMdPath); err == nil {
						fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", run.DatabaseID, firewallMdPath)))
					}
				}
			}
		}

		if verbose {
			if fetchAllInRange {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d)", batchProcessed, iteration, len(processedRuns))))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d/%d)", batchProcessed, iteration, len(processedRuns), count)))
			}
		}

		// Prepare for next iteration: set beforeDate to the oldest run from this batch
		if len(runs) > 0 {
			oldestRun := runs[len(runs)-1] // runs are typically ordered by creation date descending
			beforeDate = oldestRun.CreatedAt.Format(time.RFC3339)
		}

		// If we got fewer runs than requested in this batch, we've likely hit the end
		// IMPORTANT: Use totalFetched (API response size before filtering) not len(runs) (after filtering)
		// to detect end. When workflowName is empty, runs are filtered to only agentic workflows,
		// so len(runs) may be much smaller than totalFetched even when more data is available from GitHub.
		// Example: API returns 250 total runs, but only 5 are agentic workflows after filtering.
		//   Old buggy logic: len(runs)=5 < batchSize=250, stop iteration (WRONG - misses more agentic workflows!)
		//   Fixed logic: totalFetched=250 < batchSize=250 is false, continue iteration (CORRECT)
		if totalFetched < batchSize {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Received fewer runs than requested, likely reached end of available runs"))
			}
			break
		}
	}

	// Check if we hit the maximum iterations limit
	if iteration >= MaxIterations {
		if fetchAllInRange {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts", MaxIterations, len(processedRuns))))
		} else if len(processedRuns) < count {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts out of %d requested", MaxIterations, len(processedRuns), count)))
		}
	}

	// Report if timeout was reached
	if timeoutReached && len(processedRuns) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout reached, returning %d processed runs", len(processedRuns))))
	}

	if len(processedRuns) == 0 {
		if timeoutReached {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Timeout reached before any runs could be downloaded"))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow runs with artifacts found matching the specified criteria"))
		}
		return nil
	}

	// Apply count limit to final results (truncate to count if we fetched more)
	if len(processedRuns) > count {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Limiting output to %d most recent runs (fetched %d total)", count, len(processedRuns))))
		}
		processedRuns = processedRuns[:count]
	}

	// Update MissingToolCount in runs
	for i := range processedRuns {
		processedRuns[i].Run.MissingToolCount = len(processedRuns[i].MissingTools)
	}

	// Build continuation data if timeout was reached and there are processed runs
	var continuation *ContinuationData
	if timeoutReached && len(processedRuns) > 0 {
		// Get the oldest run ID from processed runs to use as before_run_id for continuation
		oldestRunID := processedRuns[len(processedRuns)-1].Run.DatabaseID

		continuation = &ContinuationData{
			Message:      "Timeout reached. Use these parameters to continue fetching more logs.",
			WorkflowName: workflowName,
			Count:        count,
			StartDate:    startDate,
			EndDate:      endDate,
			Engine:       engine,
			Branch:       branch,
			AfterRunID:   afterRunID,
			BeforeRunID:  oldestRunID, // Continue from where we left off
			Timeout:      timeout,
		}
	}

	// Build structured logs data
	logsData := buildLogsData(processedRuns, outputDir, continuation)

	// Render output based on format preference
	if jsonOutput {
		if err := renderLogsJSON(logsData); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	} else {
		renderLogsConsole(logsData)

		// Generate tool sequence graph if requested (console output only)
		if toolGraph {
			generateToolGraph(processedRuns, verbose)
		}
	}

	return nil
}

// downloadRunArtifactsConcurrent downloads artifacts for multiple workflow runs concurrently
func downloadRunArtifactsConcurrent(runs []WorkflowRun, outputDir string, verbose bool, maxRuns int) []DownloadResult {
	if len(runs) == 0 {
		return []DownloadResult{}
	}

	// Limit the number of runs to process if maxRuns is specified
	actualRuns := runs
	if maxRuns > 0 && len(runs) > maxRuns {
		logsLog.Printf("Limiting concurrent downloads: maxRuns=%d, totalRuns=%d", maxRuns, len(runs))
		actualRuns = runs[:maxRuns]
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processing %d runs in parallel...", len(actualRuns))))
	}

	// Use conc pool for controlled concurrency with results
	p := pool.NewWithResults[DownloadResult]().WithMaxGoroutines(MaxConcurrentDownloads)

	// Process each run concurrently
	for _, run := range actualRuns {
		run := run // capture loop variable
		p.Go(func() DownloadResult {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processing run %d (%s)...", run.DatabaseID, run.Status)))
			}

			// Download artifacts and logs for this run
			runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", run.DatabaseID))

			// Try to load cached summary first
			if summary, ok := loadRunSummary(runOutputDir, verbose); ok {
				// Valid cached summary exists, use it directly
				result := DownloadResult{
					Run:              summary.Run,
					Metrics:          summary.Metrics,
					AccessAnalysis:   summary.AccessAnalysis,
					FirewallAnalysis: summary.FirewallAnalysis,
					MissingTools:     summary.MissingTools,
					MCPFailures:      summary.MCPFailures,
					LogsPath:         runOutputDir,
				}
				return result
			}

			// No cached summary or version mismatch - download and process
			err := downloadRunArtifacts(run.DatabaseID, runOutputDir, verbose)

			result := DownloadResult{
				Run:      run,
				LogsPath: runOutputDir,
			}

			if err != nil {
				// Check if this is a "no artifacts" case
				if errors.Is(err, ErrNoArtifacts) {
					// For runs with important conclusions (timed_out, failure, cancelled),
					// still process them even without artifacts to show the failure in reports
					if isFailureConclusion(run.Conclusion) {
						// Don't skip - we want these to appear in the report
						// Just use empty metrics
						result.Metrics = LogMetrics{}

						// Try to fetch job details to get error count
						if failedJobCount, jobErr := fetchJobStatuses(run.DatabaseID, verbose); jobErr == nil {
							run.ErrorCount = failedJobCount
						}
					} else {
						// For other runs (success, neutral, etc.) without artifacts, skip them
						result.Skipped = true
						result.Error = err
					}
				} else {
					result.Error = err
				}
			} else {
				// Extract metrics from logs
				metrics, metricsErr := extractLogMetrics(runOutputDir, verbose)
				if metricsErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract metrics for run %d: %v", run.DatabaseID, metricsErr)))
					}
					// Don't fail the whole download for metrics errors
					metrics = LogMetrics{}
				}
				result.Metrics = metrics

				// Analyze access logs if available
				accessAnalysis, accessErr := analyzeAccessLogs(runOutputDir, verbose)
				if accessErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze access logs for run %d: %v", run.DatabaseID, accessErr)))
					}
				}
				result.AccessAnalysis = accessAnalysis

				// Analyze firewall logs if available
				firewallAnalysis, firewallErr := analyzeFirewallLogs(runOutputDir, verbose)
				if firewallErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall logs for run %d: %v", run.DatabaseID, firewallErr)))
					}
				}
				result.FirewallAnalysis = firewallAnalysis

				// Extract missing tools if available
				missingTools, missingErr := extractMissingToolsFromRun(runOutputDir, run, verbose)
				if missingErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing tools for run %d: %v", run.DatabaseID, missingErr)))
					}
				}
				result.MissingTools = missingTools

				// Extract MCP failures if available
				mcpFailures, mcpErr := extractMCPFailuresFromRun(runOutputDir, run, verbose)
				if mcpErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract MCP failures for run %d: %v", run.DatabaseID, mcpErr)))
					}
				}
				result.MCPFailures = mcpFailures

				// Fetch job details for the summary
				jobDetails, jobErr := fetchJobDetails(run.DatabaseID, verbose)
				if jobErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch job details for run %d: %v", run.DatabaseID, jobErr)))
					}
				}

				// List all artifacts
				artifacts, listErr := listArtifacts(runOutputDir)
				if listErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to list artifacts for run %d: %v", run.DatabaseID, listErr)))
					}
				}

				// Create and save run summary
				summary := &RunSummary{
					CLIVersion:       GetVersion(),
					RunID:            run.DatabaseID,
					ProcessedAt:      time.Now(),
					Run:              run,
					Metrics:          metrics,
					AccessAnalysis:   accessAnalysis,
					FirewallAnalysis: firewallAnalysis,
					MissingTools:     missingTools,
					MCPFailures:      mcpFailures,
					ArtifactsList:    artifacts,
					JobDetails:       jobDetails,
				}

				if saveErr := saveRunSummary(runOutputDir, summary, verbose); saveErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save run summary for run %d: %v", run.DatabaseID, saveErr)))
					}
				}
			}

			return result
		})
	}

	// Wait for all downloads to complete and collect results
	results := p.Wait()

	if verbose {
		successCount := 0
		for _, result := range results {
			if result.Error == nil && !result.Skipped {
				successCount++
			}
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Completed parallel processing: %d successful, %d total", successCount, len(results))))
	}

	return results
}

// listWorkflowRunsWithPagination fetches workflow runs from GitHub with pagination support
// Returns:
//   - []WorkflowRun: filtered workflow runs (agentic workflows only when workflowName is empty)
//   - int: total count of runs fetched from GitHub API before filtering
//   - error: any error that occurred
//
// The totalFetched count is critical for pagination - it indicates whether more data is available
// from GitHub, whereas the filtered runs count may be much smaller after filtering for agentic workflows.
//
// The limit parameter specifies the batch size for the GitHub API call (how many runs to fetch in this request),
// not the total number of matching runs the user wants to find.
//
// The processedCount and targetCount parameters are used to display progress in the spinner message.
func listWorkflowRunsWithPagination(workflowName string, limit int, startDate, endDate, beforeDate, branch string, beforeRunID, afterRunID int64, processedCount, targetCount int, verbose bool) ([]WorkflowRun, int, error) {
	args := []string{"run", "list", "--json", "databaseId,number,url,status,conclusion,workflowName,createdAt,startedAt,updatedAt,event,headBranch,headSha,displayTitle"}

	// Add filters
	if workflowName != "" {
		args = append(args, "--workflow", workflowName)
	}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	if startDate != "" {
		args = append(args, "--created", ">="+startDate)
	}
	if endDate != "" {
		args = append(args, "--created", "<="+endDate)
	}
	// Add beforeDate filter for pagination
	if beforeDate != "" {
		args = append(args, "--created", "<"+beforeDate)
	}
	// Add branch filter
	if branch != "" {
		args = append(args, "--branch", branch)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Executing: gh %s", strings.Join(args, " "))))
	}

	// Start spinner for network operation
	spinnerMsg := fmt.Sprintf("Fetching workflow runs from GitHub... (%d / %d)", processedCount, targetCount)
	spinner := console.NewSpinner(spinnerMsg)
	if !verbose {
		spinner.Start()
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()

	// Stop spinner
	if !verbose {
		spinner.Stop()
	}
	if err != nil {
		// Check for authentication errors - GitHub CLI can return different exit codes and messages
		errMsg := err.Error()
		outputMsg := string(output)
		combinedMsg := errMsg + " " + outputMsg
		if verbose {
			fmt.Println(console.FormatVerboseMessage(outputMsg))
		}
		if strings.Contains(combinedMsg, "exit status 4") ||
			strings.Contains(combinedMsg, "exit status 1") ||
			strings.Contains(combinedMsg, "not logged into any GitHub hosts") ||
			strings.Contains(combinedMsg, "To use GitHub CLI in a GitHub Actions workflow") ||
			strings.Contains(combinedMsg, "authentication required") ||
			strings.Contains(outputMsg, "gh auth login") {
			return nil, 0, fmt.Errorf("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		if len(output) > 0 {
			return nil, 0, fmt.Errorf("failed to list workflow runs: %s", string(output))
		}
		return nil, 0, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		return nil, 0, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	// Store the total count fetched from API before filtering
	totalFetched := len(runs)

	// Filter only agentic workflow runs when no specific workflow is specified
	// If a workflow name was specified, we already filtered by it in the API call
	var agenticRuns []WorkflowRun
	if workflowName == "" {
		// No specific workflow requested, filter to only agentic workflows
		// Get the list of agentic workflow names from .lock.yml files
		agenticWorkflowNames, err := getAgenticWorkflowNames(verbose)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get agentic workflow names: %w", err)
		}

		for _, run := range runs {
			if contains(agenticWorkflowNames, run.WorkflowName) {
				agenticRuns = append(agenticRuns, run)
			}
		}
	} else {
		// Specific workflow requested, return all runs (they're already filtered by GitHub API)
		agenticRuns = runs
	}

	// Apply run ID filtering if specified
	if beforeRunID > 0 || afterRunID > 0 {
		var filteredRuns []WorkflowRun
		for _, run := range agenticRuns {
			// Apply before-run-id filter (exclusive)
			if beforeRunID > 0 && run.DatabaseID >= beforeRunID {
				continue
			}
			// Apply after-run-id filter (exclusive)
			if afterRunID > 0 && run.DatabaseID <= afterRunID {
				continue
			}
			filteredRuns = append(filteredRuns, run)
		}
		agenticRuns = filteredRuns
	}

	return agenticRuns, totalFetched, nil
}

// flattenSingleFileArtifacts applies the artifact unfold rule to downloaded artifacts
// Unfold rule: If an artifact download folder contains a single file, move the file to root and delete the folder
// This simplifies artifact access by removing unnecessary nesting for single-file artifacts

// downloadWorkflowRunLogs downloads and unzips workflow run logs using GitHub API

// unzipFile extracts a zip file to a destination directory

// extractZipFile extracts a single file from a zip archive

// loadRunSummary attempts to load a run summary from disk
// Returns the summary and a boolean indicating if it was successfully loaded and is valid
func loadRunSummary(outputDir string, verbose bool) (*RunSummary, bool) {
	summaryPath := filepath.Join(outputDir, runSummaryFileName)

	// Check if summary file exists
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		return nil, false
	}

	// Read the summary file
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read run summary: %v", err)))
		}
		return nil, false
	}

	// Parse the JSON
	var summary RunSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse run summary: %v", err)))
		}
		return nil, false
	}

	// Validate CLI version matches
	currentVersion := GetVersion()
	if summary.CLIVersion != currentVersion {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run summary version mismatch (cached: %s, current: %s), will reprocess", summary.CLIVersion, currentVersion)))
		}
		return nil, false
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Loaded cached run summary for run %d (processed at %s)", summary.RunID, summary.ProcessedAt.Format(time.RFC3339))))
	}

	return &summary, true
}

// saveRunSummary saves a run summary to disk
func saveRunSummary(outputDir string, summary *RunSummary, verbose bool) error {
	summaryPath := filepath.Join(outputDir, runSummaryFileName)

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal run summary: %w", err)
	}

	// Write to file
	if err := os.WriteFile(summaryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write run summary: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Saved run summary to %s", summaryPath)))
	}

	return nil
}

// listArtifacts creates a list of all artifact files in the output directory

// downloadRunArtifacts downloads artifacts for a specific workflow run

// extractLogMetrics extracts metrics from downloaded log files

// parseAwInfo reads and parses aw_info.json file, returning the parsed data
// Handles cases where aw_info.json is a file or a directory containing the actual file

// extractEngineFromAwInfo reads aw_info.json and returns the appropriate engine
// Handles cases where aw_info.json is a file or a directory containing the actual file

// parseLogFileWithEngine parses a log file using a specific engine or falls back to auto-detection

// displayLogsOverview displays a summary table of workflow runs and metrics
func displayLogsOverview(processedRuns []ProcessedRun, verbose bool) {
	if len(processedRuns) == 0 {
		return
	}

	// Prepare table data
	headers := []string{"Run ID", "Workflow", "Status", "Duration", "Tokens", "Cost ($)", "Turns", "Errors", "Warnings", "Missing", "Created", "Logs Path"}
	var rows [][]string

	var totalTokens int
	var totalCost float64
	var totalDuration time.Duration
	var totalTurns int
	var totalErrors int
	var totalWarnings int
	var totalMissingTools int

	for _, pr := range processedRuns {
		run := pr.Run
		// Format duration
		durationStr := ""
		if run.Duration > 0 {
			durationStr = timeutil.FormatDuration(run.Duration)
			totalDuration += run.Duration
		}

		// Format cost
		costStr := ""
		if run.EstimatedCost > 0 {
			costStr = fmt.Sprintf("%.3f", run.EstimatedCost)
			totalCost += run.EstimatedCost
		}

		// Format tokens
		tokensStr := ""
		if run.TokenUsage > 0 {
			tokensStr = console.FormatNumber(run.TokenUsage)
			totalTokens += run.TokenUsage
		}

		// Format turns
		turnsStr := ""
		if run.Turns > 0 {
			turnsStr = fmt.Sprintf("%d", run.Turns)
			totalTurns += run.Turns
		}

		// Format errors
		errorsStr := fmt.Sprintf("%d", run.ErrorCount)
		totalErrors += run.ErrorCount

		// Format warnings
		warningsStr := fmt.Sprintf("%d", run.WarningCount)
		totalWarnings += run.WarningCount

		// Format missing tools
		var missingToolsStr string
		if verbose && len(pr.MissingTools) > 0 {
			// In verbose mode, show actual tool names
			toolNames := make([]string, len(pr.MissingTools))
			for i, tool := range pr.MissingTools {
				toolNames[i] = tool.Tool
			}
			missingToolsStr = strings.Join(toolNames, ", ")
			// Truncate if too long
			if len(missingToolsStr) > 30 {
				missingToolsStr = missingToolsStr[:27] + "..."
			}
		} else {
			// In normal mode, just show the count
			missingToolsStr = fmt.Sprintf("%d", run.MissingToolCount)
		}
		totalMissingTools += run.MissingToolCount

		// Truncate workflow name if too long
		workflowName := run.WorkflowName
		if len(workflowName) > 20 {
			workflowName = workflowName[:17] + "..."
		}

		// Format relative path
		relPath, _ := filepath.Rel(".", run.LogsPath)

		// Format status - show conclusion directly for completed runs
		statusStr := run.Status
		if run.Status == "completed" && run.Conclusion != "" {
			statusStr = run.Conclusion
		}

		row := []string{
			fmt.Sprintf("%d", run.DatabaseID),
			workflowName,
			statusStr,
			durationStr,
			tokensStr,
			costStr,
			turnsStr,
			errorsStr,
			warningsStr,
			missingToolsStr,
			run.CreatedAt.Format("2006-01-02"),
			relPath,
		}
		rows = append(rows, row)
	}

	// Prepare total row
	totalRow := []string{
		fmt.Sprintf("TOTAL (%d runs)", len(processedRuns)),
		"",
		"",
		timeutil.FormatDuration(totalDuration),
		console.FormatNumber(totalTokens),
		fmt.Sprintf("%.3f", totalCost),
		fmt.Sprintf("%d", totalTurns),
		fmt.Sprintf("%d", totalErrors),
		fmt.Sprintf("%d", totalWarnings),
		fmt.Sprintf("%d", totalMissingTools),
		"",
		"",
	}

	// Render table using console helper
	tableConfig := console.TableConfig{
		Title:     "Workflow Logs Overview",
		Headers:   headers,
		Rows:      rows,
		ShowTotal: true,
		TotalRow:  totalRow,
	}

	fmt.Print(console.RenderTable(tableConfig))
}

// displayToolCallReport displays a table of tool usage statistics across all runs
// ExtractLogMetricsFromRun extracts log metrics from a processed run's log directory

// findAgentOutputFile searches for a file named agent_output.json within the logDir tree.
// Returns the first path found (depth-first) and a boolean indicating success.

// findAgentLogFile searches for agent logs within the logDir.
// It uses engine.GetLogFileForParsing() to determine which log file to use:
//   - If GetLogFileForParsing() returns a non-empty value that doesn't point to agent-stdio.log,
//     look for files in the "agent_output" artifact directory
//   - Otherwise, look for the "agent-stdio.log" artifact file
//
// Returns the first path found and a boolean indicating success.

// fileExists checks if a file exists

// copyFileSimple copies a file from src to dst using buffered IO.

// dirExists checks if a directory exists

// isDirEmpty checks if a directory is empty

// getAgenticWorkflowNames reads all .lock.yml files and extracts their workflow names
func getAgenticWorkflowNames(verbose bool) ([]string, error) {
	var workflowNames []string

	// Look for .lock.yml files in .github/workflows directory
	workflowsDir := ".github/workflows"
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		if verbose {
			fmt.Println(console.FormatWarningMessage("No .github/workflows directory found"))
		}
		return workflowNames, nil
	}

	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob .lock.yml files: %w", err)
	}

	for _, file := range files {
		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Reading workflow file: %s", file)))
		}

		content, err := os.ReadFile(file)
		if err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", file, err)))
			}
			continue
		}

		// Extract the workflow name using simple string parsing
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "name:") {
				// Parse the name field
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					// Remove quotes if present
					name = strings.Trim(name, `"'`)
					if name != "" {
						workflowNames = append(workflowNames, name)
						if verbose {
							fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found agentic workflow: %s", name)))
						}
						break
					}
				}
			}
		}
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d agentic workflows", len(workflowNames))))
	}

	return workflowNames, nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// extractMissingToolsFromRun extracts missing tool reports from a workflow run's artifacts

// displayMissingToolsAnalysis displays a summary of missing tools across all runs
func displayMissingToolsAnalysis(processedRuns []ProcessedRun, verbose bool) {
	// Aggregate missing tools across all runs
	toolSummary := make(map[string]*MissingToolSummary)
	var totalReports int

	for _, pr := range processedRuns {
		for _, tool := range pr.MissingTools {
			totalReports++
			if summary, exists := toolSummary[tool.Tool]; exists {
				summary.Count++
				// Add workflow if not already in the list
				found := false
				for _, wf := range summary.Workflows {
					if wf == tool.WorkflowName {
						found = true
						break
					}
				}
				if !found {
					summary.Workflows = append(summary.Workflows, tool.WorkflowName)
				}
				summary.RunIDs = append(summary.RunIDs, tool.RunID)
			} else {
				toolSummary[tool.Tool] = &MissingToolSummary{
					Tool:        tool.Tool,
					Count:       1,
					Workflows:   []string{tool.WorkflowName},
					FirstReason: tool.Reason,
					RunIDs:      []int64{tool.RunID},
				}
			}
		}
	}

	if totalReports == 0 {
		return // No missing tools to display
	}

	// Display summary header
	fmt.Printf("\n%s\n", console.FormatListHeader("🛠️  Missing Tools Summary"))

	// Convert map to slice for sorting
	var summaries []*MissingToolSummary
	for _, summary := range toolSummary {
		summaries = append(summaries, summary)
	}

	// Sort by count (descending)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Count > summaries[j].Count
	})

	// Display summary table
	headers := []string{"Tool", "Occurrences", "Workflows", "First Reason"}
	var rows [][]string

	for _, summary := range summaries {
		workflowList := strings.Join(summary.Workflows, ", ")
		if len(workflowList) > 40 {
			workflowList = workflowList[:37] + "..."
		}

		reason := summary.FirstReason
		if len(reason) > 50 {
			reason = reason[:47] + "..."
		}

		rows = append(rows, []string{
			summary.Tool,
			fmt.Sprintf("%d", summary.Count),
			workflowList,
			reason,
		})
	}

	tableConfig := console.TableConfig{
		Headers: headers,
		Rows:    rows,
	}

	fmt.Print(console.RenderTable(tableConfig))

	// Display total summary
	uniqueTools := len(toolSummary)
	fmt.Printf("\n📊 %s: %d unique missing tools reported %d times across workflows\n",
		console.FormatCountMessage("Total"),
		uniqueTools,
		totalReports)

	// Verbose mode: Show detailed breakdown by workflow
	if verbose && totalReports > 0 {
		displayDetailedMissingToolsBreakdown(processedRuns)
	}
}

// displayDetailedMissingToolsBreakdown shows missing tools organized by workflow (verbose mode)
func displayDetailedMissingToolsBreakdown(processedRuns []ProcessedRun) {
	fmt.Printf("\n%s\n", console.FormatListHeader("🔍 Detailed Missing Tools Breakdown"))

	for _, pr := range processedRuns {
		if len(pr.MissingTools) == 0 {
			continue
		}

		fmt.Printf("\n%s (Run %d) - %d missing tools:\n",
			console.FormatInfoMessage(pr.Run.WorkflowName),
			pr.Run.DatabaseID,
			len(pr.MissingTools))

		for i, tool := range pr.MissingTools {
			fmt.Printf("  %d. %s %s\n",
				i+1,
				console.FormatListItem(tool.Tool),
				console.FormatVerboseMessage(fmt.Sprintf("- %s", tool.Reason)))

			if tool.Alternatives != "" && tool.Alternatives != "null" {
				fmt.Printf("     %s %s\n",
					console.FormatWarningMessage("Alternatives:"),
					tool.Alternatives)
			}

			if tool.Timestamp != "" {
				fmt.Printf("     %s %s\n",
					console.FormatVerboseMessage("Reported at:"),
					tool.Timestamp)
			}
		}
	}
}

// extractMCPFailuresFromRun extracts MCP server failure reports from a workflow run's logs

// extractMCPFailuresFromLogFile parses a single log file for MCP server failures

// MCPFailureSummary aggregates MCP server failures across runs
type MCPFailureSummary struct {
	ServerName       string   `json:"server_name" console:"header:Server"`
	Count            int      `json:"count" console:"header:Failures"`
	Workflows        []string `json:"workflows" console:"-"`                  // List of workflow names that had this server fail
	WorkflowsDisplay string   `json:"-" console:"header:Workflows,maxlen:60"` // Formatted display of workflows
	RunIDs           []int64  `json:"run_ids" console:"-"`                    // List of run IDs where this server failed
}

// displayMCPFailuresAnalysis displays a summary of MCP server failures across all runs
// parseAgentLog runs the JavaScript log parser on agent logs and writes markdown to log.md

// parseFirewallLogs runs the JavaScript firewall log parser and writes markdown to firewall.md
