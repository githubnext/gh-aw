package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/styles"
	"github.com/githubnext/gh-aw/pkg/tty"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var statusLog = logger.New("cli:status_command")

// WorkflowStatus represents the status of a single workflow for JSON output
type WorkflowStatus struct {
	Workflow      string   `json:"workflow" console:"header:Workflow"`
	EngineID      string   `json:"engine_id" console:"header:Engine"`
	Compiled      string   `json:"compiled" console:"header:Compiled"`
	Status        string   `json:"status" console:"header:Status"`
	TimeRemaining string   `json:"time_remaining" console:"header:Time Remaining"`
	Labels        []string `json:"labels,omitempty" console:"header:Labels,omitempty"`
	On            any      `json:"on,omitempty" console:"-"`
	RunStatus     string   `json:"run_status,omitempty" console:"header:Run Status,omitempty"`
	RunConclusion string   `json:"run_conclusion,omitempty" console:"header:Run Conclusion,omitempty"`
	ActionNeeded  string   `json:"action_needed,omitempty" console:"header:Action Needed,omitempty"`
}

func StatusWorkflows(pattern string, verbose bool, jsonOutput bool, ref string, labelFilter string, fixSuggestions bool) error {
	statusLog.Printf("Checking workflow status: pattern=%s, jsonOutput=%v, ref=%s, labelFilter=%s", pattern, jsonOutput, ref, labelFilter)
	if verbose && !jsonOutput {
		fmt.Printf("Checking status of workflow files\n")
		if pattern != "" {
			fmt.Printf("Filtering by pattern: %s\n", pattern)
		}
	}

	mdFiles, err := getMarkdownWorkflowFiles()
	if err != nil {
		statusLog.Printf("Failed to get markdown workflow files: %v", err)
		fmt.Println(err.Error())
		return nil
	}

	statusLog.Printf("Found %d markdown workflow files", len(mdFiles))
	if len(mdFiles) == 0 {
		if jsonOutput {
			// Output empty array for JSON
			output := []WorkflowStatus{}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
			return nil
		}
		fmt.Println("No workflow files found.")
		return nil
	}

	if verbose && !jsonOutput {
		fmt.Printf("Found %d markdown workflow files\n", len(mdFiles))
		fmt.Printf("Fetching GitHub workflow status...\n")
	}

	// Get GitHub workflows data
	statusLog.Print("Fetching GitHub workflow status")
	githubWorkflows, err := fetchGitHubWorkflows("", verbose && !jsonOutput)
	if err != nil {
		statusLog.Printf("Failed to fetch GitHub workflows: %v", err)
		if verbose && !jsonOutput {
			fmt.Printf("Verbose: Failed to fetch GitHub workflows: %v\n", err)
		}
		if !jsonOutput {
			fmt.Printf("Warning: Could not fetch GitHub workflow status: %v\n", err)
		}
		githubWorkflows = make(map[string]*GitHubWorkflow)
	} else {
		statusLog.Printf("Successfully fetched %d GitHub workflows", len(githubWorkflows))
		if verbose && !jsonOutput {
			fmt.Printf("Successfully fetched %d GitHub workflows\n", len(githubWorkflows))
		}
	}

	// Fetch latest workflow runs for ref if specified
	var latestRunsByWorkflow map[string]*WorkflowRun
	if ref != "" {
		if verbose && !jsonOutput {
			fmt.Printf("Fetching latest runs for ref: %s\n", ref)
		}
		latestRunsByWorkflow, err = fetchLatestRunsByRef(ref, verbose && !jsonOutput)
		if err != nil {
			statusLog.Printf("Failed to fetch workflow runs for ref %s: %v", ref, err)
			if verbose && !jsonOutput {
				fmt.Printf("Verbose: Failed to fetch workflow runs for ref: %v\n", err)
			}
			if !jsonOutput {
				fmt.Printf("Warning: Could not fetch workflow runs for ref '%s': %v\n", ref, err)
			}
			latestRunsByWorkflow = make(map[string]*WorkflowRun)
		} else {
			statusLog.Printf("Successfully fetched %d workflow runs for ref %s", len(latestRunsByWorkflow), ref)
			if verbose && !jsonOutput {
				fmt.Printf("Successfully fetched %d workflow runs for ref\n", len(latestRunsByWorkflow))
			}
		}
	}

	// Build table configuration or JSON output
	if jsonOutput {
		// Build JSON output
		var statuses []WorkflowStatus
		for _, file := range mdFiles {
			base := filepath.Base(file)
			name := strings.TrimSuffix(base, ".md")

			// Skip if pattern specified and doesn't match
			if pattern != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
				continue
			}

			// Extract engine ID from workflow file
			agent := extractEngineIDFromFile(file)

			// Check if compiled (.lock.yml file is in .github/workflows)
			lockFile := strings.TrimSuffix(file, ".md") + ".lock.yml"
			timeRemaining := "N/A"

			// Determine staleness and action needed
			staleInfo := checkWorkflowStaleness(file, lockFile)
			compiled := staleInfo.compiled
			actionNeeded := staleInfo.actionNeeded

			// Extract stop-time from lock file if it exists
			if _, err := os.Stat(lockFile); err == nil {
				if stopTime := workflow.ExtractStopTimeFromLockFile(lockFile); stopTime != "" {
					timeRemaining = calculateTimeRemaining(stopTime)
				}
			}

			// Get GitHub workflow status
			status := "Unknown"
			if workflow, exists := githubWorkflows[name]; exists {
				if workflow.State == "disabled_manually" {
					status = "disabled"
				} else {
					status = workflow.State
				}
			}

			// Extract "on" field and labels from frontmatter for JSON output
			var onField any
			var labels []string
			if content, err := os.ReadFile(file); err == nil {
				if result, err := parser.ExtractFrontmatterFromContent(string(content)); err == nil {
					if result.Frontmatter != nil {
						onField = result.Frontmatter["on"]
						// Extract labels field if present
						if labelsField, ok := result.Frontmatter["labels"]; ok {
							if labelsArray, ok := labelsField.([]any); ok {
								for _, label := range labelsArray {
									if labelStr, ok := label.(string); ok {
										labels = append(labels, labelStr)
									}
								}
							}
						}
					}
				}
			}

			// Skip if label filter specified and workflow doesn't have the label
			if labelFilter != "" {
				hasLabel := false
				for _, label := range labels {
					if strings.EqualFold(label, labelFilter) {
						hasLabel = true
						break
					}
				}
				if !hasLabel {
					continue
				}
			}

			// Get run status for ref if available
			var runStatus, runConclusion string
			if latestRunsByWorkflow != nil {
				if run, exists := latestRunsByWorkflow[name]; exists {
					runStatus = run.Status
					runConclusion = run.Conclusion
				}
			}

			// Build status object
			statuses = append(statuses, WorkflowStatus{
				Workflow:      name,
				EngineID:      agent,
				Compiled:      compiled,
				Status:        status,
				TimeRemaining: timeRemaining,
				Labels:        labels,
				On:            onField,
				RunStatus:     runStatus,
				RunConclusion: runConclusion,
				ActionNeeded:  actionNeeded,
			})
		}

		// Output JSON
		jsonBytes, err := json.MarshalIndent(statuses, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Build status list for text output
	var statuses []WorkflowStatus

	for _, file := range mdFiles {
		base := filepath.Base(file)
		name := strings.TrimSuffix(base, ".md")

		// Skip if pattern specified and doesn't match
		if pattern != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
			continue
		}

		// Extract engine ID from workflow file
		agent := extractEngineIDFromFile(file)

		// Check if compiled (.lock.yml file is in .github/workflows)
		lockFile := strings.TrimSuffix(file, ".md") + ".lock.yml"
		timeRemaining := "N/A"

		// Determine staleness and action needed
		staleInfo := checkWorkflowStaleness(file, lockFile)
		compiled := colorCodeCompiled(staleInfo.compiled) // Apply color for console output
		actionNeeded := staleInfo.actionNeeded

		// Extract stop-time from lock file if it exists
		if _, err := os.Stat(lockFile); err == nil {
			if stopTime := workflow.ExtractStopTimeFromLockFile(lockFile); stopTime != "" {
				timeRemaining = calculateTimeRemaining(stopTime)
			}
		}

		// Get GitHub workflow status
		status := "Unknown"
		if workflow, exists := githubWorkflows[name]; exists {
			if workflow.State == "disabled_manually" {
				status = "disabled"
			} else {
				status = workflow.State
			}
		}

		// Get run status for ref if available
		var runStatus, runConclusion string
		if latestRunsByWorkflow != nil {
			if run, exists := latestRunsByWorkflow[name]; exists {
				runStatus = run.Status
				runConclusion = run.Conclusion
			}
		}

		// Extract labels from frontmatter
		var labels []string
		if content, err := os.ReadFile(file); err == nil {
			if result, err := parser.ExtractFrontmatterFromContent(string(content)); err == nil {
				if result.Frontmatter != nil {
					if labelsField, ok := result.Frontmatter["labels"]; ok {
						if labelsArray, ok := labelsField.([]any); ok {
							for _, label := range labelsArray {
								if labelStr, ok := label.(string); ok {
									labels = append(labels, labelStr)
								}
							}
						}
					}
				}
			}
		}

		// Skip if label filter specified and workflow doesn't have the label
		if labelFilter != "" {
			hasLabel := false
			for _, label := range labels {
				if strings.EqualFold(label, labelFilter) {
					hasLabel = true
					break
				}
			}
			if !hasLabel {
				continue
			}
		}

		// Build status object
		statuses = append(statuses, WorkflowStatus{
			Workflow:      name,
			EngineID:      agent,
			Compiled:      compiled,
			Status:        status,
			TimeRemaining: timeRemaining,
			Labels:        labels,
			RunStatus:     runStatus,
			RunConclusion: runConclusion,
			ActionNeeded:  actionNeeded,
		})
	}

	// Handle --fix-suggestions flag
	if fixSuggestions {
		return generateFixScript(statuses)
	}

	// Render the table using struct-based rendering
	fmt.Print(console.RenderStruct(statuses))

	// Add summary footer showing compilation status
	needsCompilation := 0
	for _, s := range statuses {
		if s.ActionNeeded != "" {
			needsCompilation++
		}
	}

	if needsCompilation > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%d workflow%s need compilation", needsCompilation, pluralize(needsCompilation))))
		fmt.Fprintf(os.Stderr, "  Run: %s\n", console.FormatCommandMessage("gh aw compile"))
	} else if len(statuses) > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All workflows are up-to-date"))
	}

	return nil
}

// calculateTimeRemaining calculates and formats the time remaining until stop-time
func calculateTimeRemaining(stopTimeStr string) string {
	if stopTimeStr == "" {
		return "N/A"
	}

	// Parse the stop time in local timezone
	stopTime, err := time.ParseInLocation("2006-01-02 15:04:05", stopTimeStr, time.Local)
	if err != nil {
		return "Invalid"
	}

	now := time.Now()
	remaining := stopTime.Sub(now)

	// If already past the stop time
	if remaining <= 0 {
		return "Expired"
	}

	// Format the remaining time in a human-readable way
	days := int(remaining.Hours() / 24)
	hours := int(remaining.Hours()) % 24
	minutes := int(remaining.Minutes()) % 60

	if days > 0 {
		if days == 1 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	} else {
		return "< 1m"
	}
}

// stalenessInfo holds information about workflow staleness
type stalenessInfo struct {
	compiled     string // "Yes", "No", "N/A", or "Stale"
	actionNeeded string // Command to run if action is needed
}

// colorCodeCompiled adds color formatting to the compiled status for console output
func colorCodeCompiled(compiled string) string {
	// Only apply colors if output is a TTY
	if !tty.IsStdoutTerminal() {
		return compiled
	}

	switch compiled {
	case "Yes":
		// Green for up-to-date
		return styles.Success.Render(compiled)
	case "Stale":
		// Yellow for stale (needs recompilation)
		return styles.Warning.Render(compiled)
	case "No":
		// Red for never compiled
		return styles.Error.Render(compiled)
	default:
		// No color for N/A or other statuses
		return compiled
	}
}

// checkWorkflowStaleness determines if a workflow needs compilation and returns action needed
func checkWorkflowStaleness(mdFile, lockFile string) stalenessInfo {
	// Check if lock file exists
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		// Never compiled
		baseName := filepath.Base(mdFile)
		return stalenessInfo{
			compiled:     "No",
			actionNeeded: fmt.Sprintf("gh aw compile %s", baseName),
		}
	}

	// Lock file exists - check if up to date using timestamp comparison
	mdStat, err := os.Stat(mdFile)
	if err != nil {
		return stalenessInfo{compiled: "N/A", actionNeeded: ""}
	}

	lockStat, err := os.Stat(lockFile)
	if err != nil {
		return stalenessInfo{compiled: "N/A", actionNeeded: ""}
	}

	if mdStat.ModTime().After(lockStat.ModTime()) {
		// Stale - source modified after lock file
		baseName := filepath.Base(mdFile)
		return stalenessInfo{
			compiled:     "Stale",
			actionNeeded: fmt.Sprintf("gh aw compile %s", baseName),
		}
	}

	// Up to date
	return stalenessInfo{
		compiled:     "Yes",
		actionNeeded: "",
	}
}

// StatusWorkflows shows status of workflows
// getMarkdownWorkflowFiles finds all markdown files in .github/workflows directory
func getMarkdownWorkflowFiles() ([]string, error) {
	workflowsDir := getWorkflowsDir()

	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no .github/workflows directory found")
	}

	// Find all markdown files in .github/workflows
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow files: %w", err)
	}

	return mdFiles, nil
}

// Helper functions

func extractWorkflowNameFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Extract markdown content (excluding frontmatter)
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return "", err
	}

	// Look for first H1 header
	scanner := bufio.NewScanner(strings.NewReader(result.Markdown))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:]), nil
		}
	}

	// No H1 header found, generate default name from filename
	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	baseName = strings.ReplaceAll(baseName, "-", " ")

	// Capitalize first letter of each word
	words := strings.Fields(baseName)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " "), nil
}

// extractEngineIDFromFile extracts the engine ID from a workflow file's frontmatter
func extractEngineIDFromFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "" // Return empty string if file cannot be read
	}

	// Parse frontmatter
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return "" // Return empty string if frontmatter cannot be parsed
	}

	// Use the workflow package's extractEngineConfig to handle both string and object formats
	compiler := &workflow.Compiler{}
	engineSetting, engineConfig := compiler.ExtractEngineConfig(result.Frontmatter)

	// If engine is specified, return the ID from the config
	if engineConfig != nil && engineConfig.ID != "" {
		return engineConfig.ID
	}

	// If we have an engine setting string, return it
	if engineSetting != "" {
		return engineSetting
	}

	return "copilot" // Default engine
}

// fetchLatestRunsByRef fetches the latest workflow run for each workflow from a specific ref (branch or tag)
func fetchLatestRunsByRef(ref string, verbose bool) (map[string]*WorkflowRun, error) {
	statusLog.Printf("Fetching latest workflow runs for ref: %s", ref)

	// Start spinner for network operation (only if not in verbose mode)
	spinner := console.NewSpinner("Fetching workflow runs for ref...")
	if !verbose {
		spinner.Start()
	}

	// Fetch workflow runs for the ref (uses --branch flag which also works for tags)
	args := []string{"run", "list", "--branch", ref, "--json", "databaseId,number,url,status,conclusion,workflowName,createdAt,headBranch", "--limit", "100"}
	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()

	if err != nil {
		// Stop spinner on error
		if !verbose {
			spinner.Stop()
		}
		return nil, fmt.Errorf("failed to execute gh run list command: %w", err)
	}

	// Check if output is empty
	if len(output) == 0 {
		if !verbose {
			spinner.Stop()
		}
		return nil, fmt.Errorf("gh run list returned empty output")
	}

	// Validate JSON before unmarshaling
	if !json.Valid(output) {
		if !verbose {
			spinner.Stop()
		}
		return nil, fmt.Errorf("gh run list returned invalid JSON")
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		if !verbose {
			spinner.Stop()
		}
		return nil, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	// Stop spinner with success message
	if !verbose {
		spinner.StopWithMessage(fmt.Sprintf("✓ Fetched %d workflow runs", len(runs)))
	}

	// Build map of latest run for each workflow (first occurrence is the latest)
	latestRuns := make(map[string]*WorkflowRun)
	for i := range runs {
		run := &runs[i]
		// Extract workflow name from workflowName field
		workflowName := extractWorkflowNameFromPath(run.WorkflowName)
		// Only keep the first (latest) run for each workflow
		if _, exists := latestRuns[workflowName]; !exists {
			latestRuns[workflowName] = run
		}
	}

	statusLog.Printf("Fetched latest runs for %d workflows on ref %s", len(latestRuns), ref)
	return latestRuns, nil
}

// generateFixScript generates an executable shell script to fix stale workflows
func generateFixScript(statuses []WorkflowStatus) error {
	var staleWorkflows []string
	for _, s := range statuses {
		if s.ActionNeeded != "" {
			staleWorkflows = append(staleWorkflows, s.Workflow+".md")
		}
	}

	if len(staleWorkflows) == 0 {
		fmt.Println("#!/bin/bash")
		fmt.Println("# Generated by gh aw status --fix-suggestions")
		fmt.Println("# All workflows are up-to-date - no action needed")
		return nil
	}

	fmt.Println("#!/bin/bash")
	fmt.Println("# Generated by gh aw status --fix-suggestions")
	fmt.Println("# This script compiles all stale workflows")
	fmt.Println()
	fmt.Printf("# Found %d workflow%s needing compilation\n", len(staleWorkflows), pluralize(len(staleWorkflows)))
	fmt.Println()

	for _, workflow := range staleWorkflows {
		fmt.Printf("gh aw compile %s\n", workflow)
	}

	fmt.Println()
	fmt.Println("# Stage compiled lock files")
	fmt.Println("git add .github/workflows/*.lock.yml")
	fmt.Println()
	fmt.Println("echo \"Done! Review changes with: git diff --cached\"")

	return nil
}

// pluralize returns "s" if count != 1, otherwise empty string
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
