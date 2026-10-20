//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubHealthFailureCandidates(t *testing.T) {
	t.Helper()
	original := healthListFailureCandidates
	t.Cleanup(func() { healthListFailureCandidates = original })
	healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		return nil, time.Time{}, nil
	}
}

func TestHealthCollectsFailuresBeforeSuccessEvidence(t *testing.T) {
	for _, exhaust := range []bool{false, true} {
		t.Run(strconv.FormatBool(exhaust), func(t *testing.T) {
			originalList, originalGH := healthListWorkflowRuns, healthRunGH
			t.Cleanup(func() { healthListWorkflowRuns, healthRunGH = originalList, originalGH })
			collection := &HealthCollection{
				Host: "github.com", repoOverride: "org/repo",
				APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
				revisions:    make(map[string]*workflow.LockMetadata),
				workflows: []GitHubWorkflow{
					{ID: 1, Name: "Alpha", Path: ".github/workflows/alpha.lock.yml"},
					{ID: 2, Name: "Zulu", Path: ".github/workflows/zulu.lock.yml"},
				},
			}
			start := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
			var failed []workflowRunAPIResponse
			for i := range 2 {
				response := workflowRunAPIResponse{ID: int64(i + 1), Name: "Zulu", Path: collection.workflows[1].Path,
					Status: "completed", Conclusion: "failure", Event: "schedule", HeadBranch: "main",
					HeadSHA: strings.Repeat("a", 40), RunAttempt: 1, CreatedAt: start.Add(-time.Duration(i) * time.Hour)}
				failed = append(failed, response)
				collection.revisions[healthRevisionLookupKey(response.workflowRun())] = &workflow.LockMetadata{
					FrontmatterHash: "settings", BodyHash: "prompt",
				}
			}
			failed = append(failed, workflowRunAPIResponse{ID: 99, Path: ".github/workflows/ci.yml", Conclusion: "failure"})
			var jobs []string
			healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				endpoint := args[3]
				if strings.Contains(endpoint, "/actions/runs?status=failure") {
					assert.Contains(t, endpoint, "created=")
					out, err := json.Marshal(map[string]any{"workflow_runs": failed})
					require.NoError(t, err)
					return out, nil
				}
				jobs = append(jobs, endpoint)
				assert.NotContains(t, endpoint, "/runs/99/", "ordinary CI failures must not consume evidence requests")
				if exhaust && len(jobs) == 1 {
					collection.APIRequests = healthMaximumAPIRequests
				}
				return []byte(`{"jobs":[{"id":42}]}`), nil
			}
			var order []string
			healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, page int, _ *HealthCollection) ([]WorkflowRun, int, error) {
				require.False(t, exhaust, "no successful evidence requests after budget exhaustion")
				require.True(t, collection.FailurePassComplete)
				require.Len(t, jobs, 1, "verify one representative per scope before sampling success evidence")
				assert.True(t, collection.FailureListingComplete)
				assert.Equal(t, 2, collection.FailureCandidates)
				assert.Equal(t, 1, collection.FailureCandidateWorkflows)
				require.Equal(t, 1, page)
				order = append(order, opts.WorkflowName)
				if opts.WorkflowName == "2" {
					return []WorkflowRun{failed[0].workflowRun(), failed[1].workflowRun()}, 2, nil
				}
				return nil, 0, nil
			}
			runs, err := fetchWorkflowRuns("", start.Add(-24*time.Hour).Format(time.RFC3339), "org/repo", false, collection)
			require.NoError(t, err)
			config := HealthConfig{Threshold: 80, RepoOverride: "org/repo", LogsDir: t.TempDir(), collection: collection}
			_, plan := buildHealthFailureClusters(runs, config)
			require.NotNil(t, plan, "failure evidence survives interruption before successful sampling")
			if exhaust {
				assert.True(t, collection.Partial)
				assert.False(t, collection.FailurePassComplete)
				assert.Len(t, runs, 1)
				assert.Equal(t, 1, plan.UniqueFailedRuns)
			} else {
				assert.Equal(t, []string{"2", "1"}, order, "failing workflows precede alphabetical success-only workflows")
				assert.Len(t, runs, 2, "failure and normal passes must not duplicate runs")
				assert.Equal(t, 2, plan.UniqueFailedRuns)
			}
		})
	}
}

func TestHealthFailureCandidatesPreserveBoundsAndValidateJSON(t *testing.T) {
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	for _, response := range []string{`{"workflow_runs":[]}`, `{}`, `not-json`} {
		collection := &HealthCollection{Host: "github.example.com", APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
		opts := ListWorkflowRunsOptions{Limit: 100, StartDate: "2026-10-19T12:00:00Z", EndDate: "2026-10-20T12:00:00Z"}
		healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
			assert.Equal(t, "github.example.com", args[2])
			endpoint, err := url.Parse(args[3])
			require.NoError(t, err)
			assert.Equal(t, "repos/org/repo/actions/runs", endpoint.Path)
			assert.Equal(t, "failure", endpoint.Query().Get("status"))
			assert.Equal(t, "3", endpoint.Query().Get("page"))
			assert.Equal(t, opts.StartDate+".."+opts.EndDate, endpoint.Query().Get("created"))
			return []byte(response), nil
		}
		_, _, err := listHealthFailureCandidates(opts, 3, "github.example.com/org/repo", collection)
		if response == `{"workflow_runs":[]}` {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "failed to parse health failure runs")
		}
	}

}

func TestHealthNamedWorkflowFailureListingIsScoped(t *testing.T) {
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	collection := &HealthCollection{Host: "github.com", APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
		workflows: []GitHubWorkflow{{ID: 42, Path: ".github/workflows/example.lock.yml"}}}
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Contains(t, args[3], "repos/org/repo/actions/workflows/42/runs?status=failure")
		assert.NotContains(t, args[3], "/actions/runs?", "named health must not scan unrelated workflows")
		return []byte(`{"workflow_runs":[]}`), nil
	}
	_, _, err := listHealthFailureCandidates(ListWorkflowRunsOptions{Limit: 100}, 1, "org/repo", collection)
	require.NoError(t, err)
	assert.Equal(t, 1, collection.APIRequests)
}

func TestHealthFailurePassSkipsSupersededScopes(t *testing.T) {
	stubHealthFailureCandidates(t)
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	var batch []workflowRunAPIResponse
	collection := &HealthCollection{
		Host: "github.com", APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
		workflows: []GitHubWorkflow{{Path: ".github/workflows/example.lock.yml"}},
		revisions: make(map[string]*workflow.LockMetadata), headRevisions: make(map[string]*workflow.LockMetadata),
		sourceRevisions: make(map[string]string),
	}
	for i := range 10 {
		response := workflowRunAPIResponse{ID: int64(i + 1), Path: collection.workflows[0].Path,
			HeadSHA: strings.Repeat("a", 40), HeadBranch: "main", Event: "schedule", Conclusion: "failure"}
		if i == 1 {
			response.HeadBranch, response.HeadSHA = "feature", strings.Repeat("b", 40)
		}
		run := response.workflowRun()
		metadata := &workflow.LockMetadata{FrontmatterHash: "settings", BodyHash: "prompt"}
		collection.revisions[healthRevisionLookupKey(run)] = metadata
		collection.headRevisions[healthHeadRevisionKey(run)] = metadata
		collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadBranch)] = strings.Repeat("c", 40)
		source := strings.Repeat("d", 40)
		if i == 1 {
			source = strings.Repeat("c", 40)
		}
		collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadSha)] = source
		batch = append(batch, response)
	}
	healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		return batch, time.Time{}, nil
	}
	jobs := 0
	healthRunGH = func(context.Context, string, ...string) ([]byte, error) {
		jobs++
		return []byte(`{"jobs":[{"id":42}]}`), nil
	}
	runs, err := collectHealthFailures("", "org/repo", collection)
	require.NoError(t, err)
	assert.Len(t, runs, 2)
	assert.Equal(t, 2, jobs, "stop superseded main evidence while retaining current feature evidence")
	assert.True(t, collection.FailurePassComplete)
}

func TestHealthFailureListingErrorIsNotSuccess(t *testing.T) {
	stubHealthFailureCandidates(t)
	healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		return nil, time.Time{}, errors.New("failure listing unavailable")
	}
	collection := &HealthCollection{workflows: []GitHubWorkflow{{Path: ".github/workflows/example.lock.yml"}}}
	_, err := fetchWorkflowRuns("", "", "", false, collection)
	require.ErrorContains(t, err, "failure listing unavailable")
	assert.False(t, collection.FailurePassComplete)
}

func TestHealthFailurePassIsBoundedAndKeepsHealthyChronologicalSample(t *testing.T) {
	stubHealthFailureCandidates(t)
	originalGH, originalList := healthRunGH, healthListWorkflowRuns
	t.Cleanup(func() { healthRunGH, healthListWorkflowRuns = originalGH, originalList })
	runs, collection := healthRevisionFixture()
	collection.Host = "github.com"
	collection.APIRateLimit = &GitHubAPIRateLimitState{Remaining: 4000}
	collection.workflows = []GitHubWorkflow{{ID: 1, Path: runs[0].WorkflowPath, Name: "Example"}}
	metadata := &workflow.LockMetadata{FrontmatterHash: "settings", BodyHash: "fixed-prompt"}
	var failed []workflowRunAPIResponse
	for i := range 10 {
		response := workflowRunAPIResponse{ID: int64(100 + i), Path: runs[0].WorkflowPath, Name: "Example",
			Status: "completed", Conclusion: "failure", Event: "schedule", HeadBranch: "main", HeadSHA: "older",
			RunAttempt: 1, CreatedAt: runs[0].CreatedAt.Add(-time.Duration(i+1) * time.Hour)}
		response.Repository.FullName = runs[0].Repository
		failed = append(failed, response)
	}
	collection.revisions[healthRevisionLookupKey(failed[0].workflowRun())] = metadata
	for i := range runs {
		collection.revisions[healthRevisionLookupKey(runs[i])] = metadata
	}
	jobs := 0
	healthRunGH = func(context.Context, string, ...string) ([]byte, error) {
		jobs++
		return []byte(`{"jobs":[{"id":42}]}`), nil
	}
	healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		return failed, time.Time{}, nil
	}
	healthListWorkflowRuns = func(ListWorkflowRunsOptions, int, *HealthCollection) ([]WorkflowRun, int, error) {
		return runs, len(runs), nil
	}
	collected, err := fetchWorkflowRuns("", "", "org/repo", false, collection)
	require.NoError(t, err)
	assert.Equal(t, 1, jobs, "one representative per scope is enough for initial detection")
	config := HealthConfig{Threshold: 80, RepoOverride: "org/repo", LogsDir: t.TempDir(), collection: collection}
	health := calculateCurrentWorkflowHealth("Example", collected, config)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, healthMinimumRuns, health.SuccessCount)
	assert.Zero(t, health.FailureCount, "failure-first ordering must not replace the newest chronological sample")
	clusters, plan := buildHealthFailureClusters(collected, config)
	assert.Empty(t, clusters)
	assert.Nil(t, plan)
}

func TestHealthFailurePassPaginatesBeforeSuccessSampling(t *testing.T) {
	stubHealthFailureCandidates(t)
	collection := &HealthCollection{workflows: []GitHubWorkflow{{Path: ".github/workflows/example.lock.yml"}}}
	start := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	calls := 0
	healthListFailureCandidates = func(opts ListWorkflowRunsOptions, page int, _ string, _ *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		calls++
		assert.Equal(t, 100, opts.Limit)
		if calls <= 10 {
			assert.Equal(t, calls, page)
			assert.Empty(t, opts.BeforeDate)
			return make([]workflowRunAPIResponse, 100), start, nil
		}
		assert.Equal(t, 1, page)
		assert.Equal(t, start.Add(time.Second).Format(time.RFC3339), opts.BeforeDate)
		return nil, time.Time{}, nil
	}
	runs, err := collectHealthFailures("", "org/repo", collection)
	require.NoError(t, err)
	assert.Empty(t, runs)
	assert.Equal(t, 11, calls)
	assert.True(t, collection.FailurePassComplete)
	assert.False(t, collection.Partial)
}
