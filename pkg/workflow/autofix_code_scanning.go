package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var autofixCodeScanningLog = logger.New("workflow:autofix_code_scanning")

// AutofixCodeScanningConfig holds configuration for adding autofixes to code scanning alerts
type AutofixCodeScanningConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseAutofixCodeScanningConfig handles autofix-code-scanning configuration
func (c *Compiler) parseAutofixCodeScanningConfig(outputMap map[string]any) *AutofixCodeScanningConfig {
	if configData, exists := outputMap["autofix-code-scanning"]; exists {
		autofixCodeScanningLog.Print("Parsing autofix-code-scanning configuration")
		addCodeScanningAutofixConfig := &AutofixCodeScanningConfig{}
		addCodeScanningAutofixConfig.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields with default max of 10
			c.parseBaseSafeOutputConfig(configMap, &addCodeScanningAutofixConfig.BaseSafeOutputConfig, 10)
		}

		return addCodeScanningAutofixConfig
	}

	return nil
}
