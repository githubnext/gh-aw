package cli

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/stringutil"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var addLog = logger.New("cli:add_command")

// NewAddCommand creates the add command
func NewAddCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <workflow>...",
		Short: "Add agentic workflows from repositories to .github/workflows",
		Long: `Add one or more workflows from repositories to .github/workflows.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics                           # List available workflows
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor                # Add specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor@v1.0.0         # Add with version
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/workflows/ci-doctor.md@main
  ` + string(constants.CLIExtensionPrefix) + ` add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --create-pull-request --force
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/*
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/*@v1.0.0
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --dir shared   # Add to .github/workflows/shared/

Workflow specifications:
  - Two parts: "owner/repo[@version]" (lists available workflows in the repository)
  - Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)
  - Four+ parts: "owner/repo/workflows/workflow-name.md[@version]" (requires explicit .md extension)
  - GitHub URL: "https://github.com/owner/repo/blob/branch/path/to/workflow.md"
  - Wildcard: "owner/repo/*[@version]" (adds all workflows from the repository)
  - Version can be tag, branch, or SHA

The -n flag allows you to specify a custom name for the workflow file (only applies to the first workflow when adding multiple).
The --dir flag allows you to specify a subdirectory under .github/workflows/ where the workflow will be added.
The --create-pull-request flag (or --pr) automatically creates a pull request with the workflow changes.
The --force flag overwrites existing workflow files.

Note: To create a new workflow from scratch, use the 'new' command instead.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflows := args
			numberFlag, _ := cmd.Flags().GetInt("number")
			engineOverride, _ := cmd.Flags().GetString("engine")
			nameFlag, _ := cmd.Flags().GetString("name")
			createPRFlag, _ := cmd.Flags().GetBool("create-pull-request")
			prFlagAlias, _ := cmd.Flags().GetBool("pr")
			prFlag := createPRFlag || prFlagAlias // Support both --create-pull-request and --pr
			forceFlag, _ := cmd.Flags().GetBool("force")
			appendText, _ := cmd.Flags().GetString("append")
			verbose, _ := cmd.Flags().GetBool("verbose")
			noGitattributes, _ := cmd.Flags().GetBool("no-gitattributes")
			workflowDir, _ := cmd.Flags().GetString("dir")
			noStopAfter, _ := cmd.Flags().GetBool("no-stop-after")
			stopAfter, _ := cmd.Flags().GetString("stop-after")

			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			// Handle normal mode
			if prFlag {
				return AddWorkflows(workflows, numberFlag, verbose, engineOverride, nameFlag, forceFlag, appendText, true, noGitattributes, workflowDir, noStopAfter, stopAfter)
			} else {
				return AddWorkflows(workflows, numberFlag, verbose, engineOverride, nameFlag, forceFlag, appendText, false, noGitattributes, workflowDir, noStopAfter, stopAfter)
			}
		},
	}

	// Add number flag to add command
	cmd.Flags().Int("number", 1, "Create multiple numbered copies")

	// Add name flag to add command
	cmd.Flags().StringP("name", "n", "", "Specify name for the added workflow (without .md extension)")

	// Add AI flag to add command
	addEngineFlag(cmd)

	// Add repository flag to add command
	cmd.Flags().StringP("repo", "r", "", "Source repository containing workflows (owner/repo format)")

	// Add PR flag to add command (--create-pull-request with --pr as alias)
	cmd.Flags().Bool("create-pull-request", false, "Create a pull request with the workflow changes")
	cmd.Flags().Bool("pr", false, "Alias for --create-pull-request")
	_ = cmd.Flags().MarkHidden("pr") // Hide the short alias from help output

	// Add force flag to add command
	cmd.Flags().BoolP("force", "f", false, "Overwrite existing workflow files without confirmation")

	// Add append flag to add command
	cmd.Flags().String("append", "", "Append extra content to the end of agentic workflow on installation")

	// Add no-gitattributes flag to add command
	cmd.Flags().Bool("no-gitattributes", false, "Skip updating .gitattributes file")

	// Add workflow directory flag to add command
	cmd.Flags().StringP("dir", "d", "", "Subdirectory under .github/workflows/ (e.g., 'shared' creates .github/workflows/shared/)")

	// Add no-stop-after flag to add command
	cmd.Flags().Bool("no-stop-after", false, "Remove any stop-after field from the workflow")

	// Add stop-after flag to add command
	cmd.Flags().String("stop-after", "", "Override stop-after value in the workflow (e.g., '+48h', '2025-12-31 23:59:59')")

	// Register completions for add command
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// AddWorkflows adds one or more workflows from components to .github/workflows
// with optional repository installation and PR creation
func AddWorkflows(workflows []string, number int, verbose bool, engineOverride string, name string, force bool, appendText string, createPR bool, noGitattributes bool, workflowDir string, noStopAfter bool, stopAfter string) error {
	addLog.Printf("Adding workflows: count=%d, engineOverride=%s, createPR=%v, noGitattributes=%v, workflowDir=%s, noStopAfter=%v, stopAfter=%s", len(workflows), engineOverride, createPR, noGitattributes, workflowDir, noStopAfter, stopAfter)

	if len(workflows) == 0 {
		return fmt.Errorf("at least one workflow name is required")
	}

	for i, workflow := range workflows {
		if workflow == "" {
			return fmt.Errorf("workflow name cannot be empty (workflow %d)", i+1)
		}
	}

	// Check if this is a repo-only specification (owner/repo instead of owner/repo/workflow)
	// If so, list available workflows and exit
	if len(workflows) == 1 && isRepoOnlySpec(workflows[0]) {
		return handleRepoOnlySpec(workflows[0], verbose)
	}

	// If creating a PR, check prerequisites
	if createPR {
		// Check if GitHub CLI is available
		if !isGHCLIAvailable() {
			return fmt.Errorf("GitHub CLI (gh) is required for PR creation but not available")
		}

		// Check if we're in a git repository
		if !isGitRepo() {
			return fmt.Errorf("not in a git repository - PR creation requires a git repository")
		}

		// Check no other changes are present
		if err := checkCleanWorkingDirectory(verbose); err != nil {
			return fmt.Errorf("working directory is not clean: %w", err)
		}
	}

	// Parse workflow specifications and group by repository
	repoVersions := make(map[string]string) // repo -> version
	processedWorkflows := []*WorkflowSpec{} // List of processed workflow specs

	for _, workflow := range workflows {
		spec, err := parseWorkflowSpec(workflow)
		if err != nil {
			return fmt.Errorf("invalid workflow specification '%s': %w", workflow, err)
		}

		// Handle repository installation and workflow name extraction
		if existing, exists := repoVersions[spec.RepoSlug]; exists && existing != spec.Version {
			return fmt.Errorf("conflicting versions for repository %s: %s vs %s", spec.RepoSlug, existing, spec.Version)
		}
		repoVersions[spec.RepoSlug] = spec.Version

		// Create qualified name for processing
		processedWorkflows = append(processedWorkflows, spec)
	}

	// Check if any workflow is from the current repository
	// Skip this check if we can't determine the current repository (e.g., not in a git repo)
	currentRepoSlug, repoErr := GetCurrentRepoSlug()
	if repoErr == nil {
		// We successfully determined the current repository, check all workflow specs
		for _, spec := range processedWorkflows {
			// Skip local workflow specs (starting with "./")
			if strings.HasPrefix(spec.WorkflowPath, "./") {
				continue
			}

			if spec.RepoSlug == currentRepoSlug {
				return fmt.Errorf("cannot add workflows from the current repository (%s). The 'add' command is for installing workflows from other repositories", currentRepoSlug)
			}
		}
	}
	// If we can't determine the current repository, proceed without the check

	// Install required repositories
	for repo, version := range repoVersions {
		repoWithVersion := repo
		if version != "" {
			repoWithVersion = fmt.Sprintf("%s@%s", repo, version)
		}

		addLog.Printf("Installing repository: %s", repoWithVersion)

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing repository %s before adding workflows...", repoWithVersion)))
		}

		// Install as global package (not local) to match the behavior expected
		if err := InstallPackage(repoWithVersion, verbose); err != nil {
			addLog.Printf("Failed to install repository %s: %v", repoWithVersion, err)
			return fmt.Errorf("failed to install repository %s: %w", repoWithVersion, err)
		}
	}

	// Check if any workflow specs contain wildcards before expansion
	hasWildcard := false
	for _, spec := range processedWorkflows {
		if spec.IsWildcard {
			hasWildcard = true
			break
		}
	}

	// Expand wildcards after installation
	var err error
	processedWorkflows, err = expandWildcardWorkflows(processedWorkflows, verbose)
	if err != nil {
		return err
	}

	// Handle PR creation workflow
	if createPR {
		addLog.Print("Creating workflow with PR")
		return addWorkflowsWithPR(processedWorkflows, number, verbose, engineOverride, name, force, appendText, noGitattributes, hasWildcard, workflowDir, noStopAfter, stopAfter)
	}

	// Handle normal workflow addition
	addLog.Print("Adding workflows normally without PR")
	return addWorkflowsNormal(processedWorkflows, number, verbose, engineOverride, name, force, appendText, noGitattributes, hasWildcard, workflowDir, noStopAfter, stopAfter)
}

// handleRepoOnlySpec handles the case when user provides only owner/repo without workflow name
// It installs the package and lists available workflows with interactive selection
func handleRepoOnlySpec(repoSpec string, verbose bool) error {
	addLog.Printf("Handling repo-only specification: %s", repoSpec)

	// Parse the repository specification to extract repo slug and version
	spec, err := parseRepoSpec(repoSpec)
	if err != nil {
		return fmt.Errorf("invalid repository specification '%s': %w", repoSpec, err)
	}

	// Install the repository
	repoWithVersion := spec.RepoSlug
	if spec.Version != "" {
		repoWithVersion = fmt.Sprintf("%s@%s", spec.RepoSlug, spec.Version)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing repository %s...", repoWithVersion)))
	}

	if err := InstallPackage(repoWithVersion, verbose); err != nil {
		return fmt.Errorf("failed to install repository %s: %w", repoWithVersion, err)
	}

	// List workflows in the installed package with metadata
	workflows, err := listWorkflowsWithMetadata(spec.RepoSlug, verbose)
	if err != nil {
		return fmt.Errorf("failed to list workflows in %s: %w", spec.RepoSlug, err)
	}

	// Display the list of available workflows
	if len(workflows) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No workflows found in repository %s", spec.RepoSlug)))
		return nil
	}

	// Try interactive selection first
	selected, err := showInteractiveWorkflowSelection(spec.RepoSlug, workflows, spec.Version, verbose)
	if err == nil && selected != "" {
		// User selected a workflow, proceed to add it
		addLog.Printf("User selected workflow: %s", selected)
		return nil // Successfully displayed and allowed selection
	}

	// If interactive selection failed or was cancelled, fall back to table display
	addLog.Printf("Interactive selection failed or cancelled, showing table: %v", err)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Available workflows in %s:", spec.RepoSlug)))
	fmt.Fprintln(os.Stderr, "")

	// Render workflows as a table using console helpers
	fmt.Fprint(os.Stderr, console.RenderStruct(workflows))

	fmt.Fprintln(os.Stderr, "Example:")
	fmt.Fprintln(os.Stderr, "")

	// Show example with first workflow
	exampleSpec := fmt.Sprintf("%s/%s", spec.RepoSlug, workflows[0].ID)
	if spec.Version != "" {
		exampleSpec += "@" + spec.Version
	}

	fmt.Fprintf(os.Stderr, "  %s add %s\n", string(constants.CLIExtensionPrefix), exampleSpec)
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// showInteractiveWorkflowSelection displays an interactive list of workflows
// and allows the user to select one
func showInteractiveWorkflowSelection(repoSlug string, workflows []WorkflowInfo, version string, verbose bool) (string, error) {
	addLog.Printf("Showing interactive workflow selection: repo=%s, workflows=%d", repoSlug, len(workflows))

	// Convert WorkflowInfo to ListItems
	items := make([]console.ListItem, len(workflows))
	for i, wf := range workflows {
		items[i] = console.NewListItem(wf.Name, wf.Description, wf.ID)
	}

	// Show interactive list
	title := fmt.Sprintf("Select a workflow from %s:", repoSlug)
	selectedID, err := console.ShowInteractiveList(title, items)
	if err != nil {
		return "", err
	}

	// Build the workflow spec
	workflowSpec := fmt.Sprintf("%s/%s", repoSlug, selectedID)
	if version != "" {
		workflowSpec += "@" + version
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To add this workflow, run:"))
	fmt.Fprintf(os.Stderr, "  %s add %s\n", string(constants.CLIExtensionPrefix), workflowSpec)
	fmt.Fprintln(os.Stderr, "")

	return selectedID, nil
}

// displayAvailableWorkflows lists available workflows from an installed package
// with interactive selection when in TTY mode
func displayAvailableWorkflows(repoSlug, version string, verbose bool) error {
	addLog.Printf("Displaying available workflows for repository: %s", repoSlug)

	// List workflows in the installed package with metadata
	workflows, err := listWorkflowsWithMetadata(repoSlug, verbose)
	if err != nil {
		return fmt.Errorf("failed to list workflows in %s: %w", repoSlug, err)
	}

	// Display the list of available workflows
	if len(workflows) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No workflows found in repository %s", repoSlug)))
		return nil
	}

	// Try interactive selection first
	_, err = showInteractiveWorkflowSelection(repoSlug, workflows, version, verbose)
	if err == nil {
		// Successfully displayed and allowed selection
		return nil
	}

	// If interactive selection failed or was cancelled, fall back to table display
	addLog.Printf("Interactive selection failed or cancelled, showing table: %v", err)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Available workflows in %s:", repoSlug)))
	fmt.Fprintln(os.Stderr, "")

	// Render workflows as a table using console helpers
	fmt.Fprint(os.Stderr, console.RenderStruct(workflows))

	fmt.Fprintln(os.Stderr, "Example:")
	fmt.Fprintln(os.Stderr, "")

	// Show example with first workflow
	exampleSpec := fmt.Sprintf("%s/%s", repoSlug, workflows[0].ID)
	if version != "" {
		exampleSpec += "@" + version
	}

	fmt.Fprintf(os.Stderr, "  %s add %s\n", string(constants.CLIExtensionPrefix), exampleSpec)
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// addWorkflowsNormal handles normal workflow addition without PR creation
func addWorkflowsNormal(workflows []*WorkflowSpec, number int, verbose bool, engineOverride string, name string, force bool, appendText string, noGitattributes bool, fromWildcard bool, workflowDir string, noStopAfter bool, stopAfter string) error {
	// Create file tracker for all operations
	tracker, err := NewFileTracker()
	if err != nil {
		// If we can't create a tracker (e.g., not in git repo), fall back to non-tracking behavior
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not create file tracker: %v", err)))
		}
		tracker = nil
	}

	// Ensure .gitattributes is configured unless flag is set
	if !noGitattributes {
		addLog.Print("Configuring .gitattributes")
		if err := ensureGitAttributes(); err != nil {
			addLog.Printf("Failed to configure .gitattributes: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
			}
			// Don't fail the entire operation if gitattributes update fails
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
		}
	}

	if len(workflows) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding %d workflow(s)...", len(workflows))))
	}

	// Add each workflow
	for i, workflow := range workflows {
		if len(workflows) > 1 {
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Adding workflow %d/%d: %s", i+1, len(workflows), workflow.WorkflowName)))
		}

		// For multiple workflows, only use the name flag for the first one
		currentName := ""
		if i == 0 && name != "" {
			currentName = name
		}

		if err := addWorkflowWithTracking(workflow, number, verbose, engineOverride, currentName, force, appendText, tracker, fromWildcard, workflowDir, noStopAfter, stopAfter); err != nil {
			return fmt.Errorf("failed to add workflow '%s': %w", workflow.String(), err)
		}
	}

	if len(workflows) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully added all %d workflows", len(workflows))))
	}

	return nil
}

// addWorkflowsWithPR handles workflow addition with PR creation
func addWorkflowsWithPR(workflows []*WorkflowSpec, number int, verbose bool, engineOverride string, name string, force bool, appendText string, noGitattributes bool, fromWildcard bool, workflowDir string, noStopAfter bool, stopAfter string) error {
	// Get current branch for restoration later
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Create temporary branch with random 4-digit number
	randomNum := rand.Intn(9000) + 1000 // Generate number between 1000-9999
	branchName := fmt.Sprintf("add-workflow-%s-%04d", strings.ReplaceAll(workflows[0].WorkflowPath, "/", "-"), randomNum)

	if err := createAndSwitchBranch(branchName, verbose); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Create file tracker for rollback capability
	tracker, err := NewFileTracker()
	if err != nil {
		return fmt.Errorf("failed to create file tracker: %w", err)
	}

	// Ensure we switch back to original branch on exit
	defer func() {
		if switchErr := switchBranch(currentBranch, verbose); switchErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to switch back to branch %s: %v", currentBranch, switchErr)))
		}
	}()

	// Add workflows using the normal function logic
	if err := addWorkflowsNormal(workflows, number, verbose, engineOverride, name, force, appendText, noGitattributes, fromWildcard, workflowDir, noStopAfter, stopAfter); err != nil {
		// Rollback on error
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to add workflows: %w", err)
	}

	// Stage all files before creating PR
	if err := tracker.StageAllFiles(verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to stage workflow files: %w", err)
	}

	// Update .gitattributes and stage it if modified
	if err := stageGitAttributesIfChanged(); err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to stage .gitattributes: %v", err)))
	}

	// Commit changes
	var commitMessage, prTitle, prBody, joinedNames string
	if len(workflows) == 1 {
		joinedNames = workflows[0].WorkflowName
		commitMessage = fmt.Sprintf("Add agentic workflow %s", joinedNames)
		prTitle = fmt.Sprintf("Add agentic workflow %s", joinedNames)
		prBody = fmt.Sprintf("Add agentic workflow %s", joinedNames)
	} else {
		// Get workflow.Workflo
		workflowNames := make([]string, len(workflows))
		for i, wf := range workflows {
			workflowNames[i] = wf.WorkflowName
		}
		joinedNames = strings.Join(workflowNames, ", ")
		commitMessage = fmt.Sprintf("Add agentic workflows: %s", joinedNames)
		prTitle = fmt.Sprintf("Add agentic workflows: %s", joinedNames)
		prBody = fmt.Sprintf("Add agentic workflows: %s", joinedNames)
	}

	if err := commitChanges(commitMessage, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to commit files: %w", err)
	}

	// Push branch
	if err := pushBranch(branchName, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	// Create PR
	if err := createPR(branchName, prTitle, prBody, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// Success - no rollback needed

	// Switch back to original branch
	if err := switchBranch(currentBranch, verbose); err != nil {
		return fmt.Errorf("failed to switch back to branch %s: %w", currentBranch, err)
	}

	if len(workflows) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created PR for workflow: %s", workflows[0].WorkflowName)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created PR for workflows: %s", joinedNames)))
	}
	return nil
}

// addWorkflowWithTracking adds a workflow from components to .github/workflows with file tracking
func addWorkflowWithTracking(workflow *WorkflowSpec, number int, verbose bool, engineOverride string, name string, force bool, appendText string, tracker *FileTracker, fromWildcard bool, workflowDir string, noStopAfter bool, stopAfter string) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding workflow: %s", workflow.String())))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Number of copies: %d", number)))
		if force {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Force flag enabled: will overwrite existing files"))
		}
	}

	// Validate number of copies
	if number < 1 {
		return fmt.Errorf("number of copies must be a positive integer")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, "Locating workflow components...")
	}

	workflowPath := workflow.WorkflowPath

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Looking for workflow file: %s", workflowPath)))
	}

	// Try to read the workflow content from multiple sources
	sourceContent, sourceInfo, err := findWorkflowInPackageForRepo(workflow, verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Workflow '%s' not found.", workflowPath)))

		// Try to list available workflows from the installed package
		if err := displayAvailableWorkflows(workflow.RepoSlug, workflow.Version, verbose); err != nil {
			// If we can't list workflows, provide generic help
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To add workflows to your project:"))
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Use the 'add' command with repository/workflow specifications:"))
			fmt.Fprintf(os.Stderr, "  %s add owner/repo/workflow-name\n", string(constants.CLIExtensionPrefix))
			fmt.Fprintf(os.Stderr, "  %s add owner/repo/workflow-name@version\n", string(constants.CLIExtensionPrefix))
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Example:"))
			fmt.Fprintf(os.Stderr, "  %s add githubnext/agentics/ci-doctor\n", string(constants.CLIExtensionPrefix))
			fmt.Fprintf(os.Stderr, "  %s add githubnext/agentics/daily-plan@main\n", string(constants.CLIExtensionPrefix))
		}

		return fmt.Errorf("workflow not found: %s", workflowPath)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Read workflow content (%d bytes)", len(sourceContent))))
	}

	// Find git root to ensure consistent placement
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("add workflow requires being in a git repository: %w", err)
	}

	// Determine the target workflow directory
	var githubWorkflowsDir string
	if workflowDir != "" {
		// Validate that the path is relative
		if filepath.IsAbs(workflowDir) {
			return fmt.Errorf("workflow directory must be a relative path, got: %s", workflowDir)
		}
		// Clean the path to avoid issues with ".." or other problematic elements
		workflowDir = filepath.Clean(workflowDir)
		// Ensure the path is under .github/workflows
		if !strings.HasPrefix(workflowDir, ".github/workflows") {
			// If user provided a subdirectory name, prepend .github/workflows/
			githubWorkflowsDir = filepath.Join(gitRoot, ".github/workflows", workflowDir)
		} else {
			githubWorkflowsDir = filepath.Join(gitRoot, workflowDir)
		}
	} else {
		// Use default .github/workflows directory
		githubWorkflowsDir = filepath.Join(gitRoot, ".github/workflows")
	}

	// Ensure the target directory exists
	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow directory %s: %w", githubWorkflowsDir, err)
	}

	// Determine the workflowName to use
	var workflowName string
	if name != "" {
		// Use the explicitly provided name
		workflowName = name
	} else {
		// Extract filename from workflow path and remove .md extension for processing
		workflowName = workflow.WorkflowName
	}

	// Check if a workflow with this name already exists
	existingFile := filepath.Join(githubWorkflowsDir, workflowName+".md")
	if _, err := os.Stat(existingFile); err == nil && !force {
		// When adding with wildcard, emit warning and skip instead of error
		if fromWildcard {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow '%s' already exists in .github/workflows/. Skipping.", workflowName)))
			return nil
		}
		return fmt.Errorf("workflow '%s' already exists in .github/workflows/. Use a different name with -n flag, remove the existing workflow first, or use --force to overwrite", workflowName)
	}

	// Collect all @include dependencies from the workflow file
	includeDeps, err := collectPackageIncludeDependencies(string(sourceContent), sourceInfo.PackagePath, verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to collect include dependencies: %v", err)))
	}

	// Copy all @include dependencies to .github/workflows maintaining relative paths
	if err := copyIncludeDependenciesFromPackageWithForce(includeDeps, githubWorkflowsDir, verbose, force, tracker); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy include dependencies: %v", err)))
	}

	// Process each copy
	for i := 1; i <= number; i++ {
		// Construct the destination file path with numbering in .github/workflows
		var destFile string
		if number == 1 {
			destFile = filepath.Join(githubWorkflowsDir, workflowName+".md")
		} else {
			destFile = filepath.Join(githubWorkflowsDir, fmt.Sprintf("%s-%d.md", workflowName, i))
		}

		// Check if destination file already exists
		fileExists := false
		if _, err := os.Stat(destFile); err == nil {
			fileExists = true
			if !force {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Destination file '%s' already exists, skipping.", destFile)))
				continue
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Overwriting existing file: %s", destFile)))
		}

		// Process content for numbered workflows
		content := string(sourceContent)
		if number > 1 {
			// Update H1 title to include number
			content = updateWorkflowTitle(content, i)
		}

		// Add source field to frontmatter
		sourceString := buildSourceStringWithCommitSHA(workflow, sourceInfo.CommitSHA)
		if sourceString != "" {
			updatedContent, err := addSourceToWorkflow(content, sourceString)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to add source field: %v", err)))
				}
			} else {
				content = updatedContent
			}

			// Process imports field and replace with workflowspec
			processedImportsContent, err := processImportsWithWorkflowSpec(content, workflow, sourceInfo.CommitSHA, verbose)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process imports: %v", err)))
				}
			} else {
				content = processedImportsContent
			}

			// Process @include directives and replace with workflowspec
			processedContent, err := processIncludesWithWorkflowSpec(content, workflow, sourceInfo.CommitSHA, sourceInfo.PackagePath, verbose)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
				}
			} else {
				content = processedContent
			}
		}

		// Handle stop-after field modifications
		if noStopAfter {
			// Remove stop-after field if requested
			cleanedContent, err := RemoveFieldFromOnTrigger(content, "stop-after")
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove stop-after field: %v", err)))
				}
			} else {
				content = cleanedContent
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed stop-after field from workflow"))
				}
			}
		} else if stopAfter != "" {
			// Set custom stop-after value if provided
			updatedContent, err := SetFieldInOnTrigger(content, "stop-after", stopAfter)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set stop-after field: %v", err)))
				}
			} else {
				content = updatedContent
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Set stop-after field to: %s", stopAfter)))
				}
			}
		}

		// Append text if provided
		if appendText != "" {
			// Ensure we have a newline before appending
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += "\n" + appendText
		}

		// Track the file based on whether it existed before (if tracker is available)
		if tracker != nil {
			if fileExists {
				tracker.TrackModified(destFile)
			} else {
				tracker.TrackCreated(destFile)
			}
		}

		// Write the file with restrictive permissions (0600) to follow security best practices
		if err := os.WriteFile(destFile, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write destination file '%s': %w", destFile, err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Added workflow: %s", destFile)))

		// Extract and display description if present
		if description := ExtractWorkflowDescription(content); description != "" {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(description))
			fmt.Fprintln(os.Stderr, "")
		}

		// Try to compile the workflow and track generated files
		if tracker != nil {
			if err := compileWorkflowWithTracking(destFile, verbose, engineOverride, tracker); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			}
		} else {
			// Fall back to basic compilation without tracking
			if err := compileWorkflow(destFile, verbose, engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			}
		}
	}

	// Stage tracked files to git if in a git repository
	if isGitRepo() && tracker != nil {
		if err := tracker.StageAllFiles(verbose); err != nil {
			return fmt.Errorf("failed to stage workflow files: %w", err)
		}
	}

	return nil
}

func updateWorkflowTitle(content string, number int) string {
	// Find and update the first H1 header
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			// Extract the title part and add number
			title := strings.TrimSpace(line[2:])
			lines[i] = fmt.Sprintf("# %s %d", title, number)
			break
		}
	}
	return strings.Join(lines, "\n")
}

func compileWorkflow(filePath string, verbose bool, engineOverride string) error {
	return compileWorkflowWithRefresh(filePath, verbose, engineOverride, false)
}

func compileWorkflowWithRefresh(filePath string, verbose bool, engineOverride string, refreshStopTime bool) error {
	// Create compiler and compile the workflow
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())
	compiler.SetRefreshStopTime(refreshStopTime)
	if err := CompileWorkflowWithValidation(compiler, filePath, verbose, false, false, false, false, false); err != nil {
		return err
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
		}
	}

	// Note: Instructions are only written when explicitly requested via the compile command flag
	// This helper function is used in contexts where instructions should not be automatically written

	return nil
}

// compileWorkflowWithTracking compiles a workflow and tracks generated files
func compileWorkflowWithTracking(filePath string, verbose bool, engineOverride string, tracker *FileTracker) error {
	return compileWorkflowWithTrackingAndRefresh(filePath, verbose, engineOverride, tracker, false)
}

func compileWorkflowWithTrackingAndRefresh(filePath string, verbose bool, engineOverride string, tracker *FileTracker, refreshStopTime bool) error {
	// Generate the expected lock file path
	lockFile := stringutil.MarkdownToLockFile(filePath)

	// Check if lock file exists before compilation
	lockFileExists := false
	if _, err := os.Stat(lockFile); err == nil {
		lockFileExists = true
	}

	// Check if .gitattributes exists before ensuring it
	gitRoot, err := findGitRoot()
	if err != nil {
		return err
	}
	gitAttributesPath := filepath.Join(gitRoot, ".gitattributes")
	gitAttributesExists := false
	if _, err := os.Stat(gitAttributesPath); err == nil {
		gitAttributesExists = true
	}

	// Track the lock file before compilation
	if lockFileExists {
		tracker.TrackModified(lockFile)
	} else {
		tracker.TrackCreated(lockFile)
	}

	// Track .gitattributes file before modification
	if gitAttributesExists {
		tracker.TrackModified(gitAttributesPath)
	} else {
		tracker.TrackCreated(gitAttributesPath)
	}

	// Create compiler and set the file tracker
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())
	compiler.SetFileTracker(tracker)
	compiler.SetRefreshStopTime(refreshStopTime)
	if err := CompileWorkflowWithValidation(compiler, filePath, verbose, false, false, false, false, false); err != nil {
		return err
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
		}
	}

	return nil
}

// addSourceToWorkflow adds the source field to the workflow's frontmatter
func addSourceToWorkflow(content, source string) (string, error) {
	// Use shared frontmatter logic that preserves formatting
	return addFieldToFrontmatter(content, "source", source)
}

// expandWildcardWorkflows expands wildcard workflow specifications into individual workflow specs.
// For each wildcard spec, it discovers all workflows in the installed package and replaces
// the wildcard with the discovered workflows. Non-wildcard specs are passed through unchanged.
func expandWildcardWorkflows(specs []*WorkflowSpec, verbose bool) ([]*WorkflowSpec, error) {
	expandedWorkflows := []*WorkflowSpec{}

	for _, spec := range specs {
		if spec.IsWildcard {
			addLog.Printf("Expanding wildcard for repository: %s", spec.RepoSlug)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Discovering workflows in %s...", spec.RepoSlug)))
			}

			discovered, err := discoverWorkflowsInPackage(spec.RepoSlug, spec.Version, verbose)
			if err != nil {
				return nil, fmt.Errorf("failed to discover workflows in %s: %w", spec.RepoSlug, err)
			}

			if len(discovered) == 0 {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No workflows found in %s", spec.RepoSlug)))
			} else {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %d workflow(s) in %s", len(discovered), spec.RepoSlug)))
				}
				expandedWorkflows = append(expandedWorkflows, discovered...)
			}
		} else {
			expandedWorkflows = append(expandedWorkflows, spec)
		}
	}

	if len(expandedWorkflows) == 0 {
		return nil, fmt.Errorf("no workflows to add after expansion")
	}

	return expandedWorkflows, nil
}
