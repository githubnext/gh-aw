package parser

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetGitHubToken attempts to get GitHub token from environment or gh CLI
func GetGitHubToken() (string, error) {
	log.Print("Getting GitHub token")

	// First try environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		log.Print("Found GITHUB_TOKEN environment variable")
		return token, nil
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		log.Print("Found GH_TOKEN environment variable")
		return token, nil
	}

	// Fall back to gh auth token command
	log.Print("Attempting to get token from gh auth token command")
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to get token from gh auth token: %v", err)
		return "", fmt.Errorf("GITHUB_TOKEN environment variable not set and 'gh auth token' failed: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		log.Print("gh auth token returned empty token")
		return "", fmt.Errorf("GITHUB_TOKEN environment variable not set and 'gh auth token' returned empty token")
	}

	log.Print("Successfully retrieved token from gh auth token")
	return token, nil
}
