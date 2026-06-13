package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAWFailureInvestigatorPrefetchUsesRunLevelFailures(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "aw-failure-investigator.md"))
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	text := string(content)
	for _, fragment := range []string{
		`FAILURE_CONCLUSIONS = {"failure", "timed_out", "startup_failure", "cancelled"}`,
		`MAX_DISCOVERY_PAGES = 20`,
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
}
