package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationalValueReportCommandFlags(t *testing.T) {
	cmd := newGradersOperationalValueReportCommand()
	assert.Equal(t, "report <workflow>", cmd.Use)
	for _, flag := range []string{"until", "output", "cache-dir", "refresh", "repo", "json"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "expected --%s", flag)
	}
}

func TestRunOperationalValueReportWritesCompleteArtifacts(t *testing.T) {
	originalLoadEvaluator := operationalValueReportLoadEvaluator
	originalListRuns := operationalValueReportListRuns
	originalGradeRun := operationalValueReportGradeRun
	t.Cleanup(func() {
		operationalValueReportLoadEvaluator = originalLoadEvaluator
		operationalValueReportListRuns = originalListRuns
		operationalValueReportGradeRun = originalGradeRun
	})

	evaluatorDigest := strings.Repeat("a", 64)
	definitionJSON := json.RawMessage(`{"schemaVersion":4,"grader":"operational-value","repository":"github/gh-aw","workflowName":"Daily File Diet","sourcePath":".github/workflows/daily-file-diet.md","adoption":{"commit":"abc","adoptedAt":"2026-08-01T00:00:00Z"},"operationalValue":"Improve the assigned file.","evidence":{"opportunity":"file","assignment":"largest","accepted":"Git","repositories":["github/gh-aw"],"collection":"Git API","maturation":"two days","zeroRule":"none","missingRule":"null"},"primaryMetric":{"id":"reduction","formula":"reduction / target","direction":"higher_is_better"},"baseline":{"mode":"attainment-only","value":null,"evidenceCutoff":null,"provenance":[]}}`)
	definition, err := parseOperationalValueReportDefinition(definitionJSON)
	require.NoError(t, err)
	operationalValueReportLoadEvaluator = func(context.Context, string, string) (*operationalValueReportEvaluator, error) {
		return &operationalValueReportEvaluator{
			WorkflowID: "daily-file-diet", EvaluatorRun: ".github/graders/daily-file-diet-operational-value.sh",
			EvaluatorDigest: evaluatorDigest, Definition: definition, GraderDirection: "higher-is-better",
		}, nil
	}
	operationalValueReportListRuns = func(_ context.Context, repository, hostname, workflowFile string, _, _ time.Time) ([]operationalValueReportRun, error) {
		assert.Equal(t, "github/gh-aw", repository)
		assert.Equal(t, "github.com", hostname)
		assert.Equal(t, "daily-file-diet.lock.yml", workflowFile)
		return []operationalValueReportRun{{ID: "42", Attempt: 1, CreatedAt: time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)}}, nil
	}
	operationalValueReportGradeRun = func(_ context.Context, evaluator *operationalValueReportEvaluator, run operationalValueReportRun, _ time.Time, _ string) operationalValueReportObservation {
		value := 0.75
		return operationalValueReportObservation{Run: run, Value: &value, Status: "pass", Mature: true, OpportunityKey: "go-file:pkg/example.go", EvaluatorDigest: evaluator.EvaluatorDigest, Source: "evaluator-replay"}
	}

	outputDir := filepath.Join(t.TempDir(), "report")
	err = RunOperationalValueReport(context.Background(), OperationalValueReportConfig{
		Workflow: "daily-file-diet", RepoOverride: "github/gh-aw", Until: "2026-08-31T00:00:00Z",
		OutputDir: outputDir, CacheDir: filepath.Join(t.TempDir(), "cache"),
	})
	require.NoError(t, err)
	for _, extension := range []string{"json", "svg", "md"} {
		_, err := os.Stat(filepath.Join(outputDir, "daily-file-diet-operational-value."+extension))
		require.NoError(t, err)
	}
	reportData, err := os.ReadFile(filepath.Join(outputDir, "daily-file-diet-operational-value.json"))
	require.NoError(t, err)
	var report operationalValueReport
	require.NoError(t, json.Unmarshal(reportData, &report))
	require.Len(t, report.Observations, 1)
	assert.Equal(t, evaluatorDigest, report.Evaluator.SHA256)
	assert.Equal(t, ".github/graders/daily-file-diet-operational-value.sh", report.Evaluator.Path)
}
