package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var closeIssueLog = logger.New("workflow:close_issue")

// CloseIssuesConfig holds configuration for closing GitHub issues from agent output
type CloseIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	RequiredLabels       []string `yaml:"required-labels,omitempty"`       // Required labels for closing
	RequiredTitlePrefix  string   `yaml:"required-title-prefix,omitempty"` // Required title prefix for closing
	Target               string   `yaml:"target,omitempty"`                // Target for close: "triggering" (default), "*" (any issue), or explicit number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`           // Target repository for cross-repo operations
}

// parseCloseIssuesConfig handles close-issue configuration
func (c *Compiler) parseCloseIssuesConfig(outputMap map[string]any) *CloseIssuesConfig {
	if configData, exists := outputMap["close-issue"]; exists {
		closeIssueLog.Print("Parsing close-issue configuration")
		closeIssuesConfig := &CloseIssuesConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse required-labels
			if requiredLabels, exists := configMap["required-labels"]; exists {
				if labelList, ok := requiredLabels.([]any); ok {
					for _, label := range labelList {
						if labelStr, ok := label.(string); ok {
							closeIssuesConfig.RequiredLabels = append(closeIssuesConfig.RequiredLabels, labelStr)
						}
					}
				}
				closeIssueLog.Printf("Required labels configured: %v", closeIssuesConfig.RequiredLabels)
			}

			// Parse required-title-prefix
			if requiredTitlePrefix, exists := configMap["required-title-prefix"]; exists {
				if prefix, ok := requiredTitlePrefix.(string); ok {
					closeIssuesConfig.RequiredTitlePrefix = prefix
					closeIssueLog.Printf("Required title prefix configured: %q", prefix)
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					closeIssuesConfig.Target = targetStr
					closeIssueLog.Printf("Target configured: %q", targetStr)
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				closeIssueLog.Print("Invalid target-repo configuration")
				return nil // Invalid configuration, return nil to cause validation error
			}
			if targetRepoSlug != "" {
				closeIssueLog.Printf("Target repository configured: %s", targetRepoSlug)
			}
			closeIssuesConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &closeIssuesConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "close-issue:" with no value),
			// still set the default max
			closeIssuesConfig.Max = 1
		}

		return closeIssuesConfig
	}

	return nil
}

// buildCreateOutputCloseIssueJob creates the close_issue job
func (c *Compiler) buildCreateOutputCloseIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	closeIssueLog.Printf("Building close_issue job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.CloseIssues == nil {
		return nil, fmt.Errorf("safe-outputs.close-issue configuration is required")
	}

	// Build custom environment variables specific to close-issue
	var customEnvVars []string

	if len(data.SafeOutputs.CloseIssues.RequiredLabels) > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_ISSUE_REQUIRED_LABELS: %q\n", strings.Join(data.SafeOutputs.CloseIssues.RequiredLabels, ",")))
	}
	if data.SafeOutputs.CloseIssues.RequiredTitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX: %q\n", data.SafeOutputs.CloseIssues.RequiredTitlePrefix))
	}
	if data.SafeOutputs.CloseIssues.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_ISSUE_TARGET: %q\n", data.SafeOutputs.CloseIssues.Target))
	}
	closeIssueLog.Printf("Configured %d custom environment variables for issue close", len(customEnvVars))

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CloseIssues.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.close_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.close_issue.outputs.issue_url }}",
		"comment_url":  "${{ steps.close_issue.outputs.comment_url }}",
	}

	// Build job condition with issue event check only for "triggering" target
	// If target is "*" (any issue) or explicitly set, allow agent to provide issue_number
	jobCondition := BuildSafeOutputType("close_issue")
	if data.SafeOutputs.CloseIssues != nil &&
		(data.SafeOutputs.CloseIssues.Target == "" || data.SafeOutputs.CloseIssues.Target == "triggering") {
		// Only require event issue context for "triggering" target
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.issue.number"),
			BuildPropertyAccess("github.event.comment.issue.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "close_issue",
		StepName:       "Close Issue",
		StepID:         "close_issue",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCloseIssueScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          data.SafeOutputs.CloseIssues.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CloseIssues.TargetRepoSlug,
	})
}
