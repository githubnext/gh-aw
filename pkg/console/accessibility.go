package console

import "os"

// IsAccessibleMode detects if accessibility mode should be enabled based on environment variables.
// Accessibility mode is enabled when:
// - ACCESSIBLE environment variable is set to any value
// - TERM environment variable is set to "dumb"
// - NO_COLOR environment variable is set to any value
//
// This function should be used by UI components to determine whether to:
// - Disable animations and spinners
// - Simplify interactive elements
// - Use plain text instead of fancy formatting
func IsAccessibleMode() bool {
	return os.Getenv("ACCESSIBLE") != "" ||
		os.Getenv("TERM") == "dumb" ||
		os.Getenv("NO_COLOR") != ""
}
