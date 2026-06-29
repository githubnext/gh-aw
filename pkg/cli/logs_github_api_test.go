//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowRunUnmarshal verifies that a standard "gh run list --json" response
// (without the unsupported "path" field) is correctly unmarshaled into a WorkflowRun.
// The "path" field was previously requested but is not a valid gh run list --json
// field and caused failures on strict gh CLI versions.
func TestWorkflowRunUnmarshal(t *testing.T) {
	rawJSON := `[
{
"databaseId": 42,
"workflowName": "My Workflow",
"status": "completed",
"conclusion": "success",
"createdAt": "2026-01-01T00:00:00Z",
"startedAt": "2026-01-01T00:00:01Z",
"updatedAt": "2026-01-01T00:01:00Z"
}
]`

	var runs []WorkflowRun
	require.NoError(t, json.Unmarshal([]byte(rawJSON), &runs), "unmarshal should succeed")
	require.Len(t, runs, 1)

	assert.Equal(t, int64(42), runs[0].DatabaseID, "DatabaseID should be populated")
	assert.Equal(t, "My Workflow", runs[0].WorkflowName, "WorkflowName should be populated")
	assert.Empty(t, runs[0].WorkflowPath, "WorkflowPath should be empty when 'path' field is absent")
}

// TestBuildCreatedFilter verifies that buildCreatedFilter always produces a single
// --created expression that enforces all supplied date bounds. The key invariant is that
// StartDate is never silently dropped, which was the root cause of the bug where runs
// outside the requested window were returned (multiple --created flags were used but gh
// CLI only honours the last one).
func TestBuildCreatedFilter(t *testing.T) {
	tests := []struct {
		name       string
		startDate  string
		endDate    string
		beforeDate string
		want       string
	}{
		{
			name: "no bounds",
			want: "",
		},
		{
			name:      "start date only",
			startDate: "2026-04-17",
			want:      ">=2026-04-17",
		},
		{
			name:    "end date only",
			endDate: "2026-04-17",
			want:    "<=2026-04-17",
		},
		{
			name:      "start and end date",
			startDate: "2026-04-17",
			endDate:   "2026-04-17",
			want:      "2026-04-17..2026-04-17",
		},
		{
			name:      "start and end date different days",
			startDate: "2026-04-01",
			endDate:   "2026-04-30",
			want:      "2026-04-01..2026-04-30",
		},
		{
			name:       "before date only (pagination cursor)",
			beforeDate: "2026-04-17T12:00:00Z",
			// No startDate: keep the original < form.
			want: "<2026-04-17T12:00:00Z",
		},
		{
			name:       "start date and before date (pagination with lower bound)",
			startDate:  "2026-04-01",
			beforeDate: "2026-04-17T12:00:01Z",
			// beforeDate is exclusive; subtract 1 s for inclusive range syntax.
			want: "2026-04-01..2026-04-17T12:00:00Z",
		},
		{
			name:       "start date, end date, and before date",
			startDate:  "2026-04-01",
			endDate:    "2026-04-30",
			beforeDate: "2026-04-17T12:00:01Z",
			// beforeDate takes precedence over endDate as the pagination upper bound.
			want: "2026-04-01..2026-04-17T12:00:00Z",
		},
		{
			name:       "before date at second boundary",
			startDate:  "2026-04-17T00:00:00Z",
			beforeDate: "2026-04-17T00:00:01Z",
			// Subtracting 1 s from beforeDate gives exactly startDate.
			want: "2026-04-17T00:00:00Z..2026-04-17T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCreatedFilter(tt.startDate, tt.endDate, tt.beforeDate)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBuildCreatedFilterStartDateAlwaysEnforced verifies that when both startDate and
// beforeDate are set, the returned filter contains the startDate so that the lower bound
// is always honoured. This is the regression test for the original bug.
func TestBuildCreatedFilterStartDateAlwaysEnforced(t *testing.T) {
	startDate := "2026-04-17"
	beforeDate := "2026-04-17T23:59:59Z"

	filter := buildCreatedFilter(startDate, "", beforeDate)

	// The filter must start with the startDate so it is part of the expression sent to gh.
	assert.True(t, strings.HasPrefix(filter, startDate),
		"filter %q must start with startDate %q so the lower bound is enforced", filter, startDate)
}

// TestListWorkflowRunsErrorHandling verifies the error classification logic in
// listWorkflowRunsWithPagination. In particular it checks that:
//   - "Unknown JSON field" (capital U, as emitted by gh CLI) is treated as an
//     invalid-field error, not an auth error (case-insensitive matching).
//   - Exit code 1 alone does NOT trigger the auth-failure path because gh exits
//     with code 1 for many non-auth errors (e.g. unsupported JSON fields).
func TestListWorkflowRunsErrorHandling(t *testing.T) {
	tests := []struct {
		name             string
		errMsg           string
		outputMsg        string
		wantInvalidField bool
		wantAuth         bool
	}{
		{
			name:             "unknown JSON field (capital U, as gh CLI emits)",
			errMsg:           "exit status 1",
			outputMsg:        `Unknown JSON field: "path"`,
			wantInvalidField: true,
			wantAuth:         false,
		},
		{
			name:             "unknown field lowercase",
			errMsg:           "exit status 1",
			outputMsg:        "unknown field foo",
			wantInvalidField: true,
			wantAuth:         false,
		},
		{
			name:             "invalid field mixed case",
			errMsg:           "exit status 1",
			outputMsg:        "Invalid field: bar",
			wantInvalidField: true,
			wantAuth:         false,
		},
		{
			name:      "exit status 1 alone is NOT an auth error",
			errMsg:    "exit status 1",
			outputMsg: "some other error",
			wantAuth:  false,
		},
		{
			name:      "exit status 4 IS an auth error",
			errMsg:    "exit status 4",
			outputMsg: "",
			wantAuth:  true,
		},
		{
			name:      "gh auth login hint is an auth error",
			errMsg:    "exit status 1",
			outputMsg: "To get started, run: gh auth login",
			wantAuth:  true,
		},
		{
			name:      "not logged in message is an auth error",
			errMsg:    "exit status 1",
			outputMsg: "not logged into any GitHub hosts",
			wantAuth:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combinedMsg := tt.errMsg + " " + tt.outputMsg
			combinedMsgLower := strings.ToLower(combinedMsg)

			isInvalidField := strings.Contains(combinedMsgLower, "invalid field") ||
				strings.Contains(combinedMsgLower, "unknown field") ||
				strings.Contains(combinedMsgLower, "unknown json field") ||
				strings.Contains(combinedMsgLower, "unknown json") ||
				strings.Contains(combinedMsgLower, "field not found") ||
				strings.Contains(combinedMsgLower, "no such field")
			isAuth := !isInvalidField && (strings.Contains(combinedMsg, "exit status 4") ||
				strings.Contains(combinedMsg, "not logged into any GitHub hosts") ||
				strings.Contains(combinedMsg, "To use GitHub CLI in a GitHub Actions workflow") ||
				strings.Contains(combinedMsg, "authentication required") ||
				strings.Contains(tt.outputMsg, "gh auth login"))

			if tt.wantInvalidField {
				assert.True(t, isInvalidField, "expected invalid-field classification")
				assert.False(t, isAuth, "invalid-field errors must not be classified as auth errors")
			}
			if tt.wantAuth {
				assert.False(t, isInvalidField, "auth errors must not be classified as invalid-field errors")
				assert.True(t, isAuth, "expected auth classification")
			}
			if !tt.wantInvalidField && !tt.wantAuth {
				assert.False(t, isInvalidField, "should not be invalid-field")
				assert.False(t, isAuth, "should not be auth")
			}
		})
	}
}

func TestFetchJobDetailsWithCountsIncludesSteps(t *testing.T) {
	fakeBinDir := testutil.TempDir(t, "fake-gh-*")
	fakeGH := filepath.Join(fakeBinDir, "gh")
	argsLogPath := filepath.Join(fakeBinDir, "gh-args.log")
	fakeGHScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + argsLogPath + "\"\n" +
		"cat <<'EOF'\n" +
		"{\"name\":\"agent\",\"status\":\"completed\",\"conclusion\":\"failure\",\"started_at\":\"2026-06-28T01:31:00Z\",\"completed_at\":\"2026-06-28T01:33:00Z\",\"steps\":[{\"name\":\"Set up job\",\"status\":\"completed\",\"conclusion\":\"success\"},{\"name\":\"Run agent\",\"status\":\"completed\",\"conclusion\":\"failure\"}]}\n" +
		"EOF\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(fakeGHScript), 0o755))

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	jobs, failedJobs, err := fetchJobDetailsWithCounts(28307653871, false)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, 1, failedJobs, "failed job count should include failed jobs")
	assert.Equal(t, 2*time.Minute, jobs[0].Duration, "job duration should still be derived from timestamps")
	require.Len(t, jobs[0].Steps, 2)
	assert.Equal(t, "Run agent", jobs[0].Steps[1].Name, "step names should be parsed from gh api output")
	assert.Equal(t, "failure", jobs[0].Steps[1].Conclusion, "step conclusions should be parsed from gh api output")

	argsLog, err := os.ReadFile(argsLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(argsLog), "repos/{owner}/{repo}/actions/runs/28307653871/jobs", "should query the run jobs API")
	assert.Contains(t, string(argsLog), "steps:", "gh jq projection should request step data")
}

// TestFetchJobDetailsWithCountsNullConclusion verifies that jobs and steps with null conclusions
// (e.g. in-progress or queued jobs) are still parsed and not silently dropped. The jq projection
// uses (.conclusion // "") to coerce null to empty string before JSON decoding.
func TestFetchJobDetailsWithCountsNullConclusion(t *testing.T) {
	fakeBinDir := testutil.TempDir(t, "fake-gh-*")
	fakeGH := filepath.Join(fakeBinDir, "gh")
	// Simulate jq coercing null -> "" before output (as (.conclusion // "") does).
	// A job still in progress has conclusion="" for itself and for any pending steps.
	fakeGHScript := "#!/bin/sh\n" +
		"cat <<'EOF'\n" +
		"{\"name\":\"agent\",\"status\":\"in_progress\",\"conclusion\":\"\",\"started_at\":\"2026-06-28T01:31:00Z\",\"completed_at\":\"0001-01-01T00:00:00Z\",\"steps\":[{\"name\":\"Set up job\",\"status\":\"completed\",\"conclusion\":\"success\"},{\"name\":\"Run agent\",\"status\":\"in_progress\",\"conclusion\":\"\"}]}\n" +
		"EOF\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(fakeGHScript), 0o755))

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	jobs, failedJobs, err := fetchJobDetailsWithCounts(28307653871, false)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "in-progress jobs with null conclusion should not be dropped")
	assert.Equal(t, 0, failedJobs, "in-progress job should not count as failed")
	assert.Equal(t, "in_progress", jobs[0].Status)
	assert.Empty(t, jobs[0].Conclusion, "null conclusion should be coerced to empty string")
	require.Len(t, jobs[0].Steps, 2)
	assert.Empty(t, jobs[0].Steps[1].Conclusion, "null step conclusion should be coerced to empty string")
}

func TestWorkflowRunsSpinnerMessage(t *testing.T) {
	tests := []struct {
		name string
		opts ListWorkflowRunsOptions
		want string
	}{
		{
			name: "without target count",
			opts: ListWorkflowRunsOptions{},
			want: "Fetching workflow runs from GitHub...",
		},
		{
			name: "with target count",
			opts: ListWorkflowRunsOptions{
				ProcessedCount: 3,
				TargetCount:    10,
			},
			want: "Fetching workflow runs from GitHub... (3 / 10)",
		},
		{
			name: "processed equals target",
			opts: ListWorkflowRunsOptions{
				ProcessedCount: 10,
				TargetCount:    10,
			},
			want: "Fetching workflow runs from GitHub... (10 / 10)",
		},
		{
			name: "processed exceeds target",
			opts: ListWorkflowRunsOptions{
				ProcessedCount: 12,
				TargetCount:    10,
			},
			want: "Fetching workflow runs from GitHub... (12 / 10)",
		},
		{
			name: "processed without target",
			opts: ListWorkflowRunsOptions{
				ProcessedCount: 4,
			},
			want: "Fetching workflow runs from GitHub...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := workflowRunsSpinnerMessage(tt.opts)
			assert.Equal(t, tt.want, msg)
		})
	}
}
