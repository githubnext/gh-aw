package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var compilerActivationJobsLog = logger.New("workflow:compiler_activation_jobs")

// buildPreActivationJob creates a unified pre-activation job that combines membership checks and stop-time validation.
// This job exposes a single "activated" output that indicates whether the workflow should proceed.
func (c *Compiler) buildPreActivationJob(data *WorkflowData, needsPermissionCheck bool) (*Job, error) {
	compilerActivationJobsLog.Printf("Building pre-activation job: needsPermissionCheck=%v, hasStopTime=%v", needsPermissionCheck, data.StopTime != "")
	var steps []string
	var permissions string

	// Extract custom steps and outputs from jobs.pre-activation if present
	customSteps, customOutputs, err := c.extractPreActivationCustomFields(data.Jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to extract pre-activation custom fields: %w", err)
	}

	// Add setup step to copy activation scripts (required - no inline fallback)
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" {
		return nil, errors.New("setup action reference is required but could not be resolved")
	}

	// For dev mode (local action path), checkout the actions folder first
	// This requires contents: read permission
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)
	needsContentsRead := (c.actionMode.IsDev() || c.actionMode.IsScript()) && len(c.generateCheckoutActionsFolder(data)) > 0

	// Pre-activation job doesn't need project support (no safe outputs processed here)
	// Pre-activation runs before activation so no trace ID is available yet
	steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false, "")...)

	// Determine permissions for pre-activation job
	var perms *Permissions
	if needsContentsRead {
		perms = NewPermissionsContentsRead()
	}

	// Add actions: read permission if rate limiting is configured (needed to query workflow runs)
	if data.RateLimit != nil {
		if perms == nil {
			perms = NewPermissions()
		}
		perms.Set(PermissionActions, PermissionRead)
	}

	// Merge on.permissions into the pre-activation job permissions.
	// on.permissions lets users declare extra scopes required by their on.steps steps.
	if data.OnPermissions != nil {
		if perms == nil {
			perms = NewPermissions()
		}
		perms.Merge(data.OnPermissions)
	}

	// Set permissions if any were configured
	if perms != nil {
		permissions = perms.RenderToYAML()
	}

	// Add team member check if permission checks are needed
	if needsPermissionCheck {
		steps = c.generateMembershipCheck(data, steps)
	}

	// Add rate limit check if configured
	if data.RateLimit != nil {
		steps = c.generateRateLimitCheck(data, steps)
	}

	// Add stop-time check if configured
	if data.StopTime != "" {
		compilerActivationJobsLog.Printf("Adding stop-time check step: stop_time=%s", data.StopTime)
		// Extract workflow name for the stop-time check
		workflowName := data.Name

		steps = append(steps, "      - name: Check stop-time limit\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckStopTimeStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		// Strip ANSI escape codes from stop-time value
		cleanStopTime := stringutil.StripANSI(data.StopTime)
		steps = append(steps, fmt.Sprintf("          GH_AW_STOP_TIME: %q\n", cleanStopTime))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_stop_time.cjs"))
	}

	// Emit a single unified GitHub App token mint step if on.github-app is configured
	// and any skip-if check is present. Both checks share the same minted token.
	hasSkipIfCheck := data.SkipIfMatch != nil || data.SkipIfNoMatch != nil
	if hasSkipIfCheck && data.ActivationGitHubApp != nil {
		steps = append(steps, c.buildPreActivationAppTokenMintStep(data.ActivationGitHubApp)...)
	}

	// Resolve the token expression to use for skip-if checks (app token > custom token > default)
	skipIfToken := c.resolvePreActivationSkipIfToken(data)

	// Add skip-if-match check if configured
	if data.SkipIfMatch != nil {
		compilerActivationJobsLog.Printf("Adding skip-if-match check step: query=%s, max=%d", data.SkipIfMatch.Query, data.SkipIfMatch.Max)
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-if-match query\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipIfMatchStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_QUERY: %q\n", data.SkipIfMatch.Query))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_MAX_MATCHES: \"%d\"\n", data.SkipIfMatch.Max))
		if data.SkipIfMatch.Scope != "" {
			steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_SCOPE: %q\n", data.SkipIfMatch.Scope))
		}
		steps = append(steps, "        with:\n")
		if skipIfToken != "" {
			steps = append(steps, fmt.Sprintf("          github-token: %s\n", skipIfToken))
		}
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_skip_if_match.cjs"))
	}

	// Add skip-if-no-match check if configured
	if data.SkipIfNoMatch != nil {
		compilerActivationJobsLog.Printf("Adding skip-if-no-match check step: query=%s, min=%d", data.SkipIfNoMatch.Query, data.SkipIfNoMatch.Min)
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-if-no-match query\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipIfNoMatchStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_QUERY: %q\n", data.SkipIfNoMatch.Query))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_MIN_MATCHES: \"%d\"\n", data.SkipIfNoMatch.Min))
		if data.SkipIfNoMatch.Scope != "" {
			steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_SCOPE: %q\n", data.SkipIfNoMatch.Scope))
		}
		steps = append(steps, "        with:\n")
		if skipIfToken != "" {
			steps = append(steps, fmt.Sprintf("          github-token: %s\n", skipIfToken))
		}
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_skip_if_no_match.cjs"))
	}

	// Add skip-if-check-failing check if configured
	if data.SkipIfCheckFailing != nil {
		compilerActivationJobsLog.Printf("Adding skip-if-check-failing check step: include=%v, exclude=%v", data.SkipIfCheckFailing.Include, data.SkipIfCheckFailing.Exclude)
		steps = append(steps, "      - name: Check skip-if-check-failing\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipIfCheckFailingStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		if len(data.SkipIfCheckFailing.Include) > 0 || len(data.SkipIfCheckFailing.Exclude) > 0 || data.SkipIfCheckFailing.Branch != "" || data.SkipIfCheckFailing.AllowPending {
			steps = append(steps, "        env:\n")
			if len(data.SkipIfCheckFailing.Include) > 0 {
				includeJSON, _ := json.Marshal(data.SkipIfCheckFailing.Include)
				steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_CHECK_INCLUDE: %q\n", string(includeJSON)))
			}
			if len(data.SkipIfCheckFailing.Exclude) > 0 {
				excludeJSON, _ := json.Marshal(data.SkipIfCheckFailing.Exclude)
				steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_CHECK_EXCLUDE: %q\n", string(excludeJSON)))
			}
			if data.SkipIfCheckFailing.Branch != "" {
				steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_BRANCH: %q\n", data.SkipIfCheckFailing.Branch))
			}
			if data.SkipIfCheckFailing.AllowPending {
				steps = append(steps, "          GH_AW_SKIP_CHECK_ALLOW_PENDING: \"true\"\n")
			}
		}
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_skip_if_check_failing.cjs"))
	}

	// Add skip-roles check if configured
	if len(data.SkipRoles) > 0 {
		// Extract workflow name for the skip-roles check
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-roles\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipRolesStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_ROLES: %q\n", strings.Join(data.SkipRoles, ",")))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          github-token: ${{ secrets.GITHUB_TOKEN }}\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_skip_roles.cjs"))
	}

	// Add skip-bots check if configured
	if len(data.SkipBots) > 0 {
		// Extract workflow name for the skip-bots check
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-bots\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipBotsStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_BOTS: %q\n", strings.Join(data.SkipBots, ",")))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_skip_bots.cjs"))
	}

	// Add command position check if this is a command workflow
	if len(data.Command) > 0 {
		steps = append(steps, "      - name: Check command position\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckCommandPositionStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		// Pass commands as JSON array
		commandsJSON, _ := json.Marshal(data.Command)
		steps = append(steps, fmt.Sprintf("          GH_AW_COMMANDS: %q\n", string(commandsJSON)))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_command_position.cjs"))
	}

	// Append custom steps from jobs.pre-activation if present
	if len(customSteps) > 0 {
		compilerActivationJobsLog.Printf("Adding %d custom steps to pre-activation job", len(customSteps))
		steps = append(steps, customSteps...)
	}

	// Append on.steps if present (injected after other checks)
	var onStepIDs []string
	if len(data.OnSteps) > 0 {
		compilerActivationJobsLog.Printf("Adding %d on.steps to pre-activation job", len(data.OnSteps))
		for i, stepMap := range data.OnSteps {
			stepYAML, err := ConvertStepToYAML(stepMap)
			if err != nil {
				return nil, fmt.Errorf("failed to convert on.steps[%d] to YAML: %w", i, err)
			}
			steps = append(steps, stepYAML)
			// Collect step IDs for output wiring
			if id, ok := stepMap["id"].(string); ok && id != "" {
				onStepIDs = append(onStepIDs, id)
			}
		}
	}

	// Generate the activated output expression using expression builders
	var activatedNode ConditionNode

	// Build condition nodes for each check
	var conditions []ConditionNode

	if needsPermissionCheck {
		// Add membership check condition
		membershipCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckMembershipStepID, constants.IsTeamMemberOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, membershipCheck)
	}

	if data.StopTime != "" {
		// Add stop-time check condition
		stopTimeCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckStopTimeStepID, constants.StopTimeOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, stopTimeCheck)
	}

	if data.SkipIfMatch != nil {
		// Add skip-if-match check condition
		skipCheckOk := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckSkipIfMatchStepID, constants.SkipCheckOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, skipCheckOk)
	}

	if data.SkipIfNoMatch != nil {
		// Add skip-if-no-match check condition
		skipNoMatchCheckOk := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckSkipIfNoMatchStepID, constants.SkipNoMatchCheckOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, skipNoMatchCheckOk)
	}

	if data.SkipIfCheckFailing != nil {
		// Add skip-if-check-failing check condition
		skipIfCheckFailingOk := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckSkipIfCheckFailingStepID, constants.SkipIfCheckFailingOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, skipIfCheckFailingOk)
	}

	if len(data.SkipRoles) > 0 {
		// Add skip-roles check condition
		skipRolesCheckOk := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckSkipRolesStepID, constants.SkipRolesOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, skipRolesCheckOk)
	}

	if len(data.SkipBots) > 0 {
		// Add skip-bots check condition
		skipBotsCheckOk := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckSkipBotsStepID, constants.SkipBotsOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, skipBotsCheckOk)
	}

	if data.RateLimit != nil {
		// Add rate limit check condition
		rateLimitCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckRateLimitStepID, constants.RateLimitOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, rateLimitCheck)
	}

	if len(data.Command) > 0 {
		// Add command position check condition
		commandPositionCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckCommandPositionStepID, constants.CommandPositionOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, commandPositionCheck)
	}

	// Build the final expression
	if len(conditions) == 0 {
		// Pre-activation was created solely for on.steps injection.
		// The activated output is unconditionally true; the user controls
		// agent execution through their own if: condition referencing the
		// on.steps outputs (e.g., needs.pre_activation.outputs.gate_result).
		if len(data.OnSteps) > 0 {
			compilerActivationJobsLog.Printf("Pre-activation created with on.steps only (%d steps); activated output is unconditionally true", len(data.OnSteps))
			activatedNode = BuildStringLiteral("true")
		} else {
			// This should never happen - it means pre-activation job was created without any checks
			// If we reach this point, it's a developer error in the compiler logic
			return nil, errors.New("developer error: pre-activation job created without permission check or stop-time configuration")
		}
	} else if len(conditions) == 1 {
		// Single condition
		activatedNode = conditions[0]
	} else {
		// Multiple conditions - combine with AND
		activatedNode = conditions[0]
		for i := 1; i < len(conditions); i++ {
			activatedNode = BuildAnd(activatedNode, conditions[i])
		}
	}

	// Render the expression with ${{ }} wrapper
	activatedExpression := fmt.Sprintf("${{ %s }}", activatedNode.Render())

	outputs := map[string]string{
		"activated": activatedExpression,
	}

	// Always declare matched_command output so actionlint can resolve the type.
	// For command workflows, reference the check_command_position step output.
	// For non-command workflows, emit an empty string so the output key is defined.
	if len(data.Command) > 0 {
		outputs[constants.MatchedCommandOutput] = fmt.Sprintf("${{ steps.%s.outputs.%s }}", constants.CheckCommandPositionStepID, constants.MatchedCommandOutput)
	} else {
		outputs[constants.MatchedCommandOutput] = "''"
	}

	// Wire on.steps step outcomes as pre-activation outputs.
	// For each step with an id, emit output "<id>_result: ${{ steps.<id>.outcome }}"
	// so users can reference them with: needs.pre_activation.outputs.<id>_result
	// This is done BEFORE merging custom outputs so that explicit user-defined outputs
	// in jobs.pre-activation.outputs take precedence over the auto-wired values.
	if len(onStepIDs) > 0 {
		compilerActivationJobsLog.Printf("Wiring %d on.steps step outcomes as pre-activation outputs", len(onStepIDs))
		for _, id := range onStepIDs {
			outputKey := id + "_result"
			outputs[outputKey] = fmt.Sprintf("${{ steps.%s.outcome }}", id)
		}
	}

	// Merge custom outputs from jobs.pre-activation if present.
	// Custom outputs are applied last so they take precedence over auto-wired on.steps outputs.
	if len(customOutputs) > 0 {
		compilerActivationJobsLog.Printf("Adding %d custom outputs to pre-activation job", len(customOutputs))
		maps.Copy(outputs, customOutputs)
	}

	// Pre-activation job uses the user's original if condition (data.If)
	// The workflow_run safety check is NOT applied here - it's only on the activation job
	// Don't include conditions that reference custom job outputs (those belong on the agent job)
	// Also don't include conditions that reference pre_activation outputs - those are outputs of this
	// very job and can only be evaluated by downstream jobs (activation, agent).
	var jobIfCondition string
	if !c.referencesCustomJobOutputs(data.If, data.Jobs) && !referencesPreActivationOutputs(data.If) {
		jobIfCondition = data.If
	}

	// In script mode, explicitly add a cleanup step (mirrors post.js in dev/release/action mode).
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}

	job := &Job{
		Name:        string(constants.PreActivationJobName),
		If:          jobIfCondition,
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		Environment: c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

// extractPreActivationCustomFields extracts custom steps and outputs from jobs.pre-activation field in frontmatter.
// It validates that only steps and outputs fields are present, and errors on any other fields.
// If both jobs.pre-activation and jobs.pre_activation are defined, imports from both.
// Returns (customSteps, customOutputs, error).
func (c *Compiler) extractPreActivationCustomFields(jobs map[string]any) ([]string, map[string]string, error) {
	if jobs == nil {
		return nil, nil, nil
	}

	var customSteps []string
	var customOutputs map[string]string

	// Check both jobs.pre-activation and jobs.pre_activation (users might define both by mistake)
	// Import from both if both are defined
	jobVariants := []string{"pre-activation", string(constants.PreActivationJobName)}

	for _, jobName := range jobVariants {
		preActivationJob, exists := jobs[jobName]
		if !exists {
			continue
		}

		// jobs.pre-activation must be a map
		configMap, ok := preActivationJob.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("jobs.%s must be an object, got %T", jobName, preActivationJob)
		}

		// Validate that only steps and outputs fields are present
		allowedFields := map[string]bool{
			"steps":   true,
			"outputs": true,
		}

		for field := range configMap {
			if !allowedFields[field] {
				return nil, nil, fmt.Errorf("jobs.%s: unsupported field '%s' - only 'steps' and 'outputs' are allowed", jobName, field)
			}
		}

		// Extract steps
		if stepsValue, hasSteps := configMap["steps"]; hasSteps {
			stepsList, ok := stepsValue.([]any)
			if !ok {
				return nil, nil, fmt.Errorf("jobs.%s.steps must be an array, got %T", jobName, stepsValue)
			}

			for i, step := range stepsList {
				stepMap, ok := step.(map[string]any)
				if !ok {
					return nil, nil, fmt.Errorf("jobs.%s.steps[%d] must be an object, got %T", jobName, i, step)
				}

				// Convert step to YAML
				stepYAML, err := ConvertStepToYAML(stepMap)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert jobs.%s.steps[%d] to YAML: %w", jobName, i, err)
				}
				customSteps = append(customSteps, stepYAML)
			}
			compilerActivationJobsLog.Printf("Extracted %d custom steps from jobs.%s", len(stepsList), jobName)
		}

		// Extract outputs
		if outputsValue, hasOutputs := configMap["outputs"]; hasOutputs {
			outputsMap, ok := outputsValue.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("jobs.%s.outputs must be an object, got %T", jobName, outputsValue)
			}

			if customOutputs == nil {
				customOutputs = make(map[string]string)
			}
			for key, val := range outputsMap {
				valStr, ok := val.(string)
				if !ok {
					return nil, nil, fmt.Errorf("jobs.%s.outputs.%s must be a string, got %T", jobName, key, val)
				}
				// If the same output key is defined in both variants, the second one wins (pre_activation)
				customOutputs[key] = valStr
			}
			compilerActivationJobsLog.Printf("Extracted %d custom outputs from jobs.%s", len(outputsMap), jobName)
		}
	}

	return customSteps, customOutputs, nil
}

// buildPreActivationAppTokenMintStep generates a single GitHub App token mint step for use
// by all skip-if checks in the pre-activation job. The step ID is "pre-activation-app-token".
// Auth configuration comes from the top-level on.github-app field.
func (c *Compiler) buildPreActivationAppTokenMintStep(app *GitHubAppConfig) []string {
	var steps []string
	tokenStepID := constants.PreActivationAppTokenStepID

	steps = append(steps, "      - name: Generate GitHub App token for skip-if checks\n")
	steps = append(steps, fmt.Sprintf("        id: %s\n", tokenStepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	if len(app.Repositories) == 1 && app.Repositories[0] == "*" {
		// Org-wide access: omit repositories field entirely
	} else if len(app.Repositories) == 1 {
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	} else if len(app.Repositories) > 1 {
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range app.Repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	} else {
		steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")
	}

	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	return steps
}

// resolvePreActivationSkipIfToken returns the GitHub token expression to use for skip-if check
// steps in the pre-activation job. Priority: App token > custom github-token > empty (default).
// When non-empty, callers should emit `with.github-token: <value>` in the step.
func (c *Compiler) resolvePreActivationSkipIfToken(data *WorkflowData) string {
	if data.ActivationGitHubApp != nil {
		return fmt.Sprintf("${{ steps.%s.outputs.token }}", constants.PreActivationAppTokenStepID)
	}
	if data.ActivationGitHubToken != "" {
		return data.ActivationGitHubToken
	}
	return ""
}

// extractOnSteps extracts the 'steps' field from the 'on:' section of frontmatter.
// These steps are injected into the pre-activation job and their step outcome is wired
// as pre-activation outputs so users can reference them with:
//
//	needs.pre_activation.outputs.<id>_result   (contains outcome: success/failure/cancelled/skipped)
//
// Returns nil if on.steps is not configured.
// Returns an error if on.steps is not an array or contains non-object items.
func extractOnSteps(frontmatter map[string]any) ([]map[string]any, error) {
	onValue, exists := frontmatter["on"]
	if !exists || onValue == nil {
		return nil, nil
	}

	onMap, ok := onValue.(map[string]any)
	if !ok {
		return nil, nil
	}

	stepsValue, exists := onMap["steps"]
	if !exists || stepsValue == nil {
		return nil, nil
	}

	stepsList, ok := stepsValue.([]any)
	if !ok {
		return nil, fmt.Errorf("on.steps must be an array, got %T", stepsValue)
	}

	result := make([]map[string]any, 0, len(stepsList))
	for i, step := range stepsList {
		stepMap, ok := step.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("on.steps[%d] must be an object, got %T", i, step)
		}
		result = append(result, stepMap)
	}

	return result, nil
}

// extractOnPermissions extracts the 'permissions' field from the 'on:' section of frontmatter.
// These permissions are merged into the pre-activation job permissions, allowing users to declare
// extra scopes required by their on.steps (e.g., issues: read for GitHub API calls).
//
// Returns nil if on.permissions is not configured.
func extractOnPermissions(frontmatter map[string]any) *Permissions {
	onValue, exists := frontmatter["on"]
	if !exists || onValue == nil {
		return nil
	}

	onMap, ok := onValue.(map[string]any)
	if !ok {
		return nil
	}

	permsValue, exists := onMap["permissions"]
	if !exists || permsValue == nil {
		return nil
	}

	parser := NewPermissionsParserFromValue(permsValue)
	return parser.ToPermissions()
}

// referencesPreActivationOutputs returns true if the condition references the pre_activation job's
// own outputs (e.g., "needs.pre_activation.outputs.foo"). Such conditions cannot be applied to the
// pre_activation job itself (a job cannot reference its own outputs), so they are deferred to
// downstream jobs (activation, agent).
func referencesPreActivationOutputs(condition string) bool {
	if condition == "" {
		return false
	}
	return strings.Contains(condition, "needs."+string(constants.PreActivationJobName)+".outputs.")
}
