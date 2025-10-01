package workflow

import (
	"fmt"
	"strings"
)

// AddLabelsConfig holds configuration for adding labels to issues/PRs from agent output
type AddLabelsConfig struct {
	Allowed     []string `yaml:"allowed,omitempty"`      // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	Max         int      `yaml:"max,omitempty"`          // Optional maximum number of labels to add (default: 3)
	Min         int      `yaml:"min,omitempty"`          // Optional minimum number of labels to add
	GitHubToken string   `yaml:"github-token,omitempty"` // GitHub token for this specific output type
	Target      string   `yaml:"target,omitempty"`       // Target for labels: "triggering" (default), "*" (any issue/PR), or explicit issue/PR number
}

// buildCreateOutputLabelJob creates the add_labels job
func (c *Compiler) buildCreateOutputLabelJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	// Handle case where AddLabels is nil (equivalent to empty configuration)
	var allowedLabels []string
	maxCount := 3

	if data.SafeOutputs.AddLabels != nil {
		allowedLabels = data.SafeOutputs.AddLabels.Allowed
		if data.SafeOutputs.AddLabels.Max > 0 {
			maxCount = data.SafeOutputs.AddLabels.Max
		}
	}

	var steps []string
	steps = append(steps, "      - name: Add Labels\n")
	steps = append(steps, "        id: add_labels\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the allowed labels list (empty string if no restrictions)
	allowedLabelsStr := strings.Join(allowedLabels, ",")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_LABELS_ALLOWED: %q\n", allowedLabelsStr))
	// Pass the max limit
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_LABELS_MAX_COUNT: %d\n", maxCount))

	// Pass the target configuration
	if data.SafeOutputs.AddLabels != nil && data.SafeOutputs.AddLabels.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_LABELS_TARGET: %q\n", data.SafeOutputs.AddLabels.Target))
	}

	// Pass the staged flag if it's set to true
	if data.SafeOutputs.Staged != nil && *data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.AddLabels != nil {
		token = data.SafeOutputs.AddLabels.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(addLabelsScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"labels_added": "${{ steps.add_labels.outputs.labels_added }}",
	}

	minValue := 0
	if data.SafeOutputs.AddLabels != nil {
		minValue = data.SafeOutputs.AddLabels.Min
	}
	var jobCondition = BuildSafeOutputType("add-labels", minValue)
	if data.SafeOutputs.AddLabels == nil || data.SafeOutputs.AddLabels.Target == "" {
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.issue.number"),
			BuildPropertyAccess("github.event.pull_request.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "add_labels",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
