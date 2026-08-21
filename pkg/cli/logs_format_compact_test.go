//go:build !integration

package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func compactTestData() LogsData {
	return LogsData{
		Summary: LogsSummary{TotalRuns: 1},
		Runs: []RunData{{
			RunID:        1234567,
			WorkflowName: "Logs",
			WorkflowPath: ".github/workflows/logs.lock.yml",
			EngineID:     "copilot",
			Status:       "completed",
			Conclusion:   "success",
			Duration:     "1m2s",
			TokenUsage:   1200,
			Turns:        4,
			Event:        "push",
			Actor:        "octocat",
			Branch:       "main",
			CreatedAt:    time.Now(),
			WSRF:         "3.90",
		}},
	}
}

func TestRenderLogsCompactRendersRunsTableWithBorders(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	renderLogsCompactToWriter(&buf, compactTestData())
	out := buf.String()

	assert.Contains(t, out, "[runs]")
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "RUNID")
	assert.Contains(t, out, "1234567")
	assert.Contains(t, out, "logs")
	assert.Contains(t, out, "WSRF")
	assert.Contains(t, out, "3.90")
	assert.NotContains(t, out, "\x1b[", "non-TTY output should degrade to plain text")
}

func TestRenderLogsCompactVerboseRendersRunsTableWithBorders(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	renderLogsCompactVerboseToWriter(&buf, compactTestData())
	out := buf.String()

	assert.Contains(t, out, "[runs]")
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "CLASS")
	assert.Contains(t, out, "1234567")
	assert.Contains(t, out, "WSRF")
	assert.Contains(t, out, "3.90")
	assert.NotContains(t, out, "\x1b[", "non-TTY output should degrade to plain text")
}

func TestRenderLogsCompactSkipsSkippedAndCancelledRuns(t *testing.T) {
	t.Parallel()
	data := compactTestData()
	data.Runs = append(data.Runs,
		RunData{RunID: 222, WorkflowName: "skipped-wf", Status: "skipped", CreatedAt: time.Now()},
		RunData{RunID: 333, WorkflowName: "cancelled-wf", Status: "cancelled", CreatedAt: time.Now()},
		RunData{RunID: 444, WorkflowName: "skipped-c", Conclusion: "skipped", CreatedAt: time.Now()},
		RunData{RunID: 555, WorkflowName: "cancelled-c", Conclusion: "cancelled", CreatedAt: time.Now()},
	)

	var buf bytes.Buffer
	renderLogsCompactToWriter(&buf, data)
	out := buf.String()

	assert.NotContains(t, out, "222")
	assert.NotContains(t, out, "333")
	assert.NotContains(t, out, "444")
	assert.NotContains(t, out, "555")
	assert.Equal(t, 1, strings.Count(out, "1234567"))
}
