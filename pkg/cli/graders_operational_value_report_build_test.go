package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOperationalValueReportIncludesEveryPointAndDeduplicatesWeeklyMean(t *testing.T) {
	baseline := 0.25
	evaluator := &operationalValueReportEvaluator{
		WorkflowID: "daily-file-diet", EvaluatorDigest: strings.Repeat("a", 64),
		Definition: operationalValueReportDefinition{
			Repository: "github/gh-aw", WorkflowName: "Daily File Diet", SourcePath: ".github/workflows/daily-file-diet.md",
			OperationalValue: "Improve the assigned file.", Adoption: operationalValueReportAdoption{AdoptedAt: "2026-08-01T00:00:00Z"},
			Baseline: operationalValueReportBaseline{Mode: "baseline-comparable", Value: &baseline},
			Raw:      json.RawMessage(`{"schemaVersion":4,"grader":"operational-value"}`),
		},
	}
	valueOne, valueTwo, valueThree := 0.2, 0.6, 0.8
	observations := []operationalValueReportObservation{
		{Run: operationalValueReportRun{ID: "1", CreatedAt: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)}, Value: &valueOne, Status: "pass", OpportunityKey: "issue:1", Mature: true},
		{Run: operationalValueReportRun{ID: "2", CreatedAt: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)}, Value: &valueTwo, Status: "pass", OpportunityKey: "issue:1", Mature: true},
		{Run: operationalValueReportRun{ID: "3", CreatedAt: time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)}, Value: &valueThree, Status: "pass", OpportunityKey: "issue:2", Mature: true},
	}

	report := buildOperationalValueReport(evaluator, observations, time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC), operationalValueReportBackfillStats{CacheHits: 2, Evaluated: 1})
	require.Len(t, report.Observations, 3)
	require.Len(t, report.Weekly, 1)
	assert.Equal(t, "2026-08-01T00:00:00Z", report.Window.StartAt)
	assert.Equal(t, 2, report.Weekly[0].DistinctOpportunityCount)
	assert.InDelta(t, 0.7, *report.Weekly[0].Mean, 0.000001)
	assert.Equal(t, 1, report.Coverage.DuplicateOpportunityCount)
	assert.InDelta(t, 0.55, *report.Summary.LatestDeltaFromBaseline, 0.000001)
}

func TestRenderOperationalValueReportArtifactsContainRichReport(t *testing.T) {
	value := 0.75
	report := operationalValueReport{
		SchemaVersion: 1, WorkflowID: "example", WorkflowName: "Example <Workflow>", Repository: "owner/repo",
		Window:    operationalValueReportWindow{StartAt: "2026-08-01T00:00:00Z", EndAt: "2026-08-31T00:00:00Z"},
		Evaluator: operationalValueReportEvaluatorReference{SHA256: strings.Repeat("a", 64), Definition: json.RawMessage(`{"evidence":{},"primaryMetric":{}}`)},
		Coverage:  operationalValueReportCoverage{RunCount: 1, NumericCount: 1}, Summary: operationalValueReportSummary{Latest: &value},
		Observations: []operationalValueReportObservation{{Run: operationalValueReportRun{ID: "1", CreatedAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)}, Value: &value}},
	}
	svg := string(renderOperationalValueReportSVG(report))
	assert.Contains(t, svg, "Example &lt;Workflow&gt; operational value")
	assert.Contains(t, svg, "Per-run value")
	markdown := string(renderOperationalValueReportMarkdown(report, "example.json", "example.svg"))
	assert.Contains(t, markdown, "![Example <Workflow> operational value timeline](example.svg)")
	assert.Contains(t, markdown, "Weekly History")
}
