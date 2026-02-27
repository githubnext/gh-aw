package cli

import (
	"bytes"
	"encoding/json"
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
	"github.com/github/gh-aw/pkg/workflow"
)

// extractBaseRepo extracts the base repository (owner/repo) from an action path
// that may include subfolders (e.g., "actions/cache/restore" -> "actions/cache")
func extractBaseRepo(actionPath string) string {
	parts := strings.Split(actionPath, "/")
	if len(parts) >= 2 {
		// Return owner/repo (first two segments)
		return parts[0] + "/" + parts[1]
	}
	// If less than 2 parts, return as-is (shouldn't happen in practice)
	return actionPath
}

// isCoreAction returns true if the repo is a GitHub-maintained core action (actions/* org).
// Core actions are always updated to the latest major version without requiring --major.
func isCoreAction(repo string) bool {
	return strings.HasPrefix(repo, "actions/")
}

// UpdateActions updates GitHub Actions versions in .github/aw/actions-lock.json
// It checks each action for newer releases and updates the SHA if a newer version is found
func UpdateActions(allowMajor, verbose bool) error {
	updateLog.Print("Starting action updates")

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking for GitHub Actions updates..."))
	}

	// Get the path to actions-lock.json
	actionsLockPath := filepath.Join(".github", "aw", "actions-lock.json")

	// Check if the file exists
	if _, err := os.Stat(actionsLockPath); os.IsNotExist(err) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Actions lock file not found: "+actionsLockPath))
		}
		return nil // Not an error, just skip
	}

	// Load the current actions lock file
	data, err := os.ReadFile(actionsLockPath)
	if err != nil {
		return fmt.Errorf("failed to read actions lock file: %w", err)
	}

	var actionsLock actionsLockFile
	if err := json.Unmarshal(data, &actionsLock); err != nil {
		return fmt.Errorf("failed to parse actions lock file: %w", err)
	}

	updateLog.Printf("Loaded %d action entries from actions-lock.json", len(actionsLock.Entries))

	// Track updates
	var updatedActions []string
	var failedActions []string
	var skippedActions []string

	// Update each action
	for key, entry := range actionsLock.Entries {
		updateLog.Printf("Checking action: %s@%s", entry.Repo, entry.Version)

		// Core actions (actions/*) always update to the latest major version
		effectiveAllowMajor := allowMajor || isCoreAction(entry.Repo)

		// Check for latest release
		latestVersion, latestSHA, err := getLatestActionRelease(entry.Repo, entry.Version, effectiveAllowMajor, verbose)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check %s: %v", entry.Repo, err)))
			}
			failedActions = append(failedActions, entry.Repo)
			continue
		}

		// Check if update is available
		if latestVersion == entry.Version && latestSHA == entry.SHA {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s@%s is up to date", entry.Repo, entry.Version)))
			}
			skippedActions = append(skippedActions, entry.Repo)
			continue
		}

		// Update the entry
		updateLog.Printf("Updating %s from %s (%s) to %s (%s)", entry.Repo, entry.Version, entry.SHA[:7], latestVersion, latestSHA[:7])
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", entry.Repo, entry.Version, latestVersion)))

		// Delete the old key (which has the old version)
		delete(actionsLock.Entries, key)

		// Create a new key with the new version
		newKey := entry.Repo + "@" + latestVersion
		actionsLock.Entries[newKey] = actionsLockEntry{
			Repo:    entry.Repo,
			Version: latestVersion,
			SHA:     latestSHA,
		}

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
		for _, action := range failedActions {
			fmt.Fprintf(os.Stderr, "  %s\n", action)
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Save the updated actions lock file if there were any updates
	if len(updatedActions) > 0 {
		// Marshal with sorted keys and pretty printing
		updatedData, err := marshalActionsLockSorted(&actionsLock)
		if err != nil {
			return fmt.Errorf("failed to marshal updated actions lock: %w", err)
		}

		// Add trailing newline for prettier compliance
		updatedData = append(updatedData, '\n')

		if err := os.WriteFile(actionsLockPath, updatedData, 0644); err != nil {
			return fmt.Errorf("failed to write updated actions lock file: %w", err)
		}

		updateLog.Printf("Successfully wrote updated actions-lock.json with %d updates", len(updatedActions))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updated actions-lock.json file"))
	}

	return nil
}

// getLatestActionRelease gets the latest release for an action repository
// It respects semantic versioning and the allowMajor flag
func getLatestActionRelease(repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	updateLog.Printf("Getting latest release for %s@%s (allowMajor=%v)", repo, currentVersion, allowMajor)

	// Extract base repository (e.g., "actions/cache/restore" -> "actions/cache")
	baseRepo := extractBaseRepo(repo)
	updateLog.Printf("Using base repository: %s for action: %s", baseRepo, repo)

	// Use gh CLI to get releases
	output, err := workflow.RunGHCombined("Fetching releases...", "api", fmt.Sprintf("/repos/%s/releases", baseRepo), "--jq", ".[].tag_name")
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if gitutil.IsAuthError(outputStr) || gitutil.IsAuthError(err.Error()) {
			updateLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s", repo)
			// Try fallback using git ls-remote
			latestRelease, latestSHA, gitErr := getLatestActionReleaseViaGit(repo, currentVersion, allowMajor, verbose)
			if gitErr != nil {
				return "", "", fmt.Errorf("failed to fetch releases via GitHub API and git: API error: %w, Git Error: %w", err, gitErr)
			}
			return latestRelease, latestSHA, nil
		}
		return "", "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) == 0 || releases[0] == "" {
		return "", "", errors.New("no releases found")
	}

	// Parse current version
	currentVer := parseVersion(currentVersion)

	// Find all valid semantic version releases and sort by semver
	type releaseWithVersion struct {
		tag     string
		version *semanticVersion
	}
	var validReleases []releaseWithVersion
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer != nil {
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
		return validReleases[i].version.isNewer(validReleases[j].version)
	})

	// If current version is not valid, return the highest semver release
	if currentVer == nil {
		latestRelease := validReleases[0].tag
		sha, err := getActionSHAForTag(baseRepo, latestRelease)
		if err != nil {
			return "", "", fmt.Errorf("failed to get SHA for %s: %w", latestRelease, err)
		}
		return latestRelease, sha, nil
	}

	// Find the highest compatible release (respecting major version if !allowMajor)
	var latestCompatible string
	var latestCompatibleVersion *semanticVersion

	for _, rel := range validReleases {
		// Check if compatible based on major version
		if !allowMajor && rel.version.major != currentVer.major {
			continue
		}

		// Since releases are sorted by semver descending, first match is highest
		if latestCompatibleVersion == nil || rel.version.isNewer(latestCompatibleVersion) {
			latestCompatible = rel.tag
			latestCompatibleVersion = rel.version
		} else if !rel.version.isNewer(latestCompatibleVersion) &&
			rel.version.major == latestCompatibleVersion.major &&
			rel.version.minor == latestCompatibleVersion.minor &&
			rel.version.patch == latestCompatibleVersion.patch {
			// If versions are equal, prefer the less precise one (e.g., "v8" over "v8.0.0")
			// This follows GitHub Actions convention of using major version tags
			if !rel.version.isPreciseVersion() && latestCompatibleVersion.isPreciseVersion() {
				latestCompatible = rel.tag
				latestCompatibleVersion = rel.version
			}
		}
	}

	if latestCompatible == "" {
		return "", "", errors.New("no compatible release found")
	}

	// Get the SHA for the latest compatible release
	sha, err := getActionSHAForTag(baseRepo, latestCompatible)
	if err != nil {
		return "", "", fmt.Errorf("failed to get SHA for %s: %w", latestCompatible, err)
	}

	return latestCompatible, sha, nil
}

// getLatestActionReleaseViaGit gets the latest release using git ls-remote (fallback)
func getLatestActionReleaseViaGit(repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest release for %s via git ls-remote (current: %s, allow major: %v)", repo, currentVersion, allowMajor)))
	}

	// Extract base repository (e.g., "actions/cache/restore" -> "actions/cache")
	baseRepo := extractBaseRepo(repo)
	updateLog.Printf("Using base repository: %s for action: %s (git fallback)", baseRepo, repo)

	githubHost := getGitHubHostForRepo(baseRepo)
	repoURL := fmt.Sprintf("%s/%s.git", githubHost, baseRepo)

	// List all tags
	cmd := exec.Command("git", "ls-remote", "--tags", repoURL)
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

	// Find all valid semantic version releases and sort by semver
	type releaseWithVersion struct {
		tag     string
		version *semanticVersion
	}
	var validReleases []releaseWithVersion
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer != nil {
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
		return validReleases[i].version.isNewer(validReleases[j].version)
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
	var latestCompatibleVersion *semanticVersion

	for _, rel := range validReleases {
		// Check if compatible based on major version
		if !allowMajor && rel.version.major != currentVer.major {
			continue
		}

		// Since releases are sorted by semver descending, first match is highest
		if latestCompatibleVersion == nil || rel.version.isNewer(latestCompatibleVersion) {
			latestCompatible = rel.tag
			latestCompatibleVersion = rel.version
		} else if !rel.version.isNewer(latestCompatibleVersion) &&
			rel.version.major == latestCompatibleVersion.major &&
			rel.version.minor == latestCompatibleVersion.minor &&
			rel.version.patch == latestCompatibleVersion.patch {
			// If versions are equal, prefer the less precise one (e.g., "v8" over "v8.0.0")
			// This follows GitHub Actions convention of using major version tags
			if !rel.version.isPreciseVersion() && latestCompatibleVersion.isPreciseVersion() {
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

// getActionSHAForTag gets the commit SHA for a given tag in an action repository
func getActionSHAForTag(repo, tag string) (string, error) {
	updateLog.Printf("Getting SHA for %s@%s", repo, tag)

	// Use gh CLI to get the git ref for the tag
	output, err := workflow.RunGH("Fetching tag info...", "api", fmt.Sprintf("/repos/%s/git/ref/tags/%s", repo, tag), "--jq", ".object.sha")
	if err != nil {
		return "", fmt.Errorf("failed to resolve tag: %w", err)
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", errors.New("empty SHA returned for tag")
	}

	// Validate SHA format (should be 40 hex characters)
	if len(sha) != 40 {
		return "", fmt.Errorf("invalid SHA format: %s", sha)
	}

	return sha, nil
}

// marshalActionsLockSorted marshals the actions lock with entries sorted by key
func marshalActionsLockSorted(actionsLock *actionsLockFile) ([]byte, error) {
	// Extract and sort the keys
	keys := make([]string, 0, len(actionsLock.Entries))
	for key := range actionsLock.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build JSON using json.Marshal for proper encoding
	var buf strings.Builder
	buf.WriteString("{\n  \"entries\": {\n")

	for i, key := range keys {
		entry := actionsLock.Entries[key]

		// Marshal the entry to JSON to ensure proper escaping
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}

		// Marshal the key to ensure proper escaping
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}

		// Write the key-value pair with proper indentation
		buf.WriteString("    ")
		buf.Write(keyJSON)
		buf.WriteString(": ")

		// Pretty-print the entry JSON with proper indentation
		var prettyEntry bytes.Buffer
		if err := json.Indent(&prettyEntry, entryJSON, "    ", "  "); err != nil {
			return nil, err
		}
		buf.WriteString(prettyEntry.String())

		// Add comma if not the last entry
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString("  }\n}")
	return []byte(buf.String()), nil
}

// actionRefPattern matches "uses: actions/repo@SHA-or-tag" in workflow files.
// Captures: (1) indentation+uses prefix, (2) repo path, (3) SHA or version tag,
// (4) optional version comment (e.g., "v6.0.2" from "# v6.0.2"), (5) trailing whitespace.
var actionRefPattern = regexp.MustCompile(`(uses:\s+)(actions/[a-zA-Z0-9_.-]+(?:/[a-zA-Z0-9_.-]+)*)@([a-fA-F0-9]{40}|[^\s#\n]+?)(\s*#\s*\S+)?(\s*)$`)

// getLatestActionReleaseFn is the function used to fetch the latest release for an action.
// It can be replaced in tests to avoid network calls.
var getLatestActionReleaseFn = getLatestActionRelease

// latestReleaseResult caches a resolved version/SHA pair.
type latestReleaseResult struct {
	version string
	sha     string
}

// UpdateActionsInWorkflowFiles scans all workflow .md files under workflowsDir
// (recursively) and updates any "uses: actions/*@version" references to the latest
// major version. Updated files are recompiled. Core actions (actions/*) always update
// to latest major.
func UpdateActionsInWorkflowFiles(workflowsDir, engineOverride string, verbose bool) error {
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

		updated, newContent, err := updateActionRefsInContent(string(content), cache, verbose)
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

		// Recompile the updated workflow
		if err := compileWorkflowWithRefresh(path, verbose, false, engineOverride, false); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to recompile %s: %v", path, err)))
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

// updateActionRefsInContent replaces outdated "uses: actions/*@version" references
// in content with the latest major version and SHA. Returns (changed, newContent, error).
// cache is keyed by "repo@currentVersion" and avoids redundant API calls across lines/files.
func updateActionRefsInContent(content string, cache map[string]latestReleaseResult, verbose bool) (bool, string, error) {
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
			latestVersion, latestSHA, err := getLatestActionReleaseFn(repo, currentVersion, true, verbose)
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
