package campaign

import (
	"strings"
	"testing"
)

func TestBuildOrchestrator_BasicShape(t *testing.T) {
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
		spec := &CampaignSpec{
			ID:          "go-file-size-reduction-project64",
			Name:        "Campaign: Go File Size Reduction (Project 64)",
			Description: "Reduce oversized non-test Go files under pkg/ to ≤800 LOC via tracked refactors.",
			ProjectURL:  "https://github.com/orgs/githubnext/projects/64",
			Workflows:   []string{"daily-file-diet"},
			MemoryPaths: []string{"memory/campaigns/go-file-size-reduction-project64/**"},
			MetricsGlob: "memory/campaigns/go-file-size-reduction-project64/metrics/*.json",
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

		// Campaign orchestrators intentionally omit permissions from the generated markdown.
		// Job permissions are computed during compilation.
		if strings.TrimSpace(data.Permissions) != "" {
			t.Fatalf("expected no permissions in generated orchestrator data, got: %q", data.Permissions)
		}
	})
}

func TestBuildOrchestrator_CompletionInstructions(t *testing.T) {
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
		spec := &CampaignSpec{
			ID:          "test-campaign",
			Name:        "Test Campaign",
			Description: "A test campaign",
			ProjectURL:  "https://github.com/orgs/test/projects/1",
			Workflows:   []string{"test-workflow"},
		}

		mdPath := ".github/workflows/test-campaign.campaign.md"
		data, _ := BuildOrchestrator(spec, mdPath)

		if data == nil {
			t.Fatalf("expected non-nil WorkflowData")
		}

		// Governed invariant: completion is reported explicitly in Step 4.
		expectedPhrases := []string{
			"### Step 4 — Report",
			"completion state (work items only)",
		}
		for _, expected := range expectedPhrases {
			if !strings.Contains(data.MarkdownContent, expected) {
				t.Errorf("expected markdown to contain %q, got: %q", expected, data.MarkdownContent)
			}
		}
	})
}

func TestBuildOrchestrator_WorkflowsInDiscovery(t *testing.T) {
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
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
		}

		mdPath := ".github/workflows/test-campaign.campaign.md"
		data, _ := BuildOrchestrator(spec, mdPath)

		if data == nil {
			t.Fatalf("expected non-nil WorkflowData")
		}

		// Verify each workflow is mentioned in the header list
		for _, workflow := range spec.Workflows {
			if !strings.Contains(data.MarkdownContent, workflow) {
				t.Errorf("expected markdown to mention workflow %q, got: %q", workflow, data.MarkdownContent)
			}
		}

		// Verify that discovery is now precomputed (not agent-side)
		if !strings.Contains(data.MarkdownContent, "Discovery has been precomputed") {
			t.Errorf("expected markdown to indicate precomputed discovery, got: %q", data.MarkdownContent)
		}
		if !strings.Contains(data.MarkdownContent, "./.gh-aw/campaign.discovery.json") {
			t.Errorf("expected markdown to reference discovery manifest, got: %q", data.MarkdownContent)
		}

		// Verify that discovered results reference normalized items from manifest
		if !strings.Contains(data.MarkdownContent, "Parse discovered items from the manifest") {
			t.Errorf("expected markdown to mention parsing items from manifest, got: %q", data.MarkdownContent)
		}
	})
}

func TestBuildOrchestrator_ObjectiveAndKPIsAreRendered(t *testing.T) {
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
		spec := &CampaignSpec{
			ID:          "test-campaign",
			Name:        "Test Campaign",
			Description: "A test campaign",
			ProjectURL:  "https://github.com/orgs/test/projects/1",
			Workflows:   []string{"daily-file-diet"},
			Objective:   "Improve CI stability",
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
			"- Objective: Improve CI stability",
			"- KPIs:",
			"Build success rate",
		}
		for _, expected := range expectedPhrases {
			if !strings.Contains(data.MarkdownContent, expected) {
				t.Errorf("expected markdown to contain %q, got: %q", expected, data.MarkdownContent)
			}
		}
	})
}

func TestBuildOrchestrator_TrackerIDMonitoring(t *testing.T) {
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
		spec := &CampaignSpec{
			ID:          "test-campaign",
			Name:        "Test Campaign",
			Description: "A test campaign",
			ProjectURL:  "https://github.com/orgs/test/projects/1",
			Workflows:   []string{"daily-file-diet"},
		}

		mdPath := ".github/workflows/test-campaign.campaign.md"
		data, _ := BuildOrchestrator(spec, mdPath)

		if data == nil {
			t.Fatalf("expected non-nil WorkflowData")
		}

		// Verify that the orchestrator uses manifest-based discovery (not agent-side search)
		if !strings.Contains(data.MarkdownContent, "Correlation is explicit (tracker-id AND labels)") {
			t.Errorf("expected markdown to mention tracker-id and labels correlation rule, got: %q", data.MarkdownContent)
		}
		if !strings.Contains(data.MarkdownContent, "Read the precomputed discovery manifest") {
			t.Errorf("expected markdown to include manifest-based discovery instructions, got: %q", data.MarkdownContent)
		}
		if !strings.Contains(data.MarkdownContent, "./.gh-aw/campaign.discovery.json") {
			t.Errorf("expected markdown to reference discovery manifest file, got: %q", data.MarkdownContent)
		}

		// Verify that orchestrator does NOT monitor workflow runs by file name
		if strings.Contains(data.MarkdownContent, "list_workflow_runs") {
			t.Errorf("expected markdown to NOT use list_workflow_runs for monitoring, but it does: %q", data.MarkdownContent)
		}

		if strings.Contains(data.MarkdownContent, ".lock.yml") {
			t.Errorf("expected markdown to NOT reference .lock.yml files for monitoring, but it does: %q", data.MarkdownContent)
		}

		// Verify it follows system-agnostic rules
		if !strings.Contains(data.MarkdownContent, "Core Principles") {
			t.Errorf("expected markdown to contain core principles section, got: %q", data.MarkdownContent)
		}

		// Verify separation of steps (read / decide / write / report)
		if !strings.Contains(data.MarkdownContent, "Step 1") || !strings.Contains(data.MarkdownContent, "Read State") {
			t.Errorf("expected markdown to contain Step 1 Read State, got: %q", data.MarkdownContent)
		}
		if !strings.Contains(data.MarkdownContent, "Step 2") || !strings.Contains(data.MarkdownContent, "Make Decisions") {
			t.Errorf("expected markdown to contain Step 2 Make Decisions, got: %q", data.MarkdownContent)
		}
		if !strings.Contains(data.MarkdownContent, "Step 3") || !strings.Contains(data.MarkdownContent, "Write State") {
			t.Errorf("expected markdown to contain Step 3 Write State, got: %q", data.MarkdownContent)
		}
		if !strings.Contains(data.MarkdownContent, "Step 4") || !strings.Contains(data.MarkdownContent, "Report") {
			t.Errorf("expected markdown to contain Step 4 Report, got: %q", data.MarkdownContent)
		}
	})
}

func TestBuildOrchestrator_GitHubToken(t *testing.T) {
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
		t.Run("with custom github token", func(t *testing.T) {
			spec := &CampaignSpec{
				ID:                 "test-campaign-with-token",
				Name:               "Test Campaign",
				Description:        "A test campaign with custom GitHub token",
				ProjectURL:         "https://github.com/orgs/test/projects/1",
				Workflows:          []string{"test-workflow"},
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
				ID:          "test-campaign-no-token",
				Name:        "Test Campaign",
				Description: "A test campaign without custom GitHub token",
				ProjectURL:  "https://github.com/orgs/test/projects/1",
				Workflows:   []string{"test-workflow"},
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
	})
}

func TestBuildOrchestrator_GovernanceOverridesSafeOutputMaxima(t *testing.T) {
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
		spec := &CampaignSpec{
			ID:         "test-campaign",
			Name:       "Test Campaign",
			ProjectURL: "https://github.com/orgs/test/projects/1",
			Workflows:  []string{"test-workflow"},
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
	})
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
	withTempGitRepoWithInstalledCampaignPrompts(t, func(_ string) {
		// This test verifies that the file-glob pattern in repo-memory configuration
		// matches the pattern defined in memory-paths, including wildcards
		spec := &CampaignSpec{
			ID:          "go-file-size-reduction-project64",
			Name:        "Go File Size Reduction Campaign",
			Description: "Test campaign with dated memory paths",
			ProjectURL:  "https://github.com/orgs/githubnext/projects/64",
			Workflows:   []string{"daily-file-diet"},
			MemoryPaths: []string{"memory/campaigns/go-file-size-reduction-project64-*/**"},
			MetricsGlob: "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json",
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
	})
}
