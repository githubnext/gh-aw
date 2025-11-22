package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var closePullRequestLog = logger.New("workflow:close_pull_request")

// ClosePullRequestsConfig holds configuration for closing GitHub pull requests from agent output
type ClosePullRequestsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	RequiredLabels       []string `yaml:"required-labels,omitempty"`       // Required labels for closing
	RequiredTitlePrefix  string   `yaml:"required-title-prefix,omitempty"` // Required title prefix for closing
	Target               string   `yaml:"target,omitempty"`                // Target for close: "triggering" (default), "*" (any PR), or explicit number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`           // Target repository for cross-repo operations
}

// parseClosePullRequestsConfig handles close-pull-request configuration
func (c *Compiler) parseClosePullRequestsConfig(outputMap map[string]any) *ClosePullRequestsConfig {
	if configData, exists := outputMap["close-pull-request"]; exists {
		closePullRequestLog.Print("Parsing close-pull-request configuration")
		closePullRequestsConfig := &ClosePullRequestsConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse required-labels
			if requiredLabels, exists := configMap["required-labels"]; exists {
				if labelList, ok := requiredLabels.([]any); ok {
					for _, label := range labelList {
						if labelStr, ok := label.(string); ok {
							closePullRequestsConfig.RequiredLabels = append(closePullRequestsConfig.RequiredLabels, labelStr)
						}
					}
				}
				closePullRequestLog.Printf("Required labels configured: %v", closePullRequestsConfig.RequiredLabels)
			}

			// Parse required-title-prefix
			if requiredTitlePrefix, exists := configMap["required-title-prefix"]; exists {
				if prefix, ok := requiredTitlePrefix.(string); ok {
					closePullRequestsConfig.RequiredTitlePrefix = prefix
					closePullRequestLog.Printf("Required title prefix configured: %q", prefix)
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					closePullRequestsConfig.Target = targetStr
					closePullRequestLog.Printf("Target configured: %q", targetStr)
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				closePullRequestLog.Print("Invalid target-repo configuration")
				return nil // Invalid configuration, return nil to cause validation error
			}
			if targetRepoSlug != "" {
				closePullRequestLog.Printf("Target repository configured: %s", targetRepoSlug)
			}
			closePullRequestsConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &closePullRequestsConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "close-pull-request:" with no value),
			// still set the default max
			closePullRequestsConfig.Max = 1
		}

		return closePullRequestsConfig
	}

	return nil
}

// buildCreateOutputClosePullRequestJob creates the close_pull_request job
func (c *Compiler) buildCreateOutputClosePullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	closePullRequestLog.Printf("Building close_pull_request job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.ClosePullRequests == nil {
		return nil, fmt.Errorf("safe-outputs.close-pull-request configuration is required")
	}

	// Build custom environment variables specific to close-pull-request
	var customEnvVars []string

	if len(data.SafeOutputs.ClosePullRequests.RequiredLabels) > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_PR_REQUIRED_LABELS: %q\n", strings.Join(data.SafeOutputs.ClosePullRequests.RequiredLabels, ",")))
	}
	if data.SafeOutputs.ClosePullRequests.RequiredTitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_PR_REQUIRED_TITLE_PREFIX: %q\n", data.SafeOutputs.ClosePullRequests.RequiredTitlePrefix))
	}
	if data.SafeOutputs.ClosePullRequests.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_PR_TARGET: %q\n", data.SafeOutputs.ClosePullRequests.Target))
	}
	closePullRequestLog.Printf("Configured %d custom environment variables for PR close", len(customEnvVars))

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.ClosePullRequests.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.close_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.close_pull_request.outputs.pull_request_url }}",
		"comment_url":         "${{ steps.close_pull_request.outputs.comment_url }}",
	}

	// Build job condition with pull request event check only for "triggering" target
	// If target is "*" (any PR) or explicitly set, allow agent to provide pull_request_number
	jobCondition := BuildSafeOutputType("close_pull_request")
	if data.SafeOutputs.ClosePullRequests != nil &&
		(data.SafeOutputs.ClosePullRequests.Target == "" || data.SafeOutputs.ClosePullRequests.Target == "triggering") {
		// Only require event PR context for "triggering" target
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.pull_request.number"),
			BuildPropertyAccess("github.event.comment.pull_request.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "close_pull_request",
		StepName:       "Close Pull Request",
		StepID:         "close_pull_request",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getClosePullRequestScript(),
		Permissions:    NewPermissionsContentsReadPRWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          data.SafeOutputs.ClosePullRequests.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.ClosePullRequests.TargetRepoSlug,
	})
}
