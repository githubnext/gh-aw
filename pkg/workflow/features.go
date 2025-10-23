package workflow

import (
	"os"
	"strings"
)

// IsFeatureEnabled checks if a feature flag is enabled in the GH_AW_FEATURES environment variable.
// The function performs case-insensitive string comparison and checks if the flag is present
// in the comma-separated list of features.
//
// Example: GH_AW_FEATURES="firewall,feature2,feature3"
func IsFeatureEnabled(flag string) bool {
	features := os.Getenv("GH_AW_FEATURES")
	if features == "" {
		return false
	}

	// Split by comma and check each feature
	featureList := strings.Split(features, ",")
	flagLower := strings.ToLower(strings.TrimSpace(flag))

	for _, feature := range featureList {
		if strings.ToLower(strings.TrimSpace(feature)) == flagLower {
			return true
		}
	}

	return false
}
