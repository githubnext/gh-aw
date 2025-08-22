package workflow

import (
	"fmt"
	"strings"
)

// buildCreateOutputLabelJob creates the add_labels job
func (c *Compiler) buildCreateOutputLabelJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.Output == nil || data.Output.Labels == nil {
		return nil, fmt.Errorf("output.labels configuration is required")
	}

	// Validate that allowed labels list is not empty
	if len(data.Output.Labels.Allowed) == 0 {
		return nil, fmt.Errorf("output.labels.allowed must be non-empty")
	}

	// Get max-count with default of 3
	maxCount := 3
	if data.Output.Labels.MaxCount != nil {
		maxCount = *data.Output.Labels.MaxCount
	}

	var steps []string
	steps = append(steps, "      - name: Add Labels\n")
	steps = append(steps, "        id: add_labels\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the allowed labels list
	allowedLabelsStr := strings.Join(data.Output.Labels.Allowed, ",")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_LABELS_ALLOWED: %q\n", allowedLabelsStr))
	// Pass the max-count limit
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

	job := &Job{
		Name:           "add_labels",
		If:             "if: github.event.issue.number || github.event.pull_request.number", // Only run in issue or PR context
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
