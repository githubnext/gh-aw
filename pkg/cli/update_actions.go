package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/semverutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
)

// isCoreAction returns true if the repo is a GitHub-maintained core action (actions/* org).
// Core actions are always updated to the latest major version without requiring --major.
func isCoreAction(repo string) bool {
	return strings.HasPrefix(repo, "actions/")
}

// isGhAwNativeAction returns true if the action repo is part of the gh-aw native ecosystem
// (i.e., maintained in the github/gh-aw or github/gh-aw-actions repository). These actions
// are versioned in lock-step with the CLI and must never be updated beyond the running CLI version.
func isGhAwNativeAction(repo string) bool {
	base := gitutil.ExtractBaseRepo(repo)
	return base == "github/gh-aw" || base == "github/gh-aw-actions"
}

type actionUpdateDeps struct {
	getLatestRelease       func(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error)
	getLatestReleaseViaGit func(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error)
	runGHReleasesAPI       func(ctx context.Context, baseRepo string) ([]byte, error)
	getActionSHAForTag     func(ctx context.Context, repo, tag string) (string, error)
}

type entrySnapshot struct {
	key   string
	entry workflow.ActionCacheEntry
}

type actionUpdateSummary struct {
	updatedActions []string
	failedActions  []actionUpdateFailure
	skippedActions []string
}

type actionCacheEntryContext struct {
	deps               actionUpdateDeps
	actionCache        *workflow.ActionCache
	allowMajor         bool
	verbose            bool
	disableReleaseBump bool
	coolDown           time.Duration
	summary            *actionUpdateSummary
}

func defaultActionUpdateDeps() actionUpdateDeps {
	return actionUpdateDeps{
		getLatestRelease:       getLatestActionRelease,
		getLatestReleaseViaGit: getLatestActionReleaseViaGit,
		runGHReleasesAPI: func(ctx context.Context, baseRepo string) ([]byte, error) {
			return workflow.RunGHCombinedContext(ctx, "Fetching releases...", "api", fmt.Sprintf("/repos/%s/releases", baseRepo), "--jq", ".[].tag_name")
		},
		getActionSHAForTag: getActionSHAForTag,
	}
}

// UpdateActions updates GitHub Actions versions in .github/aw/actions-lock.json
// It checks each action for newer releases and updates the SHA if a newer version is found.
// By default all actions are updated to the latest major version; pass disableReleaseBump=true
// to revert to the old behaviour where only core (actions/*) actions bypass the --major flag.
//
// coolDown specifies the minimum age a release must have before it is applied. Repos under the
// "actions/" and "github/" namespaces are always exempt from the cooldown.
//
// The ActionCache helpers from pkg/workflow are used so that cached inputs and descriptions
// for safe-outputs.actions entries are preserved when their SHA is unchanged, and cleared
// when the SHA changes (prompting a re-fetch on the next compile).
func UpdateActions(ctx context.Context, allowMajor, verbose, disableReleaseBump bool, coolDown time.Duration) error {
	return updateActions(ctx, defaultActionUpdateDeps(), allowMajor, verbose, disableReleaseBump, coolDown)
}

func updateActions(ctx context.Context, deps actionUpdateDeps, allowMajor, verbose, disableReleaseBump bool, coolDown time.Duration) error {
	updateLog.Print("Starting action updates")
	printUpdateActionsStart(verbose)
	actionCache, err := loadActionCache(verbose)
	if err != nil || actionCache == nil {
		return err
	}

	summary := actionUpdateSummary{}
	entryContext := actionCacheEntryContext{
		deps:               deps,
		actionCache:        actionCache,
		allowMajor:         allowMajor,
		verbose:            verbose,
		disableReleaseBump: disableReleaseBump,
		coolDown:           coolDown,
		summary:            &summary,
	}
	for _, snapshotEntry := range snapshotActionCacheEntries(actionCache) {
		if err := processActionCacheEntry(ctx, snapshotEntry, entryContext); err != nil {
			return err
		}
	}

	printActionUpdateSummary(summary, verbose)
	return saveUpdatedActionCache(actionCache, summary.updatedActions)
}

func printUpdateActionsStart(verbose bool) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking for GitHub Actions updates..."))
	}
}

func loadActionCache(verbose bool) (*workflow.ActionCache, error) {
	actionsLockPath := filepath.Join(".github", "aw", "actions-lock.json")
	if _, err := os.Stat(actionsLockPath); os.IsNotExist(err) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Actions lock file not found: "+actionsLockPath))
		}
		return nil, nil
	}

	actionCache := workflow.NewActionCache(".")
	if err := actionCache.Load(); err != nil {
		return nil, fmt.Errorf("failed to parse actions lock file: %w", err)
	}
	updateLog.Printf("Loaded %d action entries from actions-lock.json", len(actionCache.Entries))
	return actionCache, nil
}

func snapshotActionCacheEntries(actionCache *workflow.ActionCache) []entrySnapshot {
	snapshot := make([]entrySnapshot, 0, len(actionCache.Entries))
	for key, entry := range actionCache.Entries {
		snapshot = append(snapshot, entrySnapshot{key: key, entry: entry})
	}
	return snapshot
}

func processActionCacheEntry(ctx context.Context, snapshotEntry entrySnapshot, entryCtx actionCacheEntryContext) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	entry := snapshotEntry.entry
	updateLog.Printf("Checking action: %s@%s", entry.Repo, entry.Version)

	latestVersion, latestSHA, err := resolveLatestActionUpdate(ctx, entryCtx.deps, entry, entryCtx.allowMajor, entryCtx.verbose, entryCtx.disableReleaseBump)
	if err != nil {
		recordActionUpdateFailure(entryCtx.summary, entryCtx.verbose, entry.Repo, err)
		return nil
	}
	if shouldSkipActionDowngrade(entry, latestVersion) {
		recordActionDowngradeSkip(entryCtx.summary, entry, latestVersion)
		return nil
	}
	if latestVersion == entry.Version && latestSHA == entry.SHA {
		recordActionUpToDateSkip(entryCtx.summary, entryCtx.verbose, entry)
		return nil
	}
	if shouldSkipActionForCoolDown(ctx, entryCtx.actionCache, entry.Repo, latestVersion, entryCtx.coolDown, entryCtx.verbose) {
		entryCtx.summary.skippedActions = append(entryCtx.summary.skippedActions, entry.Repo)
		return nil
	}

	applyActionCacheUpdate(entryCtx.actionCache, snapshotEntry, entry, latestVersion, latestSHA)
	entryCtx.summary.updatedActions = append(entryCtx.summary.updatedActions, entry.Repo)
	return nil
}

func resolveLatestActionUpdate(ctx context.Context, deps actionUpdateDeps, entry workflow.ActionCacheEntry, allowMajor, verbose, disableReleaseBump bool) (string, string, error) {
	effectiveAllowMajor := !disableReleaseBump || allowMajor || isCoreAction(entry.Repo)
	latestVersion, latestSHA, err := deps.getLatestRelease(ctx, entry.Repo, entry.Version, effectiveAllowMajor, verbose)
	if err != nil {
		return "", "", err
	}
	return capNativeActionToCLIVersion(ctx, deps, entry.Repo, latestVersion, latestSHA, verbose)
}

func capNativeActionToCLIVersion(ctx context.Context, deps actionUpdateDeps, repo, latestVersion, latestSHA string, verbose bool) (string, string, error) {
	if !isGhAwNativeAction(repo) {
		return latestVersion, latestSHA, nil
	}
	cliVersion := GetVersion()
	cliVer := parseVersion(cliVersion)
	latestVer := parseVersion(latestVersion)
	if cliVer == nil || latestVer == nil || !latestVer.IsNewer(cliVer) {
		return latestVersion, latestSHA, nil
	}

	cappedVersion := semverutil.EnsureVPrefix(cliVersion)
	updateLog.Printf("Capping %s update to CLI version %s (latest available %s exceeds running CLI)", repo, cappedVersion, latestVersion)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s: capping update target to CLI version %s (latest %s is newer than running CLI)", repo, cappedVersion, latestVersion)))
	}
	cappedSHA, err := deps.getActionSHAForTag(ctx, gitutil.ExtractBaseRepo(repo), cappedVersion)
	if err != nil {
		updateLog.Printf("Cannot resolve SHA for %s@%s (CLI version cap): %v; skipping update", repo, cappedVersion, err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: cannot resolve SHA for CLI version %s: %v", repo, cappedVersion, err)))
		return "", "", fmt.Errorf("cannot resolve SHA for CLI version %s: %w", cappedVersion, err)
	}
	return cappedVersion, cappedSHA, nil
}

func recordActionUpdateFailure(summary *actionUpdateSummary, verbose bool, repo string, err error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check %s: %v", repo, err)))
	}
	summary.failedActions = append(summary.failedActions, actionUpdateFailure{name: repo, err: err.Error()})
}

func shouldSkipActionDowngrade(entry workflow.ActionCacheEntry, latestVersion string) bool {
	currentVer := parseVersion(entry.Version)
	latestVer := parseVersion(latestVersion)
	return currentVer != nil && latestVer != nil && currentVer.IsNewer(latestVer)
}

func recordActionDowngradeSkip(summary *actionUpdateSummary, entry workflow.ActionCacheEntry, latestVersion string) {
	updateLog.Printf("Skipping %s: proposed version %s is older than current %s (would be a downgrade)", entry.Repo, latestVersion, entry.Version)
	msg := fmt.Sprintf("%s: skipping proposed update from %s to %s (would be a downgrade)", entry.Repo, entry.Version, latestVersion)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
	summary.skippedActions = append(summary.skippedActions, entry.Repo)
}

func recordActionUpToDateSkip(summary *actionUpdateSummary, verbose bool, entry workflow.ActionCacheEntry) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s@%s is up to date", entry.Repo, entry.Version)))
	}
	summary.skippedActions = append(summary.skippedActions, entry.Repo)
}

func shouldSkipActionForCoolDown(ctx context.Context, actionCache *workflow.ActionCache, repo, latestVersion string, coolDown time.Duration, verbose bool) bool {
	if isExemptFromCoolDown(repo) {
		return false
	}
	coolDownResult := lookupActionCoolDown(ctx, actionCache, repo, latestVersion, coolDown)
	if !coolDownResult.InCoolDown {
		return false
	}
	cooldownLog.Printf("Action %s: %s", repo, coolDownResult.Message)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping update for %s: %s", repo, coolDownResult.Message)))
	return true
}

func lookupActionCoolDown(ctx context.Context, actionCache *workflow.ActionCache, repo, latestVersion string, coolDown time.Duration) coolDownCheckResult {
	if cachedDate, ok := actionCache.GetReleasedAt(repo, latestVersion); ok {
		return checkReleaseCoolDownWithDate(repo, latestVersion, cachedDate, coolDown)
	}
	result := checkReleaseCoolDown(ctx, repo, latestVersion, coolDown)
	if !result.PublishedAt.IsZero() {
		actionCache.SetReleasedAt(repo, latestVersion, result.PublishedAt)
	}
	return result
}

func applyActionCacheUpdate(actionCache *workflow.ActionCache, snapshotEntry entrySnapshot, entry workflow.ActionCacheEntry, latestVersion, latestSHA string) {
	oldSHAStr := shortActionSHA(entry.SHA)
	newSHAStr := shortActionSHA(latestSHA)
	updateLog.Printf("Updating %s from %s (%s) to %s (%s)", entry.Repo, entry.Version, oldSHAStr, latestVersion, newSHAStr)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", entry.Repo, entry.Version, latestVersion)))
	if latestVersion != entry.Version {
		actionCache.DeleteByKey(snapshotEntry.key)
	}
	actionCache.Set(entry.Repo, latestVersion, latestSHA)
}

func shortActionSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func printActionUpdateSummary(summary actionUpdateSummary, verbose bool) {
	fmt.Fprintln(os.Stderr, "")
	if len(summary.updatedActions) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %d action(s):", len(summary.updatedActions))))
		for _, action := range summary.updatedActions {
			fmt.Fprintln(os.Stderr, console.FormatListItem(action))
		}
		fmt.Fprintln(os.Stderr, "")
	}
	if len(summary.skippedActions) > 0 && verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("%d action(s) already up to date", len(summary.skippedActions))))
		fmt.Fprintln(os.Stderr, "")
	}
	if len(summary.failedActions) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check %d action(s):", len(summary.failedActions))))
		for _, failure := range summary.failedActions {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", failure.name, failure.err)
		}
		fmt.Fprintln(os.Stderr, "")
	}
}

func saveUpdatedActionCache(actionCache *workflow.ActionCache, updatedActions []string) error {
	if actionCache == nil || len(updatedActions) == 0 {
		return nil
	}
	if err := actionCache.Save(); err != nil {
		return fmt.Errorf("failed to save actions lock file: %w", err)
	}
	updateLog.Printf("Successfully wrote updated actions-lock.json with %d updates", len(updatedActions))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updated actions-lock.json file"))
	return nil
}

// getLatestActionRelease gets the latest release for an action repository
// It respects semantic versioning and the allowMajor flag
func getLatestActionRelease(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	return getLatestActionReleaseWithDeps(ctx, defaultActionUpdateDeps(), repo, currentVersion, allowMajor, verbose)
}

func getLatestActionReleaseWithDeps(ctx context.Context, deps actionUpdateDeps, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	updateLog.Printf("Getting latest release for %s@%s (allowMajor=%v)", repo, currentVersion, allowMajor)

	baseRepo := gitutil.ExtractBaseRepo(repo)
	updateLog.Printf("Using base repository: %s for action: %s", baseRepo, repo)

	output, err := deps.runGHReleasesAPI(ctx, baseRepo)
	if err != nil {
		releases, handled, fallbackErr := handleActionReleaseAPIError(ctx, deps, repo, currentVersion, allowMajor, verbose, err, output)
		if handled {
			return releases.version, releases.sha, fallbackErr
		}
		return "", "", fallbackErr
	}

	releases := splitReleaseLines(output)
	if len(releases) == 0 {
		return fallbackToGitActionRelease(ctx, deps, repo, currentVersion, allowMajor, verbose, baseRepo)
	}
	latestRelease, err := selectLatestActionRelease(releases, currentVersion, allowMajor)
	if err != nil {
		return "", "", err
	}
	sha, err := deps.getActionSHAForTag(ctx, baseRepo, latestRelease)
	if err != nil {
		return "", "", fmt.Errorf("failed to get SHA for %s: %w", latestRelease, err)
	}
	return latestRelease, sha, nil
}

// getLatestActionReleaseViaGit gets the latest release using git ls-remote (fallback)
func getLatestActionReleaseViaGit(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest release for %s via git ls-remote (current: %s, allow major: %v)", repo, currentVersion, allowMajor)))
	}

	baseRepo := gitutil.ExtractBaseRepo(repo)
	updateLog.Printf("Using base repository: %s for action: %s (git fallback)", baseRepo, repo)

	githubHost := getGitHubHostForRepo(baseRepo)
	repoURL := fmt.Sprintf("%s/%s.git", githubHost, baseRepo)

	// List all tags
	// #nosec G204 -- repoURL is constructed from workflow configuration authored by the developer
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--tags", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch releases via git ls-remote: %w", err)
	}

	releases, tagToSHA := parseGitRemoteTags(output)
	if len(releases) == 0 {
		return "", "", errors.New("no releases found")
	}

	latestCompatible, err := selectLatestActionRelease(releases, currentVersion, allowMajor)
	if err != nil {
		return "", "", err
	}
	if verbose && parseVersion(currentVersion) == nil {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current version is not valid, using highest semver release: %s (via git)", latestCompatible)))
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest compatible release: %s (via git)", latestCompatible)))
	}

	return latestCompatible, tagToSHA[latestCompatible], nil
}

type actionReleaseResolution struct {
	version string
	sha     string
}

type releaseWithVersion struct {
	tag     string
	version *semverutil.SemanticVersion
}

func handleActionReleaseAPIError(ctx context.Context, deps actionUpdateDeps, repo, currentVersion string, allowMajor, verbose bool, err error, output []byte) (actionReleaseResolution, bool, error) {
	outputStr := string(output)
	if gitutil.IsAuthError(outputStr) || gitutil.IsAuthError(err.Error()) {
		updateLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s", repo)
		latestRelease, latestSHA, gitErr := deps.getLatestReleaseViaGit(ctx, repo, currentVersion, allowMajor, verbose)
		if gitErr != nil {
			return actionReleaseResolution{}, true, fmt.Errorf("failed to fetch releases via GitHub API and git: API error: %w, Git Error: %w", err, gitErr)
		}
		return actionReleaseResolution{version: latestRelease, sha: latestSHA}, true, nil
	}
	if trimmed := strings.TrimSpace(outputStr); trimmed != "" {
		return actionReleaseResolution{}, false, fmt.Errorf("failed to fetch releases: %w: %s", err, trimmed)
	}
	return actionReleaseResolution{}, false, fmt.Errorf("failed to fetch releases: %w", err)
}

func splitReleaseLines(output []byte) []string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func fallbackToGitActionRelease(ctx context.Context, deps actionUpdateDeps, repo, currentVersion string, allowMajor, verbose bool, baseRepo string) (string, string, error) {
	updateLog.Printf("No releases found via GitHub API for %s, falling back to git ls-remote tag scan", baseRepo)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(baseRepo+": no GitHub Releases found, falling back to tag scanning (safe to ignore)"))
	}
	latestRelease, latestSHA, err := deps.getLatestReleaseViaGit(ctx, repo, currentVersion, allowMajor, verbose)
	if err != nil {
		return "", "", fmt.Errorf("no releases or tags found for %s: %w", baseRepo, err)
	}
	return latestRelease, latestSHA, nil
}

func parseGitRemoteTags(output []byte) ([]string, map[string]string) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	releases := make([]string, 0, len(lines))
	tagToSHA := make(map[string]string)
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			sha := parts[0]
			tagRef := parts[1]
			if strings.HasSuffix(tagRef, "^{}") {
				continue
			}
			tag := strings.TrimPrefix(tagRef, "refs/tags/")
			releases = append(releases, tag)
			tagToSHA[tag] = sha
		}
	}
	return releases, tagToSHA
}

func selectLatestActionRelease(releases []string, currentVersion string, allowMajor bool) (string, error) {
	validReleases := collectValidStableActionReleases(releases)
	if len(validReleases) == 0 {
		return "", errors.New("no valid semantic version releases found")
	}
	currentVer := parseVersion(currentVersion)
	if currentVer == nil {
		return validReleases[0].tag, nil
	}

	latestCompatible := pickLatestCompatibleActionRelease(validReleases, currentVer, allowMajor)
	if latestCompatible == "" {
		return "", errors.New("no compatible release found")
	}
	return latestCompatible, nil
}

func collectValidStableActionReleases(releases []string) []releaseWithVersion {
	var validReleases []releaseWithVersion
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer != nil && releaseVer.Pre == "" {
			validReleases = append(validReleases, releaseWithVersion{
				tag:     release,
				version: releaseVer,
			})
		}
	}
	slices.SortFunc(validReleases, func(a, b releaseWithVersion) int {
		switch {
		case a.version.IsNewer(b.version):
			return -1
		case b.version.IsNewer(a.version):
			return 1
		default:
			return 0
		}
	})
	return validReleases
}

func pickLatestCompatibleActionRelease(validReleases []releaseWithVersion, currentVer *semverutil.SemanticVersion, allowMajor bool) string {
	var latestCompatible string
	var latestCompatibleVersion *semverutil.SemanticVersion
	for _, rel := range validReleases {
		if !allowMajor && rel.version.Major != currentVer.Major {
			continue
		}
		if latestCompatibleVersion == nil || rel.version.IsNewer(latestCompatibleVersion) {
			latestCompatible = rel.tag
			latestCompatibleVersion = rel.version
		} else if !rel.version.IsNewer(latestCompatibleVersion) &&
			rel.version.Major == latestCompatibleVersion.Major &&
			rel.version.Minor == latestCompatibleVersion.Minor &&
			rel.version.Patch == latestCompatibleVersion.Patch {
			// If versions are equal, prefer the less precise one (e.g., "v8" over "v8.0.0")
			// This follows GitHub Actions convention of using major version tags
			if !rel.version.IsPreciseVersion() && latestCompatibleVersion.IsPreciseVersion() {
				latestCompatible = rel.tag
				latestCompatibleVersion = rel.version
			}
		}
	}
	return latestCompatible
}

// getActionSHAForTag gets the commit SHA for a given tag in an action repository.
// For annotated tags (and chained tag objects), it iteratively peels until it
// reaches the underlying non-tag object SHA, matching what tools like Renovate expect.
func getActionSHAForTag(ctx context.Context, repo, tag string) (string, error) {
	updateLog.Printf("Getting SHA for %s@%s", repo, tag)

	// Fetch both SHA and object type to detect annotated tags.
	// Annotated tags have type "tag" and their SHA points to the tag object,
	// not the underlying commit. We must peel to get the commit SHA.
	output, err := workflow.RunGHContext(ctx, "Fetching tag info...", "api", fmt.Sprintf("/repos/%s/git/ref/tags/%s", repo, tag), "--jq", "[.object.sha, .object.type] | @tsv")
	if err != nil {
		return "", fmt.Errorf("failed to resolve tag: %w", err)
	}

	sha, objType, err := workflow.ParseTagRefTSV(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to parse API response for %s@%s: %w", repo, tag, err)
	}

	// Annotated tags (and chained tag objects) point to a tag object rather than
	// directly to a commit. Iteratively peel until we reach a non-tag object so
	// that emitted action pins use the stable underlying commit SHA rather than a
	// mutable tag object SHA (which changes when the tag is re-created).
	const maxTagPeelDepth = 10
	for depth := 0; objType == "tag"; depth++ {
		if depth >= maxTagPeelDepth {
			return "", fmt.Errorf("failed to peel annotated tag: exceeded max depth %d for %s@%s", maxTagPeelDepth, repo, tag)
		}
		updateLog.Printf("Detected annotated tag for %s@%s (depth %d, tag object SHA: %s), peeling to underlying object", repo, tag, depth, sha)
		output2, err := workflow.RunGHContext(ctx, "Peeling annotated tag...", "api", fmt.Sprintf("/repos/%s/git/tags/%s", repo, sha), "--jq", "[.object.sha, .object.type] | @tsv")
		if err != nil {
			return "", fmt.Errorf("failed to peel annotated tag: %w", err)
		}
		sha, objType, err = workflow.ParseTagRefTSV(string(output2))
		if err != nil {
			return "", fmt.Errorf("failed to parse peeled tag API response for %s@%s: %w", repo, tag, err)
		}
	}
	updateLog.Printf("Resolved %s@%s to %s SHA: %s", repo, tag, objType, sha)

	return sha, nil
}

// actionRefPattern matches "uses: org/repo@SHA-or-tag" in workflow files for any org.
// Requires the org to start with an alphanumeric character and contain only alphanumeric,
// hyphens, or underscores (no dots, matching GitHub's org naming rules) to exclude local
// paths (e.g. "./..."). Repository names may additionally contain dots.
// Captures: (1) indentation+uses prefix, (2) repo path, (3) SHA or version tag,
// (4) optional version comment (e.g., "v6.0.2" from "# v6.0.2"), (5) trailing whitespace.
var actionRefPattern = regexp.MustCompile(`(uses:\s+)([a-zA-Z0-9][a-zA-Z0-9_-]*/[a-zA-Z0-9_.-]+(?:/[a-zA-Z0-9_.-]+)*)@([a-fA-F0-9]{40}|[^\s#\n]+?)(\s*#\s*\S+)?(\s*)$`)

// latestReleaseResult caches a resolved version/SHA pair.
type latestReleaseResult struct {
	version string
	sha     string
}

type workflowActionRefMatch struct {
	prefix   string
	repo     string
	ref      string
	comment  string
	trailing string
	indent   string
}

type workflowActionUpdateContext struct {
	deps          actionUpdateDeps
	cache         map[string]latestReleaseResult
	coolDownCache map[string]coolDownCheckResult
	allowMajor    bool
	verbose       bool
	coolDown      time.Duration
}

// UpdateActionsInWorkflowFiles scans all workflow .md files under workflowsDir
// (recursively) and updates any "uses: org/repo@version" references to the latest
// major version. Updated files are recompiled. By default all actions are updated to
// the latest major version; pass disableReleaseBump=true to only update core
// (actions/*) references.
func UpdateActionsInWorkflowFiles(ctx context.Context, workflowsDir, engineOverride string, verbose, disableReleaseBump bool, noCompile bool, coolDown time.Duration, approve bool) error {
	return updateActionsInWorkflowFiles(ctx, defaultActionUpdateDeps(), updateActionsOptions{
		workflowsDir:       workflowsDir,
		engineOverride:     engineOverride,
		verbose:            verbose,
		disableReleaseBump: disableReleaseBump,
		noCompile:          noCompile,
		coolDown:           coolDown,
		approve:            approve,
	})
}

// updateActionsOptions bundles the configuration parameters for updateActionsInWorkflowFiles,
// collapsing a long positional parameter list into a struct.
// engineOverride sets a non-default agentic engine for recompiled workflows.
// disableReleaseBump prevents upgrading action/skill references to newer releases.
// noCompile skips recompilation of updated workflow files.
// coolDown is the minimum age a release must have before it is considered for upgrade.
// approve auto-approves any interactive prompts during recompilation.
type updateActionsOptions struct {
	workflowsDir       string
	engineOverride     string
	verbose            bool
	disableReleaseBump bool
	noCompile          bool
	coolDown           time.Duration
	approve            bool
}

func updateActionsInWorkflowFiles(ctx context.Context, deps actionUpdateDeps, opts updateActionsOptions) error {
	if opts.workflowsDir == "" {
		opts.workflowsDir = getWorkflowsDir()
	}

	updateLog.Printf("Updating action references in workflow files: dir=%s", opts.workflowsDir)

	// Per-invocation cache: key = "repo@currentVersion", avoids repeated API calls
	cache := make(map[string]latestReleaseResult)
	// Per-invocation cooldown cache: key = "repo@tag", avoids redundant date API calls
	coolDownCache := make(map[string]coolDownCheckResult)

	var updatedFiles []string

	err := filepath.WalkDir(opts.workflowsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		updated, err := updateWorkflowFileReferences(ctx, deps, path, d.Name(), cache, coolDownCache, opts)
		if err != nil {
			return err
		}
		if updated {
			updatedFiles = append(updatedFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk workflows directory: %w", err)
	}

	if len(updatedFiles) == 0 && opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No action references needed updating in workflow files"))
	}

	return nil
}

func updateWorkflowFileReferences(ctx context.Context, deps actionUpdateDeps, path, displayName string, cache map[string]latestReleaseResult, coolDownCache map[string]coolDownCheckResult, opts updateActionsOptions) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		logUpdateWorkflowFileWarning(opts.verbose, "Failed to read", path, err)
		return false, nil
	}

	updatedActions, newContent, err := updateActionRefsInContentWithDeps(ctx, deps, string(content), cache, coolDownCache, !opts.disableReleaseBump, opts.verbose, opts.coolDown)
	if err != nil {
		logUpdateWorkflowFileWarning(opts.verbose, "Failed to update action refs in", path, err)
		return false, nil
	}
	updatedSkills, newContent, err := updateSkillRefsInContent(ctx, newContent, !opts.disableReleaseBump, opts.verbose, opts.coolDown)
	if err != nil {
		logUpdateWorkflowFileWarning(opts.verbose, "Failed to update skill refs in", path, err)
		return false, nil
	}
	if !updatedActions && !updatedSkills {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(newContent), constants.FilePermPublic); err != nil {
		return false, fmt.Errorf("failed to write updated workflow %s: %w", path, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated action/skill references in "+displayName))
	recompileUpdatedWorkflowFile(ctx, path, opts)
	return true, nil
}

func logUpdateWorkflowFileWarning(verbose bool, prefix, path string, err error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%s %s: %v", prefix, path, err)))
	}
}

func recompileUpdatedWorkflowFile(ctx context.Context, path string, opts updateActionsOptions) {
	if opts.noCompile {
		return
	}
	if err := compileWorkflowWithRefresh(ctx, path, opts.verbose, false, opts.engineOverride, false, opts.approve); err != nil && opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to recompile %s: %v", path, err)))
	}
}

type skillRefUpdateResolver func(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error)

func updateSkillRefsInContent(ctx context.Context, content string, allowMajor, verbose bool, coolDown time.Duration) (bool, string, error) {
	return updateSkillRefsInContentWithResolver(ctx, content, allowMajor, verbose, coolDown, resolveLatestRef)
}

func updateSkillRefsInContentWithResolver(
	ctx context.Context,
	content string,
	allowMajor, verbose bool,
	coolDown time.Duration,
	resolver skillRefUpdateResolver,
) (bool, string, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		if verbose {
			updateLog.Printf("Skipping skill update for content without parseable frontmatter: %v", err)
		}
		return false, content, nil
	}
	if result == nil || result.Frontmatter == nil {
		return false, content, nil
	}

	rawSkills, ok := result.Frontmatter["skills"].([]any)
	if !ok || len(rawSkills) == 0 {
		return false, content, nil
	}

	changed := false
	for i, rawSkill := range rawSkills {
		switch typed := rawSkill.(type) {
		case string:
			updated, updatedRef, err := updateSkillRefValue(ctx, typed, allowMajor, verbose, coolDown, resolver)
			if err != nil {
				return false, content, err
			}
			if updated {
				rawSkills[i] = updatedRef
				changed = true
			}
		case map[string]any:
			skillRef, ok := typed["skill"].(string)
			if !ok {
				continue
			}
			updated, updatedRef, err := updateSkillRefValue(ctx, skillRef, allowMajor, verbose, coolDown, resolver)
			if err != nil {
				return false, content, err
			}
			if updated {
				typed["skill"] = updatedRef
				changed = true
			}
		}
	}
	if !changed {
		return false, content, nil
	}
	result.Frontmatter["skills"] = rawSkills

	updatedFrontmatter, err := yaml.Marshal(result.Frontmatter)
	if err != nil {
		return false, content, fmt.Errorf("failed to marshal updated frontmatter: %w", err)
	}
	updatedContent, err := parser.ReconstructWorkflowFile(parser.QuoteCronExpressions(string(updatedFrontmatter)), result.Markdown)
	if err != nil {
		return false, content, fmt.Errorf("failed to reconstruct workflow file: %w", err)
	}
	return true, updatedContent, nil
}

func updateSkillRefValue(
	ctx context.Context,
	skillRef string,
	allowMajor, verbose bool,
	coolDown time.Duration,
	resolver skillRefUpdateResolver,
) (bool, string, error) {
	trimmedSkillRef := strings.TrimSpace(skillRef)
	if trimmedSkillRef == "" || strings.Contains(trimmedSkillRef, "${{") {
		return false, skillRef, nil
	}
	spec, currentRef, ok := strings.Cut(trimmedSkillRef, "@")
	spec = strings.TrimSpace(spec)
	currentRef = strings.TrimSpace(currentRef)
	if !ok || spec == "" || currentRef == "" {
		return false, skillRef, nil
	}

	repo := gitutil.ExtractBaseRepo(spec)
	if repo == "" {
		return false, skillRef, nil
	}
	latestRef, err := resolver(ctx, repo, currentRef, allowMajor, verbose, coolDown)
	if err != nil {
		if verbose {
			updateLog.Printf("Skipping skill update for %s@%s: %v", spec, currentRef, err)
		}
		return false, skillRef, nil
	}
	if latestRef == "" || latestRef == currentRef {
		return false, skillRef, nil
	}
	return true, spec + "@" + latestRef, nil
}

func updateActionRefsInContentWithDeps(ctx context.Context, deps actionUpdateDeps, content string, cache map[string]latestReleaseResult, coolDownCache map[string]coolDownCheckResult, allowMajor, verbose bool, coolDown time.Duration) (bool, string, error) {
	changed := false
	lines := strings.Split(content, "\n")
	updateCtx := workflowActionUpdateContext{
		deps:          deps,
		cache:         cache,
		coolDownCache: coolDownCache,
		allowMajor:    allowMajor,
		verbose:       verbose,
		coolDown:      coolDown,
	}

	for i, line := range lines {
		match, ok := parseWorkflowActionRefMatch(line)
		if !ok {
			continue
		}
		newLine, updated, err := resolveUpdatedWorkflowActionLine(ctx, line, match, updateCtx)
		if err != nil {
			return false, "", err
		}
		if !updated {
			continue
		}
		updateLog.Printf("Updating %s from %s to %s in line %d", match.repo, match.ref, extractWorkflowActionTargetVersion(newLine), i+1)
		lines[i] = newLine
		changed = true
	}

	return changed, strings.Join(lines, "\n"), nil
}

func parseWorkflowActionRefMatch(line string) (workflowActionRefMatch, bool) {
	match := actionRefPattern.FindStringSubmatchIndex(line)
	if match == nil {
		return workflowActionRefMatch{}, false
	}
	refMatch := workflowActionRefMatch{
		indent: line[:match[2]],
		prefix: line[match[2]:match[3]],
		repo:   line[match[4]:match[5]],
		ref:    line[match[6]:match[7]],
	}
	if match[8] >= 0 {
		refMatch.comment = line[match[8]:match[9]]
	}
	if match[10] >= 0 {
		refMatch.trailing = line[match[10]:match[11]]
	}
	return refMatch, true
}

func resolveUpdatedWorkflowActionLine(ctx context.Context, originalLine string, match workflowActionRefMatch, updateCtx workflowActionUpdateContext) (string, bool, error) {
	effectiveAllowMajor := updateCtx.allowMajor || isCoreAction(match.repo)
	if !effectiveAllowMajor {
		return originalLine, false, nil
	}

	currentVersion, isSHA := workflowActionCurrentVersion(match)
	result, ok := resolveCachedWorkflowActionRelease(ctx, updateCtx.deps, match.repo, currentVersion, effectiveAllowMajor, updateCtx.verbose, updateCtx.cache)
	if !ok {
		return originalLine, false, nil
	}
	if shouldSkipWorkflowActionRef(match, result, isSHA) {
		return originalLine, false, nil
	}
	if shouldSkipWorkflowActionCoolDown(ctx, updateCtx.coolDownCache, match.repo, result.version, updateCtx.coolDown, updateCtx.verbose) {
		return originalLine, false, nil
	}
	return buildUpdatedWorkflowActionLine(match, result, isSHA), true, nil
}

func workflowActionCurrentVersion(match workflowActionRefMatch) (string, bool) {
	isSHA := IsCommitSHA(match.ref)
	if !isSHA {
		return match.ref, false
	}
	if match.comment == "" {
		return "", true
	}
	commentVersion := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(match.comment), "#"))
	return commentVersion, true
}

func resolveCachedWorkflowActionRelease(ctx context.Context, deps actionUpdateDeps, repo, currentVersion string, allowMajor, verbose bool, cache map[string]latestReleaseResult) (latestReleaseResult, bool) {
	cacheKey := repo + "|" + currentVersion
	if result, ok := cache[cacheKey]; ok {
		return result, true
	}
	latestVersion, latestSHA, err := deps.getLatestRelease(ctx, repo, currentVersion, allowMajor, verbose)
	if err != nil {
		updateLog.Printf("Failed to get latest release for %s: %v", repo, err)
		return latestReleaseResult{}, false
	}
	result := latestReleaseResult{version: latestVersion, sha: latestSHA}
	cache[cacheKey] = result
	return result, true
}

func shouldSkipWorkflowActionRef(match workflowActionRefMatch, result latestReleaseResult, isSHA bool) bool {
	if isSHA {
		return result.sha == match.ref
	}
	if result.version == match.ref {
		return true
	}
	currentVer := parseVersion(match.ref)
	proposedVer := parseVersion(result.version)
	if currentVer != nil && proposedVer != nil && currentVer.IsNewer(proposedVer) {
		updateLog.Printf("Skipping %s in workflow file: proposed version %s is older than current %s (would be a downgrade)", match.repo, result.version, match.ref)
		return true
	}
	return false
}

func shouldSkipWorkflowActionCoolDown(ctx context.Context, coolDownCache map[string]coolDownCheckResult, repo, latestVersion string, coolDown time.Duration, verbose bool) bool {
	if isExemptFromCoolDown(repo) {
		return false
	}
	coolDownKey := repo + "@" + latestVersion
	coolDownResult, ok := coolDownCache[coolDownKey]
	if !ok {
		coolDownResult = checkReleaseCoolDown(ctx, repo, latestVersion, coolDown)
		coolDownCache[coolDownKey] = coolDownResult
	}
	if !coolDownResult.InCoolDown {
		return false
	}
	cooldownLog.Printf("Action ref %s in workflow: %s", repo, coolDownResult.Message)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping update for %s: %s", repo, coolDownResult.Message)))
	}
	return true
}

func buildUpdatedWorkflowActionLine(match workflowActionRefMatch, result latestReleaseResult, isSHA bool) string {
	if isSHA {
		return fmt.Sprintf("%s%s%s@%s  # %s%s", match.indent, match.prefix, match.repo, result.sha, result.version, match.trailing)
	}
	return fmt.Sprintf("%s%s%s@%s%s%s", match.indent, match.prefix, match.repo, result.version, match.comment, match.trailing)
}

func extractWorkflowActionTargetVersion(line string) string {
	if match, ok := parseWorkflowActionRefMatch(line); ok {
		if version, isSHA := workflowActionCurrentVersion(match); isSHA {
			return version
		}
		return match.ref
	}
	return ""
}
