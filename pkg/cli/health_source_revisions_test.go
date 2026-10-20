//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthSamplingStopsAtPushedSourceChanges(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		sourceChanged bool
		freshResult   string
		status        string
		sampled       int
		current       int
		failures      int
	}{
		{"source-only-push", true, "", "no_current_revision_runs", 1, 0, 0},
		{"unrelated-push", false, "", "failures_observed", 10, 5, 5},
		{"new-source-success", true, "success", "insufficient_evidence", 2, 1, 0},
		{"new-source-still-fails", true, "failure", "failures_observed", 2, 1, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			originalRate, originalGH, originalList := fetchRateLimitForHostFunc, healthRunGH, healthListWorkflowRuns
			t.Cleanup(func() {
				fetchRateLimitForHostFunc, healthRunGH, healthListWorkflowRuns = originalRate, originalGH, originalList
			})
			fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
				return rateLimitResource{Limit: 5000, Remaining: 4000}, nil
			}
			oldCommit, freshCommit := strings.Repeat("a", 40), strings.Repeat("b", 40)
			currentSource, oldSource := strings.Repeat("c", 40), strings.Repeat("d", 40)
			if !scenario.sourceChanged {
				oldSource = currentSource
			}
			var batch []WorkflowRun
			start := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
			for i := range 10 {
				run := WorkflowRun{DatabaseID: int64(100 - i), WorkflowName: "Example",
					WorkflowPath: ".github/workflows/example.lock.yml", Repository: "org/repo",
					HeadBranch: "main", HeadSha: oldCommit, Event: "schedule", Attempt: 1,
					Status: "completed", Conclusion: "failure", CreatedAt: start.Add(-time.Duration(i) * time.Hour)}
				if i == 0 && scenario.freshResult != "" {
					run.HeadSha, run.Conclusion = freshCommit, scenario.freshResult
				}
				batch = append(batch, run)
			}
			healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, page int, _ *HealthCollection) ([]WorkflowRun, int, error) {
				require.Equal(t, 1, page, "source boundaries must stop older pagination")
				return batch, opts.Limit, nil
			}
			var sourceRequests, lockRequests []string
			healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				endpoint, err := url.Parse(args[3])
				require.NoError(t, err)
				ref := endpoint.Query().Get("ref")
				if strings.HasSuffix(endpoint.Path, "/example.md") {
					sourceRequests = append(sourceRequests, ref)
					assert.Equal(t, ".sha", args[5])
					if ref == oldCommit {
						return []byte(oldSource), nil
					}
					return []byte(currentSource), nil
				}
				require.True(t, strings.HasSuffix(endpoint.Path, "/example.lock.yml"))
				lockRequests = append(lockRequests, ref)
				return []byte(`# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"settings","body_hash":"unchanged-lock"}`), nil
			}
			collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo"}
			runs, err := sampleHealthWorkflow(GitHubWorkflow{ID: 1, Name: "Example", Path: batch[0].WorkflowPath},
				start.Add(-24*time.Hour).Format(time.RFC3339), "org/repo", false, collection)
			require.NoError(t, err)
			assert.Len(t, runs, scenario.sampled)
			config := HealthConfig{Threshold: 80, RepoOverride: "org/repo", LogsDir: t.TempDir(), collection: collection}
			health := calculateCurrentWorkflowHealth("Example", runs, config)
			assert.Equal(t, scenario.status, health.Status)
			assert.Equal(t, scenario.current, health.TotalRuns)
			assert.Equal(t, scenario.failures, health.FailureCount)
			require.Len(t, health.Revisions, 1)
			assert.Equal(t, currentSource, health.Revisions[0].Current.SourceSHA)
			clusters, plan := buildHealthFailureClusters(runs, config)
			if scenario.failures == 0 {
				assert.Empty(t, clusters)
				assert.Nil(t, plan, "superseded failures must not become investigation targets")
			} else {
				require.NotNil(t, plan)
				assert.Equal(t, scenario.failures, plan.UniqueFailedRuns)
			}
			assert.Equal(t, 1, countHealthSourceRequests(sourceRequests, "main"))
			assert.Equal(t, 1, countHealthSourceRequests(sourceRequests, oldCommit))
			if scenario.sourceChanged {
				assert.NotContains(t, lockRequests, oldCommit, "superseded runs need no lock metadata or further investigation")
				require.NotNil(t, health.Revisions[0].Previous)
				assert.Equal(t, oldSource, health.Revisions[0].Previous.SourceSHA)
			}
			assert.False(t, collection.Partial)
			assert.Empty(t, collection.Warning)
			if scenario.name == "source-only-push" {
				output := captureAuditGroupStdout(t, func() {
					require.NoError(t, displayHealthSummary(runs, HealthConfig{
						Threshold: 80, RepoOverride: "org/repo", JSONOutput: true, LogsDir: config.LogsDir, collection: collection,
					}))
				})
				var report HealthSummary
				require.NoError(t, json.Unmarshal([]byte(output), &report))
				assert.Zero(t, report.HealthyWorkflows)
				assert.Equal(t, 1, report.InsufficientWorkflows)
				assert.Empty(t, report.FailureClusters)
				assert.Nil(t, report.FailureInvestigation)
				output = captureAuditGroupStderr(t, func() {
					displayDetailedHealthStatus(health, 80)
					require.NoError(t, renderHealthRevisions(os.Stderr, health.Revisions))
				})
				assert.Contains(t, output, "awaiting new evidence")
				assert.Contains(t, output, `source_sha="`+currentSource+`"`)
				assert.NotContains(t, output, "below threshold")
			}
		})
	}
}

func countHealthSourceRequests(refs []string, ref string) int {
	count := 0
	for _, candidate := range refs {
		if candidate == ref {
			count++
		}
	}
	return count
}

func TestHealthSourceLookupFailuresDoNotProveAChange(t *testing.T) {
	for _, response := range []string{"", "null", "not-a-sha", "http-error"} {
		t.Run(response, func(t *testing.T) {
			originalRate, originalGH := fetchRateLimitForHostFunc, healthRunGH
			t.Cleanup(func() { fetchRateLimitForHostFunc, healthRunGH = originalRate, originalGH })
			fetchRateLimitForHostFunc = func(context.Context, string) (rateLimitResource, error) {
				return rateLimitResource{Limit: 5000, Remaining: 4000}, nil
			}
			healthRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				if strings.Contains(args[3], ".md?") {
					if response == "http-error" {
						return nil, errors.New("HTTP 404")
					}
					return []byte(response), nil
				}
				return []byte(`# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"settings","body_hash":"prompt"}`), nil
			}
			run := WorkflowRun{DatabaseID: 1, WorkflowPath: ".github/workflows/example.lock.yml", Event: "schedule",
				HeadBranch: "main", HeadSha: strings.Repeat("a", 40), Status: "completed", Conclusion: "failure"}
			collection := &HealthCollection{Host: "github.com", repoOverride: "org/repo"}
			resolveHealthRevision(run, collection)
			assert.False(t, healthSourceChanged(run, collection))
			assert.Contains(t, collection.Warning, "Workflow source revision unavailable")
			health := calculateCurrentWorkflowHealth("Example", []WorkflowRun{run}, HealthConfig{Threshold: 80, collection: collection})
			assert.Equal(t, "revision_unknown", health.Status)
		})
	}
}

func TestHealthSourceBoundaryDoesNotHideOtherScopes(t *testing.T) {
	originalList, originalGH := healthListWorkflowRuns, healthRunGH
	t.Cleanup(func() { healthListWorkflowRuns, healthRunGH = originalList, originalGH })
	healthRunGH = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("cached revisions should not require API requests")
		return nil, errors.New("unexpected request")
	}
	metadata := &workflow.LockMetadata{FrontmatterHash: "settings", BodyHash: "prompt"}
	collection := &HealthCollection{
		revisions:       make(map[string]*workflow.LockMetadata),
		headRevisions:   make(map[string]*workflow.LockMetadata),
		sourceRevisions: make(map[string]string),
	}
	start := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	var batch []WorkflowRun
	for i := range 10 {
		run := WorkflowRun{DatabaseID: int64(100 - i), WorkflowName: "Example",
			WorkflowPath: ".github/workflows/example.lock.yml", Repository: "org/repo",
			HeadBranch: "main", HeadSha: strings.Repeat("a", 40), Event: "schedule", Attempt: 1,
			Status: "completed", Conclusion: "failure", CreatedAt: start.Add(-time.Duration(i) * time.Hour)}
		source := strings.Repeat("d", 40)
		if i > 0 && i <= healthMinimumRuns {
			run.HeadBranch, run.HeadSha = "feature", strings.Repeat("b", 40)
			source = strings.Repeat("c", 40)
		}
		collection.revisions[healthRevisionLookupKey(run)] = metadata
		collection.headRevisions[healthHeadRevisionKey(run)] = metadata
		collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadSha)] = source
		collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadBranch)] = strings.Repeat("c", 40)
		batch = append(batch, run)
	}
	healthListWorkflowRuns = func(opts ListWorkflowRunsOptions, page int, _ *HealthCollection) ([]WorkflowRun, int, error) {
		require.Equal(t, 1, page)
		return batch, opts.Limit, nil
	}
	runs, err := sampleHealthWorkflow(GitHubWorkflow{ID: 1, Path: batch[0].WorkflowPath}, "", "org/repo", false, collection)
	require.NoError(t, err)
	assert.Len(t, runs, healthMinimumRuns+1, "stop old main history without skipping current feature failures")
	config := HealthConfig{Threshold: 80, RepoOverride: "org/repo", LogsDir: t.TempDir(), collection: collection}
	health := calculateCurrentWorkflowHealth("Example", runs, config)
	assert.Equal(t, "failures_observed", health.Status)
	assert.Equal(t, healthMinimumRuns, health.FailureCount)
	require.Len(t, health.Revisions, 2)
	_, plan := buildHealthFailureClusters(runs, config)
	require.NotNil(t, plan)
	assert.Equal(t, healthMinimumRuns, plan.UniqueFailedRuns)
}
