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
		AddComments: &workflow.AddCommentsConfig{},
		UpdateIssues: &workflow.UpdateIssuesConfig{},
		AssignToAgent: &workflow.AssignToAgentConfig{},
		CreateProjects: &workflow.CreateProjectsConfig{
			GitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
		},
		UpdateProjects: &workflow.UpdateProjectConfig{
			GitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
			Views: []workflow.ProjectView{
				{
					Name:   "Campaign Roadmap",
					Layout: "roadmap",
					Filter: "is:issue,is:pull_request",
				},
				{
					Name:   "Task Tracker",
					Layout: "table",
					Filter: "is:issue,is:pull_request",
				},
				{
					Name:   "Progress Board",
					Layout: "board",
					Filter: "is:issue,is:pull_request",
				},
			},
		},
		Messages: &workflow.SafeOutputMessagesConfig{
			Footer:     "> 🎯 *Campaign coordination by [{workflow_name}]({run_url})*",
			RunStarted: "🚀 Campaign Generator starting! [{workflow_name}]({run_url}) is processing your campaign request for this {event_type}...",
			RunSuccess: "✅ Campaign setup complete! [{workflow_name}]({run_url}) has successfully coordinated your campaign creation. Your project is ready! 📊",
			RunFailure: "⚠️ Campaign setup interrupted! [{workflow_name}]({run_url}) {status}. Please check the details and try again...",
		},
	}
}

// buildGeneratorPrompt creates the prompt for the campaign-generator
func buildGeneratorPrompt() string {
	var prompt strings.Builder

	prompt.WriteString("{{#runtime-import? .github/shared-instructions.md}}\n")
	prompt.WriteString("{{#runtime-import? pkg/campaign/prompts/campaign_creation_instructions.md}}\n\n")
	prompt.WriteString("# Campaign Generator\n\n")
	prompt.WriteString("You are a campaign workflow coordinator for GitHub Agentic Workflows. You handle campaign creation and project setup, then assign compilation to the Copilot Coding Agent.\n\n")

	prompt.WriteString("## IMPORTANT: Using Safe Output Tools\n\n")
	prompt.WriteString("When creating or modifying GitHub resources (project, issue, comments), you **MUST use the MCP tool calling mechanism** to invoke the safe output tools.\n\n")
	prompt.WriteString("**Do NOT write markdown code fences or JSON** - you must make actual MCP tool calls using your MCP tool calling capability.\n\n")
	prompt.WriteString("For example:\n")
	prompt.WriteString("- To create a project, invoke the `create_project` MCP tool with the required parameters\n")
	prompt.WriteString("- To update an issue, invoke the `update_issue` MCP tool with the required parameters\n")
	prompt.WriteString("- To add a comment, invoke the `add_comment` MCP tool with the required parameters\n")
	prompt.WriteString("- To assign to an agent, invoke the `assign_to_agent` MCP tool with the required parameters\n\n")
	prompt.WriteString("MCP tool calls write structured data that downstream jobs process. Without proper MCP tool invocations, follow-up actions will be skipped.\n\n")

	prompt.WriteString("## Your Task\n\n")
	prompt.WriteString("**Your Responsibilities:**\n")
	prompt.WriteString("1. Create GitHub Project board\n")
	prompt.WriteString("2. Create custom project fields (Worker/Workflow, Priority, Status, dates, Effort)\n")
	prompt.WriteString("3. Create recommended project views (Roadmap, Task Tracker, Progress Board)\n")
	prompt.WriteString("4. Parse campaign requirements from issue\n")
	prompt.WriteString("5. Discover matching workflows using the workflow catalog (local + agentics collection)\n")
	prompt.WriteString("6. Generate complete `.campaign.md` specification file\n")
	prompt.WriteString("7. Write the campaign file to the repository\n")
	prompt.WriteString("8. Update the issue with campaign details\n")
	prompt.WriteString("9. Assign to Copilot Coding Agent for compilation\n\n")

	prompt.WriteString("**Copilot Coding Agent Responsibilities:**\n")
	prompt.WriteString("1. Compile campaign using `gh aw compile` (requires CLI binary)\n")
	prompt.WriteString("2. Commit all files (spec + generated files)\n")
	prompt.WriteString("3. Create pull request\n\n")

	prompt.WriteString("## Workflow Steps\n\n")
	prompt.WriteString("See the imported campaign creation instructions for detailed step-by-step guidance.\n")

	return prompt.String()
}
