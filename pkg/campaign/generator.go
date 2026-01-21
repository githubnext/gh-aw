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
					Name:     "status",
					DataType: "SINGLE_SELECT",
					Options:  []string{"Todo", "In Progress", "Review Required", "Blocked", "Done"},
				},
				{
					Name:     "campaign_id",
					DataType: "TEXT",
				},
				{
					Name:     "worker_workflow",
					DataType: "TEXT",
				},
				{
					Name:     "repository",
					DataType: "TEXT",
				},
				{
					Name:     "priority",
					DataType: "SINGLE_SELECT",
					Options:  []string{"High", "Medium", "Low"},
				},
				{
					Name:     "size",
					DataType: "SINGLE_SELECT",
					Options:  []string{"Small", "Medium", "Large"},
				},
				{
					Name:     "start_date",
					DataType: "DATE",
				},
				{
					Name:     "end_date",
					DataType: "DATE",
				},
			},
		},
		Messages: &workflow.SafeOutputMessagesConfig{
			Footer: "> *Campaign coordination by [{workflow_name}]({run_url})*\n" +
				"Docs: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/",
			RunStarted: "[{workflow_name}]({run_url}) started generating your campaign (trigger: {event_type}).\n\n" +
				"**What’s happening**\n" +
				"1. Read requirements from this issue\n" +
				"2. Create a GitHub Project (views + standard fields)\n" +
				"3. Generate a campaign spec + orchestrator workflow\n" +
				"4. Update this issue with a handoff checklist\n\n" +
				"You don’t need to do anything yet. When it finishes, look for a PR link in the issue update.\n\n" +
				"Learn more: https://githubnext.github.io/gh-aw/guides/campaigns/flow/",
			RunSuccess: "[{workflow_name}]({run_url}) finished the initial campaign setup.\n\n" +
				"**Next steps**\n" +
				"1. Review the pull request created by the Copilot Coding Agent\n" +
				"2. Merge it\n" +
				"3. Run the campaign orchestrator from the Actions tab\n\n" +
				"Docs: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/",
			RunFailure: "[{workflow_name}]({run_url}) {status}.\n\n" +
				"**What to do**\n" +
				"- Open the run link above and check the logs for the first error\n" +
				"- Fix the issue (permissions/secret/config), then re-run by re-applying the label\n\n" +
				"Troubleshooting: https://githubnext.github.io/gh-aw/guides/campaigns/flow/#when-something-goes-wrong",
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
