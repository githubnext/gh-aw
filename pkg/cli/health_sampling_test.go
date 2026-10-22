//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthSamplingCollectsEvidenceForEachWorkflow(t *testing.T) {
	stubHealthFailureCandidates(t)
	original := healthListWorkflowRuns
	t.Cleanup(func() { healthListWorkflowRuns = original })
	runs, collection := healthRevisionFixture()
	collection.workflows = []GitHubWorkflow{{ID: 1, Name: "Example", Path: runs[0].WorkflowPath},
		{ID: 2, Name: "Sparse", Path: ".github/workflows/sparse.lock.yml"}}
	var calls []string
	healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, page int, state *HealthCollection) ([]WorkflowRun, int, error) {
		calls = append(calls, opts.WorkflowName)
		if opts.WorkflowName == "1" {
			assert.Equal(t, 1, page, "enough current revision runs should stop collection immediately")
			return runs, opts.Limit, nil
		}
		return nil, 0, nil
	}
	collected, err := fetchWorkflowRuns("", "2026-10-08", "org/repo", false, collection)
	require.NoError(t, err)
	assert.Len(t, collected, 10)
	assert.Equal(t, []string{"1", "2"}, calls, "busy workflows must not hide workflows with no recent runs")
	assert.False(t, collection.Partial)
}

func TestHealthSamplingCoversFailureScopesBeforeStopping(t *testing.T) {
	originalFailureList, originalGH, originalRunList := healthListFailureCandidates, healthRunGH, healthListWorkflowRuns
	t.Cleanup(func() {
		healthListFailureCandidates, healthRunGH, healthListWorkflowRuns = originalFailureList, originalGH, originalRunList
	})
	path := ".github/workflows/example.lock.yml"
	start := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	metadata := &workflow.LockMetadata{FrontmatterHash: "settings", BodyHash: "prompt"}
	collection := &HealthCollection{
		Host: "github.com", repoOverride: "org/repo", APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
		workflows:       []GitHubWorkflow{{ID: 1, Name: "Example", Path: path}},
		revisions:       make(map[string]*workflow.LockMetadata),
		headRevisions:   make(map[string]*workflow.LockMetadata),
		sourceRevisions: make(map[string]string),
	}
	failure := WorkflowRun{DatabaseID: 1, WorkflowName: "Example", WorkflowPath: path, Repository: "org/repo",
		HeadBranch: "main", HeadSha: strings.Repeat("a", 40), Event: "workflow_dispatch", Attempt: 1,
		Status: "completed", Conclusion: "failure", CreatedAt: start.Add(-30 * time.Hour), UpdatedAt: start.Add(-29 * time.Hour)}
	collection.revisions[healthRevisionLookupKey(failure)] = metadata
	collection.headRevisions[healthHeadRevisionKey(failure)] = metadata
	collection.sourceRevisions[healthSourceRevisionKey(failure, failure.HeadSha)] = strings.Repeat("c", 40)
	collection.sourceRevisions[healthSourceRevisionKey(failure, failure.HeadBranch)] = strings.Repeat("c", 40)
	healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		response := workflowRunAPIResponse{ID: failure.DatabaseID, Name: failure.WorkflowName, Path: path,
			HeadBranch: failure.HeadBranch, HeadSHA: failure.HeadSha,
			Event: failure.Event, Status: failure.Status, Conclusion: failure.Conclusion,
			RunAttempt: failure.Attempt, CreatedAt: failure.CreatedAt, UpdatedAt: failure.UpdatedAt}
		response.Repository.FullName = failure.Repository
		return []workflowRunAPIResponse{response}, time.Time{}, nil
	}
	healthRunGH = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"jobs":[{"id":42}]}`), nil
	}
	var sampledScopes []string
	healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, page int, _ *HealthCollection) ([]WorkflowRun, int, error) {
		require.Equal(t, "1", opts.WorkflowName)
		sampledScopes = append(sampledScopes, strconv.Itoa(page))
		if page == 1 {
			var runs []WorkflowRun
			for i := range opts.Limit {
				run := failure
				run.DatabaseID = int64(100 + i)
				run.Event = "schedule"
				run.Conclusion = "success"
				run.CreatedAt, run.UpdatedAt = start.Add(-time.Duration(i)*time.Minute), start.Add(-time.Duration(i)*time.Minute+time.Second)
				collection.revisions[healthRevisionLookupKey(run)] = metadata
				collection.headRevisions[healthHeadRevisionKey(run)] = metadata
				collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadSha)] = strings.Repeat("c", 40)
				collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadBranch)] = strings.Repeat("c", 40)
				runs = append(runs, run)
			}
			return runs, opts.Limit, nil
		}
		require.Equal(t, 2, page)
		var runs []WorkflowRun
		for i := range healthMinimumRuns {
			run := failure
			run.DatabaseID = int64(200 + i)
			run.Conclusion = "success"
			run.CreatedAt, run.UpdatedAt = start.Add(-time.Duration(20+i)*time.Hour), start.Add(-time.Duration(20+i)*time.Hour+time.Second)
			collection.revisions[healthRevisionLookupKey(run)] = metadata
			runs = append(runs, run)
		}
		return runs, len(runs), nil
	}
	collected, err := fetchWorkflowRuns("", start.Add(-48*time.Hour).Format(time.RFC3339), "org/repo", false, collection)
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, sampledScopes, "scheduled evidence cannot satisfy the failed dispatch scope")
	require.Len(t, collected, 3*healthMinimumRuns+1)
	health := calculateCurrentWorkflowHealth("Example", collected, HealthConfig{Threshold: 80, RepoOverride: "org/repo", collection: collection})
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, 2*healthMinimumRuns, health.SuccessCount)
	assert.Zero(t, health.FailureCount, "chronological dispatch successes supersede the older failure")
}

func TestHealthSamplingContinuesPastRunsAwaitingApproval(t *testing.T) {
	originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
	t.Cleanup(func() { fetchRateLimitForHostFunc, healthRunGH = originalRate, originalGH })
	fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
		return rateLimitResource{Limit: 5000, Remaining: 4000}, nil
	}
	pages, jobChecks := 0, 0
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		endpoint := args[3]
		if strings.HasSuffix(endpoint, "/jobs?per_page=1") {
			jobChecks++
			assert.Contains(t, endpoint, "/attempts/1/jobs")
			return []byte(`{"jobs":[]}`), nil
		}
		if strings.Contains(endpoint, "/contents/") {
			assert.NotContains(t, endpoint, "unapproved", "ignored runs need no revision lookups")
			if strings.Contains(endpoint, ".md?") {
				if strings.Contains(endpoint, "ref="+strings.Repeat("b", 40)) {
					return []byte(strings.Repeat("d", 40)), nil
				}
				return []byte(strings.Repeat("c", 40)), nil
			}
			return []byte(`# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"settings","body_hash":"prompt"}`), nil
		}
		pages++
		assert.Contains(t, endpoint, "page="+strconv.Itoa(pages))
		var responses []workflowRunAPIResponse
		count := 10
		if pages == 2 {
			count = 5
		}
		for i := range count {
			run := workflowRunAPIResponse{ID: int64(pages*10 + i), Name: "Example",
				Path: ".github/workflows/example.lock.yml", Status: "completed",
				Conclusion: "success", Event: "schedule", HeadBranch: "main",
				HeadSHA: strings.Repeat("a", 40), RunAttempt: 1}
			if pages == 1 {
				run.HeadSHA = "unapproved"
				if i%2 == 0 {
					run.Status, run.Conclusion = "waiting", ""
				} else {
					run.Conclusion = "action_required"
				}
			}
			responses = append(responses, run)
		}
		out, err := json.Marshal(map[string]any{"workflow_runs": responses})
		require.NoError(t, err)
		return out, nil
	}
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo"}
	runs, err := sampleHealthWorkflow(GitHubWorkflow{ID: 1, Name: "Example",
		Path: ".github/workflows/example.lock.yml"}, "2026-10-08", "org/repo", false, collection)
	require.NoError(t, err)
	assert.Equal(t, 2, pages)
	assert.Equal(t, 5, jobChecks, "only ambiguous waiting runs need jobs lookups")
	assert.Len(t, runs, 5)
	health := calculateCurrentWorkflowHealth("Example", runs, HealthConfig{Threshold: 80, collection: collection})
	assert.Equal(t, "healthy", health.Status)
	assert.InDelta(t, 100.0, health.SuccessRate, 1e-9)
}

func TestHealthWorkflowSelectionUsesStablePaths(t *testing.T) {
	original := healthDiscoverWorkflows
	t.Cleanup(func() { healthDiscoverWorkflows = original })
	healthDiscoverWorkflows = func(*HealthCollection) ([]GitHubWorkflow, error) {
		return []GitHubWorkflow{
			{Name: "Example", Path: ".github/workflows/a.lock.yml", State: "active"},
			{Name: "Example", Path: ".github/workflows/b.lock.yml", State: "active"},
		}, nil
	}
	config := HealthConfig{WorkflowName: "a", Days: 7, collection: &HealthCollection{}}
	require.NoError(t, discoverHealthWorkflows(&config))
	require.Len(t, config.collection.workflows, 1)
	assert.Equal(t, ".github/workflows/a.lock.yml", config.collection.workflows[0].Path)
	config = HealthConfig{WorkflowName: "Example", Days: 7, collection: &HealthCollection{}}
	require.ErrorContains(t, discoverHealthWorkflows(&config), "ambiguous")
}

func TestHealthAPIQuotaAndRevisionLookupReuse(t *testing.T) {
	originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
	t.Cleanup(func() { fetchRateLimitForHostFunc, healthRunGH = originalRate, originalGH })
	calls, probes := 0, 0
	fetchRateLimitForHostFunc = func(ctx context.Context, host string) (rateLimitResource, error) {
		probes++
		assert.Equal(t, "github.example.com", host)
		return rateLimitResource{Limit: 5000, Remaining: 100}, nil
	}
	healthRunGH = func(ctx context.Context, message string, args ...string) ([]byte, error) {
		calls++
		assert.Contains(t, args, "github.example.com")
		if strings.Contains(args[3], ".md?") {
			return []byte(strings.Repeat("c", 40)), nil
		}
		return []byte(`# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"config","body_hash":"prompt"}`), nil
	}
	collection := &HealthCollection{Host: "github.example.com", repoOverride: "github.example.com/org/repo"}
	run := WorkflowRun{DatabaseID: 1, WorkflowPath: ".github/workflows/example.lock.yml",
		Event: "schedule", HeadSha: strings.Repeat("a", 40)}
	resolveHealthRevision(run, collection)
	run.DatabaseID = 2
	resolveHealthRevision(run, collection)
	assert.Equal(t, 2, calls, "identical immutable source and workflow revisions should be read once")
	assert.Equal(t, 1, probes)
	require.NotNil(t, collection.revisions[healthRevisionLookupKey(run)])
	run.Event, run.HeadSha = "workflow_run", strings.Repeat("b", 40)
	resolveHealthRevision(run, collection)
	assert.Equal(t, 2, calls, "ambiguous head SHA must not be used as executing workflow revision")
	collection.APIRequests = healthMaximumAPIRequests
	_, err := healthAPI(collection, "repos/org/repo/actions/runs")
	require.ErrorIs(t, err, errHealthAPIBudget)
	assert.True(t, collection.Partial)
	assert.Equal(t, "request_budget", collection.Reason)
	assert.Equal(t, 2, calls)
}

func TestHealthLowQuotaAndSecondaryLimitProducePartialJSON(t *testing.T) {
	for _, secondary := range []bool{false, true} {
		t.Run(map[bool]string{false: "low-core-quota", true: "secondary-limit"}[secondary], func(t *testing.T) {
			originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
			originalDiscover := healthDiscoverWorkflows
			t.Cleanup(func() {
				fetchRateLimitForHostFunc, healthRunGH, healthDiscoverWorkflows = originalRate, originalGH, originalDiscover
			})
			fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
				remaining := RateLimitThreshold
				if secondary {
					remaining = 4000
				}
				return rateLimitResource{Remaining: remaining, Limit: 5000, Reset: time.Now().Add(time.Hour).Unix()}, nil
			}
			healthRunGH = func(context.Context, string, ...string) ([]byte, error) {
				require.True(t, secondary, "no listing requests should be made with exhausted core quota")
				return []byte("You have exceeded a secondary rate limit"), errors.New("HTTP 403")
			}
			healthDiscoverWorkflows = func(*HealthCollection) ([]GitHubWorkflow, error) {
				return []GitHubWorkflow{{ID: 1, Name: "Example", Path: ".github/workflows/example.lock.yml", State: "active"}}, nil
			}
			output := captureAuditGroupStdout(t, func() {
				require.NoError(t, RunHealth(HealthConfig{Days: 7, Threshold: 80, JSONOutput: true, RepoOverride: "org/repo"}))
			})
			var report HealthSummary
			require.NoError(t, json.Unmarshal([]byte(output), &report))
			assert.True(t, report.Collection.Partial)
			assert.Equal(t, "rate_limit", report.Collection.Reason)
			assert.Equal(t, 1, report.InsufficientWorkflows)
			assert.Zero(t, report.HealthyWorkflows)
		})
	}
}

func TestHealthCurrentSnapshotFromGitHubMetadata(t *testing.T) {
	for _, unexecuted := range []bool{false, true} {
		t.Run(map[bool]string{false: "recovered", true: "pushed-without-runs"}[unexecuted], func(t *testing.T) {
			originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
			t.Cleanup(func() { fetchRateLimitForHostFunc, healthRunGH = originalRate, originalGH })
			fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
				return rateLimitResource{Limit: 5000, Remaining: 4000}, nil
			}
			runs, _ := healthRevisionFixture()
			var responses []workflowRunAPIResponse
			for _, run := range runs {
				sha := strings.Repeat("a", 40)
				if run.Conclusion == "failure" {
					sha = strings.Repeat("b", 40)
				}
				responses = append(responses, workflowRunAPIResponse{ID: run.DatabaseID,
					Status: run.Status, Conclusion: run.Conclusion, Name: run.WorkflowName,
					Path: run.WorkflowPath, CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt,
					Event: run.Event, HeadBranch: run.HeadBranch, HeadSHA: sha, RunAttempt: 1})
			}
			newestFirst := make([]workflowRunAPIResponse, 0, len(responses))
			for i := range slices.Backward(responses) {
				newestFirst = append(newestFirst, responses[i])
			}
			requests, queueRequests := 0, 0
			healthRunGH = func(ctx context.Context, message string, args ...string) ([]byte, error) {
				requests++
				require.GreaterOrEqual(t, len(args), 4)
				assert.Equal(t, "api", args[0])
				endpoint := args[3]
				switch {
				case endpoint == "repos/org/repo":
					queueRequests++
					return []byte(`{"full_name":"org/repo","default_branch":"main"}`), nil
				case strings.Contains(endpoint, "/contents/.github/workflows?"):
					queueRequests++
					return []byte(`[{"path":".github/workflows/example.lock.yml","sha":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","type":"file"}]`), nil
				case strings.Contains(endpoint, "/git/blobs/"):
					queueRequests++
					return []byte("jobs: {}"), nil
				case strings.Contains(endpoint, "/actions/workflows?"):
					return []byte(`{"workflows":[{"id":11,"name":"Example","path":".github/workflows/example.lock.yml","state":"active"}]}`), nil
				case strings.Contains(endpoint, "/actions/runs?status=failure"):
					return []byte(`{"workflow_runs":[]}`), nil
				case strings.Contains(endpoint, "/actions/workflows/11/runs?"):
					if strings.Contains(endpoint, "status=failure") {
						var failures []workflowRunAPIResponse
						for _, response := range responses {
							if response.Conclusion == "failure" {
								failures = append(failures, response)
							}
						}
						out, err := json.Marshal(map[string]any{"workflow_runs": failures})
						require.NoError(t, err)
						return out, nil
					}
					assert.Contains(t, endpoint, "per_page=10&page=1&created=")
					out, err := json.Marshal(map[string]any{"workflow_runs": newestFirst})
					require.NoError(t, err)
					return out, nil
				case strings.HasSuffix(endpoint, "/jobs?per_page=1"):
					return []byte(`{"jobs":[{"id":42}]}`), nil
				case strings.Contains(endpoint, "/contents/"):
					if strings.Contains(endpoint, ".md?") {
						if strings.Contains(endpoint, "ref="+strings.Repeat("b", 40)) {
							return []byte(strings.Repeat("d", 40)), nil
						}
						return []byte(strings.Repeat("c", 40)), nil
					}
					body := "fixed-prompt"
					if strings.Contains(endpoint, "ref="+strings.Repeat("b", 40)) {
						body = "old-prompt"
					}
					if unexecuted && strings.Contains(endpoint, "ref=main") {
						body = "just-pushed"
					}
					return []byte(`# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"settings","body_hash":"` + body + `"}`), nil
				default:
					t.Fatalf("unexpected health request: %s", endpoint)
					return nil, errors.New("unexpected health request")
				}
			}
			output := captureAuditGroupStdout(t, func() {
				require.NoError(t, RunHealth(HealthConfig{Days: 7, Threshold: 80, JSONOutput: true, RepoOverride: "org/repo"}))
			})
			var report HealthSummary
			require.NoError(t, json.Unmarshal([]byte(output), &report))
			require.Len(t, report.Workflows, 1)
			health := report.Workflows[0]
			if unexecuted {
				assert.Equal(t, "no_current_revision_runs", health.Status)
				assert.Zero(t, report.HealthyWorkflows)
			} else {
				assert.Equal(t, "recovery_observed", health.Status)
				assert.Equal(t, 5, health.TotalRuns)
				assert.Equal(t, 1, health.Revisions[0].Previous.Failures)
				assert.Equal(t, strings.Repeat("d", 40), health.Revisions[0].Previous.SourceSHA)
				assert.NotEqual(t, health.Revisions[0].Previous.SourceSHA, health.Revisions[0].Current.SourceSHA)
			}
			assert.Empty(t, report.FailureClusters)
			assert.False(t, report.Collection.Partial)
			assert.LessOrEqual(t, requests-queueRequests, 14, "superseded sources do not need lock metadata lookups")
		})
	}
}
