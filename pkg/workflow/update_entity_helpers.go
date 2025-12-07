package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

// UpdateEntityType represents the type of entity being updated
type UpdateEntityType string

const (
	UpdateEntityIssue       UpdateEntityType = "issue"
	UpdateEntityPullRequest UpdateEntityType = "pull_request"
	UpdateEntityRelease     UpdateEntityType = "release"
)

// UpdateEntityConfig holds the configuration for an update entity operation
type UpdateEntityConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	// Type-specific fields are stored in the concrete config structs
}

// UpdateEntityJobParams holds the parameters needed to build an update entity job
type UpdateEntityJobParams struct {
	EntityType      UpdateEntityType
	ConfigKey       string // e.g., "update-issue", "update-pull-request"
	JobName         string // e.g., "update_issue", "update_pull_request"
	StepName        string // e.g., "Update Issue", "Update Pull Request"
	ScriptGetter    func() string
	PermissionsFunc func() *Permissions
	CustomEnvVars   []string          // Type-specific environment variables
	Outputs         map[string]string // Type-specific outputs
	Condition       ConditionNode     // Job condition expression
}

// parseUpdateEntityConfig is a generic function to parse update entity configurations
func (c *Compiler) parseUpdateEntityConfig(outputMap map[string]any, params UpdateEntityJobParams, logger *logger.Logger, parseSpecificFields func(map[string]any, *UpdateEntityConfig)) *UpdateEntityConfig {
	if configData, exists := outputMap[params.ConfigKey]; exists {
		logger.Printf("Parsing %s configuration", params.ConfigKey)
		config := &UpdateEntityConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse target config (target, target-repo) with validation
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				logger.Print("Invalid target-repo configuration")
				return nil
			}
			config.SafeOutputTargetConfig = targetConfig

			// Parse type-specific fields if provided
			if parseSpecificFields != nil {
				parseSpecificFields(configMap, config)
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

// buildUpdateEntityJob is a generic function to build update entity jobs
func (c *Compiler) buildUpdateEntityJob(data *WorkflowData, mainJobName string, config *UpdateEntityConfig, params UpdateEntityJobParams, logger *logger.Logger) (*Job, error) {
	logger.Printf("Building %s job for workflow: %s", params.JobName, data.Name)

	if config == nil {
		return nil, fmt.Errorf("safe-outputs.%s configuration is required", params.ConfigKey)
	}

	// Add standard environment variables (metadata + staged/target repo)
	allEnvVars := append(params.CustomEnvVars, c.buildStandardSafeOutputEnvVars(data, config.TargetRepoSlug)...)

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        params.JobName,
		StepName:       params.StepName,
		StepID:         params.JobName,
		MainJobName:    mainJobName,
		CustomEnvVars:  allEnvVars,
		Script:         params.ScriptGetter(),
		Permissions:    params.PermissionsFunc(),
		Outputs:        params.Outputs,
		Condition:      params.Condition,
		Token:          config.GitHubToken,
		TargetRepoSlug: config.TargetRepoSlug,
	})
}
