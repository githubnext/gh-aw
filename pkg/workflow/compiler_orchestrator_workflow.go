package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/setutil"
)

var orchestratorWorkflowLog = logger.New("workflow:compiler_orchestrator_workflow")

// workflowBuildContext captures the shared state across parse setup, validation,
// and workflow population phases.
//
// setupWorkflowBuildContext must run before validateWorkflowBuildContext or
// populateWorkflowBuildContext. engineSetup, toolsResult, and workflowData stay
// nil until setup completes successfully.
type workflowBuildContext struct {
	cleanPath   string
	content     []byte
	frontmatter *parser.FrontmatterResult
	markdownDir string

	engineSetup  *engineSetupResult
	toolsResult  *toolsProcessingResult
	workflowData *WorkflowData
}

// ParseWorkflowFile parses a workflow markdown file and returns a WorkflowData structure.
// This is the main orchestration function that coordinates all compilation phases.
func (c *Compiler) ParseWorkflowFile(markdownPath string) (*WorkflowData, error) {
	orchestratorWorkflowLog.Printf("Starting workflow file parsing: %s", markdownPath)

	parseResult, err := c.parseFrontmatterSection(markdownPath)
	if err != nil {
		return nil, err
	}

	if err := validateParseResultWorkflowType(parseResult); err != nil {
		return nil, err
	}
	ctx := newWorkflowBuildContext(parseResult)
	if err := c.setupWorkflowBuildContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateWorkflowBuildContext(ctx); err != nil {
		return nil, err
	}
	if err := c.populateWorkflowBuildContext(ctx); err != nil {
		return nil, err
	}
	orchestratorWorkflowLog.Printf("Workflow file parsing completed successfully: %s", markdownPath)
	return ctx.workflowData, nil
}

func validateParseResultWorkflowType(parseResult *frontmatterParseResult) error {
	if parseResult.isSharedWorkflow {
		return &SharedWorkflowError{Path: parseResult.cleanPath}
	}
	if parseResult.isRedirectOnly {
		return &RedirectOnlyWorkflowError{Path: parseResult.cleanPath, Target: parseResult.redirectTarget}
	}
	return nil
}

// newWorkflowBuildContext initializes build context from frontmatter parse results.
func newWorkflowBuildContext(parseResult *frontmatterParseResult) *workflowBuildContext {
	return &workflowBuildContext{
		cleanPath:   parseResult.cleanPath,
		content:     parseResult.content,
		frontmatter: parseResult.frontmatterResult,
		markdownDir: parseResult.markdownDir,
	}
}

// setupWorkflowBuildContext initializes engine/tools processing and builds base workflow data.
func (c *Compiler) setupWorkflowBuildContext(ctx *workflowBuildContext) error {
	engineSetup, err := c.setupEngineAndImports(ctx.frontmatter, ctx.cleanPath, ctx.content, ctx.markdownDir)
	if err != nil {
		return c.formatEngineSetupError(ctx, err)
	}
	toolsResult, err := c.processToolsAndMarkdown(
		ctx.frontmatter,
		ctx.cleanPath,
		ctx.markdownDir,
		engineSetup.agenticEngine,
		engineSetup.engineSetting,
		engineSetup.importsResult,
	)
	if err != nil {
		return c.formatToolsProcessingError(ctx.cleanPath, err)
	}
	ctx.engineSetup = engineSetup
	ctx.toolsResult = toolsResult
	ctx.workflowData = c.buildInitialWorkflowData(ctx.frontmatter, toolsResult, engineSetup, engineSetup.importsResult)
	ctx.workflowData.WorkflowID = GetWorkflowIDFromPath(ctx.cleanPath)
	return nil
}

func (c *Compiler) formatEngineSetupError(ctx *workflowBuildContext, err error) error {
	if isFormattedCompilerError(err) {
		return err
	}
	engineLine := findFrontmatterFieldLine(ctx.frontmatter.FrontmatterLines, ctx.frontmatter.FrontmatterStart, "engine")
	if engineLine > 0 {
		contextLines := readSourceContextLines(ctx.content, engineLine)
		return formatCompilerErrorWithContext(ctx.cleanPath, engineLine, 1, "error", err.Error(), err, contextLines)
	}
	return formatCompilerError(ctx.cleanPath, "error", err.Error(), err)
}

func (c *Compiler) formatToolsProcessingError(cleanPath string, err error) error {
	if isFormattedCompilerError(err) {
		return err
	}
	return formatCompilerError(cleanPath, "error", err.Error(), err)
}

// validateWorkflowBuildContext runs model, engine, and tool validations for the workflow.
func (c *Compiler) validateWorkflowBuildContext(ctx *workflowBuildContext) error {
	if err := c.validateWorkflowModelAliasMap(ctx); err != nil {
		return err
	}
	if err := c.validateWorkflowEngineSettings(ctx.cleanPath, ctx.workflowData); err != nil {
		return err
	}
	return c.validateWorkflowToolConfigurations(ctx)
}

func (c *Compiler) validateWorkflowModelAliasMap(ctx *workflowBuildContext) error {
	var engineModel string
	if ctx.workflowData.EngineConfig != nil {
		engineModel = ctx.workflowData.EngineConfig.Model
	}
	return c.validateModelAliasMap(ctx.workflowData.ModelMappings, nil, engineModel, ctx.cleanPath)
}

func (c *Compiler) validateWorkflowEngineSettings(cleanPath string, workflowData *WorkflowData) error {
	// Preserve legacy ParseWorkflowFile error precedence: return the first
	// engine-setting validation failure in the same order the monolithic
	// implementation executed these checks.
	checks := []func(*WorkflowData) error{
		c.validateRunInstallScripts,
		c.validateEngineVersion,
		c.validatePlaywrightMode,
		c.validateLSPSupport,
		c.validateEngineHarnessScript,
		c.validateEngineDriver,
		c.validateEngineMCPSessionTimeout,
		c.validateEngineMCPToolTimeout,
	}
	for _, check := range checks {
		if err := check(workflowData); err != nil {
			return fmt.Errorf("%s: %w", cleanPath, err)
		}
	}
	return nil
}

func (c *Compiler) validateWorkflowToolConfigurations(ctx *workflowBuildContext) error {
	if ctx.workflowData.InlinedImports && ctx.engineSetup.importsResult.AgentFile != "" {
		return formatCompilerError(ctx.cleanPath, "error",
			fmt.Sprintf("inlined-imports cannot be used with agent file imports: '%s'. "+
				"Agent files require runtime access and will not be resolved without sources. "+
				"Remove 'inlined-imports: true' or do not import agent files.",
				ctx.engineSetup.importsResult.AgentFile), nil)
	}
	if err := validateBashToolConfig(ctx.workflowData.ParsedTools, ctx.workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", ctx.cleanPath, err)
	}
	if err := validateGitHubToolConfig(ctx.workflowData.ParsedTools, ctx.workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", ctx.cleanPath, err)
	}
	if err := validateGitHubReadOnly(ctx.workflowData.ParsedTools, ctx.workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", ctx.cleanPath, err)
	}
	if err := validateGitHubGuardPolicy(ctx.workflowData.ParsedTools, ctx.workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", ctx.cleanPath, err)
	}
	emitGitHubLockdownGuardPolicyWarning(c, ctx.workflowData.ParsedTools, ctx.cleanPath)
	var gatewayConfig *MCPGatewayRuntimeConfig
	if ctx.workflowData.SandboxConfig != nil {
		gatewayConfig = ctx.workflowData.SandboxConfig.MCP
	}
	if err := validateIntegrityReactions(ctx.workflowData.ParsedTools, ctx.workflowData.Name, ctx.workflowData, gatewayConfig); err != nil {
		return fmt.Errorf("%s: %w", ctx.cleanPath, err)
	}
	return nil
}

// populateWorkflowBuildContext merges imported configuration and finalizes workflow data.
func (c *Compiler) populateWorkflowBuildContext(ctx *workflowBuildContext) error {
	c.attachSharedActionResolver(ctx.workflowData)
	if err := c.extractYAMLSections(ctx.frontmatter.Frontmatter, ctx.workflowData); err != nil {
		return formatCompilerError(ctx.cleanPath, "error", err.Error(), err)
	}
	if err := c.mergeImportedWorkflowConfiguration(ctx); err != nil {
		return err
	}
	if err := c.processAndMergeSteps(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.engineSetup.importsResult); err != nil {
		return formatCompilerError(ctx.cleanPath, "error", err.Error(), err)
	}
	if err := c.processAndMergePreSteps(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.engineSetup.importsResult); err != nil {
		return formatCompilerError(ctx.cleanPath, "error", err.Error(), err)
	}
	if err := c.processAndMergePreAgentSteps(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.engineSetup.importsResult); err != nil {
		return formatCompilerError(ctx.cleanPath, "error", err.Error(), err)
	}
	if err := c.processAndMergePostSteps(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.engineSetup.importsResult); err != nil {
		return formatCompilerError(ctx.cleanPath, "error", err.Error(), err)
	}
	c.processAndMergeServices(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.engineSetup.importsResult)
	ctx.workflowData.KnownActionCredentialEnvVars = DetectKnownCredentialLeakingActionsFromWorkflowData(ctx.workflowData)
	if err := c.extractAdditionalConfigurations(ctx.frontmatter.Frontmatter, ctx.toolsResult.tools, ctx.markdownDir, ctx.workflowData, ctx.engineSetup.importsResult, ctx.toolsResult.rawMainMarkdown, ctx.toolsResult.safeOutputs); err != nil {
		return err
	}
	if err := c.mergeImportedOnFields(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.engineSetup.importsResult); err != nil {
		return err
	}
	return c.processOnSectionAndFilters(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.cleanPath)
}

func (c *Compiler) attachSharedActionResolver(workflowData *WorkflowData) {
	actionCache, actionResolver := c.getSharedActionResolver()
	workflowData.Ctx = c.ctx
	workflowData.ActionCache = actionCache
	workflowData.ActionResolver = actionResolver
	workflowData.ActionPinWarnings = c.actionPinWarnings
	workflowData.ActionPinMappings = c.getActionPinMappings()
}

func (c *Compiler) mergeImportedWorkflowConfiguration(ctx *workflowBuildContext) error {
	c.mergeImportedObservability(ctx.workflowData, ctx.engineSetup.importsResult.MergedObservability)
	if err := c.mergeWorkflowEnv(ctx.frontmatter.Frontmatter, ctx.workflowData, ctx.engineSetup.importsResult); err != nil {
		return err
	}
	c.injectOTLPConfig(ctx.workflowData)
	if len(ctx.engineSetup.importsResult.MergedFeatures) == 0 {
		return nil
	}
	mergedFeatures, err := c.MergeFeatures(ctx.workflowData.Features, ctx.engineSetup.importsResult.MergedFeatures)
	if err != nil {
		return fmt.Errorf("failed to merge features from imports: %w", err)
	}
	ctx.workflowData.Features = mergedFeatures
	return nil
}

// mergeImportedObservability merges imported OTLP config into raw frontmatter with main precedence.
func (c *Compiler) mergeImportedObservability(workflowData *WorkflowData, mergedObservability string) {
	if mergedObservability == "" {
		return
	}
	var importedObs map[string]any
	if err := json.Unmarshal([]byte(mergedObservability), &importedObs); err != nil {
		orchestratorWorkflowLog.Printf("Skipping imported observability merge: invalid JSON: %v", err)
		return
	}
	mainObs := extractRawObservabilityMap(workflowData.RawFrontmatter)
	mergedEndpoints, mainCount, importAdded := mergeRawOTLPEndpoints(mainObs, importedObs)
	mergedAttrs := mergeOTLPStringMaps(
		extractOTLPCustomAttributesFromObsMap(mainObs),
		extractOTLPCustomAttributesFromObsMap(importedObs),
	)
	mergedResourceAttrs := mergeOTLPStringMaps(
		extractOTLPResourceAttributesFromObsMap(mainObs),
		extractOTLPResourceAttributesFromObsMap(importedObs),
	)
	githubApp := extractRawOTLPGitHubAppMap(mainObs)
	if githubApp == nil {
		githubApp = extractRawOTLPGitHubAppMap(importedObs)
	}
	applyMergedRawObservability(
		workflowData.RawFrontmatter,
		mergedEndpoints,
		mergedAttrs,
		mergedResourceAttrs,
		githubApp,
		mainCount,
		importAdded,
	)
}

func extractRawObservabilityMap(rawFrontmatter map[string]any) map[string]any {
	if rawFrontmatter == nil {
		return nil
	}
	obs, _ := rawFrontmatter["observability"].(map[string]any)
	return obs
}

func mergeRawOTLPEndpoints(mainObs map[string]any, importedObs map[string]any) (mergedEndpoints []any, mainCount int, importAdded int) {
	seen := make(map[string]struct {
	})
	for _, ep := range extractRawOTLPEndpointMaps(mainObs) {
		if url, _ := ep["url"].(string); url != "" && !setutil.Contains(seen, url) {
			seen[url] = struct {
			}{}
			mergedEndpoints = append(mergedEndpoints, ep)
		}
	}
	mainCount = len(mergedEndpoints)
	for _, ep := range extractRawOTLPEndpointMaps(importedObs) {
		if url, _ := ep["url"].(string); url != "" && !setutil.Contains(seen, url) {
			seen[url] = struct {
			}{}
			mergedEndpoints = append(mergedEndpoints, ep)
			importAdded++
		}
	}
	return mergedEndpoints, mainCount, importAdded
}

func applyMergedRawObservability(
	rawFrontmatter map[string]any,
	mergedEndpoints []any,
	mergedAttrs map[string]string,
	mergedResourceAttrs map[string]string,
	githubApp map[string]any,
	mainCount int,
	importAdded int,
) {
	if len(mergedEndpoints) == 0 && len(mergedAttrs) == 0 && len(mergedResourceAttrs) == 0 && githubApp == nil {
		return
	}
	newOTLP := map[string]any{}
	if len(mergedEndpoints) > 0 {
		newOTLP["endpoint"] = mergedEndpoints
	}
	if len(mergedAttrs) > 0 {
		newOTLP["attributes"] = mergedAttrs
	}
	if len(mergedResourceAttrs) > 0 {
		newOTLP["resource-attributes"] = mergedResourceAttrs
	}
	if githubApp != nil {
		newOTLP["github-app"] = githubApp
	}
	rawFrontmatter["observability"] = map[string]any{"otlp": newOTLP}
	orchestratorWorkflowLog.Printf("Merged OTLP endpoints into RawFrontmatter: %d from main workflow, %d from imports (%d total)", mainCount, importAdded, len(mergedEndpoints))
	if len(mergedAttrs) > 0 {
		orchestratorWorkflowLog.Printf("Merged %d custom OTLP attributes into RawFrontmatter", len(mergedAttrs))
	}
	if len(mergedResourceAttrs) > 0 {
		orchestratorWorkflowLog.Printf("Merged %d OTLP resource attributes into RawFrontmatter", len(mergedResourceAttrs))
	}
}

func (c *Compiler) mergeWorkflowEnv(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) error {
	topEnv := ExtractMapField(frontmatter, "env")
	if importsResult.MergedEnv == "" {
		setMainWorkflowEnvSources(workflowData, topEnv)
		return nil
	}
	mergedEnvMap, err := mergeEnv(topEnv, importsResult.MergedEnv)
	if err != nil {
		return fmt.Errorf("failed to merge env from imports: %w", err)
	}
	if len(mergedEnvMap) == 0 {
		return nil
	}
	workflowData.Env = c.extractTopLevelYAMLSection(map[string]any{"env": mergedEnvMap}, "env")
	workflowData.EnvSources = buildMergedEnvSources(mergedEnvMap, topEnv, importsResult.MergedEnvSources)
	return nil
}

func setMainWorkflowEnvSources(workflowData *WorkflowData, topEnv map[string]any) {
	if len(topEnv) == 0 {
		return
	}
	envSources := make(map[string]string, len(topEnv))
	for key := range topEnv {
		envSources[key] = "(main workflow)"
	}
	workflowData.EnvSources = envSources
}

func buildMergedEnvSources(mergedEnv map[string]any, topEnv map[string]any, importedSources map[string]string) map[string]string {
	envSources := make(map[string]string, len(mergedEnv))
	for key := range mergedEnv {
		if _, inTop := topEnv[key]; inTop {
			envSources[key] = "(main workflow)"
		} else if src, ok := importedSources[key]; ok {
			envSources[key] = src
		}
	}
	return envSources
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
	toolsConfig, err := ParseToolsConfig(tools)
	if err != nil {
		return err
	}
	if err := c.extractMemoryConfigurations(tools, toolsConfig, workflowData); err != nil {
		return err
	}
	c.extractCommandAndTriggerConfigurations(frontmatter, workflowData, importsResult)
	workflowData.SafeOutputs = safeOutputs
	allSafeOutputsConfigs, err := c.extractSafeOutputConfigurations(frontmatter, markdownDir, markdown, toolsConfig, workflowData, importsResult)
	if err != nil {
		return err
	}
	applyDefaultThreatDetectionFromImports(safeOutputs, workflowData, allSafeOutputsConfigs)
	applyDefaultCreateIssue(workflowData)
	applyTopLevelGitHubAppFallbacks(workflowData)
	return c.extractExperimentAndEvalConfigurations(frontmatter, workflowData)
}

func (c *Compiler) extractMemoryConfigurations(tools map[string]any, toolsConfig *ToolsConfig, workflowData *WorkflowData) error {
	cacheMemoryConfig, err := c.extractCacheMemoryConfigFromMap(tools)
	if err != nil {
		return err
	}
	workflowData.CacheMemoryConfig = cacheMemoryConfig
	repoMemoryConfig, err := c.extractRepoMemoryConfig(toolsConfig, workflowData.WorkflowID)
	if err != nil {
		return err
	}
	workflowData.RepoMemoryConfig = repoMemoryConfig
	return nil
}

func (c *Compiler) extractCommandAndTriggerConfigurations(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	workflowData.Command, workflowData.CommandEvents, workflowData.CommandCentralized, workflowData.CommandPlaceholder = c.extractCommandConfig(frontmatter)
	workflowData.LabelCommand, workflowData.LabelCommandEvents, workflowData.LabelCommandDecentralized, workflowData.LabelCommandRemoveLabel = c.extractLabelCommandConfig(frontmatter)
	workflowData.Jobs = c.extractJobsFromFrontmatter(frontmatter)
	if importsResult.MergedJobs != "" && importsResult.MergedJobs != "{}" {
		workflowData.Jobs = c.mergeJobsFromYAMLImports(workflowData.Jobs, importsResult.MergedJobs)
	}
	workflowData.Roles = c.extractRoles(frontmatter)
	workflowData.Bots = expandBotNames(mergeBots(c.extractBots(frontmatter), importsResult.MergedBots))
	workflowData.LabelNames = c.extractLabelNames(frontmatter)
	workflowData.RateLimit = c.extractRateLimitConfig(frontmatter)
	workflowData.SkipRoles = mergeSkipRoles(c.extractSkipRoles(frontmatter), importsResult.MergedSkipRoles)
	workflowData.SkipBots = expandBotNames(mergeSkipBots(c.extractSkipBots(frontmatter), importsResult.MergedSkipBots))
	workflowData.SkipAuthorAssociations = c.extractSkipAuthorAssociations(frontmatter)
	workflowData.AllowBotAuthoredTriggerComment = c.extractAllowBotAuthoredTriggerComment(frontmatter)
	workflowData.ActivationGitHubToken = c.resolveActivationGitHubToken(frontmatter, importsResult)
	workflowData.ActivationGitHubApp = c.resolveActivationGitHubApp(frontmatter, importsResult)
	workflowData.TopLevelGitHubApp = resolveTopLevelGitHubApp(frontmatter, importsResult)
}

func (c *Compiler) extractSafeOutputConfigurations(frontmatter map[string]any, markdownDir, markdown string, toolsConfig *ToolsConfig, workflowData *WorkflowData, importsResult *parser.ImportsResult) ([]string, error) {
	attachCommentMemoryConfig(c.extractCommentMemoryConfig(toolsConfig), workflowData)
	workflowData.MCPScripts = c.extractMCPScriptsConfig(frontmatter)
	if len(importsResult.MergedMCPScripts) > 0 {
		workflowData.MCPScripts = c.mergeMCPScripts(workflowData.MCPScripts, importsResult.MergedMCPScripts)
	}
	includedSafeOutputsConfigs, err := parser.ExpandIncludesForSafeOutputs(markdown, markdownDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes for safe-outputs: %w", err)
	}
	allSafeOutputsConfigs := append([]string{}, importsResult.MergedSafeOutputs...)
	allSafeOutputsConfigs = append(allSafeOutputsConfigs, includedSafeOutputsConfigs...)
	if err := c.applyIncludedSafeOutputConfigurations(frontmatter, workflowData, allSafeOutputsConfigs); err != nil {
		return nil, err
	}
	return allSafeOutputsConfigs, nil
}

func attachCommentMemoryConfig(commentMemoryConfig *CommentMemoryConfig, workflowData *WorkflowData) {
	if commentMemoryConfig == nil {
		return
	}
	if workflowData.SafeOutputs == nil {
		workflowData.SafeOutputs = &SafeOutputsConfig{}
	}
	workflowData.SafeOutputs.CommentMemory = commentMemoryConfig
}

func (c *Compiler) applyIncludedSafeOutputConfigurations(frontmatter map[string]any, workflowData *WorkflowData, allSafeOutputsConfigs []string) error {
	includedSafeJobs, err := c.mergeSafeJobsFromIncludedConfigs(extractSafeJobsFromFrontmatter(frontmatter), allSafeOutputsConfigs)
	if err != nil {
		return fmt.Errorf("failed to merge safe-jobs from includes: %w", err)
	}
	includedApp, err := c.mergeAppFromIncludedConfigs(workflowData.SafeOutputs, allSafeOutputsConfigs)
	if err != nil {
		return fmt.Errorf("failed to merge app from includes: %w", err)
	}
	if workflowData.SafeOutputs == nil && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs = &SafeOutputsConfig{}
	}
	if workflowData.SafeOutputs != nil && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs.Jobs = includedSafeJobs
	}
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.GitHubApp == nil && includedApp != nil {
		workflowData.SafeOutputs.GitHubApp = includedApp
	}
	rawSafeOutputsMap, _ := frontmatter["safe-outputs"].(map[string]any)
	mergedSafeOutputs, err := c.MergeSafeOutputs(workflowData.SafeOutputs, allSafeOutputsConfigs, rawSafeOutputsMap)
	if err != nil {
		return fmt.Errorf("failed to merge safe-outputs from imports: %w", err)
	}
	workflowData.SafeOutputs = mergedSafeOutputs
	return nil
}

func applyDefaultThreatDetectionFromImports(safeOutputs *SafeOutputsConfig, workflowData *WorkflowData, allSafeOutputsConfigs []string) {
	if safeOutputs == nil && workflowData.SafeOutputs != nil && workflowData.SafeOutputs.ThreatDetection == nil {
		if !isThreatDetectionExplicitlyDisabledInConfigs(allSafeOutputsConfigs) {
			orchestratorWorkflowLog.Print("Applying default threat-detection for safe-outputs assembled from imports/includes")
			workflowData.SafeOutputs.ThreatDetection = &ThreatDetectionConfig{}
		}
	}
}

func (c *Compiler) extractExperimentAndEvalConfigurations(frontmatter map[string]any, workflowData *WorkflowData) error {
	workflowData.ExperimentConfigs = extractExperimentConfigsFromFrontmatter(frontmatter)
	workflowData.Experiments = experimentVariantsFromConfigs(workflowData.ExperimentConfigs)
	workflowData.ExperimentsStorage = extractExperimentsStorageFromFrontmatter(frontmatter)
	evalsConfig, err := c.parseEvalsFromFrontmatter(frontmatter)
	if err != nil {
		return fmt.Errorf("invalid evals configuration: %w", err)
	}
	workflowData.Evals = evalsConfig
	if err := validateExperimentMetricReferences(workflowData.ExperimentConfigs, workflowData.Evals); err != nil {
		return fmt.Errorf("invalid experiments configuration: %w", err)
	}
	return nil
}

// mergeImportedOnFields copies import-safe on.* fields from imports into the main workflow frontmatter.
// Top-level on.* fields in the main workflow always take precedence.
func (c *Compiler) mergeImportedOnFields(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) error {
	if importsResult == nil {
		return nil
	}

	onMap := ensureOnMap(frontmatter)
	if onMap == nil {
		return nil
	}

	if _, exists := onMap["skip-if-match"]; !exists && importsResult.MergedSkipIfMatch != "" {
		var value any
		if err := json.Unmarshal([]byte(importsResult.MergedSkipIfMatch), &value); err != nil {
			return fmt.Errorf("failed to parse imported on.skip-if-match value: %w", err)
		}
		onMap["skip-if-match"] = value
		if workflowData != nil && workflowData.ParsedFrontmatter != nil {
			if workflowData.ParsedFrontmatter.On == nil {
				workflowData.ParsedFrontmatter.On = make(map[string]any)
			}
			workflowData.ParsedFrontmatter.On["skip-if-match"] = value
		}
	}

	if _, exists := onMap["skip-if-no-match"]; !exists && importsResult.MergedSkipIfNoMatch != "" {
		var value any
		if err := json.Unmarshal([]byte(importsResult.MergedSkipIfNoMatch), &value); err != nil {
			return fmt.Errorf("failed to parse imported on.skip-if-no-match value: %w", err)
		}
		onMap["skip-if-no-match"] = value
		if workflowData != nil && workflowData.ParsedFrontmatter != nil {
			if workflowData.ParsedFrontmatter.On == nil {
				workflowData.ParsedFrontmatter.On = make(map[string]any)
			}
			workflowData.ParsedFrontmatter.On["skip-if-no-match"] = value
		}
	}

	return nil
}

func ensureOnMap(frontmatter map[string]any) map[string]any {
	if frontmatter == nil {
		return nil
	}
	onValue, exists := frontmatter["on"]
	if !exists {
		on := make(map[string]any)
		frontmatter["on"] = on
		return on
	}
	onMap, ok := onValue.(map[string]any)
	if ok {
		return onMap
	}
	return nil
}

// processOnSectionAndFilters processes the on section configuration and applies various filters
func (c *Compiler) processOnSectionAndFilters(
	frontmatter map[string]any,
	workflowData *WorkflowData,
	cleanPath string,
) error {
	orchestratorWorkflowLog.Print("Processing on section and filters")
	if err := c.processOnFilterConfigurations(frontmatter, workflowData, cleanPath); err != nil {
		return err
	}
	if err := c.parseOnSection(frontmatter, workflowData, cleanPath); err != nil {
		return err
	}
	if err := c.applyDefaults(workflowData, cleanPath); err != nil {
		return err
	}
	c.applyPullRequestDraftFilter(workflowData, frontmatter)
	c.applyPullRequestForkFilter(workflowData, frontmatter)
	c.applyLabelFilter(workflowData, frontmatter)
	if err := c.extractOnWorkflowControls(frontmatter, workflowData); err != nil {
		return err
	}
	return nil
}

func (c *Compiler) processOnFilterConfigurations(frontmatter map[string]any, workflowData *WorkflowData, cleanPath string) error {
	if err := c.processStopAfterConfiguration(frontmatter, workflowData, cleanPath); err != nil {
		return err
	}
	if err := c.processSkipIfMatchConfiguration(frontmatter, workflowData); err != nil {
		return err
	}
	if err := c.processSkipIfNoMatchConfiguration(frontmatter, workflowData); err != nil {
		return err
	}
	if err := c.processSkipIfCheckFailingConfiguration(frontmatter, workflowData); err != nil {
		return err
	}
	return c.processManualApprovalConfiguration(frontmatter, workflowData)
}

func (c *Compiler) extractOnWorkflowControls(frontmatter map[string]any, workflowData *WorkflowData) error {
	onSteps, err := extractOnSteps(frontmatter)
	if err != nil {
		return err
	}
	if err := applyActionPinsToOnSteps(onSteps, workflowData); err != nil {
		return err
	}
	workflowData.OnSteps = onSteps
	workflowData.OnPermissions = extractOnPermissions(frontmatter)
	onNeeds, err := extractOnNeeds(frontmatter)
	if err != nil {
		return err
	}
	workflowData.OnNeeds = onNeeds
	onRestoreMemory, err := extractOnRestoreMemory(frontmatter)
	if err != nil {
		return err
	}
	workflowData.OnRestoreMemory = onRestoreMemory
	return nil
}

func applyActionPinsToOnSteps(onSteps []map[string]any, workflowData *WorkflowData) error {
	if len(onSteps) == 0 {
		return nil
	}
	anySteps := make([]any, len(onSteps))
	for i, s := range onSteps {
		anySteps[i] = s
	}
	typedSteps, convErr := SliceToSteps(anySteps)
	if convErr != nil {
		orchestratorWorkflowLog.Printf("Failed to convert on.steps to typed steps for action pinning: %v", convErr)
		return nil
	}
	typedSteps, convErr = applyActionPinsToTypedSteps(typedSteps, workflowData)
	if convErr != nil {
		return fmt.Errorf("on.steps: %w", convErr)
	}
	for i, s := range typedSteps {
		onSteps[i] = s.ToMap()
	}
	return nil
}
