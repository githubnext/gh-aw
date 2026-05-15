//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestAuditWorkflowsSourceUsesExpandedRepoMemoryCategories(t *testing.T) {
	workflowContent, err := os.ReadFile("../../.github/workflows/audit-workflows.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	workflowContentStr := string(workflowContent)

	expectedSnippets := []string{
		"**Repo Memory**: Store findings in `/tmp/gh-aw/repo-memory/default/`:",
		"`workflow-trends.json` — rolling per-workflow cost, duration, success, and reliability trends",
		"`known-issues.json` — recurring problems with first-seen, last-seen, recurrence count, affected workflows, and status",
		"`recommendations.json` — accumulated recommendations linked back to audits, workflows, and known issues",
		"`anomalies.json` — unusual runs or cost spikes with a multi-day persistence score and current escalation state",
		"`metrics-summary.json` — aggregate daily metrics used for charts and rollups",
		"increment recurrence and persistence counters when the same problem reappears",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(workflowContentStr, snippet) {
			t.Fatalf("expected workflow source to contain %q", snippet)
		}
	}

	if strings.Contains(workflowContentStr, "**Cache Memory**: Store findings in `/tmp/gh-aw/repo-memory/default/`:") {
		t.Fatalf("expected workflow source to use repo memory terminology")
	}
}
