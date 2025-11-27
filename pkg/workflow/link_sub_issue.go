package workflow

import (
	"fmt"
	"strings"
)

// LinkSubIssueConfig holds configuration for linking issues as sub-issues from agent output
type LinkSubIssueConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	ParentRequiredLabels []string `yaml:"parent-required-labels,omitempty"` // Required labels the parent issue must have
	ParentTitlePrefix    string   `yaml:"parent-title-prefix,omitempty"`    // Required title prefix for parent issue
	SubRequiredLabels    []string `yaml:"sub-required-labels,omitempty"`    // Required labels the sub-issue must have
	SubTitlePrefix       string   `yaml:"sub-title-prefix,omitempty"`       // Required title prefix for sub-issue
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`            // Target repository in format "owner/repo" for cross-repository operations
}

// parseLinkSubIssueConfig handles link-sub-issue configuration
func (c *Compiler) parseLinkSubIssueConfig(outputMap map[string]any) *LinkSubIssueConfig {
	if configData, exists := outputMap["link-sub-issue"]; exists {
		linkSubIssueConfig := &LinkSubIssueConfig{}
		linkSubIssueConfig.Max = 5 // Default max is 5

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields with default max of 5
			c.parseBaseSafeOutputConfig(configMap, &linkSubIssueConfig.BaseSafeOutputConfig, 5)

			// Parse parent-required-labels
			if parentLabels, exists := configMap["parent-required-labels"]; exists {
				if labelsArray, ok := parentLabels.([]any); ok {
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							linkSubIssueConfig.ParentRequiredLabels = append(linkSubIssueConfig.ParentRequiredLabels, labelStr)
						}
					}
				}
			}

			// Parse parent-title-prefix
			if parentTitlePrefix, exists := configMap["parent-title-prefix"]; exists {
				if prefixStr, ok := parentTitlePrefix.(string); ok {
					linkSubIssueConfig.ParentTitlePrefix = prefixStr
				}
			}

			// Parse sub-required-labels
			if subLabels, exists := configMap["sub-required-labels"]; exists {
				if labelsArray, ok := subLabels.([]any); ok {
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							linkSubIssueConfig.SubRequiredLabels = append(linkSubIssueConfig.SubRequiredLabels, labelStr)
						}
					}
				}
			}

			// Parse sub-title-prefix
			if subTitlePrefix, exists := configMap["sub-title-prefix"]; exists {
				if prefixStr, ok := subTitlePrefix.(string); ok {
					linkSubIssueConfig.SubTitlePrefix = prefixStr
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			linkSubIssueConfig.TargetRepoSlug = targetRepoSlug
		} else {
			// If configData is nil or not a map, still set the default max
			linkSubIssueConfig.Max = 5
		}

		return linkSubIssueConfig
	}

	return nil
}

// buildLinkSubIssueJob creates the link_sub_issue job
func (c *Compiler) buildLinkSubIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.LinkSubIssue == nil {
		return nil, fmt.Errorf("safe-outputs.link-sub-issue configuration is required")
	}

	maxCount := 5
	if data.SafeOutputs.LinkSubIssue.Max > 0 {
		maxCount = data.SafeOutputs.LinkSubIssue.Max
	}

	// Build custom environment variables specific to link-sub-issue
	var customEnvVars []string

	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LINK_SUB_ISSUE_MAX_COUNT: %d\n", maxCount))

	// Pass parent required labels
	if len(data.SafeOutputs.LinkSubIssue.ParentRequiredLabels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.LinkSubIssue.ParentRequiredLabels, ",")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LINK_SUB_ISSUE_PARENT_REQUIRED_LABELS: %q\n", labelsStr))
	}

	// Pass parent title prefix
	if data.SafeOutputs.LinkSubIssue.ParentTitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LINK_SUB_ISSUE_PARENT_TITLE_PREFIX: %q\n", data.SafeOutputs.LinkSubIssue.ParentTitlePrefix))
	}

	// Pass sub required labels
	if len(data.SafeOutputs.LinkSubIssue.SubRequiredLabels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.LinkSubIssue.SubRequiredLabels, ",")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LINK_SUB_ISSUE_SUB_REQUIRED_LABELS: %q\n", labelsStr))
	}

	// Pass sub title prefix
	if data.SafeOutputs.LinkSubIssue.SubTitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LINK_SUB_ISSUE_SUB_TITLE_PREFIX: %q\n", data.SafeOutputs.LinkSubIssue.SubTitlePrefix))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.LinkSubIssue.TargetRepoSlug)...)

	// Get token from config
	var token string
	if data.SafeOutputs.LinkSubIssue != nil {
		token = data.SafeOutputs.LinkSubIssue.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"linked_issues": "${{ steps.link_sub_issue.outputs.linked_issues }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "link_sub_issue",
		StepName:       "Link Sub-Issue",
		StepID:         "link_sub_issue",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getLinkSubIssueScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Token:          token,
		Condition:      BuildSafeOutputType("link_sub_issue"),
		TargetRepoSlug: data.SafeOutputs.LinkSubIssue.TargetRepoSlug,
	})
}
