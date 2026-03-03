package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var createAgentSessionLog = logger.New("workflow:create_agent_session")

// CreateAgentSessionConfig holds configuration for creating GitHub Copilot coding agent sessions from agent output
type CreateAgentSessionConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Base                 string   `yaml:"base,omitempty"`          // Base branch for the pull request
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository agent sessions
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"` // List of additional repositories that agent sessions can be created in (additionally to the target-repo)
}

// parseAgentSessionConfig handles create-agent-session configuration
func (c *Compiler) parseAgentSessionConfig(outputMap map[string]any) *CreateAgentSessionConfig {
	// Try new key first
	if configData, exists := outputMap["create-agent-session"]; exists {
		createAgentSessionLog.Print("Parsing create-agent-session configuration")
		agentSessionConfig := &CreateAgentSessionConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base branch
			if base, exists := configMap["base"]; exists {
				if baseStr, ok := base.(string); ok {
					agentSessionConfig.Base = baseStr
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			agentSessionConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &agentSessionConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "create-agent-session:" with no value),
			// still set the default max
			agentSessionConfig.Max = defaultIntStr(1)
		}

		return agentSessionConfig
	}

	// Fall back to deprecated key for backward compatibility
	if configData, exists := outputMap["create-agent-task"]; exists {
		createAgentSessionLog.Print("WARNING: Using deprecated 'create-agent-task' configuration. Please migrate to 'create-agent-session' using 'gh aw fix'")
		agentSessionConfig := &CreateAgentSessionConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base branch
			if base, exists := configMap["base"]; exists {
				if baseStr, ok := base.(string); ok {
					agentSessionConfig.Base = baseStr
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			agentSessionConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &agentSessionConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "create-agent-task:" with no value),
			// still set the default max
			agentSessionConfig.Max = defaultIntStr(1)
		}

		return agentSessionConfig
	}

	return nil
}
