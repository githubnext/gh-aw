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
		"Repo Memory",
		"workflow-trends.json",
		"known-issues.json",
		"recommendations.json",
		"anomalies.json",
		"metrics-summary.json",
		"stable IDs",
		"recurrence and persistence counters",
		"cross-referenced across days",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(workflowContentStr, snippet) {
			t.Fatalf("expected workflow source to contain %q", snippet)
		}
	}

	if strings.Contains(strings.ToLower(workflowContentStr), "cache memory") {
		t.Fatalf("expected workflow source to use repo memory terminology")
	}
}
