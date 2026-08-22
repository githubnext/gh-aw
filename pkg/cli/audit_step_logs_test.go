package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeStepLog(t *testing.T, logsPath, jobDir, filename, content string) {
	t.Helper()
	dir := filepath.Join(logsPath, "workflow-logs", jobDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644))
}

func TestBuildAuditJobsAttachesStepErrorExcerpt(t *testing.T) {
	t.Parallel()
	logsPath := t.TempDir()
	writeStepLog(t, logsPath, "safe_outputs", "5_Process Safe Outputs.txt",
		"2024-01-01T10:00:00.1234567Z starting handlers\n2024-01-01T10:00:01.1234567Z ##[error]create_issue handler failed: Resource not accessible by integration\n")

	jobDetails := []JobInfoWithDuration{
		{
			JobInfo: JobInfo{
				Name:       "safe_outputs",
				Status:     "completed",
				Conclusion: "failure",
				Steps: []JobStep{
					{Name: "Checkout", Conclusion: "success"},
					{Name: "Process Safe Outputs", Conclusion: "failure"},
				},
			},
		},
	}

	jobs := buildAuditJobs(jobDetails, logsPath)
	require.Len(t, jobs, 1)
	require.Len(t, jobs[0].Steps, 2)

	assert.Empty(t, jobs[0].Steps[0].ErrorExcerpt, "successful steps must not carry an excerpt")
	assert.Contains(t, jobs[0].Steps[1].ErrorExcerpt, "create_issue handler failed")
	assert.NotContains(t, jobs[0].Steps[1].ErrorExcerpt, "2024-01-01T10:00:01", "timestamps must be stripped")

	// The source job details must not be mutated.
	assert.Empty(t, jobDetails[0].Steps[1].ErrorExcerpt)
}

func TestBuildAuditJobsWithoutWorkflowLogs(t *testing.T) {
	t.Parallel()
	jobDetails := []JobInfoWithDuration{
		{
			JobInfo: JobInfo{
				Name:       "safe_outputs",
				Conclusion: "failure",
				Steps:      []JobStep{{Name: "Process Safe Outputs", Conclusion: "failure"}},
			},
		},
	}

	jobs := buildAuditJobs(jobDetails, t.TempDir())
	require.Len(t, jobs, 1)
	assert.Empty(t, jobs[0].Steps[0].ErrorExcerpt)

	jobs = buildAuditJobs(jobDetails, "")
	require.Len(t, jobs, 1)
	assert.Empty(t, jobs[0].Steps[0].ErrorExcerpt)
}

func TestExtractStepFailureExcerptFallsBackToTail(t *testing.T) {
	t.Parallel()
	logsPath := t.TempDir()
	writeStepLog(t, logsPath, "safe_outputs", "3_Process Safe Outputs.txt",
		"2024-01-01T10:00:00.1234567Z first line\n2024-01-01T10:00:01.1234567Z TypeError: undefined is not a function\n")

	path := filepath.Join(logsPath, "workflow-logs", "safe_outputs", "3_Process Safe Outputs.txt")
	excerpt := extractStepFailureExcerpt(path)
	assert.Contains(t, excerpt, "TypeError: undefined is not a function")
	assert.Contains(t, excerpt, "first line")
}

func TestExtractStepFailureExcerptTruncatesLongTail(t *testing.T) {
	t.Parallel()
	logsPath := t.TempDir()
	writeStepLog(t, logsPath, "safe_outputs", "1_Process Safe Outputs.txt",
		strings.Repeat("a", maxStepErrorExcerptLen*2)+"TAIL")

	path := filepath.Join(logsPath, "workflow-logs", "safe_outputs", "1_Process Safe Outputs.txt")
	excerpt := extractStepFailureExcerpt(path)
	assert.LessOrEqual(t, len(excerpt), maxStepErrorExcerptLen+3)
	assert.True(t, strings.HasSuffix(excerpt, "TAIL"), "the end of the log must be preserved")
}

func TestNormalizeLogName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "process-safe-outputs", normalizeLogName("Process Safe Outputs"))
	assert.Equal(t, "process-safe-outputs", normalizeLogName("process_safe_outputs"))
	assert.Equal(t, "safe-outputs", normalizeLogName("  safe outputs  "))
}
