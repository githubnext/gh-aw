package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

// CloseEntityType represents the type of entity being closed
type CloseEntityType string

const (
	CloseEntityIssue       CloseEntityType = "issue"
	CloseEntityPullRequest CloseEntityType = "pull_request"
	CloseEntityDiscussion  CloseEntityType = "discussion"
)

// CloseEntityConfig holds the configuration for a close entity operation
type CloseEntityConfig struct {
	BaseSafeOutputConfig             `yaml:",inline"`
	SafeOutputTargetConfig           `yaml:",inline"`
	SafeOutputFilterConfig           `yaml:",inline"`
	SafeOutputDiscussionFilterConfig `yaml:",inline"` // Only used for discussions
}

// CloseEntityJobParams holds the parameters needed to build a close entity job
type CloseEntityJobParams struct {
	EntityType       CloseEntityType
	ConfigKey        string // e.g., "close-issue", "close-pull-request"
	EnvVarPrefix     string // e.g., "GH_AW_CLOSE_ISSUE", "GH_AW_CLOSE_PR"
	JobName          string // e.g., "close_issue", "close_pull_request"
	StepName         string // e.g., "Close Issue", "Close Pull Request"
	OutputNumberKey  string // e.g., "issue_number", "pull_request_number"
	OutputURLKey     string // e.g., "issue_url", "pull_request_url"
	EventNumberPath1 string // e.g., "github.event.issue.number"
	EventNumberPath2 string // e.g., "github.event.comment.issue.number"
	ScriptGetter     func() string
	PermissionsFunc  func() *Permissions
}

// parseCloseEntityConfig is a generic function to parse close entity configurations
func (c *Compiler) parseCloseEntityConfig(outputMap map[string]any, params CloseEntityJobParams, logger *logger.Logger) *CloseEntityConfig {
	if configData, exists := outputMap[params.ConfigKey]; exists {
		logger.Printf("Parsing %s configuration", params.ConfigKey)
		config := &CloseEntityConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// For discussions, parse target and filter configs separately
			if params.EntityType == CloseEntityDiscussion {
				// Parse target config
				targetConfig, isInvalid := ParseTargetConfig(configMap)
				if isInvalid {
					logger.Print("Invalid target-repo configuration")
					return nil
				}
				config.SafeOutputTargetConfig = targetConfig

				// Parse discussion filter config (includes required-category)
				config.SafeOutputDiscussionFilterConfig = ParseDiscussionFilterConfig(configMap)
				config.SafeOutputFilterConfig = config.SafeOutputDiscussionFilterConfig.SafeOutputFilterConfig
			} else {
				// For issues and PRs, use the standard close job config parser
				closeJobConfig, isInvalid := ParseCloseJobConfig(configMap)
				if isInvalid {
					logger.Print("Invalid target-repo configuration")
					return nil
				}
				config.SafeOutputTargetConfig = closeJobConfig.SafeOutputTargetConfig
				config.SafeOutputFilterConfig = closeJobConfig.SafeOutputFilterConfig
			}

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map, still set the default max
			config.Max = 1
		}

		return config
	}

	return nil
}

// buildCloseEntityJob is a generic function to build close entity jobs
func (c *Compiler) buildCloseEntityJob(data *WorkflowData, mainJobName string, config *CloseEntityConfig, params CloseEntityJobParams, logger *logger.Logger) (*Job, error) {
	logger.Printf("Building %s job for workflow: %s", params.JobName, data.Name)

	if config == nil {
		return nil, fmt.Errorf("safe-outputs.%s configuration is required", params.ConfigKey)
	}

	// Build custom environment variables specific to this close entity
	closeJobConfig := CloseJobConfig{
		SafeOutputTargetConfig: config.SafeOutputTargetConfig,
		SafeOutputFilterConfig: config.SafeOutputFilterConfig,
	}
	customEnvVars := BuildCloseJobEnvVars(params.EnvVarPrefix, closeJobConfig)

	// Add required-category env var for discussions
	if params.EntityType == CloseEntityDiscussion {
		customEnvVars = append(customEnvVars, BuildRequiredCategoryEnvVar(params.EnvVarPrefix+"_REQUIRED_CATEGORY", config.RequiredCategory)...)
	}

	logger.Printf("Configured %d custom environment variables for %s close", len(customEnvVars), params.EntityType)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, config.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		params.OutputNumberKey: fmt.Sprintf("${{ steps.%s.outputs.%s }}", params.JobName, params.OutputNumberKey),
		params.OutputURLKey:    fmt.Sprintf("${{ steps.%s.outputs.%s }}", params.JobName, params.OutputURLKey),
		"comment_url":          fmt.Sprintf("${{ steps.%s.outputs.comment_url }}", params.JobName),
	}

	// Build job condition with event check only for "triggering" target
	// If target is "*" (any entity) or explicitly set, allow agent to provide the entity number
	jobCondition := BuildSafeOutputType(params.JobName)
	if config.Target == "" || config.Target == "triggering" {
		// Only require event context for "triggering" target
		eventCondition := buildOr(
			BuildPropertyAccess(params.EventNumberPath1),
			BuildPropertyAccess(params.EventNumberPath2),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        params.JobName,
		StepName:       params.StepName,
		StepID:         params.JobName,
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         params.ScriptGetter(),
		Permissions:    params.PermissionsFunc(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          config.GitHubToken,
		TargetRepoSlug: config.TargetRepoSlug,
	})
}
