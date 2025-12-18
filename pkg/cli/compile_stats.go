package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"gopkg.in/yaml.v3"
)

// WorkflowStats holds statistics about a compiled workflow
type WorkflowStats struct {
	Workflow     string
	FileSize     int64
	Jobs         int
	Steps        int
	ScriptCount  int
	ScriptSize   int
	ShellCount   int
	ShellSize    int
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

	// Print header
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflow Statistics (sorted by file size)"))
	fmt.Fprintln(os.Stderr, "")

	// Calculate column widths
	maxWorkflowLen := len("WORKFLOW")
	for _, stats := range statsList {
		if len(stats.Workflow) > maxWorkflowLen {
			maxWorkflowLen = len(stats.Workflow)
		}
	}

	// Print table header
	headerFormat := fmt.Sprintf("%%-%ds  %%10s  %%5s  %%6s  %%8s  %%8s\n", maxWorkflowLen)
	rowFormat := fmt.Sprintf("%%-%ds  %%10s  %%5d  %%6d  %%8d  %%8d\n", maxWorkflowLen)

	header := fmt.Sprintf(headerFormat, "WORKFLOW", "FILE SIZE", "JOBS", "STEPS", "SCRIPTS", "SHELLS")
	fmt.Fprint(os.Stderr, header)
	fmt.Fprintln(os.Stderr, strings.Repeat("-", len(header)-1))

	// Print each workflow's stats
	for _, stats := range statsList {
		fileSize := console.FormatFileSize(stats.FileSize)
		fmt.Fprintf(os.Stderr, rowFormat,
			stats.Workflow,
			fileSize,
			stats.Jobs,
			stats.Steps,
			stats.ScriptCount,
			stats.ShellCount,
		)
	}

	// Print summary
	fmt.Fprintln(os.Stderr, "")

	// Calculate totals
	totalSize := int64(0)
	totalJobs := 0
	totalSteps := 0
	totalScripts := 0
	totalScriptSize := 0
	totalShells := 0
	totalShellSize := 0

	for _, stats := range statsList {
		totalSize += stats.FileSize
		totalJobs += stats.Jobs
		totalSteps += stats.Steps
		totalScripts += stats.ScriptCount
		totalScriptSize += stats.ScriptSize
		totalShells += stats.ShellCount
		totalShellSize += stats.ShellSize
	}

	// Print totals
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Summary:"))
	fmt.Fprintf(os.Stderr, "  Total workflows: %d\n", len(statsList))
	fmt.Fprintf(os.Stderr, "  Total size:      %s\n", console.FormatFileSize(totalSize))
	fmt.Fprintf(os.Stderr, "  Total jobs:      %d\n", totalJobs)
	fmt.Fprintf(os.Stderr, "  Total steps:     %d\n", totalSteps)
	fmt.Fprintf(os.Stderr, "  Total scripts:   %d (%s)\n", totalScripts, console.FormatFileSize(int64(totalScriptSize)))
	fmt.Fprintf(os.Stderr, "  Total shells:    %d (%s)\n", totalShells, console.FormatFileSize(int64(totalShellSize)))
}
