package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/workflow"
)

// orgSearchResponse holds the paginated code-search results returned by the
// GitHub search/code API when discovering repositories in an organization.
type orgSearchResponse struct {
	Items []struct {
		Path       string `json:"path"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"items"`
}

var searchOrgWorkflowReposFn = searchOrgWorkflowRepos

// searchOrgWorkflowRepos searches an organization's repositories for workflow
// markdown files that include a "source:" field, indicating they are
// source-managed agentic workflows eligible for bulk updates.
//
// It paginates through all code-search results, deduplicates by repository full
// name, and returns a deterministically sorted slice of "owner/repo" strings.
func searchOrgWorkflowRepos(ctx context.Context, org string, workflowNames []string, verbose bool) ([]string, error) {
	query := buildOrgWorkflowSearchQuery(org, workflowNames)
	return searchOrgReposByQuery(ctx, query, verbose)
}

// buildOrgWorkflowSearchQuery constructs the org-mode code-search query for
// source-managed workflows. When workflowNames is empty, or every candidate
// normalizes away, it falls back to the base query and relies on the later
// per-repo workflow scan to enforce any requested filters.
func buildOrgWorkflowSearchQuery(org string, workflowNames []string) string {
	base := fmt.Sprintf(`org:%s path:.github/workflows extension:md "source:"`, org)
	if len(workflowNames) == 0 {
		return base
	}

	filenameFilters := make([]string, 0, len(workflowNames))
	seen := make(map[string]struct{}, len(workflowNames))
	for _, workflowName := range workflowNames {
		normalized := normalizeWorkflowID(workflowName)
		if normalized == "" || normalized == "." {
			continue
		}
		filename := normalized + ".md"
		if _, ok := seen[filename]; ok {
			continue
		}
		seen[filename] = struct{}{}
		filenameFilters = append(filenameFilters, "filename:"+filename)
	}
	if len(filenameFilters) == 0 {
		// CLI validation already rejects empty workflow names, so this fallback is
		// primarily a safety net for non-CLI callers and tests.
		return base
	}

	slices.Sort(filenameFilters)
	return base + " (" + strings.Join(filenameFilters, " OR ") + ")"
}

// searchOrgReposByQuery paginates through GitHub code-search results for the given
// query, deduplicates by repository full name, and returns a deterministically
// sorted slice of "owner/repo" strings.
func searchOrgReposByQuery(ctx context.Context, query string, verbose bool) ([]string, error) {
	perPage := 100
	page := 1
	seen := make(map[string]struct{})
	var repos []string

	for {
		if err := waitForOrgRateLimitFn(ctx, "search", verbose); err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after search rate limit check failure: %v", err)))
		}
		endpoint := fmt.Sprintf("/search/code?q=%s&per_page=%d&page=%d", url.QueryEscape(query), perPage, page)
		output, err := workflow.RunGHContext(ctx, "Searching repositories...", "api", endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to search organization repositories: %w", err)
		}

		var response orgSearchResponse
		if err := json.Unmarshal(output, &response); err != nil {
			return nil, fmt.Errorf("failed to parse organization search results: %w", err)
		}
		if len(response.Items) == 0 {
			break
		}

		for _, item := range response.Items {
			repo := strings.TrimSpace(item.Repository.FullName)
			if repo == "" {
				continue
			}
			if _, ok := seen[repo]; ok {
				continue
			}
			seen[repo] = struct{}{}
			repos = append(repos, repo)
		}

		if len(response.Items) < perPage {
			break
		}
		page++
	}

	slices.Sort(repos)
	return repos, nil
}

// validateRepoGlobs reports an error for any empty or syntactically invalid
// glob pattern in the --repos flag slice.
func validateRepoGlobs(globs []string) error {
	for _, glob := range globs {
		glob = strings.TrimSpace(glob)
		if glob == "" {
			return errors.New("--repos patterns cannot be empty")
		}
		if _, err := path.Match(glob, "example"); err != nil {
			return fmt.Errorf("invalid --repos pattern %q: %w", glob, err)
		}
	}
	return nil
}

// filterOrgRepos returns the subset of repos that match at least one of the
// provided glob patterns. Each pattern is tested against both the full
// "owner/repo" name and the bare repository name. When globs is empty every
// repository is returned unchanged.
func filterOrgRepos(repos []string, globs []string) []string {
	if len(globs) == 0 {
		return repos
	}
	filtered := make([]string, 0, len(repos))
	for _, repo := range repos {
		name := repo
		if _, tail, ok := strings.Cut(repo, "/"); ok {
			name = tail
		}
		for _, glob := range globs {
			if ok, _ := path.Match(glob, repo); ok {
				filtered = append(filtered, repo)
				break
			}
			if ok, _ := path.Match(glob, name); ok {
				filtered = append(filtered, repo)
				break
			}
		}
	}
	return filtered
}
