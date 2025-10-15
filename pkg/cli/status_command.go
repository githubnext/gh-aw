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
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// WorkflowStatus represents the status of a single workflow for JSON output
type WorkflowStatus struct {
	Workflow      string `json:"workflow" console:"header:Workflow"`
	EngineID      string `json:"engine_id" console:"header:Engine"`
	Compiled      string `json:"compiled" console:"header:Compiled"`
	Status        string `json:"status" console:"header:Status"`
	TimeRemaining string `json:"time_remaining" console:"header:Time Remaining"`
	On            any    `json:"on,omitempty" console:"-"`
}

func StatusWorkflows(pattern string, verbose bool, jsonOutput bool) error {
	if verbose && !jsonOutput {
		fmt.Printf("Checking status of workflow files\n")
		if pattern != "" {
			fmt.Printf("Filtering by pattern: %s\n", pattern)
		}
	}

	mdFiles, err := getMarkdownWorkflowFiles()
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

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
	githubWorkflows, err := fetchGitHubWorkflows("", verbose && !jsonOutput)
	if err != nil {
		if verbose && !jsonOutput {
			fmt.Printf("Verbose: Failed to fetch GitHub workflows: %v\n", err)
		}
		if !jsonOutput {
			fmt.Printf("Warning: Could not fetch GitHub workflow status: %v\n", err)
		}
		githubWorkflows = make(map[string]*GitHubWorkflow)
	} else if verbose && !jsonOutput {
		fmt.Printf("Successfully fetched %d GitHub workflows\n", len(githubWorkflows))
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
			compiled := "N/A"
			timeRemaining := "N/A"

			if _, err := os.Stat(lockFile); err == nil {
				// Check if up to date
				mdStat, _ := os.Stat(file)
				lockStat, _ := os.Stat(lockFile)
				if mdStat.ModTime().After(lockStat.ModTime()) {
					compiled = "No"
				} else {
					compiled = "Yes"
				}

				// Extract stop-time from lock file
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

			// Extract "on" field from frontmatter for JSON output
			var onField any
			if content, err := os.ReadFile(file); err == nil {
				if result, err := parser.ExtractFrontmatterFromContent(string(content)); err == nil {
					if result.Frontmatter != nil {
						onField = result.Frontmatter["on"]
					}
				}
			}

			// Build status object
			statuses = append(statuses, WorkflowStatus{
				Workflow:      name,
				EngineID:      agent,
				Compiled:      compiled,
				Status:        status,
				TimeRemaining: timeRemaining,
				On:            onField,
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
		compiled := "N/A"
		timeRemaining := "N/A"

		if _, err := os.Stat(lockFile); err == nil {
			// Check if up to date
			mdStat, _ := os.Stat(file)
			lockStat, _ := os.Stat(lockFile)
			if mdStat.ModTime().After(lockStat.ModTime()) {
				compiled = "No"
			} else {
				compiled = "Yes"
			}

			// Extract stop-time from lock file
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

		// Build status object
		statuses = append(statuses, WorkflowStatus{
			Workflow:      name,
			EngineID:      agent,
			Compiled:      compiled,
			Status:        status,
			TimeRemaining: timeRemaining,
		})
	}

	// Render the table using struct-based rendering
	fmt.Print(console.RenderStruct(statuses))

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
