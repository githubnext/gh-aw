package campaign

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// BuildOrchestrator constructs a minimal agentic workflow representation for a
// given CampaignSpec. The resulting WorkflowData is compiled via the standard
// CompileWorkflowDataWithValidation pipeline, and the orchestratorPath
// determines the emitted .lock.yml name.
func BuildOrchestrator(spec *CampaignSpec, campaignFilePath string) (*workflow.WorkflowData, string) {
	// Derive orchestrator markdown path alongside the campaign spec, using a
	// distinct suffix to avoid colliding with existing workflows. We use
	// a `.campaign.g.md` suffix to make it clear that the file is generated
	// from the corresponding `.campaign.md` spec.
	base := strings.TrimSuffix(campaignFilePath, ".campaign.md")
	orchestratorPath := base + ".campaign.g.md"

	name := spec.Name
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Campaign: %s", spec.ID)
	}

	description := spec.Description
	if strings.TrimSpace(description) == "" {
		description = fmt.Sprintf("Orchestrator workflow for campaign '%s' (tracker: %s)", spec.ID, spec.TrackerLabel)
	}

	// Default triggers: daily schedule plus manual workflow_dispatch.
	onSection := "on:\n  schedule:\n    - cron: \"0 18 * * *\"\n  workflow_dispatch:\n"

	// Simple markdown body giving the agent context about the campaign.
	markdownBuilder := &strings.Builder{}
	markdownBuilder.WriteString("# Campaign Orchestrator\n\n")
	markdownBuilder.WriteString(fmt.Sprintf("This workflow orchestrates the '%s' campaign.\n\n", name))

	// Track whether we have any meaningful campaign details
	hasDetails := false

	if spec.TrackerLabel != "" {
		markdownBuilder.WriteString(fmt.Sprintf("- Tracker label: `%s`\n", spec.TrackerLabel))
		hasDetails = true
	}
	if len(spec.Workflows) > 0 {
		markdownBuilder.WriteString("- Associated workflows: ")
		markdownBuilder.WriteString(strings.Join(spec.Workflows, ", "))
		markdownBuilder.WriteString("\n")
		hasDetails = true
	}
	if len(spec.MemoryPaths) > 0 {
		markdownBuilder.WriteString("- Memory paths: ")
		markdownBuilder.WriteString(strings.Join(spec.MemoryPaths, ", "))
		markdownBuilder.WriteString("\n")
		hasDetails = true
	}
	if spec.MetricsGlob != "" {
		markdownBuilder.WriteString(fmt.Sprintf("- Metrics glob: `%s`\n", spec.MetricsGlob))
		hasDetails = true
	}

	// Return nil if the campaign spec has no meaningful details for the prompt
	if !hasDetails {
		return nil, ""
	}

	markdownBuilder.WriteString("\nEach time this orchestrator runs on its daily schedule (or when manually dispatched), generate a concise status report for this campaign. Summarize current metrics, highlight blockers, and update any tracker issues using the campaign label.\n")
	markdownBuilder.WriteString("\nUse these details to coordinate workers, update metrics, and track progress for this campaign.\n")

	data := &workflow.WorkflowData{
		Name:            name,
		Description:     description,
		MarkdownContent: markdownBuilder.String(),
		On:              onSection,
		// Use a standard Ubuntu runner for the main agent job so the
		// compiled orchestrator always has a valid runs-on value.
		RunsOn: "runs-on: ubuntu-latest",
		// Default roles match the workflow compiler's defaults so that
		// membership checks have a non-empty GH_AW_REQUIRED_ROLES value.
		Roles: []string{"admin", "maintainer", "write"},
	}

	return data, orchestratorPath
}
