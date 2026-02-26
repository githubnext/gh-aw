package cli

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

// UpdateWorkflows updates workflows from their source repositories
func UpdateWorkflows(workflowNames []string, allowMajor, force, verbose bool, engineOverride string, workflowsDir string, noStopAfter bool, stopAfter string, noMerge bool) error {
	updateLog.Printf("Scanning for workflows with source field: dir=%s, filter=%v, noMerge=%v", workflowsDir, workflowNames, noMerge)

	// Use provided workflows directory or default
	if workflowsDir == "" {
		workflowsDir = getWorkflowsDir()
	}

	// Find all workflows with source field
	workflows, err := findWorkflowsWithSource(workflowsDir, workflowNames, verbose)
	if err != nil {
		return err
	}

	updateLog.Printf("Found %d workflows with source field", len(workflows))

	if len(workflows) == 0 {
		if len(workflowNames) > 0 {
			return errors.New("no workflows found matching the specified names with source field")
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("no workflows found with source field"))
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow(s) to update", len(workflows))))

	// Track update results
	var successfulUpdates []string
	var failedUpdates []updateFailure

	// Update each workflow
	for _, wf := range workflows {
		updateLog.Printf("Updating workflow: %s (source: %s)", wf.Name, wf.SourceSpec)
		if err := updateWorkflow(wf, allowMajor, force, verbose, engineOverride, noStopAfter, stopAfter, noMerge); err != nil {
			updateLog.Printf("Failed to update workflow %s: %v", wf.Name, err)
			failedUpdates = append(failedUpdates, updateFailure{
				Name:  wf.Name,
				Error: err.Error(),
			})
			continue
		}
		updateLog.Printf("Successfully updated workflow: %s", wf.Name)
		successfulUpdates = append(successfulUpdates, wf.Name)
	}

	// Show summary
	showUpdateSummary(successfulUpdates, failedUpdates)

	if len(successfulUpdates) == 0 {
		return errors.New("no workflows were successfully updated")
	}

	return nil
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Skip .lock.yml files
		if strings.HasSuffix(entry.Name(), ".lock.yml") {
			continue
		}

		workflowPath := filepath.Join(workflowsDir, entry.Name())
		workflowName := normalizeWorkflowID(entry.Name())

		// Filter by name if specified
		if len(filterNames) > 0 {
			matched := false
			for _, filterName := range filterNames {
				// Normalize filter name to handle both "workflow" and "workflow.md" formats
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

		// Read the workflow file and extract source field
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", workflowPath, err)))
			}
			continue
		}

		// Parse frontmatter
		result, err := parser.ExtractFrontmatterFromContent(string(content))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse frontmatter in %s: %v", workflowPath, err)))
			}
			continue
		}

		// Check for source field
		sourceRaw, ok := result.Frontmatter["source"]
		if !ok {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Skipping %s: no source field", workflowName)))
			}
			continue
		}

		source, ok := sourceRaw.(string)
		if !ok || source == "" {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: invalid source field", workflowName)))
			}
			continue
		}

		workflows = append(workflows, &workflowWithSource{
			Name:       workflowName,
			Path:       workflowPath,
			SourceSpec: strings.TrimSpace(source),
		})
	}

	return workflows, nil
}

// resolveLatestRef resolves the latest ref for a workflow source
func resolveLatestRef(repo, currentRef string, allowMajor, verbose bool) (string, error) {
	updateLog.Printf("Resolving latest ref: repo=%s, currentRef=%s, allowMajor=%v", repo, currentRef, allowMajor)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Resolving latest ref for %s (current: %s)", repo, currentRef)))
	}

	// Check if current ref is a tag (looks like a semantic version)
	if isSemanticVersionTag(currentRef) {
		updateLog.Print("Current ref is semantic version tag, resolving latest release")
		return resolveLatestRelease(repo, currentRef, allowMajor, verbose)
	}

	// Check if current ref is a commit SHA (40-character hex string)
	if IsCommitSHA(currentRef) {
		updateLog.Printf("Current ref is a commit SHA: %s, fetching latest from default branch", currentRef)
		// The source field only contains a pinned SHA with no branch information.
		// Fetch the latest commit from the default branch to check for updates.
		return resolveLatestCommitFromDefaultBranch(repo, currentRef, verbose)
	}

	// Otherwise, treat as branch and get latest commit
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Treating %s as branch, getting latest commit", currentRef)))
	}

	// Get the latest commit SHA for the branch
	latestSHA, err := getLatestBranchCommitSHA(repo, currentRef)
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
func resolveLatestCommitFromDefaultBranch(repo, currentSHA string, verbose bool) (string, error) {
	// Get the default branch name
	defaultBranch, err := getRepoDefaultBranch(repo)
	if err != nil {
		return "", fmt.Errorf("failed to get default branch for %s: %w", repo, err)
	}

	updateLog.Printf("Source is pinned to commit SHA, tracking default branch %q of %s", defaultBranch, repo)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Source is pinned to commit SHA, checking default branch %q for updates", defaultBranch)))
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Source has no branch ref, tracking default branch %q", defaultBranch)))

	// Get the latest commit SHA from the default branch
	latestSHA, err := getLatestBranchCommitSHA(repo, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit for default branch %s: %w", defaultBranch, err)
	}

	updateLog.Printf("Latest commit on default branch %s: %s (current: %s)", defaultBranch, latestSHA, currentSHA)

	return latestSHA, nil
}

// getRepoDefaultBranch fetches the default branch name for a repository.
func getRepoDefaultBranch(repo string) (string, error) {
	output, err := workflow.RunGH("Fetching repo info...", "api", "/repos/"+repo, "--jq", ".default_branch")
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
func getLatestBranchCommitSHA(repo, branch string) (string, error) {
	// URL-encode the branch name since it may contain slashes (e.g. "feature/foo")
	output, err := workflow.RunGH("Fetching branch info...", "api", fmt.Sprintf("/repos/%s/branches/%s", repo, url.PathEscape(branch)), "--jq", ".commit.sha")
	if err != nil {
		return "", err
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", fmt.Errorf("empty commit SHA returned for branch %s", branch)
	}

	return sha, nil
}

// resolveLatestRelease resolves the latest compatible release for a workflow source
func resolveLatestRelease(repo, currentRef string, allowMajor, verbose bool) (string, error) {
	updateLog.Printf("Resolving latest release for repo %s (current: %s, allowMajor=%v)", repo, currentRef, allowMajor)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Checking for latest release (current: %s, allow major: %v)", currentRef, allowMajor)))
	}

	// Get all releases using gh CLI
	output, err := workflow.RunGH("Fetching releases...", "api", fmt.Sprintf("/repos/%s/releases", repo), "--jq", ".[].tag_name")
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) == 0 || releases[0] == "" {
		return "", errors.New("no releases found")
	}

	// Parse current version
	currentVer := parseVersion(currentRef)
	if currentVer == nil {
		// If current version is not a valid semantic version, just return the latest release
		latestRelease := releases[0]
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Current version is not valid, using latest release: "+latestRelease))
		}
		return latestRelease, nil
	}

	// Find the latest compatible release
	var latestCompatible string
	var latestCompatibleVersion *semanticVersion

	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer == nil {
			continue
		}

		// Check if compatible based on major version
		if !allowMajor && releaseVer.major != currentVer.major {
			continue
		}

		// Check if this is newer than what we have
		if latestCompatibleVersion == nil || releaseVer.isNewer(latestCompatibleVersion) {
			latestCompatible = release
			latestCompatibleVersion = releaseVer
		}
	}

	if latestCompatible == "" {
		return "", errors.New("no compatible release found")
	}

	if verbose && latestCompatible != currentRef {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found newer release: "+latestCompatible))
	}

	return latestCompatible, nil
}

// updateWorkflow updates a single workflow from its source
func updateWorkflow(wf *workflowWithSource, allowMajor, force, verbose bool, engineOverride string, noStopAfter bool, stopAfter string, noMerge bool) error {
	updateLog.Printf("Updating workflow: name=%s, source=%s, force=%v, noMerge=%v", wf.Name, wf.SourceSpec, force, noMerge)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nUpdating workflow: "+wf.Name))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Source: "+wf.SourceSpec))
	}

	// Parse source spec
	sourceSpec, err := parseSourceSpec(wf.SourceSpec)
	if err != nil {
		updateLog.Printf("Failed to parse source spec: %v", err)
		return fmt.Errorf("failed to parse source spec: %w", err)
	}

	// If no ref specified, use default branch
	currentRef := sourceSpec.Ref
	if currentRef == "" {
		currentRef = "main"
	}

	// Resolve latest ref
	latestRef, err := resolveLatestRef(sourceSpec.Repo, currentRef, allowMajor, verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve latest ref: %w", err)
	}

	// For branch refs, resolveLatestRef returns the branch-head SHA so that
	// we can detect upstream changes (currentRef != latestRef). However the
	// source field must keep the branch *name* to avoid SHA-pinning.
	sourceFieldRef := latestRef
	if isBranchRef(currentRef) {
		sourceFieldRef = currentRef
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Current ref: "+currentRef))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Latest ref: "+latestRef))
	}

	// Check if update is needed
	if !force && currentRef == latestRef {
		updateLog.Printf("Workflow already at latest ref: %s, checking for local modifications", currentRef)

		// Download the source content to check if local file has been modified
		sourceContent, err := downloadWorkflowContent(sourceSpec.Repo, sourceSpec.Path, currentRef, verbose)
		if err != nil {
			// If we can't download for comparison, just show the up-to-date message
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download source for comparison: %v", err)))
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, shortRef(currentRef))))
			return nil
		}

		// Read current workflow content
		currentContent, err := os.ReadFile(wf.Path)
		if err != nil {
			return fmt.Errorf("failed to read current workflow: %w", err)
		}

		// Check if local file differs from source
		if hasLocalModifications(string(sourceContent), string(currentContent), wf.SourceSpec, verbose) {
			updateLog.Printf("Local modifications detected in workflow: %s", wf.Name)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, shortRef(currentRef))))
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠️  Local copy of %s has been modified from source", wf.Name)))
			return nil
		}

		updateLog.Printf("Workflow %s is up to date with no local modifications", wf.Name)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, shortRef(currentRef))))
		return nil
	}

	// Download the latest version
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Downloading latest version from %s/%s@%s", sourceSpec.Repo, sourceSpec.Path, latestRef)))
	}

	newContent, err := downloadWorkflowContent(sourceSpec.Repo, sourceSpec.Path, latestRef, verbose)
	if err != nil {
		return fmt.Errorf("failed to download workflow: %w", err)
	}

	// Determine merge mode. Merge is the default behaviour — it detects
	// local modifications and performs a 3-way merge to preserve them.
	// When --no-merge is used, local changes are overridden with upstream.
	merge := !noMerge

	// When merge mode is on, detect local modifications to confirm we
	// actually need to merge (if no local mods, override is fine either way).
	if merge {
		baseContent, dlErr := downloadWorkflowContent(sourceSpec.Repo, sourceSpec.Path, currentRef, verbose)
		if dlErr == nil {
			localContent, readErr := os.ReadFile(wf.Path)
			if readErr == nil && hasLocalModifications(string(baseContent), string(localContent), wf.SourceSpec, verbose) {
				updateLog.Printf("Local modifications detected in %s, merging to preserve changes", wf.Name)
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Local modifications detected in %s, merging to preserve your changes", wf.Name)))
			} else {
				// No local modifications — no need to merge, just override
				merge = false
			}
		}
	}

	var finalContent string
	var hasConflicts bool

	// Decide whether to merge or override
	if merge {
		// Merge mode: perform 3-way merge to preserve local changes
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Using merge mode to preserve local changes"))
		}

		// Download the base version (current ref from source)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Downloading base version from %s/%s@%s", sourceSpec.Repo, sourceSpec.Path, currentRef)))
		}

		baseContent, err := downloadWorkflowContent(sourceSpec.Repo, sourceSpec.Path, currentRef, verbose)
		if err != nil {
			return fmt.Errorf("failed to download base workflow: %w", err)
		}

		// Read current workflow content
		currentContent, err := os.ReadFile(wf.Path)
		if err != nil {
			return fmt.Errorf("failed to read current workflow: %w", err)
		}

		// Perform 3-way merge using git merge-file
		updateLog.Printf("Performing 3-way merge for workflow: %s", wf.Name)
		mergedContent, conflicts, err := MergeWorkflowContent(string(baseContent), string(currentContent), string(newContent), wf.SourceSpec, sourceFieldRef, verbose)
		if err != nil {
			updateLog.Printf("Merge failed for workflow %s: %v", wf.Name, err)
			return fmt.Errorf("failed to merge workflow content: %w", err)
		}

		finalContent = mergedContent
		hasConflicts = conflicts

		if hasConflicts {
			updateLog.Printf("Merge conflicts detected in workflow: %s", wf.Name)
		}
	} else {
		// Override mode (default): replace local file with new content from source
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Using override mode - local changes will be replaced"))
		}

		// Update the source field in the new content with the new ref
		newWithUpdatedSource, err := UpdateFieldInFrontmatter(string(newContent), "source", fmt.Sprintf("%s/%s@%s", sourceSpec.Repo, sourceSpec.Path, sourceFieldRef))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update source in new content: %v", err)))
			}
			// Continue with original new content
			finalContent = string(newContent)
		} else {
			finalContent = newWithUpdatedSource
		}

		// Process @include directives if present
		workflow := &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug: sourceSpec.Repo,
				Version:  latestRef,
			},
			WorkflowPath: sourceSpec.Path,
		}

		processedContent, err := processIncludesInContent(finalContent, workflow, latestRef, verbose)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
			}
			// Continue with unprocessed content
		} else {
			finalContent = processedContent
		}
	}

	// Handle stop-after field modifications
	if noStopAfter {
		// Remove stop-after field if requested
		cleanedContent, err := RemoveFieldFromOnTrigger(finalContent, "stop-after")
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove stop-after field: %v", err)))
			}
		} else {
			finalContent = cleanedContent
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed stop-after field from workflow"))
			}
		}
	} else if stopAfter != "" {
		// Set custom stop-after value if provided
		updatedContent, err := SetFieldInOnTrigger(finalContent, "stop-after", stopAfter)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set stop-after field: %v", err)))
			}
		} else {
			finalContent = updatedContent
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Set stop-after field to: "+stopAfter))
			}
		}
	}

	// Write updated content
	if err := os.WriteFile(wf.Path, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}

	if hasConflicts {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Updated %s from %s to %s with CONFLICTS - please review and resolve manually", wf.Name, shortRef(currentRef), shortRef(latestRef))))
		return nil // Not an error, but user needs to resolve conflicts
	}

	updateLog.Printf("Successfully updated workflow %s from %s to %s", wf.Name, currentRef, latestRef)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", wf.Name, shortRef(currentRef), shortRef(latestRef))))

	// Compile the updated workflow with refreshStopTime enabled
	updateLog.Printf("Compiling updated workflow: %s", wf.Name)
	if err := compileWorkflowWithRefresh(wf.Path, verbose, false, engineOverride, true); err != nil {
		updateLog.Printf("Compilation failed for workflow %s: %v", wf.Name, err)
		return fmt.Errorf("failed to compile updated workflow: %w", err)
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
