package gitutil

import (
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var log = logger.New("gitutil:gitutil")

// IsAuthError checks if an error message indicates an authentication issue.
// This is used to detect when GitHub API calls fail due to missing or invalid credentials.
func IsAuthError(errMsg string) bool {
	log.Printf("Checking if error is auth-related: %s", errMsg)
	lowerMsg := strings.ToLower(errMsg)
	isAuth := strings.Contains(lowerMsg, "gh_token") ||
		strings.Contains(lowerMsg, "github_token") ||
		strings.Contains(lowerMsg, "authentication") ||
		strings.Contains(lowerMsg, "not logged into") ||
		strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "forbidden") ||
		strings.Contains(lowerMsg, "permission denied")
	if isAuth {
		log.Print("Detected authentication error")
	}
	return isAuth
}

// IsHexString checks if a string contains only hexadecimal characters.
// This is used to validate Git commit SHAs and other hexadecimal identifiers.
func IsHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// GetGitHubHost resolves GitHub server URL from environment in priority order.
// Checks environment variables in this order:
//  1. GITHUB_SERVER_URL (e.g., https://MYORG.ghe.com)
//  2. GITHUB_ENTERPRISE_HOST (e.g., MYORG.ghe.com)
//  3. GITHUB_HOST (e.g., MYORG.ghe.com)
//  4. GH_HOST (e.g., MYORG.ghe.com)
//
// Returns normalized URL with https:// scheme and no trailing slash.
// Defaults to https://github.com if no environment variables are set.
func GetGitHubHost() string {
	envVars := []string{"GITHUB_SERVER_URL", "GITHUB_ENTERPRISE_HOST", "GITHUB_HOST", "GH_HOST"}

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			log.Printf("GitHub host from %s: %s", envVar, value)
			return sanitizeHostURL(value)
		}
	}

	defaultHost := "https://github.com"
	log.Printf("Using fallback GitHub host: %s", defaultHost)
	return defaultHost
}

// sanitizeHostURL ensures proper URL format with https:// and no trailing slash
func sanitizeHostURL(rawHost string) string {
	normalized := strings.TrimRight(rawHost, "/")

	if !strings.HasPrefix(normalized, "https://") && !strings.HasPrefix(normalized, "http://") {
		normalized = "https://" + normalized
	}

	return normalized
}
