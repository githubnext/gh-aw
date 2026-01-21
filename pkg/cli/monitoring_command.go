package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewMonitoringCommand creates the `gh aw monitoring` command for ProjectOps monitoring workflows
func NewMonitoringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitoring",
		Short: "Generate ProjectOps monitoring workflows with automated project board updates",
		Long: `Generate ProjectOps monitoring workflows for passive tracking and automated project board updates.

Monitoring workflows are event-driven workflows that track existing issues and PRs,
automatically updating project boards without orchestrating new work. This is different
from campaigns which actively orchestrate worker workflows toward goals.

Use monitoring workflows when you want to:
  • Track and update project boards based on existing issues/PRs
  • Monitor progress without dispatching workflows
  • Aggregate metrics from ongoing work
  • Keep project boards synchronized with repository state
  • Dashboard and reporting use cases

Available subcommands:
  • new - Generate a new monitoring workflow

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` monitoring new issue-triage      # Create issue monitoring workflow
  ` + string(constants.CLIExtensionPrefix) + ` monitoring new pr-tracker        # Create PR monitoring workflow
`,
	}

	// Subcommand: monitoring new
	newCmd := &cobra.Command{
		Use:   "new <workflow-id>",
		Short: "Generate a new monitoring workflow with automated project board updates",
		Long: `Generate a new monitoring workflow for ProjectOps with automated project board updates.

The generated workflow will include:
  • Event triggers (issues, pull_request)
  • GitHub MCP server integration with projects toolset
  • Safe outputs for update-project operations
  • AI-powered content analysis for routing and field updates

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` monitoring new issue-triage      # Monitor and triage issues
  ` + string(constants.CLIExtensionPrefix) + ` monitoring new pr-tracker        # Monitor and track PRs
  ` + string(constants.CLIExtensionPrefix) + ` monitoring new sprint-board      # Monitor sprint board progress
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID := args[0]
			return runMonitoringNew(workflowID)
		},
	}

	cmd.AddCommand(newCmd)

	return cmd
}

// runMonitoringNew creates a new monitoring workflow file
func runMonitoringNew(workflowID string) error {
	// Validate workflow ID
	if err := validateWorkflowID(workflowID); err != nil {
		return err
	}

	// Create .github/workflows if it doesn't exist
	workflowsDir := ".github/workflows"
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	// Generate workflow file path
	workflowFile := filepath.Join(workflowsDir, workflowID+".md")

	// Check if file already exists
	if _, err := os.Stat(workflowFile); err == nil {
		return fmt.Errorf("workflow file already exists: %s", workflowFile)
	}

	// Generate workflow content
	content := generateMonitoringWorkflow(workflowID)

	// Write file
	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created monitoring workflow: %s", workflowFile)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Next steps:"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("1. Edit the workflow to customize project URL and routing logic"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("2. Set up GitHub Projects v2 token: gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("3. Compile the workflow: gh aw compile "+workflowID+".md"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("4. Commit and push: git add "+workflowFile+" && git commit && git push"))

	return nil
}

// validateWorkflowID validates the workflow ID format
func validateWorkflowID(id string) error {
	if id == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}

	// Check for valid characters (lowercase letters, digits, hyphens)
	for _, ch := range id {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return fmt.Errorf("workflow ID must use only lowercase letters, digits, and hyphens: got '%s'", id)
		}
	}

	return nil
}

// generateMonitoringWorkflow generates the monitoring workflow content
func generateMonitoringWorkflow(workflowID string) string {
	// Convert workflow ID to human-readable name
	name := strings.ReplaceAll(workflowID, "-", " ")
	name = strings.Title(name)

	return fmt.Sprintf(`---
on:
  issues:
    types: [opened, reopened, labeled]
  pull_request:
    types: [opened, reopened, ready_for_review, labeled]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
safe-outputs:
  update-project:
    max: 10
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
  add-comment:
    max: 1
---

# %s - ProjectOps Monitoring

Automatically track and update project boards based on issue and PR activity.

## Project Configuration

**Project URL**: Replace this with your actual GitHub Project URL:
- User project: https://github.com/users/USERNAME/projects/PROJECT_NUMBER
- Org project: https://github.com/orgs/ORG/projects/PROJECT_NUMBER

## What This Workflow Does

When an issue or pull request is opened, reopened, or labeled, this workflow:

1. **Analyzes content** using AI to determine routing and classification
2. **Updates project boards** automatically based on content analysis
3. **Sets appropriate fields**:
   - Status (e.g., Backlog, To Do, In Progress)
   - Priority (based on severity and impact)
   - Size/Effort estimation
   - Sprint/Milestone assignment
4. **Posts confirmation** comment on the issue/PR

## Routing Logic

Customize the routing logic below based on your project structure:

### Issue Routing

Analyze the issue title and body to determine:
- **Bug reports** → Add to "Bug Triage" project, status: "Needs Triage", priority: based on severity
- **Feature requests** → Add to "Feature Roadmap" project, status: "Proposed"
- **Documentation issues** → Add to "Docs Improvements" project, status: "Todo"
- **Performance issues** → Add to "Performance Optimization" project, priority: "High"

### Pull Request Routing

Analyze the PR title, body, and changed files to determine:
- **Bug fixes** → Add to "Bug Fixes" project, link to related issue if present
- **Features** → Add to "Feature Development" project, status: "In Review"
- **Documentation** → Add to "Documentation" project, status: "In Review"
- **Refactoring** → Add to "Technical Debt" project

## Field Updates

Set project fields based on analysis:
- **Status**: Determine initial status (Backlog, To Do, In Progress, In Review, Done)
- **Priority**: High/Medium/Low based on urgency indicators in title/body
- **Size**: Estimate based on scope (Small/Medium/Large)
- **Labels**: Apply relevant labels for categorization

## After Board Update

After successfully adding the item to the project board, post a brief comment
confirming where it was added and what fields were set. Keep the comment concise
and informative.

Example: "✅ Added to [Project Name] with status 'To Do' and priority 'High'"

---

## Configuration Notes

1. **Token Setup**: This workflow requires a GitHub Projects v2 token stored as 
   GH_AW_PROJECT_GITHUB_TOKEN. See: https://githubnext.github.io/gh-aw/reference/tokens/

2. **Rate Limits**: The workflow is configured to update up to 10 items per run
   via the safe-outputs max setting. Adjust if needed.

3. **Monitoring Only**: This is a monitoring workflow - it tracks existing issues/PRs
   without creating new work or dispatching other workflows. For active orchestration,
   use campaigns instead.
`, name)
}
