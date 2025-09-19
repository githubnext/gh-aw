package cli

import "os"

// IsRunningInCI checks if we're running in a CI environment
func IsRunningInCI() bool {
	// Common CI environment variables
	ciVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
	}

	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}
