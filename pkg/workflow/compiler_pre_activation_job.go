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
	steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false)...)

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
		// Extract workflow name for the stop-time check
		workflowName := data.Name

		steps = append(steps, "      - name: Check stop-time limit\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckStopTimeStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		// Strip ANSI escape codes from stop-time value
		cleanStopTime := stringutil.StripANSI(data.StopTime)
		steps = append(steps, fmt.Sprintf("          GH_AW_STOP_TIME: %s\n", cleanStopTime))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_stop_time.cjs"))
	}

	// Add skip-if-match check if configured
	if data.SkipIfMatch != nil {
		// Extract workflow name for the skip-if-match check
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-if-match query\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipIfMatchStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_QUERY: %q\n", data.SkipIfMatch.Query))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_MAX_MATCHES: \"%d\"\n", data.SkipIfMatch.Max))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_skip_if_match.cjs"))
	}

	// Add skip-if-no-match check if configured
	if data.SkipIfNoMatch != nil {
		// Extract workflow name for the skip-if-no-match check
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-if-no-match query\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipIfNoMatchStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_QUERY: %q\n", data.SkipIfNoMatch.Query))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_MIN_MATCHES: \"%d\"\n", data.SkipIfNoMatch.Min))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_skip_if_no_match.cjs"))
	}

	// Add skip-roles check if configured
	if len(data.SkipRoles) > 0 {
		// Extract workflow name for the skip-roles check
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-roles\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipRolesStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_ROLES: %s\n", strings.Join(data.SkipRoles, ",")))
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
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_BOTS: %s\n", strings.Join(data.SkipBots, ",")))
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
		// This should never happen - it means pre-activation job was created without any checks
		// If we reach this point, it's a developer error in the compiler logic
		return nil, errors.New("developer error: pre-activation job created without permission check or stop-time configuration")
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

	// Merge custom outputs from jobs.pre-activation if present
	if len(customOutputs) > 0 {
		compilerActivationJobsLog.Printf("Adding %d custom outputs to pre-activation job", len(customOutputs))
		maps.Copy(outputs, customOutputs)
	}

	// Pre-activation job uses the user's original if condition (data.If)
	// The workflow_run safety check is NOT applied here - it's only on the activation job
	// Don't include conditions that reference custom job outputs (those belong on the agent job)
	var jobIfCondition string
	if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
		jobIfCondition = data.If
	}

	job := &Job{
		Name:        string(constants.PreActivationJobName),
		If:          jobIfCondition,
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
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
				stepYAML, err := c.convertStepToYAML(stepMap)
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
