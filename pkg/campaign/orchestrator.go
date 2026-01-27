package campaign

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
)

var orchestratorLog = logger.New("campaign:orchestrator")

// convertStringsToAny converts a slice of strings to a slice of any
func convertStringsToAny(strings []string) []any {
	result := make([]any, len(strings))
	for i, s := range strings {
		result[i] = s
	}
	return result
}

// extractFileGlobPatterns extracts all file glob patterns from memory-paths or
// metrics-glob configuration. These patterns are used for the file-glob filter in
// repo-memory configuration to match files that the agent creates.
//
// For campaigns that use dated directory patterns (e.g., campaign-id-*/), this
// function preserves all wildcard patterns from memory-paths to support multiple
// directory structures (both dated and non-dated).
//
// Examples:
//   - memory-paths: ["memory/campaigns/project64-*/**", "memory/campaigns/project64/**"]
//     -> ["project64-*/**", "project64/**"]
//   - memory-paths: ["memory/campaigns/project64-*/**"] -> ["project64-*/**"]
//   - metrics-glob: "memory/campaigns/project64-*/metrics/*.json" -> ["project64-*/**"]
//   - no patterns with wildcards -> ["project64/**"] (fallback to ID)
func extractFileGlobPatterns(spec *CampaignSpec) []string {
	var patterns []string

	// Extract all patterns from memory-paths
	for _, memPath := range spec.MemoryPaths {
		// Remove "memory/campaigns/" prefix if present
		pattern := strings.TrimPrefix(memPath, "memory/campaigns/")
		// If pattern has both wildcards and slashes, it's a valid pattern
		if strings.Contains(pattern, "*") && strings.Contains(pattern, "/") {
			patterns = append(patterns, pattern)
			orchestratorLog.Printf("Extracted file-glob pattern from memory-paths: %s", pattern)
		}
	}

	// If we found patterns from memory-paths, return them
	if len(patterns) > 0 {
		return patterns
	}

	// Try to extract pattern from metrics-glob as fallback
	if spec.MetricsGlob != "" {
		pattern := strings.TrimPrefix(spec.MetricsGlob, "memory/campaigns/")
		if strings.Contains(pattern, "*") {
			// Extract the base directory pattern (everything before /metrics/ or first file-specific part)
			if idx := strings.Index(pattern, "/metrics/"); idx > 0 {
				basePattern := pattern[:idx] + "/**"
				orchestratorLog.Printf("Extracted file-glob pattern from metrics-glob: %s", basePattern)
				return []string{basePattern}
			}
		}
	}

	// Fallback to simple ID-based pattern
	fallbackPattern := fmt.Sprintf("%s/**", spec.ID)
	orchestratorLog.Printf("Using fallback file-glob pattern: %s", fallbackPattern)
	return []string{fallbackPattern}
}



// BuildOrchestrator constructs a minimal agentic workflow representation for a
// given CampaignSpec. The resulting WorkflowData is compiled via the standard
// CompileWorkflowDataWithValidation pipeline, and the orchestratorPath
// determines the emitted .lock.yml name.
func BuildOrchestrator(spec *CampaignSpec, campaignFilePath string) (*workflow.WorkflowData, string) {
	orchestratorLog.Printf("Building orchestrator for campaign: id=%s, file=%s", spec.ID, campaignFilePath)

	// Derive orchestrator markdown path alongside the campaign spec, using a
	// distinct suffix to avoid colliding with existing workflows. We use
	// a `.campaign.g.md` suffix to make it clear that the file is generated
	// from the corresponding `.campaign.md` spec.
	base := strings.TrimSuffix(campaignFilePath, ".campaign.md")
	orchestratorPath := base + ".campaign.g.md"
	orchestratorLog.Printf("Generated orchestrator path: %s", orchestratorPath)

	name := spec.Name
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Campaign: %s", spec.ID)
	}

	description := spec.Description
	if strings.TrimSpace(description) == "" {
		description = fmt.Sprintf("Orchestrator workflow for campaign '%s'", spec.ID)
	}

	// Default triggers: hourly schedule plus manual workflow_dispatch.
	onSection := "on:\n  schedule:\n    - cron: \"0 * * * *\"\n  workflow_dispatch:\n"

	// Prevent overlapping runs. This reduces sustained automated traffic on GitHub's
	// infrastructure by ensuring only one orchestrator run executes at a time per ref.
	concurrency := fmt.Sprintf("concurrency:\n  group: \"campaign-%s-orchestrator-${{ github.ref }}\"\n  cancel-in-progress: false", spec.ID)

	// Simple markdown body giving the agent context about the campaign.
	markdownBuilder := &strings.Builder{}
	markdownBuilder.WriteString("# Campaign Orchestrator\n\n")
	fmt.Fprintf(markdownBuilder, "This workflow orchestrates the '%s' campaign.\n\n", name)

	// Track whether we have any meaningful campaign details
	hasDetails := false

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
		fmt.Fprintf(markdownBuilder, "- Metrics glob: `%s`\n", spec.MetricsGlob)
		hasDetails = true
	}
	if spec.CursorGlob != "" {
		fmt.Fprintf(markdownBuilder, "- Cursor glob: `%s`\n", spec.CursorGlob)
		hasDetails = true
	}
	if strings.TrimSpace(spec.ProjectURL) != "" {
		fmt.Fprintf(markdownBuilder, "- Project URL: %s\n", strings.TrimSpace(spec.ProjectURL))
		hasDetails = true
	}
	if spec.Governance != nil {
		if spec.Governance.MaxNewItemsPerRun > 0 {
			fmt.Fprintf(markdownBuilder, "- Governance: max new items per run: %d\n", spec.Governance.MaxNewItemsPerRun)
			hasDetails = true
		}
		if spec.Governance.MaxDiscoveryItemsPerRun > 0 {
			fmt.Fprintf(markdownBuilder, "- Governance: max discovery items per run: %d\n", spec.Governance.MaxDiscoveryItemsPerRun)
			hasDetails = true
		}
		if spec.Governance.MaxDiscoveryPagesPerRun > 0 {
			fmt.Fprintf(markdownBuilder, "- Governance: max discovery pages per run: %d\n", spec.Governance.MaxDiscoveryPagesPerRun)
			hasDetails = true
		}
		if len(spec.Governance.OptOutLabels) > 0 {
			markdownBuilder.WriteString("- Governance: opt-out labels: ")
			markdownBuilder.WriteString(strings.Join(spec.Governance.OptOutLabels, ", "))
			markdownBuilder.WriteString("\n")
			hasDetails = true
		}
		if spec.Governance.DoNotDowngradeDoneItems != nil {
			fmt.Fprintf(markdownBuilder, "- Governance: do not downgrade done items: %t\n", *spec.Governance.DoNotDowngradeDoneItems)
			hasDetails = true
		}
		if spec.Governance.MaxProjectUpdatesPerRun > 0 {
			fmt.Fprintf(markdownBuilder, "- Governance: max project updates per run: %d\n", spec.Governance.MaxProjectUpdatesPerRun)
			hasDetails = true
		}
		if spec.Governance.MaxCommentsPerRun > 0 {
			fmt.Fprintf(markdownBuilder, "- Governance: max comments per run: %d\n", spec.Governance.MaxCommentsPerRun)
			hasDetails = true
		}
	}

	// Return nil if the campaign spec has no meaningful details for the prompt
	if !hasDetails {
		orchestratorLog.Printf("Campaign '%s' has no meaningful details, skipping orchestrator build", spec.ID)
		return nil, ""
	}

	orchestratorLog.Printf("Campaign '%s' orchestrator includes: workflows=%d, memory_paths=%d",
		spec.ID, len(spec.Workflows), len(spec.MemoryPaths))

	// Render orchestrator instructions using templates
	// All orchestrators follow the same system-agnostic rules with no conditional logic
	promptData := CampaignPromptData{
		CampaignID:   spec.ID,
		CampaignName: spec.Name,
		ProjectURL:   strings.TrimSpace(spec.ProjectURL),
		CursorGlob:   strings.TrimSpace(spec.CursorGlob),
		MetricsGlob:  strings.TrimSpace(spec.MetricsGlob),
		Workflows:    spec.Workflows,
	}
	if spec.Governance != nil {
		promptData.MaxDiscoveryItemsPerRun = spec.Governance.MaxDiscoveryItemsPerRun
		promptData.MaxDiscoveryPagesPerRun = spec.Governance.MaxDiscoveryPagesPerRun
		promptData.MaxProjectUpdatesPerRun = spec.Governance.MaxProjectUpdatesPerRun
		promptData.MaxProjectCommentsPerRun = spec.Governance.MaxCommentsPerRun
	}

	// Add bootstrap configuration if present
	if spec.Bootstrap != nil {
		promptData.BootstrapMode = spec.Bootstrap.Mode
		if spec.Bootstrap.SeederWorker != nil {
			promptData.SeederWorkerID = spec.Bootstrap.SeederWorker.WorkflowID
			// Convert payload map to JSON string for template rendering
			if len(spec.Bootstrap.SeederWorker.Payload) > 0 {
				payloadBytes, err := yaml.Marshal(spec.Bootstrap.SeederWorker.Payload)
				if err == nil {
					promptData.SeederPayload = string(payloadBytes)
				}
			}
			promptData.SeederMaxItems = spec.Bootstrap.SeederWorker.MaxItems
		}
		if spec.Bootstrap.ProjectTodos != nil {
			promptData.StatusField = spec.Bootstrap.ProjectTodos.StatusField
			if promptData.StatusField == "" {
				promptData.StatusField = "Status"
			}
			promptData.TodoValue = spec.Bootstrap.ProjectTodos.TodoValue
			if promptData.TodoValue == "" {
				promptData.TodoValue = "Todo"
			}
			promptData.TodoMaxItems = spec.Bootstrap.ProjectTodos.MaxItems
			promptData.RequireFields = spec.Bootstrap.ProjectTodos.RequireFields
		}
	}

	// Add worker metadata if present
	if len(spec.Workers) > 0 {
		promptData.WorkerMetadata = spec.Workers
	}

	// Render bootstrap instructions if bootstrap is configured
	if spec.Bootstrap != nil && spec.Bootstrap.Mode != "" {
		bootstrapInstructions := RenderBootstrapInstructions(promptData)
		if bootstrapInstructions == "" {
			orchestratorLog.Print("Warning: Failed to render bootstrap instructions, template may be missing")
		} else {
			AppendPromptSection(markdownBuilder, "BOOTSTRAP INSTRUCTIONS (PHASE 0)", bootstrapInstructions)
			orchestratorLog.Printf("Campaign '%s' orchestrator includes bootstrap mode: %s", spec.ID, spec.Bootstrap.Mode)
		}
	}

	// All campaigns include workflow execution capabilities
	// The orchestrator can dispatch workflows and make decisions regardless of initial configuration
	workflowExecution := RenderWorkflowExecution(promptData)
	if workflowExecution == "" {
		orchestratorLog.Print("Warning: Failed to render workflow execution instructions, template may be missing")
	} else {
		AppendPromptSection(markdownBuilder, "WORKFLOW EXECUTION (PHASE 0)", workflowExecution)
		orchestratorLog.Printf("Campaign '%s' orchestrator includes workflow execution", spec.ID)
	}

	orchestratorInstructions := RenderOrchestratorInstructions(promptData)
	if orchestratorInstructions == "" {
		orchestratorLog.Print("Warning: Failed to render orchestrator instructions, template may be missing")
	} else {
		AppendPromptSection(markdownBuilder, "ORCHESTRATOR INSTRUCTIONS", orchestratorInstructions)
	}

	projectInstructions := RenderProjectUpdateInstructions(promptData)
	if projectInstructions == "" {
		orchestratorLog.Print("Warning: Failed to render project update instructions, template may be missing")
	} else {
		AppendPromptSection(markdownBuilder, "PROJECT UPDATE INSTRUCTIONS (AUTHORITATIVE FOR WRITES)", projectInstructions)
	}

	closingInstructions := RenderClosingInstructions()
	if closingInstructions == "" {
		orchestratorLog.Print("Warning: Failed to render closing instructions, template may be missing")
	} else {
		AppendPromptSection(markdownBuilder, "CLOSING INSTRUCTIONS (HIGHEST PRIORITY)", closingInstructions)
	}

	// Campaign orchestrators can dispatch workflows and perform limited Project operations.
	// Project writes (update-project, create-project-status-update) are allowed to enable
	// orchestrators to maintain campaign dashboards and status updates.
	//
	// Note: Campaign orchestrators intentionally omit explicit `permissions:` from
	// the generated markdown; safe-output jobs have their own scoped permissions.
	safeOutputs := &workflow.SafeOutputsConfig{}

	// Configure dispatch-workflow for worker coordination
	if len(spec.Workflows) > 0 {
		dispatchWorkflowConfig := &workflow.DispatchWorkflowConfig{
			BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 3},
			Workflows:            spec.Workflows,
		}
		safeOutputs.DispatchWorkflow = dispatchWorkflowConfig
		orchestratorLog.Printf("Campaign orchestrator '%s' configured with dispatch_workflow for %d workflows", spec.ID, len(spec.Workflows))
	}

	// Configure update-project for campaign dashboard maintenance
	maxProjectUpdates := 100 // default - increased from 10 to handle larger discovery sets
	if spec.Governance != nil && spec.Governance.MaxProjectUpdatesPerRun > 0 {
		maxProjectUpdates = spec.Governance.MaxProjectUpdatesPerRun
	}
	updateProjectConfig := &workflow.UpdateProjectConfig{
		BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxProjectUpdates},
	}
	safeOutputs.UpdateProjects = updateProjectConfig
	orchestratorLog.Printf("Campaign orchestrator '%s' configured with update-project (max: %d)", spec.ID, maxProjectUpdates)

	// Configure create-project-status-update for campaign summaries
	statusUpdateConfig := &workflow.CreateProjectStatusUpdateConfig{
		BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 1},
	}
	safeOutputs.CreateProjectStatusUpdates = statusUpdateConfig
	orchestratorLog.Printf("Campaign orchestrator '%s' configured with create-project-status-update", spec.ID)

	orchestratorLog.Printf("Campaign orchestrator '%s' built successfully with dispatch-workflow, update-project, and create-project-status-update safe outputs", spec.ID)

	// Extract file-glob patterns from memory-paths or metrics-glob to support
	// multiple directory structures (e.g., both dated "campaign-id-*/**" and non-dated "campaign-id/**")
	fileGlobPatterns := extractFileGlobPatterns(spec)

	// Determine engine to use (default to claude if not specified)
	engineID := "claude"
	if spec.Engine != "" {
		engineID = spec.Engine
		orchestratorLog.Printf("Campaign orchestrator '%s' using specified engine: %s", spec.ID, engineID)
	} else {
		orchestratorLog.Printf("Campaign orchestrator '%s' using default engine: %s", spec.ID, engineID)
	}

	// Configure GitHub MCP for discovery with budget enforcement
	maxDiscoveryItems := 100
	maxDiscoveryPages := 10
	if spec.Governance != nil {
		if spec.Governance.MaxDiscoveryItemsPerRun > 0 {
			maxDiscoveryItems = spec.Governance.MaxDiscoveryItemsPerRun
		}
		if spec.Governance.MaxDiscoveryPagesPerRun > 0 {
			maxDiscoveryPages = spec.Governance.MaxDiscoveryPagesPerRun
		}
	}

	tools := map[string]any{
		"github": map[string]any{
			"toolsets": []string{"repos", "issues", "pull_requests"},
			"mode":     "remote",
		},
		"repo-memory": []any{
			map[string]any{
				"id":          "campaigns",
				"branch-name": "memory/campaigns",
				"file-glob":   convertStringsToAny(fileGlobPatterns),
				"campaign-id": spec.ID,
			},
		},
		"bash": []any{"*"},
		"edit": nil,
	}
	orchestratorLog.Printf("Campaign orchestrator '%s' configured with GitHub MCP (max items: %d, max pages: %d)",
		spec.ID, maxDiscoveryItems, maxDiscoveryPages)

	data := &workflow.WorkflowData{
		Name:            name,
		Description:     description,
		MarkdownContent: markdownBuilder.String(),
		On:              onSection,
		Concurrency:     concurrency,
		// Use a standard Ubuntu runner for the main agent job so the
		// compiled orchestrator always has a valid runs-on value.
		RunsOn: "runs-on: ubuntu-latest",
		// Default roles match the workflow compiler's defaults so that
		// membership checks have a non-empty GH_AW_REQUIRED_ROLES value.
		Roles: []string{"admin", "maintainer", "write"},
		// Set the engine configuration from campaign spec
		EngineConfig: &workflow.EngineConfig{
			ID: engineID,
		},
		Tools:       tools,
		SafeOutputs: safeOutputs,
	}

	return data, orchestratorPath
}
