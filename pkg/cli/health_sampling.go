package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/repoutil"
)

var healthDiscoverWorkflows = fetchHealthWorkflows

func healthRepositoryEndpoint(repoOverride string) string {
	if repoOverride == "" {
		return "repos/{owner}/{repo}"
	}
	repository, _ := repoutil.NormalizeRepoForAPI(repoOverride)
	return "repos/" + repository
}

func discoverHealthWorkflows(config *HealthConfig) error {
	collection := config.collection
	collection.repoOverride = config.RepoOverride
	collection.MinimumRuns = healthMinimumRuns
	if collection.MaximumDays == 0 {
		collection.MaximumDays = config.Days
	}
	registered, err := healthDiscoverWorkflows(collection)
	if err != nil {
		if !collection.Partial {
			return err
		}
	}
	requested := config.WorkflowName
	for _, entry := range registered {
		if !strings.HasSuffix(entry.Path, ".lock.yml") {
			continue
		}
		if requested != "" {
			if requested != entry.Name && normalizeWorkflowID(requested) != normalizeWorkflowID(entry.Path) {
				continue
			}
			config.WorkflowName = entry.Name
		} else if entry.State != "active" {
			continue
		}
		collection.workflows = append(collection.workflows, entry)
	}
	if requested != "" && len(collection.workflows) > 1 {
		return fmt.Errorf("workflow name %q is ambiguous; specify its .github/workflows/ lock-file path", requested)
	}
	if config.WorkflowName != "" && len(collection.workflows) == 0 && !collection.Partial {
		return fmt.Errorf("agentic workflow %q not found in the repository workflow registry", config.WorkflowName)
	}
	slices.SortFunc(collection.workflows, func(a, b GitHubWorkflow) int { return strings.Compare(a.Path, b.Path) })
	return nil
}

func fetchHealthWorkflows(collection *HealthCollection) ([]GitHubWorkflow, error) {
	var registered []GitHubWorkflow
	for page := 1; page <= MaxIterations; page++ {
		endpoint := fmt.Sprintf("%s/actions/workflows?per_page=100&page=%d", healthRepositoryEndpoint(collection.repoOverride), page)
		out, err := healthAPI(collection, endpoint)
		if err != nil {
			return registered, err
		}
		var data struct {
			Workflows []GitHubWorkflow `json:"workflows"`
		}
		if err := json.Unmarshal(out, &data); err != nil {
			return registered, fmt.Errorf("failed to parse health workflow registry: %w", err)
		}
		registered = append(registered, data.Workflows...)
		if len(data.Workflows) < 100 {
			return registered, nil
		}
	}
	collection.Partial, collection.Reason = true, "workflow_registry_limit"
	return registered, nil
}

func listHealthWorkflowRuns(opts ListWorkflowRunsOptions, page int, collection *HealthCollection) ([]WorkflowRun, int, error) {
	endpoint := path.Join(healthRepositoryEndpoint(opts.RepoOverride), "actions/runs")
	if opts.WorkflowName != "" {
		endpoint = path.Join(healthRepositoryEndpoint(opts.RepoOverride), "actions/workflows", url.PathEscape(opts.WorkflowName), "runs")
	}
	endpoint += fmt.Sprintf("?per_page=%d&page=%d&created=%s", opts.Limit, page, url.QueryEscape(buildCreatedFilter(opts.StartDate, opts.EndDate, "")))
	out, err := healthAPI(collection, endpoint)
	if err != nil {
		return nil, 0, err
	}
	var data struct {
		Runs []workflowRunAPIResponse `json:"workflow_runs"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, 0, fmt.Errorf("failed to parse health runs: %w", err)
	}
	var runs []WorkflowRun
	for _, response := range data.Runs {
		if isNonDispatchedConclusion(response.Conclusion) {
			continue
		}
		run := response.workflowRun()
		runs = append(runs, run)
	}
	return runs, len(data.Runs), nil
}

func fetchWorkflowRuns(workflowName, startDate, repoOverride string, verbose bool, collections ...*HealthCollection) ([]WorkflowRun, error) {
	collection := &HealthCollection{Host: normalizedGitHubAPIHost(logsRateLimitHost(repoOverride)), repoOverride: repoOverride}
	for _, supplied := range collections {
		collection = supplied
	}
	targets := collection.workflows
	if len(targets) == 0 {
		if collection.Partial || collection.MaximumDays > 0 {
			return nil, nil
		}
		targets = []GitHubWorkflow{{Name: workflowName, Path: workflowName}}
	}
	var allRuns []WorkflowRun
	for _, target := range targets {
		runs, err := sampleHealthWorkflow(target, startDate, repoOverride, verbose, collection)
		allRuns = append(allRuns, runs...)
		if err != nil {
			if collection.stopped || errors.Is(err, errHealthAPIBudget) {
				return allRuns, nil
			}
			return allRuns, err
		}
	}
	return allRuns, nil
}

func sampleHealthWorkflow(target GitHubWorkflow, startDate, repoOverride string, verbose bool, collection *HealthCollection) ([]WorkflowRun, error) {
	var oldest time.Time
	workflowID := path.Base(target.Path)
	if target.ID > 0 {
		workflowID = strconv.FormatInt(target.ID, 10)
	}
	opts := ListWorkflowRunsOptions{WorkflowName: workflowID, StartDate: startDate, EndDate: collection.EndDate, Limit: healthMinimumRuns * 2,
		RepoOverride: repoOverride, Verbose: verbose, OldestFetchedCreatedAt: &oldest}
	var sample []WorkflowRun
	seen := make(map[int64]struct{})
	for page := 1; page <= MaxIterations; page++ {
		batch, total, err := healthListWorkflowRuns(opts, page, collection)
		if err != nil {
			return sample, err
		}
		if collection.visited == nil {
			collection.visited = make(map[string]struct{})
		}
		collection.visited[target.Path] = struct{}{}
		for _, run := range batch {
			if _, duplicate := seen[run.DatabaseID]; duplicate {
				continue
			}
			seen[run.DatabaseID] = struct{}{}
			if run.WorkflowPath == "" {
				run.WorkflowPath = target.Path
			}
			if run.WorkflowName == "" {
				run.WorkflowName = target.Name
			}
			if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
				run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			}
			if !collection.stopped {
				resolveHealthRevision(run, collection)
			}
			sample = append(sample, run)
		}
		if collection.stopped {
			return sample, errHealthAPIBudget
		}
		if total < opts.Limit || healthWorkflowEvidenceEnough(sample, collection) {
			return sample, nil
		}
		opts.BeforeDate = oldest.Format(time.RFC3339)
	}
	collection.Partial, collection.Reason = true, "pagination_limit"
	return sample, nil
}
