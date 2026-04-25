// This file provides command-line interface functionality for gh-aw.
// This file (logs_utils.go) contains utility functions used by the logs command.
//
// Key responsibilities:
//   - Discovering agentic workflow names from .lock.yml files
//   - Utility functions for slice operations

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsUtilsLog = logger.New("cli:logs_utils")

// getAgenticWorkflowNames reads all .lock.yml files and extracts their workflow names
func getAgenticWorkflowNames(verbose bool) ([]string, error) {
	logsUtilsLog.Print("Discovering agentic workflow names from .lock.yml files")
	var workflowNames []string

	// Look for .lock.yml files in .github/workflows directory
	workflowsDir := ".github/workflows"
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No .github/workflows directory found"))
		}
		return workflowNames, nil
	}

	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob .lock.yml files: %w", err)
	}

	logsUtilsLog.Printf("Found %d .lock.yml file(s) in %s", len(files), workflowsDir)

	for _, file := range files {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Reading workflow file: "+file))
		}

		content, err := os.ReadFile(file)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", file, err)))
			}
			continue
		}

		// Extract the workflow name using simple string parsing
		lines := strings.SplitSeq(string(content), "\n")
		for line := range lines {
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
						logsUtilsLog.Printf("Discovered workflow name: %s (from %s)", name, file)
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found agentic workflow: "+name))
						}
						break
					}
				}
			}
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d agentic workflows", len(workflowNames))))
	}

	return workflowNames, nil
}

// resolveExcludeWorkflows resolves a list of workflow names (IDs or display names) to their
// canonical display names for matching against WorkflowRun.WorkflowName.
// It loads all local workflows once and resolves all excludes against an in-memory map,
// avoiding repeated filesystem scans.
// If a name cannot be resolved (e.g., no .lock.yml files present or the workflow is from another
// repo), the raw value is kept so that isWorkflowExcluded can still attempt a slug-based match.
func resolveExcludeWorkflows(excludes []string, verbose bool) []string {
	if len(excludes) == 0 {
		return nil
	}

	// Load all local workflows once to build a fast lookup map.
	// Keys are both the display name (case-insensitive) and workflow ID (case-insensitive),
	// and the value is the canonical display name.
	displayByKey := make(map[string]string)
	allWorkflows, err := workflow.GetAllWorkflows()
	if err != nil {
		logsUtilsLog.Printf("Could not load workflow list for exclude resolution: %v", err)
	} else {
		for _, wf := range allWorkflows {
			displayByKey[strings.ToLower(wf.DisplayName)] = wf.DisplayName
			displayByKey[strings.ToLower(wf.WorkflowID)] = wf.DisplayName
		}
	}

	resolved := make([]string, 0, len(excludes))
	for _, name := range excludes {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Attempt to resolve via the in-memory map (display name or workflow ID lookup).
		if displayName, ok := displayByKey[strings.ToLower(name)]; ok {
			logsUtilsLog.Printf("Resolved exclude workflow '%s' -> '%s'", name, displayName)
			resolved = append(resolved, displayName)
		} else {
			// Cannot resolve - keep the raw value so isWorkflowExcluded can try slug matching.
			logsUtilsLog.Printf("Could not resolve exclude workflow '%s' (using raw value for matching)", name)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Exclude: using '%s' for matching (could not resolve to a known workflow)", name)))
			}
			resolved = append(resolved, name)
		}
	}
	return resolved
}

// isWorkflowExcluded reports whether a workflow display name matches any entry in the exclude list.
// Matching is case-insensitive to be resilient to capitalization differences.
// As a fallback it also compares a slugified form of the display name (spaces→hyphens) against
// each exclude entry, so that e.g. --exclude weekly-research matches a run named "Weekly Research"
// when local lock files are absent and resolution could not convert the ID to a display name.
func isWorkflowExcluded(workflowName string, excludes []string) bool {
	lowerName := strings.ToLower(workflowName)
	// Slug form: lowercase display name with spaces replaced by hyphens (mirrors common workflow ID convention).
	slugName := strings.ReplaceAll(lowerName, " ", "-")
	for _, ex := range excludes {
		lowerEx := strings.ToLower(ex)
		if lowerEx == lowerName || lowerEx == slugName {
			return true
		}
	}
	return false
}
