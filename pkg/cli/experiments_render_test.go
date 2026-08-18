//go:build !integration

package cli

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureExperimentDetailsStderr(t *testing.T, details *ExperimentDetails) string {
	t.Helper()
	original := os.Stderr
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = writer
	defer func() {
		os.Stderr = original
		_ = writer.Close()
		_ = reader.Close()
	}()

	printExperimentDetails(details)
	require.NoError(t, writer.Close())
	os.Stderr = original

	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(output)
}

func TestPrintExperimentDetailsEmpty(t *testing.T) {
	output := captureExperimentDetailsStderr(t, &ExperimentDetails{
		WorkflowID: "empty-workflow",
		Branch:     "experiments/empty-workflow",
	})

	assert.Contains(t, output, "Experiment workflow: empty-workflow")
	assert.Contains(t, output, "Branch:     experiments/empty-workflow")
	assert.Contains(t, output, "Total runs: 0")
	assert.Contains(t, output, "No experiment data found")
}

func TestPrintExperimentDetailsPopulated(t *testing.T) {
	output := captureExperimentDetailsStderr(t, &ExperimentDetails{
		WorkflowID: "test-workflow",
		Branch:     "experiments/test-workflow",
		TotalRuns:  2,
		Experiments: []ExperimentVariantStats{{
			Name:     "style",
			Variants: map[string]int{"verbose": 1, "concise": 1},
			Total:    2,
		}},
		RecentRuns: []ExperimentRunRecord{{
			RunID:       "123",
			Timestamp:   "2026-08-18T12:00:00Z",
			Assignments: map[string]string{"style": "concise"},
		}},
	})

	assert.Contains(t, output, "style (total: 2)")
	assert.Contains(t, output, "concise")
	assert.Contains(t, output, "verbose")
	assert.Contains(t, output, "50%")
	assert.Contains(t, output, "Recent runs")
	assert.Contains(t, output, "2026-08-18")
	assert.Contains(t, output, "style=concise")
}
