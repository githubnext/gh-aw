package workflow

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
)

// buildMainJobSetupSteps generates setup action steps for the main agent job.
func (c *Compiler) buildMainJobSetupSteps(data *WorkflowData) []string {
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" && !c.actionMode.IsScript() {
		return nil
	}
	var steps []string
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)
	agentTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
	agentParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
	steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, agentTraceID, agentParentSpanID)...)
	return steps
}

// buildMainJobSteps assembles all steps for the main agent job.
func (c *Compiler) buildMainJobSteps(data *WorkflowData) ([]string, error) {
	var steps []string
	steps = append(steps, c.buildMainJobSetupSteps(data)...)
	if data.SafeOutputs != nil {
		steps = append(steps, c.generateSetRuntimePathsStep()...)
	}
	var stepBuilder strings.Builder
	if err := c.generateMainJobSteps(&stepBuilder, data); err != nil {
		return nil, fmt.Errorf("failed to generate main job steps: %w", err)
	}
	if stepsContent := stepBuilder.String(); stepsContent != "" {
		steps = append(steps, stepsContent)
	}
	return steps, nil
}

// resolveAgentJobCondition computes the if: condition for the main agent job.
func (c *Compiler) resolveAgentJobCondition(data *WorkflowData, activationJobCreated bool) string {
	jobCondition := data.If
	if activationJobCreated {
		customJobsBeforeActivation := c.getCustomJobsDependingOnPreActivation(data.Jobs)
		if c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			jobCondition = ""
		} else if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			jobCondition = ""
		}
	}
	if activationJobCreated && hasMaxDailyAICGuardrail(data) {
		guard := &ExpressionNode{Expression: fmt.Sprintf("needs.%s.outputs.daily_ai_credits_exceeded != 'true'", constants.ActivationJobName)}
		if jobCondition == "" {
			jobCondition = RenderCondition(guard)
		} else {
			jobCondition = RenderCondition(BuildAnd(&ExpressionNode{Expression: stripExpressionWrapper(jobCondition)}, guard))
		}
	}
	return jobCondition
}

// buildDirectDependencies returns the initial needs list for the main agent job.
func (c *Compiler) buildDirectDependencies(data *WorkflowData, activationJobCreated bool) []string {
	var depends []string
	if activationJobCreated {
		depends = []string{string(constants.ActivationJobName)}
	}
	if data.Jobs == nil {
		return depends
	}
	for _, jobName := range sliceutil.SortedKeys(data.Jobs) {
		if isBuiltinJobName(jobName) {
			continue
		}
		if configMap, ok := data.Jobs[jobName].(map[string]any); ok {
			if !jobDependsOnPreActivation(configMap) && !jobDependsOnAgent(configMap) {
				depends = append(depends, jobName)
			}
		}
	}
	return depends
}

// augmentDependenciesFromContent adds direct dependencies for custom jobs referenced in
// workflow content or engine.env values, and returns the engine env content string.
func (c *Compiler) augmentDependenciesFromContent(data *WorkflowData, depends []string) ([]string, string) {
	var contentBuilder strings.Builder
	contentBuilder.WriteString(data.MarkdownContent)
	if data.CustomSteps != "" {
		contentBuilder.WriteByte('\n')
		contentBuilder.WriteString(data.CustomSteps)
	}
	var engineEnvContent string
	if data.EngineConfig != nil && len(data.EngineConfig.Env) > 0 {
		var b strings.Builder
		for _, envValue := range data.EngineConfig.Env {
			b.WriteByte('\n')
			b.WriteString(envValue)
		}
		engineEnvContent = b.String()
		contentBuilder.WriteString(engineEnvContent)
		compilerMainJobLog.Printf("Including %d engine.env values in agent job dependency scan", len(data.EngineConfig.Env))
	}
	for _, jobName := range c.getReferencedCustomJobs(contentBuilder.String(), data.Jobs) {
		if isBuiltinJobName(jobName) {
			continue
		}
		if !slices.Contains(depends, jobName) {
			depends = append(depends, jobName)
			compilerMainJobLog.Printf("Added direct dependency on custom job '%s' because it's referenced in workflow content or engine.env", jobName)
		}
	}
	return depends, engineEnvContent
}

// warnBuiltinEngineEnvRefs emits a warning when engine.env values reference built-in job
// outputs that are not direct dependencies of the agent job.
func (c *Compiler) warnBuiltinEngineEnvRefs(depends []string, engineEnvContent string) {
	if engineEnvContent == "" {
		return
	}
	builtinsWarned := make(map[string]struct{})
	for _, builtinJobName := range sliceutil.SortedKeys(constants.KnownBuiltInJobNames) {
		if slices.Contains(depends, builtinJobName) {
			continue
		}
		if !setutil.Contains(builtinsWarned, builtinJobName) && strings.Contains(engineEnvContent, fmt.Sprintf("needs.%s.", builtinJobName)) {
			builtinsWarned[builtinJobName] = struct{}{}
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf(
				"engine.env references built-in job '%s' in a needs expression. "+
					"Built-in jobs are managed by the compiler and cannot be added as direct agent dependencies; "+
					"this expression will silently evaluate to an empty string at runtime.",
				builtinJobName,
			)))
			c.IncrementWarningCount()
		}
	}
}

// buildCoreAgentOutputs creates the base outputs map for the main agent job.
func buildCoreAgentOutputs(data *WorkflowData) map[string]string {
	stepRef := func(name string) string {
		return fmt.Sprintf("${{ steps.%s.outputs.%s }}", constants.ParseMCPGatewayStepID, name)
	}
	outputs := map[string]string{
		"model":                       "${{ needs.activation.outputs.model }}",
		"effective_tokens":            stepRef("effective_tokens"),
		"aic":                         stepRef("aic"),
		"ambient_context":             stepRef("ambient_context"),
		"ai_credits_rate_limit_error": stepRef("ai_credits_rate_limit_error || 'false'"),
		"unknown_model_ai_credits":    stepRef("unknown_model_ai_credits || 'false'"),
		"setup-trace-id":              "${{ steps.setup.outputs.trace-id }}",
		"setup-span-id":               "${{ steps.setup.outputs.span-id }}",
		"setup-parent-span-id":        "${{ steps.setup.outputs.parent-span-id || steps.setup.outputs.span-id }}",
	}
	if hasWorkflowCallTrigger(data.On) {
		outputs[constants.ArtifactPrefixOutputName] = "${{ needs.activation.outputs.artifact_prefix }}"
		compilerMainJobLog.Print("Added artifact_prefix output to agent job (workflow_call context)")
	}
	return outputs
}

// addConditionalAgentOutputs appends safe-output, checkout, and cache-memory outputs.
func addConditionalAgentOutputs(data *WorkflowData, outputs map[string]string) {
	if data.SafeOutputs != nil {
		outputs["output"] = "${{ steps.collect_output.outputs.output }}"
		outputs["output_types"] = "${{ steps.collect_output.outputs.output_types }}"
		outputs["has_patch"] = "${{ steps.collect_output.outputs.has_patch }}"
	}
	if ShouldGeneratePRCheckoutStep(data) {
		outputs["checkout_pr_success"] = "${{ steps.checkout-pr.outputs.checkout_pr_success || 'true' }}"
		compilerMainJobLog.Print("Added checkout_pr_success output (workflow has contents read access)")
	} else {
		compilerMainJobLog.Print("Skipped checkout_pr_success output (workflow lacks contents read access)")
	}
	if data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0 {
		return
	}
	for i := range data.CacheMemoryConfig.Caches {
		stepID := fmt.Sprintf("restore_cache_memory_%d", i)
		outputs[fmt.Sprintf("cache_memory_restore_%d_matched_key", i)] = fmt.Sprintf("${{ steps.%s.outputs.cache-matched-key || '' }}", stepID)
		outputs[fmt.Sprintf("cache_memory_restore_%d_cache_hit", i)] = fmt.Sprintf("${{ steps.%s.outputs.cache-hit || 'false' }}", stepID)
	}
}

// addErrorDetectionOutputs appends engine error-detection step outputs when supported.
func (c *Compiler) addErrorDetectionOutputs(data *WorkflowData, outputs map[string]string) {
	engine, engineErr := c.getAgenticEngine(data.AI)
	if engineErr != nil || engine.GetErrorDetectionScriptId() == "" {
		return
	}
	stepRef := fmt.Sprintf("steps.%s.outputs", constants.DetectAgentErrorsStepID)
	for _, name := range []string{"inference_access_error", "mcp_policy_error", "agentic_engine_timeout", "model_not_supported_error", "http_400_response_error"} {
		outputs[name] = fmt.Sprintf("${{ %s.%s || 'false' }}", stepRef, name)
		compilerMainJobLog.Printf("Added %s output (engine=%s, step=%s)", name, engine.GetID(), constants.DetectAgentErrorsStepID)
	}
}

// buildAgentJobOutputs assembles the complete outputs map for the main agent job.
func (c *Compiler) buildAgentJobOutputs(data *WorkflowData) map[string]string {
	outputs := buildCoreAgentOutputs(data)
	addConditionalAgentOutputs(data, outputs)
	c.addErrorDetectionOutputs(data, outputs)
	return outputs
}

// buildAgentJobEnv constructs the job-level environment variables for the main agent job.
func (c *Compiler) buildAgentJobEnv(data *WorkflowData) map[string]string {
	var env map[string]string
	if data.SafeOutputs != nil {
		env = make(map[string]string)
		env["GH_AW_MCP_LOG_DIR"] = constants.TmpMcpLogsSafeOutputsDir
		if data.SafeOutputs.UploadAssets != nil {
			env["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
			env["GH_AW_ASSETS_MAX_SIZE_KB"] = strconv.Itoa(data.SafeOutputs.UploadAssets.MaxSizeKB)
			env["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
		} else {
			env["GH_AW_ASSETS_BRANCH"] = `""`
			env["GH_AW_ASSETS_MAX_SIZE_KB"] = "0"
			env["GH_AW_ASSETS_ALLOWED_EXTS"] = `""`
		}
		env["DEFAULT_BRANCH"] = "${{ github.event.repository.default_branch }}"
	}
	if data.WorkflowID != "" {
		if env == nil {
			env = make(map[string]string)
		}
		env["GH_AW_WORKFLOW_ID_SANITIZED"] = SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	}
	if utcOffset := c.getCompiledProjectUTCOffset(); utcOffset != "" {
		if env == nil {
			env = make(map[string]string)
		}
		env["GH_AW_PROJECT_UTC"] = fmt.Sprintf("%q", utcOffset)
	}
	return env
}

// collectAgentJobScripts gathers all run: scripts from agent job step sections.
func collectAgentJobScripts(data *WorkflowData) []string {
	agentJobName := string(constants.AgentJobName)
	scripts := extractRunScriptsFromSectionYAML(data.PreSteps, "pre-steps")
	scripts = append(scripts, extractRunScriptsFromSectionYAML(data.CustomSteps, "steps")...)
	scripts = append(scripts, extractRunScriptsFromSectionYAML(data.PreAgentSteps, "pre-agent-steps")...)
	scripts = append(scripts, extractRunScriptsFromSectionYAML(data.PostSteps, "post-steps")...)
	if data.Jobs != nil {
		scripts = append(scripts, extractRunScriptsFromJobSection(data.Jobs, agentJobName, "setup-steps")...)
		scripts = append(scripts, extractRunScriptsFromJobSection(data.Jobs, agentJobName, "pre-steps")...)
	}
	return scripts
}

// validateAndInferPermissions checks for disallowed write commands and infers read permissions.
func validateAndInferPermissions(data *WorkflowData, permissions string, scripts []string) (string, error) {
	writeCmds, err := detectWriteCommandsInShellScripts(scripts)
	if err != nil {
		return "", err
	}
	if len(writeCmds) > 0 {
		return "", fmt.Errorf(
			"agent job uses write gh command(s) [%s]; write operations are not permitted in agent job steps because the agent job runs with read-only permissions. Use safe-outputs for write operations. See: https://github.github.com/gh-aw/reference/safe-outputs/",
			strings.Join(writeCmds, ", "),
		)
	}
	if data.Permissions == "permissions: {}" || permissions == "" {
		return permissions, nil
	}
	inferred, err := inferPermissionsFromShellScripts(scripts)
	if err != nil {
		return "", err
	}
	if len(inferred) > 0 {
		permissions = mergeInferredIntoPermissionsYAML(permissions, inferred)
	}
	return permissions, nil
}

// applyContentsReadIfNeeded adds contents: read permission when the actions folder checkout is needed.
func (c *Compiler) applyContentsReadIfNeeded(data *WorkflowData, permissions string) string {
	needsContentsRead := (c.actionMode.IsDev() || c.actionMode.IsScript()) && len(c.generateCheckoutActionsFolder(data)) > 0
	if !needsContentsRead {
		return permissions
	}
	if permissions == "" {
		perms := NewPermissionsContentsRead()
		return perms.RenderToYAML()
	}
	parser := NewPermissionsParser(permissions)
	perms := parser.ToPermissions()
	if level, exists := perms.Get(PermissionContents); !exists || level == PermissionNone {
		perms.Set(PermissionContents, PermissionRead)
		return perms.RenderToYAML()
	}
	return permissions
}

// buildAgentJobPermissions computes the final permissions YAML for the main agent job.
func (c *Compiler) buildAgentJobPermissions(data *WorkflowData) (string, error) {
	permissions := filterJobLevelPermissions(data.Permissions, data.CachedPermissions)
	permissions = c.applyContentsReadIfNeeded(data, permissions)
	scripts := collectAgentJobScripts(data)
	if len(scripts) == 0 {
		return permissions, nil
	}
	return validateAndInferPermissions(data, permissions, scripts)
}
