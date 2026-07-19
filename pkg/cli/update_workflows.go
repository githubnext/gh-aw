package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/semverutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var defaultBranchCache sync.Map
var branchCommitCache sync.Map

// updatePublicAPIClient is a shared HTTP client used for unauthenticated GitHub
// API fallback calls in the update workflow path. It carries a timeout to
// prevent indefinite hangs on slow or unresponsive hosts.
var updatePublicAPIClient = &http.Client{Timeout: constants.DefaultHTTPClientTimeout}

// repoBranchKey is a composite cache key for branch commit SHA lookups.
type repoBranchKey struct {
	repo   string
	branch string
}

// clearUpdateResolutionCaches clears per-run ref-resolution caches so update
// operations always start from fresh repository state.
func clearUpdateResolutionCaches() {
	defaultBranchCache.Range(func(key, value any) bool {
		defaultBranchCache.Delete(key)
		return true
	})
	branchCommitCache.Range(func(key, value any) bool {
		branchCommitCache.Delete(key)
		return true
	})
	clearVersionLabelCache()
}

// UpdateWorkflowsOptions configures workflow update behavior.
type UpdateWorkflowsOptions struct {
	WorkflowNames          []string
	AllowMajor             bool
	Force                  bool
	Yes                    bool
	Verbose                bool
	EngineOverride         string
	WorkflowsDir           string
	NoStopAfter            bool
	StopAfter              string
	NoMerge                bool
	DisableReleaseBump     bool
	DisableSecurityScanner bool
	NoCompile              bool
	NoRedirect             bool
	CoolDown               time.Duration
	Approve                bool
}

// UpdateWorkflows updates workflows from their source repositories
func UpdateWorkflows(ctx context.Context, opts UpdateWorkflowsOptions) error {
	clearUpdateResolutionCaches()
	updateLog.Printf("Scanning for workflows with source field: dir=%s, filter=%v, noMerge=%v, noCompile=%v, noRedirect=%v, disableSecurityScanner=%v, coolDown=%v", opts.WorkflowsDir, opts.WorkflowNames, opts.NoMerge, opts.NoCompile, opts.NoRedirect, opts.DisableSecurityScanner, opts.CoolDown)

	// Find all workflows with source field
	workflowsDir := updateWorkflowsDir(opts.WorkflowsDir)
	workflows, err := findWorkflowsWithSource(workflowsDir, opts.WorkflowNames, opts.Verbose)
	if err != nil {
		return err
	}
	if ok, err := updateWorkflowsValidateFound(workflows, opts.WorkflowNames); err != nil || !ok {
		return err
	}

	// Track update results
	var successfulUpdates []string
	var failedUpdates []updateFailure

	manifestGroups, directWorkflows := updateWorkflowsPartition(workflows, &failedUpdates)
	successfulUpdates, failedUpdates = updateWorkflowsDirect(ctx, directWorkflows, opts, successfulUpdates, failedUpdates)
	successfulUpdates, failedUpdates = updateWorkflowsManifestGroups(ctx, manifestGroups, opts, successfulUpdates, failedUpdates)

	// Show summary
	showUpdateSummary(successfulUpdates, failedUpdates)
	return updateWorkflowsResult(successfulUpdates, failedUpdates)
}

func updateWorkflowsDir(workflowsDir string) string {
	if workflowsDir == "" {
		return getWorkflowsDir()
	}
	return workflowsDir
}

func updateWorkflowsValidateFound(workflows []*workflowWithSource, filterNames []string) (bool, error) {
	updateLog.Printf("Found %d workflows with source field", len(workflows))
	if len(workflows) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow(s) to update", len(workflows))))
		return true, nil
	}
	if len(filterNames) > 0 {
		return false, errors.New("no workflows found matching the specified names with source field")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("no workflows found with source field"))
	return false, nil
}

func updateWorkflowsPartition(workflows []*workflowWithSource, failedUpdates *[]updateFailure) (map[string][]*workflowWithSource, []*workflowWithSource) {
	manifestGroups := make(map[string][]*workflowWithSource)
	var directWorkflows []*workflowWithSource
	for _, wf := range workflows {
		if _, ok, err := parseManifestSourceSpec(wf.SourceSpec); err != nil {
			*failedUpdates = append(*failedUpdates, updateFailure{Name: wf.Name, Error: err.Error()})
			continue
		} else if ok {
			source := strings.TrimSpace(wf.SourceSpec)
			manifestGroups[source] = append(manifestGroups[source], wf)
			continue
		}
		directWorkflows = append(directWorkflows, wf)
	}
	return manifestGroups, directWorkflows
}

func updateWorkflowsDirect(ctx context.Context, directWorkflows []*workflowWithSource, opts UpdateWorkflowsOptions, successfulUpdates []string, failedUpdates []updateFailure) ([]string, []updateFailure) {
	for _, wf := range directWorkflows {
		updateLog.Printf("Updating workflow: %s (source: %s)", wf.Name, wf.SourceSpec)
		if err := updateWorkflow(ctx, wf, opts); err != nil {
			updateLog.Printf("Failed to update workflow %s: %v", wf.Name, err)
			failedUpdates = append(failedUpdates, updateFailure{Name: wf.Name, Error: err.Error()})
			continue
		}
		updateLog.Printf("Successfully updated workflow: %s", wf.Name)
		successfulUpdates = append(successfulUpdates, wf.Name)
	}
	return successfulUpdates, failedUpdates
}

func updateWorkflowsManifestGroups(ctx context.Context, manifestGroups map[string][]*workflowWithSource, opts UpdateWorkflowsOptions, successfulUpdates []string, failedUpdates []updateFailure) ([]string, []updateFailure) {
	for source, grouped := range manifestGroups {
		groupSuccesses, groupFailures := updateManifestWorkflowGroup(ctx, source, grouped, opts)
		successfulUpdates = append(successfulUpdates, groupSuccesses...)
		failedUpdates = append(failedUpdates, groupFailures...)
	}
	return successfulUpdates, failedUpdates
}

func updateWorkflowsResult(successfulUpdates []string, failedUpdates []updateFailure) error {
	if len(successfulUpdates) == 0 {
		if len(failedUpdates) > 0 && allFailuresAreRateLimited(failedUpdates) {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("All workflow updates skipped due to GitHub API rate limiting"))
			return nil
		}
		return errors.New("no workflows were successfully updated")
	}

	return nil
}

// allFailuresAreRateLimited returns true if every failed workflow update was caused
// by a GitHub API rate limit error. Used to distinguish transient rate-limiting
// (non-fatal) from genuine update failures (fatal).
func allFailuresAreRateLimited(failures []updateFailure) bool {
	for _, f := range failures {
		if !gitutil.IsRateLimitError(f.Error) {
			return false
		}
	}
	return true
}

// findWorkflowsWithSource finds all workflows that have a source field
func findWorkflowsWithSource(workflowsDir string, filterNames []string, verbose bool) ([]*workflowWithSource, error) {
	updateLog.Printf("Finding workflows with source field in %s", workflowsDir)
	var workflows []*workflowWithSource

	// Read all .md files in workflows directory
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}
	updateLog.Printf("Found %d entries in workflows directory", len(entries))

	for _, entry := range entries {
		workflow, ok := findWorkflowsWithSourceEntry(workflowsDir, entry, filterNames, verbose)
		if ok {
			workflows = append(workflows, workflow)
		}
	}

	return workflows, nil
}

func findWorkflowsWithSourceEntry(workflowsDir string, entry os.DirEntry, filterNames []string, verbose bool) (*workflowWithSource, bool) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || strings.HasSuffix(entry.Name(), ".lock.yml") {
		return nil, false
	}

	workflowPath := filepath.Join(workflowsDir, entry.Name())
	workflowName := normalizeWorkflowID(entry.Name())
	if !findWorkflowsWithSourceMatchesFilter(workflowName, filterNames) {
		return nil, false
	}

	source, ok := findWorkflowsWithSourceRead(workflowPath, workflowName, verbose)
	if !ok {
		return nil, false
	}
	return &workflowWithSource{
		Name:       workflowName,
		Path:       workflowPath,
		SourceSpec: strings.TrimSpace(source),
	}, true
}

func findWorkflowsWithSourceMatchesFilter(workflowName string, filterNames []string) bool {
	if len(filterNames) == 0 {
		return true
	}
	for _, filterName := range filterNames {
		if workflowName == normalizeWorkflowID(filterName) {
			return true
		}
	}
	return false
}

func findWorkflowsWithSourceRead(workflowPath, workflowName string, verbose bool) (string, bool) {
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", workflowPath, err)))
		}
		return "", false
	}
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse frontmatter in %s: %v", workflowPath, err)))
		}
		return "", false
	}
	sourceRaw, ok := result.Frontmatter["source"]
	if !ok {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Skipping %s: no source field", workflowName)))
		}
		return "", false
	}
	source, ok := sourceRaw.(string)
	if !ok || source == "" {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: invalid source field", workflowName)))
		}
		return "", false
	}
	return source, true
}

// resolveLatestRef resolves the latest ref for a workflow source
func resolveLatestRef(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
	updateLog.Printf("Resolving latest ref: repo=%s, currentRef=%s, allowMajor=%v", repo, currentRef, allowMajor)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Resolving latest ref for %s (current: %s)", repo, currentRef)))
	}

	// Check if current ref is a tag (looks like a semantic version)
	if isSemanticVersionTag(currentRef) {
		updateLog.Print("Current ref is semantic version tag, resolving latest release")
		return resolveLatestRelease(ctx, repo, currentRef, allowMajor, verbose, coolDown)
	}

	// Check if current ref is a commit SHA (40-character hex string)
	if IsCommitSHA(currentRef) {
		updateLog.Printf("Current ref is a commit SHA: %s, fetching latest from default branch", currentRef)
		// The source field only contains a pinned SHA with no branch information.
		// Fetch the latest commit from the default branch to check for updates.
		return resolveLatestCommitFromDefaultBranch(ctx, repo, currentRef, verbose)
	}

	// Otherwise, treat as branch and get latest commit
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Treating %s as branch, getting latest commit", currentRef)))
	}

	// Get the latest commit SHA for the branch
	latestSHA, err := getLatestBranchCommitSHACached(ctx, repo, currentRef)
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit for branch %s: %w", currentRef, err)
	}

	updateLog.Printf("Latest commit for branch %s: %s", currentRef, latestSHA)

	// Return the SHA for comparison so we can detect upstream changes.
	// The caller (updateWorkflow) preserves the branch name in the source
	// field to avoid SHA-pinning — see isBranchRef() usage there.
	return latestSHA, nil
}

// resolveLatestCommitFromDefaultBranch fetches the latest commit SHA from
// the default branch of a repo. This is used when the source field is pinned
// to a commit SHA with no branch information — in that case we can only
// logically track the default branch.
func resolveLatestCommitFromDefaultBranch(ctx context.Context, repo, currentSHA string, verbose bool) (string, error) {
	// Get the default branch name
	defaultBranch, err := getRepoDefaultBranchCached(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get default branch for %s: %w", repo, err)
	}

	updateLog.Printf("Source is pinned to commit SHA, tracking default branch %q of %s", defaultBranch, repo)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Source has no branch ref, tracking default branch %q", defaultBranch)))
	}

	// Get the latest commit SHA from the default branch
	latestSHA, err := getLatestBranchCommitSHACached(ctx, repo, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit for default branch %s: %w", defaultBranch, err)
	}

	updateLog.Printf("Latest commit on default branch %s: %s (current: %s)", defaultBranch, latestSHA, currentSHA)

	return latestSHA, nil
}

// getRepoDefaultBranchCached wraps getRepoDefaultBranch with a cache to avoid
// repeating identical GitHub API calls during batched update runs.
func getRepoDefaultBranchCached(ctx context.Context, repo string) (string, error) {
	if cached, ok := defaultBranchCache.Load(repo); ok {
		if branch, isString := cached.(string); isString {
			return branch, nil
		}
	}

	branch, err := getRepoDefaultBranch(ctx, repo)
	if err != nil {
		return "", err
	}
	defaultBranchCache.Store(repo, branch)
	return branch, nil
}

// getLatestBranchCommitSHACached wraps getLatestBranchCommitSHA with a cache
// keyed by repo+branch to reduce repeated branch-head API lookups.
func getLatestBranchCommitSHACached(ctx context.Context, repo, branch string) (string, error) {
	key := repoBranchKey{repo: repo, branch: branch}
	if cached, ok := branchCommitCache.Load(key); ok {
		if sha, isString := cached.(string); isString {
			return sha, nil
		}
	}

	sha, err := getLatestBranchCommitSHA(ctx, repo, branch)
	if err != nil {
		return "", err
	}
	branchCommitCache.Store(key, sha)
	return sha, nil
}

// fetchPublicGitHubAPI makes an unauthenticated GET request to the GitHub public
// REST API. This is used as a fallback when the current token (e.g. an enterprise
// SAML-enforced token) cannot access cross-organization public repositories.
// Unauthenticated requests are subject to a lower rate limit (60 req/hour) but
// are sufficient for the handful of calls needed during update resolution.
func fetchPublicGitHubAPI(ctx context.Context, endpoint string) ([]byte, error) {
	apiURL := "https://api.github.com" + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := updatePublicAPIClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// getRepoDefaultBranch fetches the default branch name for a repository.
func getRepoDefaultBranch(ctx context.Context, repo string) (string, error) {
	output, err := workflow.RunGHContext(ctx, "Fetching repo info...", "api", "/repos/"+repo, "--jq", ".default_branch")
	if err != nil && gitutil.IsAuthError(err.Error()) {
		updateLog.Printf("GitHub API auth failed for %s, retrying without token", repo)
		body, fallbackErr := fetchPublicGitHubAPI(ctx, "/repos/"+repo)
		if fallbackErr != nil {
			return "", fmt.Errorf("failed (with token: %w; without token: %w)", err, fallbackErr)
		}
		var result struct {
			DefaultBranch string `json:"default_branch"`
		}
		if fallbackErr = json.Unmarshal(body, &result); fallbackErr != nil {
			return "", fmt.Errorf("failed to parse repo response: %w", fallbackErr)
		}
		if result.DefaultBranch == "" {
			return "", fmt.Errorf("empty default branch returned for %s", repo)
		}
		return result.DefaultBranch, nil
	}
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("empty default branch returned for %s", repo)
	}

	return branch, nil
}

// getLatestBranchCommitSHA fetches the latest commit SHA for a given branch.
func getLatestBranchCommitSHA(ctx context.Context, repo, branch string) (string, error) {
	// URL-encode the branch name since it may contain slashes (e.g. "feature/foo")
	endpoint := fmt.Sprintf("/repos/%s/branches/%s", repo, url.PathEscape(branch))
	output, err := workflow.RunGHContext(ctx, "Fetching branch info...", "api", endpoint, "--jq", ".commit.sha")
	if err != nil && gitutil.IsAuthError(err.Error()) {
		updateLog.Printf("GitHub API auth failed for branch %s of %s, retrying without token", branch, repo)
		body, fallbackErr := fetchPublicGitHubAPI(ctx, endpoint)
		if fallbackErr != nil {
			return "", fmt.Errorf("failed (with token: %w; without token: %w)", err, fallbackErr)
		}
		var result struct {
			Commit struct {
				SHA string `json:"sha"`
			} `json:"commit"`
		}
		if fallbackErr = json.Unmarshal(body, &result); fallbackErr != nil {
			return "", fmt.Errorf("failed to parse branch response: %w", fallbackErr)
		}
		if result.Commit.SHA == "" {
			return "", fmt.Errorf("empty commit SHA returned for branch %s", branch)
		}
		return result.Commit.SHA, nil
	}
	if err != nil {
		return "", err
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", fmt.Errorf("empty commit SHA returned for branch %s", branch)
	}

	return sha, nil
}

type workflowUpdateDeps struct {
	runReleasesAPI func(ctx context.Context, repo string) ([]byte, error)
}

func defaultWorkflowUpdateDeps() workflowUpdateDeps {
	return workflowUpdateDeps{
		runReleasesAPI: func(ctx context.Context, repo string) ([]byte, error) {
			endpoint := fmt.Sprintf("/repos/%s/releases", repo)
			output, err := workflow.RunGHContext(ctx, "Fetching releases...", "api", endpoint, "--jq", ".[].tag_name")
			if err != nil && gitutil.IsAuthError(err.Error()) {
				updateLog.Printf("GitHub API auth failed for releases of %s, retrying without token", repo)
				body, fallbackErr := fetchPublicGitHubAPI(ctx, endpoint)
				if fallbackErr != nil {
					return nil, fmt.Errorf("failed (with token: %w; without token: %w)", err, fallbackErr)
				}
				var releases []struct {
					TagName string `json:"tag_name"`
				}
				if fallbackErr = json.Unmarshal(body, &releases); fallbackErr != nil {
					return nil, fmt.Errorf("failed to parse releases response: %w", fallbackErr)
				}
				var tags []string
				for _, r := range releases {
					if r.TagName != "" {
						tags = append(tags, r.TagName)
					}
				}
				return []byte(strings.Join(tags, "\n")), nil
			}
			return output, err
		},
	}
}

// resolveLatestRelease resolves the latest compatible release for a workflow source
func resolveLatestRelease(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
	return resolveLatestReleaseWithDeps(ctx, defaultWorkflowUpdateDeps(), repo, currentRef, allowMajor, verbose, coolDown)
}

func resolveLatestReleaseWithDeps(ctx context.Context, deps workflowUpdateDeps, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
	updateLog.Printf("Resolving latest release for repo %s (current: %s, allowMajor=%v)", repo, currentRef, allowMajor)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Checking for latest release (current: %s, allow major: %v)", currentRef, allowMajor)))
	}

	releases, err := resolveLatestReleaseWithDepsFetch(ctx, deps, repo)
	if err != nil {
		return "", err
	}

	// Parse current version
	currentVer := parseVersion(currentRef)
	if currentVer == nil {
		return resolveLatestReleaseWithDepsLatestStable(repo, releases, verbose)
	}

	latestCompatible, err := resolveLatestReleaseWithDepsCompatible(releases, currentVer, allowMajor)
	if err != nil {
		return "", err
	}

	if resolveLatestReleaseWithDepsInCoolDown(ctx, repo, currentRef, latestCompatible, coolDown) {
		return currentRef, nil
	}

	if verbose && latestCompatible != currentRef {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found newer release: "+latestCompatible))
	}

	return latestCompatible, nil
}

func resolveLatestReleaseWithDepsFetch(ctx context.Context, deps workflowUpdateDeps, repo string) ([]string, error) {
	output, err := deps.runReleasesAPI(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) == 0 || releases[0] == "" {
		return nil, errors.New("no releases found")
	}
	return releases, nil
}

func resolveLatestReleaseWithDepsLatestStable(repo string, releases []string, verbose bool) (string, error) {
	var latestStable string
	var latestStableVersion *semverutil.SemanticVersion
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer == nil || releaseVer.Pre != "" {
			continue
		}
		if latestStableVersion == nil || releaseVer.IsNewer(latestStableVersion) {
			latestStable = release
			latestStableVersion = releaseVer
		}
	}
	if latestStable == "" {
		return "", fmt.Errorf("no stable releases found for %s", repo)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Current version is not valid, using latest stable release: "+latestStable))
	}
	return latestStable, nil
}

func resolveLatestReleaseWithDepsCompatible(releases []string, currentVer *semverutil.SemanticVersion, allowMajor bool) (string, error) {
	var latestCompatible string
	var latestCompatibleVersion *semverutil.SemanticVersion
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer == nil || releaseVer.Pre != "" {
			continue
		}
		if !allowMajor && releaseVer.Major != currentVer.Major {
			continue
		}
		if latestCompatibleVersion == nil || releaseVer.IsNewer(latestCompatibleVersion) {
			latestCompatible = release
			latestCompatibleVersion = releaseVer
		}
	}
	if latestCompatible == "" {
		return "", errors.New("no compatible release found")
	}
	return latestCompatible, nil
}

func resolveLatestReleaseWithDepsInCoolDown(ctx context.Context, repo, currentRef, latestCompatible string, coolDown time.Duration) bool {
	if latestCompatible == currentRef || isExemptFromCoolDown(repo) {
		return false
	}
	if result := checkReleaseCoolDown(ctx, repo, latestCompatible, coolDown); result.InCoolDown {
		cooldownLog.Printf("Workflow source %s: %s", repo, result.Message)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping update for %s: %s", repo, result.Message)))
		return true
	}
	return false
}

// updateWorkflow updates a single workflow from its source
func updateWorkflow(ctx context.Context, wf *workflowWithSource, opts UpdateWorkflowsOptions) error {
	updateLog.Printf("Updating workflow: name=%s, source=%s, force=%v, noMerge=%v", wf.Name, wf.SourceSpec, opts.Force, opts.NoMerge)

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating workflow: "+wf.Name))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Source: "+wf.SourceSpec))
	}

	state, err := updateWorkflowResolve(ctx, wf, opts)
	if err != nil {
		return err
	}
	if done, err := updateWorkflowAlreadyUpToDate(ctx, state); done || err != nil {
		return err
	}
	finalContent, hasConflicts, err := updateWorkflowContent(ctx, state)
	if err != nil {
		return err
	}
	finalContent = updateWorkflowStopAfter(finalContent, opts)
	if err := updateWorkflowSecurityScan(wf, finalContent, opts); err != nil {
		return err
	}
	return updateWorkflowWriteCompile(ctx, state, finalContent, hasConflicts)
}

type updateWorkflowState struct {
	wf              *workflowWithSource
	opts            UpdateWorkflowsOptions
	sourceSpec      *SourceSpec
	currentRef      string
	latestRef       string
	sourceFieldRef  string
	newContent      []byte
	redirectHistory []string
}

func updateWorkflowResolve(ctx context.Context, wf *workflowWithSource, opts UpdateWorkflowsOptions) (*updateWorkflowState, error) {
	initialSourceSpec, err := parseSourceSpec(wf.SourceSpec)
	if err != nil {
		updateLog.Printf("Failed to parse source spec: %v", err)
		return nil, fmt.Errorf("failed to parse source spec: %w", err)
	}
	resolvedLocation, err := resolveRedirectedUpdateLocation(ctx, wf.Name, initialSourceSpec, opts.AllowMajor, opts.Verbose, opts.NoRedirect, opts.CoolDown)
	if err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Current ref: "+resolvedLocation.currentRef))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Latest ref: "+resolvedLocation.latestRef))
	}
	return &updateWorkflowState{wf: wf, opts: opts, sourceSpec: resolvedLocation.sourceSpec, currentRef: resolvedLocation.currentRef, latestRef: resolvedLocation.latestRef, sourceFieldRef: resolvedLocation.sourceFieldRef, newContent: resolvedLocation.content, redirectHistory: resolvedLocation.redirectHistory}, nil
}

func updateWorkflowAlreadyUpToDate(ctx context.Context, state *updateWorkflowState) (bool, error) {
	if state.opts.Force || state.currentRef != state.latestRef || len(state.redirectHistory) > 0 {
		if len(state.redirectHistory) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow %s source location changed; updating source to %s/%s@%s", state.wf.Name, state.sourceSpec.Repo, state.sourceSpec.Path, state.sourceFieldRef)))
		}
		return false, nil
	}
	updateLog.Printf("Workflow already at latest ref: %s, checking for local modifications", state.currentRef)
	sourceContent, err := downloadWorkflowContentFn(ctx, state.sourceSpec.Repo, state.sourceSpec.Path, state.currentRef, state.opts.Verbose)
	if err != nil {
		if state.opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download source for comparison: %v", err)))
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", state.wf.Name, shortRef(state.currentRef))))
		return true, nil
	}
	currentContent, err := os.ReadFile(state.wf.Path)
	if err != nil {
		return true, fmt.Errorf("failed to read current workflow: %w", err)
	}
	if hasLocalModifications(string(sourceContent), string(currentContent), state.wf.SourceSpec, filepath.Dir(state.wf.Path), state.opts.Verbose) {
		updateLog.Printf("Local modifications detected in workflow: %s", state.wf.Name)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", state.wf.Name, shortRef(state.currentRef))))
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠️  Local copy of %s has been modified from source", state.wf.Name)))
		return true, nil
	}
	updateLog.Printf("Workflow %s is up to date with no local modifications", state.wf.Name)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", state.wf.Name, shortRef(state.currentRef))))
	return true, nil
}

func updateWorkflowContent(ctx context.Context, state *updateWorkflowState) (string, bool, error) {
	merge := updateWorkflowShouldMerge(ctx, state)
	if merge {
		return updateWorkflowMerge(ctx, state)
	}
	return updateWorkflowOverride(state)
}

func updateWorkflowShouldMerge(ctx context.Context, state *updateWorkflowState) bool {
	if state.opts.NoMerge || len(state.redirectHistory) > 0 {
		return false
	}
	baseContent, dlErr := downloadWorkflowContentFn(ctx, state.sourceSpec.Repo, state.sourceSpec.Path, state.currentRef, state.opts.Verbose)
	if dlErr != nil {
		return true
	}
	localContent, readErr := os.ReadFile(state.wf.Path)
	if readErr == nil && hasLocalModifications(string(baseContent), string(localContent), state.wf.SourceSpec, filepath.Dir(state.wf.Path), state.opts.Verbose) {
		updateLog.Printf("Local modifications detected in %s, merging to preserve changes", state.wf.Name)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Local modifications detected in %s, merging to preserve your changes", state.wf.Name)))
		return true
	}
	return false
}

func updateWorkflowMerge(ctx context.Context, state *updateWorkflowState) (string, bool, error) {
	if state.opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Using merge mode to preserve local changes"))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Downloading base version from %s/%s@%s", state.sourceSpec.Repo, state.sourceSpec.Path, state.currentRef)))
	}
	baseContent, err := downloadWorkflowContentFn(ctx, state.sourceSpec.Repo, state.sourceSpec.Path, state.currentRef, state.opts.Verbose)
	if err != nil {
		return "", false, fmt.Errorf("failed to download base workflow: %w", err)
	}
	currentContent, err := os.ReadFile(state.wf.Path)
	if err != nil {
		return "", false, fmt.Errorf("failed to read current workflow: %w", err)
	}
	updateLog.Printf("Performing 3-way merge for workflow: %s", state.wf.Name)
	mergedContent, conflicts, err := MergeWorkflowContent(string(baseContent), string(currentContent), string(state.newContent), state.wf.SourceSpec, sourceSpecWithRef(state.sourceSpec, state.sourceFieldRef), state.wf.Path, state.opts.Verbose)
	if err != nil {
		updateLog.Printf("Merge failed for workflow %s: %v", state.wf.Name, err)
		return "", false, fmt.Errorf("failed to merge workflow content: %w", err)
	}
	if conflicts {
		updateLog.Printf("Merge conflicts detected in workflow: %s", state.wf.Name)
	}
	return mergedContent, conflicts, nil
}

func updateWorkflowOverride(state *updateWorkflowState) (string, bool, error) {
	if state.opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Using override mode - local changes will be replaced"))
	}
	finalContent := string(state.newContent)
	newWithUpdatedSource, err := UpdateFieldInFrontmatter(finalContent, "source", sourceSpecWithRef(state.sourceSpec, state.sourceFieldRef))
	if err != nil {
		if state.opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update source in new content: %v", err)))
		}
	} else {
		finalContent = newWithUpdatedSource
	}
	workflow := &WorkflowSpec{RepoSpec: RepoSpec{RepoSlug: state.sourceSpec.Repo, Version: state.latestRef}, WorkflowPath: state.sourceSpec.Path}
	processedContent, err := processIncludesInContent(finalContent, workflow, state.latestRef, filepath.Dir(state.wf.Path), state.opts.Verbose)
	if err != nil {
		if state.opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
		}
	} else {
		finalContent = processedContent
	}
	return finalContent, false, nil
}

func updateWorkflowStopAfter(finalContent string, opts UpdateWorkflowsOptions) string {
	if opts.NoStopAfter {
		cleanedContent, err := RemoveFieldFromOnTrigger(finalContent, "stop-after")
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove stop-after field: %v", err)))
			}
			return finalContent
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed stop-after field from workflow"))
		}
		return cleanedContent
	}
	if opts.StopAfter != "" {
		updatedContent, err := SetFieldInOnTrigger(finalContent, "stop-after", opts.StopAfter)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set stop-after field: %v", err)))
			}
			return finalContent
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Set stop-after field to: "+opts.StopAfter))
		}
		return updatedContent
	}
	return finalContent
}

func updateWorkflowSecurityScan(wf *workflowWithSource, finalContent string, opts UpdateWorkflowsOptions) error {
	if opts.DisableSecurityScanner {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Security scanning disabled"))
		}
		return nil
	}
	if findings := workflow.ScanMarkdownSecurity(finalContent); len(findings) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Security scan failed for workflow"))
		fmt.Fprintln(os.Stderr, workflow.FormatSecurityFindings(findings, wf.Path))
		return fmt.Errorf("workflow '%s' failed security scan: %d issue(s) detected", wf.Name, len(findings))
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Security scan passed"))
	}
	return nil
}

func updateWorkflowWriteCompile(ctx context.Context, state *updateWorkflowState, finalContent string, hasConflicts bool) error {
	if err := os.WriteFile(state.wf.Path, []byte(finalContent), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}
	if hasConflicts {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Updated %s from %s to %s with CONFLICTS - please review and resolve manually", state.wf.Name, shortRef(state.currentRef), shortRef(state.latestRef))))
		return nil
	}
	updateLog.Printf("Successfully updated workflow %s from %s to %s", state.wf.Name, state.currentRef, state.latestRef)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", state.wf.Name, shortRef(state.currentRef), shortRef(state.latestRef))))
	if !state.opts.NoCompile {
		updateLog.Printf("Compiling updated workflow: %s", state.wf.Name)
		if err := compileWorkflowWithRefresh(ctx, state.wf.Path, state.opts.Verbose, false, state.opts.EngineOverride, true, state.opts.Approve); err != nil {
			updateLog.Printf("Compilation failed for workflow %s: %v", state.wf.Name, err)
			return fmt.Errorf("failed to compile updated workflow: %w", err)
		}
	} else {
		updateLog.Printf("Skipping compilation of workflow %s (--no-compile specified)", state.wf.Name)
	}
	return nil
}

// isBranchRef returns true when the ref is a branch name — i.e. it is
// neither a semantic-version tag nor a full commit SHA.
func isBranchRef(ref string) bool {
	return !isSemanticVersionTag(ref) && !IsCommitSHA(ref)
}

// shortRef abbreviates a ref for display. Commit SHAs are truncated to 7 characters;
// other refs (branch names, tags) are returned as-is.
func shortRef(ref string) string {
	if IsCommitSHA(ref) {
		return ref[:7]
	}
	return ref
}
