package workflow

import (
	"fmt"
	"strings"
)

// buildCreateOutputLabelJob creates the add_labels job
func (c *Compiler) buildCreateOutputLabelJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	// Handle case where AddIssueLabels is nil (equivalent to empty configuration)
	var allowedLabels []string
	maxCount := 3

	if data.SafeOutputs.AddIssueLabels != nil {
		allowedLabels = data.SafeOutputs.AddIssueLabels.Allowed
		if data.SafeOutputs.AddIssueLabels.MaxCount != nil {
			maxCount = *data.SafeOutputs.AddIssueLabels.MaxCount
		}
	}

	var steps []string
	steps = append(steps, "      - name: Add Labels\n")
	steps = append(steps, "        id: add_labels\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the allowed labels list (empty string if no restrictions)
	allowedLabelsStr := strings.Join(allowedLabels, ",")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_LABELS_ALLOWED: %q\n", allowedLabelsStr))
	// Pass the max limit
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_LABELS_MAX_COUNT: %d\n", maxCount))

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(addLabelsScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"labels_added": "${{ steps.add_labels.outputs.labels_added }}",
	}

	// Determine the job condition for command workflows
	var baseCondition = "github.event.issue.number || github.event.pull_request.number" // Only run in issue or PR context
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		// Combine command condition with base condition using AND
		jobCondition = fmt.Sprintf("if: (%s) && (%s)", commandConditionStr, baseCondition)
	} else {
		// No command trigger, just use the base condition
		jobCondition = fmt.Sprintf("if: %s", baseCondition)
	}

	job := &Job{
		Name:           "add_labels",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
