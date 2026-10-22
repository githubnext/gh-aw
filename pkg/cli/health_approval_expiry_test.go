//go:build !integration

package cli

import (
	"bytes"
	"context"
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
			observeHealthApprovalExpiry(expired.workflowRun(), collection)
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
			observeHealthApprovalExpiry(run, collection)
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
			observeHealthApprovalExpiry(run, collection)
		}
	}
	assert.Equal(t, 2, collection.ApprovalRequests)
	run.UpdatedAt = run.UpdatedAt.Add(time.Second)
	observeHealthApprovalExpiry(run, collection)
	assert.Equal(t, 3, collection.ApprovalRequests, "changed update time invalidates cached evidence")

	collection.ApprovalRequests = healthMaximumApprovalRequests
	run.DatabaseID++
	observeHealthApprovalExpiry(run, collection)
	assert.Equal(t, "unknown", collection.ApprovalExpiryEvidence[run.DatabaseID].Status)
	assert.Equal(t, "approval_annotation_request_limit", collection.ApprovalExpiryEvidence[run.DatabaseID].Reason)
	assert.Equal(t, 3, collection.APIRequests)

	collection = &HealthCollection{Host: "github.com", APIRequests: healthMaximumAPIRequests}
	observeHealthApprovalExpiry(run, collection)
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
	observeHealthApprovalExpiry(run, collection)
	assert.Equal(t, "ignored", collection.ApprovalExpiryEvidence[42].Status)
	assert.Equal(t, "https://github.com/org/repo/actions/runs/42/attempts/1", collection.ApprovalExpiryEvidence[42].URL)

	run.DatabaseID++
	run.Repository = ""
	observeHealthApprovalExpiry(run, collection)
	assert.Equal(t, "unknown", collection.ApprovalExpiryEvidence[43].Status)
	assert.Equal(t, "missing_repository", collection.ApprovalExpiryEvidence[43].Reason)
	assert.Equal(t, 1, collection.ApprovalRequests)
}
