package workflow

import (
	"fmt"
	"strings"
)

var safeOutputsAllowedLabelsValidationLog = newValidationLogger("safe_outputs_allowed_labels")

// validateSafeOutputsAllowedLabelsGlobScope returns an error when any safe-outputs
// allowed-labels field contains a bare "*" glob pattern (CTR-015).
//
// A bare "*" in allowed-labels is semantically equivalent to omitting the field
// entirely: all labels are permitted and the restriction is ineffective. This is
// almost always accidental. Authors should use specific names or narrower patterns
// such as "team-*" or "priority-*" instead.
func (c *Compiler) validateSafeOutputsAllowedLabelsGlobScope(config *SafeOutputsConfig) error {
	if config == nil {
		return nil
	}

	type labelledConfig struct {
		name   string
		labels []string
	}

	var configs []labelledConfig

	if config.CreateIssues != nil && len(config.CreateIssues.AllowedLabels) > 0 {
		configs = append(configs, labelledConfig{"safe-outputs.create-issue.allowed-labels", config.CreateIssues.AllowedLabels})
	}
	if config.CreateDiscussions != nil && len(config.CreateDiscussions.AllowedLabels) > 0 {
		configs = append(configs, labelledConfig{"safe-outputs.create-discussion.allowed-labels", config.CreateDiscussions.AllowedLabels})
	}
	if config.UpdateDiscussions != nil && len(config.UpdateDiscussions.AllowedLabels) > 0 {
		configs = append(configs, labelledConfig{"safe-outputs.update-discussion.allowed-labels", config.UpdateDiscussions.AllowedLabels})
	}
	if config.CreatePullRequests != nil && len(config.CreatePullRequests.AllowedLabels) > 0 {
		configs = append(configs, labelledConfig{"safe-outputs.create-pull-request.allowed-labels", config.CreatePullRequests.AllowedLabels})
	}

	for _, lc := range configs {
		for _, pattern := range lc.labels {
			if strings.TrimSpace(pattern) == "*" {
				msg := fmt.Sprintf(
					"CTR-015: %s contains a bare \"*\" wildcard that matches any label, "+
						"effectively disabling the label restriction.\n\n"+
						"Using \"*\" in allowed-labels has the same effect as omitting the field entirely "+
						"and may allow the agent to apply labels that trigger unintended automation.\n"+
						"Replace with specific label names or narrower patterns (e.g., \"team-*\", \"priority-*\") "+
						"to restrict which labels the agent is allowed to apply.",
					lc.name,
				)
				safeOutputsAllowedLabelsValidationLog.Printf("Error: %s", msg)
				return fmt.Errorf("%s", msg)
			}
		}
	}
	return nil
}
