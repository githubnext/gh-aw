package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var assignToAgentLog = logger.New("workflow:assign_to_agent")

// AssignToAgentConfig holds configuration for assigning agents to issues from agent output
type AssignToAgentConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	DefaultAgent           string `yaml:"name,omitempty"` // Default agent to assign (e.g., "copilot")
}

// parseAssignToAgentConfig handles assign-to-agent configuration
func (c *Compiler) parseAssignToAgentConfig(outputMap map[string]any) *AssignToAgentConfig {
	if assignToAgent, exists := outputMap["assign-to-agent"]; exists {
		assignToAgentLog.Print("Parsing assign-to-agent configuration")
		if agentMap, ok := assignToAgent.(map[string]any); ok {
			agentConfig := &AssignToAgentConfig{}

			// Parse name (optional - specific to assign-to-agent)
			if defaultAgent, exists := agentMap["name"]; exists {
				if defaultAgentStr, ok := defaultAgent.(string); ok {
					agentConfig.DefaultAgent = defaultAgentStr
				}
			}

			// Parse target config (target, target-repo) - validation errors are handled gracefully
			targetConfig, _ := ParseTargetConfig(agentMap)
			agentConfig.SafeOutputTargetConfig = targetConfig

			// Parse common base fields (github-token, max) with default max of 3
			c.parseBaseSafeOutputConfig(agentMap, &agentConfig.BaseSafeOutputConfig, 3)
			assignToAgentLog.Printf("Parsed assign-to-agent config: default_agent=%s, target=%s", agentConfig.DefaultAgent, agentConfig.Target)

			return agentConfig
		} else if assignToAgent == nil {
			// Handle null case: create empty config with default max of 3
			config := &AssignToAgentConfig{}
			config.Max = 3
			return config
		}
	}

	return nil
}
