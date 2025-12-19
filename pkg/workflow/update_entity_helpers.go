package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

// UpdateEntityType represents the type of entity being updated
type UpdateEntityType string

const (
	UpdateEntityIssue       UpdateEntityType = "issue"
	UpdateEntityPullRequest UpdateEntityType = "pull_request"
	UpdateEntityDiscussion  UpdateEntityType = "discussion"
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

// parseUpdateEntityBase is a helper that reduces scaffolding duplication across update entity parsers.
// It handles the common pattern of:
//  1. Building UpdateEntityJobParams
//  2. Calling parseUpdateEntityConfig
//  3. Checking for nil result
//  4. Returning the base config and config map for entity-specific field parsing
//
// Returns:
//   - baseConfig: The parsed base configuration (nil if parsing failed)
//   - configMap: The entity-specific config map for additional field parsing (nil if not present)
//
// Callers should check if baseConfig is nil before proceeding with entity-specific parsing.
func (c *Compiler) parseUpdateEntityBase(
	outputMap map[string]any,
	entityType UpdateEntityType,
	configKey string,
	logger *logger.Logger,
) (*UpdateEntityConfig, map[string]any) {
	// Build params for base config parsing
	params := UpdateEntityJobParams{
		EntityType: entityType,
		ConfigKey:  configKey,
	}

	// Parse the base config (common fields like max, target, target-repo)
	baseConfig := c.parseUpdateEntityConfig(outputMap, params, logger, nil)
	if baseConfig == nil {
		return nil, nil
	}

	// Extract the config map for entity-specific field parsing
	var configMap map[string]any
	if configData, exists := outputMap[configKey]; exists {
		if cm, ok := configData.(map[string]any); ok {
			configMap = cm
		}
	}

	return baseConfig, configMap
}
