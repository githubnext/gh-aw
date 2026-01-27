package campaign

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectOrchestratorFeatures_ProjectIDTriggersCampaignAndDefaults(t *testing.T) {
	data := &workflow.WorkflowData{
		Name:            "Security Alert Burndown",
		WorkflowID:      "security-alert-burndown",
		MarkdownContent: "# Test",
		ParsedFrontmatter: &workflow.FrontmatterConfig{
			Project: &workflow.ProjectConfig{
				URL: "https://github.com/orgs/githubnext/projects/144",
			},
		},
	}

	err := InjectOrchestratorFeatures(data)
	require.NoError(t, err, "campaign injection should succeed")
	require.NotNil(t, data.ParsedFrontmatter, "ParsedFrontmatter should remain")
	require.NotNil(t, data.ParsedFrontmatter.Project, "Project should remain")

	project := data.ParsedFrontmatter.Project
	assert.Equal(t, "security-alert-burndown", project.ID, "Campaign ID should be inferred and normalized")
	assert.Equal(t, "z_campaign_security-alert-burndown", project.TrackerLabel, "TrackerLabel should be defaulted")
	assert.Equal(t, []string{"memory/campaigns/security-alert-burndown/**"}, project.MemoryPaths, "MemoryPaths should be defaulted")
	assert.Equal(t, "memory/campaigns/security-alert-burndown/metrics/*.json", project.MetricsGlob, "MetricsGlob should be defaulted")
	assert.Equal(t, "memory/campaigns/security-alert-burndown/cursor.json", project.CursorGlob, "CursorGlob should be defaulted")

	assert.Contains(t, data.MarkdownContent, "ORCHESTRATOR INSTRUCTIONS", "Markdown should have orchestrator instructions injected")
}

func TestInjectOrchestratorFeatures_ProjectTrackingOnly_DoesNotInject(t *testing.T) {
	data := &workflow.WorkflowData{
		Name:            "Project Tracking Only",
		FrontmatterName: "project-tracking-only",
		MarkdownContent: "# Test",
		ParsedFrontmatter: &workflow.FrontmatterConfig{
			Project: &workflow.ProjectConfig{
				URL:   "https://github.com/orgs/githubnext/projects/144",
				Scope: []string{"githubnext/gh-aw"},
			},
		},
	}

	err := InjectOrchestratorFeatures(data)
	require.NoError(t, err, "non-campaign project tracking should be a no-op")
	assert.NotContains(t, data.MarkdownContent, "ORCHESTRATOR INSTRUCTIONS", "Should not inject campaign sections")
}

func TestNormalizeCampaignID(t *testing.T) {
	assert.Equal(t, "security-alert-burndown", normalizeCampaignID("Security Alert Burndown"))
	assert.Equal(t, "security-alert-burndown", normalizeCampaignID("security_alert_burndown"))
	assert.Equal(t, "security-alert-burndown", normalizeCampaignID(" security---alert@@burndown "))
}
