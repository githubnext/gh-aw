package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAWFailureInvestigatorPrefetchUsesRunLevelFailures(t *testing.T) {
	t.Parallel()
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "aw-failure-investigator.md"))
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	text := string(content)
	for _, fragment := range []string{
		`FAILURE_CONCLUSIONS = {"failure", "timed_out", "startup_failure"}`,
		`MAX_DISCOVERY_PAGES = 20`,
		`ERROR_MARKER = re.compile(r"##\[error\]|\b(?:error|panic|exception)\b", re.IGNORECASE)`,
		`def capture_error_window(log_text):`,
		`"capture_likely_missed_fault": not has_error_marker`,
		`Path(".github/workflows").glob("*.lock.yml")`,
		`falling back to workflow path suffix matching`,
		`repos/{REPO}/actions/runs`,
		`"failed_job_names": sorted(set(failed_job_names))`,
		`"agent_job_conclusion": agent_job_conclusion`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected workflow prefetch to contain %q", fragment)
		}
	}
	if strings.Contains(text, `"--log-failed",`) {
		t.Fatal("expected workflow prefetch to use full job logs for error-marker capture")
	}
}
