package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/semverutil"
	"github.com/github/gh-aw/pkg/workflow"
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

// UpdateActions updates GitHub Actions versions in .github/aw/actions-lock.json
// It checks each action for newer releases and updates the SHA if a newer version is found.
// By default all actions are updated to the latest major version; pass disableReleaseBump=true
// to revert to the old behaviour where only core (actions/*) actions bypass the --major flag.
//
// The ActionCache helpers from pkg/workflow are used so that cached inputs and descriptions
// for safe-outputs.actions entries are preserved when their SHA is unchanged, and cleared
// when the SHA changes (prompting a re-fetch on the next compile).
func UpdateActions(ctx context.Context, allowMajor, verbose, disableReleaseBump bool) error {
	updateLog.Print("Starting action updates")

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking for GitHub Actions updates..."))
	}

	// Load the action cache (actions-lock.json) using the shared ActionCache helpers
	// so that cached inputs/descriptions for safe-outputs.actions entries are preserved.
	actionsLockPath := filepath.Join(".github", "aw", "actions-lock.json")
	if _, err := os.Stat(actionsLockPath); os.IsNotExist(err) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Actions lock file not found: "+actionsLockPath))
		}
		return nil // Not an error, just skip
	}

	actionCache := workflow.NewActionCache(".")
	if err := actionCache.Load(); err != nil {
		return fmt.Errorf("failed to parse actions lock file: %w", err)
	}

	updateLog.Printf("Loaded %d action entries from actions-lock.json", len(actionCache.Entries))

	// Track updates
	var updatedActions []string
	var failedActions []actionUpdateFailure
	var skippedActions []string

	// Snapshot entries before iteration to avoid mutating the map mid-loop.
	type entrySnapshot struct {
		key   string
		entry workflow.ActionCacheEntry
	}
	snapshot := make([]entrySnapshot, 0, len(actionCache.Entries))
	for key, entry := range actionCache.Entries {
		snapshot = append(snapshot, entrySnapshot{key: key, entry: entry})
	}

	for _, s := range snapshot {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		entry := s.entry
		updateLog.Printf("Checking action: %s@%s", entry.Repo, entry.Version)

		// By default all actions are force-updated to the latest major version.
		// When disableReleaseBump is set, only core actions (actions/*) bypass the --major flag.
		effectiveAllowMajor := !disableReleaseBump || allowMajor || isCoreAction(entry.Repo)

		// Check for latest release using the injectable function (also used by updateActionRefsInContent)
		latestVersion, latestSHA, err := getLatestActionReleaseFn(ctx, entry.Repo, entry.Version, effectiveAllowMajor, verbose)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check %s: %v", entry.Repo, err)))
			}
			failedActions = append(failedActions, actionUpdateFailure{name: entry.Repo, err: err.Error()})
			continue
		}

		// For gh-aw native actions (github/gh-aw/* and github/gh-aw-actions/*), the action
		// versions are published in lock-step with the CLI. Never update these actions beyond
		// the version of the currently running CLI — doing so would pin a newer (possibly
		// pre-release) action that may be incompatible with the user's installed CLI.
		if isGhAwNativeAction(entry.Repo) {
			cliVersion := GetVersion()
			cliVer := parseVersion(cliVersion)
			latestVer := parseVersion(latestVersion)
			if cliVer != nil && latestVer != nil && latestVer.IsNewer(cliVer) {
				cappedVersion := semverutil.EnsureVPrefix(cliVersion)
				updateLog.Printf("Capping %s update to CLI version %s (latest available %s exceeds running CLI)", entry.Repo, cappedVersion, latestVersion)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s: capping update target to CLI version %s (latest %s is newer than running CLI)", entry.Repo, cappedVersion, latestVersion)))
				}
				cappedSHA, shaErr := getActionSHAForTagFn(ctx, gitutil.ExtractBaseRepo(entry.Repo), cappedVersion)
				if shaErr != nil {
					updateLog.Printf("Cannot resolve SHA for %s@%s (CLI version cap): %v; skipping update", entry.Repo, cappedVersion, shaErr)
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: cannot resolve SHA for CLI version %s: %v", entry.Repo, cappedVersion, shaErr)))
					failedActions = append(failedActions, actionUpdateFailure{
						name: entry.Repo,
						err:  fmt.Sprintf("cannot resolve SHA for CLI version %s: %v", cappedVersion, shaErr),
					})
					continue
				}
				latestVersion = cappedVersion
				latestSHA = cappedSHA
			}
		}

		// Check if update is available
		if latestVersion == entry.Version && latestSHA == entry.SHA {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s@%s is up to date", entry.Repo, entry.Version)))
			}
			skippedActions = append(skippedActions, entry.Repo)
			continue
		}

		// Update the entry using ActionCache.Set which:
		// - Preserves cached inputs/descriptions when the SHA is unchanged (moving tag)
		// - Clears cached inputs/descriptions when the SHA changes, prompting a re-fetch
		//   of the updated action.yml on the next compile
		oldSHAStr := entry.SHA
		if len(oldSHAStr) > 7 {
			oldSHAStr = oldSHAStr[:7]
		}
		newSHAStr := latestSHA
		if len(newSHAStr) > 7 {
			newSHAStr = newSHAStr[:7]
		}
		updateLog.Printf("Updating %s from %s (%s) to %s (%s)", entry.Repo, entry.Version, oldSHAStr, latestVersion, newSHAStr)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", entry.Repo, entry.Version, latestVersion)))

		// Remove the old key when the version changes, using the original map key from
		// the snapshot to handle any key/version mismatches in the stored cache file.
		if latestVersion != entry.Version {
			actionCache.DeleteByKey(s.key)
		}
		// Set the new entry; ActionCache.Set handles inputs/description preservation.
		actionCache.Set(entry.Repo, latestVersion, latestSHA)

		updatedActions = append(updatedActions, entry.Repo)
	}

	// Show summary
	fmt.Fprintln(os.Stderr, "")

	if len(updatedActions) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %d action(s):", len(updatedActions))))
		for _, action := range updatedActions {
			fmt.Fprintln(os.Stderr, console.FormatListItem(action))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	if len(skippedActions) > 0 && verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("%d action(s) already up to date", len(skippedActions))))
		fmt.Fprintln(os.Stderr, "")
	}

	if len(failedActions) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check %d action(s):", len(failedActions))))
		for _, f := range failedActions {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", f.name, f.err)
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Save the updated actions lock file using ActionCache.Save which preserves
	// all entry fields (including inputs/descriptions for safe-outputs actions).
	if len(updatedActions) > 0 {
		if err := actionCache.Save(); err != nil {
			return fmt.Errorf("failed to save actions lock file: %w", err)
		}

		updateLog.Printf("Successfully wrote updated actions-lock.json with %d updates", len(updatedActions))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updated actions-lock.json file"))
	}

	return nil
}

// getLatestActionRelease gets the latest release for an action repository
// It respects semantic versioning and the allowMajor flag
func getLatestActionRelease(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	updateLog.Printf("Getting latest release for %s@%s (allowMajor=%v)", repo, currentVersion, allowMajor)

	// Extract base repository (e.g., "actions/cache/restore" -> "actions/cache")
	baseRepo := gitutil.ExtractBaseRepo(repo)
	updateLog.Printf("Using base repository: %s for action: %s", baseRepo, repo)

	// Use gh CLI to get releases
	output, err := runGHReleasesAPIFn(ctx, baseRepo)
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if gitutil.IsAuthError(outputStr) || gitutil.IsAuthError(err.Error()) {
			updateLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s", repo)
			// Try fallback using git ls-remote
			latestRelease, latestSHA, gitErr := getLatestActionReleaseViaGitFn(ctx, repo, currentVersion, allowMajor, verbose)
			if gitErr != nil {
				return "", "", fmt.Errorf("failed to fetch releases via GitHub API and git: API error: %w, Git Error: %w", err, gitErr)
			}
			return latestRelease, latestSHA, nil
		}
		// Include the gh output in the error for better diagnostics
		if trimmed := strings.TrimSpace(outputStr); trimmed != "" {
			return "", "", fmt.Errorf("failed to fetch releases: %w: %s", err, trimmed)
		}
		return "", "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) == 0 || releases[0] == "" {
		// No GitHub Releases found; fall back to tag scanning via git ls-remote.
		// Some repositories publish tags without creating GitHub Releases — this is safe
		// to use and the warning below is informational only.
		updateLog.Printf("No releases found via GitHub API for %s, falling back to git ls-remote tag scan", baseRepo)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(baseRepo+": no GitHub Releases found, falling back to tag scanning (safe to ignore)"))
		}
		latestRelease, latestSHA, gitErr := getLatestActionReleaseViaGitFn(ctx, repo, currentVersion, allowMajor, verbose)
		if gitErr != nil {
			return "", "", fmt.Errorf("no releases or tags found for %s: %w", baseRepo, gitErr)
		}
		return latestRelease, latestSHA, nil
	}

	// Parse current version
	currentVer := parseVersion(currentVersion)

	// Find all valid stable semantic version releases (skip prereleases such as v1.0.0-beta.1).
	// Per semver rules, v1.1.0-beta.1 > v1.0.0, so without this filter a prerelease of a
	// higher base version could be incorrectly selected as the upgrade target.
	type releaseWithVersion struct {
		tag     string
		version *semverutil.SemanticVersion
	}
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

	if len(validReleases) == 0 {
		return "", "", errors.New("no valid semantic version releases found")
	}

	// Sort releases by semver in descending order (highest first)
	sort.Slice(validReleases, func(i, j int) bool {
		return validReleases[i].version.IsNewer(validReleases[j].version)
	})

	// If current version is not valid, return the highest semver release
	if currentVer == nil {
		latestRelease := validReleases[0].tag
		sha, err := getActionSHAForTagFn(ctx, baseRepo, latestRelease)
		if err != nil {
			return "", "", fmt.Errorf("failed to get SHA for %s: %w", latestRelease, err)
		}
		return latestRelease, sha, nil
	}

	// Find the highest compatible release (respecting major version if !allowMajor)
	var latestCompatible string
	var latestCompatibleVersion *semverutil.SemanticVersion

	for _, rel := range validReleases {
		// Check if compatible based on major version
		if !allowMajor && rel.version.Major != currentVer.Major {
			continue
		}

		// Since releases are sorted by semver descending, first match is highest
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

	if latestCompatible == "" {
		return "", "", errors.New("no compatible release found")
	}

	// Get the SHA for the latest compatible release
	sha, err := getActionSHAForTagFn(ctx, baseRepo, latestCompatible)
	if err != nil {
		return "", "", fmt.Errorf("failed to get SHA for %s: %w", latestCompatible, err)
	}

	return latestCompatible, sha, nil
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

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var releases []string
	tagToSHA := make(map[string]string)

	for _, line := range lines {
		// Parse: "<sha> refs/tags/<tag>"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
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
	}

	if len(releases) == 0 {
		return "", "", errors.New("no releases found")
	}

	// Parse current version
	currentVer := parseVersion(currentVersion)

	// Find all valid stable semantic version releases (skip prereleases such as v1.0.0-beta.1).
	// Per semver rules, v1.1.0-beta.1 > v1.0.0, so without this filter a prerelease of a
	// higher base version could be incorrectly selected as the upgrade target.
	// git ls-remote --tags returns every tag, so the prerelease check is especially important
	// for this fallback path.
	type releaseWithVersion struct {
		tag     string
		version *semverutil.SemanticVersion
	}
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

	if len(validReleases) == 0 {
		return "", "", errors.New("no valid semantic version releases found")
	}

	// Sort releases by semver in descending order (highest first)
	sort.Slice(validReleases, func(i, j int) bool {
		return validReleases[i].version.IsNewer(validReleases[j].version)
	})

	// If current version is not valid, return the highest semver release
	if currentVer == nil {
		latestRelease := validReleases[0].tag
		sha := tagToSHA[latestRelease]
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current version is not valid, using highest semver release: %s (via git)", latestRelease)))
		}
		return latestRelease, sha, nil
	}

	// Find the highest compatible release (respecting major version if !allowMajor)
	var latestCompatible string
	var latestCompatibleVersion *semverutil.SemanticVersion

	for _, rel := range validReleases {
		// Check if compatible based on major version
		if !allowMajor && rel.version.Major != currentVer.Major {
			continue
		}

		// Since releases are sorted by semver descending, first match is highest
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

	if latestCompatible == "" {
		return "", "", errors.New("no compatible release found")
	}

	sha := tagToSHA[latestCompatible]
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest compatible release: %s (via git)", latestCompatible)))
	}

	return latestCompatible, sha, nil
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

// getLatestActionReleaseFn is the function used to fetch the latest release for an action.
// It is used by both UpdateActions and updateActionRefsInContent and can be replaced in
// tests to avoid network calls.
var getLatestActionReleaseFn = func(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	return getLatestActionRelease(ctx, repo, currentVersion, allowMajor, verbose)
}

// getLatestActionReleaseViaGitFn is the function used to fetch the latest release via git
// ls-remote as a fallback. It can be replaced in tests to avoid network calls.
var getLatestActionReleaseViaGitFn = func(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	return getLatestActionReleaseViaGit(ctx, repo, currentVersion, allowMajor, verbose)
}

// runGHReleasesAPIFn calls the GitHub Releases API for the given base repository and
// returns the raw output. It can be replaced in tests to avoid network calls.
var runGHReleasesAPIFn = func(ctx context.Context, baseRepo string) ([]byte, error) {
	return workflow.RunGHCombinedContext(ctx, "Fetching releases...", "api", fmt.Sprintf("/repos/%s/releases", baseRepo), "--jq", ".[].tag_name")
}

// getActionSHAForTagFn resolves the commit SHA for a given tag. It can be replaced in
// tests to avoid network calls.
var getActionSHAForTagFn = func(ctx context.Context, repo, tag string) (string, error) {
	return getActionSHAForTag(ctx, repo, tag)
}

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
func UpdateActionsInWorkflowFiles(ctx context.Context, workflowsDir, engineOverride string, verbose, disableReleaseBump bool, noCompile bool) error {
	if workflowsDir == "" {
		workflowsDir = getWorkflowsDir()
	}

	updateLog.Printf("Updating action references in workflow files: dir=%s", workflowsDir)

	// Per-invocation cache: key = "repo@currentVersion", avoids repeated API calls
	cache := make(map[string]latestReleaseResult)

	var updatedFiles []string

	err := filepath.WalkDir(workflowsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", path, err)))
			}
			return nil
		}

		updated, newContent, err := updateActionRefsInContent(ctx, string(content), cache, !disableReleaseBump, verbose)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update action refs in %s: %v", path, err)))
			}
			return nil
		}

		if !updated {
			return nil
		}

		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write updated workflow %s: %w", path, err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated action references in "+d.Name()))
		updatedFiles = append(updatedFiles, path)

		// Recompile the updated workflow (unless --no-compile is set)
		if !noCompile {
			if err := compileWorkflowWithRefresh(path, verbose, false, engineOverride, false); err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to recompile %s: %v", path, err)))
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk workflows directory: %w", err)
	}

	if len(updatedFiles) == 0 && verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No action references needed updating in workflow files"))
	}

	return nil
}

// updateActionRefsInContent replaces outdated "uses: org/repo@version" references
// in content with the latest major version and SHA. Returns (changed, newContent, error).
// cache is keyed by "repo@currentVersion" and avoids redundant API calls across lines/files.
// When allowMajor is true (the default), all matched actions are updated to the latest
// major version. When allowMajor is false (--disable-release-bump), non-core (non
// actions/*) action refs are skipped; core actions are always updated.
func updateActionRefsInContent(ctx context.Context, content string, cache map[string]latestReleaseResult, allowMajor, verbose bool) (bool, string, error) {
	changed := false
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		match := actionRefPattern.FindStringSubmatchIndex(line)
		if match == nil {
			continue
		}

		// Extract matched groups
		prefix := line[match[2]:match[3]] // "uses: "
		repo := line[match[4]:match[5]]   // e.g. "actions/checkout"
		ref := line[match[6]:match[7]]    // SHA or version tag
		comment := ""
		if match[8] >= 0 {
			comment = line[match[8]:match[9]] // e.g. " # v6.0.2"
		}
		trailing := ""
		if match[10] >= 0 {
			trailing = line[match[10]:match[11]]
		}

		// When release bumps are disabled, skip non-core (non actions/*) action refs.
		effectiveAllowMajor := allowMajor || isCoreAction(repo)
		if !effectiveAllowMajor {
			continue
		}

		// Determine the "current version" to pass to getLatestActionReleaseFn
		isSHA := IsCommitSHA(ref)
		currentVersion := ref
		if isSHA {
			// Extract version from comment (e.g., " # v6.0.2" -> "v6.0.2")
			if comment != "" {
				commentVersion := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(comment), "#"))
				if commentVersion != "" {
					currentVersion = commentVersion
				} else {
					currentVersion = ""
				}
			} else {
				currentVersion = ""
			}
		}

		// Resolve latest version/SHA, using the cache to avoid redundant API calls.
		// Use "|" as separator since GitHub repo names cannot contain "|".
		cacheKey := repo + "|" + currentVersion
		result, cached := cache[cacheKey]
		if !cached {
			latestVersion, latestSHA, err := getLatestActionReleaseFn(ctx, repo, currentVersion, effectiveAllowMajor, verbose)
			if err != nil {
				updateLog.Printf("Failed to get latest release for %s: %v", repo, err)
				continue
			}
			result = latestReleaseResult{version: latestVersion, sha: latestSHA}
			cache[cacheKey] = result
		}
		latestVersion := result.version
		latestSHA := result.sha

		if isSHA {
			if latestSHA == ref {
				continue // SHA unchanged
			}
		} else {
			if latestVersion == ref {
				continue // Version tag unchanged
			}
		}

		// Build the new uses line
		var newRef string
		if isSHA {
			// SHA-pinned references stay SHA-pinned, updated to latest SHA + version comment
			newRef = fmt.Sprintf("%s%s%s@%s  # %s%s", line[:match[2]], prefix, repo, latestSHA, latestVersion, trailing)
		} else {
			// Version tag references just get the new version tag
			newRef = fmt.Sprintf("%s%s%s@%s%s%s", line[:match[2]], prefix, repo, latestVersion, comment, trailing)
		}

		updateLog.Printf("Updating %s from %s to %s in line %d", repo, ref, latestVersion, i+1)
		lines[i] = newRef
		changed = true
	}

	return changed, strings.Join(lines, "\n"), nil
}
