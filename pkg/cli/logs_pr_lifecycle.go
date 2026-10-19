package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
)

// ObsoletePREvidence records lifecycle evidence, not a diagnosis of GitHub's failure.
type ObsoletePREvidence struct {
	Number     int       `json:"number"`
	ClosedAt   time.Time `json:"closed_at"`
	RunAttempt int       `json:"run_attempt"`
}

func applyRunExecutionEvidence(data *RunData, pr ProcessedRun) {
	if pr.JobDetails != nil {
		count := len(pr.JobDetails)
		data.JobCount = &count
	}
	data.ObsoletePR = pr.Run.ObsoletePR
}

func addRunExecutionSummary(summary *LogsSummary, runs []RunData) {
	for _, run := range runs {
		if run.JobCount != nil && *run.JobCount == 0 {
			summary.NonExecutionRuns++
		}
		if isObsoletePRRun(run) {
			summary.ObsoletePRRuns++
		}
		if run.JobCount != nil && *run.JobCount > 0 && run.Status == "completed" {
			summary.ExecutedRuns++
			if run.Conclusion == "success" {
				summary.ExecutedSuccessRuns++
			}
		}
	}
}

func obsoletePRWindow(run WorkflowRun) (time.Time, bool) {
	queuedAt := run.StartedAt
	if queuedAt.IsZero() {
		queuedAt = run.CreatedAt
	}
	return queuedAt, run.Event == "pull_request" && run.Status == "completed" &&
		(run.Conclusion == "failure" || run.Conclusion == "cancelled") &&
		run.Attempt > 0 && run.HeadSha != "" && !queuedAt.IsZero() && !run.UpdatedAt.IsZero()
}

// GitHub can drop pull_requests from a run after merge. Resolve the exact head
// commit instead of guessing a PR number from its branch name or display title.
func fetchObsoletePREvidence(ctx context.Context, run WorkflowRun) (*ObsoletePREvidence, error) {
	queuedAt, eligible := obsoletePRWindow(run)
	if !eligible {
		return nil, nil
	}
	output, err := workflow.RunGHCombinedContext(ctx, "Checking PR lifecycle...", "api",
		fmt.Sprintf("repos/{owner}/{repo}/commits/%s/pulls?per_page=100", url.PathEscape(run.HeadSha)),
		"--paginate", "--slurp")
	if err != nil {
		return nil, fmt.Errorf("failed to check PR lifecycle for run %d: %w", run.DatabaseID, err)
	}
	var pages [][]struct {
		Number   int       `json:"number"`
		State    string    `json:"state"`
		ClosedAt time.Time `json:"closed_at"`
		Head     struct {
			Ref string `json:"ref"`
		} `json:"head"`
	}
	if err := json.Unmarshal(output, &pages); err != nil {
		return nil, fmt.Errorf("failed to parse PR lifecycle for run %d: %w", run.DatabaseID, err)
	}
	total := 0
	for _, page := range pages {
		for _, pr := range page {
			if run.HeadBranch != "" && pr.Head.Ref != "" && pr.Head.Ref != run.HeadBranch {
				continue
			}
			total++
		}
	}
	// Only affirmative branch mismatches disambiguate a shared commit. Missing
	// refs remain candidates, and multiple matching PRs must remain unknown.
	if total != 1 {
		return nil, nil
	}
	for _, page := range pages {
		for _, pr := range page {
			if pr.Number > 0 && pr.State == "closed" &&
				(run.HeadBranch == "" || pr.Head.Ref == run.HeadBranch) &&
				pr.ClosedAt.After(queuedAt) && !pr.ClosedAt.After(run.UpdatedAt) {
				return &ObsoletePREvidence{Number: pr.Number, ClosedAt: pr.ClosedAt, RunAttempt: run.Attempt}, nil
			}
		}
	}
	return nil, nil
}

func isObsoletePRRun(run RunData) bool {
	proof := run.ObsoletePR
	queuedAt := run.StartedAt
	if queuedAt.IsZero() {
		queuedAt = run.CreatedAt
	}
	return proof != nil && proof.Number > 0 && run.JobCount != nil && *run.JobCount == 0 &&
		len(run.FailureSignals) == 0 && run.Event == "pull_request" &&
		run.Status == "completed" && run.RunAttempt == strconv.Itoa(proof.RunAttempt) &&
		proof.RunAttempt > 0 && !queuedAt.IsZero() && proof.ClosedAt.After(queuedAt) &&
		!run.UpdatedAt.IsZero() && !proof.ClosedAt.After(run.UpdatedAt) &&
		(run.Conclusion == "failure" || run.Conclusion == "cancelled")
}
