package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var resolverLog = logger.New("workflow:action_resolver")

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

// resolveFromGitHub uses gh CLI to resolve the SHA for an action@version
func (r *ActionResolver) resolveFromGitHub(repo, version string) (string, error) {
	// Extract base repository (for actions like "github/codeql-action/upload-sarif")
	baseRepo := gitutil.ExtractBaseRepo(repo)
	resolverLog.Printf("Extracted base repository: %s from %s", baseRepo, repo)

	// Use gh api to get the git ref for the tag
	// API endpoint: GET /repos/{owner}/{repo}/git/ref/tags/{tag}
	apiPath := fmt.Sprintf("/repos/%s/git/ref/tags/%s", baseRepo, version)
	resolverLog.Printf("Querying GitHub API: %s", apiPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch both SHA and object type to detect annotated tags.
	// Annotated tags have type "tag" and their SHA points to the tag object,
	// not the underlying commit. We must peel to get the commit SHA.
	cmd := ExecGHContext(ctx, "api", apiPath, "--jq", "[.object.sha, .object.type] | @tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s@%s: %w", repo, version, err)
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
		peelCtx, peelCancel := context.WithTimeout(context.Background(), 30*time.Second)
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
