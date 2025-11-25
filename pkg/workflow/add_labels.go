package workflow

import (
	"fmt"
)

// AddLabelsConfig holds configuration for adding labels to issues/PRs from agent output
type AddLabelsConfig struct {
	Allowed        []string `yaml:"allowed,omitempty"`      // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	Max            int      `yaml:"max,omitempty"`          // Optional maximum number of labels to add (default: 3)
	GitHubToken    string   `yaml:"github-token,omitempty"` // GitHub token for this specific output type
	Target         string   `yaml:"target,omitempty"`       // Target for labels: "triggering" (default), "*" (any issue/PR), or explicit issue/PR number
	TargetRepoSlug string   `yaml:"target-repo,omitempty"`  // Target repository in format "owner/repo" for cross-repository labels
}

// buildAddLabelsJob creates the add_labels job
func (c *Compiler) buildAddLabelsJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddLabels == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	cfg := data.SafeOutputs.AddLabels

	// Determine max count (default is 3)
	maxCount := 3
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Use the shared list-based builder
	return c.buildListSafeOutputJob(data, ListSafeOutputJobParams{
		JobName:        "add_labels",
		StepName:       "Add Labels",
		StepID:         "add_labels",
		MainJobName:    mainJobName,
		EnvPrefix:      "LABELS",
		AllowedItems:   cfg.Allowed,
		MaxCount:       maxCount,
		Target:         cfg.Target,
		TargetRepoSlug: cfg.TargetRepoSlug,
		Script:         getAddLabelsScript(),
		Permissions:    NewPermissionsContentsReadIssuesWritePRWrite(),
		Token:          cfg.GitHubToken,
		OutputKey:      "labels_added",
		// Labels can be added to both issues and PRs
		TriggeringContextConditions: []ConditionNode{
			BuildPropertyAccess("github.event.issue.number"),
			BuildPropertyAccess("github.event.pull_request.number"),
		},
	})
}
