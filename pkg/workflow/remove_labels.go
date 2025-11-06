package workflow

import (
	"fmt"
	"strings"
)

// RemoveLabelsConfig holds configuration for removing labels from issues/PRs from agent output
type RemoveLabelsConfig struct {
	Allowed        []string `yaml:"allowed,omitempty"`      // Optional list of allowed labels to remove. If omitted, any labels can be removed.
	Max            int      `yaml:"max,omitempty"`          // Optional maximum number of labels to remove (default: 3)
	GitHubToken    string   `yaml:"github-token,omitempty"` // GitHub token for this specific output type
	Target         string   `yaml:"target,omitempty"`       // Target for labels: "triggering" (default), "*" (any issue/PR), or explicit issue/PR number
	TargetRepoSlug string   `yaml:"target-repo,omitempty"`  // Target repository in format "owner/repo" for cross-repository labels
}

// buildRemoveLabelsJob creates the remove_labels job
func (c *Compiler) buildRemoveLabelsJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.RemoveLabels == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	// Handle case where RemoveLabels is nil (equivalent to empty configuration)
	var allowedLabels []string
	maxCount := 3

	allowedLabels = data.SafeOutputs.RemoveLabels.Allowed
	if data.SafeOutputs.RemoveLabels.Max > 0 {
		maxCount = data.SafeOutputs.RemoveLabels.Max
	}

	// Build custom environment variables specific to remove-labels
	var customEnvVars []string
	// Pass the allowed labels list (empty string if no restrictions)
	allowedLabelsStr := strings.Join(allowedLabels, ",")
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LABELS_ALLOWED: %q\n", allowedLabelsStr))
	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LABELS_MAX_COUNT: %d\n", maxCount))

	// Pass the target configuration
	if data.SafeOutputs.RemoveLabels.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LABELS_TARGET: %q\n", data.SafeOutputs.RemoveLabels.Target))
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.RemoveLabels.TargetRepoSlug,
	)...)

	// Get token from config
	token := ""
	if data.SafeOutputs.RemoveLabels != nil {
		token = data.SafeOutputs.RemoveLabels.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Remove Labels",
		StepID:        "remove_labels",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getRemoveLabelsScript(),
		Token:         token,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"labels_removed": "${{ steps.remove_labels.outputs.labels_removed }}",
	}

	var jobCondition = BuildSafeOutputType("remove_labels")
	if data.SafeOutputs.RemoveLabels.Target == "" {
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.issue.number"),
			BuildPropertyAccess("github.event.pull_request.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "remove_labels",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsReadIssuesWritePRWrite().RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
