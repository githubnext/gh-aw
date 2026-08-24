package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func historicalOperationalValueFixture() (operationalValueGraderManifestEntry, graderArtifactResult, graderArtifactRun) {
	digest := strings.Repeat("a", 64)
	createdAt := "2026-08-23T11:58:00Z"
	manifest := operationalValueGraderManifestEntry{
		ID:        "operational-value",
		Name:      "Operational value",
		Source:    "operational-value",
		Enabled:   true,
		Direction: "higher_is_better",
		Digest:    digest,
		Config:    map[string]any{"window": "7d"},
	}
	result := graderArtifactResult{
		ID: "operational-value",
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

func TestVerifyHistoricalOperationalValueIdentity(t *testing.T) {
	manifest, result, run := historicalOperationalValueFixture()
	if err := verifyHistoricalOperationalValueIdentity("github/gh-aw", manifest.Digest, &manifest, &result, run, run.ID); err != nil {
		t.Fatalf("expected valid identity, got %v", err)
	}

	t.Run("digest mismatch", func(t *testing.T) {
		err := verifyHistoricalOperationalValueIdentity("github/gh-aw", strings.Repeat("b", 64), &manifest, &result, run, run.ID)
		if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
			t.Fatalf("expected digest mismatch, got %v", err)
		}
	})

	t.Run("repository mismatch", func(t *testing.T) {
		err := verifyHistoricalOperationalValueIdentity("github/other", manifest.Digest, &manifest, &result, run, run.ID)
		if err == nil || !strings.Contains(err.Error(), "repository") {
			t.Fatalf("expected repository mismatch, got %v", err)
		}
	})
}

func TestExecuteHistoricalOperationalValueEvaluator(t *testing.T) {
	manifest, result, _ := historicalOperationalValueFixture()
	manifest.Config = nil
	evaluatorContent := `#!/usr/bin/env bash
set -euo pipefail
case ${1:-} in
--definition)
	printf '%s\n' '{"schemaVersion":4,"grader":"operational-value","baseline":{"mode":"baseline-comparable","value":0.25}}'
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
	evidenceAt, err := parseOperationalValueTimestamp("2026-09-01T12:00:00Z", "evidence-at")
	if err != nil {
		t.Fatal(err)
	}
	execution, err := executeHistoricalOperationalValueEvaluator(
		context.Background(), evaluatorContent, manifest, *result.Observation,
		"2026-09-01T12:00:00Z", evidenceAt, "https://github.com",
	)
	if err != nil {
		t.Fatalf("executeHistoricalOperationalValueEvaluator() error = %v", err)
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

func TestVerifyArchivedOperationalValueEvaluatorSource(t *testing.T) {
	repoRoot := t.TempDir()
	evaluatorPath := filepath.Join(repoRoot, ".github", "graders", "example.sh")
	if err := os.MkdirAll(filepath.Dir(evaluatorPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("#!/usr/bin/env bash\nprintf 'trusted\\n'\n")
	if err := os.WriteFile(evaluatorPath, content, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"add", ".github/graders/example.sh"},
		{"commit", "-m", "add evaluator"},
	} {
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v: %s", args, err, output)
		}
	}
	shaOutput, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	manifest := operationalValueGraderManifestEntry{Run: ".github/graders/example.sh"}
	subject := graderArtifactSubject{SHA: strings.TrimSpace(string(shaOutput))}
	if err := verifyArchivedOperationalValueEvaluatorSource(repoRoot, "owner/repo", "owner/repo", string(content), hex.EncodeToString(digest[:]), manifest, subject); err != nil {
		t.Fatalf("expected trusted evaluator, got %v", err)
	}
	if err := verifyArchivedOperationalValueEvaluatorSource(repoRoot, "owner/repo", "other/repo", string(content), hex.EncodeToString(digest[:]), manifest, subject); err == nil || !strings.Contains(err.Error(), "trusted local checkout") {
		t.Fatalf("expected repository trust error, got %v", err)
	}
	if err := verifyArchivedOperationalValueEvaluatorSource(repoRoot, "owner/repo", "owner/repo", string(content)+"# changed\n", hex.EncodeToString(digest[:]), manifest, subject); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected source mismatch, got %v", err)
	}
}

func TestOperationalValueEvaluatorEnvironmentUsesRequestedHost(t *testing.T) {
	env := operationalValueEvaluatorEnvironment([]string{
		"PATH=/usr/bin",
		"GH_TOKEN=token",
		"GH_HOST=stale.example.com",
		"GITHUB_API_URL=https://stale.example.com/api/v3",
	}, "https://ghe.example.com")
	joined := strings.Join(env, "\n")
	for _, expected := range []string{
		"GH_HOST=ghe.example.com",
		"GITHUB_SERVER_URL=https://ghe.example.com",
		"GITHUB_API_URL=https://ghe.example.com/api/v3",
		"GITHUB_GRAPHQL_URL=https://ghe.example.com/api/graphql",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in evaluator environment: %v", expected, env)
		}
	}
}

func TestRenderOperationalValueRegradeArtifactWithNullDelta(t *testing.T) {
	baseline := 0.25
	artifact := operationalValueRegradeArtifact{
		Run: graderArtifactRun{ID: "12345"},
		Results: []operationalValueRegradeResult{{
			BaselineValue: &baseline,
			Observation:   operationalValueRegradeObservation{},
		}},
	}
	oldStdout := os.Stdout
	readOutput, writeOutput, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writeOutput
	t.Cleanup(func() { os.Stdout = oldStdout })
	if err := renderOperationalValueRegradeArtifact(artifact, false); err != nil {
		t.Fatal(err)
	}
	_ = writeOutput.Close()
	output, err := io.ReadAll(readOutput)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "Delta from baseline: null") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestParseOperationalValueEvaluatorOutputRejectsFutureEvidence(t *testing.T) {
	evidenceAt, err := time.Parse(time.RFC3339, "2026-08-24T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	_, err = parseOperationalValueEvaluatorOutput([]byte(`{
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
	operationalValueCommand, _, err := command.Find([]string{"operational-value"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"evidence-at", "repo", "json"} {
		if operationalValueCommand.Flags().Lookup(name) == nil {
			t.Fatalf("operational-value command missing --%s", name)
		}
	}
}
