package workflow

// TriggerWorkflowConfig holds configuration for workflow dispatch triggers
type TriggerWorkflowConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Allowed              []string `yaml:"allowed,omitempty"` // List of allowed workflow filenames
}

// parseTriggerWorkflowConfig handles trigger-workflow configuration
func (c *Compiler) parseTriggerWorkflowConfig(outputMap map[string]any) *TriggerWorkflowConfig {
	if configData, exists := outputMap["trigger-workflow"]; exists {
		config := &TriggerWorkflowConfig{
			Allowed: []string{}, // Default to empty (no workflows allowed)
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse allowed workflows
			if allowed, exists := configMap["allowed"]; exists {
				if allowedArray, ok := allowed.([]any); ok {
					var allowedStrings []string
					for _, workflow := range allowedArray {
						if workflowStr, ok := workflow.(string); ok {
							allowedStrings = append(allowedStrings, workflowStr)
						}
					}
					config.Allowed = allowedStrings
				}
			}

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig)
		} else if configData == nil {
			// Handle null case: create config with empty allowed list
			return config
		}

		return config
	}

	return nil
}
