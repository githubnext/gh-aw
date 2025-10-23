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
