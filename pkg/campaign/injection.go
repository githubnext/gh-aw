package campaign

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var injectionLog = logger.New("campaign:injection")

var campaignIDSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

func normalizeCampaignID(id string) string {
	// Keep IDs stable and safe for labels/paths.
	id = strings.ToLower(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, "_", "-")
	id = strings.ReplaceAll(id, " ", "-")
	id = campaignIDSanitizer.ReplaceAllString(id, "-")
	// Collapse multiple hyphens into single hyphen
	for strings.Contains(id, "--") {
		id = strings.ReplaceAll(id, "--", "-")
	}
	id = strings.Trim(id, "-")
	return id
}

// InjectOrchestratorFeatures detects if a workflow has project field with campaign
// configuration and injects orchestrator features directly into the workflow during compilation.
// This transforms the workflow into a campaign orchestrator without generating separate files.
func InjectOrchestratorFeatures(workflowData *workflow.WorkflowData) error {
	injectionLog.Print("Checking workflow for campaign orchestrator features")

	// Check if this workflow has project configuration with campaign fields
	if workflowData.ParsedFrontmatter == nil || workflowData.ParsedFrontmatter.Project == nil {
		injectionLog.Print("No project field detected, skipping campaign injection")
		return nil
	}

	project := workflowData.ParsedFrontmatter.Project

	// Determine whether the project config looks like "project tracking" only.
	// A minimal campaign can specify only the project URL (either short or long form) and omit
	// campaign fields like id/workflows; in that case we infer the campaign ID from the workflow filename.
	hasTrackingOnlySettings := len(project.Scope) > 0 ||
		project.MaxUpdates > 0 ||
		project.MaxStatusUpdates > 0 ||
		strings.TrimSpace(project.GitHubToken) != "" ||
		project.DoNotDowngradeDoneItems != nil

	// Check if project has any campaign orchestration fields to determine if this is a campaign
	// Campaign indicators (any of these present means it's a campaign orchestrator):
	// - explicit campaign ID
	// - workflows list (predefined workers)
	// - governance policies (campaign-specific constraints)
	// - bootstrap configuration (initial work item generation)
	// - memory-paths, metrics-glob, cursor-glob (campaign state tracking)
	// If only URL and scope are present, it's simple project tracking, not a campaign
	hasCampaignIndicators := strings.TrimSpace(project.ID) != "" ||
		len(project.Workflows) > 0 ||
		project.Governance != nil ||
		project.Bootstrap != nil ||
		len(project.MemoryPaths) > 0 ||
		project.MetricsGlob != "" ||
		project.CursorGlob != ""

	// If the user used the object form with only a URL (no tracking-only knobs), treat it as a campaign
	// and infer the campaign ID from the workflow filename (minus .md).
	if !hasCampaignIndicators && !hasTrackingOnlySettings {
		if workflowData.WorkflowID != "" {
			project.ID = workflowData.WorkflowID
			hasCampaignIndicators = true
		}
	}

	isCampaign := hasCampaignIndicators

	if !isCampaign {
		injectionLog.Print("Project field present but no campaign indicators, treating as simple project tracking")
		return nil
	}

	injectionLog.Printf("Detected campaign orchestrator: workflows=%d, has_governance=%v, has_bootstrap=%v",
		len(project.Workflows), project.Governance != nil, project.Bootstrap != nil)

	// Derive campaign ID (prefer explicit id, then workflow filename, then workflow name).
	// Note: workflowData.FrontmatterName is the *frontmatter name field* (display name), not the file basename.
	campaignID := ""
	if strings.TrimSpace(project.ID) != "" {
		campaignID = project.ID
	} else if strings.TrimSpace(workflowData.WorkflowID) != "" {
		campaignID = workflowData.WorkflowID
	} else if strings.TrimSpace(workflowData.Name) != "" {
		campaignID = workflowData.Name
	} else {
		campaignID = "campaign"
	}
	campaignID = normalizeCampaignID(campaignID)
	if campaignID == "" {
		campaignID = "campaign"
	}

	// Apply campaign defaults (matching the historical .campaign.md defaults) when omitted.
	// This keeps project-based campaigns minimal: users can specify just url + id.
	project.ID = campaignID
	if strings.TrimSpace(project.TrackerLabel) == "" {
		project.TrackerLabel = fmt.Sprintf("z_campaign_%s", campaignID)
	}
	if len(project.MemoryPaths) == 0 {
		project.MemoryPaths = []string{fmt.Sprintf("memory/campaigns/%s/**", campaignID)}
	}
	if strings.TrimSpace(project.MetricsGlob) == "" {
		project.MetricsGlob = fmt.Sprintf("memory/campaigns/%s/metrics/*.json", campaignID)
	}
	if strings.TrimSpace(project.CursorGlob) == "" {
		project.CursorGlob = fmt.Sprintf("memory/campaigns/%s/cursor.json", campaignID)
	}

	// Build campaign prompt data from project configuration
	promptData := CampaignPromptData{
		CampaignID:   campaignID,
		CampaignName: workflowData.Name,
		ProjectURL:   project.URL,
		CursorGlob:   project.CursorGlob,
		MetricsGlob:  project.MetricsGlob,
		Workflows:    project.Workflows,
	}

	if project.Governance != nil {
		promptData.MaxDiscoveryItemsPerRun = project.Governance.MaxDiscoveryItemsPerRun
		promptData.MaxDiscoveryPagesPerRun = project.Governance.MaxDiscoveryPagesPerRun
		promptData.MaxProjectUpdatesPerRun = project.Governance.MaxProjectUpdatesPerRun
		promptData.MaxProjectCommentsPerRun = project.Governance.MaxCommentsPerRun
	}

	if project.Bootstrap != nil {
		promptData.BootstrapMode = project.Bootstrap.Mode
		if project.Bootstrap.SeederWorker != nil {
			promptData.SeederWorkerID = project.Bootstrap.SeederWorker.WorkflowID
			promptData.SeederMaxItems = project.Bootstrap.SeederWorker.MaxItems
		}
		if project.Bootstrap.ProjectTodos != nil {
			promptData.StatusField = project.Bootstrap.ProjectTodos.StatusField
			if promptData.StatusField == "" {
				promptData.StatusField = "Status"
			}
			promptData.TodoValue = project.Bootstrap.ProjectTodos.TodoValue
			if promptData.TodoValue == "" {
				promptData.TodoValue = "Todo"
			}
			promptData.TodoMaxItems = project.Bootstrap.ProjectTodos.MaxItems
			promptData.RequireFields = project.Bootstrap.ProjectTodos.RequireFields
		}
	}

	if len(project.Workers) > 0 {
		promptData.WorkerMetadata = make([]WorkerMetadata, len(project.Workers))
		for i, w := range project.Workers {
			promptData.WorkerMetadata[i] = WorkerMetadata{
				ID:                  w.ID,
				Name:                w.Name,
				Description:         w.Description,
				Capabilities:        w.Capabilities,
				IdempotencyStrategy: w.IdempotencyStrategy,
				Priority:            w.Priority,
			}
			// Convert payload schema
			if len(w.PayloadSchema) > 0 {
				promptData.WorkerMetadata[i].PayloadSchema = make(map[string]WorkerPayloadField)
				for key, field := range w.PayloadSchema {
					promptData.WorkerMetadata[i].PayloadSchema[key] = WorkerPayloadField{
						Type:        field.Type,
						Description: field.Description,
						Required:    field.Required,
						Example:     field.Example,
					}
				}
			}
			// Convert output labeling
			promptData.WorkerMetadata[i].OutputLabeling = WorkerOutputLabeling{
				Labels:         w.OutputLabeling.Labels,
				KeyInTitle:     w.OutputLabeling.KeyInTitle,
				KeyFormat:      w.OutputLabeling.KeyFormat,
				MetadataFields: w.OutputLabeling.MetadataFields,
			}
		}
	}

	// Append orchestrator instructions to markdown content
	markdownBuilder := &strings.Builder{}
	markdownBuilder.WriteString(workflowData.MarkdownContent)
	markdownBuilder.WriteString("\n\n")

	// Add bootstrap instructions if configured
	if project.Bootstrap != nil && project.Bootstrap.Mode != "" {
		bootstrapInstructions := RenderBootstrapInstructions(promptData)
		if bootstrapInstructions != "" {
			AppendPromptSection(markdownBuilder, "BOOTSTRAP INSTRUCTIONS (PHASE 0)", bootstrapInstructions)
		}
	}

	// Add workflow execution instructions
	workflowExecution := RenderWorkflowExecution(promptData)
	if workflowExecution != "" {
		AppendPromptSection(markdownBuilder, "WORKFLOW EXECUTION (PHASE 0)", workflowExecution)
	}

	// Add orchestrator instructions
	orchestratorInstructions := RenderOrchestratorInstructions(promptData)
	if orchestratorInstructions != "" {
		AppendPromptSection(markdownBuilder, "ORCHESTRATOR INSTRUCTIONS", orchestratorInstructions)
	}

	// Add project update instructions
	projectInstructions := RenderProjectUpdateInstructions(promptData)
	if projectInstructions != "" {
		AppendPromptSection(markdownBuilder, "PROJECT UPDATE INSTRUCTIONS (AUTHORITATIVE FOR WRITES)", projectInstructions)
	}

	// Add closing instructions
	closingInstructions := RenderClosingInstructions()
	if closingInstructions != "" {
		AppendPromptSection(markdownBuilder, "CLOSING INSTRUCTIONS (HIGHEST PRIORITY)", closingInstructions)
	}

	// Update the workflow markdown content with injected instructions
	workflowData.MarkdownContent = markdownBuilder.String()
	injectionLog.Printf("Injected campaign orchestrator instructions into workflow markdown")

	// Configure safe-outputs for campaign orchestration
	if workflowData.SafeOutputs == nil {
		workflowData.SafeOutputs = &workflow.SafeOutputsConfig{}
	}

	// Configure dispatch-workflow for worker coordination (optional - only if workflows are specified)
	if len(project.Workflows) > 0 && workflowData.SafeOutputs.DispatchWorkflow == nil {
		workflowData.SafeOutputs.DispatchWorkflow = &workflow.DispatchWorkflowConfig{
			BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 3},
			Workflows:            project.Workflows,
		}
		injectionLog.Printf("Configured dispatch-workflow safe-output for %d workflows", len(project.Workflows))
	} else if len(project.Workflows) == 0 {
		injectionLog.Print("No workflows specified - campaign will use custom discovery and dispatch logic")
	}

	// Configure update-project (already handled by applyProjectSafeOutputs, but ensure governance max is applied)
	if project.Governance != nil && project.Governance.MaxProjectUpdatesPerRun > 0 {
		if workflowData.SafeOutputs.UpdateProjects != nil {
			workflowData.SafeOutputs.UpdateProjects.Max = project.Governance.MaxProjectUpdatesPerRun
			injectionLog.Printf("Applied governance max-project-updates-per-run: %d", project.Governance.MaxProjectUpdatesPerRun)
		}
	}

	// Add concurrency control for campaigns if not already set
	if workflowData.Concurrency == "" {
		workflowData.Concurrency = fmt.Sprintf("concurrency:\n  group: \"campaign-%s-orchestrator-${{ github.ref }}\"\n  cancel-in-progress: false", campaignID)
		injectionLog.Printf("Added campaign concurrency control")
	}

	injectionLog.Printf("Successfully injected campaign orchestrator features for: %s", campaignID)
	return nil
}
