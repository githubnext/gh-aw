package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/githubnext/gh-aw/pkg/console"
	"gopkg.in/yaml.v3"
)

// WorkflowStats holds statistics about a compiled workflow
type WorkflowStats struct {
	Workflow    string
	FileSize    int64
	Jobs        int
	Steps       int
	ScriptCount int
	ScriptSize  int
	ShellCount  int
	ShellSize   int
}

// collectWorkflowStats parses a lock file and collects statistics
func collectWorkflowStats(lockFilePath string) (*WorkflowStats, error) {
	// Get file size
	fileInfo, err := os.Stat(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read and parse YAML
	content, err := os.ReadFile(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var workflowYAML map[string]any
	if err := yaml.Unmarshal(content, &workflowYAML); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	stats := &WorkflowStats{
		Workflow: filepath.Base(lockFilePath),
		FileSize: fileInfo.Size(),
	}

	// Count jobs and steps
	if jobs, ok := workflowYAML["jobs"].(map[string]any); ok {
		stats.Jobs = len(jobs)

		// Iterate through jobs to count steps and scripts
		for _, jobData := range jobs {
			if job, ok := jobData.(map[string]any); ok {
				if steps, ok := job["steps"].([]any); ok {
					stats.Steps += len(steps)

					// Check each step for scripts
					for _, stepData := range steps {
						if step, ok := stepData.(map[string]any); ok {
							// Check for "run" field (script)
							if runScript, ok := step["run"].(string); ok {
								stats.ScriptCount++
								stats.ScriptSize += len(runScript)
							}

							// Check for "shell" field
							if shell, ok := step["shell"].(string); ok {
								stats.ShellCount++
								stats.ShellSize += len(shell)
							}
						}
					}
				}
			}
		}
	}

	return stats, nil
}

// displayStatsTable displays workflow statistics in a sorted table
func displayStatsTable(statsList []*WorkflowStats) {
	if len(statsList) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow statistics to display"))
		return
	}

	// Sort by file size (descending)
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].FileSize > statsList[j].FileSize
	})

	// Calculate totals
	totalSize := int64(0)
	totalJobs := 0
	totalSteps := 0
	totalScripts := 0
	totalScriptSize := 0

	for _, stats := range statsList {
		totalSize += stats.FileSize
		totalJobs += stats.Jobs
		totalSteps += stats.Steps
		totalScripts += stats.ScriptCount
		totalScriptSize += stats.ScriptSize
	}

	// Build table rows
	rows := make([][]string, 0, len(statsList))
	for _, stats := range statsList {
		rows = append(rows, []string{
			stats.Workflow,
			console.FormatFileSize(stats.FileSize),
			fmt.Sprintf("%d", stats.Jobs),
			fmt.Sprintf("%d", stats.Steps),
			fmt.Sprintf("%d", stats.ScriptCount),
		})
	}

	// Create table config
	tableConfig := console.TableConfig{
		Title:   "",
		Headers: []string{"WORKFLOW", "FILE SIZE", "JOBS", "STEPS", "SCRIPTS"},
		Rows:    rows,
	}

	// Render and print table
	fmt.Fprint(os.Stderr, console.RenderTable(tableConfig))

	// Print summary
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Summary:"))
	fmt.Fprintf(os.Stderr, "  Total workflows: %d\n", len(statsList))
	fmt.Fprintf(os.Stderr, "  Total size:      %s\n", console.FormatFileSize(totalSize))
	fmt.Fprintf(os.Stderr, "  Total jobs:      %d\n", totalJobs)
	fmt.Fprintf(os.Stderr, "  Total steps:     %d\n", totalSteps)
	fmt.Fprintf(os.Stderr, "  Total scripts:   %d (%s)\n", totalScripts, console.FormatFileSize(int64(totalScriptSize)))
}
