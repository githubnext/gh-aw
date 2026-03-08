package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// PlaywrightDockerArgs represents the common Docker arguments for Playwright container
type PlaywrightDockerArgs struct {
	ImageVersion      string // Version for Docker image (mcr.microsoft.com/playwright:version)
	MCPPackageVersion string // Version for NPM package (@playwright/mcp@version)
}

func getPlaywrightDockerImageVersion(playwrightConfig *PlaywrightToolConfig) string {
	playwrightDockerImageVersion := string(constants.DefaultPlaywrightBrowserVersion) // Default Playwright browser Docker image version
	// Extract version setting from tool properties
	if playwrightConfig != nil && playwrightConfig.Version != "" {
		playwrightDockerImageVersion = playwrightConfig.Version
	}
	return playwrightDockerImageVersion
}

// extractExpressionsFromPlaywrightArgs extracts all GitHub Actions expressions from playwright arguments
// Returns a map of environment variable names to their original expressions
// Uses the same ExpressionExtractor as used for shell script security
func extractExpressionsFromPlaywrightArgs(customArgs []string) map[string]string {
	log.Printf("Extracting expressions from %d Playwright args", len(customArgs))
	if len(customArgs) == 0 {
		return make(map[string]string)
	}

	// Join all arguments with a separator that won't appear in expressions
	combined := strings.Join(customArgs, "\n")

	// Use ExpressionExtractor to find all expressions
	extractor := NewExpressionExtractor()
	mappings, err := extractor.ExtractExpressions(combined)
	if err != nil {
		log.Printf("Failed to extract expressions from Playwright args: %s", err)
		return make(map[string]string)
	}

	// Convert to map of env var name -> original expression
	result := make(map[string]string)
	for _, mapping := range mappings {
		result[mapping.EnvVar] = mapping.Original
	}

	log.Printf("Extracted %d unique expressions from Playwright args", len(result))
	return result
}

// replaceExpressionsInPlaywrightArgs replaces all GitHub Actions expressions with environment variable references
// This prevents any expressions from being exposed in GitHub Actions logs
func replaceExpressionsInPlaywrightArgs(args []string, expressions map[string]string) []string {
	if len(expressions) == 0 {
		return args
	}

	// Create a temporary extractor with the same mappings
	combined := strings.Join(args, "\n")
	extractor := NewExpressionExtractor()
	_, _ = extractor.ExtractExpressions(combined)

	// Replace expressions in the combined string
	replaced := extractor.ReplaceExpressionsWithEnvVars(combined)

	// Split back into individual arguments
	return strings.Split(replaced, "\n")
}
