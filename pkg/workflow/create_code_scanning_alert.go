package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var createCodeScanningAlertLog = logger.New("workflow:create_code_scanning_alert")

// CreateCodeScanningAlertsConfig holds configuration for creating repository security advisories (SARIF format) from agent output
type CreateCodeScanningAlertsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Driver               string   `yaml:"driver,omitempty"`        // Driver name for SARIF tool.driver.name field (default: "GitHub Agentic Workflows Security Scanner")
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository code scanning alert creation
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"` // List of additional repositories in format "owner/repo" that code scanning alerts can be created in
}

// parseCodeScanningAlertsConfig handles create-code-scanning-alert configuration
func (c *Compiler) parseCodeScanningAlertsConfig(outputMap map[string]any) *CreateCodeScanningAlertsConfig {
	if _, exists := outputMap["create-code-scanning-alert"]; !exists {
		return nil
	}

	createCodeScanningAlertLog.Print("Parsing create-code-scanning-alert configuration")
	configData := outputMap["create-code-scanning-alert"]
	securityReportsConfig := &CreateCodeScanningAlertsConfig{}

	if configMap, ok := configData.(map[string]any); ok {
		// Parse driver
		if driver, exists := configMap["driver"]; exists {
			if driverStr, ok := driver.(string); ok {
				securityReportsConfig.Driver = driverStr
				createCodeScanningAlertLog.Printf("Using custom SARIF driver name: %s", driverStr)
			}
		}

		// Parse target-repo
		securityReportsConfig.TargetRepoSlug = parseTargetRepoFromConfig(configMap)
		if securityReportsConfig.TargetRepoSlug != "" {
			createCodeScanningAlertLog.Printf("Target repo for code scanning alerts: %s", securityReportsConfig.TargetRepoSlug)
		}

		// Parse allowed-repos
		securityReportsConfig.AllowedRepos = parseAllowedReposFromConfig(configMap)
		if len(securityReportsConfig.AllowedRepos) > 0 {
			createCodeScanningAlertLog.Printf("Allowed repos for cross-repo alerts: %d configured", len(securityReportsConfig.AllowedRepos))
		}

		// Parse common base fields with default max of 0 (unlimited)
		c.parseBaseSafeOutputConfig(configMap, &securityReportsConfig.BaseSafeOutputConfig, 0)
	} else {
		// If configData is nil or not a map (e.g., "create-code-scanning-alert:" with no value),
		// still set the default max (nil = unlimited)
		createCodeScanningAlertLog.Print("No config map provided, using defaults (unlimited max)")
		securityReportsConfig.Max = nil
	}

	createCodeScanningAlertLog.Printf("Parsed create-code-scanning-alert config: driver=%q, target-repo=%q, allowed-repos=%d",
		securityReportsConfig.Driver, securityReportsConfig.TargetRepoSlug, len(securityReportsConfig.AllowedRepos))
	return securityReportsConfig
}
