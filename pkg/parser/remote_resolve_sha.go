//go:build !js && !wasm

package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/github/gh-aw/pkg/gitutil"
)

// resolveRefToSHAViaGit resolves a git ref to SHA using git ls-remote
// This is a fallback for when GitHub API authentication fails
func resolveRefToSHAViaGit(ctx context.Context, owner, repo, ref, host string) (string, error) {
	remoteLog.Printf("Attempting git ls-remote fallback for ref resolution: %s/%s@%s", owner, repo, ref)

	var githubHost string
	if host != "" {
		githubHost = "https://" + host
	} else {
		githubHost = GetGitHubHostForRepo(owner, repo)
	}
	repoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, owner, repo)

	// Try to resolve the ref using git ls-remote
	// Format: git ls-remote <repo> <ref>
	cmd := exec.CommandContext(ctx, "git", "ls-remote", repoURL, ref)
	output, err := cmd.Output()
	if err != nil {
		// If exact ref doesn't work, try with refs/heads/ and refs/tags/ prefixes
		for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
			cmd = exec.CommandContext(ctx, "git", "ls-remote", repoURL, prefix+ref)
			output, err = cmd.Output()
			if err == nil && len(output) > 0 {
				break
			}
		}

		if err != nil {
			return "", fmt.Errorf("failed to resolve ref via git ls-remote: %w", err)
		}
	}

	// Parse the output: "<sha> <ref>"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no matching ref found for %s", ref)
	}

	// Extract SHA from the first line
	parts := strings.Fields(lines[0])
	if len(parts) < 1 {
		return "", errors.New("invalid git ls-remote output format")
	}

	sha := parts[0]

	// Validate it's a valid SHA
	if len(sha) != 40 || !gitutil.IsHexString(sha) {
		return "", fmt.Errorf("invalid SHA format from git ls-remote: %s", sha)
	}

	remoteLog.Printf("Successfully resolved ref via git ls-remote: %s/%s@%s -> %s", owner, repo, ref, sha)
	return sha, nil
}

// resolveRefToSHA resolves a git ref (branch, tag, or SHA) to its commit SHA
func resolveRefToSHA(ctx context.Context, owner, repo, ref, host string) (string, error) {
	// If ref is already a full SHA (40 hex characters), return it as-is
	if len(ref) == 40 && gitutil.IsHexString(ref) {
		return ref, nil
	}

	// Use gh CLI to get the commit SHA for the ref
	// This works for branches, tags, and short SHAs
	// Using go-gh to properly handle enterprise GitHub instances via GH_HOST
	apiPath := buildCommitLookupAPIPath(owner, repo, ref)
	var args []string
	if host != "" {
		args = []string{"api", "--hostname", host, apiPath, "--jq", ".sha"}
	} else {
		args = []string{"api", apiPath, "--jq", ".sha"}
	}

	stdout, stderr, err := gh.Exec(args...)

	if err != nil {
		outputStr := stderr.String()
		if gitutil.IsAuthError(outputStr) {
			remoteLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s/%s@%s", owner, repo, ref)
			// Try fallback using git ls-remote for public repositories
			sha, gitErr := resolveRefToSHAViaGit(ctx, owner, repo, ref, host)
			if gitErr != nil {
				if host == "" || host == "github.com" {
					remoteLog.Printf("Git fallback also failed, attempting unauthenticated API for %s/%s@%s", owner, repo, ref)
					return resolveRefToSHAViaPublicAPI(ctx, owner, repo, ref)
				}
				return "", fmt.Errorf("failed to resolve ref via GitHub API (auth error) and git ls-remote: API error: %w, Git error: %w", err, gitErr)
			}
			return sha, nil
		}

		return "", fmt.Errorf("failed to resolve ref %s to SHA for %s/%s: %s: %w", ref, owner, repo, strings.TrimSpace(outputStr), err)
	}

	sha := strings.TrimSpace(stdout.String())
	if sha == "" {
		return "", fmt.Errorf("empty SHA returned for ref %s in %s/%s", ref, owner, repo)
	}

	// Validate it's a valid SHA (40 hex characters)
	if len(sha) != 40 || !gitutil.IsHexString(sha) {
		return "", fmt.Errorf("invalid SHA format returned: %s", sha)
	}

	return sha, nil
}

// buildCommitLookupAPIPath returns the GitHub commits API path for a ref,
// URL-escaping the ref segment so branch names containing slashes are valid.
func buildCommitLookupAPIPath(owner, repo, ref string) string {
	return fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, url.PathEscape(ref))
}

// resolveRefToSHAViaPublicAPI resolves a git ref to its commit SHA using an
// unauthenticated call to the public GitHub API. Used as a last-resort fallback
// when both authenticated API and git ls-remote fail.
func resolveRefToSHAViaPublicAPI(ctx context.Context, owner, repo, ref string) (string, error) {
	remoteLog.Printf("Attempting unauthenticated public API ref resolution for %s/%s@%s", owner, repo, ref)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s",
		owner, repo, url.PathEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := publicAPIClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unauthenticated public API failed for %s/%s@%s: HTTP %d: %s", owner, repo, ref, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse commit response: %w", err)
	}
	if result.SHA == "" || len(result.SHA) != 40 || !gitutil.IsHexString(result.SHA) {
		return "", fmt.Errorf("invalid SHA returned from public API: %q", result.SHA)
	}
	return result.SHA, nil
}

// ResolveRefToSHAForHost resolves a git ref to its full commit SHA on a specific GitHub host.
// Use this when the target repository is on a different host than the one configured via GH_HOST.
// host is the hostname without scheme (e.g., "github.com", "myorg.ghe.com").
// An empty host uses the default configured host (GH_HOST or github.com).
func ResolveRefToSHAForHost(ctx context.Context, owner, repo, ref, host string) (string, error) {
	return resolveRefToSHA(ctx, owner, repo, ref, host)
}
