package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"
)

var healthListFailureCandidates = listHealthFailureCandidates

// Collect explicit failures before spending the shared budget on successful runs.
func collectHealthFailures(startDate, repoOverride string, collection *HealthCollection) ([]WorkflowRun, error) {
	targets := make(map[string]GitHubWorkflow)
	for _, target := range collection.workflows {
		targets[target.Path] = target
	}
	opts := ListWorkflowRunsOptions{StartDate: startDate, EndDate: collection.EndDate, Limit: 100}
	seen := make(map[int64]struct{})
	scopes := make(map[string]int)
	var runs []WorkflowRun
	for page := 1; ; page++ {
		batch, oldest, err := healthListFailureCandidates(opts, page, repoOverride, collection)
		if err != nil {
			return runs, err
		}
		for _, response := range batch {
			run := response.workflowRun()
			target, registered := targets[run.WorkflowPath]
			if _, duplicate := seen[run.DatabaseID]; duplicate || !registered || run.Conclusion != "failure" {
				continue
			}
			seen[run.DatabaseID] = struct{}{}
			if run.WorkflowName == "" {
				run.WorkflowName = target.Name
			}
			scope := failureClusterKey(run.Repository, run.WorkflowPath, run.HeadBranch, run.Event)
			if scopes[scope] >= healthMinimumRuns {
				continue
			}
			hasJobs, err := healthRunHasJobs(run, collection)
			if err != nil {
				return runs, err
			}
			if !hasJobs {
				continue
			}
			resolveHealthRevision(run, collection)
			if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
				run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			}
			runs = append(runs, run)
			scopes[scope]++
			if healthSourceChanged(run, collection) {
				scopes[scope] = healthMinimumRuns
			}
			if collection.stopped {
				return runs, errHealthAPIBudget
			}
		}
		if len(batch) < opts.Limit {
			collection.FailurePassComplete = true
			return runs, nil
		}
		if page*opts.Limit >= healthMaximumRunSearchResults {
			if !advanceHealthRunWindow(&opts, oldest, collection) {
				return runs, nil
			}
			page = 0
		}
	}
}

func listHealthFailureCandidates(opts ListWorkflowRunsOptions, page int, repoOverride string, collection *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
	endpoint := fmt.Sprintf("%s/actions/runs?status=failure&per_page=%d&page=%d&created=%s",
		healthRepositoryEndpoint(repoOverride), opts.Limit, page,
		url.QueryEscape(buildCreatedFilter(opts.StartDate, opts.EndDate, opts.BeforeDate)))
	out, err := healthAPI(collection, endpoint)
	if err != nil {
		return nil, time.Time{}, err
	}
	var data struct {
		Runs []workflowRunAPIResponse `json:"workflow_runs"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to parse health failure runs: %w", err)
	}
	if data.Runs == nil {
		return nil, time.Time{}, errors.New("failed to parse health failure runs: missing workflow_runs array")
	}
	var oldest time.Time
	for _, run := range data.Runs {
		if oldest.IsZero() || run.CreatedAt.Before(oldest) {
			oldest = run.CreatedAt
		}
	}
	return data.Runs, oldest, nil
}

func prioritizeHealthFailureWorkflows(targets []GitHubWorkflow, failures []WorkflowRun) []GitHubWorkflow {
	recent := make(map[string]time.Time)
	for _, run := range failures {
		if run.CreatedAt.After(recent[run.WorkflowPath]) {
			recent[run.WorkflowPath] = run.CreatedAt
		}
	}
	ordered := slices.Clone(targets)
	slices.SortStableFunc(ordered, func(a, b GitHubWorkflow) int {
		return recent[b.Path].Compare(recent[a.Path])
	})
	return ordered
}

func mergeHealthRunSamples(failures, runs []WorkflowRun) []WorkflowRun {
	seen := make(map[int64]struct{})
	var merged []WorkflowRun
	for _, sample := range [][]WorkflowRun{runs, failures} {
		for _, run := range sample {
			if _, duplicate := seen[run.DatabaseID]; duplicate {
				continue
			}
			seen[run.DatabaseID] = struct{}{}
			merged = append(merged, run)
		}
	}
	return merged
}
