package workflow

import (
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var featuresLog = logger.New("workflow:features")

// isFeatureEnabled checks if a feature flag is enabled by merging information from
// the frontmatter features field and the GH_AW_FEATURES environment variable.
// Features from frontmatter take precedence over environment variables.
//
// If workflowData is nil or has no features, it falls back to checking the environment variable only.
func isFeatureEnabled(flag string, workflowData *WorkflowData) bool {
	flagLower := strings.ToLower(strings.TrimSpace(flag))
	featuresLog.Printf("Checking if feature is enabled: %s", flagLower)

	// First, check if the feature is explicitly set in frontmatter
	if workflowData != nil && workflowData.Features != nil {
		if enabled, exists := workflowData.Features[flagLower]; exists {
			featuresLog.Printf("Feature found in frontmatter: %s=%v", flagLower, enabled)
			return enabled
		}
		// Also check case-insensitive match
		for key, enabled := range workflowData.Features {
			if strings.ToLower(key) == flagLower {
				featuresLog.Printf("Feature found in frontmatter (case-insensitive): %s=%v", flagLower, enabled)
				return enabled
			}
		}
	}

	// Fall back to checking the environment variable
	features := os.Getenv("GH_AW_FEATURES")
	if features == "" {
		featuresLog.Printf("Feature not found, GH_AW_FEATURES empty: %s=false", flagLower)
		return false
	}

	featuresLog.Printf("Checking GH_AW_FEATURES environment variable: %s", features)

	// Split by comma and check each feature
	featureList := strings.Split(features, ",")

	for _, feature := range featureList {
		if strings.ToLower(strings.TrimSpace(feature)) == flagLower {
			featuresLog.Printf("Feature found in GH_AW_FEATURES: %s=true", flagLower)
			return true
		}
	}

	featuresLog.Printf("Feature not found: %s=false", flagLower)
	return false
}
