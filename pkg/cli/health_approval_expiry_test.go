//go:build !integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func healthApprovalTestPage(message string) []byte {
	return []byte(`<html><body><div data-target="annotation-message.annotationContainer"><div>` + message + `</div></div></body></html>`)
}

func TestHealthApprovalExpiredAnnotation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		page    []byte
		expired bool
		missing bool
	}{
		{name: "GitHub annotation", page: healthApprovalTestPage(healthApprovalExpiryMessage), expired: true},
		{name: "nested markup and whitespace", page: healthApprovalTestPage("This workflow run required <strong>approval</strong>\n but was not approved before it expired."), expired: true},
		{name: "HTML entity", page: healthApprovalTestPage("This workflow run required approval but was not approved before it expired&#46;"), expired: true},
		{name: "workflow validation failure", page: healthApprovalTestPage("Invalid workflow file")},
		{name: "mixed failure causes", page: append(healthApprovalTestPage(healthApprovalExpiryMessage), healthApprovalTestPage("Invalid workflow file")...)},
		{name: "workflow title is not evidence", page: []byte("<h1>" + healthApprovalExpiryMessage + "</h1>"), missing: true},
		{name: "script is not evidence", page: healthApprovalTestPage("<script>" + healthApprovalExpiryMessage + "</script>")},
		{name: "login page", page: []byte("<html>Sign in to GitHub</html>"), missing: true},
		{name: "invalid response", page: []byte(`{"error":"not HTML"}`), missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expired, err := healthApprovalExpiredAnnotation(tc.page)
			if tc.missing {
				require.ErrorContains(t, err, "annotations unavailable")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expired, expired)
		})
	}
}

func TestHealthApprovalExpiryFiltersCandidatesWithoutHidingRuntimeFailures(t *testing.T) {
	for _, executed := range []bool{false, true} {
		t.Run(strconv.FormatBool(executed), func(t *testing.T) {
			stubHealthFailureCandidates(t)
			original := healthRunGH
			t.Cleanup(func() { healthRunGH = original })
			updated := time.Date(2026, 10, 22, 10, 53, 27, 0, time.UTC)
			path := ".github/workflows/ai-moderator.lock.yml"
			expired := workflowRunAPIResponse{ID: 39766415060, Name: "AI Moderator", Path: path,
				Status: "completed", Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated,
				HTMLURL: "https://github.com/org/repo/actions/runs/39766415060"}
			batch := []workflowRunAPIResponse{expired}
			if executed {
				batch = append(batch, workflowRunAPIResponse{ID: 42, Name: "AI Moderator", Path: path,
					Status: "completed", Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated,
					HTMLURL: "https://github.com/org/repo/actions/runs/42"})
			}
			healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
				return batch, time.Time{}, nil
			}
			collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
				workflows:    []GitHubWorkflow{{Path: path}},
				APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
			var endpoints []string
			healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				require.True(t, collection.FailureListingComplete, "inventory must precede verification")
				endpoint := args[3]
				endpoints = append(endpoints, endpoint)
				switch {
				case endpoint == "https://github.com/org/repo/actions/runs/39766415060/attempts/1":
					assert.Equal(t, []string{"--header", "Accept: text/html"}, args[4:])
					return healthApprovalTestPage(healthApprovalExpiryMessage), nil
				case strings.Contains(endpoint, "/runs/39766415060/"):
					assert.Contains(t, endpoint, "/attempts/1/jobs?")
					return []byte(`{"total_count":0,"jobs":[]}`), nil
				default:
					assert.Contains(t, endpoint, "/runs/42/attempts/1/jobs?")
					return []byte(`{"total_count":1,"jobs":[{"id":99,"name":"agent","status":"completed","conclusion":"failure"}]}`), nil
				}
			}
			runs, err := collectHealthFailures("", "org/repo", collection)
			require.NoError(t, err)
			require.Len(t, collection.ApprovalExpiryEvidence, 1)
			evidence := collection.ApprovalExpiryEvidence[expired.ID]
			assert.Equal(t, "ignored", evidence.Status)
			assert.Equal(t, "approval_expired", evidence.Reason)
			assert.Equal(t, healthApprovalExpiryMessage, evidence.Message)
			assert.Equal(t, expired.UpdatedAt, evidence.UpdatedAt)
			assert.Equal(t, 1, collection.ApprovalRequests)
			require.NoError(t, observeHealthApprovalExpiry(expired.workflowRun(), collection))
			assert.Equal(t, 1, collection.ApprovalRequests, "overlapping samples reuse evidence without subtracting twice")
			if executed {
				require.Len(t, runs, 1)
				assert.Equal(t, int64(42), runs[0].DatabaseID)
				require.Len(t, collection.PotentialIssues, 1)
				assert.Equal(t, int64(42), collection.PotentialIssues[0].RunID)
				assert.Equal(t, "revision_unknown", collection.PotentialIssues[0].RepresentativeStatus)
				assert.Equal(t, 1, collection.PotentialIssues[0].FailedRuns)
				assert.Equal(t, 1, collection.FailureCandidates)
				assert.Equal(t, 1, collection.FailureCandidateWorkflows)
				assert.Len(t, endpoints, 4, "only the jobless attempt needs an annotation request")
			} else {
				assert.Empty(t, runs)
				assert.Empty(t, collection.PotentialIssues)
				assert.Zero(t, collection.FailureCandidates)
				assert.Zero(t, collection.FailureCandidateWorkflows)
				health := calculateCurrentWorkflowHealth("AI Moderator", runs, HealthConfig{Threshold: 80, collection: collection})
				assert.Equal(t, "no_recent_runs", health.Status)
				assert.Zero(t, health.SuccessCount, "filtered failures are not successes")
			}
			var text bytes.Buffer
			require.NoError(t, renderHealthCollection(&text, collection))
			assert.Contains(t, text.String(), `[health_approval_expiry] run=39766415060 attempt=1 status=ignored reason="approval_expired"`)
		})
	}
}

func TestHealthApprovalExpiryRequiresAffirmativeEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		page   []byte
		err    error
		status string
	}{
		{name: "annotation unavailable", page: []byte("<html>Sign in</html>"), status: "unknown"},
		{name: "request denied", err: errors.New("HTTP 403"), status: "unknown"},
		{name: "validation failure", page: healthApprovalTestPage("Invalid workflow file"), status: "retained"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := healthRunGH
			t.Cleanup(func() { healthRunGH = original })
			run := WorkflowRun{DatabaseID: 42, Attempt: 2, Status: "completed", Conclusion: "failure",
				UpdatedAt: time.Now(), WorkflowPath: ".github/workflows/example.lock.yml"}
			collection := &HealthCollection{Host: "github.example.com", repoOverride: "github.example.com/org/repo",
				APIRateLimit:     &GitHubAPIRateLimitState{Remaining: 4000},
				failureInventory: []WorkflowRun{run}, FailureCandidates: 1, FailureCandidateWorkflows: 1,
				PotentialIssues: []HealthPotentialIssue{{WorkflowPath: run.WorkflowPath, RunID: run.DatabaseID, FailedRuns: 1}}}
			healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				assert.Equal(t, "github.example.com", args[2])
				assert.Equal(t, "https://github.example.com/org/repo/actions/runs/42/attempts/2", args[3])
				return tc.page, tc.err
			}
			require.NoError(t, observeHealthApprovalExpiry(run, collection))
			assert.Equal(t, tc.status, collection.ApprovalExpiryEvidence[42].Status)
			assert.Equal(t, 1, collection.FailureCandidates)
			assert.Len(t, collection.PotentialIssues, 1)
		})
	}
}

func TestHealthApprovalExpiryCacheIdentityAndBudget(t *testing.T) {
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
		APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
	healthRunGH = func(context.Context, string, ...string) ([]byte, error) {
		return healthApprovalTestPage(healthApprovalExpiryMessage), nil
	}
	run := WorkflowRun{DatabaseID: 42, Attempt: 1, Status: "completed", Conclusion: "failure", UpdatedAt: time.Now(), Repository: "org/repo"}
	for _, attempt := range []int{1, 2} {
		run.Attempt = attempt
		for range 2 {
			require.NoError(t, observeHealthApprovalExpiry(run, collection))
		}
	}
	assert.Equal(t, 2, collection.ApprovalRequests)
	run.UpdatedAt = run.UpdatedAt.Add(time.Second)
	require.NoError(t, observeHealthApprovalExpiry(run, collection))
	assert.Equal(t, 3, collection.ApprovalRequests, "changed update time invalidates cached evidence")

	collection.ApprovalRequests = healthMaximumApprovalRequests
	run.DatabaseID++
	require.NoError(t, observeHealthApprovalExpiry(run, collection))
	assert.Equal(t, "unknown", collection.ApprovalExpiryEvidence[run.DatabaseID].Status)
	assert.Equal(t, "approval_annotation_request_limit", collection.ApprovalExpiryEvidence[run.DatabaseID].Reason)
	assert.Equal(t, 3, collection.APIRequests)

	collection = &HealthCollection{Host: "github.com", APIRequests: healthMaximumAPIRequests}
	require.ErrorIs(t, observeHealthApprovalExpiry(run, collection), errHealthAPIBudget)
	assert.True(t, collection.Partial)
	assert.Equal(t, "request_budget", collection.Reason)
	assert.Equal(t, "unknown", collection.ApprovalExpiryEvidence[run.DatabaseID].Status)
	assert.Zero(t, collection.ApprovalRequests)
}

func TestHealthApprovalExpiryUsesRepositoryIdentityNotRunURL(t *testing.T) {
	original := healthRunGH
	t.Cleanup(func() { healthRunGH = original })
	collection := &HealthCollection{Host: "github.com", APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000}}
	run := WorkflowRun{DatabaseID: 42, Attempt: 1, Status: "completed", Conclusion: "failure",
		UpdatedAt: time.Now(), Repository: "org/repo", URL: "https://unrelated.example/ignore"}
	healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, "https://github.com/org/repo/actions/runs/42/attempts/1", args[3])
		return healthApprovalTestPage(healthApprovalExpiryMessage), nil
	}
	require.NoError(t, observeHealthApprovalExpiry(run, collection))
	assert.Equal(t, "ignored", collection.ApprovalExpiryEvidence[42].Status)
	assert.Equal(t, "https://github.com/org/repo/actions/runs/42/attempts/1", collection.ApprovalExpiryEvidence[42].URL)

	run.DatabaseID++
	run.Repository = ""
	require.NoError(t, observeHealthApprovalExpiry(run, collection))
	assert.Equal(t, "unknown", collection.ApprovalExpiryEvidence[43].Status)
	assert.Equal(t, "missing_repository", collection.ApprovalExpiryEvidence[43].Reason)
	assert.Equal(t, 1, collection.ApprovalRequests)
}

func TestHealthApprovalExpiryStopsBothCollectionPaths(t *testing.T) {
	for _, caller := range []string{"failure_first", "chronological"} {
		for _, stop := range []string{"request_budget", "quota_check", "rate_limit_response", "last_request"} {
			t.Run(caller+"/"+stop, func(t *testing.T) {
				originalGH, originalRate := healthRunGH, fetchRateLimitForHostFunc
				t.Cleanup(func() { healthRunGH, fetchRateLimitForHostFunc = originalGH, originalRate })
				path := ".github/workflows/example.lock.yml"
				collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
					APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
					workflows:    []GitHubWorkflow{{ID: 1, Path: path}}}
				updated := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
				batch := []workflowRunAPIResponse{
					{ID: 1, Path: path, Status: "completed", Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated},
					{ID: 2, Path: path, Status: "completed", Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated},
				}
				quotaChecks, annotationRequests := 0, 0
				fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
					quotaChecks++
					return rateLimitResource{Remaining: RateLimitThreshold, Reset: updated.Unix()}, nil
				}
				var endpoints []string
				healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
					endpoint := args[3]
					endpoints = append(endpoints, endpoint)
					switch {
					case strings.Contains(endpoint, "/runs/1/attempts/1/jobs?"):
						switch stop {
						case "request_budget":
							collection.APIRequests = healthMaximumAPIRequests
						case "quota_check":
							collection.APIRateLimit.Remaining = RateLimitThreshold
						case "last_request":
							collection.APIRequests = healthMaximumAPIRequests - 1
							collection.checkedRequests = collection.APIRequests
						}
						return []byte(`{"jobs":[]}`), nil
					case endpoint == "https://github.com/org/repo/actions/runs/1/attempts/1":
						annotationRequests++
						if stop == "rate_limit_response" {
							return []byte("<html>API rate limit exceeded</html>"), errors.New("HTTP 403")
						}
						return healthApprovalTestPage("Invalid workflow file"), nil
					default:
						require.Len(t, endpoints, 1, "stop before requesting another candidate or page")
						out, err := json.Marshal(map[string]any{"workflow_runs": batch})
						require.NoError(t, err)
						return out, nil
					}
				}
				var err error
				if caller == "failure_first" {
					_, err = collectHealthFailures("", "org/repo", collection)
				} else {
					_, err = sampleHealthWorkflow(collection.workflows[0], "", "org/repo", false, collection)
				}
				require.Error(t, err)
				if stop == "request_budget" || stop == "last_request" {
					require.ErrorIs(t, err, errHealthAPIBudget)
					assert.Equal(t, "request_budget", collection.Reason)
				} else {
					require.ErrorIs(t, err, errLogsAPIRateLimitReached)
					assert.Equal(t, "rate_limit", collection.Reason)
				}
				assert.True(t, collection.stopped)
				assert.True(t, collection.Partial)
				assert.False(t, collection.FailurePassComplete)
				if stop == "quota_check" {
					assert.Equal(t, 1, quotaChecks)
				}
				if stop == "request_budget" || stop == "quota_check" {
					assert.Zero(t, annotationRequests)
					assert.Len(t, endpoints, 2)
				} else {
					assert.Equal(t, 1, annotationRequests)
					assert.Len(t, endpoints, 3)
				}
				assert.Equal(t, annotationRequests, collection.ApprovalRequests)
				assert.NotContains(t, collection.Warning, "<html>")
				require.Contains(t, collection.ApprovalExpiryEvidence, int64(1))
				assert.NotEqual(t, "ignored", collection.ApprovalExpiryEvidence[1].Status)
			})
		}
	}
}

func TestHealthApprovalExpiryRequestErrorsStayBoundedAndRetainCandidates(t *testing.T) {
	for _, response := range []string{"", "<html>" + strings.Repeat("inaccessible page\n", 20000) + "</html>"} {
		t.Run(strconv.Itoa(len(response)), func(t *testing.T) {
			stubHealthFailureCandidates(t)
			original := healthRunGH
			t.Cleanup(func() { healthRunGH = original })
			path := ".github/workflows/example.lock.yml"
			updated := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
			collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo",
				APIRateLimit: &GitHubAPIRateLimitState{Remaining: 4000},
				workflows:    []GitHubWorkflow{{Path: path}}}
			healthListFailureCandidates = func(ListWorkflowRunsOptions, int, string, *HealthCollection) ([]workflowRunAPIResponse, time.Time, error) {
				return []workflowRunAPIResponse{
					{ID: 1, Path: path, Status: "completed", Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated},
					{ID: 2, Path: path, Status: "completed", Conclusion: "failure", RunAttempt: 1, UpdatedAt: updated},
				}, time.Time{}, nil
			}
			requests := 0
			healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				requests++
				if strings.HasPrefix(args[3], "https://") {
					return []byte(response), errors.New("HTTP 502")
				}
				return []byte(`{"jobs":[]}`), nil
			}
			runs, err := collectHealthFailures("", "org/repo", collection)
			require.NoError(t, err)
			assert.Empty(t, runs)
			assert.Equal(t, 4, requests, "ordinary errors must not stop later candidate checks")
			assert.False(t, collection.stopped)
			assert.False(t, collection.Partial)
			assert.True(t, collection.FailurePassComplete)
			assert.Equal(t, 2, collection.FailureCandidates)
			require.Len(t, collection.PotentialIssues, 1)
			assert.Equal(t, 2, collection.PotentialIssues[0].FailedRuns)
			for _, evidence := range collection.ApprovalExpiryEvidence {
				assert.Equal(t, "unknown", evidence.Status)
				assert.LessOrEqual(t, len(evidence.Reason), 256)
				assert.NotContains(t, evidence.Reason, "<html>")
			}
			out, err := json.Marshal(collection)
			require.NoError(t, err)
			assert.Less(t, len(out), 4096)
			var text bytes.Buffer
			require.NoError(t, renderHealthCollection(&text, collection))
			assert.Less(t, text.Len(), 2048)
		})
	}
}
