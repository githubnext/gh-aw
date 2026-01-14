package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var addCodeScanningAutofixLog = logger.New("workflow:add_code_scanning_autofix")

// AddCodeScanningAutofixConfig holds configuration for adding autofixes to code scanning alerts
type AddCodeScanningAutofixConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseAddCodeScanningAutofixConfig handles add-code-scanning-autofix configuration
func (c *Compiler) parseAddCodeScanningAutofixConfig(outputMap map[string]any) *AddCodeScanningAutofixConfig {
	if configData, exists := outputMap["add-code-scanning-autofix"]; exists {
		addCodeScanningAutofixLog.Print("Parsing add-code-scanning-autofix configuration")
		addCodeScanningAutofixConfig := &AddCodeScanningAutofixConfig{}
		addCodeScanningAutofixConfig.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &addCodeScanningAutofixConfig.BaseSafeOutputConfig)
		}

		return addCodeScanningAutofixConfig
	}

	return nil
}
