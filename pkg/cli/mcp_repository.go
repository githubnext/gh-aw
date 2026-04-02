package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

// getRepository retrieves the current repository name (owner/repo format).
// Results are cached for 1 hour to avoid repeated queries.
// Checks GITHUB_REPOSITORY environment variable first, then falls back to gh repo view.
func getRepository() (string, error) {
	// Check cache first
	if repo, ok := mcpCache.GetRepo(); ok {
		mcpLog.Printf("Using cached repository: %s", repo)
		return repo, nil
	}

	// Try GITHUB_REPOSITORY environment variable first
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo != "" {
		mcpLog.Printf("Got repository from GITHUB_REPOSITORY: %s", repo)
		mcpCache.SetRepo(repo)
		return repo, nil
	}

	// Fall back to gh repo view
	mcpLog.Print("Querying repository using gh repo view")
	cmd := workflow.ExecGH("repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		mcpLog.Printf("Failed to get repository: %v", err)
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	repo = strings.TrimSpace(string(output))
	if repo == "" {
		return "", errors.New("repository not found")
	}

	mcpLog.Printf("Got repository from gh repo view: %s", repo)
	mcpCache.SetRepo(repo)
	return repo, nil
}
