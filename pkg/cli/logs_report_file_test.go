//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderLogsOutputReportFileWritesMarkdownToFile verifies that when --report-file
// is set with --format markdown, the markdown report is written to the file and
// stdout receives no markdown report content.
func TestRenderLogsOutputReportFileWritesMarkdownToFile(t *testing.T) {
	tmpDir := t.TempDir()
	reportFile := filepath.Join(tmpDir, "report.md")

	processedRuns := []ProcessedRun{{
		Run: WorkflowRun{
			DatabaseID:   1,
			Status:       "completed",
			WorkflowName: "logs",
			CreatedAt:    time.Now(),
		},
	}}

	stdout, _ := captureOutput(t, func() error {
		return renderLogsOutput(processedRuns, renderLogsOutputOptions{
			outputDir:      tmpDir,
			format:         "markdown",
			reportFile:     reportFile,
			artifactFilter: []string{"usage"},
		})
	})

	assert.NotContains(t, stdout, "# Audit Report", "stdout should not contain the markdown report when --report-file is set")

	content, err := os.ReadFile(reportFile)
	require.NoError(t, err, "report file should be created by renderLogsOutput")
	assert.Contains(t, string(content), "# Audit Report — Cross-Run Analysis", "report file should contain the markdown report header")
}
