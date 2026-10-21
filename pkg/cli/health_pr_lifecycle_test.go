//go:build !integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func healthLifecycleRun() WorkflowRun {
	start := time.Date(2026, 10, 21, 14, 39, 3, 0, time.UTC)
	return WorkflowRun{DatabaseID: 42, Attempt: 1, Repository: "org/repo", WorkflowName: "Reviewer",
		WorkflowPath: ".github/workflows/review.lock.yml", Event: "pull_request",
		HeadSha: strings.Repeat("a", 40), HeadBranch: "feature", Status: "completed", Conclusion: "cancelled",
		CreatedAt: start, StartedAt: start, UpdatedAt: start.Add(5 * time.Minute)}
}

func healthLifecycleCollection() *HealthCollection {
	return &HealthCollection{Host: "github.example.com", repoOverride: "org/repo",
		APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
}

func lifecyclePRJSON(run WorkflowRun, state string, closed, merged time.Time) string {
	pr := healthLifecyclePR{Number: 123, State: state, ClosedAt: closed, MergedAt: merged}
	pr.Head.Ref = run.HeadBranch
	data, _ := json.Marshal([]healthLifecyclePR{pr})
	return string(data)
}

func lifecycleJobsJSON(run WorkflowRun, failure bool) string {
	job := JobInfoWithDuration{JobInfo: JobInfo{ID: 99, Name: "agent", Status: "completed", Conclusion: "cancelled",
		CompletedAt: run.UpdatedAt.Add(-time.Minute)}}
	if failure {
		job.Conclusion = "failure"
		job.Steps = []JobStep{{Name: "Checkout PR branch", Conclusion: "failure", Status: "completed",
			CompletedAt: job.CompletedAt}}
	}
	jobs := []JobInfoWithDuration{job}
	if failure {
		jobs = append(jobs, JobInfoWithDuration{JobInfo: JobInfo{ID: 100, Name: "conclusion",
			Status: "completed", Conclusion: "failure", Steps: []JobStep{{Name: "Handle agent failure",
				Conclusion: "failure", StartedAt: run.UpdatedAt.Add(-30 * time.Second)}}}})
	}
	data, _ := json.Marshal(map[string]any{"jobs": jobs, "total_count": len(jobs)})
	return string(data)
}

func stubHealthLifecycleAPI(t *testing.T, responses map[string]string, apiError error) *int {
	t.Helper()
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	requests := 0
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		requests++
		require.Equal(t, "github.example.com", args[2])
		require.True(t, strings.HasPrefix(args[3], "repos/org/repo/"), args[3])
		if strings.HasSuffix(args[3], "/logs") {
			assert.Contains(t, args, "--allow-escape-sequences", "logs are inspected internally, never printed to the terminal")
		}
		for suffix, response := range responses {
			if strings.HasSuffix(args[3], suffix) {
				return []byte(response), nil
			}
		}
		if apiError != nil {
			return nil, apiError
		}
		t.Fatalf("unexpected API request: %s", args[3])
		return nil, nil
	}
	return &requests
}

func TestHealthPRLifecycleClassification(t *testing.T) {
	for _, name := range []string{
		"merged", "closed", "deleted-head", "restored-head", "open", "closed-later", "closed-before-cancellation",
		"closed-after-cancelled-job", "missing-cancelled-job-time",
		"genuine-job-failure", "genuine-step-failure", "missing-jobs", "malformed-jobs",
		"incomplete-jobs", "missing-job-status", "missing-pr", "ambiguous-pr", "missing-pr-ref", "missing-closure-time",
		"unavailable-pr", "unavailable-events", "incomplete-events", "missing-run-time", "missing-attempt",
		"failed-closed-checkout", "failed-old-closed-checkout", "generic-checkout-failure",
		"other-failed-step", "unavailable-checkout-log", "failure-without-checkout",
		"unrelated-conclusion-failure", "wrong-pr-diagnostic", "closure-after-checkout", "premature-aftermath",
		"timeout", "schedule", "success",
	} {
		t.Run(name, func(t *testing.T) {
			run := healthLifecycleRun()
			collection := healthLifecycleCollection()
			closed := run.StartedAt.Add(time.Minute)
			responses := map[string]string{
				"/pulls?per_page=100":         lifecyclePRJSON(run, "closed", closed, closed),
				"/jobs?per_page=100&page=1":   lifecycleJobsJSON(run, false),
				"/events?per_page=100&page=1": `[]`,
				"/99/logs":                    "warning: PR #123 closed before its head could be fetched. Stopping rather than running on the previous checkout.",
			}
			expected := "retained"
			var apiError error
			switch name {
			case "merged":
				expected = "ignored"
			case "closed":
				expected = "ignored"
				responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "closed", closed, time.Time{})
			case "deleted-head", "restored-head":
				responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "open", time.Time{}, time.Time{})
				events := fmt.Sprintf(`[{"event":"head_ref_deleted","created_at":%q}]`, closed.Format(time.RFC3339))
				expected = "ignored"
				if name == "restored-head" {
					events = strings.TrimSuffix(events, "]") + fmt.Sprintf(`,{"event":"head_ref_restored","created_at":%q}]`,
						closed.Add(time.Minute).Format(time.RFC3339))
					expected = "retained"
				}
				responses["/events?per_page=100&page=1"] = events
			case "open", "unavailable-events", "incomplete-events":
				responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "open", time.Time{}, time.Time{})
				switch name {
				case "unavailable-events":
					delete(responses, "/events?per_page=100&page=1")
					apiError, expected = errors.New("HTTP 403 forbidden"), "unknown"
				case "incomplete-events":
					page := "[" + strings.Repeat(`{"event":"commented"},`, 99) + `{"event":"commented"}]`
					responses["/events?per_page=100&page=1"], responses["/events?per_page=100&page=2"] = page, page
					expected = "unknown"
				}
			case "closed-later":
				responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "closed", run.UpdatedAt.Add(time.Minute), time.Time{})
			case "closed-before-cancellation":
				responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "closed", run.StartedAt.Add(-time.Minute), time.Time{})
			case "closed-after-cancelled-job", "missing-cancelled-job-time":
				expected = "unknown"
				completed := closed.Add(-time.Second)
				if name == "missing-cancelled-job-time" {
					completed = time.Time{}
				}
				responses["/jobs?per_page=100&page=1"] = fmt.Sprintf(`{"jobs":[{"status":"completed","conclusion":"cancelled","completed_at":%q}]}`,
					completed.Format(time.RFC3339))
			case "genuine-job-failure":
				responses["/jobs?per_page=100&page=1"] = lifecycleJobsJSON(run, true)
			case "genuine-step-failure":
				responses["/jobs?per_page=100&page=1"] = `{"jobs":[{"status":"completed","conclusion":"cancelled","steps":[{"conclusion":"failure"}]}]}`
			case "missing-jobs", "malformed-jobs", "incomplete-jobs", "missing-job-status":
				expected = "unknown"
				jobs := map[string]string{"missing-jobs": `{}`, "malformed-jobs": `not-json`,
					"incomplete-jobs": `{"jobs":[],"total_count":1}`, "missing-job-status": `{"jobs":[{"id":1}]}`}
				responses["/jobs?per_page=100&page=1"] = jobs[name]
				responses["/jobs?per_page=100&page=2"], responses["/jobs?per_page=100&page=3"] = jobs[name], jobs[name]
			case "missing-pr":
				expected, responses["/pulls?per_page=100"] = "unknown", `[]`
			case "ambiguous-pr":
				expected = "unknown"
				prs := responses["/pulls?per_page=100"]
				responses["/pulls?per_page=100"] = strings.TrimSuffix(prs, "]") + "," + strings.TrimPrefix(prs, "[")
			case "missing-pr-ref":
				expected = "unknown"
				responses["/pulls?per_page=100"] = `[{"number":123,"state":"closed"}]`
			case "missing-closure-time":
				expected = "unknown"
				responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "closed", time.Time{}, time.Time{})
			case "unavailable-pr":
				delete(responses, "/pulls?per_page=100")
				apiError, expected = errors.New("HTTP 404 not found"), "unknown"
			case "missing-run-time":
				run.UpdatedAt, expected = time.Time{}, "unknown"
			case "missing-attempt":
				run.Attempt, expected = 0, "unknown"
			case "failed-closed-checkout", "failed-old-closed-checkout", "generic-checkout-failure",
				"other-failed-step", "unavailable-checkout-log", "failure-without-checkout", "unrelated-conclusion-failure",
				"wrong-pr-diagnostic", "closure-after-checkout", "premature-aftermath":
				run.Conclusion = "failure"
				responses["/jobs?per_page=100&page=1"] = lifecycleJobsJSON(run, true)
				switch name {
				case "failed-closed-checkout":
					expected = "ignored"
				case "failed-old-closed-checkout":
					expected = "ignored"
					responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "closed", run.StartedAt.Add(-time.Minute), time.Time{})
				case "generic-checkout-failure":
					responses["/99/logs"] = "fatal: couldn't find remote ref feature"
				case "other-failed-step":
					responses["/jobs?per_page=100&page=1"] = strings.ReplaceAll(lifecycleJobsJSON(run, true), "Checkout PR branch", "Run agent")
				case "unavailable-checkout-log":
					delete(responses, "/99/logs")
					apiError, expected = errors.New("HTTP 410 logs expired"), "unknown"
				case "failure-without-checkout":
					responses["/jobs?per_page=100&page=1"] = lifecycleJobsJSON(run, false)
					expected = "unknown"
				case "unrelated-conclusion-failure":
					responses["/jobs?per_page=100&page=1"] = strings.ReplaceAll(lifecycleJobsJSON(run, true), "Handle agent failure", "Upload report")
				case "wrong-pr-diagnostic":
					responses["/99/logs"] = strings.ReplaceAll(responses["/99/logs"], "#123", "#456")
				case "closure-after-checkout":
					responses["/pulls?per_page=100"] = lifecyclePRJSON(run, "closed", run.UpdatedAt.Add(-time.Second), time.Time{})
				case "premature-aftermath":
					responses["/jobs?per_page=100&page=1"] = strings.ReplaceAll(lifecycleJobsJSON(run, true),
						run.UpdatedAt.Add(-30*time.Second).Format(time.RFC3339), run.StartedAt.Format(time.RFC3339))
				}
			case "timeout":
				run.Conclusion = "timed_out"
			case "schedule":
				run.Event = "schedule"
			case "success":
				run.Conclusion = "success"
			}
			requests := stubHealthLifecycleAPI(t, responses, apiError)
			ignored := filterHealthPRLifecycleRun(run, collection)
			assert.Equal(t, expected == "ignored", ignored)
			if name != "timeout" && name != "schedule" && name != "success" {
				assert.Equal(t, expected, collection.PRLifecycleEvidence[run.DatabaseID].Status)
				var output bytes.Buffer
				require.NoError(t, renderHealthCollection(&output, collection))
				assert.Contains(t, output.String(), "status="+expected)
			} else {
				assert.Zero(t, *requests)
			}
			before := *requests
			assert.Equal(t, ignored, filterHealthPRLifecycleRun(run, collection))
			assert.Equal(t, before, *requests, "completed attempt classification is cached")
			assert.LessOrEqual(t, collection.LifecycleRequests, healthMaximumLifecycleRequests)
		})
	}
}

func TestHealthLifecycleRequestBudgetAndSharedCache(t *testing.T) {
	run := healthLifecycleRun()
	collection := healthLifecycleCollection()
	requests := stubHealthLifecycleAPI(t, map[string]string{
		"/pulls?per_page=100":       lifecyclePRJSON(run, "closed", run.StartedAt.Add(time.Minute), time.Time{}),
		"/jobs?per_page=100&page=1": lifecycleJobsJSON(run, false),
	}, nil)
	require.True(t, filterHealthPRLifecycleRun(run, collection))
	run.DatabaseID++
	require.True(t, filterHealthPRLifecycleRun(run, collection))
	assert.Equal(t, 3, *requests, "same commit lifecycle is reused across workflows; jobs remain run-specific")
	collection.LifecycleRequests = healthMaximumLifecycleRequests
	run.DatabaseID++
	run.HeadSha = strings.Repeat("b", 40)
	require.False(t, filterHealthPRLifecycleRun(run, collection))
	assert.Equal(t, "unknown", collection.PRLifecycleEvidence[run.DatabaseID].Status)
	assert.Contains(t, collection.PRLifecycleEvidence[run.DatabaseID].Reason, "limit")
	assert.Equal(t, 3, *requests)
}

func TestHealthLifecycleCachesUnavailableEvidence(t *testing.T) {
	run := healthLifecycleRun()
	collection := healthLifecycleCollection()
	requests := stubHealthLifecycleAPI(t, nil, errors.New("HTTP 403 forbidden"))
	assert.False(t, filterHealthPRLifecycleRun(run, collection))
	run.DatabaseID++
	assert.False(t, filterHealthPRLifecycleRun(run, collection))
	assert.Equal(t, 1, *requests)
	assert.Equal(t, "unknown", collection.PRLifecycleEvidence[run.DatabaseID].Status)
	assert.Contains(t, collection.PRLifecycleEvidence[run.DatabaseID].Reason, "403")
}

func TestHealthLifecycleFilteringKeepsMetricsAndInventoryConsistent(t *testing.T) {
	run := healthLifecycleRun()
	collection := healthLifecycleCollection()
	closed := run.StartedAt.Add(time.Minute)
	failed := run
	failed.DatabaseID, failed.Conclusion = 43, "failure"
	genuine := run
	genuine.DatabaseID, genuine.Conclusion, genuine.Event = 44, "failure", "schedule"
	success := genuine
	success.DatabaseID, success.Conclusion = 45, "success"
	var responses []workflowRunAPIResponse
	for _, item := range []WorkflowRun{run, failed, genuine, success} {
		response := workflowRunAPIResponse{ID: item.DatabaseID, RunAttempt: item.Attempt, Name: item.WorkflowName,
			Path: item.WorkflowPath, Status: item.Status, Conclusion: item.Conclusion, Event: item.Event,
			HeadBranch: item.HeadBranch, HeadSHA: item.HeadSha, CreatedAt: item.CreatedAt,
			RunStartedAt: item.StartedAt, UpdatedAt: item.UpdatedAt}
		response.Repository.FullName = item.Repository
		responses = append(responses, response)
	}
	data, err := json.Marshal(map[string]any{"workflow_runs": responses})
	require.NoError(t, err)
	requests := stubHealthLifecycleAPI(t, map[string]string{
		"/runs?per_page=10&page=1&created=":            string(data),
		"/pulls?per_page=100":                          lifecyclePRJSON(run, "closed", closed, closed),
		"/runs/42/attempts/1/jobs?per_page=100&page=1": lifecycleJobsJSON(run, false),
		"/runs/43/attempts/1/jobs?per_page=100&page=1": lifecycleJobsJSON(failed, true),
		"/runs/44/attempts/1/jobs?per_page=1":          `{"jobs":[{"id":1}]}`,
		"/99/logs":                                     "PR #123 closed before its head could be fetched. Stopping rather than running on the previous checkout.",
	}, nil)
	runs, total, err := listHealthWorkflowRuns(ListWorkflowRunsOptions{Limit: 10, RepoOverride: "org/repo"}, 1, collection)
	require.NoError(t, err)
	assert.Equal(t, 4, total, "raw pagination count includes ignored runs")
	require.Len(t, runs, 2)
	assert.Equal(t, int64(44), runs[0].DatabaseID)
	collection.revisions, collection.headRevisions = make(map[string]*workflow.LockMetadata), make(map[string]*workflow.LockMetadata)
	collection.sourceRevisions = make(map[string]string)
	for _, item := range runs {
		metadata := &workflow.LockMetadata{FrontmatterHash: "settings", BodyHash: "prompt"}
		collection.revisions[healthRevisionLookupKey(item)] = metadata
		collection.headRevisions[healthHeadRevisionKey(item)] = metadata
		collection.sourceRevisions[healthSourceRevisionKey(item, item.HeadSha)] = strings.Repeat("c", 40)
		collection.sourceRevisions[healthSourceRevisionKey(item, item.HeadBranch)] = strings.Repeat("c", 40)
	}
	runs, err = sampleHealthWorkflow(GitHubWorkflow{Path: run.WorkflowPath}, "", "org/repo", false, collection)
	require.NoError(t, err)
	health := CalculateWorkflowHealth("Reviewer", runs, 80)
	assert.Equal(t, 2, health.TotalRuns)
	assert.Equal(t, 1, health.SuccessCount)
	assert.Equal(t, 1, health.FailureCount)
	assert.InDelta(t, 50.0, health.SuccessRate, 0.001)
	assert.Equal(t, 5*time.Minute, health.AvgDuration)

	statuses := healthRevisionStatuses(runs, collection)
	require.Len(t, statuses, 1)
	assert.Equal(t, 1, statuses[0].Current.Failures)
	assert.Equal(t, 1, statuses[0].Current.Successes)
	assert.Equal(t, 2, statuses[0].Current.IndependentRuns)
	config := HealthConfig{collection: collection, RepoOverride: "org/repo", LogsDir: t.TempDir(), Threshold: 80}
	clusters, investigation := buildHealthFailureClusters(runs, config)
	require.Len(t, clusters, 1)
	require.NotNil(t, investigation)
	assert.Equal(t, 1, investigation.UniqueFailedRuns)
	before := *requests
	inventory := appendHealthFailureCandidates(responses,
		map[string]GitHubWorkflow{run.WorkflowPath: {Path: run.WorkflowPath}},
		make(map[int64]struct{}), make(map[string]int), make(map[string]int), nil, collection)
	require.Len(t, inventory, 1)
	assert.Equal(t, genuine.DatabaseID, inventory[0].DatabaseID)
	assert.Equal(t, 1, collection.FailureCandidates)
	assert.Equal(t, before, *requests, "inventory and chronological passes reuse lifecycle and job observations")
	emptyHealth := calculateCurrentWorkflowHealth("Reviewer", nil, config)
	assert.Equal(t, "no_recent_runs", emptyHealth.Status)
	assert.Zero(t, emptyHealth.SuccessCount)
	assert.Zero(t, CalculateHealthSummary([]WorkflowHealth{emptyHealth}, "test", 80).HealthyWorkflows)
}

func TestHealthLifecycleSamplingContinuesPastIgnoredRuns(t *testing.T) {
	run := healthLifecycleRun()
	collection := healthLifecycleCollection()
	var ignored, successful []workflowRunAPIResponse
	for i := range 15 {
		response := workflowRunAPIResponse{ID: int64(i + 1), RunAttempt: 1, Name: run.WorkflowName, Path: run.WorkflowPath,
			Status: "completed", Conclusion: "cancelled", Event: run.Event, HeadBranch: run.HeadBranch, HeadSHA: run.HeadSha,
			CreatedAt: run.CreatedAt, RunStartedAt: run.StartedAt, UpdatedAt: run.UpdatedAt}
		if i < 10 {
			ignored = append(ignored, response)
		} else {
			response.Conclusion = "success"
			successful = append(successful, response)
		}
	}
	first, err := json.Marshal(map[string]any{"workflow_runs": ignored})
	require.NoError(t, err)
	second, err := json.Marshal(map[string]any{"workflow_runs": successful})
	require.NoError(t, err)
	requests := stubHealthLifecycleAPI(t, map[string]string{
		"/runs?per_page=10&page=1&created=": string(first),
		"/runs?per_page=10&page=2&created=": string(second),
		"/pulls?per_page=100":               lifecyclePRJSON(run, "closed", run.StartedAt.Add(time.Minute), time.Time{}),
		"/jobs?per_page=100&page=1":         lifecycleJobsJSON(run, false),
	}, nil)
	runs, err := sampleHealthWorkflow(GitHubWorkflow{Path: run.WorkflowPath}, "", "org/repo", false, collection)
	require.NoError(t, err)
	assert.Len(t, runs, healthMinimumRuns)
	assert.Len(t, collection.PRLifecycleEvidence, 10)
	assert.Equal(t, 13, *requests, "two listing pages, one lifecycle lookup, ten attempt-specific job inventories")
	assert.Equal(t, healthMinimumRuns, CalculateWorkflowHealth("Reviewer", runs, 80).SuccessCount)
	assert.Equal(t, "revision_unknown", calculateCurrentWorkflowHealth("Reviewer", runs,
		HealthConfig{collection: collection, LogsDir: t.TempDir()}).Status, "exclusions do not invent revision evidence")
}
