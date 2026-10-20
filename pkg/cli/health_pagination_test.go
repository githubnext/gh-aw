//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthSamplingContinuesPastFilteredPageLimit(t *testing.T) {
	original := healthListWorkflowRuns
	t.Cleanup(func() { healthListWorkflowRuns = original })
	runs, collection := healthRevisionFixture()
	calls := 0
	healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, page int, _ *HealthCollection) ([]WorkflowRun, int, error) {
		calls++
		assert.Equal(t, calls, page)
		assert.Equal(t, "2026-10-08", opts.StartDate)
		if page <= MaxIterations {
			return nil, opts.Limit, nil
		}
		return runs[5:], 5, nil
	}
	sample, err := sampleHealthWorkflow(GitHubWorkflow{ID: 1, Name: "Example",
		Path: runs[0].WorkflowPath}, "2026-10-08", "org/repo", false, collection)
	require.NoError(t, err)
	assert.Equal(t, MaxIterations+1, calls)
	assert.Len(t, sample, healthMinimumRuns)
	assert.False(t, collection.Partial)
	health := calculateCurrentWorkflowHealth("Example", sample, HealthConfig{Threshold: 80, collection: collection})
	assert.Equal(t, "healthy", health.Status)
}

func TestHealthSamplingRollsSearchWindowWithoutSkippingTimestampTies(t *testing.T) {
	originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
	t.Cleanup(func() { fetchRateLimitForHostFunc, healthRunGH = originalRate, originalGH })
	fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
		return rateLimitResource{Limit: 5000, Remaining: 4000}, nil
	}
	end := time.Date(2026, 10, 16, 4, 0, 0, 0, time.UTC)
	start := end.Add(-24 * time.Hour).Format(time.RFC3339)
	boundary := end.Add(-990 * time.Second)
	var responses []workflowRunAPIResponse
	for i := range healthMaximumRunSearchResults + healthMinimumRuns - 1 {
		created := end.Add(-time.Duration(i) * time.Second)
		if i >= 990 {
			created = boundary
		}
		run := workflowRunAPIResponse{ID: int64(i + 1), Name: "Example",
			Path: ".github/workflows/example.lock.yml", Status: "completed",
			Conclusion: "action_required", Event: "schedule", HeadBranch: "main",
			HeadSHA: "unapproved", RunAttempt: 1, CreatedAt: created}
		if i >= healthMaximumRunSearchResults-1 {
			run.Conclusion, run.HeadSHA = "success", strings.Repeat("a", 40)
		}
		responses = append(responses, run)
	}
	rolledPages := []int{}
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		endpoint := args[3]
		if strings.Contains(endpoint, "/contents/") {
			assert.NotContains(t, endpoint, "unapproved")
			if strings.Contains(endpoint, ".md?") {
				return []byte(strings.Repeat("c", 40)), nil
			}
			return []byte(`# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"settings","body_hash":"prompt"}`), nil
		}
		parsed, err := url.Parse(endpoint)
		require.NoError(t, err)
		created := strings.Split(parsed.Query().Get("created"), "..")
		require.Len(t, created, 2)
		assert.Equal(t, start, created[0])
		upper, err := time.Parse(time.RFC3339, created[1])
		require.NoError(t, err)
		assert.False(t, upper.After(end), "the requested end bound must not expand")
		page, err := strconv.Atoi(parsed.Query().Get("page"))
		require.NoError(t, err)
		limit, err := strconv.Atoi(parsed.Query().Get("per_page"))
		require.NoError(t, err)
		if upper.Equal(boundary) {
			rolledPages = append(rolledPages, page)
		}
		var matching []workflowRunAPIResponse
		for _, run := range responses {
			if !run.CreatedAt.After(upper) {
				matching = append(matching, run)
			}
		}
		first := min((page-1)*limit, len(matching), healthMaximumRunSearchResults)
		last := min(first+limit, len(matching), healthMaximumRunSearchResults)
		out, err := json.Marshal(map[string]any{"workflow_runs": matching[first:last]})
		require.NoError(t, err)
		return out, nil
	}
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo", EndDate: end.Format(time.RFC3339)}
	runs, err := sampleHealthWorkflow(GitHubWorkflow{ID: 1, Name: "Example",
		Path: ".github/workflows/example.lock.yml"}, start, "org/repo", false, collection)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, rolledPages)
	require.Len(t, runs, healthMinimumRuns)
	assert.Equal(t, int64(healthMaximumRunSearchResults), runs[0].DatabaseID)
	assert.False(t, collection.Partial)
	assert.Equal(t, end.Format(time.RFC3339), collection.EndDate)
	health := calculateCurrentWorkflowHealth("Example", runs, HealthConfig{Threshold: 80, collection: collection})
	assert.Equal(t, "healthy", health.Status)
}

func TestHealthUnadvanceableSearchWindowRemainsExplicitlyPartial(t *testing.T) {
	original := healthListWorkflowRuns
	t.Cleanup(func() { healthListWorkflowRuns = original })
	tests := []struct {
		name   string
		oldest time.Time
		pages  int
	}{
		{"saturated timestamp", time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC), 200},
		{"unavailable timestamp", time.Time{}, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, _ int, _ *HealthCollection) ([]WorkflowRun, int, error) {
				calls++
				*opts.OldestFetchedCreatedAt = tt.oldest
				return nil, opts.Limit, nil
			}
			collection := &HealthCollection{}
			runs, err := sampleHealthWorkflow(GitHubWorkflow{Name: "Example",
				Path: ".github/workflows/example.lock.yml"}, "2026-10-08", "org/repo", false, collection)
			require.NoError(t, err)
			assert.Empty(t, runs)
			assert.Equal(t, tt.pages, calls)
			assert.True(t, collection.Partial)
			assert.Equal(t, "pagination_limit", collection.Reason)
			assert.Contains(t, collection.Warning, "timestamp boundary")
		})
	}
}

func TestHealthSamplingReusesJobProbesAcrossOverlappingWindows(t *testing.T) {
	originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
	t.Cleanup(func() { fetchRateLimitForHostFunc, healthRunGH = originalRate, originalGH })
	fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
		return rateLimitResource{Limit: 5000, Remaining: 4000}, nil
	}
	end := time.Date(2026, 10, 16, 4, 0, 0, 0, time.UTC)
	boundary := end.Add(-100 * time.Second)
	jobChecks := make(map[string]int)
	rolledPages := []int{}
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		endpoint := args[3]
		switch {
		case strings.HasSuffix(endpoint, "/jobs?per_page=1"):
			jobChecks[endpoint]++
			assert.Contains(t, endpoint, "/attempts/2/jobs")
			if strings.Contains(endpoint, "/runs/1000/") {
				return []byte(`{"jobs":[{"id":42}]}`), nil
			}
			return []byte(`{"jobs":[]}`), nil
		case strings.Contains(endpoint, "/contents/"):
			if strings.Contains(endpoint, ".md?") {
				return []byte(strings.Repeat("c", 40)), nil
			}
			return []byte(`# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"settings","body_hash":"prompt"}`), nil
		}
		parsed, err := url.Parse(endpoint)
		require.NoError(t, err)
		page, err := strconv.Atoi(parsed.Query().Get("page"))
		require.NoError(t, err)
		rolled := strings.HasSuffix(parsed.Query().Get("created"), ".."+boundary.Format(time.RFC3339))
		if rolled {
			rolledPages = append(rolledPages, page)
		}
		count := 2 * healthMinimumRuns
		if rolled && page == 2 {
			count = healthMinimumRuns
		}
		var responses []workflowRunAPIResponse
		for i := range count {
			run := workflowRunAPIResponse{ID: int64(page*10 + i), Name: "Example",
				Path: ".github/workflows/example.lock.yml", Status: "completed", Conclusion: "action_required",
				Event: "schedule", HeadBranch: "main", HeadSHA: strings.Repeat("a", 40), RunAttempt: 1,
				CreatedAt: end.Add(-time.Duration(page) * time.Second)}
			if page == 100 || rolled {
				run.ID, run.Conclusion, run.RunAttempt, run.CreatedAt = int64(1000+i), "failure", 2, boundary
			}
			if rolled && page == 2 {
				run.ID, run.Conclusion, run.RunAttempt = int64(2000+i), "success", 1
			}
			responses = append(responses, run)
		}
		out, err := json.Marshal(map[string]any{"workflow_runs": responses})
		require.NoError(t, err)
		return out, nil
	}
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo", EndDate: end.Format(time.RFC3339)}
	runs, err := sampleHealthWorkflow(GitHubWorkflow{ID: 1, Name: "Example",
		Path: ".github/workflows/example.lock.yml"}, end.Add(-time.Hour).Format(time.RFC3339), "org/repo", false, collection)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, rolledPages)
	require.Len(t, runs, healthMinimumRuns+1)
	assert.Equal(t, int64(1000), runs[0].DatabaseID, "retain job-backed failure only once")
	require.Len(t, jobChecks, 10)
	for endpoint, checks := range jobChecks {
		assert.Equal(t, 1, checks, "overlap must reuse both empty and nonempty jobs responses: %s", endpoint)
	}
	assert.False(t, collection.Partial)
	assert.Equal(t, 116, collection.APIRequests, "102 run pages, 10 jobs probes, 4 revision lookups")
}

func TestHealthFilteredPagesStillRespectRequestBudget(t *testing.T) {
	originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
	t.Cleanup(func() { fetchRateLimitForHostFunc, healthRunGH = originalRate, originalGH })
	fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
		return rateLimitResource{Limit: 5000, Remaining: 4000}, nil
	}
	calls := 0
	healthRunGH = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		var runs []workflowRunAPIResponse
		for i := range 2 * healthMinimumRuns {
			runs = append(runs, workflowRunAPIResponse{ID: int64(calls*10 + i), Conclusion: "action_required"})
		}
		out, err := json.Marshal(map[string]any{"workflow_runs": runs})
		require.NoError(t, err)
		return out, nil
	}
	collection := &HealthCollection{Host: "github.com", APIRequests: healthMaximumAPIRequests - 2}
	_, err := sampleHealthWorkflow(GitHubWorkflow{ID: 1, Name: "Example",
		Path: ".github/workflows/example.lock.yml"}, "2026-10-08", "org/repo", false, collection)
	require.ErrorIs(t, err, errHealthAPIBudget)
	assert.Equal(t, 2, calls)
	assert.True(t, collection.Partial)
	assert.Equal(t, "request_budget", collection.Reason)
}
