package parser

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var githubLog = logger.New("parser:github")

// GetGitHubToken attempts to get GitHub token from environment or gh CLI
func GetGitHubToken() (string, error) {
	githubLog.Print("Attempting to get GitHub token")
	// First try environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		githubLog.Print("Found GitHub token in GITHUB_TOKEN environment variable")
		return token, nil
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		githubLog.Print("Found GitHub token in GH_TOKEN environment variable")
		return token, nil
	}

	// Fall back to gh auth token command
	githubLog.Print("Environment variables not set, trying gh auth token command")
	cmd := exec.Command("gh", "auth", "token")
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

	githubLog.Print("Successfully retrieved GitHub token from gh auth token")
	return token, nil
}
