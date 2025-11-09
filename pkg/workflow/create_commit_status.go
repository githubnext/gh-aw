package workflow

// CreateCommitStatusConfig holds configuration for creating commit statuses from agent output
type CreateCommitStatusConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Context              string   `yaml:"context,omitempty"`         // Context string to differentiate status (default: workflow name)
	AllowedDomains       []string `yaml:"allowed-domains,omitempty"` // Allowed domains for target_url validation (defaults to network allowed domains)
}

// parseCommitStatusConfig handles create-commit-status configuration
func (c *Compiler) parseCommitStatusConfig(outputMap map[string]any) *CreateCommitStatusConfig {
	if _, exists := outputMap["create-commit-status"]; !exists {
		return nil
	}

	configData := outputMap["create-commit-status"]
	commitStatusConfig := &CreateCommitStatusConfig{}
	commitStatusConfig.Max = 1 // Default and enforced max is 1 (only one commit status supported)

	if configMap, ok := configData.(map[string]any); ok {
		// Parse common base fields
		c.parseBaseSafeOutputConfig(configMap, &commitStatusConfig.BaseSafeOutputConfig)

		// Enforce max=1 for commit status (only one status per workflow run)
		if commitStatusConfig.Max != 1 {
			commitStatusConfig.Max = 1
		}

		// Parse context
		if context, exists := configMap["context"]; exists {
			if contextStr, ok := context.(string); ok {
				commitStatusConfig.Context = contextStr
			}
		}

		// Parse allowed-domains
		if allowedDomains, exists := configMap["allowed-domains"]; exists {
			if domainsArray, ok := allowedDomains.([]any); ok {
				var domainStrings []string
				for _, domain := range domainsArray {
					if domainStr, ok := domain.(string); ok {
						domainStrings = append(domainStrings, domainStr)
					}
				}
				commitStatusConfig.AllowedDomains = domainStrings
			}
		}
	}

	return commitStatusConfig
}
