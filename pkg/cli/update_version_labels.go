package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/workflow"
)

// repoTagEntry holds a tag name and its resolved commit SHA.
type repoTagEntry struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

var (
	versionLabelMu    sync.Mutex
	versionLabelCache = make(map[string]map[string]string) // repo -> sha -> tag name
)

// resolveVersionLabel returns a human-readable label for a git ref in the given
// source repo. If the ref is a semver tag or branch name it is returned as-is.
// For commit SHAs it tries to find a matching tag in the source repo (first page
// of tags). If no tag is found it falls back to shortRef.
//
// Results are cached per source repo so that multiple workflows sharing a source
// only trigger one API call.
func resolveVersionLabel(ctx context.Context, sourceRepo, ref string) string {
	if !IsCommitSHA(ref) {
		// Already a tag or branch – just return as-is.
		return ref
	}

	versionLabelMu.Lock()
	if tagMap, ok := versionLabelCache[sourceRepo]; ok {
		versionLabelMu.Unlock()
		if tag, ok := tagMap[ref]; ok {
			return tag
		}
		return shortRef(ref)
	}
	versionLabelMu.Unlock()

	tagMap := loadRepoTagMap(ctx, sourceRepo)

	versionLabelMu.Lock()
	versionLabelCache[sourceRepo] = tagMap
	versionLabelMu.Unlock()

	if tag, ok := tagMap[ref]; ok {
		return tag
	}
	return shortRef(ref)
}

// loadRepoTagMap fetches the first page of tags for sourceRepo and returns a
// map from full commit SHA to tag name. On error an empty map is returned so
// callers gracefully fall back to the short SHA.
func loadRepoTagMap(ctx context.Context, sourceRepo string) map[string]string {
	tagMap := make(map[string]string)
	owner, repoName, ok := splitOwnerRepo(sourceRepo)
	if !ok {
		return tagMap
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/tags?per_page=100", owner, repoName)
	output, err := workflow.RunGHContext(ctx, "Fetching version tags...", "api", endpoint)
	if err != nil {
		return tagMap
	}

	var tags []repoTagEntry
	if err := json.Unmarshal(output, &tags); err != nil {
		return tagMap
	}

	for _, t := range tags {
		sha := strings.TrimSpace(t.Commit.SHA)
		name := strings.TrimSpace(t.Name)
		if sha != "" && name != "" {
			tagMap[sha] = name
		}
	}
	return tagMap
}

// splitOwnerRepo splits "owner/repo" into (owner, repo, true). Returns
// ("", "", false) if the string does not contain exactly one slash.
func splitOwnerRepo(ownerRepo string) (string, string, bool) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
