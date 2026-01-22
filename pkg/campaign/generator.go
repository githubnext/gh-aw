package campaign

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var generatorLog = logger.New("campaign:generator")

// BuildCampaignGenerator constructs the campaign-generator workflow
// This workflow is triggered when users label issues with "create-agentic-campaign"
// and handles campaign creation, project setup, and assignment to Copilot Coding Agent
func BuildCampaignGenerator() *workflow.WorkflowData {
	generatorLog.Print("Building campaign-generator workflow")

	data := &workflow.WorkflowData{
		Name:            "Agentic Campaign Generator",
		Description:     "Agentic Campaign generator that discovers workflows, generates a campaign spec and a project board, and assigns to Copilot agent for compilation",
		On:              buildGeneratorTrigger(),
		Permissions:     buildGeneratorPermissions(),
		Concurrency:     "",
		RunsOn:          "runs-on: ubuntu-latest",
		Roles:           []string{"admin", "maintainer", "write"},
		EngineConfig:    &workflow.EngineConfig{ID: "claude"},
		Tools:           buildGeneratorTools(),
		SafeOutputs:     buildGeneratorSafeOutputs(),
		MarkdownContent: buildGeneratorPrompt(),
		TimeoutMinutes:  "10",
	}

	return data
}

// buildGeneratorTrigger creates the trigger configuration for campaign-generator
func buildGeneratorTrigger() string {
	return `on:
  issues:
    types: [labeled]
    names: ["create-agentic-campaign"]
    lock-for-agent: true
  workflow_dispatch:
  reaction: "eyes"`
}

// buildGeneratorPermissions creates the permissions configuration
func buildGeneratorPermissions() string {
	return `permissions:
  contents: read
  issues: read
  pull-requests: read`
}

// buildGeneratorTools creates the tools configuration
func buildGeneratorTools() map[string]any {
	return map[string]any{
		"github": map[string]any{
			"toolsets": []any{"default"},
		},
	}
}

// buildGeneratorSafeOutputs creates the safe-outputs configuration
func buildGeneratorSafeOutputs() *workflow.SafeOutputsConfig {
	return &workflow.SafeOutputsConfig{
		UpdateIssues:  &workflow.UpdateIssuesConfig{},
		AssignToAgent: &workflow.AssignToAgentConfig{},
		CreateProjects: &workflow.CreateProjectsConfig{
			GitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
			TargetOwner: "${{ github.repository_owner }}",
			Views: []workflow.ProjectView{
				{
					Name:   "Progress Board",
					Layout: "board",
					Filter: "is:issue is:pr",
				},
				{
					Name:   "Task Tracker",
					Layout: "table",
					Filter: "is:issue is:pr",
				},
				{
					Name:   "Campaign Roadmap",
					Layout: "roadmap",
					Filter: "is:issue is:pr",
				},
			},
		},
		UpdateProjects: &workflow.UpdateProjectConfig{
			GitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
			FieldDefinitions: []workflow.ProjectFieldDefinition{
				{
					Name:     "Campaign Id",
					DataType: "TEXT",
				},
				{
					Name:     "Worker Workflow",
					DataType: "TEXT",
				},
				{
					Name:     "Priority",
					DataType: "SINGLE_SELECT",
					Options:  []string{"High", "Medium", "Low"},
				},
				{
					Name:     "Size",
					DataType: "SINGLE_SELECT",
					Options:  []string{"Small", "Medium", "Large"},
				},
				{
					Name:     "Start Date",
					DataType: "DATE",
				},
				{
					Name:     "End Date",
					DataType: "DATE",
				},
			},
		},
		Messages: &workflow.SafeOutputMessagesConfig{
			AppendOnlyComments: true,
			RunStarted: "### :rocket: Campaign setup started\n\n" +
				"Creating a tracking Project and generating campaign files + orchestrator workflow.\n\n" +
				"No action needed — the [{workflow_name}]({run_url}) will open a pull request and post the link + checklist back on this issue when ready.\n\n" +
				"> To stop this run: remove the label that started it.\n\n" +
				"> **Docs**: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/",
			RunSuccess: "### :white_check_mark: Campaign setup complete\n\n" +
				"Tracking Project created and pull request with generated campaign files is ready.\n\n" +
				"**Next steps**: Review + merge the PR, then run the campaign from the Actions tab.\n\n" +
				"> **Docs**: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/",
			RunFailure: "### :x: Campaign setup {status}\n\n" +
				"**Common causes**:\n\n" +
				"- `GH_AW_PROJECT_GITHUB_TOKEN` is missing or invalid\n\n" +
				"- Token lacks access to GitHub Projects\n\n" +
				"**Action required**:\n\n" +
				"- Fix the first error in the logs\n\n" +
				"- Re-apply the label to re-run\n\n" +
				"> **Troubleshooting**: https://githubnext.github.io/gh-aw/guides/campaigns/flow/#when-something-goes-wrong\n\n" +
				"> **Docs**: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/",
		},
	}
}

// buildGeneratorPrompt creates the prompt for the campaign-generator
func buildGeneratorPrompt() string {
	var prompt strings.Builder

	prompt.WriteString("{{#runtime-import? .github/shared-instructions.md}}\n")
	prompt.WriteString("{{#runtime-import? .github/aw/generate-agentic-campaign.md}}\n")

	return prompt.String()
}
