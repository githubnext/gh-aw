package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/workflow"
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
type LogMetrics struct {
	Duration      time.Duration
	TokenUsage    int
	EstimatedCost float64
	ErrorCount    int
	WarningCount  int
}

// JSONMetrics represents metrics extracted from JSON log entries
type JSONMetrics struct {
	TokenUsage    int
	EstimatedCost float64
	Timestamp     time.Time
}

// ErrNoArtifacts indicates that a workflow run has no artifacts
var ErrNoArtifacts = errors.New("no artifacts found for this run")

// Constants for the iterative algorithm
const (
	// MaxIterations limits how many batches we fetch to prevent infinite loops
	MaxIterations = 10
	// BatchSize is the number of runs to fetch in each iteration
	BatchSize = 20
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

The agentic-workflow-id is the basename of the markdown file without the .md extension.
For example, for 'weekly-research.md', use 'weekly-research' as the workflow ID.

Examples:
  ` + constants.CLIExtensionPrefix + ` logs                           # Download logs for all workflows
  ` + constants.CLIExtensionPrefix + ` logs weekly-research           # Download logs for specific agentic workflow
  ` + constants.CLIExtensionPrefix + ` logs -c 10                     # Download last 10 runs
  ` + constants.CLIExtensionPrefix + ` logs --start-date 2024-01-01   # Filter runs after date
  ` + constants.CLIExtensionPrefix + ` logs --end-date 2024-01-31     # Filter runs before date
  ` + constants.CLIExtensionPrefix + ` logs -o ./my-logs              # Custom output directory`,
		Run: func(cmd *cobra.Command, args []string) {
			var workflowName string
			if len(args) > 0 && args[0] != "" {
				// Convert agentic workflow ID to GitHub Actions workflow name
				resolvedName, err := workflow.ResolveWorkflowName(args[0])
				if err != nil {
					fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
						Type:    "error",
						Message: err.Error(),
					}))
					os.Exit(1)
				}
				workflowName = resolvedName
			}

			count, _ := cmd.Flags().GetInt("count")
			startDate, _ := cmd.Flags().GetString("start-date")
			endDate, _ := cmd.Flags().GetString("end-date")
			outputDir, _ := cmd.Flags().GetString("output")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if err := DownloadWorkflowLogs(workflowName, count, startDate, endDate, outputDir, verbose); err != nil {
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
	logsCmd.Flags().String("start-date", "", "Filter runs created after this date (YYYY-MM-DD)")
	logsCmd.Flags().String("end-date", "", "Filter runs created before this date (YYYY-MM-DD)")
	logsCmd.Flags().StringP("output", "o", "./logs", "Output directory for downloaded logs and artifacts")

	return logsCmd
}

// DownloadWorkflowLogs downloads and analyzes workflow logs with metrics
func DownloadWorkflowLogs(workflowName string, count int, startDate, endDate, outputDir string, verbose bool) error {
	if verbose {
		fmt.Println(console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}

	var processedRuns []WorkflowRun
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
		if count-len(processedRuns) < BatchSize {
			// If we need fewer runs than the batch size, request exactly what we need
			// but add some buffer since many runs might not have artifacts
			needed := count - len(processedRuns)
			batchSize = needed * 3 // Request 3x what we need to account for runs without artifacts
			if batchSize > BatchSize {
				batchSize = BatchSize
			}
		}

		runs, err := listWorkflowRunsWithPagination(workflowName, batchSize, startDate, endDate, beforeDate, verbose)
		if err != nil {
			return fmt.Errorf("failed to list workflow runs: %w", err)
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
		for _, run := range runs {
			// Stop if we've reached our target count
			if len(processedRuns) >= count {
				break
			}

			if verbose {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Processing run %d (%s)...", run.DatabaseID, run.Status)))
			}

			// Download artifacts and logs for this run
			runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", run.DatabaseID))
			if err := downloadRunArtifacts(run.DatabaseID, runOutputDir, verbose); err != nil {
				// Check if this is a "no artifacts" case - skip silently for cancelled/failed runs
				if errors.Is(err, ErrNoArtifacts) {
					if verbose {
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", run.DatabaseID, err)))
					}
					continue
				}
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", run.DatabaseID, err)))
				continue
			}

			// Extract metrics from logs
			metrics, err := extractLogMetrics(runOutputDir, verbose)
			if err != nil {
				if verbose {
					fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to extract metrics for run %d: %v", run.DatabaseID, err)))
				}
			}

			// Update run with metrics and path
			run.Duration = metrics.Duration
			run.TokenUsage = metrics.TokenUsage
			run.EstimatedCost = metrics.EstimatedCost
			run.LogsPath = runOutputDir

			// Calculate duration from GitHub data if not extracted from logs
			if run.Duration == 0 && !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
				run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			}

			processedRuns = append(processedRuns, run)
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
	displayLogsOverview(processedRuns, outputDir)

	// Display logs location prominently
	absOutputDir, _ := filepath.Abs(outputDir)
	fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Downloaded %d logs to %s", len(processedRuns), absOutputDir)))
	return nil
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
	output, err := cmd.Output()

	// Stop spinner
	if !verbose {
		spinner.Stop()
	}
	if err != nil {
		// Check for authentication errors
		if strings.Contains(err.Error(), "exit status 4") {
			return nil, fmt.Errorf("GitHub CLI authentication required. Run 'gh auth login' first")
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
		for _, run := range runs {
			if strings.HasSuffix(run.WorkflowName, ".lock.yml") || strings.Contains(run.WorkflowName, "agentic") ||
				strings.Contains(run.WorkflowName, "Agentic") || strings.Contains(run.WorkflowName, "@") {
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

			fileMetrics, err := parseLogFile(path, verbose)
			if err != nil && verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to parse log file %s: %v", path, err)))
				return nil // Continue processing other files
			}

			// Aggregate metrics
			metrics.TokenUsage += fileMetrics.TokenUsage
			metrics.EstimatedCost += fileMetrics.EstimatedCost
			metrics.ErrorCount += fileMetrics.ErrorCount
			metrics.WarningCount += fileMetrics.WarningCount

			if fileMetrics.Duration > metrics.Duration {
				metrics.Duration = fileMetrics.Duration
			}
		}

		return nil
	})

	return metrics, err
}

// parseLogFile parses a single log file and extracts metrics
func parseLogFile(filePath string, verbose bool) (LogMetrics, error) {
	var metrics LogMetrics
	var startTime, endTime time.Time
	var maxTokenUsage int

	file, err := os.Open(filePath)
	if err != nil {
		return metrics, err
	}
	defer file.Close()

	content := make([]byte, 0)
	buffer := make([]byte, 4096)
	for {
		n, err := file.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return metrics, err
		}
		if n == 0 {
			break
		}
		content = append(content, buffer[:n]...)
	}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Try to parse as streaming JSON first
		jsonMetrics := extractJSONMetrics(line, verbose)
		if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 || !jsonMetrics.Timestamp.IsZero() {
			// Successfully extracted from JSON, update metrics
			if jsonMetrics.TokenUsage > maxTokenUsage {
				maxTokenUsage = jsonMetrics.TokenUsage
			}
			if jsonMetrics.EstimatedCost > 0 {
				metrics.EstimatedCost += jsonMetrics.EstimatedCost
			}
			if !jsonMetrics.Timestamp.IsZero() {
				if startTime.IsZero() || jsonMetrics.Timestamp.Before(startTime) {
					startTime = jsonMetrics.Timestamp
				}
				if endTime.IsZero() || jsonMetrics.Timestamp.After(endTime) {
					endTime = jsonMetrics.Timestamp
				}
			}
			continue
		}

		// Fall back to text pattern extraction
		// Extract timestamps for duration calculation
		timestamp := extractTimestamp(line)
		if !timestamp.IsZero() {
			if startTime.IsZero() || timestamp.Before(startTime) {
				startTime = timestamp
			}
			if endTime.IsZero() || timestamp.After(endTime) {
				endTime = timestamp
			}
		}

		// Extract token usage - keep the maximum found
		tokenUsage := extractTokenUsage(line)
		if tokenUsage > maxTokenUsage {
			maxTokenUsage = tokenUsage
		}

		// Extract cost information
		cost := extractCost(line)
		if cost > 0 {
			metrics.EstimatedCost += cost
		}

		// Count errors and warnings
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") {
			metrics.ErrorCount++
		}
		if strings.Contains(lowerLine, "warning") {
			metrics.WarningCount++
		}
	}

	// Set the max token usage found
	metrics.TokenUsage = maxTokenUsage

	// Calculate duration
	if !startTime.IsZero() && !endTime.IsZero() {
		metrics.Duration = endTime.Sub(startTime)
	}

	return metrics, nil
}

// extractTimestamp extracts timestamp from log line
func extractTimestamp(line string) time.Time {
	// Common timestamp patterns
	patterns := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05", // Codex format without Z
		"2006-01-02 15:04:05",
		"Jan 02 15:04:05",
	}

	// First try to extract the timestamp string from the line
	// Updated regex to handle timestamps both with and without Z, and in brackets
	timestampRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})Z?`)
	matches := timestampRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		timestampStr := matches[1]
		for _, pattern := range patterns {
			if t, err := time.Parse(pattern, timestampStr); err == nil {
				return t
			}
		}
	}

	return time.Time{}
}

// extractTokenUsage extracts token usage from log line
func extractTokenUsage(line string) int {
	// Look for patterns like "tokens: 1234", "token_count: 1234", etc.
	patterns := []string{
		`tokens?[:\s]+(\d+)`,
		`token[_\s]count[:\s]+(\d+)`,
		`input[_\s]tokens[:\s]+(\d+)`,
		`output[_\s]tokens[:\s]+(\d+)`,
		`total[_\s]tokens[_\s]used[:\s]+(\d+)`,
		`tokens\s+used[:\s]+(\d+)`, // Codex format: "tokens used: 13934"
	}

	for _, pattern := range patterns {
		if match := extractFirstMatch(line, pattern); match != "" {
			if count, err := strconv.Atoi(match); err == nil {
				return count
			}
		}
	}

	return 0
}

// extractCost extracts cost information from log line
func extractCost(line string) float64 {
	// Look for patterns like "cost: $1.23", "price: 0.45", etc.
	patterns := []string{
		`cost[:\s]+\$?(\d+\.?\d*)`,
		`price[:\s]+\$?(\d+\.?\d*)`,
		`\$(\d+\.?\d+)`,
	}

	for _, pattern := range patterns {
		if match := extractFirstMatch(line, pattern); match != "" {
			if cost, err := strconv.ParseFloat(match, 64); err == nil {
				return cost
			}
		}
	}

	return 0
}

// extractFirstMatch extracts the first regex match from a string
func extractFirstMatch(text, pattern string) string {
	re := regexp.MustCompile(`(?i)` + pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractJSONMetrics extracts metrics from streaming JSON log lines
func extractJSONMetrics(line string, verbose bool) JSONMetrics {
	var metrics JSONMetrics

	// Skip lines that don't look like JSON
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return metrics
	}

	// Try to parse as generic JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &jsonData); err != nil {
		return metrics
	}

	// Extract timestamp from various possible fields
	if ts := extractJSONTimestamp(jsonData); !ts.IsZero() {
		metrics.Timestamp = ts
	}

	// Extract token usage from various possible fields and structures
	if tokens := extractJSONTokenUsage(jsonData); tokens > 0 {
		metrics.TokenUsage = tokens
	}

	// Extract cost information from various possible fields
	if cost := extractJSONCost(jsonData); cost > 0 {
		metrics.EstimatedCost = cost
	}

	return metrics
}

// extractJSONTimestamp extracts timestamp from JSON data
func extractJSONTimestamp(data map[string]interface{}) time.Time {
	// Common timestamp field names
	timestampFields := []string{"timestamp", "time", "created_at", "updated_at", "ts"}

	for _, field := range timestampFields {
		if val, exists := data[field]; exists {
			if timeStr, ok := val.(string); ok {
				// Try common timestamp formats
				formats := []string{
					time.RFC3339,
					time.RFC3339Nano,
					"2006-01-02T15:04:05Z",
					"2006-01-02T15:04:05.000Z",
					"2006-01-02 15:04:05",
				}

				for _, format := range formats {
					if t, err := time.Parse(format, timeStr); err == nil {
						return t
					}
				}
			}
		}
	}

	return time.Time{}
}

// extractJSONTokenUsage extracts token usage from JSON data
func extractJSONTokenUsage(data map[string]interface{}) int {
	// Check top-level token fields
	tokenFields := []string{"tokens", "token_count", "input_tokens", "output_tokens", "total_tokens"}
	for _, field := range tokenFields {
		if val, exists := data[field]; exists {
			if tokens := convertToInt(val); tokens > 0 {
				return tokens
			}
		}
	}

	// Check nested usage objects (Claude API format)
	if usage, exists := data["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			// Claude format: {"usage": {"input_tokens": 10, "output_tokens": 5, "cache_creation_input_tokens": 100, "cache_read_input_tokens": 200}}
			inputTokens := convertToInt(usageMap["input_tokens"])
			outputTokens := convertToInt(usageMap["output_tokens"])
			cacheCreationTokens := convertToInt(usageMap["cache_creation_input_tokens"])
			cacheReadTokens := convertToInt(usageMap["cache_read_input_tokens"])

			totalTokens := inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
			if totalTokens > 0 {
				return totalTokens
			}

			// Generic token count in usage
			for _, field := range tokenFields {
				if val, exists := usageMap[field]; exists {
					if tokens := convertToInt(val); tokens > 0 {
						return tokens
					}
				}
			}
		}
	}

	// Check for delta structures (streaming format)
	if delta, exists := data["delta"]; exists {
		if deltaMap, ok := delta.(map[string]interface{}); ok {
			if usage, exists := deltaMap["usage"]; exists {
				if usageMap, ok := usage.(map[string]interface{}); ok {
					inputTokens := convertToInt(usageMap["input_tokens"])
					outputTokens := convertToInt(usageMap["output_tokens"])
					if inputTokens > 0 || outputTokens > 0 {
						return inputTokens + outputTokens
					}
				}
			}
		}
	}

	return 0
}

// extractJSONCost extracts cost information from JSON data
func extractJSONCost(data map[string]interface{}) float64 {
	// Common cost field names
	costFields := []string{"cost", "price", "amount", "total_cost", "estimated_cost", "total_cost_usd"}

	for _, field := range costFields {
		if val, exists := data[field]; exists {
			if cost := convertToFloat(val); cost > 0 {
				return cost
			}
		}
	}

	// Check nested billing or pricing objects
	if billing, exists := data["billing"]; exists {
		if billingMap, ok := billing.(map[string]interface{}); ok {
			for _, field := range costFields {
				if val, exists := billingMap[field]; exists {
					if cost := convertToFloat(val); cost > 0 {
						return cost
					}
				}
			}
		}
	}

	return 0
}

// convertToInt safely converts interface{} to int
func convertToInt(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// convertToFloat safely converts interface{} to float64
func convertToFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

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
			tokensStr = fmt.Sprintf("%d", run.TokenUsage)
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
		fmt.Sprintf("%d", totalTokens),
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
