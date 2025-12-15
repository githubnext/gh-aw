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

// UpdateEntityJobBuilder encapsulates entity-specific configuration for building update jobs
type UpdateEntityJobBuilder struct {
	EntityType          UpdateEntityType
	ConfigKey           string
	JobName             string
	StepName            string
	ScriptGetter        func() string
	PermissionsFunc     func() *Permissions
	BuildCustomEnvVars  func(*UpdateEntityConfig) []string
	BuildOutputs        func() map[string]string
	BuildEventCondition func(string) ConditionNode // Optional: builds event condition if target is empty
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

// buildUpdateEntityJobWithConfig is a higher-level helper that encapsulates the common pattern
// of building update entity jobs, reducing duplication across issue/PR/release builders
func (c *Compiler) buildUpdateEntityJobWithConfig(
	data *WorkflowData,
	mainJobName string,
	config *UpdateEntityConfig,
	builder UpdateEntityJobBuilder,
	logger *logger.Logger,
) (*Job, error) {
	if config == nil {
		return nil, fmt.Errorf("safe-outputs.%s configuration is required", builder.ConfigKey)
	}

	// Build entity-specific custom environment variables
	customEnvVars := builder.BuildCustomEnvVars(config)

	// Append target configuration environment variables
	customEnvVars = append(customEnvVars, BuildTargetEnvVar("GH_AW_UPDATE_TARGET", config.Target)...)

	// Build entity-specific outputs
	outputs := builder.BuildOutputs()

	// Build job condition with safe output type check
	jobCondition := BuildSafeOutputType(builder.JobName)

	// Add optional event condition if target is empty and event condition builder is provided
	if builder.BuildEventCondition != nil && config.Target == "" {
		eventCondition := builder.BuildEventCondition(config.Target)
		jobCondition = BuildAnd(jobCondition, eventCondition)
	}

	// Create UpdateEntityJobParams with all the configuration
	params := UpdateEntityJobParams{
		EntityType:      builder.EntityType,
		ConfigKey:       builder.ConfigKey,
		JobName:         builder.JobName,
		StepName:        builder.StepName,
		ScriptGetter:    builder.ScriptGetter,
		PermissionsFunc: builder.PermissionsFunc,
		CustomEnvVars:   customEnvVars,
		Outputs:         outputs,
		Condition:       jobCondition,
	}

	// Use the existing buildUpdateEntityJob to create the job
	return c.buildUpdateEntityJob(data, mainJobName, config, params, logger)
}
