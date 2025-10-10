package workflow

import (
	"fmt"
	"strings"
)

// AddLabelsConfig holds configuration for adding labels to issues/PRs from agent output
type AddLabelsConfig struct {
	Allowed        []string `yaml:"allowed,omitempty"`      // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	Max            int      `yaml:"max,omitempty"`          // Optional maximum number of labels to add (default: 3)
	Min            int      `yaml:"min,omitempty"`          // Optional minimum number of labels to add
	GitHubToken    string   `yaml:"github-token,omitempty"` // GitHub token for this specific output type
	Target         string   `yaml:"target,omitempty"`       // Target for labels: "triggering" (default), "*" (any issue/PR), or explicit issue/PR number
	TargetRepoSlug string   `yaml:"target-repo,omitempty"`  // Target repository in format "owner/repo" for cross-repository labels
}

// buildAddLabelsJob creates the add_labels job
func (c *Compiler) buildAddLabelsJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddLabels == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	// Handle case where AddLabels is nil (equivalent to empty configuration)
	var allowedLabels []string
	maxCount := 3
	minValue := 0

	allowedLabels = data.SafeOutputs.AddLabels.Allowed
	if data.SafeOutputs.AddLabels.Max > 0 {
		maxCount = data.SafeOutputs.AddLabels.Max
	}
	minValue = data.SafeOutputs.AddLabels.Min

	// Build custom environment variables specific to add-labels
	var customEnvVars []string
	// Pass the allowed labels list (empty string if no restrictions)
	allowedLabelsStr := strings.Join(allowedLabels, ",")
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_LABELS_ALLOWED: %q\n", allowedLabelsStr))
	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_LABELS_MAX_COUNT: %d\n", maxCount))

	// Pass the target configuration
	if data.SafeOutputs.AddLabels.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_LABELS_TARGET: %q\n", data.SafeOutputs.AddLabels.Target))
	}

	// Pass the staged flag if it's set to true
	if c.trialMode || data.SafeOutputs.Staged {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Pass target repository - prefer explicit config over trial mode setting
	if data.SafeOutputs.AddLabels.TargetRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", data.SafeOutputs.AddLabels.TargetRepoSlug))
	} else if c.trialMode && c.trialSimulatedRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", c.trialSimulatedRepoSlug))
	}

	// Get token from config
	token := ""
	if data.SafeOutputs.AddLabels != nil {
		token = data.SafeOutputs.AddLabels.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Add Labels",
		StepID:        "add_labels",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        addLabelsScript,
		Token:         token,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"labels_added": "${{ steps.add_labels.outputs.labels_added }}",
	}

	var jobCondition = BuildSafeOutputType("add-labels", minValue)
	if data.SafeOutputs.AddLabels.Target == "" {
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
