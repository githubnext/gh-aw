package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// EnableWorkflows enables workflows matching a pattern
func EnableWorkflows(pattern string) error {
	return toggleWorkflows(pattern, true)
}

// DisableWorkflows disables workflows matching a pattern
func DisableWorkflows(pattern string) error {
	return toggleWorkflows(pattern, false)
}

// Helper function to toggle workflows
func toggleWorkflows(pattern string, enable bool) error {
	action := "enable"
	if !enable {
		action = "disable"
	}

	// Check if gh CLI is available
	if !isGHCLIAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is required but not available")
	}

	// Get the core set of workflows from markdown files in .github/workflows
	mdFiles, err := getMarkdownWorkflowFiles()
	if err != nil {
		// Handle missing .github/workflows directory gracefully
		fmt.Printf("No workflow files found to %s.\n", action)
		return fmt.Errorf("no workflow files found to %s: %v", action, err)
	}

	if len(mdFiles) == 0 {
		fmt.Printf("No markdown workflow files found to %s.\n", action)
		return fmt.Errorf("no markdown workflow files found to %s", action)
	}

	// Get GitHub workflows status for comparison; warn but continue if unavailable
	githubWorkflows, err := fetchGitHubWorkflows(false)
	if err != nil {
		fmt.Printf("Warning: Unable to fetch GitHub workflows (gh CLI may not be authenticated): %v\n", err)
		githubWorkflows = make(map[string]*GitHubWorkflow)
	}

	// Internal target model to support enabling by ID or lock filename
	type workflowTarget struct {
		Name           string
		ID             int64  // 0 if unknown
		LockFileBase   string // e.g., dev.lock.yml
		CurrentState   string // known state or "unknown"
		HasGitHubEntry bool
	}

	var targets []workflowTarget
	matchedCount := 0
	alreadyDesired := 0

	// Find matching workflows from the markdown files
	for _, file := range mdFiles {
		base := filepath.Base(file)
		name := strings.TrimSuffix(base, ".md")

		// Skip if pattern specified and doesn't match
		if pattern != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
			continue
		}

		// Determine lock file and GitHub status (if available)
		lockFile := strings.TrimSuffix(file, ".md") + ".lock.yml"
		lockFileBase := filepath.Base(lockFile)

		githubWorkflow, exists := githubWorkflows[name]

		// Count this as matched regardless of state
		matchedCount++

		// If enabling and lock file doesn't exist locally, try to compile it
		if enable {
			if _, err := os.Stat(lockFile); os.IsNotExist(err) {
				if err := compileWorkflow(file, false, ""); err != nil {
					fmt.Printf("Warning: Failed to compile workflow %s to create lock file: %v\n", name, err)
					// If we can't compile and there's no GitHub entry, skip because we can't address it
					if !exists {
						continue
					}
				}
			}
		}

		// Skip if no work is needed based on known GitHub state
		if exists {
			if enable && githubWorkflow.State == "active" {
				// Already enabled
				alreadyDesired++
				continue
			}
			if !enable && githubWorkflow.State == "disabled_manually" {
				// Already disabled
				alreadyDesired++
				continue
			}
		}

		t := workflowTarget{
			Name:           name,
			ID:             0,
			LockFileBase:   lockFileBase,
			CurrentState:   "unknown",
			HasGitHubEntry: exists,
		}
		if exists {
			t.ID = githubWorkflow.ID
			t.CurrentState = githubWorkflow.State
		}
		targets = append(targets, t)
	}

	if len(targets) == 0 {
		if matchedCount > 0 {
			// Nothing to change; consider as success for idempotency
			fmt.Printf("All workflows matching pattern '%s' are already %sd.\n", pattern, action)
			return nil
		}
		fmt.Printf("No workflows found matching pattern '%s'.\n", pattern)
		return fmt.Errorf("no workflows found matching pattern '%s'", pattern)
	}

	// Show what will be changed
	fmt.Printf("The following workflows will be %sd:\n", action)
	for _, t := range targets {
		fmt.Printf("  %s (current state: %s)\n", t.Name, t.CurrentState)
	}

	// Perform the action
	var failures []string

	for _, t := range targets {
		var cmd *exec.Cmd
		if enable {
			// Prefer enabling by ID, otherwise fall back to lock file name
			if t.ID != 0 {
				cmd = exec.Command("gh", "workflow", "enable", strconv.FormatInt(t.ID, 10))
			} else {
				cmd = exec.Command("gh", "workflow", "enable", t.LockFileBase)
			}
		} else {
			// First cancel any running workflows (by ID when available, else by lock file name)
			if t.ID != 0 {
				if err := cancelWorkflowRuns(t.ID); err != nil {
					fmt.Printf("Warning: Failed to cancel runs for workflow %s: %v\n", t.Name, err)
				}
				// Prefer disabling by lock file name for reliability
				cmd = exec.Command("gh", "workflow", "disable", t.LockFileBase)
			} else {
				if err := cancelWorkflowRunsByLockFile(t.LockFileBase); err != nil {
					fmt.Printf("Warning: Failed to cancel runs for workflow %s: %v\n", t.Name, err)
				}
				cmd = exec.Command("gh", "workflow", "disable", t.LockFileBase)
			}
		}

		if output, err := cmd.CombinedOutput(); err != nil {
			if len(output) > 0 {
				fmt.Printf("Failed to %s workflow %s: %v\n%s\n", action, t.Name, err, string(output))
				// Provide clearer hint on common permission issues
				outStr := strings.ToLower(string(output))
				if strings.Contains(outStr, "http 403") || strings.Contains(outStr, "resource not accessible by integration") {
					fmt.Printf("Hint: Disabling/enabling workflows requires repository admin or maintainer permissions. Ensure your gh auth has write/admin access to this repo.\n")
				}
			} else {
				fmt.Printf("Failed to %s workflow %s: %v\n", action, t.Name, err)
			}
			failures = append(failures, t.Name)
		} else {
			fmt.Printf("%sd workflow: %s\n", strings.ToUpper(action[:1])+action[1:], t.Name)
		}
	}

	// Return error if any workflows failed to be processed
	if len(failures) > 0 {
		if enable {
			return fmt.Errorf("failed to enable %d workflow(s): %s", len(failures), strings.Join(failures, ", "))
		} else {
			return fmt.Errorf("failed to disable %d workflow(s): %s", len(failures), strings.Join(failures, ", "))
		}
	}

	return nil
}
