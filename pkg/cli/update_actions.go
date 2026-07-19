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

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking for GitHub Actions updates..."))
	}

	actionCache, ok, err := updateActionsLoadCache(verbose)
	if err != nil || !ok {
		return err
	}

	updateLog.Printf("Loaded %d action entries from actions-lock.json", len(actionCache.Entries))

	result := &updateActionsResult{}
	for _, s := range updateActionsSnapshot(actionCache) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		updateActionsProcessEntry(updateActionsProcessEntryParams{
			Ctx:                ctx,
			Deps:               deps,
			ActionCache:        actionCache,
			Snapshot:           s,
			Result:             result,
			AllowMajor:         allowMajor,
			Verbose:            verbose,
			DisableReleaseBump: disableReleaseBump,
			CoolDown:           coolDown,
		})
	}

	updateActionsShowSummary(result, verbose)
	return updateActionsSaveCache(actionCache, result.updatedActions)
}

type updateActionsEntrySnapshot struct {
	key   string
	entry workflow.ActionCacheEntry
}

type updateActionsResult struct {
	updatedActions []string
	failedActions  []actionUpdateFailure
	skippedActions []string
}

type updateActionsProcessEntryParams struct {
	Ctx                context.Context
	Deps               actionUpdateDeps
	ActionCache        *workflow.ActionCache
	Snapshot           updateActionsEntrySnapshot
	Result             *updateActionsResult
	AllowMajor         bool
	Verbose            bool
	DisableReleaseBump bool
	CoolDown           time.Duration
}

func updateActionsLoadCache(verbose bool) (*workflow.ActionCache, bool, error) {
	// Load the action cache (actions-lock.json) using the shared ActionCache helpers
	// so that cached inputs/descriptions for safe-outputs.actions entries are preserved.
	actionsLockPath := filepath.Join(".github", "aw", "actions-lock.json")
	if _, err := os.Stat(actionsLockPath); os.IsNotExist(err) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Actions lock file not found: "+actionsLockPath))
		}
		return nil, false, nil // Not an error, just skip
	}

	actionCache := workflow.NewActionCache(".")
	if err := actionCache.Load(); err != nil {
		return nil, false, fmt.Errorf("failed to parse actions lock file: %w", err)
	}
	return actionCache, true, nil
}

func updateActionsSnapshot(actionCache *workflow.ActionCache) []updateActionsEntrySnapshot {
	snapshot := make([]updateActionsEntrySnapshot, 0, len(actionCache.Entries))
	for key, entry := range actionCache.Entries {
		snapshot = append(snapshot, updateActionsEntrySnapshot{key: key, entry: entry})
	}
	return snapshot
}

func updateActionsProcessEntry(p updateActionsProcessEntryParams) {
	entry := p.Snapshot.entry
	updateLog.Printf("Checking action: %s@%s", entry.Repo, entry.Version)
	effectiveAllowMajor := !p.DisableReleaseBump || p.AllowMajor || isCoreAction(entry.Repo)
	latestVersion, latestSHA, err := p.Deps.getLatestRelease(p.Ctx, entry.Repo, entry.Version, effectiveAllowMajor, p.Verbose)
	if err != nil {
		if p.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check %s: %v", entry.Repo, err)))
		}
		p.Result.failedActions = append(p.Result.failedActions, actionUpdateFailure{name: entry.Repo, err: err.Error()})
		return
	}

	latestVersion, latestSHA, ok := updateActionsCapNative(p.Ctx, p.Deps, entry, latestVersion, latestSHA, p.Result, p.Verbose)
	if !ok || updateActionsSkipOlder(entry, latestVersion, p.Result) || updateActionsSkipUnchanged(entry, latestVersion, latestSHA, p.Result, p.Verbose) {
		return
	}
	if updateActionsInCoolDown(p.Ctx, p.ActionCache, entry.Repo, latestVersion, p.CoolDown, p.Result) {
		return
	}
	updateActionsApply(p.ActionCache, p.Snapshot, latestVersion, latestSHA, p.Result)
}

func updateActionsCapNative(ctx context.Context, deps actionUpdateDeps, entry workflow.ActionCacheEntry, latestVersion, latestSHA string, result *updateActionsResult, verbose bool) (string, string, bool) {
	if !isGhAwNativeAction(entry.Repo) {
		return latestVersion, latestSHA, true
	}
	cliVersion := GetVersion()
	cliVer := parseVersion(cliVersion)
	latestVer := parseVersion(latestVersion)
	if cliVer == nil || latestVer == nil || !latestVer.IsNewer(cliVer) {
		return latestVersion, latestSHA, true
	}

	cappedVersion := semverutil.EnsureVPrefix(cliVersion)
	updateLog.Printf("Capping %s update to CLI version %s (latest available %s exceeds running CLI)", entry.Repo, cappedVersion, latestVersion)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s: capping update target to CLI version %s (latest %s is newer than running CLI)", entry.Repo, cappedVersion, latestVersion)))
	}
	cappedSHA, shaErr := deps.getActionSHAForTag(ctx, gitutil.ExtractBaseRepo(entry.Repo), cappedVersion)
	if shaErr != nil {
		updateLog.Printf("Cannot resolve SHA for %s@%s (CLI version cap): %v; skipping update", entry.Repo, cappedVersion, shaErr)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: cannot resolve SHA for CLI version %s: %v", entry.Repo, cappedVersion, shaErr)))
		result.failedActions = append(result.failedActions, actionUpdateFailure{
			name: entry.Repo,
			err:  fmt.Sprintf("cannot resolve SHA for CLI version %s: %v", cappedVersion, shaErr),
		})
		return latestVersion, latestSHA, false
	}
	return cappedVersion, cappedSHA, true
}

func updateActionsSkipOlder(entry workflow.ActionCacheEntry, latestVersion string, result *updateActionsResult) bool {
	currentVer := parseVersion(entry.Version)
	latestVer := parseVersion(latestVersion)
	if currentVer == nil || latestVer == nil || !currentVer.IsNewer(latestVer) {
		return false
	}
	updateLog.Printf("Skipping %s: proposed version %s is older than current %s (would be a downgrade)", entry.Repo, latestVersion, entry.Version)
	msg := fmt.Sprintf("%s: skipping proposed update from %s to %s (would be a downgrade)", entry.Repo, entry.Version, latestVersion)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
	result.skippedActions = append(result.skippedActions, entry.Repo)
	return true
}

func updateActionsSkipUnchanged(entry workflow.ActionCacheEntry, latestVersion, latestSHA string, result *updateActionsResult, verbose bool) bool {
	if latestVersion != entry.Version || latestSHA != entry.SHA {
		return false
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s@%s is up to date", entry.Repo, entry.Version)))
	}
	result.skippedActions = append(result.skippedActions, entry.Repo)
	return true
}

func updateActionsInCoolDown(ctx context.Context, actionCache *workflow.ActionCache, repo, latestVersion string, coolDown time.Duration, result *updateActionsResult) bool {
	if isExemptFromCoolDown(repo) {
		return false
	}
	var coolDownResult coolDownCheckResult
	if cachedDate, ok := actionCache.GetReleasedAt(repo, latestVersion); ok {
		coolDownResult = checkReleaseCoolDownWithDate(repo, latestVersion, cachedDate, coolDown)
	} else {
		coolDownResult = checkReleaseCoolDown(ctx, repo, latestVersion, coolDown)
		if !coolDownResult.PublishedAt.IsZero() {
			actionCache.SetReleasedAt(repo, latestVersion, coolDownResult.PublishedAt)
		}
	}
	if coolDownResult.InCoolDown {
		cooldownLog.Printf("Action %s: %s", repo, coolDownResult.Message)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping update for %s: %s", repo, coolDownResult.Message)))
		result.skippedActions = append(result.skippedActions, repo)
	}
	return coolDownResult.InCoolDown
}

func updateActionsApply(actionCache *workflow.ActionCache, s updateActionsEntrySnapshot, latestVersion, latestSHA string, result *updateActionsResult) {
	entry := s.entry
	oldSHAStr := updateActionsShortSHA(entry.SHA)
	newSHAStr := updateActionsShortSHA(latestSHA)
	updateLog.Printf("Updating %s from %s (%s) to %s (%s)", entry.Repo, entry.Version, oldSHAStr, latestVersion, newSHAStr)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", entry.Repo, entry.Version, latestVersion)))

	if latestVersion != entry.Version {
		actionCache.DeleteByKey(s.key)
	}
	actionCache.Set(entry.Repo, latestVersion, latestSHA)
	result.updatedActions = append(result.updatedActions, entry.Repo)
}

func updateActionsShortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func updateActionsShowSummary(result *updateActionsResult, verbose bool) {
	fmt.Fprintln(os.Stderr, "")
	if len(result.updatedActions) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %d action(s):", len(result.updatedActions))))
		for _, action := range result.updatedActions {
			fmt.Fprintln(os.Stderr, console.FormatListItem(action))
		}
		fmt.Fprintln(os.Stderr, "")
	}
	if len(result.skippedActions) > 0 && verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("%d action(s) already up to date", len(result.skippedActions))))
		fmt.Fprintln(os.Stderr, "")
	}
	if len(result.failedActions) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check %d action(s):", len(result.failedActions))))
		for _, f := range result.failedActions {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", f.name, f.err)
		}
		fmt.Fprintln(os.Stderr, "")
	}
}

func updateActionsSaveCache(actionCache *workflow.ActionCache, updatedActions []string) error {
	if len(updatedActions) == 0 {
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

	// Extract base repository (e.g., "actions/cache/restore" -> "actions/cache")
	baseRepo := gitutil.ExtractBaseRepo(repo)
	updateLog.Printf("Using base repository: %s for action: %s", baseRepo, repo)

	releases, err := getLatestActionReleaseWithDepsFetch(ctx, deps, repo, baseRepo, currentVersion, allowMajor, verbose)
	if err != nil {
		return "", "", err
	}
	if len(releases) == 1 && strings.Contains(releases[0], "\x00") {
		latestRelease, latestSHA, _ := strings.Cut(releases[0], "\x00")
		return latestRelease, latestSHA, nil
	}

	// Parse current version
	currentVer := parseVersion(currentVersion)
	validReleases, err := getLatestActionReleaseWithDepsValidReleases(releases)
	if err != nil {
		return "", "", err
	}

	latestCompatible, err := getLatestActionReleaseWithDepsSelect(currentVer, validReleases, allowMajor)
	if err != nil {
		return "", "", err
	}

	sha, err := deps.getActionSHAForTag(ctx, baseRepo, latestCompatible)
	if err != nil {
		return "", "", fmt.Errorf("failed to get SHA for %s: %w", latestCompatible, err)
	}
	return latestCompatible, sha, nil
}

type actionReleaseWithVersion struct {
	tag     string
	version *semverutil.SemanticVersion
}

func getLatestActionReleaseWithDepsFetch(ctx context.Context, deps actionUpdateDeps, repo, baseRepo, currentVersion string, allowMajor, verbose bool) ([]string, error) {
	output, err := deps.runGHReleasesAPI(ctx, baseRepo)
	if err != nil {
		outputStr := string(output)
		if gitutil.IsAuthError(outputStr) || gitutil.IsAuthError(err.Error()) {
			updateLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s", repo)
			latestRelease, latestSHA, gitErr := deps.getLatestReleaseViaGit(ctx, repo, currentVersion, allowMajor, verbose)
			if gitErr != nil {
				return nil, fmt.Errorf("failed to fetch releases via GitHub API and git: API error: %w, Git Error: %w", err, gitErr)
			}
			return []string{latestRelease + "\x00" + latestSHA}, nil
		}
		if trimmed := strings.TrimSpace(outputStr); trimmed != "" {
			return nil, fmt.Errorf("failed to fetch releases: %w: %s", err, trimmed)
		}
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) != 0 && releases[0] != "" {
		return releases, nil
	}
	return getLatestActionReleaseWithDepsFallback(ctx, deps, repo, baseRepo, currentVersion, allowMajor, verbose)
}

func getLatestActionReleaseWithDepsFallback(ctx context.Context, deps actionUpdateDeps, repo, baseRepo, currentVersion string, allowMajor, verbose bool) ([]string, error) {
	updateLog.Printf("No releases found via GitHub API for %s, falling back to git ls-remote tag scan", baseRepo)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(baseRepo+": no GitHub Releases found, falling back to tag scanning (safe to ignore)"))
	}
	latestRelease, latestSHA, gitErr := deps.getLatestReleaseViaGit(ctx, repo, currentVersion, allowMajor, verbose)
	if gitErr != nil {
		return nil, fmt.Errorf("no releases or tags found for %s: %w", baseRepo, gitErr)
	}
	return []string{latestRelease + "\x00" + latestSHA}, nil
}

func getLatestActionReleaseWithDepsValidReleases(releases []string) ([]actionReleaseWithVersion, error) {
	var validReleases []actionReleaseWithVersion
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer != nil && releaseVer.Pre == "" {
			validReleases = append(validReleases, actionReleaseWithVersion{tag: release, version: releaseVer})
		}
	}
	if len(validReleases) == 0 {
		return nil, errors.New("no valid semantic version releases found")
	}
	getLatestActionReleaseWithDepsSort(validReleases)
	return validReleases, nil
}

func getLatestActionReleaseWithDepsSort(validReleases []actionReleaseWithVersion) {
	slices.SortFunc(validReleases, func(a, b actionReleaseWithVersion) int {
		switch {
		case a.version.IsNewer(b.version):
			return -1
		case b.version.IsNewer(a.version):
			return 1
		default:
			return 0
		}
	})
}

func getLatestActionReleaseWithDepsSelect(currentVer *semverutil.SemanticVersion, validReleases []actionReleaseWithVersion, allowMajor bool) (string, error) {
	if currentVer == nil {
		return validReleases[0].tag, nil
	}
	var latestCompatible string
	var latestCompatibleVersion *semverutil.SemanticVersion
	for _, rel := range validReleases {
		if !allowMajor && rel.version.Major != currentVer.Major {
			continue
		}
		if latestCompatibleVersion == nil || rel.version.IsNewer(latestCompatibleVersion) {
			latestCompatible = rel.tag
			latestCompatibleVersion = rel.version
		} else if getLatestActionReleaseWithDepsEqualVersion(rel.version, latestCompatibleVersion) && !rel.version.IsPreciseVersion() && latestCompatibleVersion.IsPreciseVersion() {
			latestCompatible = rel.tag
			latestCompatibleVersion = rel.version
		}
	}
	if latestCompatible == "" {
		return "", errors.New("no compatible release found")
	}
	return latestCompatible, nil
}

func getLatestActionReleaseWithDepsEqualVersion(a, b *semverutil.SemanticVersion) bool {
	return !a.IsNewer(b) && a.Major == b.Major && a.Minor == b.Minor && a.Patch == b.Patch
}

// getLatestActionReleaseViaGit gets the latest release using git ls-remote (fallback)
func getLatestActionReleaseViaGit(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest release for %s via git ls-remote (current: %s, allow major: %v)", repo, currentVersion, allowMajor)))
	}

	// Extract base repository (e.g., "actions/cache/restore" -> "actions/cache")
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

	releases, tagToSHA, err := getLatestActionReleaseViaGitParseTags(output)
	if err != nil {
		return "", "", err
	}
	validReleases, err := getLatestActionReleaseViaGitValidReleases(releases)
	if err != nil {
		return "", "", err
	}
	currentVer := parseVersion(currentVersion)

	// If current version is not valid, return the highest semver release
	if currentVer == nil {
		latestRelease := validReleases[0].tag
		sha := tagToSHA[latestRelease]
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current version is not valid, using highest semver release: %s (via git)", latestRelease)))
		}
		return latestRelease, sha, nil
	}

	latestCompatible, err := getLatestActionReleaseViaGitSelect(currentVer, validReleases, allowMajor)
	if err != nil {
		return "", "", err
	}
	sha := tagToSHA[latestCompatible]
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest compatible release: %s (via git)", latestCompatible)))
	}

	return latestCompatible, sha, nil
}

func getLatestActionReleaseViaGitParseTags(output []byte) ([]string, map[string]string, error) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var releases []string
	tagToSHA := make(map[string]string)
	for _, line := range lines {
		// Parse: "<sha> refs/tags/<tag>"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		sha := parts[0]
		tagRef := parts[1]
		// Skip ^{} annotations (they point to the commit object)
		if strings.HasSuffix(tagRef, "^{}") {
			continue
		}
		tag := strings.TrimPrefix(tagRef, "refs/tags/")
		releases = append(releases, tag)
		tagToSHA[tag] = sha
	}
	if len(releases) == 0 {
		return nil, nil, errors.New("no releases found")
	}
	return releases, tagToSHA, nil
}

func getLatestActionReleaseViaGitValidReleases(releases []string) ([]actionReleaseWithVersion, error) {
	var validReleases []actionReleaseWithVersion
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer != nil && releaseVer.Pre == "" {
			validReleases = append(validReleases, actionReleaseWithVersion{tag: release, version: releaseVer})
		}
	}
	if len(validReleases) == 0 {
		return nil, errors.New("no valid semantic version releases found")
	}
	getLatestActionReleaseWithDepsSort(validReleases)
	return validReleases, nil
}

func getLatestActionReleaseViaGitSelect(currentVer *semverutil.SemanticVersion, validReleases []actionReleaseWithVersion, allowMajor bool) (string, error) {
	latestCompatible, err := getLatestActionReleaseWithDepsSelect(currentVer, validReleases, allowMajor)
	if err != nil {
		return "", err
	}
	return latestCompatible, nil
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
		return updateActionsInWorkflowFilesVisit(updateActionsInWorkflowFilesVisitParams{
			Ctx:           ctx,
			Deps:          deps,
			Opts:          opts,
			Cache:         cache,
			CoolDownCache: coolDownCache,
			UpdatedFiles:  &updatedFiles,
			Path:          path,
			DirEntry:      d,
			WalkErr:       walkErr,
		})
	})
	if err != nil {
		return fmt.Errorf("failed to walk workflows directory: %w", err)
	}

	if len(updatedFiles) == 0 && opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No action references needed updating in workflow files"))
	}

	return nil
}

type updateActionsInWorkflowFilesVisitParams struct {
	Ctx           context.Context
	Deps          actionUpdateDeps
	Opts          updateActionsOptions
	Cache         map[string]latestReleaseResult
	CoolDownCache map[string]coolDownCheckResult
	UpdatedFiles  *[]string
	Path          string
	DirEntry      os.DirEntry
	WalkErr       error
}

func updateActionsInWorkflowFilesVisit(p updateActionsInWorkflowFilesVisitParams) error {
	if p.WalkErr != nil {
		return p.WalkErr
	}
	if p.Ctx.Err() != nil {
		return p.Ctx.Err()
	}
	if p.DirEntry.IsDir() || !strings.HasSuffix(p.DirEntry.Name(), ".md") {
		return nil
	}

	newContent, updated, err := updateActionsInWorkflowFilesUpdateContent(p.Ctx, p.Deps, p.Opts, p.Cache, p.CoolDownCache, p.Path)
	if err != nil || !updated {
		return err
	}

	if err := os.WriteFile(p.Path, []byte(newContent), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write updated workflow %s: %w", p.Path, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated action/skill references in "+p.DirEntry.Name()))
	*p.UpdatedFiles = append(*p.UpdatedFiles, p.Path)
	updateActionsInWorkflowFilesCompile(p.Ctx, p.Opts, p.Path)
	return nil
}

func updateActionsInWorkflowFilesUpdateContent(ctx context.Context, deps actionUpdateDeps, opts updateActionsOptions, cache map[string]latestReleaseResult, coolDownCache map[string]coolDownCheckResult, path string) (string, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", path, err)))
		}
		return "", false, nil
	}

	updatedActions, newContent, err := updateActionRefsInContentWithDeps(ctx, deps, string(content), cache, coolDownCache, !opts.disableReleaseBump, opts.verbose, opts.coolDown)
	if err != nil {
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update action refs in %s: %v", path, err)))
		}
		return "", false, nil
	}
	updatedSkills, newContent, err := updateSkillRefsInContent(ctx, newContent, !opts.disableReleaseBump, opts.verbose, opts.coolDown)
	if err != nil {
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update skill refs in %s: %v", path, err)))
		}
		return "", false, nil
	}
	return newContent, updatedActions || updatedSkills, nil
}

func updateActionsInWorkflowFilesCompile(ctx context.Context, opts updateActionsOptions, path string) {
	// Recompile the updated workflow (unless --no-compile is set)
	if opts.noCompile {
		return
	}
	if err := compileWorkflowWithRefresh(ctx, path, opts.verbose, false, opts.engineOverride, false, opts.approve); err != nil {
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to recompile %s: %v", path, err)))
		}
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

	for i, line := range lines {
		newLine, lineChanged := updateActionRefsInContentWithDepsLine(updateActionRefsInContentWithDepsLineParams{
			Ctx:           ctx,
			Deps:          deps,
			Line:          line,
			Index:         i,
			Cache:         cache,
			CoolDownCache: coolDownCache,
			AllowMajor:    allowMajor,
			Verbose:       verbose,
			CoolDown:      coolDown,
		})
		if lineChanged {
			lines[i] = newLine
			changed = true
		}
	}

	return changed, strings.Join(lines, "\n"), nil
}

type updateActionRefsInContentWithDepsMatch struct {
	prefix   string
	repo     string
	ref      string
	comment  string
	trailing string
	start    string
}

type updateActionRefsInContentWithDepsLineParams struct {
	Ctx           context.Context
	Deps          actionUpdateDeps
	Line          string
	Index         int
	Cache         map[string]latestReleaseResult
	CoolDownCache map[string]coolDownCheckResult
	AllowMajor    bool
	Verbose       bool
	CoolDown      time.Duration
}

func updateActionRefsInContentWithDepsLine(p updateActionRefsInContentWithDepsLineParams) (string, bool) {
	match := actionRefPattern.FindStringSubmatchIndex(p.Line)
	if match == nil {
		return p.Line, false
	}
	refMatch := updateActionRefsInContentWithDepsParseMatch(p.Line, match)
	effectiveAllowMajor := p.AllowMajor || isCoreAction(refMatch.repo)
	if !effectiveAllowMajor {
		return p.Line, false
	}

	isSHA := IsCommitSHA(refMatch.ref)
	currentVersion := updateActionRefsInContentWithDepsCurrentVersion(refMatch.ref, refMatch.comment, isSHA)
	result, ok := updateActionRefsInContentWithDepsResolve(p.Ctx, p.Deps, refMatch.repo, currentVersion, effectiveAllowMajor, p.Verbose, p.Cache)
	if !ok || updateActionRefsInContentWithDepsUnchangedOrDowngrade(refMatch, result, isSHA) {
		return p.Line, false
	}
	if updateActionRefsInContentWithDepsInCooldown(p.Ctx, refMatch.repo, result.version, p.CoolDownCache, p.Verbose, p.CoolDown) {
		return p.Line, false
	}

	updateLog.Printf("Updating %s from %s to %s in line %d", refMatch.repo, refMatch.ref, result.version, p.Index+1)
	return updateActionRefsInContentWithDepsBuildLine(refMatch, result, isSHA), true
}

func updateActionRefsInContentWithDepsParseMatch(line string, match []int) updateActionRefsInContentWithDepsMatch {
	refMatch := updateActionRefsInContentWithDepsMatch{
		start:  line[:match[2]],
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
	return refMatch
}

func updateActionRefsInContentWithDepsCurrentVersion(ref, comment string, isSHA bool) string {
	if !isSHA {
		return ref
	}
	if comment == "" {
		return ""
	}
	commentVersion := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(comment), "#"))
	return commentVersion
}

func updateActionRefsInContentWithDepsResolve(ctx context.Context, deps actionUpdateDeps, repo, currentVersion string, allowMajor, verbose bool, cache map[string]latestReleaseResult) (latestReleaseResult, bool) {
	cacheKey := repo + "|" + currentVersion
	result, cached := cache[cacheKey]
	if cached {
		return result, true
	}
	latestVersion, latestSHA, err := deps.getLatestRelease(ctx, repo, currentVersion, allowMajor, verbose)
	if err != nil {
		updateLog.Printf("Failed to get latest release for %s: %v", repo, err)
		return latestReleaseResult{}, false
	}
	result = latestReleaseResult{version: latestVersion, sha: latestSHA}
	cache[cacheKey] = result
	return result, true
}

func updateActionRefsInContentWithDepsUnchangedOrDowngrade(refMatch updateActionRefsInContentWithDepsMatch, result latestReleaseResult, isSHA bool) bool {
	if isSHA {
		return result.sha == refMatch.ref
	}
	if result.version == refMatch.ref {
		return true
	}
	currentVer := parseVersion(refMatch.ref)
	proposedVer := parseVersion(result.version)
	if currentVer != nil && proposedVer != nil && currentVer.IsNewer(proposedVer) {
		updateLog.Printf("Skipping %s in workflow file: proposed version %s is older than current %s (would be a downgrade)", refMatch.repo, result.version, refMatch.ref)
		return true
	}
	return false
}

func updateActionRefsInContentWithDepsInCooldown(ctx context.Context, repo, latestVersion string, coolDownCache map[string]coolDownCheckResult, verbose bool, coolDown time.Duration) bool {
	if isExemptFromCoolDown(repo) {
		return false
	}
	coolDownKey := repo + "@" + latestVersion
	coolDownResult, coolDownCached := coolDownCache[coolDownKey]
	if !coolDownCached {
		coolDownResult = checkReleaseCoolDown(ctx, repo, latestVersion, coolDown)
		coolDownCache[coolDownKey] = coolDownResult
	}
	if coolDownResult.InCoolDown {
		cooldownLog.Printf("Action ref %s in workflow: %s", repo, coolDownResult.Message)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping update for %s: %s", repo, coolDownResult.Message)))
		}
	}
	return coolDownResult.InCoolDown
}

func updateActionRefsInContentWithDepsBuildLine(refMatch updateActionRefsInContentWithDepsMatch, result latestReleaseResult, isSHA bool) string {
	if isSHA {
		// SHA-pinned references stay SHA-pinned, updated to latest SHA + version comment
		return fmt.Sprintf("%s%s%s@%s  # %s%s", refMatch.start, refMatch.prefix, refMatch.repo, result.sha, result.version, refMatch.trailing)
	}
	// Version tag references just get the new version tag
	return fmt.Sprintf("%s%s%s@%s%s%s", refMatch.start, refMatch.prefix, refMatch.repo, result.version, refMatch.comment, refMatch.trailing)
}
