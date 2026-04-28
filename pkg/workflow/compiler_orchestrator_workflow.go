package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
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
			// Read source context lines (±3 lines around the error) for Rust-style rendering
			contextLines := readSourceContextLines(content, engineLine)
			return nil, formatCompilerErrorWithContext(cleanPath, engineLine, 1, "error", err.Error(), err, contextLines)
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

	// Validate run-install-scripts setting (warning in non-strict mode, error in strict mode)
	if err := c.validateRunInstallScripts(workflowData); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Validate engine version: warn when engine.version is explicitly set to "latest"
	if err := c.validateEngineVersion(workflowData); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Validate optional custom engine harness script configuration.
	if err := c.validateEngineHarnessScript(workflowData); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

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

	// Validate integrity-reactions feature configuration
	var gatewayConfig *MCPGatewayRuntimeConfig
	if workflowData.SandboxConfig != nil {
		gatewayConfig = workflowData.SandboxConfig.MCP
	}
	if err := validateIntegrityReactions(workflowData.ParsedTools, workflowData.Name, workflowData, gatewayConfig); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Use shared action cache and resolver from the compiler
	actionCache, actionResolver := c.getSharedActionResolver()
	workflowData.ActionCache = actionCache
	workflowData.ActionResolver = actionResolver
	workflowData.ActionPinWarnings = c.actionPinWarnings

	// Extract YAML configuration sections from frontmatter
	c.extractYAMLSections(result.Frontmatter, workflowData)

	// Merge observability config from imports into RawFrontmatter so that injectOTLPConfig
	// can see an OTLP endpoint defined in an imported workflow (first-wins from imports).
	if obs := engineSetup.importsResult.MergedObservability; obs != "" {
		if _, hasObs := workflowData.RawFrontmatter["observability"]; !hasObs {
			var obsMap map[string]any
			if err := json.Unmarshal([]byte(obs), &obsMap); err == nil {
				workflowData.RawFrontmatter["observability"] = obsMap
				orchestratorWorkflowLog.Printf("Merged observability config from imports into RawFrontmatter")
			}
		}
	}

	// Merge env from imports (main workflow env vars take precedence over imported env vars)
	if engineSetup.importsResult.MergedEnv != "" {
		topEnv := ExtractMapField(result.Frontmatter, "env")
		mergedEnvMap, err := mergeEnv(topEnv, engineSetup.importsResult.MergedEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to merge env from imports: %w", err)
		}
		if len(mergedEnvMap) > 0 {
			workflowData.Env = c.extractTopLevelYAMLSection(map[string]any{"env": mergedEnvMap}, "env")
			// Build source attribution: imported vars get the import path; main-workflow vars are labelled accordingly
			envSources := make(map[string]string, len(mergedEnvMap))
			for key := range mergedEnvMap {
				if _, inTop := topEnv[key]; inTop {
					envSources[key] = "(main workflow)"
				} else if src, ok := engineSetup.importsResult.MergedEnvSources[key]; ok {
					envSources[key] = src
				}
			}
			workflowData.EnvSources = envSources
		}
	} else if topEnv := ExtractMapField(result.Frontmatter, "env"); len(topEnv) > 0 {
		// No imports provided env — still label main workflow vars so the header can show them
		envSources := make(map[string]string, len(topEnv))
		for key := range topEnv {
			envSources[key] = "(main workflow)"
		}
		workflowData.EnvSources = envSources
	}

	// Inject OTLP configuration: add endpoint domain to firewall allowlist and
	// set OTEL env vars in the workflow env block (no-op when not configured).
	c.injectOTLPConfig(workflowData)

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

	// Process and merge pre-steps
	c.processAndMergePreSteps(result.Frontmatter, workflowData, engineSetup.importsResult)

	// Process and merge pre-agent-steps
	c.processAndMergePreAgentSteps(result.Frontmatter, workflowData, engineSetup.importsResult)

	// Process and merge post-steps
	c.processAndMergePostSteps(result.Frontmatter, workflowData, engineSetup.importsResult)

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
	workflowData.LabelCommand, workflowData.LabelCommandEvents, workflowData.LabelCommandRemoveLabel = c.extractLabelCommandConfig(frontmatter)
	workflowData.Jobs = c.extractJobsFromFrontmatter(frontmatter)

	// Merge jobs from imported YAML workflows
	if importsResult.MergedJobs != "" && importsResult.MergedJobs != "{}" {
		workflowData.Jobs = c.mergeJobsFromYAMLImports(workflowData.Jobs, importsResult.MergedJobs)
	}

	workflowData.Roles = c.extractRoles(frontmatter)
	workflowData.Bots = c.extractBots(frontmatter)
	workflowData.LabelNames = c.extractLabelNames(frontmatter)
	workflowData.RateLimit = c.extractRateLimitConfig(frontmatter)
	workflowData.SkipRoles = c.mergeSkipRoles(c.extractSkipRoles(frontmatter), importsResult.MergedSkipRoles)
	workflowData.SkipBots = c.mergeSkipBots(c.extractSkipBots(frontmatter), importsResult.MergedSkipBots)
	workflowData.ActivationGitHubToken = c.resolveActivationGitHubToken(frontmatter, importsResult)
	workflowData.ActivationGitHubApp = c.resolveActivationGitHubApp(frontmatter, importsResult)
	workflowData.TopLevelGitHubApp = resolveTopLevelGitHubApp(frontmatter, importsResult)

	// Use the already extracted output configuration
	workflowData.SafeOutputs = safeOutputs

	// Extract comment-memory from tools and attach to safe-outputs configuration.
	// comment-memory now belongs under tools: next to cache-memory and repo-memory.
	commentMemoryConfig := c.extractCommentMemoryConfig(toolsConfig)
	if commentMemoryConfig != nil {
		if workflowData.SafeOutputs == nil {
			workflowData.SafeOutputs = &SafeOutputsConfig{}
		}
		workflowData.SafeOutputs.CommentMemory = commentMemoryConfig
	}

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

	// Merge safe-outputs types from imports.
	// Pass the raw safe-outputs map from frontmatter so MergeSafeOutputs can distinguish
	// between types the user explicitly configured and types that were auto-defaulted by
	// extractSafeOutputsConfig. Without this, auto-defaults (e.g. threat-detection) would
	// prevent imported configurations for those types from being merged.
	rawSafeOutputsMap, _ := frontmatter["safe-outputs"].(map[string]any)
	mergedSafeOutputs, err := c.MergeSafeOutputs(workflowData.SafeOutputs, allSafeOutputsConfigs, rawSafeOutputsMap)
	if err != nil {
		return fmt.Errorf("failed to merge safe-outputs from imports: %w", err)
	}
	workflowData.SafeOutputs = mergedSafeOutputs

	// Apply default threat detection when safe-outputs came entirely from imports/includes
	// (i.e. the main frontmatter has no safe-outputs: section). In this case the merge
	// produces a non-nil SafeOutputs but leaves ThreatDetection nil, which would suppress
	// the detection gate on the safe_outputs job. Mirroring the behaviour of
	// extractSafeOutputsConfig for direct frontmatter declarations, we enable detection by
	// default unless any imported config explicitly sets threat-detection: false.
	if safeOutputs == nil && workflowData.SafeOutputs != nil && workflowData.SafeOutputs.ThreatDetection == nil {
		if !isThreatDetectionExplicitlyDisabledInConfigs(allSafeOutputsConfigs) {
			orchestratorWorkflowLog.Print("Applying default threat-detection for safe-outputs assembled from imports/includes")
			workflowData.SafeOutputs.ThreatDetection = &ThreatDetectionConfig{}
		}
	}

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

	// Process skip-if-check-failing configuration from the on: section
	if err := c.processSkipIfCheckFailingConfiguration(frontmatter, workflowData); err != nil {
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

	// Apply action pinning to on.steps
	if len(onSteps) > 0 {
		anySteps := make([]any, len(onSteps))
		for i, s := range onSteps {
			anySteps[i] = s
		}
		typedSteps, convErr := SliceToSteps(anySteps)
		if convErr == nil {
			typedSteps = applyActionPinsToTypedSteps(typedSteps, workflowData)
			for i, s := range typedSteps {
				onSteps[i] = s.ToMap()
			}
		} else {
			orchestratorWorkflowLog.Printf("Failed to convert on.steps to typed steps for action pinning: %v", convErr)
		}
	}

	workflowData.OnSteps = onSteps

	// Extract on.permissions for pre-activation job permissions
	workflowData.OnPermissions = extractOnPermissions(frontmatter)

	// Extract on.needs for pre-activation/activation job dependencies
	onNeeds, err := extractOnNeeds(frontmatter)
	if err != nil {
		return err
	}
	workflowData.OnNeeds = onNeeds

	return nil
}
