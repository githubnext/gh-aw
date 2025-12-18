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

	// Campaign orchestrators intentionally omit permissions from the generated markdown.
	// Job permissions are computed during compilation.
	if strings.TrimSpace(data.Permissions) != "" {
		t.Fatalf("expected no permissions in generated orchestrator data, got: %q", data.Permissions)
	}
}

func TestBuildOrchestrator_CompletionInstructions(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		Description:  "A test campaign",
		ProjectURL:   "https://github.com/orgs/test/projects/1",
		Workflows:    []string{"test-workflow"},
		TrackerLabel: "campaign:test",
	}

	mdPath := ".github/workflows/test-campaign.campaign.md"
	data, _ := BuildOrchestrator(spec, mdPath)

	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	// Verify that the prompt includes completion instructions
	if !strings.Contains(data.MarkdownContent, "campaign is complete") {
		t.Errorf("expected markdown to mention campaign completion, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, "normal terminal state") {
		t.Errorf("expected markdown to mention normal terminal state, got: %q", data.MarkdownContent)
	}

	// Verify that the prompt explicitly states not to report closed issues as blockers
	if !strings.Contains(data.MarkdownContent, "Do not report closed issues as blockers") {
		t.Errorf("expected markdown to explicitly state not to report closed issues as blockers, got: %q", data.MarkdownContent)
	}

	// Verify that "highlight blockers" is not in the prompt
	if strings.Contains(data.MarkdownContent, "highlight blockers") {
		t.Errorf("expected markdown to NOT contain 'highlight blockers', but it does: %q", data.MarkdownContent)
	}
}

func TestBuildOrchestrator_TrackerIDMonitoring(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		Description:  "A test campaign",
		ProjectURL:   "https://github.com/orgs/test/projects/1",
		Workflows:    []string{"daily-file-diet"},
		TrackerLabel: "campaign:test",
	}

	mdPath := ".github/workflows/test-campaign.campaign.md"
	data, _ := BuildOrchestrator(spec, mdPath)

	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	// Verify that the orchestrator uses tracker-id for monitoring
	if !strings.Contains(data.MarkdownContent, "tracker-id") {
		t.Errorf("expected markdown to mention tracker-id for worker monitoring, got: %q", data.MarkdownContent)
	}

	// Verify that it searches for issues containing tracker-id
	if !strings.Contains(data.MarkdownContent, "<!-- tracker-id:") {
		t.Errorf("expected markdown to mention searching for tracker-id HTML comments, got: %q", data.MarkdownContent)
	}

	// Verify that orchestrator does NOT monitor workflow runs by file name
	if strings.Contains(data.MarkdownContent, "list_workflow_runs") {
		t.Errorf("expected markdown to NOT use list_workflow_runs for monitoring, but it does: %q", data.MarkdownContent)
	}

	if strings.Contains(data.MarkdownContent, ".lock.yml") {
		t.Errorf("expected markdown to NOT reference .lock.yml files for monitoring, but it does: %q", data.MarkdownContent)
	}

	// Verify that it uses github-search_issues
	if !strings.Contains(data.MarkdownContent, "github-search_issues") {
		t.Errorf("expected markdown to use github-search_issues for discovering worker output, got: %q", data.MarkdownContent)
	}
}
