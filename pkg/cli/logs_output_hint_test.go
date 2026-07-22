//go:build !integration

package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRenderLogsCompactEmitsSingleHintLine(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	renderLogsCompactToWriter(&buf, LogsData{
		Summary: LogsSummary{TotalRuns: 1},
		Runs: []RunData{{
			RunID:        1,
			WorkflowName: "logs",
			Status:       "completed",
			CreatedAt:    time.Now(),
		}},
		Message: usageOnlyArtifactHintMessage(),
	})
	stdout := buf.String()

	assert.Equal(t, 1, strings.Count(stdout, "[hint] "), "compact output should emit a single hint line")
	assert.Contains(t, stdout, usageOnlyArtifactHintMessage())
	assert.Contains(t, stdout, "use --json for full details")
}

func TestRenderLogsOutputWritesArtifactHintForNonCompactFormats(t *testing.T) {
	processedRuns := []ProcessedRun{{
		Run: WorkflowRun{
			DatabaseID:   1,
			Status:       "completed",
			WorkflowName: "logs",
			CreatedAt:    time.Now(),
		},
	}}

	for _, format := range []string{"console", "tsv", "markdown", "pretty"} {
		t.Run(format, func(t *testing.T) {
			_, stderr := captureOutput(t, func() error {
				return renderLogsOutput(processedRuns, renderLogsOutputOptions{
					outputDir:      t.TempDir(),
					format:         format,
					artifactFilter: []string{"usage"},
				})
			})

			assert.Contains(t, stderr, usageOnlyArtifactHintMessage())
		})
	}
}
