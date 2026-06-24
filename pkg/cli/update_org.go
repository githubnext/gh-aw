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
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

// orgUpdateCoreBuffer preserves a 500-request safety margin on the core API budget
// before org-mode applies a delay to avoid exhausting the hourly quota mid-run.
const orgUpdateCoreBuffer = 500

// orgUpdateSearchBuffer preserves at least one search request because GitHub's search
// quota is much smaller and org discovery only needs a narrow cushion between pages.
const orgUpdateSearchBuffer = 1

var searchOrgWorkflowReposFn = searchOrgWorkflowRepos
var previewOrgRepoUpdatesFn = previewOrgRepoUpdates
var runUpdateForTargetRepoFn = runUpdateForTargetRepo
var waitForOrgRateLimitFn = waitForOrgRateLimit

type orgSearchResponse struct {
	Items []struct {
		Path       string `json:"path"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"items"`
}

type orgRateLimitResponse struct {
	Resources struct {
		Core   rateLimitResource `json:"core"`
		Search rateLimitResource `json:"search"`
	} `json:"resources"`
}

type orgWorkflowPreview struct {
	Name       string
	Path       string
	CurrentRef string
	LatestRef  string
	Redirected bool
	EditedAt   time.Time
}

type orgRepoPreview struct {
	Repo       string
	Workflows  []orgWorkflowPreview
	OldestEdit time.Time
}

func runUpdateForOrg(ctx context.Context, org string, repoGlobs []string, opts UpdateWorkflowsOptions, createPR bool, verbose bool) error {
	if strings.TrimSpace(org) == "" {
		return errors.New("--org cannot be empty")
	}
	if err := validateRepoGlobs(repoGlobs); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Discovering repositories in "+org+" with source-managed workflows..."))
	repoPaths, err := searchOrgWorkflowReposFn(ctx, org, verbose)
	if err != nil {
		return err
	}

	if len(repoPaths) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No repositories found with source-managed workflows"))
		return nil
	}

	repos := filterOrgRepos(repoPaths, repoGlobs)
	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No repositories matched the requested --repos filters"))
		return nil
	}

	previewByRepo := make([]orgRepoPreview, 0, len(repos))
	for _, repo := range repos {
		if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s: %v", repo, err)))
		}

		preview, err := previewOrgRepoUpdatesFn(ctx, repo, opts, verbose)
		if err != nil {
			return fmt.Errorf("failed to preview updates for %s: %w", repo, err)
		}
		if len(preview.Workflows) == 0 {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Skipping "+repo+": already up to date"))
			}
			continue
		}
		previewByRepo = append(previewByRepo, preview)
	}

	if len(previewByRepo) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("All matching repositories are already up to date"))
		return nil
	}

	slices.SortStableFunc(previewByRepo, func(a, b orgRepoPreview) int {
		if a.OldestEdit.IsZero() && b.OldestEdit.IsZero() {
			return strings.Compare(a.Repo, b.Repo)
		}
		if a.OldestEdit.IsZero() {
			return 1
		}
		if b.OldestEdit.IsZero() {
			return -1
		}
		if a.OldestEdit.Equal(b.OldestEdit) {
			return strings.Compare(a.Repo, b.Repo)
		}
		if a.OldestEdit.Before(b.OldestEdit) {
			return -1
		}
		return 1
	})

	if !createPR {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dry-run preview of update pull requests:"))
		for _, repo := range previewByRepo {
			fmt.Fprintf(os.Stderr, "- %s\n", repo.Repo)
			for _, wf := range repo.Workflows {
				fmt.Fprintf(os.Stderr, "  - %s: %s -> %s\n", wf.Name, shortRef(wf.CurrentRef), shortRef(wf.LatestRef))
			}
		}
		return nil
	}

	for _, repo := range previewByRepo {
		if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s: %v", repo.Repo, err)))
		}
		if err := runUpdateForTargetRepoFn(ctx, repo.Repo, opts, true, verbose); err != nil {
			return err
		}
	}

	return nil
}

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

func searchOrgWorkflowRepos(ctx context.Context, org string, verbose bool) ([]string, error) {
	query := fmt.Sprintf(`org:%s path:.github/workflows extension:md "source:"`, org)
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

func previewOrgRepoUpdates(ctx context.Context, repo string, opts UpdateWorkflowsOptions, verbose bool) (orgRepoPreview, error) {
	workflowsPath := constants.GetWorkflowDir()
	endpoint := fmt.Sprintf("/repos/%s/contents/%s", repo, workflowsPath)
	output, err := workflow.RunGHContext(ctx, "Listing workflows...", "api", endpoint)
	if err != nil {
		return orgRepoPreview{}, fmt.Errorf("failed to list workflows: %w", err)
	}

	var entries []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(output, &entries); err != nil {
		return orgRepoPreview{}, fmt.Errorf("failed to parse workflow listing: %w", err)
	}

	defaultBranch, err := getRepoDefaultBranch(ctx, repo)
	if err != nil {
		return orgRepoPreview{}, fmt.Errorf("failed to resolve default branch: %w", err)
	}

	preview := orgRepoPreview{Repo: repo, Workflows: make([]orgWorkflowPreview, 0, len(entries))}
	for _, entry := range entries {
		if entry.Type != "file" || !strings.HasSuffix(entry.Name, ".md") || strings.HasSuffix(entry.Name, ".lock.yml") {
			continue
		}

		if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s/%s: %v", repo, entry.Path, err)))
		}

		content, err := downloadWorkflowContentFn(ctx, repo, entry.Path, defaultBranch, verbose)
		if err != nil {
			return orgRepoPreview{}, fmt.Errorf("failed to download %s: %w", entry.Path, err)
		}
		result, err := parser.ExtractFrontmatterFromContent(string(content))
		if err != nil {
			return orgRepoPreview{}, fmt.Errorf("failed to parse %s: %w", entry.Path, err)
		}
		sourceRaw, ok := result.Frontmatter["source"]
		if !ok {
			continue
		}
		source, ok := sourceRaw.(string)
		if !ok || strings.TrimSpace(source) == "" {
			continue
		}

		sourceSpec, err := parseSourceSpec(source)
		if err != nil {
			return orgRepoPreview{}, fmt.Errorf("failed to parse source for %s: %w", entry.Path, err)
		}
		name := normalizeWorkflowID(entry.Name)
		resolved, err := resolveRedirectedUpdateLocation(ctx, name, sourceSpec, opts.AllowMajor, verbose, opts.NoRedirect, opts.CoolDown)
		if err != nil {
			return orgRepoPreview{}, err
		}
		if resolved.currentRef == resolved.latestRef && len(resolved.redirectHistory) == 0 && !opts.Force {
			continue
		}

		editedAt, err := getLatestWorkflowEditTime(ctx, repo, entry.Path)
		if err == nil {
			if preview.OldestEdit.IsZero() || editedAt.Before(preview.OldestEdit) {
				preview.OldestEdit = editedAt
			}
		}
		preview.Workflows = append(preview.Workflows, orgWorkflowPreview{
			Name:       name,
			Path:       entry.Path,
			CurrentRef: resolved.currentRef,
			LatestRef:  resolved.latestRef,
			Redirected: len(resolved.redirectHistory) > 0,
			EditedAt:   editedAt,
		})
	}

	return preview, nil
}

func getLatestWorkflowEditTime(ctx context.Context, repo, workflowPath string) (time.Time, error) {
	endpoint := fmt.Sprintf("/repos/%s/commits?path=%s&per_page=1", repo, url.QueryEscape(workflowPath))
	output, err := workflow.RunGHContext(ctx, "Fetching workflow history...", "api", endpoint)
	if err != nil {
		return time.Time{}, err
	}

	var commits []struct {
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(output, &commits); err != nil {
		return time.Time{}, err
	}
	if len(commits) == 0 || strings.TrimSpace(commits[0].Commit.Committer.Date) == "" {
		return time.Time{}, fmt.Errorf("no commit date available for %s", workflowPath)
	}
	return time.Parse(time.RFC3339, commits[0].Commit.Committer.Date)
}

func waitForOrgRateLimit(ctx context.Context, resource string, verbose bool) error {
	output, err := workflow.RunGHContext(ctx, "Checking rate limit...", "api", "rate_limit")
	if err != nil {
		return err
	}

	var response orgRateLimitResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return err
	}

	limit := response.Resources.Core
	buffer := orgUpdateCoreBuffer
	if resource == "search" {
		limit = response.Resources.Search
		buffer = orgUpdateSearchBuffer
	}
	if limit.Limit == 0 {
		return nil
	}

	if limit.Remaining <= buffer {
		resetAt := time.Unix(limit.Reset, 0)
		waitFor := time.Until(resetAt) + rateLimitResetBuffer
		if waitFor > 0 {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("Applying a %s delay to avoid reaching the GitHub %s rate limit (%d/%d remaining)", waitFor.Round(time.Second), resource, limit.Remaining, limit.Limit),
			))
			timer := time.NewTimer(waitFor)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
			return nil
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(
			fmt.Sprintf("GitHub %s rate limit OK: %d/%d remaining", resource, limit.Remaining, limit.Limit),
		))
	}
	return nil
}
