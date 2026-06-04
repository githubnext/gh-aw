package workflow

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
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
	var steps []string

	// Add setup action steps at the beginning of the job
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)

		// Main job doesn't need project support (no safe outputs processed here)
		// Pass activation's trace ID so all agent spans share the same OTLP trace
		agentTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		agentParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, agentTraceID, agentParentSpanID)...)
	}

	// Set runtime paths that depend on RUNNER_TEMP via $GITHUB_ENV.
	// These cannot be set in job-level env: because the runner context is not
	// available there (only in step-level env: and run: blocks).
	if data.SafeOutputs != nil {
		steps = append(steps, c.generateSetRuntimePathsStep()...)
	}

	// Checkout .github folder is now done in activation job (before prompt generation)
	// This ensures the activation job has access to .github and .agents folders for runtime imports

	// Find custom jobs that depend on pre_activation - these are handled by the activation job
	customJobsBeforeActivation := c.getCustomJobsDependingOnPreActivation(data.Jobs)

	var jobCondition = data.If
	if activationJobCreated {
		// If the if condition references custom jobs that run before activation,
		// the activation job handles the condition, so clear it here
		if c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			jobCondition = "" // Activation job handles this condition
		} else if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			jobCondition = "" // Main job depends on activation job, so no need for inline condition
		}
		// Note: If data.If references custom jobs that DON'T depend on pre_activation,
		// we keep the condition on the agent job
	}
	if activationJobCreated && hasMaxDailyEffectiveTokensGuardrail(data) {
		guard := &ExpressionNode{Expression: fmt.Sprintf("needs.%s.outputs.daily_effective_workflow_exceeded != 'true'", constants.ActivationJobName)}
		if jobCondition == "" {
			jobCondition = RenderCondition(guard)
		} else {
			jobCondition = RenderCondition(BuildAnd(&ExpressionNode{Expression: stripExpressionWrapper(jobCondition)}, guard))
		}
	}

	// Note: workflow_run repository safety check is applied exclusively to activation job

	// Permission checks are now handled by the separate check_membership job
	// No role checks needed in the main job

	// Build step content using the generateMainJobSteps helper method
	// but capture it into a string instead of writing directly
	var stepBuilder strings.Builder
	if err := c.generateMainJobSteps(&stepBuilder, data); err != nil {
		return nil, fmt.Errorf("failed to generate main job steps: %w", err)
	}

	// Checkout app tokens (checkout-app-token-*) are now minted directly in the agent job,
	// for the same reason as the GitHub MCP App token: actions/create-github-app-token calls
	// ::add-mask:: on the produced token, and the GitHub Actions runner silently drops masked
	// values when used as job outputs (runner v2.308+). Minting within the agent job avoids
	// the activation→agent output hop entirely.
	stepsContent := stepBuilder.String()

	// Split the steps content into individual step entries
	if stepsContent != "" {
		steps = append(steps, stepsContent)
	}

	var depends []string
	if activationJobCreated {
		depends = []string{string(constants.ActivationJobName)} // Depend on the activation job only if it exists
	}

	// Add custom jobs as dependencies only if they don't depend on pre_activation or agent
	// Custom jobs that depend on pre_activation are now dependencies of activation,
	// so the agent job gets them transitively through activation
	// Custom jobs that depend on agent should run AFTER the agent job, not before it
	if data.Jobs != nil {
		for _, jobName := range slices.Sorted(maps.Keys(data.Jobs)) {
			// Skip built-in jobs as they are handled separately and should not become custom dependencies.
			if isBuiltinJobName(jobName) {
				continue
			}

			// Only add as direct dependency if it doesn't depend on pre_activation or agent
			// (jobs that depend on pre_activation are handled through activation)
			// (jobs that depend on agent are post-execution jobs like failure handlers)
			if configMap, ok := data.Jobs[jobName].(map[string]any); ok {
				if !jobDependsOnPreActivation(configMap) && !jobDependsOnAgent(configMap) {
					depends = append(depends, jobName)
				}
			}
		}
	}

	// IMPORTANT: Even though jobs that depend on pre_activation are transitively accessible
	// through the activation job, if the workflow content directly references their outputs
	// (e.g., ${{ needs.search_issues.outputs.* }}), we MUST add them as direct dependencies.
	// This is required for GitHub Actions expression evaluation and actionlint validation.
	// Also check custom steps from the frontmatter, which are also added to the agent job.
	// Also check engine.env values, which may contain needs.<job>.outputs.* expressions.
	var contentBuilder strings.Builder
	contentBuilder.WriteString(data.MarkdownContent)
	if data.CustomSteps != "" {
		contentBuilder.WriteByte('\n')
		contentBuilder.WriteString(data.CustomSteps)
	}
	// Compute engine.env content once; reuse for both the dependency scan and the built-in
	// job reference warning below.
	var engineEnvContent string
	if data.EngineConfig != nil && len(data.EngineConfig.Env) > 0 {
		var engineEnvBuilder strings.Builder
		for _, envValue := range data.EngineConfig.Env {
			engineEnvBuilder.WriteByte('\n')
			engineEnvBuilder.WriteString(envValue)
		}
		engineEnvContent = engineEnvBuilder.String()
		// Include engine.env values so that needs.<job>.outputs.* expressions there are also
		// scanned for custom job dependencies that must be added to the agent job's needs list.
		contentBuilder.WriteString(engineEnvContent)
		compilerMainJobLog.Printf("Including %d engine.env values in agent job dependency scan", len(data.EngineConfig.Env))
	}
	referencedJobs := c.getReferencedCustomJobs(contentBuilder.String(), data.Jobs)
	for _, jobName := range referencedJobs {
		// Skip built-in jobs as they are handled separately and should not become custom dependencies.
		if isBuiltinJobName(jobName) {
			continue
		}

		// Check if this job is already in depends
		alreadyDepends := slices.Contains(depends, jobName)
		// Add it if not already present
		if !alreadyDepends {
			depends = append(depends, jobName)
			compilerMainJobLog.Printf("Added direct dependency on custom job '%s' because it's referenced in workflow content or engine.env", jobName)
		}
	}

	// Warn when built-in job names appear in needs expressions inside engine.env values.
	// engine.env values are emitted as step-level environment variables in the agent job;
	// for a needs expression like ${{ needs.X.outputs.Y }} to evaluate correctly at runtime,
	// X must be a direct dependency of the agent job. Built-in jobs (e.g., detection,
	// safe_outputs) are managed by the compiler and cannot be added as direct dependencies,
	// so referencing them here will silently produce empty strings at runtime.
	// Exception: skip any built-in that is already in `depends` (e.g., `activation`),
	// as those expressions are valid and will evaluate correctly.
	if engineEnvContent != "" {
		builtinNames := make([]string, 0, len(constants.KnownBuiltInJobNames))
		for name := range constants.KnownBuiltInJobNames {
			builtinNames = append(builtinNames, name)
		}
		sort.Strings(builtinNames)
		builtinsWarned := make(map[string]bool)
		for _, builtinJobName := range builtinNames {
			// Skip built-ins that are already direct dependencies (e.g., activation) —
			// their outputs are accessible and the expression is valid.
			if slices.Contains(depends, builtinJobName) {
				continue
			}
			if !builtinsWarned[builtinJobName] && strings.Contains(engineEnvContent, fmt.Sprintf("needs.%s.", builtinJobName)) {
				builtinsWarned[builtinJobName] = true
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

	// Build outputs for all engines (GH_AW_SAFE_OUTPUTS functionality)
	// Build job outputs
	// Always include model output for reuse in other jobs - now sourced from activation job
	outputs := map[string]string{
		"model": "${{ needs.activation.outputs.model }}",
		// effective_tokens is the total ET for the run, captured by the MCP gateway log parser step.
		// It is exposed here so that the safe_outputs job can set GH_AW_EFFECTIVE_TOKENS and render
		// the {effective_tokens_suffix} template expression in footer templates.
		"effective_tokens": fmt.Sprintf("${{ steps.%s.outputs.effective_tokens }}", constants.ParseMCPGatewayStepID),
		// aic is the total AI Credits cost for the run (1 AIC == 0.01 USD), captured by the
		// MCP gateway log parser step and passed to downstream jobs for footer rendering.
		"aic": fmt.Sprintf("${{ steps.%s.outputs.aic }}", constants.ParseMCPGatewayStepID),
		// effective_tokens_rate_limit_error is true when MCP gateway logs indicate ET budget
		// exhaustion or API rate limiting attributable to ET constraints.
		"effective_tokens_rate_limit_error": fmt.Sprintf("${{ steps.%s.outputs.effective_tokens_rate_limit_error || 'false' }}", constants.ParseMCPGatewayStepID),
		// setup-trace-id propagates the shared OTLP trace ID to downstream jobs (detection, safe_outputs, cache, etc.)
		"setup-trace-id": "${{ steps.setup.outputs.trace-id }}",
		// setup-span-id propagates the setup span parent so downstream setup spans form one tree.
		"setup-span-id": "${{ steps.setup.outputs.span-id }}",
		// setup-parent-span-id propagates the global setup parent span ID across jobs.
		"setup-parent-span-id": "${{ steps.setup.outputs.parent-span-id || steps.setup.outputs.span-id }}",
	}

	// Note: secret_verification_result is now an output of the activation job (not the agent job).
	// The validate-secret step runs in the activation job, before context variable validation.

	// Propagate the artifact prefix from the activation job so that downstream jobs depending
	// only on the agent job (e.g. update_cache_memory, safe-jobs) can still access the prefix
	// without needing a direct dependency on the activation job.
	if hasWorkflowCallTrigger(data.On) {
		outputs[constants.ArtifactPrefixOutputName] = "${{ needs.activation.outputs.artifact_prefix }}"
		compilerMainJobLog.Print("Added artifact_prefix output to agent job (workflow_call context)")
	}

	// Add safe-output specific outputs if the workflow uses the safe-outputs feature
	if data.SafeOutputs != nil {
		outputs["output"] = "${{ steps.collect_output.outputs.output }}"
		outputs["output_types"] = "${{ steps.collect_output.outputs.output_types }}"
		outputs["has_patch"] = "${{ steps.collect_output.outputs.has_patch }}"
	}

	// Add checkout_pr_success output to track PR checkout status only if the checkout-pr step will be generated
	// This is used by the conclusion job to skip failure handling when checkout fails
	// (e.g., when PR is merged and branch is deleted)
	// The checkout-pr step is only generated when the workflow has contents read permission
	if ShouldGeneratePRCheckoutStep(data) {
		outputs["checkout_pr_success"] = "${{ steps.checkout-pr.outputs.checkout_pr_success || 'true' }}"
		compilerMainJobLog.Print("Added checkout_pr_success output (workflow has contents read access)")
	} else {
		compilerMainJobLog.Print("Skipped checkout_pr_success output (workflow lacks contents read access)")
	}

	// Add inference_access_error, mcp_policy_error, agentic_engine_timeout, and
	// model_not_supported_error outputs for engines that provide an error detection step.
	// These outputs are written by the host-runner detect-agent-errors step (via the
	// engine's GetErrorDetectionScriptId script) rather than from inside the AWF container,
	// because GITHUB_OUTPUT is not accessible inside the sandbox.
	engine, engineErr := c.getAgenticEngine(data.AI)
	if engineErr == nil {
		if engine.GetErrorDetectionScriptId() != "" {
			stepRef := fmt.Sprintf("steps.%s.outputs", constants.DetectAgentErrorsStepID)
			outputs["inference_access_error"] = fmt.Sprintf("${{ %s.inference_access_error || 'false' }}", stepRef)
			compilerMainJobLog.Printf("Added inference_access_error output (engine=%s, step=%s)", engine.GetID(), constants.DetectAgentErrorsStepID)

			outputs["mcp_policy_error"] = fmt.Sprintf("${{ %s.mcp_policy_error || 'false' }}", stepRef)
			compilerMainJobLog.Printf("Added mcp_policy_error output (engine=%s, step=%s)", engine.GetID(), constants.DetectAgentErrorsStepID)

			outputs["agentic_engine_timeout"] = fmt.Sprintf("${{ %s.agentic_engine_timeout || 'false' }}", stepRef)
			compilerMainJobLog.Printf("Added agentic_engine_timeout output (engine=%s, step=%s)", engine.GetID(), constants.DetectAgentErrorsStepID)

			outputs["model_not_supported_error"] = fmt.Sprintf("${{ %s.model_not_supported_error || 'false' }}", stepRef)
			compilerMainJobLog.Printf("Added model_not_supported_error output (engine=%s, step=%s)", engine.GetID(), constants.DetectAgentErrorsStepID)
		}
	}

	// Build job-level environment variables for safe outputs
	var env map[string]string
	if data.SafeOutputs != nil {
		env = make(map[string]string)

		// Set GH_AW_MCP_LOG_DIR for safe outputs MCP server logging
		// Store in mcp-logs directory so it's included in mcp-logs artifact
		env["GH_AW_MCP_LOG_DIR"] = "/tmp/gh-aw/mcp-logs/safeoutputs"

		// Note: GH_AW_SAFE_OUTPUTS, GH_AW_SAFE_OUTPUTS_CONFIG_PATH, and
		// GH_AW_SAFE_OUTPUTS_TOOLS_PATH are set via a run step (see generateSetRuntimePathsStep)
		// because the runner context is not available in job-level env: blocks.

		// Add asset-related environment variables
		// These must always be set (even to empty) because awmg v0.0.12+ validates ${VAR} references
		if data.SafeOutputs.UploadAssets != nil {
			env["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
			env["GH_AW_ASSETS_MAX_SIZE_KB"] = strconv.Itoa(data.SafeOutputs.UploadAssets.MaxSizeKB)
			env["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
		} else {
			// Set empty defaults when upload-assets is not configured
			env["GH_AW_ASSETS_BRANCH"] = `""`
			env["GH_AW_ASSETS_MAX_SIZE_KB"] = "0"
			env["GH_AW_ASSETS_ALLOWED_EXTS"] = `""`
		}

		// DEFAULT_BRANCH is used by safeoutputs MCP server
		// Use repository default branch from GitHub context
		env["DEFAULT_BRANCH"] = "${{ github.event.repository.default_branch }}"
	}

	// Set GH_AW_WORKFLOW_ID_SANITIZED for cache-memory keys
	// This contains the workflow ID with all hyphens removed and lowercased
	// Used in cache keys to avoid spaces and special characters
	if data.WorkflowID != "" {
		if env == nil {
			env = make(map[string]string)
		}
		sanitizedID := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
		env["GH_AW_WORKFLOW_ID_SANITIZED"] = sanitizedID
	}

	// Generate agent concurrency configuration
	agentConcurrency := GenerateJobConcurrencyConfig(data)

	// Set up permissions for the agent job
	// In dev/script mode, automatically add contents: read if the actions folder checkout is needed
	// In release mode, use the permissions as specified by the user (no automatic augmentation)
	//
	// GitHub App-only permissions (e.g., members, administration) must be filtered out before
	// rendering to the job-level permissions block. These scopes are not valid GitHub Actions
	// workflow permissions and cause a parse error when queued. They are handled separately
	// when minting GitHub App installation access tokens (as permission-* inputs).
	permissions := filterJobLevelPermissions(data.Permissions, data.CachedPermissions)
	needsContentsRead := (c.actionMode.IsDev() || c.actionMode.IsScript()) && len(c.generateCheckoutActionsFolder(data)) > 0
	if needsContentsRead {
		if permissions == "" {
			perms := NewPermissionsContentsRead()
			permissions = perms.RenderToYAML()
		} else {
			// Parse the already-filtered permissions string (not the raw data.Permissions)
			// since filterJobLevelPermissions may have adjusted the indentation/format.
			parser := NewPermissionsParser(permissions)
			perms := parser.ToPermissions()
			if level, exists := perms.Get(PermissionContents); !exists || level == PermissionNone {
				perms.Set(PermissionContents, PermissionRead)
				permissions = perms.RenderToYAML()
			}
		}
	}

	// Infer permissions required by gh CLI calls in all agent job step sections.
	// Detects write commands (which are not permitted since the agent job is read-only),
	// and merges inferred read permissions into the existing permissions block.
	// Skipped only when the user explicitly opted out of all permissions (permissions: {}).
	//
	// Top-level frontmatter sections (pre-steps, steps, pre-agent-steps, post-steps) are
	// all applied to the agent job and must be fully scanned.
	// For jobs.agent.* sections, only jobs.agent.pre-steps is actually injected by
	// applyBuiltinJobPreSteps; jobs.agent.steps, jobs.agent.pre-agent-steps, and
	// jobs.agent.post-steps are ignored for built-in jobs, so they are intentionally
	// excluded to avoid false-positive errors or unneeded permission grants.
	agentJobName := string(constants.AgentJobName)
	agentAllScripts := extractRunScriptsFromSectionYAML(data.PreSteps, "pre-steps")
	agentAllScripts = append(agentAllScripts, extractRunScriptsFromSectionYAML(data.CustomSteps, "steps")...)
	agentAllScripts = append(agentAllScripts, extractRunScriptsFromSectionYAML(data.PreAgentSteps, "pre-agent-steps")...)
	agentAllScripts = append(agentAllScripts, extractRunScriptsFromSectionYAML(data.PostSteps, "post-steps")...)
	if data.Jobs != nil {
		agentAllScripts = append(agentAllScripts, extractRunScriptsFromJobSection(data.Jobs, agentJobName, "pre-steps")...)
	}
	if len(agentAllScripts) > 0 {
		if writeCmds := detectWriteCommandsInShellScripts(agentAllScripts); len(writeCmds) > 0 {
			return nil, fmt.Errorf(
				"agent job uses write gh command(s) [%s]; write operations are not permitted in agent job steps because the agent job runs with read-only permissions. Use safe-outputs for write operations. See: https://github.github.com/gh-aw/reference/safe-outputs/",
				strings.Join(writeCmds, ", "),
			)
		}
		// Infer read permissions unless the user explicitly zeroed out all permissions.
		// Check data.Permissions (the original value) since needsContentsRead above may have
		// already expanded "permissions: {}" into an explicit block.
		// Uses the same exact-string check as tools.go (the YAML parser always normalizes
		// "permissions: {}" to this canonical form when parsing the frontmatter).
		if data.Permissions != "permissions: {}" && permissions != "" {
			inferred := inferPermissionsFromShellScripts(agentAllScripts)
			if len(inferred) > 0 {
				permissions = mergeInferredIntoPermissionsYAML(permissions, inferred)
			}
		}
	}

	// In script mode, explicitly add a cleanup step (mirrors post.js in dev/release/action mode).
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}

	job := &Job{
		Name:        string(constants.AgentJobName),
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Container:   c.indentYAMLLines(data.Container, "    "),
		Services:    c.indentYAMLLines(data.Services, "    "),
		Permissions: c.indentYAMLLines(permissions, "    "),
		Concurrency: c.indentYAMLLines(agentConcurrency, "    "),
		Env:         env,
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}

	return job, nil
}
