package campaign

import (
	"strings"
	"testing"
)

func TestBuildOrchestrator_BasicShape(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "go-file-size-reduction-project64",
		Name:         "Campaign: Go File Size Reduction (Project 64)",
		Description:  "Reduce oversized non-test Go files under pkg/ to ≤800 LOC via tracked refactors.",
		ProjectURL:   "https://github.com/orgs/githubnext/projects/64",
		Workflows:    []string{"daily-file-diet"},
		MemoryPaths:  []string{"memory/campaigns/go-file-size-reduction-project64/**"},
		MetricsGlob:  "memory/campaigns/go-file-size-reduction-project64/metrics/*.json",
		TrackerLabel: "campaign:go-file-size-reduction-project64",
	}

	mdPath := ".github/workflows/go-file-size-reduction-project64.campaign.md"
	data, orchestratorPath := BuildOrchestrator(spec, mdPath)

	if orchestratorPath != ".github/workflows/go-file-size-reduction-project64.campaign.g.md" {
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

	if strings.TrimSpace(data.Concurrency) == "" || !strings.Contains(data.Concurrency, "concurrency:") {
		t.Fatalf("expected workflow-level concurrency to be set, got %q", data.Concurrency)
	}
	if !strings.Contains(data.Concurrency, "campaign-go-file-size-reduction-project64-orchestrator") {
		t.Fatalf("expected concurrency group to include campaign id, got %q", data.Concurrency)
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
	if !strings.Contains(data.MarkdownContent, "Campaign complete") {
		t.Errorf("expected markdown to mention campaign completion, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, "terminal state") {
		t.Errorf("expected markdown to mention terminal state, got: %q", data.MarkdownContent)
	}

	// Verify that the prompt uses system-agnostic completion logic
	if !strings.Contains(data.MarkdownContent, "Decide completion") {
		t.Errorf("expected markdown to include decision phase for completion, got: %q", data.MarkdownContent)
	}

	// Verify explicit completion criteria
	if !strings.Contains(data.MarkdownContent, "all discovered issues are closed") {
		t.Errorf("expected markdown to have explicit completion criteria, got: %q", data.MarkdownContent)
	}
}

func TestBuildOrchestrator_ObjectiveAndKPIsAreRendered(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		Description:  "A test campaign",
		ProjectURL:   "https://github.com/orgs/test/projects/1",
		Workflows:    []string{"daily-file-diet"},
		TrackerLabel: "campaign:test",
		Objective:    "Improve CI stability",
		KPIs: []CampaignKPI{
			{
				Name:           "Build success rate",
				Priority:       "primary",
				Unit:           "ratio",
				Baseline:       0.8,
				Target:         0.95,
				TimeWindowDays: 7,
				Direction:      "increase",
				Source:         "ci",
			},
		},
	}

	mdPath := ".github/workflows/test-campaign.campaign.md"
	data, _ := BuildOrchestrator(spec, mdPath)
	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	// Golden assertions: these should only change if we intentionally change the orchestrator contract.
	expectedPhrases := []string{
		"### Objective and KPIs (first-class)",
		"Objective: Improve CI stability",
		"Build success rate",
		"Deterministic planner step",
	}
	for _, expected := range expectedPhrases {
		if !strings.Contains(data.MarkdownContent, expected) {
			t.Errorf("expected markdown to contain %q, got: %q", expected, data.MarkdownContent)
		}
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
	if !strings.Contains(data.MarkdownContent, "tracker-id") {
		t.Errorf("expected markdown to mention searching for tracker-id, got: %q", data.MarkdownContent)
	}

	// Verify it explains the XML comment correlation mechanism
	if !strings.Contains(data.MarkdownContent, "XML comment") || !strings.Contains(data.MarkdownContent, "Correlation Mechanism") {
		t.Errorf("expected markdown to explain correlation mechanism with XML comments, got: %q", data.MarkdownContent)
	}

	// Verify that orchestrator does NOT monitor workflow runs by file name
	if strings.Contains(data.MarkdownContent, "list_workflow_runs") {
		t.Errorf("expected markdown to NOT use list_workflow_runs for monitoring, but it does: %q", data.MarkdownContent)
	}

	if strings.Contains(data.MarkdownContent, ".lock.yml") {
		t.Errorf("expected markdown to NOT reference .lock.yml files for monitoring, but it does: %q", data.MarkdownContent)
	}

	// Verify that it uses tracker-id based discovery
	if !strings.Contains(data.MarkdownContent, "tracker-id") {
		t.Errorf("expected markdown to use tracker-id for discovering worker output, got: %q", data.MarkdownContent)
	}

	// Verify it follows system-agnostic rules
	if !strings.Contains(data.MarkdownContent, "Campaign Orchestrator Rules") {
		t.Errorf("expected markdown to contain Campaign Orchestrator Rules section, got: %q", data.MarkdownContent)
	}

	// Verify separation of phases
	if !strings.Contains(data.MarkdownContent, "Phase 1: Read State") {
		t.Errorf("expected markdown to contain Phase 1: Read State, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, "Phase 2: Make Decisions") {
		t.Errorf("expected markdown to contain Phase 2: Make Decisions, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, "Phase 3: Write State") {
		t.Errorf("expected markdown to contain Phase 3: Write State, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, "Phase 4: Report") {
		t.Errorf("expected markdown to contain Phase 4: Report, got: %q", data.MarkdownContent)
	}
}

func TestBuildOrchestrator_GitHubToken(t *testing.T) {
	t.Run("with custom github token", func(t *testing.T) {
		spec := &CampaignSpec{
			ID:                 "test-campaign-with-token",
			Name:               "Test Campaign",
			Description:        "A test campaign with custom GitHub token",
			ProjectURL:         "https://github.com/orgs/test/projects/1",
			Workflows:          []string{"test-workflow"},
			TrackerLabel:       "campaign:test",
			ProjectGitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
		}

		mdPath := ".github/workflows/test-campaign.campaign.md"
		data, _ := BuildOrchestrator(spec, mdPath)

		if data == nil {
			t.Fatalf("expected non-nil WorkflowData")
		}

		// Verify that SafeOutputs is configured
		if data.SafeOutputs == nil {
			t.Fatalf("expected SafeOutputs to be configured")
		}

		// Verify that UpdateProjects is configured
		if data.SafeOutputs.UpdateProjects == nil {
			t.Fatalf("expected UpdateProjects to be configured")
		}

		// Verify that the GitHubToken is set
		if data.SafeOutputs.UpdateProjects.GitHubToken != "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}" {
			t.Errorf("expected GitHubToken to be %q, got %q",
				"${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
				data.SafeOutputs.UpdateProjects.GitHubToken)
		}
	})

	t.Run("without custom github token", func(t *testing.T) {
		spec := &CampaignSpec{
			ID:           "test-campaign-no-token",
			Name:         "Test Campaign",
			Description:  "A test campaign without custom GitHub token",
			ProjectURL:   "https://github.com/orgs/test/projects/1",
			Workflows:    []string{"test-workflow"},
			TrackerLabel: "campaign:test",
			// ProjectGitHubToken is intentionally omitted
		}

		mdPath := ".github/workflows/test-campaign.campaign.md"
		data, _ := BuildOrchestrator(spec, mdPath)

		if data == nil {
			t.Fatalf("expected non-nil WorkflowData")
		}

		// Verify that SafeOutputs is configured
		if data.SafeOutputs == nil {
			t.Fatalf("expected SafeOutputs to be configured")
		}

		// Verify that UpdateProjects is configured
		if data.SafeOutputs.UpdateProjects == nil {
			t.Fatalf("expected UpdateProjects to be configured")
		}

		// Verify that the GitHubToken is empty when not specified
		if data.SafeOutputs.UpdateProjects.GitHubToken != "" {
			t.Errorf("expected GitHubToken to be empty when not specified, got %q",
				data.SafeOutputs.UpdateProjects.GitHubToken)
		}
	})
}

func TestBuildOrchestrator_GovernanceOverridesSafeOutputMaxima(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/test/projects/1",
		Workflows:    []string{"test-workflow"},
		TrackerLabel: "campaign:test",
		Governance: &CampaignGovernancePolicy{
			MaxCommentsPerRun:       3,
			MaxProjectUpdatesPerRun: 4,
		},
	}

	mdPath := ".github/workflows/test-campaign.campaign.md"
	data, _ := BuildOrchestrator(spec, mdPath)
	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}
	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil || data.SafeOutputs.UpdateProjects == nil {
		t.Fatalf("expected SafeOutputs add-comment and update-project to be configured")
	}
	if data.SafeOutputs.AddComments.Max != 3 {
		t.Fatalf("unexpected add-comment max: got %d, want %d", data.SafeOutputs.AddComments.Max, 3)
	}
	if data.SafeOutputs.UpdateProjects.Max != 4 {
		t.Fatalf("unexpected update-project max: got %d, want %d", data.SafeOutputs.UpdateProjects.Max, 4)
	}
}
