package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var resolverLog = logger.New("workflow:action_resolver")

// defaultAPITimeout is the maximum time allowed for a single GitHub API call.
const defaultAPITimeout = 30 * time.Second

// ActionSHAResolver is the minimal interface for resolving an action tag to its commit SHA.
type ActionSHAResolver interface {
	ResolveSHA(repo, version string) (string, error)
}

// ActionResolver handles resolving action SHAs using GitHub CLI
type ActionResolver struct {
	cache             *ActionCache
	failedResolutions map[string]bool // tracks failed resolution attempts in current run (key: "repo@version")
}

// NewActionResolver creates a new action resolver
func NewActionResolver(cache *ActionCache) *ActionResolver {
	return &ActionResolver{
		cache:             cache,
		failedResolutions: make(map[string]bool),
	}
}

// ResolveSHA resolves the SHA for a given action@version using GitHub CLI
// Returns the SHA and an error if resolution fails
func (r *ActionResolver) ResolveSHA(repo, version string) (string, error) {
	resolverLog.Printf("Resolving SHA for action: %s@%s", repo, version)

	// Create a cache key for tracking failed resolutions
	cacheKey := formatActionCacheKey(repo, version)

	// Check if we've already failed to resolve this action in this run
	if r.failedResolutions[cacheKey] {
		resolverLog.Printf("Skipping resolution for %s@%s: already failed in this run", repo, version)
		return "", fmt.Errorf("previously failed to resolve %s@%s in this compilation run", repo, version)
	}

	// Check cache first
	if sha, found := r.cache.Get(repo, version); found {
		resolverLog.Printf("Cache hit for %s@%s: %s", repo, version, sha)
		return sha, nil
	}

	resolverLog.Printf("Cache miss for %s@%s, querying GitHub API", repo, version)
	resolverLog.Printf("This may take a moment as we query GitHub API at /repos/%s/git/ref/tags/%s", gitutil.ExtractBaseRepo(repo), version)

	// Resolve using GitHub CLI
	sha, err := r.resolveFromGitHub(repo, version)
	if err != nil {
		resolverLog.Printf("Failed to resolve %s@%s: %v", repo, version, err)
		// Mark this resolution as failed for this compilation run
		r.failedResolutions[cacheKey] = true
		resolverLog.Printf("Marked %s as failed, will not retry in this run", cacheKey)
		return "", err
	}

	resolverLog.Printf("Successfully resolved %s@%s to SHA: %s", repo, version, sha)

	// Cache the result
	resolverLog.Printf("Caching result: %s@%s → %s", repo, version, sha)
	r.cache.Set(repo, version, sha)

	return sha, nil
}

// ParseTagRefTSV parses the tab-separated output from the GitHub API
// `[.object.sha, .object.type] | @tsv` jq expression.
// It returns the object SHA and type, or an error if the output is malformed.
// This is a standalone helper so that the parsing logic can be unit-tested
// independently of network calls.
func ParseTagRefTSV(line string) (sha, objType string, err error) {
	line = strings.TrimSpace(line)
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format: %q", line)
	}
	sha = parts[0]
	objType = parts[1]
	if len(sha) != 40 || !gitutil.IsHexString(sha) {
		return "", "", fmt.Errorf("invalid SHA format: expected 40 hex characters, got %d (%s)", len(sha), sha)
	}
	return sha, objType, nil
}

// resolveFromGitHub uses gh CLI to resolve the SHA for an action@version.
// If the gh CLI call fails (e.g. no auth), it falls back to the public
// GitHub REST API for public repositories.
func (r *ActionResolver) resolveFromGitHub(repo, version string) (string, error) {
	// Extract base repository (for actions like "github/codeql-action/upload-sarif")
	baseRepo := gitutil.ExtractBaseRepo(repo)
	resolverLog.Printf("Extracted base repository: %s from %s", baseRepo, repo)

	// Use gh api to get the git ref for the tag
	// API endpoint: GET /repos/{owner}/{repo}/git/ref/tags/{tag}
	apiPath := fmt.Sprintf("/repos/%s/git/ref/tags/%s", baseRepo, version)
	resolverLog.Printf("Querying GitHub API: %s", apiPath)

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	// Fetch both SHA and object type to detect annotated tags.
	// Annotated tags have type "tag" and their SHA points to the tag object,
	// not the underlying commit. We must peel to get the commit SHA.
	cmd := ExecGHContext(ctx, "api", apiPath, "--jq", "[.object.sha, .object.type] | @tsv")
	output, err := cmd.Output()
	if err != nil {
		resolverLog.Printf("gh CLI failed for %s@%s (%v), trying public API", repo, version, err)
		return resolveTagSHAPublic(baseRepo, version)
	}

	sha, objType, err := ParseTagRefTSV(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to parse API response for %s@%s: %w", repo, version, err)
	}

	// Annotated tags (and chained tag objects) point to a tag object rather than
	// directly to a commit. Iteratively peel until we reach a non-tag object so
	// that emitted action pins use the stable underlying commit SHA rather than a
	// mutable tag object SHA (which changes when the tag is re-created).
	const maxTagPeelDepth = 10
	for depth := 0; objType == "tag"; depth++ {
		if depth >= maxTagPeelDepth {
			return "", fmt.Errorf("failed to resolve %s@%s: exceeded max tag peel depth %d", repo, version, maxTagPeelDepth)
		}
		resolverLog.Printf("Detected annotated tag for %s@%s (depth %d, tag object SHA: %s), peeling to underlying object", repo, version, depth, sha)
		tagPath := fmt.Sprintf("/repos/%s/git/tags/%s", baseRepo, sha)
		peelCtx, peelCancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		cmd2 := ExecGHContext(peelCtx, "api", tagPath, "--jq", "[.object.sha, .object.type] | @tsv")
		output2, peelErr := cmd2.Output()
		peelCancel()
		if peelErr != nil {
			return "", fmt.Errorf("failed to peel annotated tag %s@%s: %w", repo, version, peelErr)
		}
		sha, objType, err = ParseTagRefTSV(string(output2))
		if err != nil {
			return "", fmt.Errorf("failed to parse peeled tag API response for %s@%s: %w", repo, version, err)
		}
	}
	resolverLog.Printf("Resolved %s@%s to %s SHA: %s", repo, version, objType, sha)

	return sha, nil
}

// resolveTagSHAPublic resolves the commit SHA for a given tag using the
// unauthenticated public GitHub REST API. This is used as a fallback when
// the authenticated gh CLI is unavailable (e.g., no token configured).
// It handles annotated tags by following the object chain to the commit.
func resolveTagSHAPublic(baseRepo, tag string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/ref/tags/%s", baseRepo, tag)
	resolverLog.Printf("Querying public GitHub API for tag SHA: %s", url)

	const maxTagPeelDepth = 10
	for range maxTagPeelDepth {
		sha, objType, err := fetchGitObjectPublic(url)
		if err != nil {
			return "", fmt.Errorf("failed to resolve tag SHA for %s@%s via public API: %w", baseRepo, tag, err)
		}
		if objType != "tag" {
			resolverLog.Printf("Resolved %s@%s to %s SHA via public API: %s", baseRepo, tag, objType, sha)
			return sha, nil
		}
		// Annotated tag — peel to the underlying object.
		resolverLog.Printf("Peeling annotated tag for %s@%s (tag object SHA: %s)", baseRepo, tag, sha)
		url = fmt.Sprintf("https://api.github.com/repos/%s/git/tags/%s", baseRepo, sha)
	}
	return "", fmt.Errorf("failed to resolve %s@%s: exceeded max tag peel depth", baseRepo, tag)
}

// fetchGitObjectPublic fetches the SHA and object type from a public GitHub git API URL.
func fetchGitObjectPublic(apiURL string) (sha, objType string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// #nosec G107 -- apiURL is constructed from validated owner/repo and tag strings.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, apiURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	// The git/ref/tags/{tag} endpoint returns:
	//   { "object": { "sha": "...", "type": "commit|tag" } }
	// The git/tags/{sha} endpoint returns:
	//   { "object": { "sha": "...", "type": "commit|tag" } }
	var result struct {
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err = json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Object.SHA == "" || result.Object.Type == "" {
		return "", "", fmt.Errorf("missing object fields in response from %s", apiURL)
	}
	return result.Object.SHA, result.Object.Type, nil
}

// ResolveLatestTag returns the latest release tag and its commit SHA for a
// given repository. It queries the GitHub API for the latest release and then
// resolves the tag to a commit SHA using ResolveSHA. The result is cached via
// the action cache so subsequent calls within the same compilation run are free.
//
// Resolution order:
//  1. Authenticated `gh api` call (requires GH_TOKEN / GITHUB_TOKEN)
//  2. Unauthenticated public GitHub API (no token required, works for public repos)
//
// This is used as a fallback when pinning github_ref inputs that have no
// explicit ref and no entry in the embedded action_pins.json.
func (r *ActionResolver) ResolveLatestTag(repo string) (tag, sha string, err error) {
	baseRepo := gitutil.ExtractBaseRepo(repo)
	resolverLog.Printf("Querying latest release for %s", baseRepo)

	tag, err = r.queryLatestReleaseTag(baseRepo)
	if err != nil {
		resolverLog.Printf("Failed to query latest release for %s: %v", baseRepo, err)
		return "", "", fmt.Errorf("failed to get latest release for %s: %w", baseRepo, err)
	}

	resolverLog.Printf("Latest release tag for %s: %s", baseRepo, tag)
	sha, err = r.ResolveSHA(repo, tag)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve SHA for %s@%s: %w", repo, tag, err)
	}
	return tag, sha, nil
}

// queryLatestReleaseTag fetches the latest release tag for the given base repo
// (owner/repo format). It tries `gh api` first (authenticated) and falls back
// to the public GitHub REST API for public repositories.
func (r *ActionResolver) queryLatestReleaseTag(baseRepo string) (string, error) {
	apiPath := fmt.Sprintf("/repos/%s/releases/latest", baseRepo)

	// --- Attempt 1: authenticated gh CLI ---
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	cmd := ExecGHContext(ctx, "api", apiPath, "--jq", ".tag_name")
	output, cmdErr := cmd.Output()
	if cmdErr == nil {
		tag := strings.TrimSpace(string(output))
		if tag != "" {
			resolverLog.Printf("Got latest tag via gh CLI for %s: %s", baseRepo, tag)
			return tag, nil
		}
	} else {
		resolverLog.Printf("gh CLI failed for %s (%v), falling back to public API", baseRepo, cmdErr)
	}

	// --- Attempt 2: unauthenticated public GitHub REST API ---
	return queryLatestReleaseTagPublic(baseRepo)
}

// queryLatestReleaseTagPublic fetches the latest release tag for a public
// repository using the unauthenticated GitHub REST API.
func queryLatestReleaseTagPublic(baseRepo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", baseRepo)
	return queryLatestReleaseTagPublicFromURL(url)
}

// queryLatestReleaseTagPublicFromURL fetches the latest release tag from the
// given URL (public GitHub REST API releases endpoint format). Separated for
// testability so tests can inject a local httptest server URL.
func queryLatestReleaseTagPublicFromURL(url string) (string, error) {
	resolverLog.Printf("Querying public GitHub API for latest release: %s", url)

	httpCtx, httpCancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer httpCancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create public API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// #nosec G107 -- URL is constructed from the repo name passed by the caller;
	// baseRepo is validated as owner/repo format before reaching this function.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("public API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("public API returned HTTP %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read public API response: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err = json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("failed to parse public API response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("no release found via public API at %s", url)
	}

	resolverLog.Printf("Got latest tag via public API: %s", release.TagName)
	return release.TagName, nil
}
