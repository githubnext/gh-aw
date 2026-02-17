package cli

import (
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var githubLog = logger.New("cli:github")

// getGitHubHost returns the GitHub host URL from environment variables.
// Environment variables are checked in priority order for GitHub Enterprise support:
// 1. GITHUB_SERVER_URL - GitHub Actions standard (e.g., https://MYORG.ghe.com)
// 2. GITHUB_ENTERPRISE_HOST - GitHub Enterprise standard (e.g., MYORG.ghe.com)
// 3. GITHUB_HOST - GitHub Enterprise standard (e.g., MYORG.ghe.com)
// 4. GH_HOST - GitHub CLI standard (e.g., MYORG.ghe.com)
// 5. Defaults to https://github.com if none are set
//
// The function normalizes the URL by adding https:// if missing and removing trailing slashes.
func getGitHubHost() string {
	envVars := []string{"GITHUB_SERVER_URL", "GITHUB_ENTERPRISE_HOST", "GITHUB_HOST", "GH_HOST"}

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			githubLog.Printf("Resolved GitHub host from %s: %s", envVar, value)
			return normalizeGitHubHostURL(value)
		}
	}

	defaultHost := "https://github.com"
	githubLog.Printf("No GitHub host environment variable set, using default: %s", defaultHost)
	return defaultHost
}

// normalizeGitHubHostURL ensures the host URL has https:// scheme and no trailing slashes
func normalizeGitHubHostURL(rawHostURL string) string {
	// Remove all trailing slashes
	normalized := strings.TrimRight(rawHostURL, "/")

	// Add https:// scheme if no scheme is present
	if !strings.HasPrefix(normalized, "https://") && !strings.HasPrefix(normalized, "http://") {
		normalized = "https://" + normalized
	}

	return normalized
}
