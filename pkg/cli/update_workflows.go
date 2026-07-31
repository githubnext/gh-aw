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
var updateNoticeCache sync.Map

// updatePublicAPIClient is a shared HTTP client used for unauthenticated GitHub
// API fallback calls in the update workflow path. It carries a timeout to
// prevent indefinite hangs on slow or unresponsive hosts.
var updatePublicAPIClient = &http.Client{Timeout: constants.DefaultHTTPClientTimeout}

// repoBranchKey is a composite cache key for branch commit SHA lookups.
type repoBranchKey struct {
	repo   string
	branch string
}

type latestBranchCommitInfo struct {
	SHA         string
	CommittedAt time.Time
}

type cachedDefaultBranch struct {
	branch string
	err    error
}

type cachedBranchCommit struct {
	info latestBranchCommitInfo
	err  error
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
	updateNoticeCache.Range(func(key, value any) bool {
		updateNoticeCache.Delete(key)
		return true
	})
	clearVersionLabelCache()
}

func printUpdateInfoOnce(message string) {
	if _, loaded := updateNoticeCache.LoadOrStore(message, struct{}{}); loaded {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(message))
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

// classifyWorkflowsForUpdate separates workflows into manifest-grouped and direct update lists.
// Workflows whose source spec cannot be parsed are returned as immediate failures.
func classifyWorkflowsForUpdate(workflows []*workflowWithSource) (manifestGroups map[string][]*workflowWithSource, directWorkflows []*workflowWithSource, failedUpdates []updateFailure) {
	manifestGroups = make(map[string][]*workflowWithSource)
	for _, wf := range workflows {
		if _, ok, err := parseManifestSourceSpec(wf.SourceSpec); err != nil {
			failedUpdates = append(failedUpdates, updateFailure{Name: wf.Name, Error: err.Error()})
			continue
		} else if ok {
			manifestGroups[strings.TrimSpace(wf.SourceSpec)] = append(manifestGroups[strings.TrimSpace(wf.SourceSpec)], wf)
			continue
		}
		directWorkflows = append(directWorkflows, wf)
	}
	return
}

// runWorkflowUpdates updates direct and manifest-grouped workflows, returning success/failure lists.
func runWorkflowUpdates(ctx context.Context, directWorkflows []*workflowWithSource, manifestGroups map[string][]*workflowWithSource, opts UpdateWorkflowsOptions) ([]string, []updateFailure) {
	var successfulUpdates []string
	var failedUpdates []updateFailure
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
	for source, grouped := range manifestGroups {
		groupSuccesses, groupFailures := updateManifestWorkflowGroup(ctx, source, grouped, opts)
		successfulUpdates = append(successfulUpdates, groupSuccesses...)
		failedUpdates = append(failedUpdates, groupFailures...)
	}
	return successfulUpdates, failedUpdates
}

// UpdateWorkflows updates workflows from their source repositories
func UpdateWorkflows(ctx context.Context, opts UpdateWorkflowsOptions) error {
	clearUpdateResolutionCaches()
	updateLog.Printf("Scanning for workflows with source field: dir=%s, filter=%v, noMerge=%v, noCompile=%v, noRedirect=%v, disableSecurityScanner=%v, coolDown=%v", opts.WorkflowsDir, opts.WorkflowNames, opts.NoMerge, opts.NoCompile, opts.NoRedirect, opts.DisableSecurityScanner, opts.CoolDown)

	workflowsDir := opts.WorkflowsDir
	if workflowsDir == "" {
		workflowsDir = getWorkflowsDir()
	}

	workflows, err := findWorkflowsWithSource(workflowsDir, opts.WorkflowNames, opts.Verbose)
	if err != nil {
		return err
	}

	updateLog.Printf("Found %d workflows with source field", len(workflows))

	if len(workflows) == 0 {
		if len(opts.WorkflowNames) > 0 {
			return errors.New("no workflows found matching the specified names with source field")
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("no workflows found with source field"))
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow(s) to update", len(workflows))))

	manifestGroups, directWorkflows, failedUpdates := classifyWorkflowsForUpdate(workflows)
	successfulUpdates, moreFailures := runWorkflowUpdates(ctx, directWorkflows, manifestGroups, opts)
	failedUpdates = append(failedUpdates, moreFailures...)

	showUpdateSummary(successfulUpdates, failedUpdates, opts.NoCompile)

	if len(successfulUpdates) == 0 {
		if len(failedUpdates) > 0 && allFailuresAreRateLimited(failedUpdates) {
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

// readWorkflowSourceSpec reads a workflow file and returns its source field value.
// Returns ("", false) when the file should be skipped (no source, invalid source, or read error).
func readWorkflowSourceSpec(workflowPath, workflowName string, verbose bool) (string, bool) {
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
	return strings.TrimSpace(source), true
}

// findWorkflowsWithSource finds all workflows that have a source field
func findWorkflowsWithSource(workflowsDir string, filterNames []string, verbose bool) ([]*workflowWithSource, error) {
	updateLog.Printf("Finding workflows with source field in %s", workflowsDir)
	var workflows []*workflowWithSource

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}
	updateLog.Printf("Found %d entries in workflows directory", len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".lock.yml") {
			continue
		}

		workflowPath := filepath.Join(workflowsDir, entry.Name())
		workflowName := normalizeWorkflowID(entry.Name())

		if len(filterNames) > 0 {
			matched := false
			for _, filterName := range filterNames {
				filterName = normalizeWorkflowID(filterName)
				if workflowName == filterName {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		source, ok := readWorkflowSourceSpec(workflowPath, workflowName, verbose)
		if !ok {
			continue
		}
		workflows = append(workflows, &workflowWithSource{
			Name:       workflowName,
			Path:       workflowPath,
			SourceSpec: source,
		})
	}

	return workflows, nil
}

// resolveLatestRef resolves the latest ref for a workflow source
type latestRefResolution struct {
	Ref             string
	CoolDownBlocked bool
}

func resolveLatestRef(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
	result, err := resolveLatestRefWithDeps(ctx, defaultWorkflowUpdateDeps(), repo, currentRef, allowMajor, verbose, coolDown)
	if err != nil {
		return "", err
	}
	return result.Ref, nil
}

func resolveLatestRefWithResult(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (latestRefResolution, error) {
	return resolveLatestRefWithDeps(ctx, defaultWorkflowUpdateDeps(), repo, currentRef, allowMajor, verbose, coolDown)
}

func resolveLatestRefWithDeps(ctx context.Context, deps workflowUpdateDeps, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (latestRefResolution, error) {
	updateLog.Printf("Resolving latest ref: repo=%s, currentRef=%s, allowMajor=%v", repo, currentRef, allowMajor)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Resolving latest ref for %s (current: %s)", repo, currentRef)))
	}

	// Check if current ref is a tag (looks like a semantic version)
	if isSemanticVersionTag(currentRef) {
		updateLog.Print("Current ref is semantic version tag, resolving latest release")
		ref, err := resolveLatestReleaseWithDeps(ctx, deps, repo, currentRef, allowMajor, verbose, coolDown)
		return latestRefResolution{Ref: ref}, err
	}

	// Check if current ref is a commit SHA (40-character hex string)
	if IsCommitSHA(currentRef) {
		updateLog.Printf("Current ref is a commit SHA: %s, fetching latest from default branch", currentRef)
		// The source field only contains a pinned SHA with no branch information.
		// Fetch the latest commit from the default branch to check for updates.
		return resolveLatestCommitFromDefaultBranchWithDeps(ctx, deps, repo, currentRef, verbose, effectiveCommitCoolDown(coolDown))
	}

	// Otherwise, treat as branch and get latest commit
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Treating %s as branch, getting latest commit", currentRef)))
	}

	// Get the latest commit SHA for the branch
	latestCommit, err := deps.getLatestBranchCommit(ctx, repo, currentRef)
	if err != nil {
		return latestRefResolution{}, fmt.Errorf("failed to get latest commit for branch %s: %w", currentRef, err)
	}

	updateLog.Printf("Latest commit for branch %s: %s", currentRef, latestCommit.SHA)

	commitCoolDown := effectiveCommitCoolDown(coolDown)
	if !isExemptFromCoolDown(repo) && commitCoolDown > 0 {
		if result := checkCommitCoolDownWithDate(repo, currentRef, latestCommit.CommittedAt, commitCoolDown); result.InCoolDown {
			cooldownLog.Printf("Workflow source %s branch %s: %s", repo, currentRef, result.Message)
			printUpdateInfoOnce(fmt.Sprintf("Skipping commit candidate %s@%s: %s", repo, currentRef, result.Message))
			return latestRefResolution{Ref: currentRef, CoolDownBlocked: true}, nil
		}
	}

	// Return the SHA for comparison so we can detect upstream changes.
	// The caller (updateWorkflow) preserves the branch name in the source
	// field to avoid SHA-pinning — see isBranchRef() usage there.
	return latestRefResolution{Ref: latestCommit.SHA}, nil
}

// resolveLatestCommitFromDefaultBranchWithDeps fetches the latest commit SHA from
// the default branch of a repo. This is used when the source field is pinned
// to a commit SHA with no branch information — in that case we can only
// logically track the default branch.
func resolveLatestCommitFromDefaultBranchWithDeps(ctx context.Context, deps workflowUpdateDeps, repo, currentSHA string, verbose bool, coolDown time.Duration) (latestRefResolution, error) {
	// Get the default branch name
	defaultBranch, err := deps.getRepoDefaultBranch(ctx, repo)
	if err != nil {
		return latestRefResolution{}, fmt.Errorf("failed to get default branch for %s: %w", repo, err)
	}

	updateLog.Printf("Source is pinned to commit SHA, tracking default branch %q of %s", defaultBranch, repo)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Source has no branch ref, tracking default branch %q", defaultBranch)))
	}

	// Get the latest commit SHA from the default branch
	latestCommit, err := deps.getLatestBranchCommit(ctx, repo, defaultBranch)
	if err != nil {
		return latestRefResolution{}, fmt.Errorf("failed to get latest commit for default branch %s: %w", defaultBranch, err)
	}

	updateLog.Printf("Latest commit on default branch %s: %s (current: %s)", defaultBranch, latestCommit.SHA, currentSHA)
	if !isExemptFromCoolDown(repo) && coolDown > 0 {
		if result := checkCommitCoolDownWithDate(repo, defaultBranch, latestCommit.CommittedAt, coolDown); result.InCoolDown {
			cooldownLog.Printf("Workflow source %s default branch %s: %s", repo, defaultBranch, result.Message)
			printUpdateInfoOnce(fmt.Sprintf("Skipping commit candidate %s@%s: %s", repo, defaultBranch, result.Message))
			return latestRefResolution{Ref: currentSHA, CoolDownBlocked: true}, nil
		}
	}

	return latestRefResolution{Ref: latestCommit.SHA}, nil
}

// getRepoDefaultBranchCached wraps getRepoDefaultBranch with a cache to avoid
// repeating identical GitHub API calls during batched update runs.
func getRepoDefaultBranchCached(ctx context.Context, repo string) (string, error) {
	if cached, ok := defaultBranchCache.Load(repo); ok {
		if result, ok := cached.(cachedDefaultBranch); ok {
			return result.branch, result.err
		}
	}

	branch, err := getRepoDefaultBranch(ctx, repo)
	defaultBranchCache.Store(repo, cachedDefaultBranch{branch: branch, err: err})
	return branch, err
}

// getLatestBranchCommitInfoCached wraps getLatestBranchCommitInfo with a cache
// keyed by repo+branch to reduce repeated branch-head API lookups.
func getLatestBranchCommitInfoCached(ctx context.Context, repo, branch string) (latestBranchCommitInfo, error) {
	key := repoBranchKey{repo: repo, branch: branch}
	if cached, ok := branchCommitCache.Load(key); ok {
		if result, ok := cached.(cachedBranchCommit); ok {
			return result.info, result.err
		}
	}

	info, err := getLatestBranchCommitInfo(ctx, repo, branch)
	branchCommitCache.Store(key, cachedBranchCommit{info: info, err: err})
	return info, err
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

func fetchPublicReleaseTagsPaginated(ctx context.Context, repo string) ([]string, error) {
	var tags []string
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("/repos/%s/releases?per_page=100&page=%d", repo, page)
		body, err := fetchPublicGitHubAPI(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, fmt.Errorf("failed to parse releases response: %w", err)
		}
		if len(releases) == 0 {
			break
		}
		for _, r := range releases {
			if r.TagName != "" {
				tags = append(tags, r.TagName)
			}
		}
		if len(releases) < 100 {
			break
		}
	}
	return tags, nil
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

// getLatestBranchCommitInfo fetches the latest commit SHA and commit date for a given branch.
func getLatestBranchCommitInfo(ctx context.Context, repo, branch string) (latestBranchCommitInfo, error) {
	// URL-encode the branch name since it may contain slashes (e.g. "feature/foo")
	endpoint := fmt.Sprintf("/repos/%s/commits/%s", repo, url.PathEscape(branch))
	output, err := workflow.RunGHContext(ctx, "Fetching commit info...", "api", endpoint)
	if err != nil && gitutil.IsAuthError(err.Error()) {
		updateLog.Printf("GitHub API auth failed for branch %s of %s, retrying without token", branch, repo)
		body, fallbackErr := fetchPublicGitHubAPI(ctx, endpoint)
		if fallbackErr != nil {
			return latestBranchCommitInfo{}, fmt.Errorf("failed (with token: %w; without token: %w)", err, fallbackErr)
		}
		return parseLatestBranchCommitInfo(branch, body)
	}
	if err != nil {
		return latestBranchCommitInfo{}, err
	}
	return parseLatestBranchCommitInfo(branch, output)
}

func parseLatestBranchCommitInfo(branch string, data []byte) (latestBranchCommitInfo, error) {
	var result struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
			Author struct {
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return latestBranchCommitInfo{}, fmt.Errorf("failed to parse commit response: %w", err)
	}
	if result.SHA == "" {
		return latestBranchCommitInfo{}, fmt.Errorf("empty commit SHA returned for branch %s", branch)
	}
	dateStr := result.Commit.Committer.Date
	if dateStr == "" {
		return latestBranchCommitInfo{SHA: result.SHA}, nil
	}
	committedAt, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return latestBranchCommitInfo{}, fmt.Errorf("invalid commit date for branch %s: %w", branch, err)
	}
	return latestBranchCommitInfo{SHA: result.SHA, CommittedAt: committedAt}, nil
}

type workflowUpdateDeps struct {
	runReleasesAPI        func(ctx context.Context, repo string) ([]byte, error)
	checkCoolDown         func(ctx context.Context, repo, tag string, coolDown time.Duration) coolDownCheckResult
	getRepoDefaultBranch  func(ctx context.Context, repo string) (string, error)
	getLatestBranchCommit func(ctx context.Context, repo, branch string) (latestBranchCommitInfo, error)
}

func defaultWorkflowUpdateDeps() workflowUpdateDeps {
	return workflowUpdateDeps{
		checkCoolDown:         checkReleaseCoolDown,
		getRepoDefaultBranch:  getRepoDefaultBranchCached,
		getLatestBranchCommit: getLatestBranchCommitInfoCached,
		runReleasesAPI: func(ctx context.Context, repo string) ([]byte, error) {
			endpoint := fmt.Sprintf("/repos/%s/releases", repo)
			output, err := workflow.RunGHContext(ctx, "Fetching releases...", "api", "--paginate", endpoint, "--jq", ".[].tag_name")
			if err != nil && gitutil.IsAuthError(err.Error()) {
				updateLog.Printf("GitHub API auth failed for releases of %s, retrying without token", repo)
				tags, fallbackErr := fetchPublicReleaseTagsPaginated(ctx, repo)
				if fallbackErr != nil {
					return nil, fmt.Errorf("failed (with token: %w; without token: %w)", err, fallbackErr)
				}
				return []byte(strings.Join(tags, "\n")), nil
			}
			return output, err
		},
	}
}

func resolveLatestReleaseWithDeps(ctx context.Context, deps workflowUpdateDeps, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
	updateLog.Printf("Resolving latest release for repo %s (current: %s, allowMajor=%v)", repo, currentRef, allowMajor)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Checking for latest release (current: %s, allow major: %v)", currentRef, allowMajor)))
	}

	output, err := deps.runReleasesAPI(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) == 0 || releases[0] == "" {
		return "", errors.New("no releases found")
	}

	currentVer := parseVersion(currentRef)
	if currentVer == nil {
		return findLatestStableRelease(releases, repo, verbose)
	}

	compatibleReleases := sortedCompatibleReleaseCandidates(releases, currentVer, allowMajor)
	if len(compatibleReleases) == 0 {
		return "", errors.New("no compatible release found")
	}

	upgradeCandidates := newerReleaseCandidates(compatibleReleases, currentVer)
	if len(upgradeCandidates) == 0 {
		return currentRef, nil
	}

	if tag := selectReleaseWithCooldown(ctx, deps, repo, upgradeCandidates, coolDown, verbose); tag != "" {
		return tag, nil
	}
	return currentRef, nil
}

// findLatestStableRelease returns the newest stable release tag from releases,
// selected by semantic version rather than API ordering.
func findLatestStableRelease(releases []string, repo string, verbose bool) (string, error) {
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

// selectReleaseWithCooldown returns the newest upgrade candidate that has passed the cooldown
// window, or empty string if all candidates are still cooling down.
func selectReleaseWithCooldown(ctx context.Context, deps workflowUpdateDeps, repo string, candidates []releaseCandidate, coolDown time.Duration, verbose bool) string {
	if isExemptFromCoolDown(repo) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found newer release: "+candidates[0].tag))
		}
		return candidates[0].tag
	}
	loggedCandidateSkip := false
	cooldownSkippedCount := 0
	for _, c := range candidates {
		result := deps.checkCoolDown(ctx, repo, c.tag, coolDown)
		if !result.InCoolDown {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found newer release: "+c.tag))
			}
			return c.tag
		}
		cooldownSkippedCount++
		cooldownLog.Printf("Workflow source %s: %s", repo, result.Message)
		if !loggedCandidateSkip {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping release candidate %s@%s: %s", repo, c.tag, result.Message)))
			loggedCandidateSkip = true
		}
	}
	if cooldownSkippedCount > 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Evaluated %d release candidates for %s; all are still in cooldown", cooldownSkippedCount, repo)))
	}
	return ""
}

// handleWorkflowUpToDate handles the case when a workflow is already at the latest ref.
// It checks for local modifications and returns nil when no update is needed.
func handleWorkflowUpToDate(ctx context.Context, wf *workflowWithSource, sourceSpec *SourceSpec, currentRef string, opts UpdateWorkflowsOptions) error {
	updateLog.Printf("Workflow already at latest ref: %s, checking for local modifications", currentRef)
	sourceContent, err := downloadWorkflowContentFn(ctx, sourceSpec.Repo, sourceSpec.Path, currentRef, opts.Verbose)
	if err != nil {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download source for comparison: %v", err)))
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, shortRef(currentRef))))
		return nil
	}
	currentContent, err := os.ReadFile(wf.Path)
	if err != nil {
		return fmt.Errorf("failed to read current workflow: %w", err)
	}
	if hasLocalModifications(string(sourceContent), string(currentContent), wf.SourceSpec, filepath.Dir(wf.Path), opts.Verbose) {
		updateLog.Printf("Local modifications detected in workflow: %s", wf.Name)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, shortRef(currentRef))))
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Local copy of %s has been modified from source", wf.Name)))
		return nil
	}
	updateLog.Printf("Workflow %s is up to date with no local modifications", wf.Name)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, shortRef(currentRef))))
	return nil
}

// detectWorkflowMergeMode determines whether to merge (true) or override (false).
func detectWorkflowMergeMode(ctx context.Context, wf *workflowWithSource, sourceSpec *SourceSpec, currentRef string, hasRedirectHistory bool, opts UpdateWorkflowsOptions) bool {
	merge := !opts.NoMerge
	if hasRedirectHistory {
		merge = false
	}
	if !merge {
		return false
	}
	baseContent, dlErr := downloadWorkflowContentFn(ctx, sourceSpec.Repo, sourceSpec.Path, currentRef, opts.Verbose)
	if dlErr != nil {
		return merge
	}
	localContent, readErr := os.ReadFile(wf.Path)
	if readErr != nil {
		return merge
	}
	if hasLocalModifications(string(baseContent), string(localContent), wf.SourceSpec, filepath.Dir(wf.Path), opts.Verbose) {
		updateLog.Printf("Local modifications detected in %s, merging to preserve changes", wf.Name)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Local modifications detected in %s, merging to preserve your changes", wf.Name)))
		return true
	}
	return false
}

// buildMergedWorkflowContent performs a 3-way merge and returns the merged content and conflict flag.
func buildMergedWorkflowContent(ctx context.Context, wf *workflowWithSource, sourceSpec *SourceSpec, currentRef, sourceFieldRef string, newContent []byte, opts UpdateWorkflowsOptions) (string, bool, error) {
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Using merge mode to preserve local changes"))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Downloading base version from %s/%s@%s", sourceSpec.Repo, sourceSpec.Path, currentRef)))
	}
	baseContent, err := downloadWorkflowContentFn(ctx, sourceSpec.Repo, sourceSpec.Path, currentRef, opts.Verbose)
	if err != nil {
		return "", false, fmt.Errorf("failed to download base workflow: %w", err)
	}
	currentContent, err := os.ReadFile(wf.Path)
	if err != nil {
		return "", false, fmt.Errorf("failed to read current workflow: %w", err)
	}
	updateLog.Printf("Performing 3-way merge for workflow: %s", wf.Name)
	merged, conflicts, err := MergeWorkflowContent(string(baseContent), string(currentContent), string(newContent), wf.SourceSpec, sourceSpecWithRef(sourceSpec, sourceFieldRef), wf.Path, opts.Verbose)
	if err != nil {
		updateLog.Printf("Merge failed for workflow %s: %v", wf.Name, err)
		return "", false, fmt.Errorf("failed to merge workflow content: %w", err)
	}
	if conflicts {
		updateLog.Printf("Merge conflicts detected in workflow: %s", wf.Name)
	}
	return merged, conflicts, nil
}

// buildOverrideWorkflowContent replaces the workflow content from source, applying include directives.
func buildOverrideWorkflowContent(wf *workflowWithSource, sourceSpec *SourceSpec, latestRef, sourceFieldRef string, newContent []byte, opts UpdateWorkflowsOptions) (string, error) {
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Using override mode - local changes will be replaced"))
	}
	finalContent := string(newContent)
	newWithUpdatedSource, err := UpdateFieldInFrontmatter(finalContent, "source", sourceSpecWithRef(sourceSpec, sourceFieldRef))
	if err != nil {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update source in new content: %v", err)))
		}
	} else {
		finalContent = newWithUpdatedSource
	}
	wfSpec := &WorkflowSpec{
		RepoSpec:     RepoSpec{RepoSlug: sourceSpec.Repo, Version: latestRef},
		WorkflowPath: sourceSpec.Path,
	}
	processedContent, err := processIncludesInContent(finalContent, wfSpec, latestRef, filepath.Dir(wf.Path), opts.Verbose)
	if err != nil {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
		}
	} else {
		finalContent = processedContent
	}
	return finalContent, nil
}

// applyStopAfterModifications updates the stop-after field in content based on opts.
func applyWorkflowStopAfter(content string, opts UpdateWorkflowsOptions) string {
	if opts.NoStopAfter {
		cleaned, err := RemoveFieldFromOnTrigger(content, "stop-after")
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove stop-after field: %v", err)))
			}
			return content
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed stop-after field from workflow"))
		}
		return cleaned
	}
	if opts.StopAfter != "" {
		updated, err := SetFieldInOnTrigger(content, "stop-after", opts.StopAfter)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set stop-after field: %v", err)))
			}
			return content
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Set stop-after field to: "+opts.StopAfter))
		}
		return updated
	}
	return content
}

// securityScanWorkflow checks the final content for dangerous patterns; returns an error if found.
func securityScanWorkflow(wf *workflowWithSource, finalContent string, opts UpdateWorkflowsOptions) error {
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

// writeAndCompileWorkflow writes the final content to disk and, if requested, compiles the workflow.
func writeAndCompileWorkflow(ctx context.Context, wf *workflowWithSource, finalContent string, hasConflicts bool, currentRef, latestRef string, opts UpdateWorkflowsOptions) error {
	if err := os.WriteFile(wf.Path, []byte(finalContent), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}
	if hasConflicts {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Updated %s from %s to %s with CONFLICTS - please review and resolve manually", wf.Name, shortRef(currentRef), shortRef(latestRef))))
		return nil
	}
	updateLog.Printf("Successfully updated workflow %s from %s to %s", wf.Name, currentRef, latestRef)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", wf.Name, shortRef(currentRef), shortRef(latestRef))))
	if !opts.NoCompile {
		updateLog.Printf("Compiling updated workflow: %s", wf.Name)
		if err := compileWorkflowsForUpdate(ctx, []string{wf.Path}, opts.WorkflowsDir, opts.EngineOverride, opts.Verbose, opts.Approve); err != nil {
			updateLog.Printf("Compilation failed for workflow %s: %v", wf.Name, err)
			return fmt.Errorf("failed to compile updated workflow: %w", err)
		}
	} else {
		updateLog.Printf("Skipping compilation of workflow %s (--no-compile specified)", wf.Name)
	}
	return nil
}

// updateWorkflow updates a single workflow from its source
func updateWorkflow(ctx context.Context, wf *workflowWithSource, opts UpdateWorkflowsOptions) error {
	updateLog.Printf("Updating workflow: name=%s, source=%s, force=%v, noMerge=%v", wf.Name, wf.SourceSpec, opts.Force, opts.NoMerge)

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating workflow: "+wf.Name))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Source: "+wf.SourceSpec))
	}

	// Parse source spec
	initialSourceSpec, err := parseSourceSpec(wf.SourceSpec)
	if err != nil {
		updateLog.Printf("Failed to parse source spec: %v", err)
		return fmt.Errorf("failed to parse source spec: %w", err)
	}

	resolvedLocation, err := resolveRedirectedUpdateLocation(ctx, wf.Name, initialSourceSpec, opts.AllowMajor, opts.Verbose, opts.NoRedirect, opts.CoolDown)
	if err != nil {
		return err
	}

	sourceSpec := resolvedLocation.sourceSpec
	currentRef := resolvedLocation.currentRef
	latestRef := resolvedLocation.latestRef
	sourceFieldRef := resolvedLocation.sourceFieldRef

	if resolvedLocation.coolDownBlocked {
		return nil
	}

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Current ref: "+currentRef))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Latest ref: "+latestRef))
	}

	if !opts.Force && currentRef == latestRef && len(resolvedLocation.redirectHistory) == 0 {
		return handleWorkflowUpToDate(ctx, wf, sourceSpec, currentRef, opts)
	}

	if len(resolvedLocation.redirectHistory) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow %s source location changed; updating source to %s/%s@%s", wf.Name, sourceSpec.Repo, sourceSpec.Path, sourceFieldRef)))
	}

	return applyResolvedWorkflowUpdate(ctx, wf, resolvedLocation, opts)
}

func applyResolvedWorkflowUpdate(ctx context.Context, wf *workflowWithSource, resolvedLocation *resolvedUpdateLocation, opts UpdateWorkflowsOptions) error {
	sourceSpec := resolvedLocation.sourceSpec
	currentRef := resolvedLocation.currentRef
	latestRef := resolvedLocation.latestRef
	sourceFieldRef := resolvedLocation.sourceFieldRef
	newContent := resolvedLocation.content

	merge := detectWorkflowMergeMode(ctx, wf, sourceSpec, currentRef, len(resolvedLocation.redirectHistory) > 0, opts)

	var err error
	var finalContent string
	var hasConflicts bool
	if merge {
		finalContent, hasConflicts, err = buildMergedWorkflowContent(ctx, wf, sourceSpec, currentRef, sourceFieldRef, newContent, opts)
		if err != nil {
			return err
		}
	} else {
		finalContent, err = buildOverrideWorkflowContent(wf, sourceSpec, latestRef, sourceFieldRef, newContent, opts)
		if err != nil {
			return err
		}
	}

	finalContent = applyWorkflowStopAfter(finalContent, opts)

	if err := securityScanWorkflow(wf, finalContent, opts); err != nil {
		return err
	}

	return writeAndCompileWorkflow(ctx, wf, finalContent, hasConflicts, currentRef, latestRef, opts)
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
