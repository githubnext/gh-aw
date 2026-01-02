package campaign

import (
	"strings"
	"testing"
)

func TestRenderOrchestratorInstructions(t *testing.T) {
	tests := []struct {
		name          string
		data          CampaignPromptData
		shouldContain []string
	}{
		{
			name: "system-agnostic rules",
			data: CampaignPromptData{},
			shouldContain: []string{
				"Campaign Orchestrator Rules",
				"Traffic and rate limits (required)",
				"Prefer incremental processing",
				"strict pagination budgets",
				"durable cursor/checkpoint",
				"On throttling",
				"Workers are immutable",
				"Workers are campaign-agnostic",
				"Campaign logic is external",
				"Phase 1: Read State",
				"Phase 2: Make Decisions",
				"Phase 3: Write State",
				"Phase 4: Report",
				"Predefined Project Fields",
				"Correlation Mechanism",
				"Idempotent operations",
			},
		},
		{
			name: "explicit state management",
			data: CampaignPromptData{},
			shouldContain: []string{
				"Query worker-created content",
				"Query current project state",
				"Compare and identify gaps",
				"Decide additions",
				"Decide updates",
				"Decide field updates",
				"Execute additions",
				"Execute status updates",
				"Execute field updates",
				"Generate status report",
			},
		},
		{
			name: "separation of concerns",
			data: CampaignPromptData{},
			shouldContain: []string{
				"State reads and state writes are separate operations",
				"Explicit outcomes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderOrchestratorInstructions(tt.data)

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but it didn't. Result: %s", expected, result)
				}
			}
		})
	}
}

func TestRenderProjectUpdateInstructions(t *testing.T) {
	tests := []struct {
		name          string
		data          CampaignPromptData
		shouldContain []string
		shouldBeEmpty bool
	}{
		{
			name: "with project URL",
			data: CampaignPromptData{
				ProjectURL: "https://github.com/orgs/test/projects/1",
			},
			shouldContain: []string{
				"Project Board Integration",
				"update-project",
				"https://github.com/orgs/test/projects/1",
				"Adding New Issues",
				"Updating Existing Items",
				"Idempotency",
				"Write Operation Rules",
			},
			shouldBeEmpty: false,
		},
		{
			name: "with project URL and tracker label",
			data: CampaignPromptData{
				ProjectURL:   "https://github.com/orgs/test/projects/1",
				TrackerLabel: "campaign:my-campaign",
			},
			shouldContain: []string{
				"Project Board Integration",
				"update-project",
				"https://github.com/orgs/test/projects/1",
				"Campaign ID",
				"campaign:my-campaign",
				"campaign_id:",
				"CAMPAIGN_ID",
			},
			shouldBeEmpty: false,
		},
		{
			name: "without project URL",
			data: CampaignPromptData{
				ProjectURL: "",
			},
			shouldContain: []string{},
			shouldBeEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderProjectUpdateInstructions(tt.data)

			if tt.shouldBeEmpty && result != "" {
				t.Errorf("Expected empty result, but got: %s", result)
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but it didn't. Result: %s", expected, result)
				}
			}
		})
	}
}

func TestRenderClosingInstructions(t *testing.T) {
	result := RenderClosingInstructions()

	expectedPhrases := []string{
		"Execute all four phases",
		"Read State",
		"Make Decisions",
		"Write State",
		"Report",
		"Workers are immutable and campaign-agnostic",
		"GitHub Project board is the single source of truth",
	}

	for _, expected := range expectedPhrases {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain %q, but it didn't. Result: %s", expected, result)
		}
	}
}
