package campaign

import (
	"strings"
	"testing"
)

func TestBuildOrchestrator_BasicShape(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "go-file-size-reduction",
		Name:         "Campaign: Go File Size Reduction",
		Description:  "Reduce oversized non-test Go files under pkg/ to ≤800 LOC via tracked refactors.",
		ProjectURL:   "https://github.com/orgs/githubnext/projects/1",
		Workflows:    []string{"daily-file-diet"},
		MemoryPaths:  []string{"memory/campaigns/go-file-size-reduction-*/**"},
		MetricsGlob:  "memory/campaigns/go-file-size-reduction-*/metrics/*.json",
		TrackerLabel: "campaign:go-file-size-reduction",
	}

	mdPath := ".github/workflows/go-file-size-reduction.campaign.md"
	data, orchestratorPath := BuildOrchestrator(spec, mdPath)

	if orchestratorPath != ".github/workflows/go-file-size-reduction.campaign.g.md" {
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

	if !strings.Contains(data.MarkdownContent, "Go File Size Reduction") {
		t.Fatalf("expected markdown content to mention campaign name, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, spec.TrackerLabel) {
		t.Fatalf("expected markdown content to mention tracker label %q, got: %q", spec.TrackerLabel, data.MarkdownContent)
	}

	// Verify that proper permissions are set for campaign orchestrators
	if !strings.Contains(data.Permissions, "issues: write") {
		t.Fatalf("expected permissions to include 'issues: write', got: %q", data.Permissions)
	}

	if !strings.Contains(data.Permissions, "organization-projects: write") {
		t.Fatalf("expected permissions to include 'organization-projects: write', got: %q", data.Permissions)
	}

	if !strings.Contains(data.Permissions, "contents: read") {
		t.Fatalf("expected permissions to include 'contents: read', got: %q", data.Permissions)
	}
}
