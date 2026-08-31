package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOperationalValueReportEvaluatorResolvesRelativeWorkflowPath(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	t.Chdir(filepath.Join(workingDirectory, "..", ".."))

	evaluator, err := loadOperationalValueReportEvaluator(context.Background(), "daily-file-diet", "https://github.com")
	require.NoError(t, err)
	assert.Equal(t, ".github/workflows/daily-file-diet.md", evaluator.Definition.SourcePath)
	assert.Equal(t, ".github/graders/daily-file-diet-operational-value.sh", evaluator.EvaluatorRun)
}

func TestParseOperationalValueReportDefinition(t *testing.T) {
	definition, err := parseOperationalValueReportDefinition([]byte(`{
		"schemaVersion":4,
		"grader":"operational-value",
		"repository":"github/gh-aw",
		"workflowName":"Daily File Diet",
		"sourcePath":".github/workflows/daily-file-diet.md",
		"adoption":{"commit":"abc123","adoptedAt":"2025-11-15T13:36:21Z"},
		"operationalValue":"Decompose the assigned oversized file.",
		"evidence":{"opportunity":"An oversized file","accepted":"Git evidence","repositories":["github/gh-aw"]},
		"primaryMetric":{"id":"decomposition","formula":"reduction / target","direction":"higher_is_better"},
		"baseline":{"mode":"baseline-comparable","value":0.25,"evidenceCutoff":"2025-11-15T13:27:11Z","provenance":[]}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "github/gh-aw", definition.Repository)
	require.NotNil(t, definition.Baseline.Value)
	assert.Equal(t, 0.25, *definition.Baseline.Value)
}

func TestParseOperationalValueReportDefinitionRejectsIncompleteContract(t *testing.T) {
	_, err := parseOperationalValueReportDefinition([]byte(`{
		"schemaVersion":4,
		"grader":"operational-value",
		"baseline":{"mode":"attainment-only","value":null}
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository, workflowName, and sourcePath")
}
