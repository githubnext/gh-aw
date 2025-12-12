package campaign

import (
	"strings"
	"testing"
)

func TestBuildOrchestrator_BasicShape(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "security-q1-2025",
		Name:         "Security Q1 2025",
		Description:  "Security compliance campaign",
		Workflows:    []string{"security-compliance"},
		MemoryPaths:  []string{"memory/campaigns/security-q1-2025/**"},
		MetricsGlob:  "memory/campaigns/security-q1-2025/*.json",
		TrackerLabel: "campaign:security-q1-2025",
	}

	mdPath := ".github/workflows/security-compliance.campaign.md"
	data, orchestratorPath := BuildOrchestrator(spec, mdPath)

	if orchestratorPath != ".github/workflows/security-compliance.campaign.g.md" {
		t.Fatalf("unexpected orchestrator path: got %q", orchestratorPath)
	}

	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	if data.Name != spec.Name {
		t.Fatalf("unexpected workflow name: got %q, want %q", data.Name, spec.Name)
	}

	if strings.TrimSpace(data.On) == "" || !strings.Contains(data.On, "workflow_dispatch") {
		t.Fatalf("expected On section with workflow_dispatch trigger, got %q", data.On)
	}

	if !strings.Contains(data.On, "schedule:") || !strings.Contains(data.On, "0 18 * * *") {
		t.Fatalf("expected On section with daily schedule cron, got %q", data.On)
	}

	if !strings.Contains(data.MarkdownContent, "Security Q1 2025") {
		t.Fatalf("expected markdown content to mention campaign name, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, spec.TrackerLabel) {
		t.Fatalf("expected markdown content to mention tracker label %q, got: %q", spec.TrackerLabel, data.MarkdownContent)
	}
}
