package workflow

import (
	"fmt"
	"os/exec"
	"strings"
)

// ActionResolver handles resolving action SHAs using GitHub CLI
type ActionResolver struct {
	cache *ActionCache
}

// NewActionResolver creates a new action resolver
func NewActionResolver(cache *ActionCache) *ActionResolver {
	return &ActionResolver{
		cache: cache,
	}
}

// ResolveSHA resolves the SHA for a given action@version using GitHub CLI
// Returns the SHA and an error if resolution fails
func (r *ActionResolver) ResolveSHA(repo, version string) (string, error) {
	// Check cache first
	if sha, found := r.cache.Get(repo, version); found {
		return sha, nil
	}

	// Resolve using GitHub CLI
	sha, err := r.resolveFromGitHub(repo, version)
	if err != nil {
		return "", err
	}

	// Cache the result
	r.cache.Set(repo, version, sha)

	return sha, nil
}

// resolveFromGitHub uses gh CLI to resolve the SHA for an action@version
func (r *ActionResolver) resolveFromGitHub(repo, version string) (string, error) {
	// Extract base repository (for actions like "github/codeql-action/upload-sarif")
	baseRepo := extractBaseRepo(repo)

	// Use gh api to get the git ref for the tag
	// API endpoint: GET /repos/{owner}/{repo}/git/ref/tags/{tag}
	apiPath := fmt.Sprintf("/repos/%s/git/ref/tags/%s", baseRepo, version)

	cmd := exec.Command("gh", "api", apiPath, "--jq", ".object.sha")
	output, err := cmd.Output()
	if err != nil {
		// Try without "refs/tags/" prefix in case version is already a ref
		return "", fmt.Errorf("failed to resolve %s@%s: %w", repo, version, err)
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", fmt.Errorf("empty SHA returned for %s@%s", repo, version)
	}

	// Validate SHA format (should be 40 hex characters)
	if len(sha) != 40 {
		return "", fmt.Errorf("invalid SHA format for %s@%s: %s", repo, version, sha)
	}

	return sha, nil
}

// extractBaseRepo extracts the base repository from a repo path
// For "actions/checkout" -> "actions/checkout"
// For "github/codeql-action/upload-sarif" -> "github/codeql-action"
func extractBaseRepo(repo string) string {
	parts := strings.Split(repo, "/")
	if len(parts) >= 2 {
		// Take first two parts (owner/repo)
		return parts[0] + "/" + parts[1]
	}
	return repo
}
