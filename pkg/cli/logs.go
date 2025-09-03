package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	LogsPath      string
}

// LogMetrics represents extracted metrics from log files
// This is now an alias to the shared type in workflow package
type LogMetrics = workflow.LogMetrics

// ProcessedRun represents a workflow run with its associated analysis
type ProcessedRun struct {
	Run            WorkflowRun
	AccessAnalysis *DomainAnalysis
}

// ErrNoArtifacts indicates that a workflow run has no artifacts
var ErrNoArtifacts = errors.New("no artifacts found for this run")

// DownloadResult represents the result of downloading artifacts for a single run
type DownloadResult struct {
	Run            WorkflowRun
	Metrics        LogMetrics
	AccessAnalysis *DomainAnalysis
	Error          error
	Skipped        bool
	LogsPath       string
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
- aw_output.txt: Agent's final output content (available when non-empty)
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
			run.LogsPath = result.LogsPath

			// Store access analysis for later display (we'll access it via the result)
			// No need to modify the WorkflowRun struct for this

			// Always use GitHub API timestamps for duration calculation
			if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
				run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			}

			processedRun := ProcessedRun{
				Run:            run,
				AccessAnalysis: result.AccessAnalysis,
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
	displayLogsOverview(workflowRuns, outputDir)

	// Display access log analysis
	displayAccessLogAnalysis(processedRuns, verbose)

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
	var detectedEngine workflow.AgenticEngine
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

	// Check for aw_output.txt artifact file
	awOutputPath := filepath.Join(logDir, "aw_output.txt")
	if _, err := os.Stat(awOutputPath); err == nil {
		if verbose {
			// Report that the agentic output file was found
			fileInfo, statErr := os.Stat(awOutputPath)
			if statErr == nil {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found agentic output file: aw_output.txt (%s)", formatFileSize(fileInfo.Size()))))
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
		}

		return nil
	})

	return metrics, err
}

// extractEngineFromAwInfo reads aw_info.json and returns the appropriate engine
// Handles cases where aw_info.json is a file or a directory containing the actual file
func extractEngineFromAwInfo(infoFilePath string, verbose bool) workflow.AgenticEngine {
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
func parseLogFileWithEngine(filePath string, detectedEngine workflow.AgenticEngine, verbose bool) (LogMetrics, error) {
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
func displayLogsOverview(runs []WorkflowRun, outputDir string) {
	if len(runs) == 0 {
		return
	}

	// Prepare table data
	headers := []string{"Run ID", "Workflow", "Status", "Duration", "Tokens", "Cost ($)", "Created", "Logs Path"}
	var rows [][]string

	var totalTokens int
	var totalCost float64
	var totalDuration time.Duration

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
