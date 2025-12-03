package workflow

import (
	"fmt"
)

// AddLabelsConfig holds configuration for adding labels to issues/PRs from agent output
type AddLabelsConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
}

// parseAddLabelsConfig handles add-labels configuration
func (c *Compiler) parseAddLabelsConfig(outputMap map[string]any) *AddLabelsConfig {
	if labels, exists := outputMap["add-labels"]; exists {
		if labelsMap, ok := labels.(map[string]any); ok {
			labelConfig := &AddLabelsConfig{}

			// Parse list job config (target, target-repo, allowed)
			listJobConfig, _ := ParseListJobConfig(labelsMap, "allowed")
			labelConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
			labelConfig.Allowed = listJobConfig.Allowed

			// Parse common base fields (github-token, max)
			c.parseBaseSafeOutputConfig(labelsMap, &labelConfig.BaseSafeOutputConfig, 0)

			return labelConfig
		} else if labels == nil {
			// Handle null case: create empty config (allows any labels)
			return &AddLabelsConfig{}
		}
	}

	return nil
}

// buildAddLabelsJob creates the add_labels job
func (c *Compiler) buildAddLabelsJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddLabels == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	cfg := data.SafeOutputs.AddLabels

	// Handle max count with default of 3
	maxCount := 3
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables using shared helpers
	listJobConfig := ListJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		Allowed:                cfg.Allowed,
	}
	customEnvVars := BuildListJobEnvVars("GH_AW_LABELS", listJobConfig, maxCount)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"labels_added": "${{ steps.add_labels.outputs.labels_added }}",
	}

	var jobCondition = BuildSafeOutputType("add_labels")

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "add_labels",
		StepName:       "Add Labels",
		StepID:         "add_labels",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAddLabelsScript(),
		Permissions:    NewPermissionsContentsReadIssuesWritePRWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
