package workflow

import (
	"strings"
)

// computeAllowedDomainsForSanitization computes the allowed domains for sanitization
// based on the engine and network configuration, matching what's provided to the firewall
func (c *Compiler) computeAllowedDomainsForSanitization(data *WorkflowData) string {
	// Determine which engine is being used
	var engineID string
	if data.EngineConfig != nil {
		engineID = data.EngineConfig.ID
	} else if data.AI != "" {
		engineID = data.AI
	}

	// Compute domains based on engine type
	// For Copilot with firewall support, use GetCopilotAllowedDomains which merges
	// Copilot defaults with network permissions
	// For other engines, use GetAllowedDomains which uses network permissions only
	if engineID == "copilot" {
		return GetCopilotAllowedDomains(data.NetworkPermissions)
	}

	// For Claude, Codex, and other engines, use network permissions
	domains := GetAllowedDomains(data.NetworkPermissions)
	return strings.Join(domains, ",")
}
