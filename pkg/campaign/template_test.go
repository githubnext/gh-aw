package campaign

import (
	"strings"
	"testing"
)

func TestRenderOrchestratorInstructions(t *testing.T) {
	tests := []struct {
		name             string
		data             CampaignPromptData
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "completion guidance enabled, blockers disabled",
			data: CampaignPromptData{
				ReportBlockers:     false,
				CompletionGuidance: true,
			},
			shouldContain: []string{
				"campaign orchestrator",
				"Summarize campaign status",
				"campaign is complete",
				"normal terminal state",
				"Do not report closed issues as blockers",
			},
			shouldNotContain: []string{
				"highlight blockers",
			},
		},
		{
			name: "blockers enabled, completion guidance disabled",
			data: CampaignPromptData{
				ReportBlockers:     true,
				CompletionGuidance: false,
			},
			shouldContain: []string{
				"campaign orchestrator",
				"Summarize campaign status",
				"highlight blockers",
			},
			shouldNotContain: []string{
				"campaign is complete",
				"Do not report closed issues as blockers",
			},
		},
		{
			name: "both enabled",
			data: CampaignPromptData{
				ReportBlockers:     true,
				CompletionGuidance: true,
			},
			shouldContain: []string{
				"campaign orchestrator",
				"Summarize campaign status",
				"highlight blockers",
				"campaign is complete",
				"Do not report closed issues as blockers",
			},
			shouldNotContain: []string{},
		},
		{
			name: "both disabled",
			data: CampaignPromptData{
				ReportBlockers:     false,
				CompletionGuidance: false,
			},
			shouldContain: []string{
				"campaign orchestrator",
				"Summarize campaign status",
			},
			shouldNotContain: []string{
				"highlight blockers",
				"campaign is complete",
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

			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(result, unexpected) {
					t.Errorf("Expected result NOT to contain %q, but it did. Result: %s", unexpected, result)
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
				"update-project",
				"https://github.com/orgs/test/projects/1",
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
		"coordinate workers",
		"track progress",
	}

	for _, expected := range expectedPhrases {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain %q, but it didn't. Result: %s", expected, result)
		}
	}
}
