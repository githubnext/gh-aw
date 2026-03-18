package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var orchestratorWorkflowLog = logger.New("workflow:compiler_orchestrator_workflow")

// ParseWorkflowFile parses a workflow markdown file and returns a WorkflowData structure.
// This is the main orchestration function that coordinates all compilation phases.
func (c *Compiler) ParseWorkflowFile(markdownPath string) (*WorkflowData, error) {
	orchestratorWorkflowLog.Printf("Starting workflow file parsing: %s", markdownPath)

	// Parse frontmatter section
	parseResult, err := c.parseFrontmatterSection(markdownPath)
	if err != nil {
		return nil, err
	}

	// Handle shared workflows
	if parseResult.isSharedWorkflow {
		return nil, &SharedWorkflowError{Path: parseResult.cleanPath}
	}

	// Unpack parse result for convenience
	cleanPath := parseResult.cleanPath
	content := parseResult.content
	result := parseResult.frontmatterResult
	markdownDir := parseResult.markdownDir

	// Setup engine and process imports
	engineSetup, err := c.setupEngineAndImports(result, cleanPath, content, markdownDir)
	if err != nil {
		// Wrap unformatted errors with file location.  Errors produced by
		// formatCompilerError/formatCompilerErrorWithPosition are already
		// console-formatted and must not be double-wrapped.
		if isFormattedCompilerError(err) {
			return nil, err
		}
		// Try to point at the exact line of the "engine:" field so the user can
		// navigate directly to the problem location.
		engineLine := findFrontmatterFieldLine(result.FrontmatterLines, result.FrontmatterStart, "engine")
		if engineLine > 0 {
			return nil, formatCompilerErrorWithPosition(cleanPath, engineLine, 1, "error", err.Error(), err)
		}
		return nil, formatCompilerError(cleanPath, "error", err.Error(), err)
	}

	// Process tools and markdown
	toolsResult, err := c.processToolsAndMarkdown(result, cleanPath, markdownDir, engineSetup.agenticEngine, engineSetup.engineSetting, engineSetup.importsResult)
	if err != nil {
		if isFormattedCompilerError(err) {
			return nil, err
		}
		return nil, formatCompilerError(cleanPath, "error", err.Error(), err)
	}

	// Build initial workflow data structure
	workflowData := c.buildInitialWorkflowData(result, toolsResult, engineSetup, engineSetup.importsResult)
	// Store a stable workflow identifier derived from the file name.
	workflowData.WorkflowID = GetWorkflowIDFromPath(cleanPath)

	// Validate that inlined-imports is not used with agent file imports.
	// Agent files require runtime access and cannot be resolved without sources.
	if workflowData.InlinedImports && engineSetup.importsResult.AgentFile != "" {
		return nil, formatCompilerError(cleanPath, "error",
			fmt.Sprintf("inlined-imports cannot be used with agent file imports: '%s'. "+
				"Agent files require runtime access and will not be resolved without sources. "+
				"Remove 'inlined-imports: true' or do not import agent files.",
				engineSetup.importsResult.AgentFile), nil)
	}

	// Validate bash tool configuration BEFORE applying defaults
	// This must happen before applyDefaults() which converts nil bash to default commands
	if err := validateBashToolConfig(workflowData.ParsedTools, workflowData.Name); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Validate GitHub tool configuration
	if err := validateGitHubToolConfig(workflowData.ParsedTools, workflowData.Name); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Validate GitHub tool read-only configuration
	if err := validateGitHubReadOnly(workflowData.ParsedTools, workflowData.Name); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Validate GitHub guard policy configuration
	if err := validateGitHubGuardPolicy(workflowData.ParsedTools, workflowData.Name); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Use shared action cache and resolver from the compiler
	actionCache, actionResolver := c.getSharedActionResolver()
	workflowData.ActionCache = actionCache
	workflowData.ActionResolver = actionResolver
	workflowData.ActionPinWarnings = c.actionPinWarnings

	// Extract YAML configuration sections from frontmatter
	c.extractYAMLSections(result.Frontmatter, workflowData)

	// Merge features from imports
	if len(engineSetup.importsResult.MergedFeatures) > 0 {
		mergedFeatures, err := c.MergeFeatures(workflowData.Features, engineSetup.importsResult.MergedFeatures)
		if err != nil {
			return nil, fmt.Errorf("failed to merge features from imports: %w", err)
		}
		workflowData.Features = mergedFeatures
	}

	// Process and merge custom steps with imported steps
	c.processAndMergeSteps(result.Frontmatter, workflowData, engineSetup.importsResult)

	// Process and merge post-steps
	c.processAndMergePostSteps(result.Frontmatter, workflowData)

	// Process and merge services
	c.processAndMergeServices(result.Frontmatter, workflowData, engineSetup.importsResult)

	// Extract additional configurations (cache, mcp-scripts, safe-outputs, etc.)
	if err := c.extractAdditionalConfigurations(
		result.Frontmatter,
		toolsResult.tools,
		markdownDir,
		workflowData,
		engineSetup.importsResult,
		result.Markdown,
		toolsResult.safeOutputs,
	); err != nil {
		return nil, err
	}

	// Note: Git commands are automatically injected when safe-outputs needs them (see compiler_safe_outputs.go)
	// No validation needed here - the compiler handles adding git to bash allowlist

	// Process on section configuration and apply filters
	if err := c.processOnSectionAndFilters(result.Frontmatter, workflowData, cleanPath); err != nil {
		return nil, err
	}

	orchestratorWorkflowLog.Printf("Workflow file parsing completed successfully: %s", markdownPath)
	return workflowData, nil
}

// buildInitialWorkflowData creates the initial WorkflowData struct with basic fields populated
func (c *Compiler) buildInitialWorkflowData(
	result *parser.FrontmatterResult,
	toolsResult *toolsProcessingResult,
	engineSetup *engineSetupResult,
	importsResult *parser.ImportsResult,
) *WorkflowData {
	orchestratorWorkflowLog.Print("Building initial workflow data")

	inlinedImports := resolveInlinedImports(result.Frontmatter)

	// When inlined-imports is true, agent file content is already inlined via ImportPaths → step 1b.
	// Clear AgentFile/AgentImportSpec so engines don't read it from disk separately at runtime.
	agentFile := importsResult.AgentFile
	agentImportSpec := importsResult.AgentImportSpec
	if inlinedImports {
		agentFile = ""
		agentImportSpec = ""
	}

	workflowData := &WorkflowData{
		Name:                  toolsResult.workflowName,
		FrontmatterName:       toolsResult.frontmatterName,
		FrontmatterYAML:       strings.Join(result.FrontmatterLines, "\n"),
		Description:           c.extractDescription(result.Frontmatter),
		Source:                c.extractSource(result.Frontmatter),
		TrackerID:             toolsResult.trackerID,
		ImportedFiles:         importsResult.ImportedFiles,
		ImportedMarkdown:      toolsResult.importedMarkdown, // Only imports WITH inputs
		ImportPaths:           toolsResult.importPaths,      // Import paths for runtime-import macros (imports without inputs)
		MainWorkflowMarkdown:  toolsResult.mainWorkflowMarkdown,
		IncludedFiles:         toolsResult.allIncludedFiles,
		ImportInputs:          importsResult.ImportInputs,
		Tools:                 toolsResult.tools,
		ParsedTools:           NewTools(toolsResult.tools),
		Runtimes:              toolsResult.runtimes,
		PluginInfo:            toolsResult.pluginInfo,
		APMDependencies:       toolsResult.apmDependencies,
		MarkdownContent:       toolsResult.markdownContent,
		AI:                    engineSetup.engineSetting,
		EngineConfig:          engineSetup.engineConfig,
		AgentFile:             agentFile,
		AgentImportSpec:       agentImportSpec,
		RepositoryImports:     importsResult.RepositoryImports,
		NetworkPermissions:    engineSetup.networkPermissions,
		SandboxConfig:         applySandboxDefaults(engineSetup.sandboxConfig, engineSetup.engineConfig),
		NeedsTextOutput:       toolsResult.needsTextOutput,
		ToolsTimeout:          toolsResult.toolsTimeout,
		ToolsStartupTimeout:   toolsResult.toolsStartupTimeout,
		TrialMode:             c.trialMode,
		TrialLogicalRepo:      c.trialLogicalRepoSlug,
		StrictMode:            c.strictMode,
		SecretMasking:         toolsResult.secretMasking,
		ParsedFrontmatter:     toolsResult.parsedFrontmatter,
		RawFrontmatter:        result.Frontmatter,
		HasExplicitGitHubTool: toolsResult.hasExplicitGitHubTool,
		ActionMode:            c.actionMode,
		InlinedImports:        inlinedImports,
		EngineConfigSteps:     engineSetup.configSteps,
	}

	// Populate checkout configs from parsed frontmatter.
	// Fall back to raw frontmatter parsing when full ParseFrontmatterConfig fails
	// (e.g. due to unrecognised tool config shapes like bash: ["*"]).
	if toolsResult.parsedFrontmatter != nil {
		workflowData.CheckoutConfigs = toolsResult.parsedFrontmatter.CheckoutConfigs
		workflowData.CheckoutDisabled = toolsResult.parsedFrontmatter.CheckoutDisabled
	} else if rawCheckout, ok := result.Frontmatter["checkout"]; ok {
		if checkoutValue, ok := rawCheckout.(bool); ok && !checkoutValue {
			workflowData.CheckoutDisabled = true
		} else if configs, err := ParseCheckoutConfigs(rawCheckout); err == nil {
			workflowData.CheckoutConfigs = configs
		}
	}

	return workflowData
}

// resolveInlinedImports returns true if inlined-imports is enabled.
// It reads the value directly from the raw (pre-parsed) frontmatter map, which is always
// populated regardless of whether ParseFrontmatterConfig succeeded.
func resolveInlinedImports(rawFrontmatter map[string]any) bool {
	return ParseBoolFromConfig(rawFrontmatter, "inlined-imports", nil)
}

// extractYAMLSections extracts YAML configuration sections from frontmatter
func (c *Compiler) extractYAMLSections(frontmatter map[string]any, workflowData *WorkflowData) {
	orchestratorWorkflowLog.Print("Extracting YAML sections from frontmatter")

	workflowData.On = c.extractTopLevelYAMLSection(frontmatter, "on")
	workflowData.HasDispatchItemNumber = extractDispatchItemNumber(frontmatter)
	workflowData.Permissions = c.extractPermissions(frontmatter)
	workflowData.Network = c.extractTopLevelYAMLSection(frontmatter, "network")
	workflowData.ConcurrencyJobDiscriminator = extractConcurrencyJobDiscriminator(frontmatter)
	workflowData.Concurrency = c.extractConcurrencySection(frontmatter)
	workflowData.RunName = c.extractTopLevelYAMLSection(frontmatter, "run-name")
	workflowData.Env = c.extractTopLevelYAMLSection(frontmatter, "env")
	workflowData.Features = c.extractFeatures(frontmatter)
	workflowData.If = c.extractIfCondition(frontmatter)

	// Extract timeout-minutes (canonical form)
	workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(frontmatter, "timeout-minutes")

	workflowData.RunsOn = c.extractTopLevelYAMLSection(frontmatter, "runs-on")
	workflowData.Environment = c.extractTopLevelYAMLSection(frontmatter, "environment")
	workflowData.Container = c.extractTopLevelYAMLSection(frontmatter, "container")
	workflowData.Cache = c.extractTopLevelYAMLSection(frontmatter, "cache")
}

// extractConcurrencyJobDiscriminator reads the job-discriminator value from the
// frontmatter concurrency block without modifying the original map.
// Returns the discriminator expression string or empty string if not present.
func extractConcurrencyJobDiscriminator(frontmatter map[string]any) string {
	concurrencyRaw, ok := frontmatter["concurrency"]
	if !ok {
		return ""
	}
	concurrencyMap, ok := concurrencyRaw.(map[string]any)
	if !ok {
		return ""
	}
	discriminator, ok := concurrencyMap["job-discriminator"]
	if !ok {
		return ""
	}
	discriminatorStr, ok := discriminator.(string)
	if !ok {
		return ""
	}
	return discriminatorStr
}

// extractConcurrencySection extracts the workflow-level concurrency YAML section,
// stripping the gh-aw-specific job-discriminator field so it does not appear in
// the compiled lock file (which must be valid GitHub Actions YAML).
func (c *Compiler) extractConcurrencySection(frontmatter map[string]any) string {
	concurrencyRaw, ok := frontmatter["concurrency"]
	if !ok {
		return ""
	}
	concurrencyMap, ok := concurrencyRaw.(map[string]any)
	if !ok || len(concurrencyMap) == 0 {
		// String or empty format: serialize as-is (no job-discriminator possible)
		return c.extractTopLevelYAMLSection(frontmatter, "concurrency")
	}

	_, hasDiscriminator := concurrencyMap["job-discriminator"]
	if !hasDiscriminator {
		return c.extractTopLevelYAMLSection(frontmatter, "concurrency")
	}

	// Build a copy of the concurrency map without job-discriminator for serialization.
	// Use len(concurrencyMap) for capacity: at most one entry (job-discriminator) will be
	// omitted, so this is a slight over-allocation that avoids a subtle negative-capacity
	// edge case if job-discriminator were the only key.
	cleanMap := make(map[string]any, len(concurrencyMap))
	for k, v := range concurrencyMap {
		if k != "job-discriminator" {
			cleanMap[k] = v
		}
	}
	// When job-discriminator is the only field, there is no user-specified workflow-level
	// group to emit; return empty so the compiler can generate the default concurrency.
	if len(cleanMap) == 0 {
		return ""
	}
	// Use a minimal temporary frontmatter containing only the concurrency key to avoid
	// copying the entire (potentially large) frontmatter map.
	return c.extractTopLevelYAMLSection(map[string]any{"concurrency": cleanMap}, "concurrency")
}

// extractDispatchItemNumber reports whether the frontmatter's on.workflow_dispatch
// trigger exposes an item_number input. This is the signature produced by the label
// trigger shorthand (e.g. "on: pull_request labeled my-label"). Reading the
// structured map avoids re-parsing the rendered YAML string later.
func extractDispatchItemNumber(frontmatter map[string]any) bool {
	onVal, ok := frontmatter["on"]
	if !ok {
		return false
	}
	onMap, ok := onVal.(map[string]any)
	if !ok {
		return false
	}
	wdVal, ok := onMap["workflow_dispatch"]
	if !ok {
		return false
	}
	wdMap, ok := wdVal.(map[string]any)
	if !ok {
		return false
	}
	inputsVal, ok := wdMap["inputs"]
	if !ok {
		return false
	}
	inputsMap, ok := inputsVal.(map[string]any)
	if !ok {
		return false
	}
	_, ok = inputsMap["item_number"]
	return ok
}

// processAndMergeSteps handles the merging of imported steps with main workflow steps
func (c *Compiler) processAndMergeSteps(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	orchestratorWorkflowLog.Print("Processing and merging custom steps")

	workflowData.CustomSteps = c.extractTopLevelYAMLSection(frontmatter, "steps")

	// Parse copilot-setup-steps if present (these go at the start)
	var copilotSetupSteps []any
	if importsResult.CopilotSetupSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.CopilotSetupSteps), &copilotSetupSteps); err != nil {
			orchestratorWorkflowLog.Printf("Failed to unmarshal copilot-setup steps: %v", err)
		} else {
			// Convert to typed steps for action pinning
			typedCopilotSteps, err := SliceToSteps(copilotSetupSteps)
			if err != nil {
				orchestratorWorkflowLog.Printf("Failed to convert copilot-setup steps to typed steps: %v", err)
			} else {
				// Apply action pinning to copilot-setup steps
				typedCopilotSteps = ApplyActionPinsToTypedSteps(typedCopilotSteps, workflowData)
				// Convert back to []any for YAML marshaling
				copilotSetupSteps = StepsToSlice(typedCopilotSteps)
			}
		}
	}

	// Parse other imported steps if present (these go after copilot-setup but before main steps)
	var otherImportedSteps []any
	if importsResult.MergedSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.MergedSteps), &otherImportedSteps); err == nil {
			// Convert to typed steps for action pinning
			typedOtherSteps, err := SliceToSteps(otherImportedSteps)
			if err != nil {
				orchestratorWorkflowLog.Printf("Failed to convert other imported steps to typed steps: %v", err)
			} else {
				// Apply action pinning to other imported steps
				typedOtherSteps = ApplyActionPinsToTypedSteps(typedOtherSteps, workflowData)
				// Convert back to []any for YAML marshaling
				otherImportedSteps = StepsToSlice(typedOtherSteps)
			}
		}
	}

	// If there are main workflow steps, parse them
	var mainSteps []any
	if workflowData.CustomSteps != "" {
		var mainStepsWrapper map[string]any
		if err := yaml.Unmarshal([]byte(workflowData.CustomSteps), &mainStepsWrapper); err == nil {
			if mainStepsVal, hasSteps := mainStepsWrapper["steps"]; hasSteps {
				if steps, ok := mainStepsVal.([]any); ok {
					mainSteps = steps
					// Convert to typed steps for action pinning
					typedMainSteps, err := SliceToSteps(mainSteps)
					if err != nil {
						orchestratorWorkflowLog.Printf("Failed to convert main steps to typed steps: %v", err)
					} else {
						// Apply action pinning to main steps
						typedMainSteps = ApplyActionPinsToTypedSteps(typedMainSteps, workflowData)
						// Convert back to []any for YAML marshaling
						mainSteps = StepsToSlice(typedMainSteps)
					}
				}
			}
		}
	}

	// Merge steps in the correct order:
	// 1. copilot-setup-steps (at start)
	// 2. other imported steps (after copilot-setup)
	// 3. main frontmatter steps (last)
	var allSteps []any
	if len(copilotSetupSteps) > 0 || len(mainSteps) > 0 || len(otherImportedSteps) > 0 {
		allSteps = append(allSteps, copilotSetupSteps...)
		allSteps = append(allSteps, otherImportedSteps...)
		allSteps = append(allSteps, mainSteps...)

		// Convert back to YAML with "steps:" wrapper
		stepsWrapper := map[string]any{"steps": allSteps}
		stepsYAML, err := yaml.Marshal(stepsWrapper)
		if err == nil {
			// Remove quotes from uses values with version comments
			workflowData.CustomSteps = unquoteUsesWithComments(string(stepsYAML))
		}
	}
}

// processAndMergePostSteps handles the processing of post-steps with action pinning
func (c *Compiler) processAndMergePostSteps(frontmatter map[string]any, workflowData *WorkflowData) {
	orchestratorWorkflowLog.Print("Processing post-steps")

	workflowData.PostSteps = c.extractTopLevelYAMLSection(frontmatter, "post-steps")

	// Apply action pinning to post-steps if any
	if workflowData.PostSteps != "" {
		var postStepsWrapper map[string]any
		if err := yaml.Unmarshal([]byte(workflowData.PostSteps), &postStepsWrapper); err == nil {
			if postStepsVal, hasPostSteps := postStepsWrapper["post-steps"]; hasPostSteps {
				if postSteps, ok := postStepsVal.([]any); ok {
					// Convert to typed steps for action pinning
					typedPostSteps, err := SliceToSteps(postSteps)
					if err != nil {
						orchestratorWorkflowLog.Printf("Failed to convert post-steps to typed steps: %v", err)
					} else {
						// Apply action pinning to post steps using type-safe version
						typedPostSteps = ApplyActionPinsToTypedSteps(typedPostSteps, workflowData)
						// Convert back to []any for YAML marshaling
						postSteps = StepsToSlice(typedPostSteps)
					}

					// Convert back to YAML with "post-steps:" wrapper
					stepsWrapper := map[string]any{"post-steps": postSteps}
					stepsYAML, err := yaml.Marshal(stepsWrapper)
					if err == nil {
						// Remove quotes from uses values with version comments
						workflowData.PostSteps = unquoteUsesWithComments(string(stepsYAML))
					}
				}
			}
		}
	}
}

// processAndMergeServices handles the merging of imported services with main workflow services
func (c *Compiler) processAndMergeServices(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	orchestratorWorkflowLog.Print("Processing and merging services")

	workflowData.Services = c.extractTopLevelYAMLSection(frontmatter, "services")

	// Merge imported services if any
	if importsResult.MergedServices != "" {
		// Parse imported services from YAML
		var importedServices map[string]any
		if err := yaml.Unmarshal([]byte(importsResult.MergedServices), &importedServices); err == nil {
			// If there are main workflow services, parse and merge them
			if workflowData.Services != "" {
				// Parse main workflow services
				var mainServicesWrapper map[string]any
				if err := yaml.Unmarshal([]byte(workflowData.Services), &mainServicesWrapper); err == nil {
					if mainServices, ok := mainServicesWrapper["services"].(map[string]any); ok {
						// Merge: main workflow services take precedence over imported
						for key, value := range importedServices {
							if _, exists := mainServices[key]; !exists {
								mainServices[key] = value
							}
						}
						// Convert back to YAML with "services:" wrapper
						servicesWrapper := map[string]any{"services": mainServices}
						servicesYAML, err := yaml.Marshal(servicesWrapper)
						if err == nil {
							workflowData.Services = string(servicesYAML)
						}
					}
				}
			} else {
				// Only imported services exist, wrap in "services:" format
				servicesWrapper := map[string]any{"services": importedServices}
				servicesYAML, err := yaml.Marshal(servicesWrapper)
				if err == nil {
					workflowData.Services = string(servicesYAML)
				}
			}
		}
	}
}

// mergeJobsFromYAMLImports merges jobs from imported YAML workflows with main workflow jobs
// Main workflow jobs take precedence over imported jobs (override behavior)
func (c *Compiler) mergeJobsFromYAMLImports(mainJobs map[string]any, mergedJobsJSON string) map[string]any {
	orchestratorWorkflowLog.Print("Merging jobs from imported YAML workflows")

	if mergedJobsJSON == "" || mergedJobsJSON == "{}" {
		orchestratorWorkflowLog.Print("No imported jobs to merge")
		return mainJobs
	}

	// Initialize result with main jobs or create empty map
	result := make(map[string]any)
	maps.Copy(result, mainJobs)

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(mergedJobsJSON, "\n")
	orchestratorWorkflowLog.Printf("Processing %d job definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to map
		var importedJobs map[string]any
		if err := json.Unmarshal([]byte(line), &importedJobs); err != nil {
			orchestratorWorkflowLog.Printf("Skipping malformed job entry: %v", err)
			continue
		}

		// Merge jobs - main workflow jobs take precedence (don't override)
		for jobName, jobConfig := range importedJobs {
			if _, exists := result[jobName]; !exists {
				orchestratorWorkflowLog.Printf("Adding imported job: %s", jobName)
				result[jobName] = jobConfig
			} else {
				orchestratorWorkflowLog.Printf("Skipping imported job %s (already defined in main workflow)", jobName)
			}
		}
	}

	orchestratorWorkflowLog.Printf("Successfully merged jobs: total=%d, imported=%d", len(result), len(result)-len(mainJobs))
	return result
}

// extractTopLevelGitHubApp extracts the 'github-app' field from the top-level frontmatter.
// This provides a single GitHub App configuration that serves as a fallback for all nested
// github-app token minting operations (on, safe-outputs, checkout, tools.github, dependencies).
func extractTopLevelGitHubApp(frontmatter map[string]any) *GitHubAppConfig {
	appAny, ok := frontmatter["github-app"]
	if !ok {
		return nil
	}
	appMap, ok := appAny.(map[string]any)
	if !ok {
		return nil
	}
	app := parseAppConfig(appMap)
	if app.AppID == "" || app.PrivateKey == "" {
		return nil
	}
	return app
}

// resolveTopLevelGitHubApp resolves the top-level github-app for token minting fallback.
// Precedence:
//  1. Current workflow's top-level github-app (explicit override wins)
//  2. First top-level github-app found across imported shared workflows
//  3. Nil (no fallback configured)
func resolveTopLevelGitHubApp(frontmatter map[string]any, importsResult *parser.ImportsResult) *GitHubAppConfig {
	if app := extractTopLevelGitHubApp(frontmatter); app != nil {
		return app
	}
	if importsResult != nil && importsResult.MergedTopLevelGitHubApp != "" {
		var appMap map[string]any
		if err := json.Unmarshal([]byte(importsResult.MergedTopLevelGitHubApp), &appMap); err == nil {
			app := parseAppConfig(appMap)
			if app.AppID != "" && app.PrivateKey != "" {
				orchestratorWorkflowLog.Print("Using top-level github-app from imported shared workflow")
				return app
			}
		}
	}
	return nil
}

// topLevelFallbackNeeded reports whether the top-level github-app should be applied as a
// fallback for a given section. It returns true when the section has neither an explicit
// github-app nor an explicit github-token already configured.
//
// Rules (consistent across all sections):
//   - If a section-specific github-app is set → keep it, no fallback needed.
//   - If a section-specific github-token is set → keep it, no fallback needed (a token
//     already provides the auth; injecting a github-app would silently change precedence).
//   - Otherwise → apply the top-level fallback.
func topLevelFallbackNeeded(app *GitHubAppConfig, token string) bool {
	return app == nil && token == ""
}

// applyTopLevelGitHubAppFallbacks applies the top-level github-app as a fallback for all
// nested github-app token minting operations when no section-specific github-app is configured.
// Precedence: section-specific github-app > section-specific github-token > top-level github-app.
//
// Every section uses topLevelFallbackNeeded to decide whether the fallback is required,
// ensuring consistent behaviour across all token-minting sites.
func applyTopLevelGitHubAppFallbacks(data *WorkflowData) {
	fallback := data.TopLevelGitHubApp
	if fallback == nil {
		return
	}

	// Fallback for activation (on.github-app / on.github-token)
	if topLevelFallbackNeeded(data.ActivationGitHubApp, data.ActivationGitHubToken) {
		orchestratorWorkflowLog.Print("Applying top-level github-app fallback for activation")
		data.ActivationGitHubApp = fallback
	}

	// Fallback for safe-outputs (safe-outputs.github-app / safe-outputs.github-token)
	if data.SafeOutputs != nil && topLevelFallbackNeeded(data.SafeOutputs.GitHubApp, data.SafeOutputs.GitHubToken) {
		orchestratorWorkflowLog.Print("Applying top-level github-app fallback for safe-outputs")
		data.SafeOutputs.GitHubApp = fallback
	}

	// Fallback for checkout configs (checkout.github-app / checkout.github-token per entry)
	for _, cfg := range data.CheckoutConfigs {
		if topLevelFallbackNeeded(cfg.GitHubApp, cfg.GitHubToken) {
			orchestratorWorkflowLog.Print("Applying top-level github-app fallback for checkout")
			cfg.GitHubApp = fallback
		}
	}

	// Fallback for tools.github (tools.github.github-app / tools.github.github-token).
	// Also skipped when tools.github is explicitly disabled (github: false) — do not re-enable it.
	if data.ParsedTools != nil && data.ParsedTools.GitHub != nil &&
		topLevelFallbackNeeded(data.ParsedTools.GitHub.GitHubApp, data.ParsedTools.GitHub.GitHubToken) &&
		data.Tools["github"] != false {
		orchestratorWorkflowLog.Print("Applying top-level github-app fallback for tools.github")
		data.ParsedTools.GitHub.GitHubApp = fallback
		// Also update the raw tools map so applyDefaultTools (called from applyDefaults in
		// processOnSectionAndFilters) does not lose the fallback when it rebuilds ParsedTools
		// from the map.
		appMap := map[string]any{
			"app-id":      fallback.AppID,
			"private-key": fallback.PrivateKey,
		}
		if fallback.Owner != "" {
			appMap["owner"] = fallback.Owner
		}
		if len(fallback.Repositories) > 0 {
			repos := make([]any, len(fallback.Repositories))
			for i, r := range fallback.Repositories {
				repos[i] = r
			}
			appMap["repositories"] = repos
		}
		// Normalize data.Tools["github"] to a map so the github-app survives re-parsing.
		// Configurations like `github: true` are normalized here rather than losing the fallback.
		if github, ok := data.Tools["github"].(map[string]any); ok {
			// Already a map; inject into existing settings.
			github["github-app"] = appMap
		} else {
			// Non-map value (e.g. true) — create a fresh map.
			data.Tools["github"] = map[string]any{"github-app": appMap}
		}
	}

	// Fallback for APM dependencies (dependencies.github-app; no github-token field)
	if data.APMDependencies != nil && topLevelFallbackNeeded(data.APMDependencies.GitHubApp, "") {
		orchestratorWorkflowLog.Print("Applying top-level github-app fallback for dependencies")
		data.APMDependencies.GitHubApp = fallback
	}
}

// extractAdditionalConfigurations extracts cache-memory, repo-memory, mcp-scripts, and safe-outputs configurations
func (c *Compiler) extractAdditionalConfigurations(
	frontmatter map[string]any,
	tools map[string]any,
	markdownDir string,
	workflowData *WorkflowData,
	importsResult *parser.ImportsResult,
	markdown string,
	safeOutputs *SafeOutputsConfig,
) error {
	orchestratorWorkflowLog.Print("Extracting additional configurations")

	// Extract cache-memory config and check for errors
	cacheMemoryConfig, err := c.extractCacheMemoryConfigFromMap(tools)
	if err != nil {
		return err
	}
	workflowData.CacheMemoryConfig = cacheMemoryConfig

	// Extract repo-memory config and check for errors
	toolsConfig, err := ParseToolsConfig(tools)
	if err != nil {
		return err
	}
	repoMemoryConfig, err := c.extractRepoMemoryConfig(toolsConfig, workflowData.WorkflowID)
	if err != nil {
		return err
	}
	workflowData.RepoMemoryConfig = repoMemoryConfig

	// Extract and process mcp-scripts and safe-outputs
	workflowData.Command, workflowData.CommandEvents = c.extractCommandConfig(frontmatter)
	workflowData.LabelCommand, workflowData.LabelCommandEvents = c.extractLabelCommandConfig(frontmatter)
	workflowData.Jobs = c.extractJobsFromFrontmatter(frontmatter)

	// Merge jobs from imported YAML workflows
	if importsResult.MergedJobs != "" && importsResult.MergedJobs != "{}" {
		workflowData.Jobs = c.mergeJobsFromYAMLImports(workflowData.Jobs, importsResult.MergedJobs)
	}

	workflowData.Roles = c.extractRoles(frontmatter)
	workflowData.Bots = c.extractBots(frontmatter)
	workflowData.RateLimit = c.extractRateLimitConfig(frontmatter)
	workflowData.SkipRoles = c.mergeSkipRoles(c.extractSkipRoles(frontmatter), importsResult.MergedSkipRoles)
	workflowData.SkipBots = c.mergeSkipBots(c.extractSkipBots(frontmatter), importsResult.MergedSkipBots)
	workflowData.ActivationGitHubToken = c.resolveActivationGitHubToken(frontmatter, importsResult)
	workflowData.ActivationGitHubApp = c.resolveActivationGitHubApp(frontmatter, importsResult)
	workflowData.TopLevelGitHubApp = resolveTopLevelGitHubApp(frontmatter, importsResult)

	// Use the already extracted output configuration
	workflowData.SafeOutputs = safeOutputs

	// Extract mcp-scripts configuration
	workflowData.MCPScripts = c.extractMCPScriptsConfig(frontmatter)

	// Merge mcp-scripts from imports
	if len(importsResult.MergedMCPScripts) > 0 {
		workflowData.MCPScripts = c.mergeMCPScripts(workflowData.MCPScripts, importsResult.MergedMCPScripts)
	}

	// Extract safe-jobs from safe-outputs.jobs location
	topSafeJobs := extractSafeJobsFromFrontmatter(frontmatter)

	// Process @include directives to extract additional safe-outputs configurations
	includedSafeOutputsConfigs, err := parser.ExpandIncludesForSafeOutputs(markdown, markdownDir)
	if err != nil {
		return fmt.Errorf("failed to expand includes for safe-outputs: %w", err)
	}

	// Combine imported safe-outputs with included safe-outputs
	var allSafeOutputsConfigs []string
	if len(importsResult.MergedSafeOutputs) > 0 {
		allSafeOutputsConfigs = append(allSafeOutputsConfigs, importsResult.MergedSafeOutputs...)
	}
	if len(includedSafeOutputsConfigs) > 0 {
		allSafeOutputsConfigs = append(allSafeOutputsConfigs, includedSafeOutputsConfigs...)
	}

	// Merge safe-jobs from all safe-outputs configurations (imported and included)
	includedSafeJobs, err := c.mergeSafeJobsFromIncludedConfigs(topSafeJobs, allSafeOutputsConfigs)
	if err != nil {
		return fmt.Errorf("failed to merge safe-jobs from includes: %w", err)
	}

	// Merge app configuration from included safe-outputs configurations
	includedApp, err := c.mergeAppFromIncludedConfigs(workflowData.SafeOutputs, allSafeOutputsConfigs)
	if err != nil {
		return fmt.Errorf("failed to merge app from includes: %w", err)
	}

	// Ensure SafeOutputs exists and populate the Jobs field with merged jobs
	if workflowData.SafeOutputs == nil && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs = &SafeOutputsConfig{}
	}
	// Always use the merged includedSafeJobs as it contains both main and imported jobs
	if workflowData.SafeOutputs != nil && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs.Jobs = includedSafeJobs
	}

	// Populate the App field if it's not set in the top-level workflow but is in an included config
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.GitHubApp == nil && includedApp != nil {
		workflowData.SafeOutputs.GitHubApp = includedApp
	}

	// Merge safe-outputs types from imports
	mergedSafeOutputs, err := c.MergeSafeOutputs(workflowData.SafeOutputs, allSafeOutputsConfigs)
	if err != nil {
		return fmt.Errorf("failed to merge safe-outputs from imports: %w", err)
	}
	workflowData.SafeOutputs = mergedSafeOutputs

	// Auto-inject create-issues if safe-outputs is configured but has no non-builtin outputs.
	// This ensures every workflow with safe-outputs has at least one meaningful action handler.
	applyDefaultCreateIssue(workflowData)

	// Apply the top-level github-app as a fallback for all nested github-app token minting operations.
	// This runs last so that all section-specific configurations have been resolved first.
	applyTopLevelGitHubAppFallbacks(workflowData)

	return nil
}

// processOnSectionAndFilters processes the on section configuration and applies various filters
func (c *Compiler) processOnSectionAndFilters(
	frontmatter map[string]any,
	workflowData *WorkflowData,
	cleanPath string,
) error {
	orchestratorWorkflowLog.Print("Processing on section and filters")

	// Process stop-after configuration from the on: section
	if err := c.processStopAfterConfiguration(frontmatter, workflowData, cleanPath); err != nil {
		return err
	}

	// Process skip-if-match configuration from the on: section
	if err := c.processSkipIfMatchConfiguration(frontmatter, workflowData); err != nil {
		return err
	}

	// Process skip-if-no-match configuration from the on: section
	if err := c.processSkipIfNoMatchConfiguration(frontmatter, workflowData); err != nil {
		return err
	}

	// Process manual-approval configuration from the on: section
	if err := c.processManualApprovalConfiguration(frontmatter, workflowData); err != nil {
		return err
	}

	// Parse the "on" section for command triggers, reactions, and other events
	if err := c.parseOnSection(frontmatter, workflowData, cleanPath); err != nil {
		return err
	}

	// Apply defaults
	if err := c.applyDefaults(workflowData, cleanPath); err != nil {
		return err
	}

	// Apply pull request draft filter if specified
	c.applyPullRequestDraftFilter(workflowData, frontmatter)

	// Apply pull request fork filter if specified
	c.applyPullRequestForkFilter(workflowData, frontmatter)

	// Apply label filter if specified
	c.applyLabelFilter(workflowData, frontmatter)

	// Extract on.steps for pre-activation step injection
	onSteps, err := extractOnSteps(frontmatter)
	if err != nil {
		return err
	}
	workflowData.OnSteps = onSteps

	// Extract on.permissions for pre-activation job permissions
	workflowData.OnPermissions = extractOnPermissions(frontmatter)

	return nil
}
