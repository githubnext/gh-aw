package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var linkSubIssueLog = logger.New("workflow:link_sub_issue")

// LinkSubIssueConfig holds configuration for linking issues as sub-issues from agent output
type LinkSubIssueConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	ParentRequiredLabels   []string `yaml:"parent-required-labels,omitempty"` // Required labels the parent issue must have
	ParentTitlePrefix      string   `yaml:"parent-title-prefix,omitempty"`    // Required title prefix for parent issue
	SubRequiredLabels      []string `yaml:"sub-required-labels,omitempty"`    // Required labels the sub-issue must have
	SubTitlePrefix         string   `yaml:"sub-title-prefix,omitempty"`       // Required title prefix for sub-issue
}

// parseLinkSubIssueConfig handles link-sub-issue configuration
func (c *Compiler) parseLinkSubIssueConfig(outputMap map[string]any) *LinkSubIssueConfig {
	linkSubIssueLog.Print("Parsing link-sub-issue configuration")
	if configData, exists := outputMap["link-sub-issue"]; exists {
		linkSubIssueConfig := &LinkSubIssueConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			linkSubIssueLog.Print("Found link-sub-issue config map")

			// Parse target config (target-repo)
			targetConfig, _ := ParseTargetConfig(configMap)
			linkSubIssueConfig.SafeOutputTargetConfig = targetConfig

			// Parse common base fields with default max of 5
			c.parseBaseSafeOutputConfig(configMap, &linkSubIssueConfig.BaseSafeOutputConfig, 5)

			// Parse parent-required-labels
			linkSubIssueConfig.ParentRequiredLabels = ParseStringArrayFromConfig(configMap, "parent-required-labels")

			// Parse parent-title-prefix
			linkSubIssueConfig.ParentTitlePrefix = ParseStringFromConfig(configMap, "parent-title-prefix")

			// Parse sub-required-labels
			linkSubIssueConfig.SubRequiredLabels = ParseStringArrayFromConfig(configMap, "sub-required-labels")

			// Parse sub-title-prefix
			linkSubIssueConfig.SubTitlePrefix = ParseStringFromConfig(configMap, "sub-title-prefix")

			linkSubIssueLog.Printf("Parsed link-sub-issue config: max=%d, parent_labels=%d, sub_labels=%d, target_repo=%s",
				linkSubIssueConfig.Max, len(linkSubIssueConfig.ParentRequiredLabels),
				len(linkSubIssueConfig.SubRequiredLabels), linkSubIssueConfig.TargetRepoSlug)
		} else {
			// If configData is nil or not a map, still set the default max
			linkSubIssueConfig.Max = 5
		}

		return linkSubIssueConfig
	}

	return nil
}

// buildLinkSubIssueJob creates the link_sub_issue job
func (c *Compiler) buildLinkSubIssueJob(data *WorkflowData, mainJobName string, createIssueJobName string) (*Job, error) {
	linkSubIssueLog.Printf("Building link_sub_issue job: main_job=%s, create_issue_job=%s", mainJobName, createIssueJobName)
	if data.SafeOutputs == nil || data.SafeOutputs.LinkSubIssue == nil {
		return nil, fmt.Errorf("safe-outputs.link-sub-issue configuration is required")
	}

	cfg := data.SafeOutputs.LinkSubIssue

	maxCount := 5
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables specific to link-sub-issue
	var customEnvVars []string

	// Pass the max limit
	customEnvVars = append(customEnvVars, BuildMaxCountEnvVar("GH_AW_LINK_SUB_ISSUE_MAX_COUNT", maxCount)...)

	// Pass parent required labels
	if len(cfg.ParentRequiredLabels) > 0 {
		customEnvVars = append(customEnvVars, BuildRequiredLabelsEnvVar("GH_AW_LINK_SUB_ISSUE_PARENT_REQUIRED_LABELS", cfg.ParentRequiredLabels)...)
	}

	// Pass parent title prefix
	customEnvVars = append(customEnvVars, BuildRequiredTitlePrefixEnvVar("GH_AW_LINK_SUB_ISSUE_PARENT_TITLE_PREFIX", cfg.ParentTitlePrefix)...)

	// Pass sub required labels
	if len(cfg.SubRequiredLabels) > 0 {
		customEnvVars = append(customEnvVars, BuildRequiredLabelsEnvVar("GH_AW_LINK_SUB_ISSUE_SUB_REQUIRED_LABELS", cfg.SubRequiredLabels)...)
	}

	// Pass sub title prefix
	customEnvVars = append(customEnvVars, BuildRequiredTitlePrefixEnvVar("GH_AW_LINK_SUB_ISSUE_SUB_TITLE_PREFIX", cfg.SubTitlePrefix)...)

	// Add environment variable for temporary ID map from create_issue job
	if createIssueJobName != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TEMPORARY_ID_MAP: ${{ needs.%s.outputs.temporary_id_map }}\n", createIssueJobName))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"linked_issues": "${{ steps.link_sub_issue.outputs.linked_issues }}",
	}

	// Build the needs list - always depend on mainJobName, and conditionally on create_issue
	needs := []string{mainJobName}
	if createIssueJobName != "" {
		needs = append(needs, createIssueJobName)
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
		Needs:          needs,
		Token:          cfg.GitHubToken,
		Condition:      BuildSafeOutputType("link_sub_issue"),
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
