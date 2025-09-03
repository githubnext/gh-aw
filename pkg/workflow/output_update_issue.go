package workflow

import (
	"fmt"
)

// buildCreateOutputUpdateIssueJob creates the update_issue job
func (c *Compiler) buildCreateOutputUpdateIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.update-issue configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Update Issue\n")
	steps = append(steps, "        id: update_issue\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))

	// Pass the configuration flags
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_STATUS: %t\n", data.SafeOutputs.UpdateIssues.Status != nil))
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_TITLE: %t\n", data.SafeOutputs.UpdateIssues.Title != nil))
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_BODY: %t\n", data.SafeOutputs.UpdateIssues.Body != nil))

	// Pass the target configuration
	if data.SafeOutputs.UpdateIssues.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_TARGET: %q\n", data.SafeOutputs.UpdateIssues.Target))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(updateIssueScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.update_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.update_issue.outputs.issue_url }}",
	}

	// Determine the job condition based on target configuration
	var baseCondition string
	if data.SafeOutputs.UpdateIssues.Target == "*" {
		// Allow updates to any issue - no specific context required
		baseCondition = "always()"
	} else if data.SafeOutputs.UpdateIssues.Target != "" {
		// Explicit issue number specified - no specific context required
		baseCondition = "always()"
	} else {
		// Default behavior: only update triggering issue
		baseCondition = "github.event.issue.number"
	}

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		if baseCondition == "always()" {
			// If base condition is always(), just use the command condition
			jobCondition = fmt.Sprintf("if: %s", commandConditionStr)
		} else {
			// Combine both conditions with AND
			jobCondition = fmt.Sprintf("if: (%s) && (%s)", commandConditionStr, baseCondition)
		}
	} else {
		// No command trigger, just use the base condition
		jobCondition = fmt.Sprintf("if: %s", baseCondition)
	}

	job := &Job{
		Name:           "update_issue",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      issues: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
