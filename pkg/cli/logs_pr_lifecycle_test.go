//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lifecycleTestRun() WorkflowRun {
	return WorkflowRun{
		DatabaseID: 42, Event: "pull_request", Status: "completed", Conclusion: "failure",
		Attempt: 1, HeadSha: "abc123", HeadBranch: "feature",
		Repository: "owner/repo", WorkflowName: "Reviewer",
		CreatedAt: time.Date(2026, 10, 18, 20, 0, 0, 0, time.UTC),
		StartedAt: time.Date(2026, 10, 18, 20, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 18, 20, 5, 1, 0, time.UTC),
	}
}

func TestParseJobsEvidenceRequiresAffirmativeEmptyArray(t *testing.T) {
	t.Parallel()
	for _, input := range []string{`[]`, `null`, `[{}]`, `[{"jobs":null}]`, `[{"jobs":"invalid"}]`, `[{"jobs":[{"name":42}]}]`, `[{"jobs":[]},{}]`, `[{"total_count":1,"jobs":[]}]`} {
		jobs, _, err := parseJobsAPIResponses([]byte(input), 42, false)
		require.Error(t, err, input)
		assert.Nil(t, jobs, input)
	}
	jobs, _, err := parseJobsAPIResponses([]byte(`[{"jobs":[],"total_count":0}]`), 42, false)
	require.NoError(t, err)
	assert.NotNil(t, jobs)
	assert.Empty(t, jobs)
}

func TestFetchObsoletePREvidence(t *testing.T) {
	run := lifecycleTestRun()
	for _, tc := range []struct {
		name, state, sha, closed string
		conclusion               string
		want                     bool
	}{
		{"merged while queued", "closed", "abc123", "2026-10-18T20:05:00Z", "failure", true},
		{"cancelled while queued", "closed", "abc123", "2026-10-18T20:05:00Z", "cancelled", true},
		{"still open", "open", "abc123", "", "failure", false},
		{"associated PR head advanced before closure", "closed", "newhead", "2026-10-18T20:05:00Z", "failure", true},
		{"closed trigger", "closed", "abc123", "2026-10-18T19:59:59Z", "failure", false},
		{"closure after failure", "closed", "abc123", "2026-10-18T20:06:00Z", "failure", false},
		{"validation conclusion", "closed", "abc123", "2026-10-18T20:05:00Z", "startup_failure", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			payload := `[[{"number":7,"state":"` + tc.state + `","head":{"sha":"` + tc.sha + `","ref":"feature"}`
			if tc.closed != "" {
				payload += `,"closed_at":"` + tc.closed + `"`
			}
			payload += `}]]`
			require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nprintf '%s' '"+payload+"'\n"), 0o755))
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			current := run
			current.Conclusion = tc.conclusion
			proof, err := fetchObsoletePREvidence(context.Background(), current)
			require.NoError(t, err)
			if tc.want {
				require.NotNil(t, proof)
				assert.Equal(t, 7, proof.Number)
				assert.Equal(t, 1, proof.RunAttempt)
			} else {
				assert.Nil(t, proof)
			}
		})
	}
	t.Run("API failure remains unknown", func(t *testing.T) {
		bin := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\necho forbidden >&2\nexit 1\n"), 0o755))
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		proof, err := fetchObsoletePREvidence(context.Background(), run)
		require.Error(t, err)
		assert.Nil(t, proof)
	})
}

func TestFetchObsoletePREvidenceBranchAssociation(t *testing.T) {
	const closedMatch = `{"number":7,"state":"closed","head":{"sha":"abc123","ref":"feature"},"closed_at":"2026-10-18T20:05:00Z"}`
	const openMatch = `{"number":8,"state":"open","head":{"sha":"abc123","ref":"feature"}}`
	const openOtherBranch = `{"number":9,"state":"open","head":{"sha":"abc123","ref":"backport"}}`
	const closedOtherBranch = `{"number":10,"state":"closed","head":{"sha":"abc123","ref":"backport"},"closed_at":"2026-10-18T20:05:00Z"}`
	const missingBranch = `{"number":11,"state":"open","head":{"sha":"abc123"}}`
	for _, tc := range []struct {
		name, branch, payload string
		want                  bool
	}{
		{"unique branch match with another open PR", "feature", `[[` + closedMatch + `,` + openOtherBranch + `]]`, true},
		{"unique branch match with another closed PR", "feature", `[[` + closedMatch + `,` + closedOtherBranch + `]]`, true},
		{"unique branch match on a later page", "feature", `[[` + openOtherBranch + `],[` + closedMatch + `]]`, true},
		{"multiple same-branch matches remain unknown", "feature", `[[` + closedMatch + `,` + openMatch + `]]`, false},
		{"multiple same-branch matches across pages remain unknown", "feature", `[[` + closedMatch + `],[` + openMatch + `]]`, false},
		{"open matching PR is not invalidated by another closed PR", "feature", `[[` + openMatch + `,` + closedOtherBranch + `]]`, false},
		{"no branch match remains unknown", "feature", `[[` + closedOtherBranch + `]]`, false},
		{"missing run branch requires global uniqueness", "", `[[` + closedMatch + `,` + openOtherBranch + `]]`, false},
		{"missing run branch preserves unique association", "", `[[` + closedMatch + `]]`, true},
		{"missing PR branch cannot disambiguate another association", "feature", `[[` + closedMatch + `,` + missingBranch + `]]`, false},
		{"missing PR head cannot disambiguate another association", "feature", `[[` + closedMatch + `,{"number":12,"state":"open"}]]`, false},
		{"missing PR branch is not an affirmative match", "feature", `[[{"number":7,"state":"closed","head":{"sha":"abc123"},"closed_at":"2026-10-18T20:05:00Z"}]]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nprintf '%s' '"+tc.payload+"'\n"), 0o755))
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			run := lifecycleTestRun()
			run.HeadBranch = tc.branch
			proof, err := fetchObsoletePREvidence(context.Background(), run)
			require.NoError(t, err)
			if tc.want {
				require.NotNil(t, proof)
				assert.Equal(t, 7, proof.Number)
				assert.Equal(t, 1, proof.RunAttempt)
			} else {
				assert.Nil(t, proof)
			}
		})
	}
}

func TestBuildProcessedRunUsesAttemptEvidenceForLifecycle(t *testing.T) {
	oldJobs, oldLifecycle := fetchJobDetailsForProcessedRun, fetchPRLifecycleForProcessedRun
	t.Cleanup(func() {
		fetchJobDetailsForProcessedRun, fetchPRLifecycleForProcessedRun = oldJobs, oldLifecycle
	})
	run := lifecycleTestRun()
	run.Attempt = 2
	lifecycleCalls := 0
	fetchPRLifecycleForProcessedRun = func(_ context.Context, current WorkflowRun) (*ObsoletePREvidence, error) {
		lifecycleCalls++
		assert.Equal(t, 2, current.Attempt)
		return &ObsoletePREvidence{Number: 7, ClosedAt: run.UpdatedAt.Add(-time.Second), RunAttempt: current.Attempt}, nil
	}
	fetchJobDetailsForProcessedRun = func(_ context.Context, _ int64, _ string, _ bool, attempts ...int) ([]JobInfoWithDuration, int, error) {
		assert.Equal(t, []int{2}, attempts)
		return []JobInfoWithDuration{}, 0, nil
	}
	result := DownloadResult{RunAnalysis: RunAnalysis{Run: run}}
	processed := buildProcessedRun(context.Background(), result, false, false)
	require.NotNil(t, processed.Run.ObsoletePR)
	assert.Zero(t, processed.Run.ActionMinutes)
	assert.Equal(t, 1, lifecycleCalls)

	fetchJobDetailsForProcessedRun = func(context.Context, int64, string, bool, ...int) ([]JobInfoWithDuration, int, error) {
		return nil, 0, errors.New("403 forbidden")
	}
	result.JobDetails = []JobInfoWithDuration{}
	result.Run.ObsoletePR = processed.Run.ObsoletePR
	unknown := buildProcessedRun(context.Background(), result, false, false)
	assert.Nil(t, unknown.Run.ObsoletePR)
	assert.Nil(t, unknown.JobDetails)
	assert.Equal(t, 1, lifecycleCalls, "API errors must not be taken as no-job evidence")
}

func TestLifecycleReportAndCacheConsistency(t *testing.T) {
	base := lifecycleTestRun()
	base.ObsoletePR = &ObsoletePREvidence{Number: 7, ClosedAt: base.UpdatedAt.Add(-time.Second), RunAttempt: 1}
	processed := []ProcessedRun{{Run: base, JobDetails: []JobInfoWithDuration{}}}
	for i, conclusion := range []string{"failure", "startup_failure", "success"} {
		run := base
		run.DatabaseID = int64(i + 43)
		run.Conclusion = conclusion
		run.ObsoletePR = nil
		pr := ProcessedRun{Run: run}
		if i == 1 {
			pr.JobDetails = []JobInfoWithDuration{} // Confirmed jobless validation remains visible.
		}
		if i == 2 {
			pr.JobDetails = []JobInfoWithDuration{{JobInfo: JobInfo{Name: "agent", Conclusion: "success"}}}
		}
		processed = append(processed, pr)
	}
	report := buildLogsData(processed, "", nil)
	assert.Equal(t, "failure", report.Runs[0].Conclusion)
	assert.Equal(t, "obsolete_pr", report.Runs[0].ExclusionReason)
	assert.Empty(t, report.Runs[0].FailureClusterIDs)
	assert.Len(t, report.FailureClusters, 2)
	assert.Equal(t, 1, report.Summary.ObsoletePRRuns)
	assert.Equal(t, 2, report.Summary.NonExecutionRuns)
	assert.Equal(t, 4, report.Summary.TotalRuns)
	assert.Equal(t, 1, report.Summary.ExecutedRuns)
	assert.Equal(t, 1, report.Summary.ExecutedSuccessRuns)
	require.NotNil(t, report.FailureInvestigation)
	assert.Equal(t, 2, report.FailureInvestigation.UniqueFailedRuns)
	require.Len(t, report.FailureInvestigation.SelectedRuns, 1)
	assert.Equal(t, int64(43), report.FailureInvestigation.SelectedRuns[0].RunID)
	for _, cluster := range report.FailureClusters {
		assert.NotContains(t, cluster.RunIDs, int64(42))
		assert.Equal(t, 2, cluster.Statistics.FailedPopulation)
	}
	data, err := json.Marshal(report.Runs)
	require.NoError(t, err)
	var cached []RunData
	require.NoError(t, json.Unmarshal(data, &cached))
	for i := range processed {
		processed[i].cachedData = &cached[i]
	}
	again := buildLogsData(processed, "", nil)
	assert.Equal(t, report.FailureClusters, again.FailureClusters)
	assert.Equal(t, report.FailureInvestigation, again.FailureInvestigation)
	assert.Equal(t, report.Summary, again.Summary)
}

func TestLifecycleExclusionDoesNotHideObservedFailures(t *testing.T) {
	t.Parallel()
	base := lifecycleTestRun()
	base.ObsoletePR = &ObsoletePREvidence{Number: 7, ClosedAt: base.UpdatedAt.Add(-time.Second), RunAttempt: 1}
	pr := ProcessedRun{Run: base, JobDetails: []JobInfoWithDuration{}, MCPFailures: []MCPFailureReport{{ServerName: "github"}}}
	report := buildLogsData([]ProcessedRun{pr}, "", nil)
	assert.Empty(t, report.Runs[0].ExclusionReason)
	assert.Len(t, report.FailureClusters, 1)
	assert.Zero(t, report.Summary.ObsoletePRRuns)
}
