package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/workflow"
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
	LogsPath      string
}

// LogMetrics represents extracted metrics from log files
// This is now an alias to the shared type in workflow package
type LogMetrics = workflow.LogMetrics

// ProcessedRun represents a workflow run with its associated analysis
type ProcessedRun struct {
	Run                   WorkflowRun
	AccessAnalysis        *DomainAnalysis
	MissingTools          []MissingToolReport
	SanitizationAnalysis  *SanitizationAnalysis
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

// MissingToolSummary aggregates missing tool reports across runs
type MissingToolSummary struct {
	Tool        string
	Count       int
	Workflows   []string // List of workflow names that reported this tool
	FirstReason string   // Reason from the first occurrence
	RunIDs      []int64  // List of run IDs where this tool was reported
}

// SanitizationChange represents a specific change made during output sanitization
type SanitizationChange struct {
	Type        string `json:"type"`        // "mention", "bot_trigger", "url", "truncation", "xml_escape", "ansi_removal", "control_char"
	Original    string `json:"original"`    // Original text that was changed
	Sanitized   string `json:"sanitized"`   // Text after sanitization
	Context     string `json:"context"`     // Surrounding context for the change
	LineNumber  int    `json:"line_number"` // Line number where the change occurred
	Description string `json:"description"` // Human-readable description of the change
}

// SanitizationAnalysis represents the analysis of sanitization changes for a workflow run
type SanitizationAnalysis struct {
	HasRawOutput       bool                  `json:"has_raw_output"`       // Whether raw output was available for comparison
	HasSanitizedOutput bool                  `json:"has_sanitized_output"` // Whether sanitized output was available
	TotalChanges       int                   `json:"total_changes"`        // Total number of sanitization changes detected
	ChangesByType      map[string]int        `json:"changes_by_type"`      // Count of changes by type
	Changes            []SanitizationChange  `json:"changes"`              // Detailed list of changes
	WasContentTruncated bool                 `json:"was_content_truncated"` // Whether content was truncated due to size/lines
	TruncationReason   string                `json:"truncation_reason"`    // Reason for truncation if applicable
	SummaryPreview     string                `json:"summary_preview"`      // Brief preview of major changes for display
}

// ErrNoArtifacts indicates that a workflow run has no artifacts
var ErrNoArtifacts = errors.New("no artifacts found for this run")

// DownloadResult represents the result of downloading artifacts for a single run
type DownloadResult struct {
	Run                  WorkflowRun
	Metrics              LogMetrics
	AccessAnalysis       *DomainAnalysis
	MissingTools         []MissingToolReport
	SanitizationAnalysis *SanitizationAnalysis
	Error                error
	Skipped              bool
	LogsPath             string
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
		Short: "Download and analyze agentic workflow logs with aggregated metrics and sanitization analysis",
		Long: `Download workflow run logs and artifacts from GitHub Actions for agentic workflows.

This command fetches workflow runs, downloads their artifacts, and extracts them into
organized folders named by run ID. It also provides an overview table with aggregate
metrics including duration, token usage, and cost information.

Additionally, when both raw and sanitized output files are available, the command analyzes
what changes were made during output sanitization (e.g., @mentions neutralized, URLs redacted,
content truncated) and provides a detailed report of these security modifications.

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
  ` + constants.CLIExtensionPrefix + ` logs -o ./my-logs              # Custom output directory`,
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
			verbose, _ := cmd.Flags().GetBool("verbose")

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

			if err := DownloadWorkflowLogs(workflowName, count, startDate, endDate, outputDir, engine, verbose); err != nil {
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
	logsCmd.Flags().String("engine", "", "Filter logs by agentic engine type (claude, codex)")
	logsCmd.Flags().BoolP("verbose", "v", false, "Show individual tool names instead of grouping by MCP server")

	return logsCmd
}

// DownloadWorkflowLogs downloads and analyzes workflow logs with metrics
func DownloadWorkflowLogs(workflowName string, count int, startDate, endDate, outputDir, engine string, verbose bool) error {
	if verbose {
		fmt.Println(console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}

	var processedRuns []ProcessedRun
	var beforeDate string
	iteration := 0

	// Iterative algorithm: keep fetching runs until we have enough with artifacts
	for len(processedRuns) < count && iteration < MaxIterations {
		iteration++

		if verbose && iteration > 1 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Need %d more runs with artifacts, fetching more...", iteration, count-len(processedRuns))))
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

		runs, err := listWorkflowRunsWithPagination(workflowName, batchSize, startDate, endDate, beforeDate, verbose)
		if err != nil {
			return err
		}

		if len(runs) == 0 {
			if verbose {
				fmt.Println(console.FormatInfoMessage("No more workflow runs found, stopping iteration"))
			}
			break
		}

		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d workflow runs in batch %d", len(runs), iteration)))
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
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", result.Run.DatabaseID, result.Error)))
					}
				}
				continue
			}

			if result.Error != nil {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", result.Run.DatabaseID, result.Error)))
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
					for _, supportedEngine := range []string{"claude", "codex"} {
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
							for _, supportedEngine := range []string{"claude", "codex"} {
								if testEngine, err := registry.GetEngine(supportedEngine); err == nil && testEngine == detectedEngine {
									engineName = supportedEngine
									break
								}
							}
						}
						fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: engine '%s' does not match filter '%s'", result.Run.DatabaseID, engineName, engine)))
					}
					continue
				}
			}

			// Update run with metrics and path
			run := result.Run
			run.TokenUsage = result.Metrics.TokenUsage
			run.EstimatedCost = result.Metrics.EstimatedCost
			run.Turns = result.Metrics.Turns
			run.LogsPath = result.LogsPath

			// Store access analysis for later display (we'll access it via the result)
			// No need to modify the WorkflowRun struct for this

			// Always use GitHub API timestamps for duration calculation
			if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
				run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			}

			processedRun := ProcessedRun{
				Run:                  run,
				AccessAnalysis:       result.AccessAnalysis,
				MissingTools:         result.MissingTools,
				SanitizationAnalysis: result.SanitizationAnalysis,
			}
			processedRuns = append(processedRuns, processedRun)
			batchProcessed++
		}

		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d/%d)", batchProcessed, iteration, len(processedRuns), count)))
		}

		// Prepare for next iteration: set beforeDate to the oldest run from this batch
		if len(runs) > 0 {
			oldestRun := runs[len(runs)-1] // runs are typically ordered by creation date descending
			beforeDate = oldestRun.CreatedAt.Format(time.RFC3339)
		}

		// If we got fewer runs than requested in this batch, we've likely hit the end
		if len(runs) < batchSize {
			if verbose {
				fmt.Println(console.FormatInfoMessage("Received fewer runs than requested, likely reached end of available runs"))
			}
			break
		}
	}

	// Check if we hit the maximum iterations limit
	if iteration >= MaxIterations && len(processedRuns) < count {
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts out of %d requested", MaxIterations, len(processedRuns), count)))
	}

	if len(processedRuns) == 0 {
		fmt.Println(console.FormatWarningMessage("No workflow runs with artifacts found matching the specified criteria"))
		return nil
	}

	// Display overview table
	workflowRuns := make([]WorkflowRun, len(processedRuns))
	for i, pr := range processedRuns {
		workflowRuns[i] = pr.Run
	}
	displayLogsOverview(workflowRuns)

	// Display tool call report
	displayToolCallReport(processedRuns, verbose)

	// Display access log analysis
	displayAccessLogAnalysis(processedRuns, verbose)

	// Display missing tools analysis
	displayMissingToolsAnalysis(processedRuns, verbose)

	// Display sanitization analysis
	displaySanitizationAnalysis(processedRuns, verbose)

	// Display logs location prominently
	absOutputDir, _ := filepath.Abs(outputDir)
	fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Downloaded %d logs to %s", len(processedRuns), absOutputDir)))
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
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Processing %d runs in parallel...", len(actualRuns))))
	}

	// Use conc pool for controlled concurrency with results
	p := pool.NewWithResults[DownloadResult]().WithMaxGoroutines(MaxConcurrentDownloads)

	// Process each run concurrently
	for _, run := range actualRuns {
		run := run // capture loop variable
		p.Go(func() DownloadResult {
			if verbose {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Processing run %d (%s)...", run.DatabaseID, run.Status)))
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
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to extract metrics for run %d: %v", run.DatabaseID, metricsErr)))
					}
					// Don't fail the whole download for metrics errors
					metrics = LogMetrics{}
				}
				result.Metrics = metrics

				// Analyze access logs if available
				accessAnalysis, accessErr := analyzeAccessLogs(runOutputDir, verbose)
				if accessErr != nil {
					if verbose {
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to analyze access logs for run %d: %v", run.DatabaseID, accessErr)))
					}
				}
				result.AccessAnalysis = accessAnalysis

				// Extract missing tools if available
				missingTools, missingErr := extractMissingToolsFromRun(runOutputDir, run, verbose)
				if missingErr != nil {
					if verbose {
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing tools for run %d: %v", run.DatabaseID, missingErr)))
					}
				}
				result.MissingTools = missingTools

				// Analyze sanitization changes if both raw and sanitized outputs are available
				sanitizationAnalysis, sanitizationErr := analyzeSanitizationChanges(runOutputDir, run, verbose)
				if sanitizationErr != nil {
					if verbose {
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to analyze sanitization changes for run %d: %v", run.DatabaseID, sanitizationErr)))
					}
				}
				result.SanitizationAnalysis = sanitizationAnalysis
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
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Completed parallel processing: %d successful, %d total", successCount, len(results))))
	}

	return results
}

// listWorkflowRunsWithPagination fetches workflow runs from GitHub with pagination support
func listWorkflowRunsWithPagination(workflowName string, count int, startDate, endDate, beforeDate string, verbose bool) ([]WorkflowRun, error) {
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
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found agentic output file: safe_output.jsonl (%s)", formatFileSize(fileInfo.Size()))))
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
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found git patch file: aw.patch (%s)", formatFileSize(fileInfo.Size()))))
			}
		}
	}

	// Check for agent_output.json artifact (some workflows may store this under a nested directory)
	agentOutputPath, agentOutputFound := findAgentOutputFile(logDir)
	if agentOutputFound {
		if verbose {
			fileInfo, statErr := os.Stat(agentOutputPath)
			if statErr == nil {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found agent output file: %s (%s)", filepath.Base(agentOutputPath), formatFileSize(fileInfo.Size()))))
			}
		}
		// If the file is not already in the logDir root, copy it for convenience
		if filepath.Dir(agentOutputPath) != logDir {
			rootCopy := filepath.Join(logDir, "agent_output.json")
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

		// Process log files
		if strings.HasSuffix(strings.ToLower(info.Name()), ".log") ||
			strings.HasSuffix(strings.ToLower(info.Name()), ".txt") ||
			strings.Contains(strings.ToLower(info.Name()), "log") {

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
		}

		return nil
	})

	return metrics, err
}

// extractEngineFromAwInfo reads aw_info.json and returns the appropriate engine
// Handles cases where aw_info.json is a file or a directory containing the actual file
func extractEngineFromAwInfo(infoFilePath string, verbose bool) workflow.CodingAgentEngine {
	var data []byte
	var err error

	// Check if the path exists and determine if it's a file or directory
	stat, statErr := os.Stat(infoFilePath)
	if statErr != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to stat aw_info.json: %v", statErr)))
		}
		return nil
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
		return nil
	}

	var info map[string]interface{}
	if err := json.Unmarshal(data, &info); err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to parse aw_info.json: %v", err)))
		}
		return nil
	}

	engineID, ok := info["engine_id"].(string)
	if !ok || engineID == "" {
		if verbose {
			fmt.Println(console.FormatWarningMessage("No engine_id found in aw_info.json"))
		}
		return nil
	}

	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine(engineID)
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Unknown engine in aw_info.json: %s", engineID)))
		}
		return nil
	}

	return engine
}

// parseLogFileWithEngine parses a log file using a specific engine or falls back to auto-detection
func parseLogFileWithEngine(filePath string, detectedEngine workflow.CodingAgentEngine, verbose bool) (LogMetrics, error) {
	// Read the log file content
	file, err := os.Open(filePath)
	if err != nil {
		return LogMetrics{}, fmt.Errorf("error opening log file: %w", err)
	}
	defer file.Close()

	var content []byte
	buffer := make([]byte, 4096)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return LogMetrics{}, fmt.Errorf("error reading log file: %w", err)
		}
		if n == 0 {
			break
		}
		content = append(content, buffer[:n]...)
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
	headers := []string{"Run ID", "Workflow", "Status", "Duration", "Tokens", "Cost ($)", "Turns", "Created", "Logs Path"}
	var rows [][]string

	var totalTokens int
	var totalCost float64
	var totalDuration time.Duration
	var totalTurns int

	for _, run := range runs {
		// Format duration
		durationStr := "N/A"
		if run.Duration > 0 {
			durationStr = formatDuration(run.Duration)
			totalDuration += run.Duration
		}

		// Format cost
		costStr := "N/A"
		if run.EstimatedCost > 0 {
			costStr = fmt.Sprintf("%.3f", run.EstimatedCost)
			totalCost += run.EstimatedCost
		}

		// Format tokens
		tokensStr := "N/A"
		if run.TokenUsage > 0 {
			tokensStr = formatNumber(run.TokenUsage)
			totalTokens += run.TokenUsage
		}

		// Format turns
		turnsStr := "N/A"
		if run.Turns > 0 {
			turnsStr = fmt.Sprintf("%d", run.Turns)
			totalTurns += run.Turns
		}

		// Truncate workflow name if too long
		workflowName := run.WorkflowName
		if len(workflowName) > 20 {
			workflowName = workflowName[:17] + "..."
		}

		// Format relative path
		relPath, _ := filepath.Rel(".", run.LogsPath)

		row := []string{
			fmt.Sprintf("%d", run.DatabaseID),
			workflowName,
			run.Status,
			durationStr,
			tokensStr,
			costStr,
			turnsStr,
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
		logMetrics := extractLogMetricsFromRun(processedRun)

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
			} else {
				toolStats[displayKey] = &workflow.ToolCallInfo{
					Name:          displayKey,
					CallCount:     toolCall.CallCount,
					MaxOutputSize: toolCall.MaxOutputSize,
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
	headers := []string{"Tool", "Calls", "Max Output (tokens)"}
	var rows [][]string

	for _, toolCall := range toolCalls {
		outputStr := "N/A"
		if toolCall.MaxOutputSize > 0 {
			outputStr = formatNumber(toolCall.MaxOutputSize)
		}

		row := []string{
			toolCall.Name,
			fmt.Sprintf("%d", toolCall.CallCount),
			outputStr,
		}
		rows = append(rows, row)
	}

	// Render compact table without title as requested
	tableConfig := console.TableConfig{
		Headers:   headers,
		Rows:      rows,
		ShowTotal: false, // Keep it simple and compact
	}

	fmt.Print(console.RenderTable(tableConfig))
}

// extractLogMetricsFromRun extracts log metrics from a processed run's log directory
func extractLogMetricsFromRun(processedRun ProcessedRun) workflow.LogMetrics {
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

// formatFileSize formats file sizes in a human-readable way (e.g., "1.2 KB", "3.4 MB")
func formatFileSize(size int64) string {
	if size == 0 {
		return "0 B"
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(units) {
		exp = len(units) - 1
		div = int64(1) << (10 * (exp + 1))
	}

	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}

// findAgentOutputFile searches for a file named agent_output.json within the logDir tree.
// Returns the first path found (depth-first) and a boolean indicating success.
func findAgentOutputFile(logDir string) (string, bool) {
	var foundPath string
	_ = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), "agent_output.json") {
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
	agentOutputPath := filepath.Join(runDir, "agent_output.json")
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
	fmt.Printf("%s\n\n", console.FormatListHeader("======================="))

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
	fmt.Printf("%s\n", console.FormatListHeader("===================================="))

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

// analyzeSanitizationChanges analyzes changes made during output sanitization by comparing raw and sanitized outputs
func analyzeSanitizationChanges(runDir string, run WorkflowRun, verbose bool) (*SanitizationAnalysis, error) {
	analysis := &SanitizationAnalysis{
		ChangesByType: make(map[string]int),
	}

	// Check for raw output (agent_output.json)
	rawOutputPath := filepath.Join(runDir, "agent_output.json")
	rawOutputExists := false
	if _, err := os.Stat(rawOutputPath); err == nil {
		analysis.HasRawOutput = true
		rawOutputExists = true
	}

	// Check for sanitized output (safe_output.jsonl)
	sanitizedOutputPath := filepath.Join(runDir, "safe_output.jsonl")
	if _, err := os.Stat(sanitizedOutputPath); err == nil {
		analysis.HasSanitizedOutput = true
	}

	// If we don't have both outputs, we can't do a comparison
	if !rawOutputExists || !analysis.HasSanitizedOutput {
		if verbose && rawOutputExists && !analysis.HasSanitizedOutput {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Run %d has raw output but no sanitized output for comparison", run.DatabaseID)))
		}
		if verbose && !rawOutputExists && analysis.HasSanitizedOutput {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Run %d has sanitized output but no raw output for comparison", run.DatabaseID)))
		}
		return analysis, nil
	}

	// Read raw output
	rawContent, err := readOutputContent(rawOutputPath)
	if err != nil {
		return analysis, fmt.Errorf("failed to read raw output: %w", err)
	}

	// Read sanitized output
	sanitizedContent, err := readOutputContent(sanitizedOutputPath)
	if err != nil {
		return analysis, fmt.Errorf("failed to read sanitized output: %w", err)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Comparing outputs for run %d: raw=%d chars, sanitized=%d chars", run.DatabaseID, len(rawContent), len(sanitizedContent))))
	}

	// Analyze differences
	changes := detectSanitizationChanges(rawContent, sanitizedContent)
	analysis.Changes = changes
	analysis.TotalChanges = len(changes)

	// Count changes by type
	for _, change := range changes {
		analysis.ChangesByType[change.Type]++
	}

	// Check for content truncation
	if strings.Contains(sanitizedContent, "[Content truncated due to length]") {
		analysis.WasContentTruncated = true
		analysis.TruncationReason = "length"
	} else if strings.Contains(sanitizedContent, "[Content truncated due to line count]") {
		analysis.WasContentTruncated = true
		analysis.TruncationReason = "line_count"
	}

	// Generate summary preview
	analysis.SummaryPreview = generateSanitizationSummary(analysis)

	return analysis, nil
}
// readOutputContent reads content from an output file, handling different formats
func readOutputContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Handle JSONL format (safe_output.jsonl)
	if strings.HasSuffix(filePath, ".jsonl") {
		return extractContentFromJSONL(string(content))
	}

	// Handle JSON format (agent_output.json)
	if strings.HasSuffix(filePath, ".json") {
		return extractContentFromJSON(string(content))
	}

	// Default: return content as-is
	return string(content), nil
}

// extractContentFromJSONL extracts text content from JSONL format
func extractContentFromJSONL(jsonlContent string) (string, error) {
	var result strings.Builder
	lines := strings.Split(jsonlContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue // Skip invalid JSON lines
		}

		// Extract text content based on item type
		if itemType, ok := item["type"].(string); ok {
			if itemType == "text" {
				if text, ok := item["text"].(string); ok {
					result.WriteString(text)
				}
			}
		}
	}

	return result.String(), nil
}

// extractContentFromJSON extracts text content from JSON format
func extractContentFromJSON(jsonContent string) (string, error) {
	// Try to parse as structured JSON first
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err == nil {
		// Look for common fields that might contain the text content
		if content, ok := data["content"].(string); ok {
			return content, nil
		}
		if output, ok := data["output"].(string); ok {
			return output, nil
		}
		if text, ok := data["text"].(string); ok {
			return text, nil
		}
		// If it's structured JSON with items array (like safe_output format)
		if items, ok := data["items"].([]interface{}); ok {
			var result strings.Builder
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if itemType, ok := itemMap["type"].(string); ok && itemType == "text" {
						if text, ok := itemMap["text"].(string); ok {
							result.WriteString(text)
						}
					}
				}
			}
			return result.String(), nil
		}
	}

	// If not structured, treat the entire content as text
	return jsonContent, nil
}

// detectSanitizationChanges compares raw and sanitized content to detect changes
func detectSanitizationChanges(rawContent, sanitizedContent string) []SanitizationChange {
	var changes []SanitizationChange

	// Split into lines for line-by-line comparison
	rawLines := strings.Split(rawContent, "\n")
	sanitizedLines := strings.Split(sanitizedContent, "\n")

	// Check for truncation first
	if len(sanitizedLines) > 0 {
		lastLine := sanitizedLines[len(sanitizedLines)-1]
		if strings.Contains(lastLine, "[Content truncated due to") {
			changes = append(changes, SanitizationChange{
				Type:        "truncation",
				Original:    "", // Not applicable for truncation
				Sanitized:   lastLine,
				Context:     "End of content",
				LineNumber:  len(sanitizedLines),
				Description: "Content was truncated due to size limits",
			})
		}
	}

	// Compare line by line to detect changes
	minLines := len(rawLines)
	if len(sanitizedLines) < minLines {
		minLines = len(sanitizedLines)
	}

	for i := 0; i < minLines; i++ {
		rawLine := rawLines[i]
		sanitizedLine := sanitizedLines[i]

		if rawLine != sanitizedLine {
			// Detect specific types of changes
			lineChanges := detectLineChanges(rawLine, sanitizedLine, i+1)
			changes = append(changes, lineChanges...)
		}
	}

	return changes
}

// detectLineChanges detects specific sanitization changes within a line
func detectLineChanges(rawLine, sanitizedLine string, lineNumber int) []SanitizationChange {
	var changes []SanitizationChange

	// Detect @mention changes (e.g., @user -> `@user`)
	mentionChanges := detectMentionChanges(rawLine, sanitizedLine, lineNumber)
	changes = append(changes, mentionChanges...)

	// Detect bot trigger changes (e.g., fixes #123 -> `fixes #123`)
	botTriggerChanges := detectBotTriggerChanges(rawLine, sanitizedLine, lineNumber)
	changes = append(changes, botTriggerChanges...)

	// Detect URL redaction changes (e.g., http://example.com -> (redacted))
	urlChanges := detectURLChanges(rawLine, sanitizedLine, lineNumber)
	changes = append(changes, urlChanges...)

	// Detect XML escaping changes (e.g., <script> -> &lt;script&gt;)
	xmlChanges := detectXMLEscapingChanges(rawLine, sanitizedLine, lineNumber)
	changes = append(changes, xmlChanges...)

	// Detect ANSI escape sequence removal
	ansiChanges := detectANSIChanges(rawLine, sanitizedLine, lineNumber)
	changes = append(changes, ansiChanges...)

	return changes
}

// detectMentionChanges detects changes to @mentions
func detectMentionChanges(rawLine, sanitizedLine string, lineNumber int) []SanitizationChange {
	var changes []SanitizationChange

	// Pattern to match @mentions that got wrapped in backticks
	mentionRegex := regexp.MustCompile(`@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:/[A-Za-z0-9._-]+)?)`)
	
	rawMentions := mentionRegex.FindAllString(rawLine, -1)
	for _, mention := range rawMentions {
		backtickMention := "`" + mention + "`"
		if strings.Contains(sanitizedLine, backtickMention) && !strings.Contains(rawLine, backtickMention) {
			changes = append(changes, SanitizationChange{
				Type:        "mention",
				Original:    mention,
				Sanitized:   backtickMention,
				Context:     getContext(rawLine, mention),
				LineNumber:  lineNumber,
				Description: fmt.Sprintf("@mention '%s' was neutralized with backticks", mention),
			})
		}
	}

	return changes
}

// detectBotTriggerChanges detects changes to bot trigger phrases
func detectBotTriggerChanges(rawLine, sanitizedLine string, lineNumber int) []SanitizationChange {
	var changes []SanitizationChange

	// Pattern to match bot trigger phrases that got wrapped in backticks
	triggerRegex := regexp.MustCompile(`(?i)\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)`)
	
	matches := triggerRegex.FindAllString(rawLine, -1)
	for _, trigger := range matches {
		backtickTrigger := "`" + trigger + "`"
		if strings.Contains(sanitizedLine, backtickTrigger) && !strings.Contains(rawLine, backtickTrigger) {
			changes = append(changes, SanitizationChange{
				Type:        "bot_trigger",
				Original:    trigger,
				Sanitized:   backtickTrigger,
				Context:     getContext(rawLine, trigger),
				LineNumber:  lineNumber,
				Description: fmt.Sprintf("Bot trigger phrase '%s' was neutralized with backticks", trigger),
			})
		}
	}

	return changes
}

// detectURLChanges detects URL redaction and filtering
func detectURLChanges(rawLine, sanitizedLine string, lineNumber int) []SanitizationChange {
	var changes []SanitizationChange

	// Pattern to match URLs that might have been redacted
	urlRegex := regexp.MustCompile(`\b\w+://[^\s\])}'"<>&\x00-\x1f]+`)
	
	rawURLs := urlRegex.FindAllString(rawLine, -1)
	for _, url := range rawURLs {
		if !strings.Contains(sanitizedLine, url) && strings.Contains(sanitizedLine, "(redacted)") {
			changes = append(changes, SanitizationChange{
				Type:        "url",
				Original:    url,
				Sanitized:   "(redacted)",
				Context:     getContext(rawLine, url),
				LineNumber:  lineNumber,
				Description: fmt.Sprintf("URL '%s' was redacted due to protocol or domain restrictions", url),
			})
		}
	}

	return changes
}

// detectXMLEscapingChanges detects XML character escaping
func detectXMLEscapingChanges(rawLine, sanitizedLine string, lineNumber int) []SanitizationChange {
	var changes []SanitizationChange

	xmlChars := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&apos;",
	}

	for original, escaped := range xmlChars {
		if strings.Contains(rawLine, original) && strings.Contains(sanitizedLine, escaped) {
			// Count occurrences to avoid duplicate reporting
			escapedCount := strings.Count(sanitizedLine, escaped)
			
			if escapedCount > strings.Count(rawLine, escaped) {
				changes = append(changes, SanitizationChange{
					Type:        "xml_escape",
					Original:    original,
					Sanitized:   escaped,
					Context:     getContext(rawLine, original),
					LineNumber:  lineNumber,
					Description: fmt.Sprintf("XML character '%s' was escaped to '%s'", original, escaped),
				})
			}
		}
	}

	return changes
}

// detectANSIChanges detects ANSI escape sequence removal
func detectANSIChanges(rawLine, sanitizedLine string, lineNumber int) []SanitizationChange {
	var changes []SanitizationChange

	// Pattern to match ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mGKH]`)
	
	ansiSequences := ansiRegex.FindAllString(rawLine, -1)
	if len(ansiSequences) > 0 && !ansiRegex.MatchString(sanitizedLine) {
		for _, ansi := range ansiSequences {
			changes = append(changes, SanitizationChange{
				Type:        "ansi_removal",
				Original:    ansi,
				Sanitized:   "",
				Context:     getContext(rawLine, ansi),
				LineNumber:  lineNumber,
				Description: "ANSI escape sequence was removed",
			})
		}
	}

	return changes
}

// getContext returns surrounding context for a change
func getContext(line, target string) string {
	index := strings.Index(line, target)
	if index == -1 {
		return line[:minInt(50, len(line))] + "..."
	}

	start := maxInt(0, index-20)
	end := minInt(len(line), index+len(target)+20)
	
	context := line[start:end]
	if start > 0 {
		context = "..." + context
	}
	if end < len(line) {
		context = context + "..."
	}
	
	return context
}

// generateSanitizationSummary creates a brief summary of sanitization changes
func generateSanitizationSummary(analysis *SanitizationAnalysis) string {
	if analysis.TotalChanges == 0 {
		return "No sanitization changes detected"
	}

	var summary strings.Builder
	var parts []string

	for changeType, count := range analysis.ChangesByType {
		switch changeType {
		case "mention":
			parts = append(parts, fmt.Sprintf("%d @mention(s)", count))
		case "bot_trigger":
			parts = append(parts, fmt.Sprintf("%d bot trigger(s)", count))
		case "url":
			parts = append(parts, fmt.Sprintf("%d URL(s) redacted", count))
		case "xml_escape":
			parts = append(parts, fmt.Sprintf("%d XML character(s) escaped", count))
		case "ansi_removal":
			parts = append(parts, fmt.Sprintf("%d ANSI sequence(s) removed", count))
		case "truncation":
			parts = append(parts, "content truncated")
		}
	}

	summary.WriteString(strings.Join(parts, ", "))
	
	return summary.String()
}

// displaySanitizationAnalysis displays a summary of sanitization changes across all runs
func displaySanitizationAnalysis(processedRuns []ProcessedRun, verbose bool) {
	// Count runs with sanitization analysis
	var runsWithAnalysis []ProcessedRun
	var runsWithChanges []ProcessedRun
	totalChanges := 0
	changeTypeCount := make(map[string]int)

	for _, pr := range processedRuns {
		if pr.SanitizationAnalysis != nil {
			runsWithAnalysis = append(runsWithAnalysis, pr)
			if pr.SanitizationAnalysis.TotalChanges > 0 {
				runsWithChanges = append(runsWithChanges, pr)
				totalChanges += pr.SanitizationAnalysis.TotalChanges
				for changeType, count := range pr.SanitizationAnalysis.ChangesByType {
					changeTypeCount[changeType] += count
				}
			}
		}
	}

	if len(runsWithChanges) == 0 {
		if len(runsWithAnalysis) > 0 && verbose {
			fmt.Printf("\n%s\n", console.FormatListHeader("🔒 Output Sanitization Analysis"))
			fmt.Printf("%s\n", console.FormatListHeader("==============================="))
			fmt.Printf("\n📊 %s: No sanitization changes detected across %d runs with comparable outputs\n",
				console.FormatSuccessMessage("Result"),
				len(runsWithAnalysis))
		}
		return
	}

	// Display summary header
	fmt.Printf("\n%s\n", console.FormatListHeader("🔒 Output Sanitization Analysis"))
	fmt.Printf("%s\n\n", console.FormatListHeader("==============================="))

	// Display summary table
	headers := []string{"Run ID", "Workflow", "Changes", "Types", "Summary"}
	var rows [][]string

	for _, pr := range runsWithChanges {
		analysis := pr.SanitizationAnalysis
		
		// Format change types
		var changeTypes []string
		for changeType, count := range analysis.ChangesByType {
			changeTypes = append(changeTypes, fmt.Sprintf("%s(%d)", changeType, count))
		}
		sort.Strings(changeTypes) // Ensure consistent ordering
		changeTypesStr := strings.Join(changeTypes, ", ")
		if len(changeTypesStr) > 30 {
			changeTypesStr = changeTypesStr[:27] + "..."
		}

		// Truncate workflow name if too long
		workflowName := pr.Run.WorkflowName
		if len(workflowName) > 20 {
			workflowName = workflowName[:17] + "..."
		}

		// Truncate summary if too long
		summary := analysis.SummaryPreview
		if len(summary) > 40 {
			summary = summary[:37] + "..."
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", pr.Run.DatabaseID),
			workflowName,
			fmt.Sprintf("%d", analysis.TotalChanges),
			changeTypesStr,
			summary,
		})
	}

	tableConfig := console.TableConfig{
		Headers: headers,
		Rows:    rows,
	}

	fmt.Print(console.RenderTable(tableConfig))

	// Display total summary
	fmt.Printf("\n📊 %s: %d runs had sanitization changes (%d total changes across %d types)\n",
		console.FormatCountMessage("Summary"),
		len(runsWithChanges),
		totalChanges,
		len(changeTypeCount))

	// Display change type breakdown
	if len(changeTypeCount) > 0 {
		fmt.Printf("\n🔍 %s:\n", console.FormatInfoMessage("Change Type Breakdown"))
		
		// Sort change types by count (descending)
		type changeTypeStat struct {
			Type  string
			Count int
		}
		var stats []changeTypeStat
		for changeType, count := range changeTypeCount {
			stats = append(stats, changeTypeStat{Type: changeType, Count: count})
		}
		sort.Slice(stats, func(i, j int) bool {
			return stats[i].Count > stats[j].Count
		})

		for _, stat := range stats {
			var description string
			switch stat.Type {
			case "mention":
				description = "@mentions neutralized with backticks"
			case "bot_trigger":
				description = "bot trigger phrases neutralized"
			case "url":
				description = "URLs redacted due to protocol/domain restrictions"
			case "xml_escape":
				description = "XML characters escaped for safety"
			case "ansi_removal":
				description = "ANSI color codes removed"
			case "truncation":
				description = "content truncated due to size limits"
			default:
				description = "other sanitization changes"
			}
			fmt.Printf("  • %s - %s\n",
				console.FormatListItem(fmt.Sprintf("%d %s changes", stat.Count, stat.Type)),
				console.FormatVerboseMessage(description))
		}
	}

	// Verbose mode: Show detailed breakdown by run
	if verbose && len(runsWithChanges) > 0 {
		displayDetailedSanitizationBreakdown(runsWithChanges)
	}
}

// displayDetailedSanitizationBreakdown shows sanitization changes organized by run (verbose mode)
func displayDetailedSanitizationBreakdown(runsWithChanges []ProcessedRun) {
	fmt.Printf("\n%s\n", console.FormatListHeader("🔍 Detailed Sanitization Changes"))
	fmt.Printf("%s\n", console.FormatListHeader("==================================="))

	for _, pr := range runsWithChanges {
		analysis := pr.SanitizationAnalysis
		fmt.Printf("\n%s (Run %d) - %d changes:\n",
			console.FormatInfoMessage(pr.Run.WorkflowName),
			pr.Run.DatabaseID,
			analysis.TotalChanges)

		// Group changes by type for better organization
		changesByType := make(map[string][]SanitizationChange)
		for _, change := range analysis.Changes {
			changesByType[change.Type] = append(changesByType[change.Type], change)
		}

		changeTypeOrder := []string{"mention", "bot_trigger", "url", "xml_escape", "ansi_removal", "truncation"}
		
		for _, changeType := range changeTypeOrder {
			if changes, exists := changesByType[changeType]; exists {
				fmt.Printf("  %s (%d changes):\n", 
					console.FormatWarningMessage(strings.Title(strings.ReplaceAll(changeType, "_", " "))),
					len(changes))
				
				for i, change := range changes {
					if i >= 3 { // Limit to first 3 examples per type
						fmt.Printf("    %s\n", console.FormatVerboseMessage(fmt.Sprintf("... and %d more", len(changes)-3)))
						break
					}
					
					fmt.Printf("    %d. %s\n",
						i+1,
						console.FormatListItem(change.Description))
					
					if change.Original != "" && change.Sanitized != "" {
						fmt.Printf("       %s %s -> %s\n",
							console.FormatVerboseMessage("Change:"),
							console.FormatVerboseMessage(fmt.Sprintf("'%s'", truncateString(change.Original, 30))),
							console.FormatVerboseMessage(fmt.Sprintf("'%s'", truncateString(change.Sanitized, 30))))
					}
					
					if change.Context != "" {
						fmt.Printf("       %s %s\n",
							console.FormatVerboseMessage("Context:"),
							console.FormatVerboseMessage(truncateString(change.Context, 50)))
					}
					
					if change.LineNumber > 0 {
						fmt.Printf("       %s Line %d\n",
							console.FormatVerboseMessage("Location:"),
							change.LineNumber)
					}
				}
				fmt.Println()
			}
		}
		
		if analysis.WasContentTruncated {
			fmt.Printf("  %s Content was truncated due to %s limits\n",
				console.FormatWarningMessage("⚠️  Truncation:"),
				analysis.TruncationReason)
		}
	}
}

// Helper functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}