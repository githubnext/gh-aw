package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

// NewUpdateCommand creates the update command
func NewUpdateCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [workflow-name]...",
		Short: "Update workflows from their source repositories",
		Long: `Update one or more workflows from their source repositories.

The command uses the 'source' field in the workflow frontmatter to determine
the source repository and version. It then fetches the latest version based on
the current ref:
- If the ref is a tag, it updates to the latest release (use --major for major version updates)
- If the ref is a branch, it fetches the latest commit from that branch
- Otherwise, it fetches the latest commit from the default branch

Examples:
  ` + constants.CLIExtensionPrefix + ` update                    # Update all workflows with source field
  ` + constants.CLIExtensionPrefix + ` update ci-doctor         # Update specific workflow
  ` + constants.CLIExtensionPrefix + ` update ci-doctor --major # Allow major version updates
  ` + constants.CLIExtensionPrefix + ` update --force           # Force update even if no changes`,
		Run: func(cmd *cobra.Command, args []string) {
			majorFlag, _ := cmd.Flags().GetBool("major")
			forceFlag, _ := cmd.Flags().GetBool("force")
			engineOverride, _ := cmd.Flags().GetString("engine")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if err := validateEngine(engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}

			if err := UpdateWorkflows(args, majorFlag, forceFlag, verbose, engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().Bool("major", false, "Allow major version updates when updating tagged releases")
	cmd.Flags().Bool("force", false, "Force update even if no changes detected")
	cmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex, copilot)")

	return cmd
}

// UpdateWorkflows updates workflows from their source repositories
func UpdateWorkflows(workflowNames []string, allowMajor, force, verbose bool, engineOverride string) error {
	workflowsDir := getWorkflowsDir()

	// Find all workflows with source field
	workflows, err := findWorkflowsWithSource(workflowsDir, workflowNames, verbose)
	if err != nil {
		return err
	}

	if len(workflows) == 0 {
		if len(workflowNames) > 0 {
			return fmt.Errorf("no workflows found matching the specified names with source field")
		}
		return fmt.Errorf("no workflows found with source field")
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow(s) to update", len(workflows))))

	// Update each workflow
	updatedCount := 0
	for _, wf := range workflows {
		if err := updateWorkflow(wf, allowMajor, force, verbose, engineOverride); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update %s: %v", wf.Name, err)))
			continue
		}
		updatedCount++
	}

	if updatedCount == 0 {
		return fmt.Errorf("no workflows were successfully updated")
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully updated %d workflow(s)", updatedCount)))
	return nil
}

// workflowWithSource represents a workflow with its source information
type workflowWithSource struct {
	Name       string
	Path       string
	SourceSpec string // e.g., "owner/repo/path@ref"
}

// findWorkflowsWithSource finds all workflows that have a source field
func findWorkflowsWithSource(workflowsDir string, filterNames []string, verbose bool) ([]*workflowWithSource, error) {
	var workflows []*workflowWithSource

	// Read all .md files in workflows directory
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Skip .lock.yml files
		if strings.HasSuffix(entry.Name(), ".lock.yml") {
			continue
		}

		workflowPath := filepath.Join(workflowsDir, entry.Name())
		workflowName := strings.TrimSuffix(entry.Name(), ".md")

		// Filter by name if specified
		if len(filterNames) > 0 {
			matched := false
			for _, filterName := range filterNames {
				// Remove .md extension if present
				filterName = strings.TrimSuffix(filterName, ".md")
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
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Resolving latest ref for %s (current: %s)", repo, currentRef)))
	}

	// Check if current ref is a tag (looks like a semantic version)
	if isSemanticVersionTag(currentRef) {
		return resolveLatestRelease(repo, currentRef, allowMajor, verbose)
	}

	// Check if current ref is a branch by checking if it exists as a branch
	isBranch, err := isBranchRef(repo, currentRef)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check if ref is branch: %v", err)))
		}
		// If we can't determine, treat as default branch case
		return resolveDefaultBranchHead(repo, verbose)
	}

	if isBranch {
		return resolveBranchHead(repo, currentRef, verbose)
	}

	// Otherwise, use default branch
	return resolveDefaultBranchHead(repo, verbose)
}

// isSemanticVersionTag checks if a ref looks like a semantic version tag
// resolveLatestRelease finds the latest release, respecting semantic versioning
func resolveLatestRelease(repo, currentRef string, allowMajor, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest release for %s (current: %s, allow major: %v)", repo, currentRef, allowMajor)))
	}

	// Use gh CLI to get releases
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/releases", repo), "--jq", ".[].tag_name")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) == 0 || releases[0] == "" {
		return "", fmt.Errorf("no releases found")
	}

	// Parse current version
	currentVersion := parseVersion(currentRef)
	if currentVersion == nil {
		// If current ref is not a valid version, just return the latest release
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current ref is not a valid version, using latest release: %s", releases[0])))
		}
		return releases[0], nil
	}

	// Find the latest compatible release
	var latestCompatible string
	var latestCompatibleVersion *semanticVersion

	for _, release := range releases {
		releaseVersion := parseVersion(release)
		if releaseVersion == nil {
			continue
		}

		// Check if compatible based on major version
		if !allowMajor && releaseVersion.major != currentVersion.major {
			continue
		}

		// Check if this is newer than what we have
		if latestCompatibleVersion == nil || releaseVersion.isNewer(latestCompatibleVersion) {
			latestCompatible = release
			latestCompatibleVersion = releaseVersion
		}
	}

	if latestCompatible == "" {
		return "", fmt.Errorf("no compatible release found")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest compatible release: %s", latestCompatible)))
	}

	return latestCompatible, nil
}

// isBranchRef checks if a ref is a branch in the repository
func isBranchRef(repo, ref string) (bool, error) {
	// Use gh CLI to list branches
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/branches", repo), "--jq", ".[].name")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, branch := range branches {
		if branch == ref {
			return true, nil
		}
	}

	return false, nil
}

// resolveBranchHead gets the latest commit SHA for a branch
func resolveBranchHead(repo, branch string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest commit for branch %s in %s", branch, repo)))
	}

	// Use gh CLI to get branch info
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/branches/%s", repo, branch), "--jq", ".commit.sha")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch branch info: %w", err)
	}

	sha := strings.TrimSpace(string(output))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest commit on %s: %s", branch, sha)))
	}

	return sha, nil
}

// resolveDefaultBranchHead gets the latest commit SHA for the default branch
func resolveDefaultBranchHead(repo string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching default branch for %s", repo)))
	}

	// First get the default branch name
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s", repo), "--jq", ".default_branch")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch repository info: %w", err)
	}

	defaultBranch := strings.TrimSpace(string(output))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Default branch: %s", defaultBranch)))
	}

	return resolveBranchHead(repo, defaultBranch, verbose)
}

// updateWorkflow updates a single workflow from its source
func updateWorkflow(wf *workflowWithSource, allowMajor, force, verbose bool, engineOverride string) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("\nUpdating workflow: %s", wf.Name)))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Source: %s", wf.SourceSpec)))
	}

	// Parse source spec
	sourceSpec, err := parseSourceSpec(wf.SourceSpec)
	if err != nil {
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

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current ref: %s", currentRef)))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest ref: %s", latestRef)))
	}

	// Check if update is needed
	if !force && currentRef == latestRef {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, currentRef)))
		return nil
	}

	// Download the base version (current ref from source)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Downloading base version from %s/%s@%s", sourceSpec.Repo, sourceSpec.Path, currentRef)))
	}

	baseContent, err := downloadWorkflowContent(sourceSpec.Repo, sourceSpec.Path, currentRef, verbose)
	if err != nil {
		return fmt.Errorf("failed to download base workflow: %w", err)
	}

	// Download the latest version
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Downloading latest version from %s/%s@%s", sourceSpec.Repo, sourceSpec.Path, latestRef)))
	}

	newContent, err := downloadWorkflowContent(sourceSpec.Repo, sourceSpec.Path, latestRef, verbose)
	if err != nil {
		return fmt.Errorf("failed to download workflow: %w", err)
	}

	// Read current workflow content
	currentContent, err := os.ReadFile(wf.Path)
	if err != nil {
		return fmt.Errorf("failed to read current workflow: %w", err)
	}

	// Perform 3-way merge using git merge-file
	mergedContent, hasConflicts, err := mergeWorkflowContent(string(baseContent), string(currentContent), string(newContent), wf.SourceSpec, latestRef, verbose)
	if err != nil {
		return fmt.Errorf("failed to merge workflow content: %w", err)
	}

	// Write updated content
	if err := os.WriteFile(wf.Path, []byte(mergedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}

	if hasConflicts {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Updated %s from %s to %s with CONFLICTS - please review and resolve manually", wf.Name, currentRef, latestRef)))
		return nil // Not an error, but user needs to resolve conflicts
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", wf.Name, currentRef, latestRef)))

	// Compile the updated workflow
	if err := compileWorkflow(wf.Path, verbose, engineOverride); err != nil {
		return fmt.Errorf("failed to compile updated workflow: %w", err)
	}

	return nil
}

// downloadWorkflowContent downloads the content of a workflow file from GitHub
func downloadWorkflowContent(repo, path, ref string, verbose bool) ([]byte, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching %s/%s@%s", repo, path, ref)))
	}

	// Use gh CLI to download the file
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, path, ref), "--jq", ".content")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file content: %w", err)
	}

	// The content is base64 encoded, decode it
	contentBase64 := strings.TrimSpace(string(output))
	cmd = exec.Command("base64", "-d")
	cmd.Stdin = strings.NewReader(contentBase64)
	content, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return content, nil
}

// mergeWorkflowContent performs a 3-way merge of workflow content using git merge-file
// It returns the merged content, whether conflicts exist, and any error
func mergeWorkflowContent(base, current, new, oldSourceSpec, newRef string, verbose bool) (string, bool, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Performing 3-way merge using git merge-file"))
	}

	// First, update the source field in the new content before merging
	// This ensures the source field is correct even if there are conflicts
	newWithUpdatedSource, err := updateSourceFieldInContent(new, oldSourceSpec, newRef)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update source in new content: %v", err)))
		}
		// Continue with original new content
		newWithUpdatedSource = new
	}

	// Create temporary directory for merge files
	tmpDir, err := os.MkdirTemp("", "gh-aw-merge-*")
	if err != nil {
		return "", false, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write base, current, and new versions to temporary files
	baseFile := filepath.Join(tmpDir, "base.md")
	currentFile := filepath.Join(tmpDir, "current.md")
	newFile := filepath.Join(tmpDir, "new.md")

	if err := os.WriteFile(baseFile, []byte(base), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write base file: %w", err)
	}
	if err := os.WriteFile(currentFile, []byte(current), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write current file: %w", err)
	}
	if err := os.WriteFile(newFile, []byte(newWithUpdatedSource), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write new file: %w", err)
	}

	// Execute git merge-file
	// Format: git merge-file <current> <base> <new>
	cmd := exec.Command("git", "merge-file",
		"-L", "current (local changes)",
		"-L", "base (original)",
		"-L", "new (upstream)",
		"--diff3", // Use diff3 style conflict markers for better context
		currentFile, baseFile, newFile)

	output, err := cmd.CombinedOutput()

	// git merge-file returns:
	// - 0 if merge was successful without conflicts
	// - >0 if conflicts were found (appears to return number of conflicts, but file is still updated)
	// The exit code can be >1 for multiple conflicts, not just errors
	hasConflicts := false
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode > 0 && exitCode < 128 {
				// Conflicts found (exit codes 1-127 indicate conflicts)
				// Exit codes >= 128 typically indicate system errors
				hasConflicts = true
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Merge conflicts detected (exit code: %d)", exitCode)))
				}
			} else {
				// Real error (exit code >= 128)
				return "", false, fmt.Errorf("git merge-file failed: %w\nOutput: %s", err, output)
			}
		} else {
			return "", false, fmt.Errorf("failed to execute git merge-file: %w", err)
		}
	}

	// Read the merged content from the current file (git merge-file updates it in-place)
	mergedContent, err := os.ReadFile(currentFile)
	if err != nil {
		return "", false, fmt.Errorf("failed to read merged content: %w", err)
	}

	mergedStr := string(mergedContent)

	// Process @include directives if present and no conflicts
	// Skip include processing if there are conflicts to avoid errors
	if !hasConflicts {
		sourceSpec, err := parseSourceSpec(oldSourceSpec)
		if err == nil {
			workflow := &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: sourceSpec.Repo,
					Version:  newRef,
				},
				WorkflowPath: sourceSpec.Path,
			}

			processedContent, err := processIncludesInContent(mergedStr, workflow, newRef, verbose)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
				}
				// Return unprocessed content on error
			} else {
				mergedStr = processedContent
			}
		}
	}

	return mergedStr, hasConflicts, nil
}

// updateSourceFieldInContent updates the source field in workflow content
func updateSourceFieldInContent(content, oldSourceSpec, newRef string) (string, error) {
	// Parse the content
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if result.Frontmatter == nil {
		result.Frontmatter = make(map[string]any)
	}

	// Parse the old source spec to construct the new one
	sourceSpec, err := parseSourceSpec(oldSourceSpec)
	if err != nil {
		return "", fmt.Errorf("failed to parse source spec: %w", err)
	}

	// Update with new ref
	newSourceSpec := fmt.Sprintf("%s/%s@%s", sourceSpec.Repo, sourceSpec.Path, newRef)
	result.Frontmatter["source"] = newSourceSpec

	// Reconstruct the workflow file
	return reconstructWorkflowFile(result.Frontmatter, result.Markdown)
}

// reconstructWorkflowFile reconstructs a workflow file from frontmatter and markdown
func reconstructWorkflowFile(frontmatter map[string]any, markdown string) (string, error) {
	// Convert frontmatter to YAML with proper field ordering
	// Use PriorityWorkflowFields to ensure consistent ordering of top-level fields
	updatedFrontmatter, err := workflow.MarshalWithFieldOrder(frontmatter, constants.PriorityWorkflowFields)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Clean up the YAML - remove trailing newline and unquote the "on" key
	frontmatterStr := strings.TrimSuffix(string(updatedFrontmatter), "\n")
	frontmatterStr = workflow.UnquoteYAMLKey(frontmatterStr, "on")

	// Reconstruct the file
	var lines []string
	lines = append(lines, "---")
	if frontmatterStr != "" {
		lines = append(lines, strings.Split(frontmatterStr, "\n")...)
	}
	lines = append(lines, "---")
	if markdown != "" {
		lines = append(lines, markdown)
	}

	return strings.Join(lines, "\n"), nil
}
