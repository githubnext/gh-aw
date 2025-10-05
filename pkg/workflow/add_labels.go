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
	minValue := 0

	if data.SafeOutputs.AddLabels != nil {
		allowedLabels = data.SafeOutputs.AddLabels.Allowed
		if data.SafeOutputs.AddLabels.Max > 0 {
			maxCount = data.SafeOutputs.AddLabels.Max
		}
		minValue = data.SafeOutputs.AddLabels.Min
	}

	var steps []string

	// Build environment variables
	env := make(map[string]string)
	targetValue := ""
	targetEnvName := "GITHUB_AW_LABELS_TARGET"
	if data.SafeOutputs.AddLabels != nil {
		targetValue = data.SafeOutputs.AddLabels.Target
	}

	// Build with parameters
	withParams := make(map[string]string)
	token := ""
	if data.SafeOutputs.AddLabels != nil {
		token = data.SafeOutputs.AddLabels.GitHubToken
	}

	envConfig := &SafeOutputEnvConfig{
		TargetValue:   targetValue,
		TargetEnvName: targetEnvName,
		GitHubToken:   token,
	}
	c.getCustomSafeOutputEnvVars(env, data, mainJobName, envConfig, withParams)

	// Pass the allowed labels list (empty string if no restrictions)
	allowedLabelsStr := strings.Join(allowedLabels, ",")
	env["GITHUB_AW_LABELS_ALLOWED"] = fmt.Sprintf("%q", allowedLabelsStr)
	// Pass the max limit
	env["GITHUB_AW_LABELS_MAX_COUNT"] = fmt.Sprintf("%d", maxCount)

	// Build github-script step
	stepLines := BuildGitHubScriptStepLines("Add Labels", "add_labels", addLabelsScript, env, withParams)
	steps = append(steps, stepLines...)

	// Create outputs for the job
	outputs := map[string]string{
		"labels_added": "${{ steps.add_labels.outputs.labels_added }}",
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
