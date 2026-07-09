package workflow

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var compilerMainJobLog = logger.New("workflow:compiler_main_job")

func isBuiltinJobName(jobName string) bool {
	_, isBuiltIn := constants.KnownBuiltInJobNames[jobName]
	return isBuiltIn
}

// buildMainJob creates the main agent job that runs the AI agent with the configured engine and tools.
// This job depends on the activation job if it exists, and handles the main workflow logic.
func (c *Compiler) buildMainJob(data *WorkflowData, activationJobCreated bool) (*Job, error) {
	workflowLog.Printf("Building main job for workflow: %s", data.Name)
	customJobsBeforeActivation := c.getCustomJobsDependingOnPreActivation(data.Jobs)

	steps, err := c.buildMainJobInitialSteps(data)
	if err != nil {
		return nil, err
	}

	jobCondition := c.buildMainJobCondition(data, activationJobCreated, customJobsBeforeActivation)
	depends, engineEnvContent := c.buildMainJobDependencies(data, activationJobCreated)
	c.warnMainJobBuiltinEnvRefs(depends, engineEnvContent)

	outputs := c.buildMainJobOutputs(data)
	env := c.buildMainJobEnv(data)

	permissions, err := c.buildMainJobPermissions(data)
	if err != nil {
		return nil, err
	}

	// In script mode, explicitly add a cleanup step (mirrors post.js in dev/release/action mode).
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}

	return &Job{
		Name:        string(constants.AgentJobName),
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Container:   c.indentYAMLLines(data.Container, "    "),
		Services:    c.indentYAMLLines(data.Services, "    "),
		Permissions: c.indentYAMLLines(permissions, "    "),
		Concurrency: c.indentYAMLLines(GenerateJobConcurrencyConfig(data), "    "),
		Env:         env,
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}, nil
}

// buildMainJobInitialSteps generates the initial steps for the main job: setup action, runtime
// paths, and the main job step content.
func (c *Compiler) buildMainJobInitialSteps(data *WorkflowData) ([]string, error) {
	var steps []string

	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		// Pass activation's trace ID so all agent spans share the same OTLP trace
		agentTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		agentParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, agentTraceID, agentParentSpanID)...)
	}

	// Set runtime paths that depend on RUNNER_TEMP via $GITHUB_ENV.
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

// buildMainJobCondition computes the if condition for the main agent job.
func (c *Compiler) buildMainJobCondition(data *WorkflowData, activationJobCreated bool, customJobsBeforeActivation []string) string {
	jobCondition := data.If
	if activationJobCreated {
		// If the if condition references custom jobs that run before activation,
		// the activation job handles the condition, so clear it here.
		if c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			jobCondition = "" // Activation job handles this condition
		} else if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			jobCondition = "" // Main job depends on activation job, so no inline condition needed
		}
		// Note: If data.If references custom jobs that DON'T depend on pre_activation,
		// we keep the condition on the agent job.
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

// buildMainJobDependencies computes the needs list for the main agent job, returning it alongside
// the engine.env content string used for built-in job reference warnings.
func (c *Compiler) buildMainJobDependencies(data *WorkflowData, activationJobCreated bool) ([]string, string) {
	var depends []string
	if activationJobCreated {
		depends = []string{string(constants.ActivationJobName)}
	}

	// Add custom jobs as direct dependencies only if they don't depend on pre_activation or agent.
	// Jobs that depend on pre_activation are handled through activation transitively.
	// Jobs that depend on agent are post-execution jobs and should run AFTER the agent job.
	if data.Jobs != nil {
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
	}

	return c.expandMainJobDepsFromContent(data, depends)
}

// expandMainJobDepsFromContent scans markdown content and engine.env for needs.<job>.outputs.*
// references and adds any referenced custom jobs as direct dependencies.
func (c *Compiler) expandMainJobDepsFromContent(data *WorkflowData, depends []string) ([]string, string) {
	// IMPORTANT: Even though jobs that depend on pre_activation are transitively accessible
	// through the activation job, if the workflow content directly references their outputs
	// (e.g., ${{ needs.search_issues.outputs.* }}), they MUST be direct dependencies.
	var contentBuilder strings.Builder
	contentBuilder.WriteString(data.MarkdownContent)
	if data.CustomSteps != "" {
		contentBuilder.WriteByte('\n')
		contentBuilder.WriteString(data.CustomSteps)
	}

	// Compute engine.env content once; reuse for dependency scan and built-in job ref warning.
	var engineEnvContent string
	if data.EngineConfig != nil && len(data.EngineConfig.Env) > 0 {
		var engineEnvBuilder strings.Builder
		for _, envValue := range data.EngineConfig.Env {
			engineEnvBuilder.WriteByte('\n')
			engineEnvBuilder.WriteString(envValue)
		}
		engineEnvContent = engineEnvBuilder.String()
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

// warnMainJobBuiltinEnvRefs warns when engine.env values reference built-in job names in
// needs expressions that will silently evaluate to empty strings at runtime.
func (c *Compiler) warnMainJobBuiltinEnvRefs(depends []string, engineEnvContent string) {
	if engineEnvContent == "" {
		return
	}
	builtinNames := sliceutil.SortedKeys(constants.KnownBuiltInJobNames)
	builtinsWarned := make(map[string]struct{})
	for _, builtinJobName := range builtinNames {
		// Skip built-ins that are already direct dependencies (e.g., activation) —
		// their outputs are accessible and the expression is valid.
		if slices.Contains(depends, builtinJobName) {
			continue
		}
		if !setutil.Contains(builtinsWarned, builtinJobName) && strings.Contains(engineEnvContent, fmt.Sprintf("needs.%s.", builtinJobName)) {
			builtinsWarned[builtinJobName] = struct{}{}
			warningMsg := fmt.Sprintf(
				"engine.env references built-in job '%s' in a needs expression. "+
					"Built-in jobs are managed by the compiler and cannot be added as direct agent dependencies; "+
					"this expression will silently evaluate to an empty string at runtime.",
				builtinJobName,
			)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
			c.IncrementWarningCount()
		}
	}
}

// buildMainJobBaseOutputs returns the baseline outputs map common to all agent jobs.
func buildMainJobBaseOutputs() map[string]string {
	return map[string]string{
		"model":            "${{ needs.activation.outputs.model }}",
		"effective_tokens": fmt.Sprintf("${{ steps.%s.outputs.effective_tokens }}", constants.ParseMCPGatewayStepID),
		// aic is the total AI Credits cost for the run (1 AIC == 0.01 USD).
		"aic": fmt.Sprintf("${{ steps.%s.outputs.aic }}", constants.ParseMCPGatewayStepID),
		// ambient_context: input_tokens + (cache_tokens / 10), first-request context size metric.
		"ambient_context":             fmt.Sprintf("${{ steps.%s.outputs.ambient_context }}", constants.ParseMCPGatewayStepID),
		"ai_credits_rate_limit_error": fmt.Sprintf("${{ steps.%s.outputs.ai_credits_rate_limit_error || 'false' }}", constants.ParseMCPGatewayStepID),
		// unknown_model_ai_credits is true when the AWF API proxy rejects a request because the
		// model is not in the built-in pricing table and maxAiCredits is active.
		"unknown_model_ai_credits": fmt.Sprintf("${{ steps.%s.outputs.unknown_model_ai_credits || 'false' }}", constants.ParseMCPGatewayStepID),
		// setup-trace-id propagates the shared OTLP trace ID to downstream jobs.
		"setup-trace-id": "${{ steps.setup.outputs.trace-id }}",
		// setup-span-id propagates the setup span parent so downstream setup spans form one tree.
		"setup-span-id":        "${{ steps.setup.outputs.span-id }}",
		"setup-parent-span-id": "${{ steps.setup.outputs.parent-span-id || steps.setup.outputs.span-id }}",
	}
}

// addMainJobEngineOutputs adds engine-specific error detection outputs to the outputs map.
func (c *Compiler) addMainJobEngineOutputs(data *WorkflowData, outputs map[string]string) {
	engine, err := c.getAgenticEngine(data.AI)
	if err != nil || engine.GetErrorDetectionScriptId() == "" {
		return
	}
	stepRef := fmt.Sprintf("steps.%s.outputs", constants.DetectAgentErrorsStepID)
	for _, name := range []string{
		"inference_access_error",
		"mcp_policy_error",
		"agentic_engine_timeout",
		"model_not_supported_error",
		"http_400_response_error",
	} {
		outputs[name] = fmt.Sprintf("${{ %s.%s || 'false' }}", stepRef, name)
		compilerMainJobLog.Printf("Added %s output (engine=%s, step=%s)", name, engine.GetID(), constants.DetectAgentErrorsStepID)
	}
}

// buildMainJobOutputs builds the complete outputs map for the main agent job.
func (c *Compiler) buildMainJobOutputs(data *WorkflowData) map[string]string {
	outputs := buildMainJobBaseOutputs()

	// Propagate artifact prefix so downstream jobs can access it without depending on activation.
	if hasWorkflowCallTrigger(data.On) {
		outputs[constants.ArtifactPrefixOutputName] = "${{ needs.activation.outputs.artifact_prefix }}"
		compilerMainJobLog.Print("Added artifact_prefix output to agent job (workflow_call context)")
	}

	if data.SafeOutputs != nil {
		outputs["output"] = "${{ steps.collect_output.outputs.output }}"
		outputs["output_types"] = "${{ steps.collect_output.outputs.output_types }}"
		outputs["has_patch"] = "${{ steps.collect_output.outputs.has_patch }}"
	}

	// checkout_pr_success tracks PR checkout status for conclusion job failure handling.
	if ShouldGeneratePRCheckoutStep(data) {
		outputs["checkout_pr_success"] = "${{ steps.checkout-pr.outputs.checkout_pr_success || 'true' }}"
		compilerMainJobLog.Print("Added checkout_pr_success output (workflow has contents read access)")
	} else {
		compilerMainJobLog.Print("Skipped checkout_pr_success output (workflow lacks contents read access)")
	}

	// Expose restore step outputs so downstream failure handling can compute cache hit status.
	if data.CacheMemoryConfig != nil && len(data.CacheMemoryConfig.Caches) > 0 {
		for i := range data.CacheMemoryConfig.Caches {
			stepID := fmt.Sprintf("restore_cache_memory_%d", i)
			outputs[fmt.Sprintf("cache_memory_restore_%d_matched_key", i)] = fmt.Sprintf("${{ steps.%s.outputs.cache-matched-key || '' }}", stepID)
			outputs[fmt.Sprintf("cache_memory_restore_%d_cache_hit", i)] = fmt.Sprintf("${{ steps.%s.outputs.cache-hit || 'false' }}", stepID)
		}
	}

	c.addMainJobEngineOutputs(data, outputs)
	return outputs
}

// buildMainJobEnv builds the job-level environment variable map for the main agent job.
func (c *Compiler) buildMainJobEnv(data *WorkflowData) map[string]string {
	var env map[string]string

	if data.SafeOutputs != nil {
		env = make(map[string]string)
		env["GH_AW_MCP_LOG_DIR"] = constants.TmpMcpLogsSafeOutputsDir
		// Note: GH_AW_SAFE_OUTPUTS, GH_AW_SAFE_OUTPUTS_CONFIG_PATH, and
		// GH_AW_SAFE_OUTPUTS_TOOLS_PATH are set via a run step (generateSetRuntimePathsStep)
		// because the runner context is not available in job-level env: blocks.

		// Asset-related env vars must always be set (even empty) for awmg v0.0.12+ validation.
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

	// GH_AW_WORKFLOW_ID_SANITIZED is used in cache-memory keys.
	if data.WorkflowID != "" {
		if env == nil {
			env = make(map[string]string)
		}
		env["GH_AW_WORKFLOW_ID_SANITIZED"] = SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	}

	// Bake the repository project UTC offset into job env so runtime JS helpers
	// do not need to read aw.json on the runner.
	if utcOffset := c.getCompiledProjectUTCOffset(); utcOffset != "" {
		if env == nil {
			env = make(map[string]string)
		}
		env["GH_AW_PROJECT_UTC"] = fmt.Sprintf("%q", utcOffset)
	}

	return env
}

// buildMainJobPermissions computes the final permissions string for the main agent job, applying
// automatic augmentation in dev/script mode and inferring permissions from shell scripts.
func (c *Compiler) buildMainJobPermissions(data *WorkflowData) (string, error) {
	// GitHub App-only permissions (e.g., members, administration) must be filtered out before
	// rendering to the job-level permissions block.
	permissions := filterJobLevelPermissions(data.Permissions, data.CachedPermissions)

	// In dev/script mode, automatically add contents: read if the actions folder checkout is needed.
	needsContentsRead := (c.actionMode.IsDev() || c.actionMode.IsScript()) && len(c.generateCheckoutActionsFolder(data)) > 0
	if needsContentsRead {
		if permissions == "" {
			permissions = NewPermissionsContentsRead().RenderToYAML()
		} else {
			parser := NewPermissionsParser(permissions)
			perms := parser.ToPermissions()
			if level, exists := perms.Get(PermissionContents); !exists || level == PermissionNone {
				perms.Set(PermissionContents, PermissionRead)
				permissions = perms.RenderToYAML()
			}
		}
	}

	scripts := c.extractAgentJobScripts(data)
	return c.inferMainJobPermissions(data, permissions, scripts)
}

// extractAgentJobScripts collects all shell scripts from agent job step sections that must be
// scanned for write commands and permission inference.
func (c *Compiler) extractAgentJobScripts(data *WorkflowData) []string {
	agentJobName := string(constants.AgentJobName)
	scripts := extractRunScriptsFromSectionYAML(data.PreSteps, "pre-steps")
	scripts = append(scripts, extractRunScriptsFromSectionYAML(data.CustomSteps, "steps")...)
	scripts = append(scripts, extractRunScriptsFromSectionYAML(data.PreAgentSteps, "pre-agent-steps")...)
	scripts = append(scripts, extractRunScriptsFromSectionYAML(data.PostSteps, "post-steps")...)
	if data.Jobs != nil {
		// For built-in jobs, only setup-steps and pre-steps are injected by applyBuiltinJobPreSteps.
		scripts = append(scripts, extractRunScriptsFromJobSection(data.Jobs, agentJobName, "setup-steps")...)
		scripts = append(scripts, extractRunScriptsFromJobSection(data.Jobs, agentJobName, "pre-steps")...)
	}
	return scripts
}

// inferMainJobPermissions detects write commands in shell scripts and merges inferred read
// permissions into the existing permissions block.
func (c *Compiler) inferMainJobPermissions(data *WorkflowData, permissions string, scripts []string) (string, error) {
	if len(scripts) == 0 {
		return permissions, nil
	}
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
	// Infer read permissions unless the user explicitly zeroed out all permissions.
	// Uses the same exact-string check as tools.go (YAML parser always normalizes
	// "permissions: {}" to this canonical form when parsing the frontmatter).
	if data.Permissions != "permissions: {}" && permissions != "" {
		inferred, err := inferPermissionsFromShellScripts(scripts)
		if err != nil {
			return "", err
		}
		if len(inferred) > 0 {
			permissions = mergeInferredIntoPermissionsYAML(permissions, inferred)
		}
	}
	return permissions, nil
}
