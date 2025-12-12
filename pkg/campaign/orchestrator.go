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

	// Minimal on: section - workflow_dispatch trigger with no inputs.
	onSection := "on:\n  workflow_dispatch:\n"

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

	markdownBuilder.WriteString("\nUse these details to coordinate workers, update metrics, and track progress for this campaign.\n")

	data := &workflow.WorkflowData{
		Name:            name,
		Description:     description,
		MarkdownContent: markdownBuilder.String(),
		On:              onSection,
	}

	return data, orchestratorPath
}
