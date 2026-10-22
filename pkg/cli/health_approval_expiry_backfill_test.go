//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthApprovalExpiryBackfillsExecutedFailureBeforeNewerSuccesses(t *testing.T) {
	for _, nextPage := range []bool{false, true} {
		t.Run(strconv.FormatBool(nextPage), func(t *testing.T) {
			stubHealthFailureCandidates(t)
			originalGH, originalList := healthRunGH, healthListWorkflowRuns
			t.Cleanup(func() { healthRunGH, healthListWorkflowRuns = originalGH, originalList })
			path := ".github/workflows/example.lock.yml"
			otherPath := ".github/workflows/other.lock.yml"
			updated := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
			collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
				APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
				workflows:    []GitHubWorkflow{{ID: 1, Path: path}}}
			var first []workflowRunAPIResponse
			for i := range healthMinimumRuns {
				first = append(first, workflowRunAPIResponse{ID: int64(i + 1), Path: path, Status: "completed",
					Conclusion: "failure", RunAttempt: 1, CreatedAt: updated.Add(-time.Duration(i) * time.Hour), UpdatedAt: updated})
			}
			executed := workflowRunAPIResponse{ID: 6, Path: path, Status: "completed", Conclusion: "failure", RunAttempt: 1,
				CreatedAt: updated.Add(-12 * time.Hour), UpdatedAt: updated}
			if nextPage {
				collection.workflows = append(collection.workflows, GitHubWorkflow{ID: 2, Path: otherPath})
				for len(first) < 100 {
					first = append(first, workflowRunAPIResponse{ID: int64(100 + len(first)), Path: otherPath,
						Conclusion: "failure", RunAttempt: 1})
				}
			} else {
				first = append(first, executed)
			}
			listingRequests := 0
			healthListFailureCandidates = func(opts ListWorkflowRunsOptions, page int, _ string, _ *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
				listingRequests++
				assert.Equal(t, 100, opts.Limit)
				assert.Equal(t, listingRequests, page, "resume without re-fetching earlier pages")
				if page == 1 {
					assert.Zero(t, collection.JobRequests, "initial inventory remains list-only")
					return first, updated.Add(-12 * time.Hour), nil
				}
				require.True(t, nextPage)
				require.Equal(t, 2, page)
				assert.Positive(t, collection.ApprovalRequests, "only confirmed exclusions trigger additional listing")
				return []workflowRunAPIResponse{executed}, executed.CreatedAt, nil
			}
			healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				require.True(t, collection.FailureListingComplete, "finish inventory/backfill before verifying candidates")
				if strings.HasPrefix(args[3], "https://") {
					assert.NotContains(t, args[3], "/runs/6/")
					return healthApprovalTestPage(healthApprovalExpiryMessage), nil
				}
				for id := 1; id <= healthMinimumRuns; id++ {
					if strings.Contains(args[3], fmt.Sprintf("/runs/%d/", id)) {
						return []byte(`{"jobs":[]}`), nil
					}
				}
				return []byte(`{"total_count":1,"jobs":[{"id":42,"name":"agent","conclusion":"failure"}]}`), nil
			}
			healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, page int, _ *HealthCollection) ([]WorkflowRun, int, error) {
				require.True(t, collection.FailurePassComplete)
				require.Equal(t, 1, page)
				if opts.WorkflowName == "2" {
					return nil, 0, nil
				}
				var successes []WorkflowRun
				for i := range healthMinimumRuns {
					successes = append(successes, WorkflowRun{DatabaseID: int64(1000 + i), WorkflowPath: path,
						Status: "completed", Conclusion: "success", CreatedAt: updated.Add(-time.Duration(6+i) * time.Hour)})
				}
				return successes, len(successes), nil
			}
			runs, err := fetchWorkflowRuns("", "", "org/repo", false, collection)
			require.NoError(t, err)
			assert.True(t, collection.FailureListingComplete)
			assert.True(t, collection.FailurePassComplete)
			assert.False(t, collection.Partial)
			assert.Equal(t, healthMinimumRuns, collection.ApprovalRequests)
			assert.LessOrEqual(t, listingRequests, 2)
			found := false
			for _, run := range runs {
				if run.DatabaseID == executed.ID {
					found = true
				}
			}
			assert.True(t, found, "newer successes must not hide the older executed failure")
			for _, issue := range collection.PotentialIssues {
				assert.LessOrEqual(t, issue.FailedRuns, healthMinimumRuns)
				if issue.WorkflowPath == path {
					assert.Equal(t, int64(6), issue.RunID)
					assert.Equal(t, 1, issue.FailedRuns)
					assert.Equal(t, "revision_unknown", issue.RepresentativeStatus)
				}
			}
			if nextPage {
				assert.Equal(t, 2, listingRequests)
				assert.Equal(t, healthMinimumRuns+1, collection.FailureCandidates)
				assert.Equal(t, 2, collection.FailureCandidateWorkflows)
			} else {
				assert.Equal(t, 1, listingRequests, "reuse overflow from the same fetched page")
				assert.Equal(t, 1, collection.FailureCandidates)
				assert.Equal(t, 1, collection.FailureCandidateWorkflows)
			}
		})
	}
}

func TestHealthApprovalExpiryBackfillPreservesRequestAndCandidateLimits(t *testing.T) {
	stubHealthFailureCandidates(t)
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	path := ".github/workflows/example.lock.yml"
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
		APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}, workflows: []GitHubWorkflow{{Path: path}}}
	updated := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	listingRequests := 0
	healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		listingRequests++
		var batch []workflowRunAPIResponse
		for i := range healthMaximumApprovalRequests + healthMinimumRuns {
			batch = append(batch, workflowRunAPIResponse{ID: int64(i + 1), Path: path, Status: "completed",
				Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated})
		}
		return batch, updated, nil
	}
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if strings.HasPrefix(args[3], "https://") {
			return healthApprovalTestPage(healthApprovalExpiryMessage), nil
		}
		return []byte(`{"jobs":[]}`), nil
	}
	runs, err := collectHealthFailures("", "org/repo", collection)
	require.NoError(t, err)
	assert.Empty(t, runs)
	assert.Equal(t, 1, listingRequests)
	assert.Equal(t, healthMaximumApprovalRequests, collection.ApprovalRequests)
	assert.Equal(t, healthMinimumRuns, collection.FailureCandidates, "unverified overflow stays visible after annotation cap")
	require.Len(t, collection.PotentialIssues, 1)
	assert.Equal(t, healthMinimumRuns, collection.PotentialIssues[0].FailedRuns)
	assert.Equal(t, int64(healthMaximumApprovalRequests+1), collection.PotentialIssues[0].RunID)
	assert.Equal(t, "jobless", collection.PotentialIssues[0].RepresentativeStatus)
	assert.Equal(t, "approval_annotation_request_limit", collection.ApprovalExpiryEvidence[healthMaximumApprovalRequests+1].Reason)
}

func TestHealthApprovalExpiryBackfillPreservesListingPageLimit(t *testing.T) {
	stubHealthFailureCandidates(t)
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	path, otherPath := ".github/workflows/example.lock.yml", ".github/workflows/other.lock.yml"
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
		APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
		workflows:    []GitHubWorkflow{{Path: path}, {Path: otherPath}}}
	updated := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	listingRequests := 0
	healthListFailureCandidates = func(opts ListWorkflowRunsOptions, page int, _ string, _ *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
		listingRequests++
		require.Equal(t, listingRequests, page)
		batch := make([]workflowRunAPIResponse, opts.Limit)
		for i := range batch {
			candidatePath := ".github/workflows/ci.yml"
			if page == 1 {
				candidatePath = otherPath
				if i < healthMinimumRuns {
					candidatePath = path
				}
			}
			batch[i] = workflowRunAPIResponse{ID: int64(page*100 + i), Path: candidatePath,
				Status: "completed", Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated}
		}
		return batch, updated.Add(-time.Duration(page) * time.Hour), nil
	}
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if strings.HasPrefix(args[3], "https://") {
			return healthApprovalTestPage(healthApprovalExpiryMessage), nil
		}
		for id := 100; id < 100+healthMinimumRuns; id++ {
			if strings.Contains(args[3], fmt.Sprintf("/runs/%d/", id)) {
				return []byte(`{"jobs":[]}`), nil
			}
		}
		return json.Marshal(map[string]any{"total_count": 1, "jobs": []map[string]any{{"id": 42}}})
	}
	_, err := collectHealthFailures("", "org/repo", collection)
	require.NoError(t, err)
	assert.Equal(t, healthMaximumFailureListingPages, listingRequests)
	assert.Equal(t, healthMinimumRuns, collection.ApprovalRequests)
	assert.True(t, collection.Partial)
	assert.False(t, collection.FailureListingComplete)
	assert.False(t, collection.FailurePassComplete)
	assert.Equal(t, "failure_listing_limit", collection.Reason)
}
