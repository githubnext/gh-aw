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

// buildDiscoverySteps creates GitHub Actions steps for campaign discovery precomputation
func buildDiscoverySteps(spec *CampaignSpec) []map[string]any {
	// Only add discovery steps if we have workflows or a tracker label
	if len(spec.Workflows) == 0 && spec.TrackerLabel == "" {
		orchestratorLog.Printf("Skipping discovery steps: no workflows or tracker label configured")
		return nil
	}

	orchestratorLog.Printf("Building discovery steps for campaign: %s", spec.ID)

	// Build environment variables for discovery
	envVars := map[string]any{
		"GH_AW_CAMPAIGN_ID":         spec.ID,
		"GH_AW_WORKFLOWS":           strings.Join(spec.Workflows, ","),
		"GH_AW_TRACKER_LABEL":       spec.TrackerLabel,
		"GH_AW_PROJECT_URL":         spec.ProjectURL,
		"GH_AW_MAX_DISCOVERY_ITEMS": fmt.Sprintf("%d", getMaxDiscoveryItems(spec)),
		"GH_AW_MAX_DISCOVERY_PAGES": fmt.Sprintf("%d", getMaxDiscoveryPages(spec)),
		"GH_AW_CURSOR_PATH":         getCursorPath(spec),
	}

	// Add GH_AW_DISCOVERY_REPOS from spec.DiscoveryRepos
	if len(spec.DiscoveryRepos) > 0 {
		envVars["GH_AW_DISCOVERY_REPOS"] = strings.Join(spec.DiscoveryRepos, ",")
		orchestratorLog.Printf("Setting GH_AW_DISCOVERY_REPOS from discovery-repos: %v", spec.DiscoveryRepos)
	}

	// Add GH_AW_DISCOVERY_ORGS from spec.DiscoveryOrgs if provided
	if len(spec.DiscoveryOrgs) > 0 {
		envVars["GH_AW_DISCOVERY_ORGS"] = strings.Join(spec.DiscoveryOrgs, ",")
		orchestratorLog.Printf("Setting GH_AW_DISCOVERY_ORGS from discovery-orgs: %v", spec.DiscoveryOrgs)
	}

	steps := []map[string]any{
		{
			"name": "Create workspace directory",
			"run":  "mkdir -p ./.gh-aw",
		},
		{
			"name": "Run campaign discovery precomputation",
			"id":   "discovery",
			"uses": "actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd", // v8.0.0
			"env":  envVars,
			"with": map[string]any{
				"github-token": "${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
				"script": `
const { setupGlobals } = require('/opt/gh-aw/actions/setup_globals.cjs');
setupGlobals(core, github, context, exec, io);
const { main } = require('/opt/gh-aw/actions/campaign_discovery.cjs');
await main();
`,
			},
		},
	}

	return steps
}

// getMaxDiscoveryItems returns the max discovery items budget from governance or default
func getMaxDiscoveryItems(spec *CampaignSpec) int {
	if spec.Governance != nil && spec.Governance.MaxDiscoveryItemsPerRun > 0 {
		return spec.Governance.MaxDiscoveryItemsPerRun
	}
	return 100 // default
}

// getMaxDiscoveryPages returns the max discovery pages budget from governance or default
func getMaxDiscoveryPages(spec *CampaignSpec) int {
	if spec.Governance != nil && spec.Governance.MaxDiscoveryPagesPerRun > 0 {
		return spec.Governance.MaxDiscoveryPagesPerRun
	}
	return 10 // default
}

// getCursorPath returns the cursor path for the campaign or empty if not configured
func getCursorPath(spec *CampaignSpec) string {
	if spec.CursorGlob != "" {
		// Convert glob to actual path - remove wildcards and use repo-memory path
		// For now, use a simple convention: /tmp/gh-aw/repo-memory/campaigns/<campaign-id>/cursor.json
		return fmt.Sprintf("/tmp/gh-aw/repo-memory/campaigns/%s/cursor.json", spec.ID)
	}
	return ""
}

// renderStepsAsYAML renders a list of steps as YAML string for CustomSteps field
func renderStepsAsYAML(steps []map[string]any) string {
	if len(steps) == 0 {
		return ""
	}

	// Marshal steps to YAML
	data, err := yaml.Marshal(steps)
	if err != nil {
		orchestratorLog.Printf("Failed to marshal discovery steps to YAML: %v", err)
		return ""
	}

	return string(data)
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

	// Default triggers: daily schedule plus manual workflow_dispatch.
	onSection := "on:\n  schedule:\n    - cron: \"0 18 * * *\"\n  workflow_dispatch:\n"

	// Prevent overlapping runs. This reduces sustained automated traffic on GitHub's
	// infrastructure by ensuring only one orchestrator run executes at a time per ref.
	concurrency := fmt.Sprintf("concurrency:\n  group: \"campaign-%s-orchestrator-${{ github.ref }}\"\n  cancel-in-progress: false", spec.ID)

	// Simple markdown body giving the agent context about the campaign.
	markdownBuilder := &strings.Builder{}
	markdownBuilder.WriteString("# Campaign Orchestrator\n\n")
	fmt.Fprintf(markdownBuilder, "This workflow orchestrates the '%s' campaign.\n\n", name)

	// Track whether we have any meaningful campaign details
	hasDetails := false

	if strings.TrimSpace(spec.Objective) != "" {
		fmt.Fprintf(markdownBuilder, "- Objective: %s\n", strings.TrimSpace(spec.Objective))
		hasDetails = true
	}
	if len(spec.KPIs) > 0 {
		markdownBuilder.WriteString("- KPIs:\n")
		for _, kpi := range spec.KPIs {
			name := strings.TrimSpace(kpi.Name)
			if name == "" {
				name = "(unnamed)"
			}
			priority := strings.TrimSpace(kpi.Priority)
			if priority == "" && len(spec.KPIs) == 1 {
				priority = "primary"
			}
			unit := strings.TrimSpace(kpi.Unit)
			if unit != "" {
				unit = " " + unit
			}
			if priority != "" {
				priority = " (" + priority + ")"
			}
			fmt.Fprintf(markdownBuilder, "  - %s%s: baseline %.4g → target %.4g over %d days%s\n", name, priority, kpi.Baseline, kpi.Target, kpi.TimeWindowDays, unit)
		}
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
		Objective:    strings.TrimSpace(spec.Objective),
		ProjectURL:   strings.TrimSpace(spec.ProjectURL),
		CursorGlob:   strings.TrimSpace(spec.CursorGlob),
		MetricsGlob:  strings.TrimSpace(spec.MetricsGlob),
		Workflows:    spec.Workflows,
	}
	if len(spec.KPIs) > 0 {
		promptData.KPIs = spec.KPIs
	}
	if spec.Governance != nil {
		promptData.MaxDiscoveryItemsPerRun = spec.Governance.MaxDiscoveryItemsPerRun
		promptData.MaxDiscoveryPagesPerRun = spec.Governance.MaxDiscoveryPagesPerRun
		promptData.MaxProjectUpdatesPerRun = spec.Governance.MaxProjectUpdatesPerRun
		promptData.MaxProjectCommentsPerRun = spec.Governance.MaxCommentsPerRun
	}

	// All campaigns include workflow execution capabilities
	// The orchestrator can dispatch workflows and make decisions regardless of initial configuration
	workflowExecution := RenderWorkflowExecution(promptData)
	if workflowExecution == "" {
		orchestratorLog.Print("Warning: Failed to render workflow execution instructions, template may be missing")
	} else {
		appendPromptSection(markdownBuilder, "WORKFLOW EXECUTION (PHASE 0)", workflowExecution)
		orchestratorLog.Printf("Campaign '%s' orchestrator includes workflow execution", spec.ID)
	}

	orchestratorInstructions := RenderOrchestratorInstructions(promptData)
	if orchestratorInstructions == "" {
		orchestratorLog.Print("Warning: Failed to render orchestrator instructions, template may be missing")
	} else {
		appendPromptSection(markdownBuilder, "ORCHESTRATOR INSTRUCTIONS", orchestratorInstructions)
	}

	projectInstructions := RenderProjectUpdateInstructions(promptData)
	if projectInstructions == "" {
		orchestratorLog.Print("Warning: Failed to render project update instructions, template may be missing")
	} else {
		appendPromptSection(markdownBuilder, "PROJECT UPDATE INSTRUCTIONS (AUTHORITATIVE FOR WRITES)", projectInstructions)
	}

	closingInstructions := RenderClosingInstructions()
	if closingInstructions == "" {
		orchestratorLog.Print("Warning: Failed to render closing instructions, template may be missing")
	} else {
		appendPromptSection(markdownBuilder, "CLOSING INSTRUCTIONS (HIGHEST PRIORITY)", closingInstructions)
	}

	// Enable safe outputs needed for campaign coordination.
	// Note: Campaign orchestrators intentionally omit explicit `permissions:` from
	// the generated markdown; safe-output jobs have their own scoped permissions.
	maxComments := 10
	maxProjectUpdates := 10
	if spec.Governance != nil {
		if spec.Governance.MaxCommentsPerRun > 0 {
			maxComments = spec.Governance.MaxCommentsPerRun
		}
		if spec.Governance.MaxProjectUpdatesPerRun > 0 {
			maxProjectUpdates = spec.Governance.MaxProjectUpdatesPerRun
		}
	}

	safeOutputs := &workflow.SafeOutputsConfig{}
	// Allow creating the Epic issue for the campaign (max: 1, only created once).
	safeOutputs.CreateIssues = &workflow.CreateIssuesConfig{BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 1}}
	// Allow commenting on related issues/PRs as part of campaign coordination.
	safeOutputs.AddComments = &workflow.AddCommentsConfig{BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxComments}}
	// Allow updating the campaign's GitHub Project dashboard.
	updateProjectConfig := &workflow.UpdateProjectConfig{BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxProjectUpdates}}
	// If the campaign spec specifies a custom GitHub token for Projects v2 operations,
	// pass it to the update-project configuration.
	if strings.TrimSpace(spec.ProjectGitHubToken) != "" {
		updateProjectConfig.GitHubToken = strings.TrimSpace(spec.ProjectGitHubToken)
		orchestratorLog.Printf("Campaign orchestrator '%s' configured with custom GitHub token for update-project", spec.ID)
	}
	safeOutputs.UpdateProjects = updateProjectConfig

	// Allow creating project status updates for campaign summaries.
	statusUpdateConfig := &workflow.CreateProjectStatusUpdateConfig{BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 1}}
	// Use the same custom GitHub token for status updates as for project operations.
	if strings.TrimSpace(spec.ProjectGitHubToken) != "" {
		statusUpdateConfig.GitHubToken = strings.TrimSpace(spec.ProjectGitHubToken)
		orchestratorLog.Printf("Campaign orchestrator '%s' configured with custom GitHub token for create-project-status-update", spec.ID)
	}
	safeOutputs.CreateProjectStatusUpdates = statusUpdateConfig

	// Add dispatch_workflow if workflows are configured
	// This allows the orchestrator to dispatch worker workflows for the campaign
	if len(spec.Workflows) > 0 {
		dispatchWorkflowConfig := &workflow.DispatchWorkflowConfig{
			BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 3},
			Workflows:            spec.Workflows,
		}
		safeOutputs.DispatchWorkflow = dispatchWorkflowConfig
		orchestratorLog.Printf("Campaign orchestrator '%s' configured with dispatch_workflow for %d workflows", spec.ID, len(spec.Workflows))
	}

	// Enable append-only comments for campaigns - create new comments instead of updating existing ones
	// This ensures full history of campaign run updates is preserved
	safeOutputs.Messages = &workflow.SafeOutputMessagesConfig{
		AppendOnlyComments: true,
	}

	orchestratorLog.Printf("Campaign orchestrator '%s' built successfully with safe outputs enabled", spec.ID)

	// Extract file-glob patterns from memory-paths or metrics-glob to support
	// multiple directory structures (e.g., both dated "campaign-id-*/**" and non-dated "campaign-id/**")
	fileGlobPatterns := extractFileGlobPatterns(spec)

	// Build discovery step configuration
	// This runs before the agent to precompute campaign discovery
	discoverySteps := buildDiscoverySteps(spec)

	// Determine engine to use (default to claude if not specified)
	engineID := "claude"
	if spec.Engine != "" {
		engineID = spec.Engine
		orchestratorLog.Printf("Campaign orchestrator '%s' using specified engine: %s", spec.ID, engineID)
	} else {
		orchestratorLog.Printf("Campaign orchestrator '%s' using default engine: %s", spec.ID, engineID)
	}

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
		Tools: map[string]any{
			"github": map[string]any{
				"toolsets": []any{"default", "actions", "code_security"},
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
		},
		SafeOutputs: safeOutputs,
	}

	// Add discovery steps if configuration is valid
	if len(discoverySteps) > 0 {
		data.CustomSteps = renderStepsAsYAML(discoverySteps)
	}

	return data, orchestratorPath
}
