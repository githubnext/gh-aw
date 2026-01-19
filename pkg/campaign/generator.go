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
		Name:            "Campaign Generator",
		Description:     "Campaign generator that creates project board, discovers workflows, generates campaign spec, and assigns to Copilot agent for compilation",
		On:              buildGeneratorTrigger(),
		Permissions:     buildGeneratorPermissions(),
		Concurrency:     "", // No concurrency control for this workflow
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
		AddComments:   &workflow.AddCommentsConfig{},
		UpdateIssues:  &workflow.UpdateIssuesConfig{},
		AssignToAgent: &workflow.AssignToAgentConfig{},
		CreateProjects: &workflow.CreateProjectsConfig{
			GitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
			TargetOwner: "${{ github.repository_owner }}",
		},
		UpdateProjects: &workflow.UpdateProjectConfig{
			GitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
			Views: []workflow.ProjectView{
				{
					Name:   "Campaign Roadmap",
					Layout: "roadmap",
					Filter: "is:issue is:pr",
				},
				{
					Name:   "Task Tracker",
					Layout: "table",
					Filter: "is:issue is:pr",
				},
				{
					Name:   "Progress Board",
					Layout: "board",
					Filter: "is:issue is:pr",
				},
			},
		},
		Messages: &workflow.SafeOutputMessagesConfig{
			Footer:     "> *Campaign coordination by [{workflow_name}]({run_url})*",
			RunStarted: "Campaign Generator starting! [{workflow_name}]({run_url}) is processing your campaign request for this {event_type}...",
			RunSuccess: "Campaign setup complete! [{workflow_name}]({run_url}) has successfully coordinated your campaign creation. Your project is ready!",
			RunFailure: "Campaign setup interrupted! [{workflow_name}]({run_url}) {status}. Please check the details and try again...",
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
