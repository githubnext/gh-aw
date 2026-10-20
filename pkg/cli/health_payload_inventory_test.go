//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthPayloadInventoryFindsFleetSignalsWithoutEnrichment(t *testing.T) {
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	collection := &HealthCollection{Host: "github.com", APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
	var responses []workflowRunAPIResponse
	for i := range 50 {
		path := fmt.Sprintf(".github/workflows/example-%d.lock.yml", i)
		collection.workflows = append(collection.workflows, GitHubWorkflow{Path: path, Name: path})
		for j := range 3 {
			responses = append(responses, workflowRunAPIResponse{ID: int64(i*3 + j + 1),
				Path: path, Conclusion: "failure", HTMLURL: fmt.Sprintf("https://github.com/org/repo/actions/runs/%d", i*3+j+1)})
		}
	}
	calls := 0
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls++
		require.Contains(t, args[3], "/actions/runs?status=failure", "inventory must not fetch jobs, files, or individual workflows")
		first := min((calls-1)*100, len(responses))
		last := min(calls*100, len(responses))
		out, err := json.Marshal(map[string]any{"workflow_runs": responses[first:last]})
		require.NoError(t, err)
		return out, nil
	}
	runs, err := listHealthFailureInventory("", "org/repo", collection)
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "150 failed runs across 50 workflows need only two listing requests")
	assert.Len(t, runs, 150)
	assert.Equal(t, 150, collection.FailureCandidates)
	assert.Equal(t, 50, collection.FailureCandidateWorkflows)
	assert.Len(t, collection.PotentialIssues, 50)
	for _, issue := range collection.PotentialIssues {
		assert.Equal(t, "run_list_only", issue.Evidence)
		assert.Equal(t, "unverified", issue.RepresentativeStatus)
		assert.Equal(t, 3, issue.FailedRuns)
		assert.NotEmpty(t, issue.URL)
	}
	assert.True(t, collection.FailureListingComplete)
	assert.False(t, collection.FailurePassComplete)
	assert.Zero(t, collection.RevisionLookups)
	assert.Empty(t, collection.jobPresence)
}

func TestHealthPayloadSignalsSurviveInventoryBudgetExhaustion(t *testing.T) {
	stubHealthFailureCandidates(t)
	path := ".github/workflows/example.lock.yml"
	collection := &HealthCollection{workflows: []GitHubWorkflow{{Path: path}}}
	healthListFailureCandidates = func(opts ListWorkflowRunsOptions, page int, _ string, state *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		if page == 2 {
			stopHealthCollection(state, errHealthAPIBudget)
			return nil, time.Time{}, errHealthAPIBudget
		}
		batch := make([]workflowRunAPIResponse, opts.Limit)
		for i := range batch {
			batch[i] = workflowRunAPIResponse{ID: int64(i + 1), Path: path, Conclusion: "failure"}
		}
		return batch, time.Time{}, nil
	}
	runs, err := fetchWorkflowRuns("", "", "org/repo", false, collection)
	require.NoError(t, err)
	assert.Empty(t, runs, "unverified payloads must not become confirmed health evidence")
	assert.Equal(t, 100, collection.FailureCandidates)
	require.Len(t, collection.PotentialIssues, 1)
	assert.Equal(t, "unverified", collection.PotentialIssues[0].RepresentativeStatus)
	assert.True(t, collection.Partial)
	assert.False(t, collection.FailureListingComplete)
	assert.False(t, collection.FailurePassComplete)
	assert.Zero(t, collection.RevisionLookups)
}

func TestHealthJoblessCandidateDoesNotHideExecutedRepresentative(t *testing.T) {
	stubHealthFailureCandidates(t)
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	path := ".github/workflows/example.lock.yml"
	collection := &HealthCollection{Host: "github.com", workflows: []GitHubWorkflow{{Path: path}},
		APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
	healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		return []workflowRunAPIResponse{{ID: 1, Path: path, Conclusion: "failure"}, {ID: 2, Path: path, Conclusion: "failure"}}, time.Time{}, nil
	}
	jobs := 0
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		require.True(t, collection.FailureListingComplete, "complete the cheap inventory before any verification")
		jobs++
		if jobs == 1 {
			return []byte(`{"jobs":[]}`), nil
		}
		return []byte(`{"jobs":[{"id":42}]}`), nil
	}
	runs, err := collectHealthFailures("", "org/repo", collection)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, int64(2), runs[0].DatabaseID)
	assert.Equal(t, 2, jobs)
	assert.Equal(t, "jobless", collection.PotentialIssues[0].RepresentativeStatus)
	assert.Equal(t, 2, collection.FailureCandidates)
}
