//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthSharedJobBoundaryWithoutArtifacts(t *testing.T) {
	originalGH, originalRate := healthRunGH, fetchRateLimitForHostFunc
	t.Cleanup(func() { healthRunGH, fetchRateLimitForHostFunc = originalGH, originalRate })
	fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
		return rateLimitResource{Remaining: 4000}, nil
	}
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
		APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		endpoint := args[3]
		assert.Contains(t, endpoint, "/attempts/1/jobs?")
		if strings.HasSuffix(endpoint, "?per_page=1") {
			return []byte(`{"total_count":2,"jobs":[{"name":"activation","conclusion":"failure","steps":[{"name":"Restore daily AIC scan observations","number":3,"conclusion":"failure"}]}]}`), nil
		}
		assert.True(t, strings.HasSuffix(endpoint, "?per_page=100&page=1"))
		return []byte(`{"total_count":2,"jobs":[
			{"name":"activation","conclusion":"failure","steps":[{"name":"Restore daily AIC scan observations","number":3,"conclusion":"failure"}]},
			{"name":"conclusion","conclusion":"failure","steps":[{"name":"Fail workflow if agent failed","conclusion":"failure"}]}
		]}`), nil
	}
	var runs []WorkflowRun
	for i := range 12 {
		run := WorkflowRun{DatabaseID: int64(i + 1), WorkflowName: fmt.Sprintf("Workflow %d", i),
			WorkflowPath: fmt.Sprintf(".github/workflows/test-%d.lock.yml", i),
			URL:          fmt.Sprintf("https://github.com/org/repo/actions/runs/%d", i+1), Repository: "org/repo",
			Status: "completed", Conclusion: "failure", HeadBranch: "main", Event: "schedule",
			Attempt: 1, UpdatedAt: time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)}
		hasJobs, err := healthRunHasJobs(run, collection)
		require.NoError(t, err)
		require.True(t, hasJobs)
		require.NoError(t, collectHealthRepresentativeJobs(run, collection))
		runs = append(runs, run)
	}
	clusters, plan := buildHealthFailureClusters(runs, HealthConfig{
		RepoOverride: "org/repo", LogsDir: t.TempDir(), collection: collection})
	require.NotNil(t, plan)
	assert.Equal(t, 12, plan.UniqueFailedRuns)
	assert.Equal(t, 12, plan.ObservedEvidenceRuns)
	assert.Zero(t, plan.MetadataOnlyRuns)
	var boundaries []FailureCluster
	for _, cluster := range clusters {
		if cluster.Signal.Step == "Restore daily AIC scan observations" {
			boundaries = append(boundaries, cluster)
		}
	}
	require.Len(t, boundaries, 1, "twelve workflows share one observed failed-step boundary")
	assert.Equal(t, 12, boundaries[0].Count)
	assert.Len(t, boundaries[0].Workflows, 12)
	assert.Equal(t, "activation", boundaries[0].Signal.Job)
	assert.Equal(t, "failure_boundary", boundaries[0].SignalRole)
	require.Len(t, plan.SelectedRuns, 1, "overlapping symptoms must not spend extra audit slots")
	assert.Equal(t, "main", plan.SelectedRuns[0].Branch)
	assert.Equal(t, "schedule", plan.SelectedRuns[0].Event)
	assert.Equal(t, "1", plan.SelectedRuns[0].Attempt)
	assert.Equal(t, runs[0].UpdatedAt, plan.SelectedRuns[0].UpdatedAt)
	assert.Equal(t, 24, collection.JobRequests)
	assert.Equal(t, collection.APIRequests, collection.JobRequests)
	for _, evidence := range collection.JobEvidence {
		assert.True(t, evidence.Complete)
		assert.Equal(t, 2, evidence.Jobs)
	}
}

func TestHealthRepresentativeJobPaginationIsBounded(t *testing.T) {
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	total := 301
	key := healthRunAttempt{id: 42, attempt: 2}
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
		APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
		jobs:         map[healthRunAttempt]*healthJobs{key: {total: &total, details: []JobInfoWithDuration{{}}}}}
	pages := 0
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		pages++
		assert.Equal(t, fmt.Sprintf("repos/org/repo/actions/runs/42/attempts/2/jobs?per_page=100&page=%d", pages), args[3])
		out, err := json.Marshal(map[string]any{"total_count": total, "jobs": make([]JobInfo, 100)})
		require.NoError(t, err)
		return out, nil
	}
	run := WorkflowRun{DatabaseID: 42, Attempt: 2}
	require.NoError(t, collectHealthRepresentativeJobs(run, collection))
	assert.Equal(t, healthMaximumJobPages, pages)
	assert.Len(t, collection.jobs[key].details, 300)
	assert.False(t, collection.JobEvidence[42].Complete)
	assert.Equal(t, 300, collection.JobEvidence[42].Jobs)
}

func TestHealthRepresentativeJobErrorsPreservePresenceEvidence(t *testing.T) {
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	for _, response := range []string{"api-error", "{", `{}`, `{"jobs":null}`} {
		t.Run(response, func(t *testing.T) {
			total := 2
			key := healthRunAttempt{id: 42, attempt: 1}
			collection := &HealthCollection{Host: "github.com",
				APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
				jobs: map[healthRunAttempt]*healthJobs{key: {total: &total, details: []JobInfoWithDuration{
					{Name: "activation", Conclusion: "failure"},
				}}}}
			healthRunGH = func(context.Context, string, ...string) ([]byte, error) {
				if response == "api-error" {
					return nil, errors.New("HTTP 502")
				}
				return []byte(response), nil
			}
			err := collectHealthRepresentativeJobs(WorkflowRun{DatabaseID: 42, Attempt: 1}, collection)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "health job evidence")
			require.Len(t, collection.jobs[key].details, 1)
			assert.Equal(t, "activation", collection.jobs[key].details[0].Name)
			assert.False(t, collection.JobEvidence[42].Complete)
		})
	}
}
