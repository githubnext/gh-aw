//go:build !js && !wasm

package parser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
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
		if value := os.Getenv(envVar); value != "" { //nolint:osgetenvlibrary
			githubLog.Printf("Resolved GitHub host from %s: %s", envVar, value)
			return stringutil.NormalizeGitHubHostURL(value)
		}
	}

	defaultHost := string(constants.PublicGitHubHost)
	githubLog.Printf("No GitHub host environment variable set, using default: %s", defaultHost)
	return defaultHost
}

// IsAnyGitHubHostEnvVarSet returns true when at least one of the environment
// variables consulted by GetGitHubHost is explicitly set to a non-empty value.
// This indicates the caller has made an explicit host choice and automatic
// fallback heuristics (such as git-remote detection) should not be consulted.
func IsAnyGitHubHostEnvVarSet() bool {
	for _, envVar := range []string{"GITHUB_SERVER_URL", "GITHUB_ENTERPRISE_HOST", "GITHUB_HOST", "GH_HOST"} {
		if os.Getenv(envVar) != "" { //nolint:osgetenvlibrary
			return true
		}
	}
	return false
}

// GetGitHubHostForRepo returns the GitHub host URL for a specific repository.
// Repositories under the github, githubnext, and microsoft organizations are
// fetched from public GitHub (https://github.com) in cross-host contexts.
// microsoft/* is included because canonical shared workflows (for example
// microsoft/apm/.github/workflows/shared/apm.md) are maintained on github.com.
// For all other repositories, it uses GetGitHubHost().
func GetGitHubHostForRepo(owner, repo string) string {
	switch owner {
	case "github", "githubnext", "microsoft":
		githubLog.Printf("Using public GitHub host for %s/%s repository in %s organization", owner, repo, owner)
		return string(constants.PublicGitHubHost)
	}

	// For all other repositories, use the configured GitHub host
	return GetGitHubHost()
}

// GetGitHubToken attempts to get GitHub token from environment or gh CLI
func GetGitHubToken() (string, error) {
	githubLog.Print("Getting GitHub token")

	// First try environment variable
	if token := githubTokenFromEnv(githubLog); token != "" {
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
		return "", errors.New("GITHUB_TOKEN environment variable not set and 'gh auth token' returned empty token")
	}

	githubLog.Print("Successfully retrieved token from gh auth token")
	return token, nil
}
