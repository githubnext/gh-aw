package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// This file contains the buildPreActivationJob function for building the pre-activation job.
// The pre-activation job combines membership checks, stop-time validation, skip-if-match check,
// and command position checking into a single job that acts as a gatekeeper before activation.

// buildPreActivationJob creates the pre-activation job if permission or validation checks are needed.
// This job handles:
// 1. Permission checks (team member validation)
// 2. Stop-time validation (workflow expiration)
// 3. Skip-if-match queries (workflow skip conditions)
// 4. Command position checking (for command workflows)
func (c *Compiler) buildPreActivationJob(data *WorkflowData, needsPermissionCheck bool) (*Job, error) {
	compilerJobsLog.Printf("Building pre-activation job: needsPermissionCheck=%v, hasStopTime=%v", needsPermissionCheck, data.StopTime != "")
	var steps []string
	var permissions string

	// Add team member check if permission checks are needed
	if needsPermissionCheck {
		steps = c.generateMembershipCheck(data, steps)
	}

	// Add stop-time check if configured
	if data.StopTime != "" {
		// Extract workflow name for the stop-time check
		workflowName := data.Name

		steps = append(steps, "      - name: Check stop-time limit\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckStopTimeStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_STOP_TIME: %s\n", data.StopTime))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkStopTimeScript)
		steps = append(steps, formattedScript...)
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

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkSkipIfMatchScript)
		steps = append(steps, formattedScript...)
	}

	// Add command position check if this is a command workflow
	if data.Command != "" {
		steps = append(steps, "      - name: Check command position\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckCommandPositionStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_COMMAND: %s\n", data.Command))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkCommandPositionScript)
		steps = append(steps, formattedScript...)
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

	if data.Command != "" {
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
		return nil, fmt.Errorf("developer error: pre-activation job created without permission check or stop-time configuration")
	} else if len(conditions) == 1 {
		// Single condition
		activatedNode = conditions[0]
	} else {
		// Multiple conditions - combine with AND
		activatedNode = conditions[0]
		for i := 1; i < len(conditions); i++ {
			activatedNode = buildAnd(activatedNode, conditions[i])
		}
	}

	// Render the expression with ${{ }} wrapper
	activatedExpression := fmt.Sprintf("${{ %s }}", activatedNode.Render())

	outputs := map[string]string{
		"activated": activatedExpression,
	}

	// Pre-activation job uses the user's original if condition (data.If)
	// The workflow_run safety check is NOT applied here - it's only on the activation job
	// Don't include conditions that reference custom job outputs (those belong on the agent job)
	var jobIfCondition string
	if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
		jobIfCondition = data.If
	}

	job := &Job{
		Name:        constants.PreActivationJobName,
		If:          jobIfCondition,
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}
