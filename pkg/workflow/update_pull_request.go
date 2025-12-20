package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updatePullRequestLog = logger.New("workflow:update_pull_request")

// UpdatePullRequestsConfig holds configuration for updating GitHub pull requests from agent output
type UpdatePullRequestsConfig struct {
	UpdateEntityConfig `yaml:",inline"`
	Title              *bool `yaml:"title,omitempty"` // Allow updating PR title - defaults to true, set to false to disable
	Body               *bool `yaml:"body,omitempty"`  // Allow updating PR body - defaults to true, set to false to disable
}

// parseUpdatePullRequestsConfig handles update-pull-request configuration
func (c *Compiler) parseUpdatePullRequestsConfig(outputMap map[string]any) *UpdatePullRequestsConfig {
	updatePullRequestLog.Print("Parsing update pull request configuration")

	// Create config struct
	cfg := &UpdatePullRequestsConfig{}

	// Parse base config and entity-specific fields using generic helper
	baseConfig, _ := c.parseUpdateEntityConfigWithFields(outputMap, UpdateEntityParseOptions{
		EntityType: UpdateEntityPullRequest,
		ConfigKey:  "update-pull-request",
		Logger:     updatePullRequestLog,
		Fields: []UpdateEntityFieldSpec{
			{Name: "title", Mode: FieldParsingBoolValue, Dest: &cfg.Title},
			{Name: "body", Mode: FieldParsingBoolValue, Dest: &cfg.Body},
		},
	})
	if baseConfig == nil {
		return nil
	}

	// Set base fields
	cfg.UpdateEntityConfig = *baseConfig
	return cfg
}
