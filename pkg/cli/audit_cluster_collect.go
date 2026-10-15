package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

type auditClusterRequest struct {
	groupedAuditRunRequest
	metadata *WorkflowRun
}

func collectAuditClusters(ctx context.Context, args []string, opts auditCommandOptions, now time.Time) (AuditClusterReport, error) {
	report := AuditClusterReport{
		Schema: "audit-clusters/v1", Complete: true, AsOf: now,
		StalledAfter: opts.stalledAfter.String(),
		Runs:         []AuditClusterRun{}, Groups: []AuditClusterGroup{},
		Outcomes:     map[string]int{},
		FailedNoJobs: opts.failedNoJobs,
	}
	if len(args) == 0 {
		if start, end, err := auditClusterDateRange(opts, now); err == nil {
			report.StartDate, report.EndDate = start.Format(time.RFC3339), end.Format(time.RFC3339)
		}
	}
	requests, continuation, err := selectAuditClusterRuns(ctx, args, opts, now)
	report.Continuation = continuation
	if err != nil {
		report.Complete = false
		report.Errors = append(report.Errors, err.Error())
		report.Continuation = auditClusterRetryContinuation(args, opts, now)
		return report, err
	}
	if continuation != nil {
		report.Complete = false
	}
	for _, request := range requests {
		if err := ctx.Err(); err != nil {
			report.Complete = false
			report.Errors = append(report.Errors, err.Error())
			report.Continuation = auditClusterRetryContinuation(args, opts, now)
			report.Groups = groupAuditClusterRuns(report.Runs)
			return report, err
		}
		run := collectAuditClusterRun(ctx, request, opts, now)
		report.RunsExamined++
		if len(run.Evidence) > 0 {
			report.Complete = false
			report.Errors = append(report.Errors, fmt.Sprintf("%s/%s run %d: %s", run.Hostname, run.Repository, run.RunID, strings.Join(run.Evidence, "; ")))
		}
		if !opts.failedNoJobs || (run.Conclusion == "failure" && run.JobsState == "available" && run.JobCount != nil && *run.JobCount == 0) {
			report.Runs = append(report.Runs, run)
			report.RunsSelected++
			report.Outcomes[run.Outcome]++
		}
		if err := ctx.Err(); err != nil {
			report.Complete = false
			// Retry this run too: cancellation may have interrupted its evidence.
			report.Continuation = auditClusterRetryContinuation(args, opts, now)
			report.Errors = append(report.Errors, err.Error())
			report.Groups = groupAuditClusterRuns(report.Runs)
			return report, err
		}
	}
	report.Groups = groupAuditClusterRuns(report.Runs)
	slices.Sort(report.Errors)
	return report, nil
}

func auditClusterRetryContinuation(runArgs []string, opts auditCommandOptions, now time.Time) *AuditClusterContinuation {
	args := auditClusterCommonArgs(opts)
	if opts.repoFlag != "" {
		args = append(args, "--repo", opts.repoFlag)
	}
	if len(runArgs) > 0 {
		args = append(args, runArgs...)
	} else if start, end, err := auditClusterDateRange(opts, now); err == nil {
		args = append(args, "--start-date", start.Format(time.RFC3339), "--end-date", end.Format(time.RFC3339),
			"--cluster-offset", strconv.Itoa(opts.clusterOffset))
	}
	return &AuditClusterContinuation{Arguments: args, Reason: "collection interrupted; retry this selection"}
}

func auditClusterCommonArgs(opts auditCommandOptions) []string {
	args := []string{"--cluster", "--count", strconv.Itoa(opts.count), "--stalled-after", opts.stalledAfter.String()}
	if opts.failedNoJobs {
		args = append(args, "--failed-no-jobs")
	}
	if opts.jsonOutput {
		args = append(args, "--json")
	}
	if opts.outputDir != "" {
		args = append(args, "--output", opts.outputDir)
	}
	return args
}

func auditClusterPendingContinuation(requests []auditClusterRequest, opts auditCommandOptions, reason string) *AuditClusterContinuation {
	args := auditClusterCommonArgs(opts)
	for _, request := range requests {
		args = append(args, fmt.Sprintf("https://%s/%s/%s/actions/runs/%d", request.hostname, request.owner, request.repo, request.runID))
	}
	return &AuditClusterContinuation{Arguments: args, Reason: reason}
}

func selectAuditClusterRuns(ctx context.Context, args []string, opts auditCommandOptions, now time.Time) ([]auditClusterRequest, *AuditClusterContinuation, error) {
	if len(args) == 0 {
		return discoverAuditClusterRuns(ctx, opts, now)
	}
	var requests []auditClusterRequest
	seen := map[string]struct{}{}
	var defaultHost string
	for _, arg := range args {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		components, err := parser.ParseRunURLExtended(arg)
		if err != nil {
			return nil, nil, err
		}
		if components.Number <= 0 {
			return nil, nil, errors.New("clustering run IDs must be positive")
		}
		if components.JobID != 0 || components.StepNumber != 0 {
			return nil, nil, errors.New("clustering accepts run IDs or run URLs, not job or step URLs")
		}
		if err := applyAuditRepoFlag(opts.repoFlag, components); err != nil {
			return nil, nil, err
		}
		if components.Owner == "" || components.Repo == "" {
			return nil, nil, errors.New("bare run IDs require --repo owner/repo for clustering")
		}
		if err := validateAuditClusterRepository(components.Owner, components.Repo); err != nil {
			return nil, nil, err
		}
		host := components.Host
		if host == "" {
			if defaultHost == "" {
				defaultHost = resolveAuditHostname("")
			}
			host = defaultHost
		}
		if host == "." || host == ".." || strings.ContainsAny(host, "/\\") {
			return nil, nil, errors.New("invalid hostname for clustering")
		}
		request := auditClusterRequest{groupedAuditRunRequest: groupedAuditRunRequest{
			runID: components.Number, owner: components.Owner, repo: components.Repo,
			hostname: host,
		}}
		key := fmt.Sprintf("%s/%s/%s/%d", request.hostname, request.owner, request.repo, request.runID)
		if _, exists := seen[key]; !exists {
			requests = append(requests, request)
			seen[key] = struct{}{}
		}
	}
	slices.SortFunc(requests, func(a, b auditClusterRequest) int {
		return strings.Compare(fmt.Sprintf("%s/%s/%s/%020d", a.hostname, a.owner, a.repo, a.runID), fmt.Sprintf("%s/%s/%s/%020d", b.hostname, b.owner, b.repo, b.runID))
	})
	if len(requests) > opts.count {
		return requests[:opts.count], auditClusterPendingContinuation(requests[opts.count:], opts, "count limit reached"), nil
	}
	return requests, nil, nil
}

func discoverAuditClusterRuns(ctx context.Context, opts auditCommandOptions, now time.Time) ([]auditClusterRequest, *AuditClusterContinuation, error) {
	components := &parser.GitHubURLComponents{}
	if opts.repoFlag == "" {
		return nil, nil, errors.New("date-range clustering requires --repo owner/repo")
	}
	if err := applyAuditRepoFlag(opts.repoFlag, components); err != nil {
		return nil, nil, err
	}
	if err := validateAuditClusterRepository(components.Owner, components.Repo); err != nil {
		return nil, nil, err
	}
	host := resolveAuditHostname("")
	start, end, err := auditClusterDateRange(opts, now)
	if err != nil {
		return nil, nil, err
	}
	startDate, endDate := start.Format(time.RFC3339), end.Format(time.RFC3339)
	requests := []auditClusterRequest{}
	seen := map[int64]struct{}{}
	position := opts.clusterOffset
	for len(requests) < opts.count {
		if err := ctx.Err(); err != nil {
			return requests, nil, err
		}
		page := position/100 + 1
		base := groupedAuditRunRequest{owner: components.Owner, repo: components.Repo, hostname: host}
		response, err := fetchAuditClusterPage(ctx, base, startDate, endDate, page)
		if err != nil {
			return requests, nil, err
		}
		requests, position, err = appendAuditClusterPage(requests, response.Runs, base, seen, position, opts.count)
		if err != nil {
			return requests, nil, err
		}
		if position >= *response.Total {
			return requests, nil, nil
		}
		if len(requests) == opts.count || position >= 1000 || len(response.Runs) < 100 {
			return requests, auditClusterDiscoveryContinuation(opts, startDate, endDate, position, page, len(response.Runs)), nil
		}
	}
	return requests, nil, nil
}

type auditClusterPage struct {
	Total *int          `json:"total_count"`
	Runs  []WorkflowRun `json:"runs"`
}

func fetchAuditClusterPage(ctx context.Context, base groupedAuditRunRequest, startDate, endDate string, page int) (auditClusterPage, error) {
	query := url.Values{
		"created":  {buildCreatedFilter(startDate, endDate, "")},
		"per_page": {"100"}, "page": {strconv.Itoa(page)},
	}
	endpoint := fmt.Sprintf("repos/%s/%s/actions/runs?%s", base.owner, base.repo, query.Encode())
	apiArgs := []string{"api", endpoint, "--jq", fmt.Sprintf("{total_count: .total_count, runs: [.workflow_runs[] | %s]}", workflowRunMetadataQuery)}
	if base.hostname != "github.com" {
		apiArgs = append(apiArgs, "--hostname", base.hostname)
	}
	output, err := workflow.RunGHCombinedContext(ctx, "Listing runs for clustering...", apiArgs...)
	if err != nil {
		return auditClusterPage{}, errors.Join(fmt.Errorf("failed to list clustering runs: %w", err), ctx.Err())
	}
	var response auditClusterPage
	if err := json.Unmarshal(output, &response); err != nil {
		return response, fmt.Errorf("failed to parse clustering runs: %w", err)
	}
	if response.Total == nil || *response.Total < 0 || response.Runs == nil {
		return response, errors.New("incomplete run-list response: total_count and runs are required")
	}
	return response, nil
}

func appendAuditClusterPage(requests []auditClusterRequest, runs []WorkflowRun, base groupedAuditRunRequest, seen map[int64]struct{}, position, count int) ([]auditClusterRequest, int, error) {
	offset := position % 100
	for index, run := range runs {
		if index < offset {
			continue
		}
		if run.DatabaseID <= 0 {
			return requests, position, errors.New("run-list response contains a missing run ID")
		}
		position++
		if _, exists := seen[run.DatabaseID]; exists {
			continue
		}
		seen[run.DatabaseID] = struct{}{}
		base.runID = run.DatabaseID
		requests = append(requests, auditClusterRequest{groupedAuditRunRequest: base, metadata: &run})
		if len(requests) == count {
			break
		}
	}
	return requests, position, nil
}

func auditClusterDiscoveryContinuation(opts auditCommandOptions, startDate, endDate string, position, page, pageSize int) *AuditClusterContinuation {
	if position >= 1000 || (pageSize < 100 && position >= (page-1)*100+pageSize) {
		return &AuditClusterContinuation{Reason: "GitHub search result limit reached; narrow the date range", RequiresNarrowerRange: true}
	}
	args := append(auditClusterCommonArgs(opts), "--repo", opts.repoFlag, "--start-date", startDate, "--end-date", endDate, "--cluster-offset", strconv.Itoa(position))
	return &AuditClusterContinuation{Arguments: args, Reason: "count limit reached"}
}

func validateAuditClusterRepository(owner, repo string) error {
	for _, component := range []string{owner, repo} {
		if component == "" || component == "." || component == ".." || strings.ContainsAny(component, "/\\?#%") {
			return errors.New("invalid repository: expected owner/repo without path or query components")
		}
	}
	return nil
}

func collectAuditClusterRun(ctx context.Context, request auditClusterRequest, opts auditCommandOptions, now time.Time) AuditClusterRun {
	result := AuditClusterRun{
		RunID: request.runID, Hostname: request.hostname, Repository: fmt.Sprintf("%s/%s", request.owner, request.repo),
		URL:     fmt.Sprintf("https://%s/%s/%s/actions/runs/%d", request.hostname, request.owner, request.repo, request.runID),
		Outcome: "unknown", JobsState: "unknown", UsageState: "unknown", Signatures: []string{},
	}
	run := applyAuditClusterMetadata(ctx, request, opts, now, &result)
	request.metadata = run
	collectAuditClusterJobEvidence(ctx, request, &result)
	// Usage collection is deliberately independent of job availability, count,
	// and conclusion. Startup failures can still carry useful usage evidence.
	outputDir := resolveAuditOutputDir(filepath.Join(opts.outputDir, "clusters", request.hostname, request.owner, request.repo), request.runID)
	if run != nil && run.Attempt > 0 {
		outputDir = filepath.Join(outputDir, fmt.Sprintf("attempt-%d", run.Attempt))
	}
	collectAuditClusterUsage(ctx, request, outputDir, opts.verbose, &result)
	if len(result.Signatures) == 0 {
		result.Signatures = append(result.Signatures, "outcome:"+result.Outcome)
	}
	slices.Sort(result.Signatures)
	result.Signatures = slices.Compact(result.Signatures)
	return result
}

func applyAuditClusterMetadata(ctx context.Context, request auditClusterRequest, opts auditCommandOptions, now time.Time, result *AuditClusterRun) *WorkflowRun {
	run := request.metadata
	if run == nil {
		metadata, err := fetchWorkflowRunMetadata(ctx, request.runID, request.owner, request.repo, request.hostname, opts.verbose)
		if err != nil {
			result.Evidence = append(result.Evidence, "metadata: "+err.Error())
		} else {
			run = &metadata
		}
	}
	if run != nil {
		if run.URL != "" {
			result.URL = run.URL
		}
		result.Workflow = run.WorkflowName
		result.WorkflowPath, result.HeadSHA, result.Event = run.WorkflowPath, run.HeadSha, run.Event
		result.CreatedAt, result.HeadBranch = run.CreatedAt, run.HeadBranch
		result.Status, result.Conclusion, result.UpdatedAt = run.Status, run.Conclusion, run.UpdatedAt
		result.Outcome = auditClusterOutcome(*run, now, opts.stalledAfter)
	}
	return run
}

func collectAuditClusterJobEvidence(ctx context.Context, request auditClusterRequest, result *AuditClusterRun) {
	jobs, err := fetchAuditClusterJobs(ctx, request)
	if err != nil {
		result.Evidence = append(result.Evidence, "jobs: "+err.Error())
		result.Signatures = append(result.Signatures, "jobs:unknown")
		return
	}
	count := len(jobs)
	result.JobCount, result.JobsState = &count, "available"
	if count == 0 {
		result.Signatures = append(result.Signatures, "jobs:none")
	}
	for _, job := range jobs {
		if job.Conclusion != "" && job.Conclusion != "success" && job.Conclusion != "skipped" && job.Conclusion != "neutral" {
			result.Signatures = append(result.Signatures, auditClusterJobSignature(job))
		}
	}
}

func auditClusterJobSignature(job JobInfo) string {
	var failedSteps []string
	for _, step := range job.Steps {
		if step.Conclusion != "" && step.Conclusion != "success" && step.Conclusion != "skipped" {
			failedSteps = append(failedSteps, fmt.Sprintf("/step:%s:%s", step.Name, step.Conclusion))
		}
	}
	slices.Sort(failedSteps)
	return fmt.Sprintf("job:%s:%s%s", job.Name, job.Conclusion, strings.Join(failedSteps, ""))
}

func fetchAuditClusterJobs(ctx context.Context, request auditClusterRequest) ([]JobInfo, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs?per_page=100&filter=latest", request.owner, request.repo, request.runID)
	args := []string{"api", endpoint, "--paginate", "--slurp"}
	if request.hostname != "github.com" && request.hostname != "" {
		args = append(args, "--hostname", request.hostname)
	}
	output, err := workflow.RunGHCombinedContext(ctx, "Fetching cluster job evidence...", args...)
	if err != nil {
		return nil, err
	}
	return parseAuditClusterJobs(output)
}

// Unlike the deep-audit parser, malformed pages must not silently become zero jobs.
func parseAuditClusterJobs(output []byte) ([]JobInfo, error) {
	var pages []struct {
		Total *int      `json:"total_count"`
		Jobs  []JobInfo `json:"jobs"`
	}
	if err := json.Unmarshal(output, &pages); err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, errors.New("jobs response contains no pages")
	}
	jobs := []JobInfo{}
	seen := map[int64]struct{}{}
	total := -1
	for _, page := range pages {
		if page.Total == nil || page.Jobs == nil || *page.Total < 0 {
			return nil, errors.New("jobs response lacks total_count or jobs")
		}
		if total >= 0 && total != *page.Total {
			return nil, errors.New("jobs changed during pagination; retry")
		}
		total = *page.Total
		for _, job := range page.Jobs {
			if job.ID == 0 || job.Name == "" {
				return nil, errors.New("jobs response contains a malformed job")
			}
			if _, exists := seen[job.ID]; exists {
				return nil, errors.New("jobs response contains duplicate jobs; retry")
			}
			seen[job.ID] = struct{}{}
			jobs = append(jobs, job)
		}
	}
	if len(jobs) != total {
		return nil, errors.New("jobs response is incomplete")
	}
	return jobs, nil
}

func collectAuditClusterUsage(ctx context.Context, request auditClusterRequest, outputDir string, verbose bool, result *AuditClusterRun) {
	outputDir, cleanup, err := auditClusterUsageDirectory(outputDir)
	if err != nil {
		result.Evidence = append(result.Evidence, "usage directory: "+err.Error())
		result.Signatures = append(result.Signatures, "usage:unknown")
		return
	}
	defer cleanup()
	err = downloadRunArtifacts(ctx, downloadArtifactsOptions{
		runID: request.runID, owner: request.owner, repo: request.repo, hostname: request.hostname,
		outputDir: outputDir, verbose: verbose, artifactFilter: ResolveArtifactFilter([]string{"usage"}),
	})
	if err != nil {
		result.Evidence = append(result.Evidence, "usage download: "+err.Error())
	}
	if ctx.Err() != nil {
		return
	}
	collectAuditClusterUsageArtifacts(outputDir, request, verbose, result)
	if result.UsageState == "unknown" {
		result.Signatures = append(result.Signatures, "usage:unknown")
		if err == nil && len(result.Evidence) == 0 {
			result.Evidence = append(result.Evidence, "usage: no supported usage summary or token records found")
		}
	}
}
