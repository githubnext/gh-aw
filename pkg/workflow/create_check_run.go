package workflow

import "github.com/github/gh-aw/pkg/logger"

var createCheckRunLog = logger.New("workflow:create_check_run")

// CreateCheckRunOutputConfig holds optional static defaults for the check run output fields
type CreateCheckRunOutputConfig struct {
	Title   string `yaml:"title,omitempty"`   // Optional fallback title (max 256 chars)
	Summary string `yaml:"summary,omitempty"` // Optional fallback summary (max 65535 chars)
}

// CreateCheckRunConfig holds configuration for creating GitHub Check Runs from agent output
type CreateCheckRunConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Name                 string                      `yaml:"name,omitempty"`   // Check run name shown in the GitHub Checks UI
	Output               *CreateCheckRunOutputConfig `yaml:"output,omitempty"` // Optional static output defaults
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

		// Parse optional output defaults block
		if outputVal, exists := configMap["output"]; exists {
			if outputConfigMap, ok := outputVal.(map[string]any); ok {
				outputCfg := &CreateCheckRunOutputConfig{}
				if title, ok := outputConfigMap["title"].(string); ok {
					outputCfg.Title = title
					createCheckRunLog.Printf("Using config output.title: %q", title)
				}
				if summary, ok := outputConfigMap["summary"].(string); ok {
					outputCfg.Summary = summary
					createCheckRunLog.Printf("Using config output.summary (len=%d)", len(summary))
				}
				checkRunConfig.Output = outputCfg
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
