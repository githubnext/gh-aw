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

	if spec.TrackerLabel != "" && !strings.Contains(data.MarkdownContent, spec.TrackerLabel) {
		t.Fatalf("expected markdown content to mention tracker label %q, got: %q", spec.TrackerLabel, data.MarkdownContent)
	}

	// Campaign orchestrators intentionally omit permissions from the generated markdown.
	// Job permissions are computed during compilation.
	if strings.TrimSpace(data.Permissions) != "" {
		t.Fatalf("expected no permissions in generated orchestrator data, got: %q", data.Permissions)
	}
}

func TestBuildOrchestrator_NoTrackerLabelDoesNotMentionTracker(t *testing.T) {
	spec := &CampaignSpec{
		ID:         "test-campaign",
		Name:       "Test Campaign",
		ProjectURL:  "https://github.com/orgs/test/projects/1",
		Workflows:   []string{"test-workflow"},
		TrackerLabel: "",
	}

	mdPath := ".github/workflows/test-campaign.campaign.md"
	data, _ := BuildOrchestrator(spec, mdPath)

	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	if strings.Contains(data.MarkdownContent, "- Tracker label") {
		t.Fatalf("did not expect tracker label bullet when tracker-label is omitted, got: %q", data.MarkdownContent)
	}

	if strings.Contains(data.Description, "tracker:") {
		t.Fatalf("did not expect default description to include tracker when tracker-label is omitted, got: %q", data.Description)
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
	if !strings.Contains(data.MarkdownContent, "all discovered items") {
		t.Errorf("expected markdown to have explicit completion criteria, got: %q", data.MarkdownContent)
	}
}

func TestBuildOrchestrator_WorkflowsInDiscovery(t *testing.T) {
	spec := &CampaignSpec{
		ID:          "test-campaign",
		Name:        "Test Campaign",
		Description: "A test campaign",
		ProjectURL:  "https://github.com/orgs/test/projects/1",
		Workflows: []string{
			"daily-doc-updater",
			"docs-noob-tester",
			"daily-multi-device-docs-tester",
		},
		TrackerLabel: "campaign:test",
	}

	mdPath := ".github/workflows/test-campaign.campaign.md"
	data, _ := BuildOrchestrator(spec, mdPath)

	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	// Verify that the workflows are explicitly listed in the discovery instructions
	if !strings.Contains(data.MarkdownContent, "Worker workflows:") {
		t.Errorf("expected markdown to list worker workflows, got: %q", data.MarkdownContent)
	}

	// Verify each workflow is mentioned
	for _, workflow := range spec.Workflows {
		if !strings.Contains(data.MarkdownContent, workflow) {
			t.Errorf("expected markdown to mention workflow %q, got: %q", workflow, data.MarkdownContent)
		}
	}

	// Verify the worker discovery step is present
	if !strings.Contains(data.MarkdownContent, "Query worker-created content") {
		t.Errorf("expected markdown to include worker discovery step, got: %q", data.MarkdownContent)
	}

	if !strings.Contains(data.MarkdownContent, "tracker-id:") {
		t.Errorf("expected markdown to mention tracker-id search, got: %q", data.MarkdownContent)
	}

	// Verify that IMPORTANT notice is present
	if !strings.Contains(data.MarkdownContent, "**IMPORTANT**: You MUST perform SEPARATE searches for EACH worker workflow") {
		t.Errorf("expected markdown to have IMPORTANT notice about separate searches, got: %q", data.MarkdownContent)
	}

	// Verify that each workflow has an explicit search query enumerated
	for _, workflow := range spec.Workflows {
		expectedSearchLine := "Search for `" + workflow + "`:"
		if !strings.Contains(data.MarkdownContent, expectedSearchLine) {
			t.Errorf("expected markdown to have explicit search query for %q, got: %q", workflow, data.MarkdownContent)
		}
		expectedTrackerID := "tracker-id: " + workflow
		if !strings.Contains(data.MarkdownContent, expectedTrackerID) {
			t.Errorf("expected markdown to have tracker-id for %q, got: %q", workflow, data.MarkdownContent)
		}
	}

	// Verify that all four search types are mentioned (issues, PRs, discussions, comments)
	if !strings.Contains(data.MarkdownContent, "type:issue") {
		t.Errorf("expected markdown to include issue search type, got: %q", data.MarkdownContent)
	}
	if !strings.Contains(data.MarkdownContent, "type:pr") {
		t.Errorf("expected markdown to include PR search type, got: %q", data.MarkdownContent)
	}
	if !strings.Contains(data.MarkdownContent, "Discussions:") {
		t.Errorf("expected markdown to include discussion search, got: %q", data.MarkdownContent)
	}
	if !strings.Contains(data.MarkdownContent, "in:comments") {
		t.Errorf("expected markdown to include comment search, got: %q", data.MarkdownContent)
	}

	// Verify instructions to combine results
	if !strings.Contains(data.MarkdownContent, "Combine results from all worker searches") {
		t.Errorf("expected markdown to mention combining results from all searches, got: %q", data.MarkdownContent)
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

func TestExtractFileGlobPatterns(t *testing.T) {
	tests := []struct {
		name           string
		spec           *CampaignSpec
		expectedGlobs  []string
		expectedLogMsg string
	}{
		{
			name: "flexible pattern matching both dated and non-dated",
			spec: &CampaignSpec{
				ID:          "go-file-size-reduction-project64",
				MemoryPaths: []string{"memory/campaigns/go-file-size-reduction-project64*/**"},
				MetricsGlob: "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json",
			},
			expectedGlobs:  []string{"go-file-size-reduction-project64*/**"},
			expectedLogMsg: "Extracted file-glob pattern from memory-paths",
		},
		{
			name: "dated pattern in memory-paths",
			spec: &CampaignSpec{
				ID:          "go-file-size-reduction-project64",
				MemoryPaths: []string{"memory/campaigns/go-file-size-reduction-project64-*/**"},
				MetricsGlob: "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json",
			},
			expectedGlobs:  []string{"go-file-size-reduction-project64-*/**"},
			expectedLogMsg: "Extracted file-glob pattern from memory-paths",
		},
		{
			name: "multiple patterns in memory-paths",
			spec: &CampaignSpec{
				ID: "go-file-size-reduction-project64",
				MemoryPaths: []string{
					"memory/campaigns/go-file-size-reduction-project64-*/**",
					"memory/campaigns/go-file-size-reduction-project64/**",
				},
				MetricsGlob: "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json",
			},
			expectedGlobs:  []string{"go-file-size-reduction-project64-*/**", "go-file-size-reduction-project64/**"},
			expectedLogMsg: "Extracted file-glob pattern from memory-paths",
		},
		{
			name: "dated pattern in metrics-glob only",
			spec: &CampaignSpec{
				ID:          "go-file-size-reduction-project64",
				MetricsGlob: "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json",
			},
			expectedGlobs:  []string{"go-file-size-reduction-project64-*/**"},
			expectedLogMsg: "Extracted file-glob pattern from metrics-glob",
		},
		{
			name: "simple pattern without wildcards",
			spec: &CampaignSpec{
				ID:          "simple-campaign",
				MemoryPaths: []string{"memory/campaigns/simple-campaign/**"},
			},
			expectedGlobs:  []string{"simple-campaign/**"},
			expectedLogMsg: "Extracted file-glob pattern from memory-paths",
		},
		{
			name: "no memory paths or metrics glob",
			spec: &CampaignSpec{
				ID: "minimal-campaign",
			},
			expectedGlobs:  []string{"minimal-campaign/**"},
			expectedLogMsg: "Using fallback file-glob pattern",
		},
		{
			name: "multiple memory paths with wildcard",
			spec: &CampaignSpec{
				ID: "multi-path",
				MemoryPaths: []string{
					"memory/campaigns/multi-path-staging/**",
					"memory/campaigns/multi-path-*/data/**",
				},
			},
			expectedGlobs:  []string{"multi-path-staging/**", "multi-path-*/data/**"},
			expectedLogMsg: "Extracted file-glob pattern from memory-paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFileGlobPatterns(tt.spec)
			if len(result) != len(tt.expectedGlobs) {
				t.Errorf("extractFileGlobPatterns(%q) returned %d patterns, want %d", tt.spec.ID, len(result), len(tt.expectedGlobs))
				return
			}
			for i, expected := range tt.expectedGlobs {
				if result[i] != expected {
					t.Errorf("extractFileGlobPatterns(%q)[%d] = %q, want %q", tt.spec.ID, i, result[i], expected)
				}
			}
		})
	}
}

func TestBuildOrchestrator_FileGlobMatchesMemoryPaths(t *testing.T) {
	// This test verifies that the file-glob pattern in repo-memory configuration
	// matches the pattern defined in memory-paths, including wildcards
	spec := &CampaignSpec{
		ID:           "go-file-size-reduction-project64",
		Name:         "Go File Size Reduction Campaign",
		Description:  "Test campaign with dated memory paths",
		ProjectURL:   "https://github.com/orgs/githubnext/projects/64",
		Workflows:    []string{"daily-file-diet"},
		MemoryPaths:  []string{"memory/campaigns/go-file-size-reduction-project64-*/**"},
		MetricsGlob:  "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json",
		TrackerLabel: "campaign:go-file-size-reduction-project64",
	}

	mdPath := ".github/workflows/go-file-size-reduction-project64.campaign.md"
	data, _ := BuildOrchestrator(spec, mdPath)

	if data == nil {
		t.Fatalf("expected non-nil WorkflowData")
	}

	// Extract repo-memory configuration from Tools
	repoMemoryConfig, ok := data.Tools["repo-memory"]
	if !ok {
		t.Fatalf("expected repo-memory to be configured in Tools")
	}

	repoMemoryArray, ok := repoMemoryConfig.([]any)
	if !ok || len(repoMemoryArray) == 0 {
		t.Fatalf("expected repo-memory to be an array with at least one entry")
	}

	repoMemoryEntry, ok := repoMemoryArray[0].(map[string]any)
	if !ok {
		t.Fatalf("expected repo-memory entry to be a map")
	}

	fileGlob, ok := repoMemoryEntry["file-glob"]
	if !ok {
		t.Fatalf("expected file-glob to be present in repo-memory entry")
	}

	fileGlobArray, ok := fileGlob.([]any)
	if !ok || len(fileGlobArray) == 0 {
		t.Fatalf("expected file-glob to be an array with at least one entry")
	}

	fileGlobPattern, ok := fileGlobArray[0].(string)
	if !ok {
		t.Fatalf("expected file-glob pattern to be a string")
	}

	// Verify that the file-glob pattern includes the wildcard for dated directories
	expectedPattern := "go-file-size-reduction-project64-*/**"
	if fileGlobPattern != expectedPattern {
		t.Errorf("file-glob pattern = %q, want %q", fileGlobPattern, expectedPattern)
	}

	// Verify that the pattern would match dated directories
	if !strings.Contains(fileGlobPattern, "*") {
		t.Errorf("file-glob pattern should include wildcard for dated directories, got %q", fileGlobPattern)
	}
}
