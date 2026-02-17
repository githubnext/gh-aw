//go:build !js && !wasm

package parser

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var githubLog = logger.New("parser:github")

// GetGitHubHost returns the GitHub host URL from environment variables.
// Environment variables are checked in priority order for GitHub Enterprise support:
// 1. GITHUB_SERVER_URL - GitHub Actions standard (e.g., https://MYORG.ghe.com)
// 2. GITHUB_ENTERPRISE_HOST - GitHub Enterprise standard (e.g., MYORG.ghe.com)
// 3. GITHUB_HOST - GitHub Enterprise standard (e.g., MYORG.ghe.com)
// 4. GH_HOST - GitHub CLI standard (e.g., MYORG.ghe.com)
// 5. Defaults to https://github.com if none are set
//
// The function normalizes the URL by adding https:// if missing and removing trailing slashes.
func GetGitHubHost() string {
	envVars := []string{"GITHUB_SERVER_URL", "GITHUB_ENTERPRISE_HOST", "GITHUB_HOST", "GH_HOST"}

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			githubLog.Printf("Resolved GitHub host from %s: %s", envVar, value)
			return normalizeGitHubHostURL(value)
		}
	}

	defaultHost := string(constants.PublicGitHubHost)
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

// GetGitHubHostForRepo returns the GitHub host URL for a specific repository.
// The gh-aw repository (github/gh-aw) always uses public GitHub (https://github.com)
// regardless of enterprise GitHub host settings, since gh-aw itself is only available
// on public GitHub. For all other repositories, it uses GetGitHubHost().
func GetGitHubHostForRepo(owner, repo string) string {
	// The gh-aw repository is always on public GitHub
	if owner == "github" && repo == "gh-aw" {
		githubLog.Print("Using public GitHub host for github/gh-aw repository")
		return string(constants.PublicGitHubHost)
	}

	// For all other repositories, use the configured GitHub host
	return GetGitHubHost()
}

// GetGitHubToken attempts to get GitHub token from environment or gh CLI
func GetGitHubToken() (string, error) {
	githubLog.Print("Getting GitHub token")

	// First try environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		githubLog.Print("Found GITHUB_TOKEN environment variable")
		return token, nil
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		githubLog.Print("Found GH_TOKEN environment variable")
		return token, nil
	}

	// Fall back to gh auth token command
	githubLog.Print("Attempting to get token from gh auth token command")
	cmd := exec.Command("gh", "auth", "token")
	// Note: gh auth token should respect GH_HOST environment variable for enterprise
	output, err := cmd.Output()
	if err != nil {
		githubLog.Printf("Failed to get token from gh auth token: %v", err)
		return "", fmt.Errorf("GITHUB_TOKEN environment variable not set and 'gh auth token' failed: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		githubLog.Print("gh auth token returned empty token")
		return "", fmt.Errorf("GITHUB_TOKEN environment variable not set and 'gh auth token' returned empty token")
	}

	githubLog.Print("Successfully retrieved token from gh auth token")
	return token, nil
}
