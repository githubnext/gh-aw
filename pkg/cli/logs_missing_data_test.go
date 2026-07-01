//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
)

func TestExtractMissingDataFromRun_IncludesExperimentProvenance(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	testRun := WorkflowRun{
		DatabaseID:   12345,
		WorkflowName: "Missing Data Experiment",
	}

	stateJSON := `{
		"counts": {
			"prompt_style": { "verbose": 1, "terse": 0 }
		},
		"runs": [
			{
				"run_id": "12345",
				"timestamp": "2026-07-01T00:00:00Z",
				"assignments": { "prompt_style": "verbose" }
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "state.json"), []byte(stateJSON), 0o644); err != nil {
		t.Fatalf("Failed to create state.json: %v", err)
	}

	safeOutput := `{
		"items": [
			{
				"type": "missing_data",
				"data_type": "issue_body",
				"reason": "Issue body was empty",
				"context": "triage workflow",
				"timestamp": "2026-07-01T01:00:00Z"
			}
		],
		"errors": []
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, constants.AgentOutputArtifactName), []byte(safeOutput), 0o644); err != nil {
		t.Fatalf("Failed to create safe output file: %v", err)
	}

	expName, expVariant, _ := firstExperimentAssignment(extractExperimentData(tmpDir))
	reports, err := extractMissingDataFromRun(tmpDir, testRun, false, expName, expVariant)
	if err != nil {
		t.Fatalf("Error extracting missing data: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("Expected 1 missing data report, got %d", len(reports))
	}
	report := reports[0]
	if report.DataType != "issue_body" {
		t.Errorf("Expected data_type issue_body, got %q", report.DataType)
	}
	if report.ExperimentName != "prompt_style" {
		t.Fatalf("Expected experiment_name prompt_style, got %q", report.ExperimentName)
	}
	if report.Variant != "verbose" {
		t.Fatalf("Expected variant verbose, got %q", report.Variant)
	}
}

func TestExtractMissingDataFromRun_NoExperimentProvenance(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	testRun := WorkflowRun{
		DatabaseID:   99999,
		WorkflowName: "No Experiment Workflow",
	}

	safeOutput := `{
		"items": [
			{
				"type": "missing_data",
				"data_type": "pr_diff",
				"reason": "Diff too large",
				"timestamp": "2026-07-01T02:00:00Z"
			}
		],
		"errors": []
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, constants.AgentOutputArtifactName), []byte(safeOutput), 0o644); err != nil {
		t.Fatalf("Failed to create safe output file: %v", err)
	}

	reports, err := extractMissingDataFromRun(tmpDir, testRun, false, "", "")
	if err != nil {
		t.Fatalf("Error extracting missing data: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("Expected 1 missing data report, got %d", len(reports))
	}
	report := reports[0]
	if report.ExperimentName != "" {
		t.Errorf("Expected empty experiment_name, got %q", report.ExperimentName)
	}
	if report.Variant != "" {
		t.Errorf("Expected empty variant, got %q", report.Variant)
	}
}
