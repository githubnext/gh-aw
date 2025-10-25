package workflow

import (
	"os"
	"strings"
)

// isFeatureEnabled checks if a feature flag is enabled by merging information from
// the frontmatter features field and the GH_AW_FEATURES environment variable.
// Features from frontmatter take precedence over environment variables.
//
// If workflowData is nil or has no features, it falls back to checking the environment variable only.
//
// Special behavior for the "firewall" feature with copilot engine:
// - Defaults to true for copilot engine when not explicitly set
// - Can be explicitly disabled by setting features: { firewall: false }
func isFeatureEnabled(flag string, workflowData *WorkflowData) bool {
	flagLower := strings.ToLower(strings.TrimSpace(flag))

	// First, check if the feature is explicitly set in frontmatter
	if workflowData != nil && workflowData.Features != nil {
		if enabled, exists := workflowData.Features[flagLower]; exists {
			return enabled
		}
		// Also check case-insensitive match
		for key, enabled := range workflowData.Features {
			if strings.ToLower(key) == flagLower {
				return enabled
			}
		}
	}

	// Special handling: default firewall to true for copilot engine
	if flagLower == "firewall" && workflowData != nil && workflowData.EngineConfig != nil {
		if workflowData.EngineConfig.ID == "copilot" {
			// Firewall is enabled by default for copilot engine
			// (only if not explicitly set to false in Features above)
			return true
		}
	}

	// Fall back to checking the environment variable
	features := os.Getenv("GH_AW_FEATURES")
	if features == "" {
		return false
	}

	// Split by comma and check each feature
	featureList := strings.Split(features, ",")

	for _, feature := range featureList {
		if strings.ToLower(strings.TrimSpace(feature)) == flagLower {
			return true
		}
	}

	return false
}
