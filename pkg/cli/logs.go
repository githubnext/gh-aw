package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
)

// WorkflowRun represents a GitHub Actions workflow run with metrics
type WorkflowRun struct {
	DatabaseID    int64     `json:"databaseId"`
	Number        int       `json:"number"`
	URL           string    `json:"url"`
	Status        string    `json:"status"`
	Conclusion    string    `json:"conclusion"`
	WorkflowName  string    `json:"workflowName"`
	CreatedAt     time.Time `json:"createdAt"`
	StartedAt     time.Time `json:"startedAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Event         string    `json:"event"`
	HeadBranch    string    `json:"headBranch"`
	HeadSha       string    `json:"headSha"`
	DisplayTitle  string    `json:"displayTitle"`
	Duration      time.Duration
	TokenUsage    int
	EstimatedCost float64
	Turns         int
	ErrorCount    int
	WarningCount  int
	LogsPath      string
}

// LogMetrics represents extracted metrics from log files
// This is now an alias to the shared type in workflow package
type LogMetrics = workflow.LogMetrics

// ProcessedRun represents a workflow run with its associated analysis
type ProcessedRun struct {
	Run            WorkflowRun
	AccessAnalysis *DomainAnalysis
	MissingTools   []MissingToolReport
	MCPFailures    []MCPFailureReport
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
	Tool        string
	Count       int
	Workflows   []string // List of workflow names that reported this tool
	FirstReason string   // Reason from the first occurrence
	RunIDs      []int64  // List of run IDs where this tool was reported
}

// ErrNoArtifacts indicates that a workflow run has no artifacts
var ErrNoArtifacts = errors.New("no artifacts found for this run")

// fetchJobStatuses gets job information for a workflow run and counts failed jobs
func fetchJobStatuses(runID int64, verbose bool) (int, error) {
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
		if job.Conclusion == "failure" || job.Conclusion == "cancelled" || job.Conclusion == "timed_out" {
			failedJobs++
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Found failed job '%s' with conclusion '%s'", job.Name, job.Conclusion)))
			}
		}
	}

	return failedJobs, nil
}

// DownloadResult represents the result of downloading artifacts for a single run
type DownloadResult struct {
	Run            WorkflowRun
	Metrics        LogMetrics
	AccessAnalysis *DomainAnalysis
	MissingTools   []MissingToolReport
	MCPFailures    []MCPFailureReport
	Error          error
	Skipped        bool
	LogsPath       string
}

// JobInfo represents basic information about a workflow job
type JobInfo struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// AwInfo represents the structure of aw_info.json files
type AwInfo struct {
	EngineID     string `json:"engine_id"`
	EngineName   string `json:"engine_name"`
	Model        string `json:"model"`
	Version      string `json:"version"`
	WorkflowName string `json:"workflow_name"`
	Staged       bool   `json:"staged"`
	CreatedAt    string `json:"created_at"`
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
		Use:   "logs [agentic-workflow-id]",
		Short: "Download and analyze agentic workflow logs with aggregated metrics",
		Long: `Download workflow run logs and artifacts from GitHub Actions for agentic workflows.

This command fetches workflow runs, downloads their artifacts, and extracts them into
organized folders named by run ID. It also provides an overview table with aggregate
metrics including duration, token usage, and cost information.

Downloaded artifacts include:
- aw_info.json: Engine configuration and workflow metadata
- safe_output.jsonl: Agent's final output content (available when non-empty)
- agent_output.json: Full/raw agent output (if the workflow uploaded this artifact)
- aw.patch: Git patch of changes made during execution
- Various log files with execution details and metrics

The agentic-workflow-id is the basename of the markdown file without the .md extension.
For example, for 'weekly-research.md', use 'weekly-research' as the workflow ID.

Examples:
  ` + constants.CLIExtensionPrefix + ` logs                           # Download logs for all workflows
  ` + constants.CLIExtensionPrefix + ` logs weekly-research           # Download logs for specific agentic workflow
  ` + constants.CLIExtensionPrefix + ` logs -c 10                     # Download last 10 runs
  ` + constants.CLIExtensionPrefix + ` logs --start-date 2024-01-01   # Filter runs after date
  ` + constants.CLIExtensionPrefix + ` logs --end-date 2024-01-31     # Filter runs before date
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1w          # Filter runs from last week
  ` + constants.CLIExtensionPrefix + ` logs --end-date -1d            # Filter runs until yesterday
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1mo         # Filter runs from last month
  ` + constants.CLIExtensionPrefix + ` logs --engine claude           # Filter logs by claude engine
  ` + constants.CLIExtensionPrefix + ` logs --engine codex            # Filter logs by codex engine
  ` + constants.CLIExtensionPrefix + ` logs --engine copilot          # Filter logs by copilot engine
  ` + constants.CLIExtensionPrefix + ` logs -o ./my-logs              # Custom output directory
  ` + constants.CLIExtensionPrefix + ` logs --branch main             # Filter logs by branch name
  ` + constants.CLIExtensionPrefix + ` logs --branch feature-xyz      # Filter logs by feature branch
  ` + constants.CLIExtensionPrefix + ` logs --after-run-id 1000       # Filter runs after run ID 1000
  ` + constants.CLIExtensionPrefix + ` logs --before-run-id 2000      # Filter runs before run ID 2000
  ` + constants.CLIExtensionPrefix + ` logs --after-run-id 1000 --before-run-id 2000  # Filter runs in range
  ` + constants.CLIExtensionPrefix + ` logs -o ./my-logs              # Custom output directory
  ` + constants.CLIExtensionPrefix + ` logs --tool-graph              # Generate Mermaid tool sequence graph`,
		Run: func(cmd *cobra.Command, args []string) {
			var workflowName string
			if len(args) > 0 && args[0] != "" {
				// Convert agentic workflow ID to GitHub Actions workflow name
				// First try to resolve as an agentic workflow ID
				resolvedName, err := workflow.ResolveWorkflowName(args[0])
				if err != nil {
					// If that fails, check if it's already a GitHub Actions workflow name
					// by checking if any .lock.yml files have this as their name
					agenticWorkflowNames, nameErr := getAgenticWorkflowNames(false)
					if nameErr == nil && contains(agenticWorkflowNames, args[0]) {
						// It's already a valid GitHub Actions workflow name
						workflowName = args[0]
					} else {
						// Neither agentic workflow ID nor valid GitHub Actions workflow name
						fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
							Type:    "error",
							Message: fmt.Sprintf("workflow '%s' not found. Expected either an agentic workflow ID (e.g., 'test-claude') or GitHub Actions workflow name (e.g., 'Test Claude'). Original error: %v", args[0], err),
						}))
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

			if err := DownloadWorkflowLogs(workflowName, count, startDate, endDate, outputDir, engine, branch, beforeRunID, afterRunID, verbose, toolGraph, noStaged); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
					Type:    "error",
					Message: err.Error(),
				}))
				os.Exit(1)
			}
		},
	}

	// Add flags to logs command
	logsCmd.Flags().IntP("count", "c", 20, "Maximum number of workflow runs to fetch")
	logsCmd.Flags().String("start-date", "", "Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	logsCmd.Flags().String("end-date", "", "Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	logsCmd.Flags().StringP("output", "o", "./logs", "Output directory for downloaded logs and artifacts")
	logsCmd.Flags().String("engine", "", "Filter logs by agentic engine type (claude, codex, copilot)")
	logsCmd.Flags().String("branch", "", "Filter runs by branch name (e.g., main, feature-branch)")
	logsCmd.Flags().Int64("before-run-id", 0, "Filter runs with database ID before this value (exclusive)")
	logsCmd.Flags().Int64("after-run-id", 0, "Filter runs with database ID after this value (exclusive)")
	logsCmd.Flags().BoolP("verbose", "v", false, "Show individual tool names instead of grouping by MCP server")
	logsCmd.Flags().Bool("tool-graph", false, "Generate Mermaid tool sequence graph from agent logs")
	logsCmd.Flags().Bool("no-staged", false, "Filter out staged workflow runs (exclude runs with staged: true in aw_info.json)")

	return logsCmd
}

// DownloadWorkflowLogs downloads and analyzes workflow logs with metrics
func DownloadWorkflowLogs(workflowName string, count int, startDate, endDate, outputDir, engine, branch string, beforeRunID, afterRunID int64, verbose bool, toolGraph bool, noStaged bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}

	var processedRuns []ProcessedRun
	var beforeDate string
	iteration := 0

	// Iterative algorithm: keep fetching runs until we have enough with artifacts
	for len(processedRuns) < count && iteration < MaxIterations {
		iteration++

		if verbose && iteration > 1 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Need %d more runs with artifacts, fetching more...", iteration, count-len(processedRuns))))
		}

		// Fetch a batch of runs
		batchSize := BatchSize
		if workflowName == "" {
			// When searching for all agentic workflows, use a larger batch size
			// since there may be many CI runs interspersed with agentic runs
			batchSize = BatchSizeForAllWorkflows
		}
		if count-len(processedRuns) < batchSize {
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

		runs, err := listWorkflowRunsWithPagination(workflowName, batchSize, startDate, endDate, beforeDate, branch, beforeRunID, afterRunID, verbose)
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
		downloadResults := downloadRunArtifactsConcurrent(runs, outputDir, verbose, count-len(processedRuns))

		for _, result := range downloadResults {
			// Stop if we've reached our target count
			if len(processedRuns) >= count {
				break
			}

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

			// Apply engine filtering if specified
			if engine != "" {
				// Check if the run's engine matches the filter
				awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
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
				// Check if the run is staged
				awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
				info, err := parseAwInfo(awInfoPath, verbose)
				var isStaged bool
				if err == nil && info != nil {
					isStaged = info.Staged
				}

				if isStaged {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow is staged (filtered out by --no-staged)", result.Run.DatabaseID)))
					}
					continue
				}
			}

			// Update run with metrics and path
			run := result.Run
			run.TokenUsage = result.Metrics.TokenUsage
			run.EstimatedCost = result.Metrics.EstimatedCost
			run.Turns = result.Metrics.Turns
			run.ErrorCount = result.Metrics.ErrorCount
			run.WarningCount = result.Metrics.WarningCount
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
				Run:            run,
				AccessAnalysis: result.AccessAnalysis,
				MissingTools:   result.MissingTools,
				MCPFailures:    result.MCPFailures,
			}
			processedRuns = append(processedRuns, processedRun)
			batchProcessed++
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d/%d)", batchProcessed, iteration, len(processedRuns), count)))
		}

		// Prepare for next iteration: set beforeDate to the oldest run from this batch
		if len(runs) > 0 {
			oldestRun := runs[len(runs)-1] // runs are typically ordered by creation date descending
			beforeDate = oldestRun.CreatedAt.Format(time.RFC3339)
		}

		// If we got fewer runs than requested in this batch, we've likely hit the end
		if len(runs) < batchSize {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Received fewer runs than requested, likely reached end of available runs"))
			}
			break
		}
	}

	// Check if we hit the maximum iterations limit
	if iteration >= MaxIterations && len(processedRuns) < count {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts out of %d requested", MaxIterations, len(processedRuns), count)))
	}

	if len(processedRuns) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow runs with artifacts found matching the specified criteria"))
		return nil
	}

	// Display overview table
	workflowRuns := make([]WorkflowRun, len(processedRuns))
	for i, pr := range processedRuns {
		workflowRuns[i] = pr.Run
	}
	displayLogsOverview(workflowRuns)

	// Display MCP failures analysis
	displayMCPFailuresAnalysis(processedRuns, verbose)

	// Display tool call report
	displayToolCallReport(processedRuns, verbose)
	// Display missing tools analysis
	displayMissingToolsAnalysis(processedRuns, verbose)

	// Display access log analysis
	displayAccessLogAnalysis(processedRuns, verbose)

	// Generate tool sequence graph if requested
	if toolGraph {
		generateToolGraph(processedRuns, verbose)
	}

	// Display logs location prominently
	absOutputDir, _ := filepath.Abs(outputDir)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Downloaded %d logs to %s", len(processedRuns), absOutputDir)))
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
			err := downloadRunArtifacts(run.DatabaseID, runOutputDir, verbose)

			result := DownloadResult{
				Run:      run,
				LogsPath: runOutputDir,
			}

			if err != nil {
				// Check if this is a "no artifacts" case - mark as skipped for cancelled/failed runs
				if errors.Is(err, ErrNoArtifacts) {
					result.Skipped = true
					result.Error = err
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
func listWorkflowRunsWithPagination(workflowName string, count int, startDate, endDate, beforeDate, branch string, beforeRunID, afterRunID int64, verbose bool) ([]WorkflowRun, error) {
	args := []string{"run", "list", "--json", "databaseId,number,url,status,conclusion,workflowName,createdAt,startedAt,updatedAt,event,headBranch,headSha,displayTitle"}

	// Add filters
	if workflowName != "" {
		args = append(args, "--workflow", workflowName)
	}
	if count > 0 {
		args = append(args, "--limit", strconv.Itoa(count))
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
	spinner := console.NewSpinner("Fetching workflow runs from GitHub...")
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
			return nil, fmt.Errorf("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		if len(output) > 0 {
			return nil, fmt.Errorf("failed to list workflow runs: %s", string(output))
		}
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		return nil, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	// Filter only agentic workflow runs when no specific workflow is specified
	// If a workflow name was specified, we already filtered by it in the API call
	var agenticRuns []WorkflowRun
	if workflowName == "" {
		// No specific workflow requested, filter to only agentic workflows
		// Get the list of agentic workflow names from .lock.yml files
		agenticWorkflowNames, err := getAgenticWorkflowNames(verbose)
		if err != nil {
			return nil, fmt.Errorf("failed to get agentic workflow names: %w", err)
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

	return agenticRuns, nil
}

// downloadRunArtifacts downloads artifacts for a specific workflow run
func downloadRunArtifacts(runID int64, outputDir string, verbose bool) error {
	// Check if artifacts already exist on disk (since they're immutable)
	if dirExists(outputDir) && !isDirEmpty(outputDir) {
		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Artifacts for run %d already exist at %s, skipping download", runID, outputDir)))
		}
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create run output directory: %w", err)
	}

	args := []string{"run", "download", strconv.FormatInt(runID, 10), "--dir", outputDir}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Executing: gh %s", strings.Join(args, " "))))
	}

	// Start spinner for network operation
	spinner := console.NewSpinner(fmt.Sprintf("Downloading artifacts for run %d...", runID))
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
		if verbose {
			fmt.Println(console.FormatVerboseMessage(string(output)))
		}

		// Check if it's because there are no artifacts
		if strings.Contains(string(output), "no valid artifacts") || strings.Contains(string(output), "not found") {
			// Clean up empty directory
			os.RemoveAll(outputDir)
			return ErrNoArtifacts
		}
		// Check for authentication errors
		if strings.Contains(err.Error(), "exit status 4") {
			return fmt.Errorf("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		return fmt.Errorf("failed to download artifacts for run %d: %w (output: %s)", runID, err, string(output))
	}

	if verbose {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Downloaded artifacts for run %d to %s", runID, outputDir)))
	}

	return nil
}

// extractLogMetrics extracts metrics from downloaded log files
func extractLogMetrics(logDir string, verbose bool) (LogMetrics, error) {
	var metrics LogMetrics

	// First check for aw_info.json to determine the engine
	var detectedEngine workflow.CodingAgentEngine
	infoFilePath := filepath.Join(logDir, "aw_info.json")
	if _, err := os.Stat(infoFilePath); err == nil {
		// aw_info.json exists, try to extract engine information
		if engine := extractEngineFromAwInfo(infoFilePath, verbose); engine != nil {
			detectedEngine = engine
			if verbose {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Detected engine from aw_info.json: %s", engine.GetID())))
			}
		}
	}

	// Check for safe_output.jsonl artifact file
	awOutputPath := filepath.Join(logDir, "safe_output.jsonl")
	if _, err := os.Stat(awOutputPath); err == nil {
		if verbose {
			// Report that the agentic output file was found
			fileInfo, statErr := os.Stat(awOutputPath)
			if statErr == nil {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found agentic output file: safe_output.jsonl (%s)", pretty.FormatFileSize(fileInfo.Size()))))
			}
		}
	}

	// Check for aw.patch artifact file
	awPatchPath := filepath.Join(logDir, "aw.patch")
	if _, err := os.Stat(awPatchPath); err == nil {
		if verbose {
			// Report that the git patch file was found
			fileInfo, statErr := os.Stat(awPatchPath)
			if statErr == nil {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found git patch file: aw.patch (%s)", pretty.FormatFileSize(fileInfo.Size()))))
			}
		}
	}

	// Check for agent_output.json artifact (some workflows may store this under a nested directory)
	agentOutputPath, agentOutputFound := findAgentOutputFile(logDir)
	if agentOutputFound {
		if verbose {
			fileInfo, statErr := os.Stat(agentOutputPath)
			if statErr == nil {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found agent output file: %s (%s)", filepath.Base(agentOutputPath), pretty.FormatFileSize(fileInfo.Size()))))
			}
		}
		// If the file is not already in the logDir root, copy it for convenience
		if filepath.Dir(agentOutputPath) != logDir {
			rootCopy := filepath.Join(logDir, constants.AgentOutputArtifactName)
			if _, err := os.Stat(rootCopy); errors.Is(err, os.ErrNotExist) {
				if copyErr := copyFileSimple(agentOutputPath, rootCopy); copyErr == nil && verbose {
					fmt.Println(console.FormatInfoMessage("Copied agent_output.json to run root for easy access"))
				}
			}
		}
	}

	// Walk through all files in the log directory
	err := filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process log files - exclude output artifacts like aw_output.txt
		fileName := strings.ToLower(info.Name())
		if (strings.HasSuffix(fileName, ".log") ||
			(strings.HasSuffix(fileName, ".txt") && strings.Contains(fileName, "log"))) &&
			!strings.Contains(fileName, "aw_output") &&
			!strings.Contains(fileName, "agent_output") {

			fileMetrics, err := parseLogFileWithEngine(path, detectedEngine, verbose)
			if err != nil && verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to parse log file %s: %v", path, err)))
				return nil // Continue processing other files
			}

			// Aggregate metrics
			metrics.TokenUsage += fileMetrics.TokenUsage
			metrics.EstimatedCost += fileMetrics.EstimatedCost
			metrics.ErrorCount += fileMetrics.ErrorCount
			metrics.WarningCount += fileMetrics.WarningCount
			if fileMetrics.Turns > metrics.Turns {
				// For turns, take the maximum rather than summing, since turns represent
				// the total conversation turns for the entire workflow run
				metrics.Turns = fileMetrics.Turns
			}

			// Aggregate tool sequences and tool calls
			metrics.ToolSequences = append(metrics.ToolSequences, fileMetrics.ToolSequences...)
			metrics.ToolCalls = append(metrics.ToolCalls, fileMetrics.ToolCalls...)
		}

		return nil
	})

	return metrics, err
}

// parseAwInfo reads and parses aw_info.json file, returning the parsed data
// Handles cases where aw_info.json is a file or a directory containing the actual file
func parseAwInfo(infoFilePath string, verbose bool) (*AwInfo, error) {
	var data []byte
	var err error

	// Check if the path exists and determine if it's a file or directory
	stat, statErr := os.Stat(infoFilePath)
	if statErr != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to stat aw_info.json: %v", statErr)))
		}
		return nil, statErr
	}

	if stat.IsDir() {
		// It's a directory - look for nested aw_info.json
		nestedPath := filepath.Join(infoFilePath, "aw_info.json")
		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("aw_info.json is a directory, trying nested file: %s", nestedPath)))
		}
		data, err = os.ReadFile(nestedPath)
	} else {
		// It's a regular file
		data, err = os.ReadFile(infoFilePath)
	}

	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to read aw_info.json: %v", err)))
		}
		return nil, err
	}

	var info AwInfo
	if err := json.Unmarshal(data, &info); err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to parse aw_info.json: %v", err)))
		}
		return nil, err
	}

	return &info, nil
}

// extractEngineFromAwInfo reads aw_info.json and returns the appropriate engine
// Handles cases where aw_info.json is a file or a directory containing the actual file
func extractEngineFromAwInfo(infoFilePath string, verbose bool) workflow.CodingAgentEngine {
	info, err := parseAwInfo(infoFilePath, verbose)
	if err != nil {
		return nil
	}

	if info.EngineID == "" {
		if verbose {
			fmt.Println(console.FormatWarningMessage("No engine_id found in aw_info.json"))
		}
		return nil
	}

	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine(info.EngineID)
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Unknown engine in aw_info.json: %s", info.EngineID)))
		}
		return nil
	}

	return engine
}

// parseLogFileWithEngine parses a log file using a specific engine or falls back to auto-detection
func parseLogFileWithEngine(filePath string, detectedEngine workflow.CodingAgentEngine, verbose bool) (LogMetrics, error) {
	// Read the entire log file at once to avoid JSON parsing issues from chunked reading
	content, err := os.ReadFile(filePath)
	if err != nil {
		return LogMetrics{}, fmt.Errorf("error reading log file: %w", err)
	}

	logContent := string(content)

	// If we have a detected engine from aw_info.json, use it directly
	if detectedEngine != nil {
		return detectedEngine.ParseLogMetrics(logContent, verbose), nil
	}

	// No aw_info.json metadata available - return empty metrics
	if verbose {
		fmt.Println(console.FormatWarningMessage("No aw_info.json found, unable to parse engine-specific metrics"))
	}
	return LogMetrics{}, nil
}

// Shared utilities are now in workflow package
// extractJSONMetrics is available as an alias
var extractJSONMetrics = workflow.ExtractJSONMetrics

// displayLogsOverview displays a summary table of workflow runs and metrics
func displayLogsOverview(runs []WorkflowRun) {
	if len(runs) == 0 {
		return
	}

	// Prepare table data
	headers := []string{"Run ID", "Workflow", "Status", "Duration", "Tokens", "Cost ($)", "Turns", "Errors", "Warnings", "Created", "Logs Path"}
	var rows [][]string

	var totalTokens int
	var totalCost float64
	var totalDuration time.Duration
	var totalTurns int
	var totalErrors int
	var totalWarnings int

	for _, run := range runs {
		// Format duration
		durationStr := ""
		if run.Duration > 0 {
			durationStr = formatDuration(run.Duration)
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
			tokensStr = formatNumber(run.TokenUsage)
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
			run.CreatedAt.Format("2006-01-02"),
			relPath,
		}
		rows = append(rows, row)
	}

	// Prepare total row
	totalRow := []string{
		fmt.Sprintf("TOTAL (%d runs)", len(runs)),
		"",
		"",
		formatDuration(totalDuration),
		formatNumber(totalTokens),
		fmt.Sprintf("%.3f", totalCost),
		fmt.Sprintf("%d", totalTurns),
		fmt.Sprintf("%d", totalErrors),
		fmt.Sprintf("%d", totalWarnings),
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
func displayToolCallReport(processedRuns []ProcessedRun, verbose bool) {
	if len(processedRuns) == 0 {
		return
	}

	// Aggregate tool call statistics across all runs
	// In verbose mode: show individual tools, in non-verbose mode: group by MCP server
	toolStats := make(map[string]*workflow.ToolCallInfo)

	for _, processedRun := range processedRuns {
		// Extract tool calls from the run's metrics - we need to get the LogMetrics
		// This requires getting the metrics from the processed run

		// For now, let's extract metrics from the run if available
		// We'll process log files to get tool call information
		logMetrics := ExtractLogMetricsFromRun(processedRun)

		for _, toolCall := range logMetrics.ToolCalls {
			var displayKey string

			if verbose {
				// Verbose mode: show individual prettified tool names
				displayKey = workflow.PrettifyToolName(toolCall.Name)
			} else {
				// Non-verbose mode: group by MCP server for MCP tools, keep individual entries for others
				if strings.HasPrefix(toolCall.Name, "mcp__") {
					// Extract server name for MCP tools
					displayKey = workflow.ExtractMCPServer(toolCall.Name)
				} else if strings.HasPrefix(toolCall.Name, "bash_") {
					// Keep bash commands as individual entries since they include command details
					displayKey = toolCall.Name
				} else {
					// For other tools, check if they follow the new server_method pattern
					// This handles tools that have been prettified to server_method format
					parts := strings.SplitN(toolCall.Name, "_", 2)
					if len(parts) == 2 && !strings.HasPrefix(toolCall.Name, "bash_") {
						// This looks like it could be a server_method format, group by server
						displayKey = parts[0]
					} else {
						// Keep as individual entry
						displayKey = toolCall.Name
					}
				}
			}

			if existing, exists := toolStats[displayKey]; exists {
				existing.CallCount += toolCall.CallCount
				if toolCall.MaxOutputSize > existing.MaxOutputSize {
					existing.MaxOutputSize = toolCall.MaxOutputSize
				}
				if toolCall.MaxDuration > existing.MaxDuration {
					existing.MaxDuration = toolCall.MaxDuration
				}
			} else {
				toolStats[displayKey] = &workflow.ToolCallInfo{
					Name:          displayKey,
					CallCount:     toolCall.CallCount,
					MaxOutputSize: toolCall.MaxOutputSize,
					MaxDuration:   toolCall.MaxDuration,
				}
			}
		}
	}

	// Convert to slice and sort by call count (descending), then by name
	var toolCalls []workflow.ToolCallInfo
	for _, toolInfo := range toolStats {
		toolCalls = append(toolCalls, *toolInfo)
	}

	if len(toolCalls) == 0 {
		return // No tool calls found
	}

	sort.Slice(toolCalls, func(i, j int) bool {
		if toolCalls[i].CallCount != toolCalls[j].CallCount {
			return toolCalls[i].CallCount > toolCalls[j].CallCount // Descending by call count
		}
		return toolCalls[i].Name < toolCalls[j].Name // Ascending by name
	})

	// Prepare table data
	headers := []string{"Tool", "Calls", "Max Output (tokens)", "Max Duration"}
	var rows [][]string

	for _, toolCall := range toolCalls {
		outputStr := "N/A"
		if toolCall.MaxOutputSize > 0 {
			outputStr = formatNumber(toolCall.MaxOutputSize)
		}

		durationStr := "N/A"
		if toolCall.MaxDuration > 0 {
			durationStr = formatDuration(toolCall.MaxDuration)
		}

		row := []string{
			toolCall.Name,
			fmt.Sprintf("%d", toolCall.CallCount),
			outputStr,
			durationStr,
		}
		rows = append(rows, row)
	}

	// Render compact table without title as requested
	tableConfig := console.TableConfig{
		Title:     "MCP Tool Calls",
		Headers:   headers,
		Rows:      rows,
		ShowTotal: false, // Keep it simple and compact
	}

	fmt.Print(console.RenderTable(tableConfig))
}

// ExtractLogMetricsFromRun extracts log metrics from a processed run's log directory
func ExtractLogMetricsFromRun(processedRun ProcessedRun) workflow.LogMetrics {
	// Use the LogsPath from the WorkflowRun to get metrics
	if processedRun.Run.LogsPath == "" {
		return workflow.LogMetrics{}
	}

	// Extract metrics from the log directory
	metrics, err := extractLogMetrics(processedRun.Run.LogsPath, false)
	if err != nil {
		return workflow.LogMetrics{}
	}

	return metrics
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// formatNumber formats large numbers in a human-readable way (e.g., "1k", "1.2k", "1.12M")
func formatNumber(n int) string {
	if n == 0 {
		return "0"
	}

	f := float64(n)

	if f < 1000 {
		return fmt.Sprintf("%d", n)
	} else if f < 1000000 {
		// Format as thousands (k)
		k := f / 1000
		if k >= 100 {
			return fmt.Sprintf("%.0fk", k)
		} else if k >= 10 {
			return fmt.Sprintf("%.1fk", k)
		} else {
			return fmt.Sprintf("%.2fk", k)
		}
	} else if f < 1000000000 {
		// Format as millions (M)
		m := f / 1000000
		if m >= 100 {
			return fmt.Sprintf("%.0fM", m)
		} else if m >= 10 {
			return fmt.Sprintf("%.1fM", m)
		} else {
			return fmt.Sprintf("%.2fM", m)
		}
	} else {
		// Format as billions (B)
		b := f / 1000000000
		if b >= 100 {
			return fmt.Sprintf("%.0fB", b)
		} else if b >= 10 {
			return fmt.Sprintf("%.1fB", b)
		} else {
			return fmt.Sprintf("%.2fB", b)
		}
	}
}

// findAgentOutputFile searches for a file named agent_output.json within the logDir tree.
// Returns the first path found (depth-first) and a boolean indicating success.
func findAgentOutputFile(logDir string) (string, bool) {
	var foundPath string
	_ = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), constants.AgentOutputArtifactName) {
			foundPath = path
			return errors.New("stop") // sentinel to stop walking early
		}
		return nil
	})
	if foundPath == "" {
		return "", false
	}
	return foundPath, true
}

// copyFileSimple copies a file from src to dst using buffered IO.
func copyFileSimple(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(path string) bool {
	files, err := os.ReadDir(path)
	if err != nil {
		return true // Consider it empty if we can't read it
	}
	return len(files) == 0
}

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
func extractMissingToolsFromRun(runDir string, run WorkflowRun, verbose bool) ([]MissingToolReport, error) {
	var missingTools []MissingToolReport

	// Look for the safe output artifact file that contains structured JSON with items array
	// This file is created by the collect_ndjson_output.cjs script during workflow execution
	agentOutputPath := filepath.Join(runDir, constants.AgentOutputArtifactName)
	if _, err := os.Stat(agentOutputPath); err == nil {
		// Read the safe output artifact file
		content, readErr := os.ReadFile(agentOutputPath)
		if readErr != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to read safe output file %s: %v", agentOutputPath, readErr)))
			}
			return missingTools, nil // Continue processing without this file
		}

		// Parse the structured JSON output from the collect script
		var safeOutput struct {
			Items  []json.RawMessage `json:"items"`
			Errors []string          `json:"errors,omitempty"`
		}

		if err := json.Unmarshal(content, &safeOutput); err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to parse safe output JSON from %s: %v", agentOutputPath, err)))
			}
			return missingTools, nil // Continue processing without this file
		}

		// Extract missing-tool entries from the items array
		for _, itemRaw := range safeOutput.Items {
			var item struct {
				Type         string `json:"type"`
				Tool         string `json:"tool,omitempty"`
				Reason       string `json:"reason,omitempty"`
				Alternatives string `json:"alternatives,omitempty"`
				Timestamp    string `json:"timestamp,omitempty"`
			}

			if err := json.Unmarshal(itemRaw, &item); err != nil {
				if verbose {
					fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to parse item from safe output: %v", err)))
				}
				continue // Skip malformed items
			}

			// Check if this is a missing-tool entry
			if item.Type == "missing-tool" {
				missingTool := MissingToolReport{
					Tool:         item.Tool,
					Reason:       item.Reason,
					Alternatives: item.Alternatives,
					Timestamp:    item.Timestamp,
					WorkflowName: run.WorkflowName,
					RunID:        run.DatabaseID,
				}
				missingTools = append(missingTools, missingTool)

				if verbose {
					fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found missing-tool entry: %s (%s)", item.Tool, item.Reason)))
				}
			}
		}

		if verbose && len(missingTools) > 0 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d missing tool reports in safe output artifact for run %d", len(missingTools), run.DatabaseID)))
		}
	} else {
		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("No safe output artifact found at %s for run %d", agentOutputPath, run.DatabaseID)))
		}
	}

	return missingTools, nil
}

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
func extractMCPFailuresFromRun(runDir string, run WorkflowRun, verbose bool) ([]MCPFailureReport, error) {
	var mcpFailures []MCPFailureReport

	// Look for agent output logs that contain the system init entry with MCP server status
	// This information is available in the raw log files, typically with names containing "log"
	err := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process log files - exclude output artifacts
		fileName := strings.ToLower(info.Name())
		if (strings.HasSuffix(fileName, ".log") ||
			(strings.HasSuffix(fileName, ".txt") && strings.Contains(fileName, "log"))) &&
			!strings.Contains(fileName, "aw_output") &&
			!strings.Contains(fileName, "agent_output") &&
			!strings.Contains(fileName, "access") {

			failures, parseErr := extractMCPFailuresFromLogFile(path, run, verbose)
			if parseErr != nil {
				if verbose {
					fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to parse MCP failures from %s: %v", filepath.Base(path), parseErr)))
				}
				return nil // Continue processing other files
			}
			mcpFailures = append(mcpFailures, failures...)
		}

		return nil
	})

	if err != nil {
		return mcpFailures, fmt.Errorf("error walking run directory: %w", err)
	}

	if verbose && len(mcpFailures) > 0 {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d MCP server failures for run %d", len(mcpFailures), run.DatabaseID)))
	}

	return mcpFailures, nil
}

// extractMCPFailuresFromLogFile parses a single log file for MCP server failures
func extractMCPFailuresFromLogFile(logPath string, run WorkflowRun, verbose bool) ([]MCPFailureReport, error) {
	var mcpFailures []MCPFailureReport

	content, err := os.ReadFile(logPath)
	if err != nil {
		return mcpFailures, fmt.Errorf("error reading log file: %w", err)
	}

	logContent := string(content)

	// First try to parse as JSON array
	var logEntries []map[string]any
	if err := json.Unmarshal(content, &logEntries); err == nil {
		// Successfully parsed as JSON array, process entries
		for _, entry := range logEntries {
			if entryType, ok := entry["type"].(string); ok && entryType == "system" {
				if subtype, ok := entry["subtype"].(string); ok && subtype == "init" {
					if mcpServers, ok := entry["mcp_servers"].([]any); ok {
						for _, serverInterface := range mcpServers {
							if server, ok := serverInterface.(map[string]any); ok {
								serverName, hasName := server["name"].(string)
								status, hasStatus := server["status"].(string)

								if hasName && hasStatus && status == "failed" {
									failure := MCPFailureReport{
										ServerName:   serverName,
										Status:       status,
										WorkflowName: run.WorkflowName,
										RunID:        run.DatabaseID,
									}

									// Try to extract timestamp if available
									if timestamp, hasTimestamp := entry["timestamp"].(string); hasTimestamp {
										failure.Timestamp = timestamp
									}

									mcpFailures = append(mcpFailures, failure)

									if verbose {
										fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Found MCP server failure: %s (status: %s)", serverName, status)))
									}
								}
							}
						}
					}
				}
			}
		}
	} else {
		// Fallback: Try to parse as JSON lines (Claude logs are typically NDJSON format)
		lines := strings.Split(logContent, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "{") {
				continue
			}

			// Try to parse each line as JSON
			var entry map[string]any
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue // Skip non-JSON lines
			}

			// Look for system init entries that contain MCP server information
			if entryType, ok := entry["type"].(string); ok && entryType == "system" {
				if subtype, ok := entry["subtype"].(string); ok && subtype == "init" {
					if mcpServers, ok := entry["mcp_servers"].([]any); ok {
						for _, serverInterface := range mcpServers {
							if server, ok := serverInterface.(map[string]any); ok {
								serverName, hasName := server["name"].(string)
								status, hasStatus := server["status"].(string)

								if hasName && hasStatus && status == "failed" {
									failure := MCPFailureReport{
										ServerName:   serverName,
										Status:       status,
										WorkflowName: run.WorkflowName,
										RunID:        run.DatabaseID,
									}

									// Try to extract timestamp if available
									if timestamp, hasTimestamp := entry["timestamp"].(string); hasTimestamp {
										failure.Timestamp = timestamp
									}

									mcpFailures = append(mcpFailures, failure)

									if verbose {
										fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Found MCP server failure: %s (status: %s)", serverName, status)))
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return mcpFailures, nil
}

// MCPFailureSummary aggregates MCP server failures across runs
type MCPFailureSummary struct {
	ServerName string
	Count      int
	Workflows  []string // List of workflow names that had this server fail
	RunIDs     []int64  // List of run IDs where this server failed
}

// displayMCPFailuresAnalysis displays a summary of MCP server failures across all runs
func displayMCPFailuresAnalysis(processedRuns []ProcessedRun, verbose bool) {
	// Aggregate MCP failures across all runs
	failureSummary := make(map[string]*MCPFailureSummary)
	var totalFailures int

	for _, pr := range processedRuns {
		for _, failure := range pr.MCPFailures {
			totalFailures++
			if summary, exists := failureSummary[failure.ServerName]; exists {
				summary.Count++
				// Add workflow if not already in the list
				found := false
				for _, wf := range summary.Workflows {
					if wf == failure.WorkflowName {
						found = true
						break
					}
				}
				if !found {
					summary.Workflows = append(summary.Workflows, failure.WorkflowName)
				}
				summary.RunIDs = append(summary.RunIDs, failure.RunID)
			} else {
				failureSummary[failure.ServerName] = &MCPFailureSummary{
					ServerName: failure.ServerName,
					Count:      1,
					Workflows:  []string{failure.WorkflowName},
					RunIDs:     []int64{failure.RunID},
				}
			}
		}
	}

	if totalFailures == 0 {
		return // No MCP failures to display
	}

	// Convert map to slice for sorting
	var summaries []*MCPFailureSummary
	for _, summary := range failureSummary {
		summaries = append(summaries, summary)
	}

	// Sort by count (descending)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Count > summaries[j].Count
	})

	// Display summary table
	headers := []string{"MCP Server", "Failures", "Workflows", "Run IDs"}
	var rows [][]string

	for _, summary := range summaries {
		workflowList := strings.Join(summary.Workflows, ", ")
		if len(workflowList) > 30 {
			workflowList = workflowList[:27] + "..."
		}

		runIDStrs := make([]string, len(summary.RunIDs))
		for i, runID := range summary.RunIDs {
			runIDStrs[i] = fmt.Sprintf("%d", runID)
		}
		runIDList := strings.Join(runIDStrs, ", ")
		if len(runIDList) > 40 {
			runIDList = runIDList[:37] + "..."
		}

		rows = append(rows, []string{
			summary.ServerName,
			fmt.Sprintf("%d", summary.Count),
			workflowList,
			runIDList,
		})
	}

	tableConfig := console.TableConfig{
		Title:   "MCP Server Failures",
		Headers: headers,
		Rows:    rows,
	}

	fmt.Print(console.RenderTable(tableConfig))

	// Verbose mode: Show detailed breakdown by workflow
	if verbose && totalFailures > 0 {
		displayDetailedMCPFailuresBreakdown(processedRuns)
	}
}

// displayDetailedMCPFailuresBreakdown shows MCP failures organized by workflow (verbose mode)
func displayDetailedMCPFailuresBreakdown(processedRuns []ProcessedRun) {
	fmt.Printf("\n%s\n", console.FormatListHeader("🔍 Detailed MCP Failures Breakdown"))

	for _, pr := range processedRuns {
		if len(pr.MCPFailures) == 0 {
			continue
		}

		fmt.Printf("\n%s (Run %d) - %d failed MCP servers:\n",
			console.FormatInfoMessage(pr.Run.WorkflowName),
			pr.Run.DatabaseID,
			len(pr.MCPFailures))

		for i, failure := range pr.MCPFailures {
			fmt.Printf("  %d. %s %s\n",
				i+1,
				console.FormatListItem(failure.ServerName),
				console.FormatErrorMessage(fmt.Sprintf("- Status: %s", failure.Status)))

			if failure.Timestamp != "" {
				fmt.Printf("     %s %s\n",
					console.FormatVerboseMessage("Failed at:"),
					failure.Timestamp)
			}
		}
	}
}
