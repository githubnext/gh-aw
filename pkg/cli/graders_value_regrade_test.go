package cli

import (
	"context"
	"strings"
	"testing"
	"time"
)

func historicalValueFixture() (valueGraderManifestEntry, graderArtifactResult, graderArtifactRun) {
	digest := strings.Repeat("a", 64)
	createdAt := "2026-08-23T11:58:00Z"
	manifest := valueGraderManifestEntry{
		ID:        "value",
		Name:      "Operational value",
		Source:    "value",
		Enabled:   true,
		Direction: "higher_is_better",
		Digest:    digest,
		Config:    map[string]any{"window": "7d"},
	}
	result := graderArtifactResult{
		ID: "value",
		Implementation: graderArtifactImplementation{
			ID:      "gh-aw-graders",
			Version: 1,
			Digest:  digest,
		},
		Observation: &graderArtifactObservation{
			Subject: graderArtifactSubject{
				Type:       "workflow-run",
				RunID:      "12345",
				Attempt:    2,
				Repository: "github/gh-aw",
				Workflow:   "Example",
				Ref:        "refs/heads/main",
				SHA:        "0123456789abcdef",
				EventName:  "schedule",
				CreatedAt:  &createdAt,
			},
			EvidenceAt: "2026-08-24T12:00:00Z",
			Case:       map[string]any{"issue": float64(42)},
		},
	}
	return manifest, result, graderArtifactRun{ID: "12345", Attempt: 2}
}

func TestVerifyHistoricalValueIdentity(t *testing.T) {
	manifest, result, run := historicalValueFixture()
	if err := verifyHistoricalValueIdentity("github/gh-aw", manifest.Digest, &manifest, &result, run, run.ID); err != nil {
		t.Fatalf("expected valid identity, got %v", err)
	}

	t.Run("digest mismatch", func(t *testing.T) {
		err := verifyHistoricalValueIdentity("github/gh-aw", strings.Repeat("b", 64), &manifest, &result, run, run.ID)
		if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
			t.Fatalf("expected digest mismatch, got %v", err)
		}
	})

	t.Run("repository mismatch", func(t *testing.T) {
		err := verifyHistoricalValueIdentity("github/other", manifest.Digest, &manifest, &result, run, run.ID)
		if err == nil || !strings.Contains(err.Error(), "repository") {
			t.Fatalf("expected repository mismatch, got %v", err)
		}
	})
}

func TestExecuteHistoricalValueFunction(t *testing.T) {
	manifest, result, _ := historicalValueFixture()
	manifest.Config = nil
	functionContent := `#!/usr/bin/env bash
set -euo pipefail
case ${1:-} in
--definition)
  printf '%s\n' '{"schemaVersion":4,"grader":"value","baseline":{"mode":"baseline-comparable","value":0.25}}'
  ;;
--grade-run)
  request=$(cat)
  [[ "$request" == *'"evidenceAt":"2026-09-01T12:00:00Z"'* ]]
  [[ "$request" == *'"case":{"issue":42}'* ]]
	[[ "$request" == *'"config":{}'* ]]
  printf '%s\n' '{"value":0.75,"opportunityKey":"issue:42","case":{"issue":42},"evidenceCutoff":"2026-08-30T12:00:00Z","maturesAt":"2026-08-30T12:00:00Z","provenance":[{"repository":"github/gh-aw","kind":"issue","ref":"42"}]}'
  ;;
*) exit 1 ;;
esac
`
	evidenceAt, err := parseValueTimestamp("2026-09-01T12:00:00Z", "evidence-at")
	if err != nil {
		t.Fatal(err)
	}
	execution, err := executeHistoricalValueFunction(
		context.Background(), functionContent, manifest, *result.Observation,
		"2026-09-01T12:00:00Z", evidenceAt,
	)
	if err != nil {
		t.Fatalf("executeHistoricalValueFunction() error = %v", err)
	}
	if execution.Value == nil || *execution.Value != 0.75 {
		t.Fatalf("value = %v, want 0.75", execution.Value)
	}
	if execution.DeltaFromBaseline == nil || *execution.DeltaFromBaseline != 0.5 {
		t.Fatalf("delta = %v, want 0.5", execution.DeltaFromBaseline)
	}
	if !execution.Observation.Mature || execution.Observation.Subject.RunID != "12345" {
		t.Fatalf("unexpected replay observation: %+v", execution.Observation)
	}
}

func TestParseValueFunctionOutputRejectsFutureEvidence(t *testing.T) {
	evidenceAt, err := time.Parse(time.RFC3339, "2026-08-24T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	_, err = parseValueFunctionOutput([]byte(`{
  "value": 1,
  "opportunityKey": "issue:42",
  "case": {"issue": 42},
  "evidenceCutoff": "2026-08-25T12:00:00Z",
  "maturesAt": "2026-08-30T12:00:00Z",
  "provenance": [{"repository":"github/gh-aw","kind":"issue","ref":"42"}]
}`), graderArtifactSubject{}, "2026-08-24T12:00:00Z", evidenceAt, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot follow evidenceAt") {
		t.Fatalf("expected future evidence rejection, got %v", err)
	}
}

func TestNewGradersCommand(t *testing.T) {
	command := NewGradersCommand()
	valueCommand, _, err := command.Find([]string{"value"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"evidence-at", "repo", "json"} {
		if valueCommand.Flags().Lookup(name) == nil {
			t.Fatalf("value command missing --%s", name)
		}
	}
}
