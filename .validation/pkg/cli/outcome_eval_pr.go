package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var outcomeEvalPRLog = logger.New("cli:outcome_eval_pr")

// findPRByTimestamp searches for a PR created by github-actions[bot] around the given timestamp.
// This is a fallback for when the manifest doesn't record the PR number.
func findPRByTimestamp(repo string, timestamp string) int {
	outcomeEvalPRLog.Printf("Searching for PR by timestamp: repo=%s, timestamp=%s", repo, timestamp)
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		outcomeEvalPRLog.Printf("Failed to parse timestamp %q: %v", timestamp, err)
		return 0
	}
	// Search in a 5-minute window around the timestamp
	since := ts.Add(-2 * time.Minute).Format("2006-01-02T15:04:05Z")
	until := ts.Add(5 * time.Minute).Format("2006-01-02T15:04:05Z")

	// Use gh CLI to search for PRs
	output, err := workflow.RunGH("Searching for PR...",
		"pr", "list",
		"--repo", repo,
		"--state", "all",
		"--author", "app/github-actions",
		"--search", fmt.Sprintf("created:%s..%s", since, until),
		"--limit", "1",
		"--json", "number",
		"--jq", ".[0].number")
	if err != nil {
		return 0
	}
	numStr := strings.TrimSpace(string(output))
	var n int
	if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil {
		return n
	}
	return 0
}

// evalCreatePullRequest checks whether a PR was merged, closed, or is still open.
func evalCreatePullRequest(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	outcomeEvalPRLog.Printf("Evaluating create_pull_request: repo=%s, num=%d, url=%s", repo, num, item.URL)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}

	// If no PR number, try to find the PR by searching recent PRs from github-actions
	if num == 0 && repo != "" {
		found := findPRByTimestamp(repo, item.Timestamp)
		if found > 0 {
			outcomeEvalPRLog.Printf("Resolved missing PR number via timestamp search: num=%d", found)
			num = found
			report.ObjectNumber = num
		}
	}

	if num == 0 || repo == "" {
		report.Result = OutcomeError
		report.EvalError = "missing PR number or repo"
		return report
	}

	data, err := ghAPIGet(fmt.Sprintf("pulls/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	merged, _ := data["merged"].(bool)
	state, _ := data["state"].(string)
	mergedAt, _ := data["merged_at"].(string)
	closedAt, _ := data["closed_at"].(string)

	switch {
	case merged:
		report.Result = OutcomeAccepted
		report.Detail = "merged"
		if mergedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, mergedAt)
		}
	case state == "closed":
		report.Result = OutcomeRejected
		report.Detail = "closed without merge"
		if closedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, closedAt)
		}
	default:
		report.Result = OutcomePending
		report.Detail = "open"
	}

	// Count human comments (non-bot)
	comments, err := ghAPIGetArray(fmt.Sprintf("issues/%d/comments", num), repo)
	if err == nil {
		for _, c := range comments {
			user, _ := c["user"].(map[string]any)
			login, _ := user["login"].(string)
			if !isBotUser(login) {
				report.HumanComments++
			}
		}
	}

	// Count reviews (used for ZeroTouch, stored separately from edits to avoid conflation)
	reviews, err := ghAPIGetArray(fmt.Sprintf("pulls/%d/reviews", num), repo)
	if err == nil {
		report.HumanReviews = len(reviews)
	}

	if report.Result == OutcomeAccepted {
		report.ZeroTouch = report.HumanComments == 0 && report.HumanReviews == 0
	}

	return report
}
