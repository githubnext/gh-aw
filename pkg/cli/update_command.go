package cli

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/gitutil"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var updateLog = logger.New("cli:update_command")

// NewUpdateCommand creates the update command
func NewUpdateCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [workflow-id]...",
		Short: "Update workflows from their source repositories and check for gh-aw updates",
		Long: `Update one or more workflows from their source repositories and check for gh-aw updates.

The command:
1. Checks if a newer version of gh-aw is available
2. Updates GitHub Actions versions in .github/aw/actions-lock.json (unless --no-actions is set)
3. Updates workflows using the 'source' field in the workflow frontmatter
4. Compiles each workflow immediately after update

By default, the update command replaces local workflow files with the latest version from the source
repository, overriding any local changes. Use the --merge flag to preserve local changes by performing
a 3-way merge between the base version, your local changes, and the latest upstream version.

For workflow updates, it fetches the latest version based on the current ref:
- If the ref is a tag, it updates to the latest release (use --major for major version updates)
- If the ref is a branch, it fetches the latest commit from that branch
- Otherwise, it fetches the latest commit from the default branch

For action updates, it checks each action in .github/aw/actions-lock.json for newer releases
and updates the SHA to pin to the latest version. Use --no-actions to skip action updates.

` + WorkflowIDExplanation + `

Examples:
  ` + constants.CLIExtensionPrefix + ` update                    # Check gh-aw updates, update actions, and update all workflows
  ` + constants.CLIExtensionPrefix + ` update ci-doctor         # Check gh-aw updates, update actions, and update specific workflow
  ` + constants.CLIExtensionPrefix + ` update --no-actions      # Skip action updates, only update workflows
  ` + constants.CLIExtensionPrefix + ` update ci-doctor.md      # Check gh-aw updates, update actions, and update specific workflow (alternative format)
  ` + constants.CLIExtensionPrefix + ` update ci-doctor --major # Allow major version updates
  ` + constants.CLIExtensionPrefix + ` update --merge           # Update with 3-way merge to preserve local changes
  ` + constants.CLIExtensionPrefix + ` update --pr              # Create PR with changes
  ` + constants.CLIExtensionPrefix + ` update --force           # Force update even if no changes
  ` + constants.CLIExtensionPrefix + ` update --dir custom/workflows  # Update workflows in custom directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			majorFlag, _ := cmd.Flags().GetBool("major")
			forceFlag, _ := cmd.Flags().GetBool("force")
			engineOverride, _ := cmd.Flags().GetString("engine")
			verbose, _ := cmd.Flags().GetBool("verbose")
			prFlag, _ := cmd.Flags().GetBool("pr")
			workflowDir, _ := cmd.Flags().GetString("dir")
			noStopAfter, _ := cmd.Flags().GetBool("no-stop-after")
			stopAfter, _ := cmd.Flags().GetString("stop-after")
			mergeFlag, _ := cmd.Flags().GetBool("merge")
			noActions, _ := cmd.Flags().GetBool("no-actions")

			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			return UpdateWorkflowsWithExtensionCheck(args, majorFlag, forceFlag, verbose, engineOverride, prFlag, workflowDir, noStopAfter, stopAfter, mergeFlag, noActions)
		},
	}

	cmd.Flags().Bool("major", false, "Allow major version updates when updating tagged releases")
	cmd.Flags().Bool("force", false, "Force update even if no changes are detected")
	addEngineFlag(cmd)
	cmd.Flags().Bool("pr", false, "Create a pull request with the workflow changes")
	cmd.Flags().String("dir", "", "Workflow directory (default: .github/workflows)")
	cmd.Flags().Bool("no-stop-after", false, "Remove any stop-after field from the workflow")
	cmd.Flags().String("stop-after", "", "Override stop-after value in the workflow (e.g., '+48h', '2025-12-31 23:59:59')")
	cmd.Flags().Bool("merge", false, "Merge local changes with upstream updates instead of overriding")
	cmd.Flags().Bool("no-actions", false, "Skip updating GitHub Actions versions")

	// Register completions for update command
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// checkExtensionUpdate checks if a newer version of gh-aw is available
func checkExtensionUpdate(verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Checking for gh-aw extension updates..."))
	}

	// Run gh extension upgrade --dry-run to check for updates
	cmd := exec.Command("gh", "extension", "upgrade", "githubnext/gh-aw", "--dry-run")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check for extension updates: %v", err)))
		}
		return nil // Don't fail the whole command if update check fails
	}

	outputStr := strings.TrimSpace(string(output))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Extension update check output: %s", outputStr)))
	}

	// Parse the output to see if an update is available
	// Expected format: "[aw]: would have upgraded from v0.14.0 to v0.18.1"
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "[aw]: would have upgraded from") {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(line))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Run 'gh extension upgrade githubnext/gh-aw' to update"))
			return nil
		}
	}

	if strings.Contains(outputStr, "✓ Successfully checked extension upgrades") {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("gh-aw extension is up to date"))
		}
	}

	return nil
}

// UpdateWorkflowsWithExtensionCheck performs the complete update process:
// 1. Check for gh-aw extension updates
// 2. Update GitHub Actions versions (unless --no-actions flag is set)
// 3. Update workflows from source repositories (compiles each workflow after update)
// 4. Optionally create a PR
func UpdateWorkflowsWithExtensionCheck(workflowNames []string, allowMajor, force, verbose bool, engineOverride string, createPR bool, workflowsDir string, noStopAfter bool, stopAfter string, merge bool, noActions bool) error {
	updateLog.Printf("Starting update process: workflows=%v, allowMajor=%v, force=%v, createPR=%v, merge=%v, noActions=%v", workflowNames, allowMajor, force, createPR, merge, noActions)

	// Step 1: Check for gh-aw extension updates
	if err := checkExtensionUpdate(verbose); err != nil {
		return fmt.Errorf("extension update check failed: %w", err)
	}

	// Step 2: Update GitHub Actions versions (unless disabled)
	if !noActions {
		if err := UpdateActions(allowMajor, verbose); err != nil {
			return fmt.Errorf("action update failed: %w", err)
		}
	}

	// Step 3: Update workflows from source repositories
	// Note: Each workflow is compiled immediately after update
	if err := UpdateWorkflows(workflowNames, allowMajor, force, verbose, engineOverride, workflowsDir, noStopAfter, stopAfter, merge); err != nil {
		return fmt.Errorf("workflow update failed: %w", err)
	}

	// Step 4: Optionally create PR if flag is set
	if createPR {
		if err := createUpdatePR(verbose); err != nil {
			return fmt.Errorf("failed to create PR: %w", err)
		}
	}

	return nil
}

// hasGitChanges checks if there are any uncommitted changes
func hasGitChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// runGitCommand runs a git command with the specified arguments
func runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}

// createUpdatePR creates a pull request with the workflow changes
func createUpdatePR(verbose bool) error {
	// Check if GitHub CLI is available
	if !isGHCLIAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is required for PR creation but not found in PATH")
	}

	// Check if there are any changes to commit
	hasChanges, err := hasGitChanges()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if !hasChanges {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes to create PR for"))
		return nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Creating pull request with workflow updates..."))
	}

	// Create a branch name with timestamp
	randomNum := rand.Intn(9000) + 1000 // Generate number between 1000-9999
	branchName := fmt.Sprintf("update-workflows-%d", randomNum)

	// Create and checkout new branch
	if err := runGitCommand("checkout", "-b", branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Add all changes
	if err := runGitCommand("add", "."); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit changes
	commitMsg := "Update workflows and recompile"
	if err := runGitCommand("commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push branch
	if err := runGitCommand("push", "-u", "origin", branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	// Create PR
	cmd := exec.Command("gh", "pr", "create",
		"--title", "Update workflows and recompile",
		"--body", "This PR updates workflows from their source repositories and recompiles them.\n\nGenerated by `gh aw update --pr`")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create PR: %w\nOutput: %s", err, string(output))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully created pull request"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(strings.TrimSpace(string(output))))

	return nil
}

// UpdateWorkflows updates workflows from their source repositories
func UpdateWorkflows(workflowNames []string, allowMajor, force, verbose bool, engineOverride string, workflowsDir string, noStopAfter bool, stopAfter string, merge bool) error {
	updateLog.Printf("Scanning for workflows with source field: dir=%s, filter=%v, merge=%v", workflowsDir, workflowNames, merge)

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
			return fmt.Errorf("no workflows found matching the specified names with source field")
		}
		return fmt.Errorf("no workflows found with source field")
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow(s) to update", len(workflows))))

	// Track update results
	var successfulUpdates []string
	var failedUpdates []updateFailure

	// Update each workflow
	for _, wf := range workflows {
		if err := updateWorkflow(wf, allowMajor, force, verbose, engineOverride, noStopAfter, stopAfter, merge); err != nil {
			failedUpdates = append(failedUpdates, updateFailure{
				Name:  wf.Name,
				Error: err.Error(),
			})
			continue
		}
		successfulUpdates = append(successfulUpdates, wf.Name)
	}

	// Show summary
	showUpdateSummary(successfulUpdates, failedUpdates)

	if len(successfulUpdates) == 0 {
		return fmt.Errorf("no workflows were successfully updated")
	}

	return nil
}

// workflowWithSource represents a workflow with its source information
type workflowWithSource struct {
	Name       string
	Path       string
	SourceSpec string // e.g., "owner/repo/path@ref"
}

// updateFailure represents a failed workflow update
type updateFailure struct {
	Name  string
	Error string
}

// actionsLockEntry represents a single action pin entry
type actionsLockEntry struct {
	Repo    string `json:"repo"`
	Version string `json:"version"`
	SHA     string `json:"sha"`
}

// actionsLockFile represents the structure of actions-lock.json
type actionsLockFile struct {
	Entries map[string]actionsLockEntry `json:"entries"`
}

// showUpdateSummary displays a summary of workflow updates using console helpers
func showUpdateSummary(successfulUpdates []string, failedUpdates []updateFailure) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("=== Update Summary ==="))
	fmt.Fprintln(os.Stderr, "")

	// Show successful updates
	if len(successfulUpdates) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully updated and compiled %d workflow(s):", len(successfulUpdates))))
		for _, name := range successfulUpdates {
			fmt.Fprintln(os.Stderr, console.FormatListItem(name))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Show failed updates
	if len(failedUpdates) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to update %d workflow(s):", len(failedUpdates))))
		for _, failure := range failedUpdates {
			fmt.Fprintf(os.Stderr, "  %s %s: %s\n", console.FormatErrorMessage("✗"), failure.Name, failure.Error)
		}
		fmt.Fprintln(os.Stderr, "")
	}
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
	updateLog.Printf("Resolving latest ref: repo=%s, currentRef=%s, allowMajor=%v", repo, currentRef, allowMajor)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Resolving latest ref for %s (current: %s)", repo, currentRef)))
	}

	// Check if current ref is a tag (looks like a semantic version)
	if isSemanticVersionTag(currentRef) {
		updateLog.Print("Current ref is semantic version tag, resolving latest release")
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
		updateLog.Printf("Current ref is branch: %s", currentRef)
		return resolveBranchHead(repo, currentRef, verbose)
	}

	// Otherwise, use default branch
	updateLog.Print("Using default branch for ref resolution")
	return resolveDefaultBranchHead(repo, verbose)
}

// isSemanticVersionTag checks if a ref looks like a semantic version tag
// resolveLatestRelease finds the latest release, respecting semantic versioning
func resolveLatestRelease(repo, currentRef string, allowMajor, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest release for %s (current: %s, allow major: %v)", repo, currentRef, allowMajor)))
	}

	// Use gh CLI to get releases
	cmd := workflow.ExecGH("api", fmt.Sprintf("/repos/%s/releases", repo), "--jq", ".[].tag_name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if gitutil.IsAuthError(outputStr) || gitutil.IsAuthError(err.Error()) {
			updateLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback")
			// Try fallback using git ls-remote
			release, gitErr := resolveLatestReleaseViaGit(repo, currentRef, allowMajor, verbose)
			if gitErr != nil {
				return "", fmt.Errorf("failed to fetch releases via GitHub API and git: API error: %w, Git error: %v", err, gitErr)
			}
			return release, nil
		}
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

// updateWorkflow updates a single workflow from its source
func updateWorkflow(wf *workflowWithSource, allowMajor, force, verbose bool, engineOverride string, noStopAfter bool, stopAfter string, merge bool) error {
	updateLog.Printf("Updating workflow: name=%s, source=%s, force=%v, merge=%v", wf.Name, wf.SourceSpec, force, merge)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("\nUpdating workflow: %s", wf.Name)))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Source: %s", wf.SourceSpec)))
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

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current ref: %s", currentRef)))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest ref: %s", latestRef)))
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
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, currentRef)))
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
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, currentRef)))
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠️  Local copy of %s has been modified from source", wf.Name)))
			return nil
		}

		updateLog.Printf("Workflow %s is up to date with no local modifications", wf.Name)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, currentRef)))
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
		mergedContent, conflicts, err := MergeWorkflowContent(string(baseContent), string(currentContent), string(newContent), wf.SourceSpec, latestRef, verbose)
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
		newWithUpdatedSource, err := UpdateFieldInFrontmatter(string(newContent), "source", fmt.Sprintf("%s/%s@%s", sourceSpec.Repo, sourceSpec.Path, latestRef))
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
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Set stop-after field to: %s", stopAfter)))
			}
		}
	}

	// Write updated content
	if err := os.WriteFile(wf.Path, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}

	if hasConflicts {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Updated %s from %s to %s with CONFLICTS - please review and resolve manually", wf.Name, currentRef, latestRef)))
		return nil // Not an error, but user needs to resolve conflicts
	}

	updateLog.Printf("Successfully updated workflow %s from %s to %s", wf.Name, currentRef, latestRef)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", wf.Name, currentRef, latestRef)))

	// Compile the updated workflow with refreshStopTime enabled
	updateLog.Printf("Compiling updated workflow: %s", wf.Name)
	if err := compileWorkflowWithRefresh(wf.Path, verbose, engineOverride, true); err != nil {
		updateLog.Printf("Compilation failed for workflow %s: %v", wf.Name, err)
		return fmt.Errorf("failed to compile updated workflow: %w", err)
	}

	return nil
}

// normalizeWhitespace normalizes trailing whitespace and newlines to reduce spurious conflicts
func normalizeWhitespace(content string) string {
	// Split into lines and trim trailing whitespace from each line
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Join back and ensure exactly one trailing newline if content is not empty
	normalized := strings.Join(lines, "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if len(normalized) > 0 {
		normalized += "\n"
	}

	return normalized
}

// hasLocalModifications checks if the local workflow file has been modified from its source
// It resolves the source field and imports on the remote content, then compares with local
// Note: stop-after field is ignored during comparison as it's a deployment-specific setting
func hasLocalModifications(sourceContent, localContent, sourceSpec string, verbose bool) bool {
	// Normalize both contents
	sourceNormalized := normalizeWhitespace(sourceContent)
	localNormalized := normalizeWhitespace(localContent)

	// Remove stop-after field from both contents for comparison
	// This field is deployment-specific and should not trigger "local modifications" warnings
	sourceNormalized, _ = RemoveFieldFromOnTrigger(sourceNormalized, "stop-after")
	localNormalized, _ = RemoveFieldFromOnTrigger(localNormalized, "stop-after")

	// Parse the source spec to get repo and ref information
	parsedSourceSpec, err := parseSourceSpec(sourceSpec)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to parse source spec: %v", err)))
		}
		// Fall back to simple comparison
		return sourceNormalized != localNormalized
	}

	// Add the source field to the remote content
	sourceWithSource, err := UpdateFieldInFrontmatter(sourceNormalized, "source", sourceSpec)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to add source field to remote content: %v", err)))
		}
		// Fall back to simple comparison
		return sourceNormalized != localNormalized
	}

	// Resolve imports on the remote content
	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: parsedSourceSpec.Repo,
			Version:  parsedSourceSpec.Ref,
		},
		WorkflowPath: parsedSourceSpec.Path,
	}

	sourceResolved, err := processIncludesInContent(sourceWithSource, workflow, parsedSourceSpec.Ref, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to process imports on remote content: %v", err)))
		}
		// Use the version with source field but without resolved imports
		sourceResolved = sourceWithSource
	}

	// Normalize again after processing
	sourceResolvedNormalized := normalizeWhitespace(sourceResolved)

	// Compare the normalized contents
	hasModifications := sourceResolvedNormalized != localNormalized

	if verbose && hasModifications {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Local modifications detected"))
	}

	return hasModifications
}

// MergeWorkflowContent performs a 3-way merge of workflow content using git merge-file
// It returns the merged content, whether conflicts exist, and any error
func MergeWorkflowContent(base, current, new, oldSourceSpec, newRef string, verbose bool) (string, bool, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Performing 3-way merge using git merge-file"))
	}

	// Parse the old source spec to get the current ref
	sourceSpec, err := parseSourceSpec(oldSourceSpec)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse source spec: %w", err)
	}
	currentSourceSpec := fmt.Sprintf("%s/%s@%s", sourceSpec.Repo, sourceSpec.Path, sourceSpec.Ref)

	// Fix the base version by adding the source field to match what both current and new have
	// This prevents unnecessary conflicts over the source field
	baseWithSource, err := UpdateFieldInFrontmatter(base, "source", currentSourceSpec)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to add source to base content: %v", err)))
		}
		// Continue with original base content
		baseWithSource = base
	}

	// Update the source field in the new content with the new ref
	newWithUpdatedSource, err := UpdateFieldInFrontmatter(new, "source", fmt.Sprintf("%s/%s@%s", sourceSpec.Repo, sourceSpec.Path, newRef))
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update source in new content: %v", err)))
		}
		// Continue with original new content
		newWithUpdatedSource = new
	}

	// Normalize whitespace in all three versions to reduce spurious conflicts
	baseNormalized := normalizeWhitespace(baseWithSource)
	currentNormalized := normalizeWhitespace(current)
	newNormalized := normalizeWhitespace(newWithUpdatedSource)

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

	if err := os.WriteFile(baseFile, []byte(baseNormalized), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write base file: %w", err)
	}
	if err := os.WriteFile(currentFile, []byte(currentNormalized), 0644); err != nil {
		return "", false, fmt.Errorf("failed to write current file: %w", err)
	}
	if err := os.WriteFile(newFile, []byte(newNormalized), 0644); err != nil {
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
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Actions lock file not found: %s", actionsLockPath)))
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

		// Check for latest release
		latestVersion, latestSHA, err := getLatestActionRelease(entry.Repo, entry.Version, allowMajor, verbose)
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

		// Update the map entry
		actionsLock.Entries[key] = actionsLockEntry{
			Repo:    entry.Repo,
			Version: latestVersion,
			SHA:     latestSHA,
		}

		updatedActions = append(updatedActions, entry.Repo)
	}

	// Show summary
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("=== Actions Update Summary ==="))
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

	// Use gh CLI to get releases
	cmd := workflow.ExecGH("api", fmt.Sprintf("/repos/%s/releases", repo), "--jq", ".[].tag_name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if gitutil.IsAuthError(outputStr) || gitutil.IsAuthError(err.Error()) {
			updateLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s", repo)
			// Try fallback using git ls-remote
			latestRelease, latestSHA, gitErr := getLatestActionReleaseViaGit(repo, currentVersion, allowMajor, verbose)
			if gitErr != nil {
				return "", "", fmt.Errorf("failed to fetch releases via GitHub API and git: API error: %w, Git error: %v", err, gitErr)
			}
			return latestRelease, latestSHA, nil
		}
		return "", "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(releases) == 0 || releases[0] == "" {
		return "", "", fmt.Errorf("no releases found")
	}

	// Parse current version
	currentVer := parseVersion(currentVersion)
	if currentVer == nil {
		// If current version is not a valid semantic version, just return the latest release
		latestRelease := releases[0]
		sha, err := getActionSHAForTag(repo, latestRelease)
		if err != nil {
			return "", "", fmt.Errorf("failed to get SHA for %s: %w", latestRelease, err)
		}
		return latestRelease, sha, nil
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
		return "", "", fmt.Errorf("no compatible release found")
	}

	// Get the SHA for the latest compatible release
	sha, err := getActionSHAForTag(repo, latestCompatible)
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

	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

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
		return "", "", fmt.Errorf("no releases found")
	}

	// Parse current version
	currentVer := parseVersion(currentVersion)
	if currentVer == nil {
		// If current version is not a valid semantic version, just return the first release
		latestRelease := releases[0]
		sha := tagToSHA[latestRelease]
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current version is not valid, using first release: %s (via git)", latestRelease)))
		}
		return latestRelease, sha, nil
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
		return "", "", fmt.Errorf("no compatible release found")
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
	cmd := workflow.ExecGH("api", fmt.Sprintf("/repos/%s/git/ref/tags/%s", repo, tag), "--jq", ".object.sha")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve tag: %w", err)
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", fmt.Errorf("empty SHA returned for tag")
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

	// Manually construct JSON with sorted keys using proper JSON marshaling for values
	var result []byte
	result = append(result, []byte("{\n  \"entries\": {\n")...)

	for i, key := range keys {
		entry := actionsLock.Entries[key]

		// Marshal the key using proper JSON encoding
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}

		// Marshal individual fields using proper JSON encoding
		repoJSON, err := json.Marshal(entry.Repo)
		if err != nil {
			return nil, err
		}
		versionJSON, err := json.Marshal(entry.Version)
		if err != nil {
			return nil, err
		}
		shaJSON, err := json.Marshal(entry.SHA)
		if err != nil {
			return nil, err
		}

		// Construct the entry JSON with properly escaped values
		entryJSON := fmt.Sprintf("    %s: {\n      \"repo\": %s,\n      \"version\": %s,\n      \"sha\": %s\n    }",
			string(keyJSON), string(repoJSON), string(versionJSON), string(shaJSON))

		result = append(result, []byte(entryJSON)...)

		// Add comma if not the last entry
		if i < len(keys)-1 {
			result = append(result, ',')
		}
		result = append(result, '\n')
	}

	result = append(result, []byte("  }\n}")...)
	return result, nil
}
