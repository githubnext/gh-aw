package workflow

import "github.com/github/gh-aw/pkg/logger"

var createCheckRunLog = logger.New("workflow:create_check_run")

// CreateCheckRunConfig holds configuration for creating GitHub Check Runs from agent output
type CreateCheckRunConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Name                 string `yaml:"name,omitempty"` // Check run name shown in the GitHub Checks UI
}

// parseCreateCheckRunConfig handles create-check-run configuration
func (c *Compiler) parseCreateCheckRunConfig(outputMap map[string]any) *CreateCheckRunConfig {
	if _, exists := outputMap["create-check-run"]; !exists {
		return nil
	}

	createCheckRunLog.Print("Parsing create-check-run configuration")
	configData := outputMap["create-check-run"]
	checkRunConfig := &CreateCheckRunConfig{}

	if configMap, ok := configData.(map[string]any); ok {
		// Parse name
		if name, exists := configMap["name"]; exists {
			if nameStr, ok := name.(string); ok {
				checkRunConfig.Name = nameStr
				createCheckRunLog.Printf("Using custom check run name: %s", nameStr)
			}
		}

		// Parse common base fields with default max of 1
		c.parseBaseSafeOutputConfig(configMap, &checkRunConfig.BaseSafeOutputConfig, 1)
	} else {
		// If configData is nil or not a map (e.g., "create-check-run:" with no value),
		// still set the default max of 1
		createCheckRunLog.Print("No config map provided, using defaults (max=1)")
		checkRunConfig.Max = defaultIntStr(1)
	}

	createCheckRunLog.Printf("Parsed create-check-run config: name=%q", checkRunConfig.Name)
	return checkRunConfig
}
