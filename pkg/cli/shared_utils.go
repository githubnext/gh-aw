package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
)

// PullRequest represents a GitHub Pull Request
type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	IsDraft   bool      `json:"isDraft"`
	Mergeable string    `json:"mergeable"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AutoMergePullRequestsCreatedAfter checks for open PRs in the repository created after a specific time and auto-merges them
// This function filters PRs to only those created after the specified time to avoid merging unrelated PRs
func AutoMergePullRequestsCreatedAfter(repoSlug string, createdAfter time.Time, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking for open pull requests in %s created after %s", repoSlug, createdAfter.Format(time.RFC3339))))
	}

	// List open PRs with creation time information
	listCmd := exec.Command("gh", "pr", "list", "--repo", repoSlug, "--json", "number,title,isDraft,mergeable,createdAt,updatedAt")
	output, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list pull requests: %w", err)
	}

	var prs []PullRequest
	if err := json.Unmarshal(output, &prs); err != nil {
		return fmt.Errorf("failed to parse pull request list: %w", err)
	}

	if len(prs) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No open pull requests found"))
		}
		return nil
	}

	// Filter PRs to only those created after the specified time
	var eligiblePRs []PullRequest
	for _, pr := range prs {
		if pr.CreatedAt.After(createdAfter) {
			eligiblePRs = append(eligiblePRs, pr)
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Skipping PR #%d: created at %s (before workflow start time)", pr.Number, pr.CreatedAt.Format(time.RFC3339))))
		}
	}

	if len(eligiblePRs) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No pull requests found created after %s", createdAfter.Format(time.RFC3339))))
		}
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d pull request(s) created after workflow start time", len(eligiblePRs))))

	for _, pr := range eligiblePRs {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processing PR #%d: %s (draft: %t, mergeable: %s, created: %s)", 
				pr.Number, pr.Title, pr.IsDraft, pr.Mergeable, pr.CreatedAt.Format(time.RFC3339))))
		}

		// Convert from draft to non-draft if necessary
		if pr.IsDraft {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Converting PR #%d from draft to ready for review", pr.Number)))
			readyCmd := exec.Command("gh", "pr", "ready", fmt.Sprintf("%d", pr.Number), "--repo", repoSlug)
			if output, err := readyCmd.CombinedOutput(); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to convert PR #%d from draft: %v (output: %s)", pr.Number, err, string(output))))
				continue
			}
		}

		// Check if PR is mergeable
		if pr.Mergeable != "MERGEABLE" {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("PR #%d is not mergeable (status: %s), skipping auto-merge", pr.Number, pr.Mergeable)))
			continue
		}

		// Auto-merge the PR
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auto-merging PR #%d", pr.Number)))
		mergeCmd := exec.Command("gh", "pr", "merge", fmt.Sprintf("%d", pr.Number), "--repo", repoSlug, "--auto", "--squash")
		if output, err := mergeCmd.CombinedOutput(); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to auto-merge PR #%d: %v (output: %s)", pr.Number, err, string(output))))
			continue
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully enabled auto-merge for PR #%d", pr.Number)))
	}

	return nil
}

// AutoMergePullRequestsLegacy is the legacy function that auto-merges all open PRs (used by trial command for backward compatibility)
func AutoMergePullRequestsLegacy(repoSlug string, verbose bool) error {
	// Use a very old time (Unix epoch) to include all PRs
	return AutoMergePullRequestsCreatedAfter(repoSlug, time.Unix(0, 0), verbose)
}

// WaitForWorkflowCompletion waits for a workflow run to complete, with a specified timeout
func WaitForWorkflowCompletion(repoSlug, runID string, timeoutMinutes int, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Waiting for workflow completion (timeout: %d minutes)", timeoutMinutes)))
	}

	// Use the repository slug directly
	fullRepoName := repoSlug

	timeout := time.Duration(timeoutMinutes) * time.Minute
	start := time.Now()

	for {
		// Check if timeout exceeded
		if time.Since(start) > timeout {
			return fmt.Errorf("workflow execution timed out after %d minutes", timeoutMinutes)
		}

		// Check workflow status
		cmd := exec.Command("gh", "run", "view", runID, "--repo", fullRepoName, "--json", "status,conclusion")
		output, err := cmd.Output()

		if err != nil {
			return fmt.Errorf("failed to check workflow status: %w", err)
		}

		status := string(output)

		// Check if completed
		if strings.Contains(status, `"status":"completed"`) {
			if strings.Contains(status, `"conclusion":"success"`) {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow completed successfully"))
				}
				return nil
			} else if strings.Contains(status, `"conclusion":"failure"`) {
				return fmt.Errorf("workflow failed")
			} else if strings.Contains(status, `"conclusion":"cancelled"`) {
				return fmt.Errorf("workflow was cancelled")
			} else {
				return fmt.Errorf("workflow completed with unknown conclusion")
			}
		}

		// Still running, wait before checking again
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Workflow still running..."))
		}
		time.Sleep(10 * time.Second)
	}
}

// GetCurrentRepoSlug gets the current repository slug (owner/repo) using gh CLI
func GetCurrentRepoSlug() (string, error) {
	cmd := exec.Command("gh", "repo", "view", "--json", "owner,name", "--jq", ".owner.login + \"/\" + .name")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current repository: %w", err)
	}

	repoSlug := strings.TrimSpace(string(output))
	if repoSlug == "" {
		return "", fmt.Errorf("repository slug is empty")
	}

	// Validate format (should be owner/repo)
	parts := strings.Split(repoSlug, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository format: %s", repoSlug)
	}

	return repoSlug, nil
}