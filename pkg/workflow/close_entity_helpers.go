// Package workflow provides helper functions for closing GitHub entities.
//
// This file contains shared utilities for building close entity jobs (issues,
// pull requests, discussions). These helpers extract common patterns used across
// the three close entity implementations to reduce code duplication and ensure
// consistency in configuration parsing and job generation.
//
// # Organization Rationale
//
// These close entity helpers are grouped here because they:
//   - Provide generic close entity functionality used by 3 entity types
//   - Share common configuration patterns (target, filters, max)
//   - Follow a consistent entity registry pattern
//   - Enable DRY principles for close operations
//
// This follows the helper file conventions documented in the developer instructions.
// See skills/developer/SKILL.md#helper-file-conventions for details.
//
// # Key Functions
//
// Configuration Parsing:
//   - parseCloseEntityConfig() - Generic close entity configuration parser
//   - parseCloseIssuesConfig() - Parse close-issue configuration
//   - parseClosePullRequestsConfig() - Parse close-pull-request configuration
//   - parseCloseDiscussionsConfig() - Parse close-discussion configuration
//
// Entity Registry:
//   - closeEntityRegistry - Central registry of all close entity definitions
//   - closeEntityDefinition - Definition structure for close entity types
//
// # Usage Patterns
//
// The close entity helpers follow a registry pattern where each entity type
// (issue, pull request, discussion) is defined with its specific parameters
// (config keys, environment variables, permissions, scripts). This allows:
//   - Consistent configuration parsing across entity types
//   - Easy addition of new close entity types
//   - Centralized entity type definitions
//
// # When to Use vs Alternatives
//
// Use these helpers when:
//   - Implementing close operations for GitHub entities
//   - Parsing close entity configurations from workflow YAML
//   - Building close entity jobs with consistent patterns
//
// For create/update operations, see:
//   - create_*.go files for entity creation logic
//   - update_entity_helpers.go for entity update logic
package workflow

import (
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
				logger.Printf("Parsing discussion-specific configuration for %s", params.ConfigKey)
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
				logger.Printf("Parsing standard close job configuration for %s", params.EntityType)
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
			logger.Printf("Parsed %s configuration: max=%d, target=%s", params.ConfigKey, config.Max, config.Target)
		} else {
			// If configData is nil or not a map, still set the default max
			logger.Print("Config is not a map, using default max=1")
			config.Max = 1
		}

		return config
	}

	logger.Printf("No configuration found for %s", params.ConfigKey)
	return nil
}

// closeEntityDefinition holds all parameters for a close entity type
type closeEntityDefinition struct {
	EntityType       CloseEntityType
	ConfigKey        string
	EnvVarPrefix     string
	JobName          string
	StepName         string
	OutputNumberKey  string
	OutputURLKey     string
	EventNumberPath1 string
	EventNumberPath2 string
	ScriptGetter     func() string
	PermissionsFunc  func() *Permissions
	Logger           *logger.Logger
}

// closeEntityRegistry holds all close entity definitions
var closeEntityRegistry = []closeEntityDefinition{
	{
		EntityType:       CloseEntityIssue,
		ConfigKey:        "close-issue",
		EnvVarPrefix:     "GH_AW_CLOSE_ISSUE",
		JobName:          "close_issue",
		StepName:         "Close Issue",
		OutputNumberKey:  "issue_number",
		OutputURLKey:     "issue_url",
		EventNumberPath1: "github.event.issue.number",
		EventNumberPath2: "github.event.comment.issue.number",
		ScriptGetter:     getCloseIssueScript,
		PermissionsFunc:  NewPermissionsContentsReadIssuesWrite,
		Logger:           logger.New("workflow:close_issue"),
	},
	{
		EntityType:       CloseEntityPullRequest,
		ConfigKey:        "close-pull-request",
		EnvVarPrefix:     "GH_AW_CLOSE_PR",
		JobName:          "close_pull_request",
		StepName:         "Close Pull Request",
		OutputNumberKey:  "pull_request_number",
		OutputURLKey:     "pull_request_url",
		EventNumberPath1: "github.event.pull_request.number",
		EventNumberPath2: "github.event.comment.pull_request.number",
		ScriptGetter:     getClosePullRequestScript,
		PermissionsFunc:  NewPermissionsContentsReadPRWrite,
		Logger:           logger.New("workflow:close_pull_request"),
	},
	{
		EntityType:       CloseEntityDiscussion,
		ConfigKey:        "close-discussion",
		EnvVarPrefix:     "GH_AW_CLOSE_DISCUSSION",
		JobName:          "close_discussion",
		StepName:         "Close Discussion",
		OutputNumberKey:  "discussion_number",
		OutputURLKey:     "discussion_url",
		EventNumberPath1: "github.event.discussion.number",
		EventNumberPath2: "github.event.comment.discussion.number",
		ScriptGetter:     getCloseDiscussionScript,
		PermissionsFunc:  NewPermissionsContentsReadDiscussionsWrite,
		Logger:           logger.New("workflow:close_discussion"),
	},
}

// Type aliases for backward compatibility
type CloseIssuesConfig = CloseEntityConfig
type ClosePullRequestsConfig = CloseEntityConfig
type CloseDiscussionsConfig = CloseEntityConfig

// parseCloseIssuesConfig handles close-issue configuration
func (c *Compiler) parseCloseIssuesConfig(outputMap map[string]any) *CloseIssuesConfig {
	def := closeEntityRegistry[0] // issue
	params := CloseEntityJobParams{
		EntityType: def.EntityType,
		ConfigKey:  def.ConfigKey,
	}
	return c.parseCloseEntityConfig(outputMap, params, def.Logger)
}

// parseClosePullRequestsConfig handles close-pull-request configuration
func (c *Compiler) parseClosePullRequestsConfig(outputMap map[string]any) *ClosePullRequestsConfig {
	def := closeEntityRegistry[1] // pull request
	params := CloseEntityJobParams{
		EntityType: def.EntityType,
		ConfigKey:  def.ConfigKey,
	}
	return c.parseCloseEntityConfig(outputMap, params, def.Logger)
}

// parseCloseDiscussionsConfig handles close-discussion configuration
func (c *Compiler) parseCloseDiscussionsConfig(outputMap map[string]any) *CloseDiscussionsConfig {
	def := closeEntityRegistry[2] // discussion
	params := CloseEntityJobParams{
		EntityType: def.EntityType,
		ConfigKey:  def.ConfigKey,
	}
	return c.parseCloseEntityConfig(outputMap, params, def.Logger)
}
