package campaign

import (
	"strings"
	"testing"
)

func TestBuildLauncher_BasicShape(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "go-file-size-reduction",
		Name:         "Campaign: Go File Size Reduction",
		Description:  "Reduce oversized non-test Go files under pkg/ to <= 800 LOC via tracked refactors.",
		ProjectURL:   "https://github.com/orgs/githubnext/projects/1",
		TrackerLabel: "campaign:go-file-size-reduction",
	}

	mdPath := ".github/workflows/go-file-size-reduction.campaign.md"
	data, launcherPath := BuildLauncher(spec, mdPath)

	if launcherPath != ".github/workflows/go-file-size-reduction.campaign.launcher.g.md" {
		t.Fatalf("unexpected launcher path: got %q", launcherPath)
	}
	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	if strings.TrimSpace(data.On) == "" || !strings.Contains(data.On, "workflow_dispatch") {
		t.Fatalf("expected On section with workflow_dispatch trigger, got %q", data.On)
	}
	if !strings.Contains(data.On, "schedule:") || !strings.Contains(data.On, "0 17 * * *") {
		t.Fatalf("expected On section with daily schedule cron, got %q", data.On)
	}

	if strings.TrimSpace(data.Concurrency) == "" || !strings.Contains(data.Concurrency, "concurrency:") {
		t.Fatalf("expected workflow-level concurrency to be set, got %q", data.Concurrency)
	}
	if !strings.Contains(data.Concurrency, "campaign-go-file-size-reduction-launcher") {
		t.Fatalf("expected concurrency group to include campaign id, got %q", data.Concurrency)
	}

	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil || data.SafeOutputs.UpdateProjects == nil {
		t.Fatalf("expected SafeOutputs add-comment and update-project to be configured")
	}
	if data.SafeOutputs.AddComments.Max <= 0 {
		t.Fatalf("expected add-comment max to be > 0")
	}
	if data.SafeOutputs.UpdateProjects.Max <= 0 {
		t.Fatalf("expected update-project max to be > 0")
	}

	if !strings.Contains(data.MarkdownContent, "Traffic and rate limits (required)") {
		t.Fatalf("expected launcher instructions to include baked-in traffic guidance")
	}
}

func TestBuildLauncher_GovernanceOverridesSafeOutputMaxima(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/test/projects/1",
		TrackerLabel: "campaign:test",
		Governance: &CampaignGovernancePolicy{
			MaxCommentsPerRun:       2,
			MaxProjectUpdatesPerRun: 3,
		},
	}

	mdPath := ".github/workflows/test-campaign.campaign.md"
	data, _ := BuildLauncher(spec, mdPath)
	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}
	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil || data.SafeOutputs.UpdateProjects == nil {
		t.Fatalf("expected SafeOutputs add-comment and update-project to be configured")
	}
	if data.SafeOutputs.AddComments.Max != 2 {
		t.Fatalf("unexpected add-comment max: got %d, want %d", data.SafeOutputs.AddComments.Max, 2)
	}
	if data.SafeOutputs.UpdateProjects.Max != 3 {
		t.Fatalf("unexpected update-project max: got %d, want %d", data.SafeOutputs.UpdateProjects.Max, 3)
	}
}
