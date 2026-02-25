package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/tty"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var addLog = logger.New("cli:add_command")

// AddOptions contains all configuration options for adding workflows
type AddOptions struct {
	Verbose                bool
	Quiet                  bool
	EngineOverride         string
	Name                   string
	Force                  bool
	AppendText             string
	CreatePR               bool
	Push                   bool
	NoGitattributes        bool
	FromWildcard           bool
	WorkflowDir            string
	NoStopAfter            bool
	StopAfter              string
	DisableSecurityScanner bool
}

// AddWorkflowsResult contains the result of adding workflows
type AddWorkflowsResult struct {
	// PRNumber is the PR number if a PR was created, or 0 if no PR was created
	PRNumber int
	// PRURL is the URL of the created PR, or empty if no PR was created
	PRURL string
	// HasWorkflowDispatch is true if any of the added workflows has a workflow_dispatch trigger
	HasWorkflowDispatch bool
}

// NewAddCommand creates the add command
func NewAddCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <workflow>...",
		Aliases: []string{"add-wizard"},
		Short:   "Add agentic workflows from repositories to .github/workflows",
		Long: `Add one or more workflows from repositories to .github/workflows.

By default, this command runs in interactive mode, which guides you through:
  - Selecting an AI engine (Copilot, Claude, or Codex)
  - Configuring API keys and secrets
  - Creating a pull request with the workflow
  - Optionally running the workflow

Use --non-interactive to skip the guided setup and add workflows directly.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/daily-repo-status        # Interactive setup (recommended)
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --non-interactive  # Skip interactive mode
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor@v1.0.0         # Add with version
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/workflows/ci-doctor.md@main
  ` + string(constants.CLIExtensionPrefix) + ` add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --create-pull-request --force
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --push         # Add and push changes
  ` + string(constants.CLIExtensionPrefix) + ` add ./my-workflow.md                             # Add local workflow
  ` + string(constants.CLIExtensionPrefix) + ` add ./*.md                                       # Add all local workflows
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --dir shared   # Add to .github/workflows/shared/

Workflow specifications:
  - Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)
  - Four+ parts: "owner/repo/workflows/workflow-name.md[@version]" (requires explicit .md extension)
  - GitHub URL: "https://github.com/owner/repo/blob/branch/path/to/workflow.md"
  - Local file: "./path/to/workflow.md" (adds a workflow from local filesystem)
  - Local wildcard: "./*.md" or "./dir/*.md" (adds all .md files matching pattern)
  - Version can be tag, branch, or SHA (for remote workflows)

The -n flag allows you to specify a custom name for the workflow file (only applies to the first workflow when adding multiple).
The --dir flag allows you to specify a subdirectory under .github/workflows/ where the workflow will be added.
The --create-pull-request flag (or --pr) automatically creates a pull request with the workflow changes.
The --push flag automatically commits and pushes changes after successful workflow addition.
The --force flag overwrites existing workflow files.
The --non-interactive flag skips the guided setup and uses traditional behavior.

Note: To create a new workflow from scratch, use the 'new' command instead.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing workflow specification\n\nUsage:\n  %s <workflow>...\n\nExamples:\n  %[1]s githubnext/agentics/daily-repo-status      Add from repository\n  %[1]s ./my-workflow.md                           Add local workflow\n\nRun '%[1]s --help' for more information", cmd.CommandPath())
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			workflows := args
			engineOverride, _ := cmd.Flags().GetString("engine")
			nameFlag, _ := cmd.Flags().GetString("name")
			createPRFlag, _ := cmd.Flags().GetBool("create-pull-request")
			prFlagAlias, _ := cmd.Flags().GetBool("pr")
			prFlag := createPRFlag || prFlagAlias // Support both --create-pull-request and --pr
			pushFlag, _ := cmd.Flags().GetBool("push")
			forceFlag, _ := cmd.Flags().GetBool("force")
			appendText, _ := cmd.Flags().GetString("append")
			verbose, _ := cmd.Flags().GetBool("verbose")
			noGitattributes, _ := cmd.Flags().GetBool("no-gitattributes")
			workflowDir, _ := cmd.Flags().GetString("dir")
			noStopAfter, _ := cmd.Flags().GetBool("no-stop-after")
			stopAfter, _ := cmd.Flags().GetString("stop-after")
			nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
			disableSecurityScanner, _ := cmd.Flags().GetBool("disable-security-scanner")
			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			// Determine if we should use interactive mode
			// Interactive mode is the default for TTY unless:
			// - --non-interactive flag is set
			// - Any of the batch/automation flags are set (--create-pull-request, --force, --name, --append)
			// - Not a TTY (piped input/output)
			// - In CI environment
			useInteractive := !nonInteractive &&
				!prFlag &&
				!forceFlag &&
				nameFlag == "" &&
				appendText == "" &&
				tty.IsStdoutTerminal() &&
				os.Getenv("CI") == "" &&
				os.Getenv("GO_TEST_MODE") != "true"

			if useInteractive {
				addLog.Print("Using interactive mode")
				return RunAddInteractive(cmd.Context(), workflows, verbose, engineOverride, noGitattributes, workflowDir, noStopAfter, stopAfter)
			}

			// Handle normal (non-interactive) mode
			opts := AddOptions{
				Verbose:                verbose,
				EngineOverride:         engineOverride,
				Name:                   nameFlag,
				Force:                  forceFlag,
				AppendText:             appendText,
				CreatePR:               prFlag,
				Push:                   pushFlag,
				NoGitattributes:        noGitattributes,
				WorkflowDir:            workflowDir,
				NoStopAfter:            noStopAfter,
				StopAfter:              stopAfter,
				DisableSecurityScanner: disableSecurityScanner,
			}
			_, err := AddWorkflows(workflows, opts)
			return err
		},
	}

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

	// Add push flag to add command
	cmd.Flags().Bool("push", false, "Automatically commit and push changes after successful workflow addition")

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

	// Add non-interactive flag to add command
	cmd.Flags().Bool("non-interactive", false, "Skip interactive setup and use traditional behavior (for CI/automation)")

	// Add disable-security-scanner flag to add command
	cmd.Flags().Bool("disable-security-scanner", false, "Disable security scanning of workflow markdown content")

	// Register completions for add command
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// AddWorkflows adds one or more workflows from components to .github/workflows
// with optional repository installation and PR creation.
// Returns AddWorkflowsResult containing PR number (if created) and other metadata.
func AddWorkflows(workflows []string, opts AddOptions) (*AddWorkflowsResult, error) {
	// Resolve workflows first - fetches content directly from GitHub
	resolved, err := ResolveWorkflows(workflows, opts.Verbose)
	if err != nil {
		return nil, err
	}

	return AddResolvedWorkflows(workflows, resolved, opts)
}

// AddResolvedWorkflows adds workflows using pre-resolved workflow data.
// This allows callers to resolve workflows early (e.g., to show descriptions) and then add them later.
// The opts.Quiet parameter suppresses detailed output (useful for interactive mode where output is already shown).
func AddResolvedWorkflows(workflowStrings []string, resolved *ResolvedWorkflows, opts AddOptions) (*AddWorkflowsResult, error) {
	addLog.Printf("Adding workflows: count=%d, engineOverride=%s, createPR=%v, noGitattributes=%v, opts.WorkflowDir=%s, noStopAfter=%v, stopAfter=%s", len(workflowStrings), opts.EngineOverride, opts.CreatePR, opts.NoGitattributes, opts.WorkflowDir, opts.NoStopAfter, opts.StopAfter)

	result := &AddWorkflowsResult{}

	// If creating a PR, check prerequisites
	if opts.CreatePR {
		// Check if GitHub CLI is available
		if !isGHCLIAvailable() {
			return nil, errors.New("GitHub CLI (gh) is required for PR creation but not available")
		}

		// Check if we're in a git repository
		if !isGitRepo() {
			return nil, errors.New("not in a git repository - PR creation requires a git repository")
		}

		// Check no other changes are present
		if err := checkCleanWorkingDirectory(opts.Verbose); err != nil {
			return nil, fmt.Errorf("working directory is not clean: %w", err)
		}
	}

	// Set workflow_dispatch result
	result.HasWorkflowDispatch = resolved.HasWorkflowDispatch

	// Set FromWildcard flag based on resolved workflows
	opts.FromWildcard = resolved.HasWildcard

	// Handle PR creation workflow
	if opts.CreatePR {
		addLog.Print("Creating workflow with PR")
		prNumber, prURL, err := addWorkflowsWithPR(resolved.Workflows, opts)
		if err != nil {
			return nil, err
		}
		result.PRNumber = prNumber
		result.PRURL = prURL
		return result, nil
	}

	// Handle normal workflow addition - pass resolved workflows with content
	addLog.Print("Adding workflows normally without PR")
	return result, addWorkflows(resolved.Workflows, opts)
}

// addWorkflows handles workflow addition using pre-fetched content
func addWorkflows(workflows []*ResolvedWorkflow, opts AddOptions) error {
	// Create file tracker for all operations
	tracker, err := NewFileTracker()
	if err != nil {
		// If we can't create a tracker (e.g., not in git repo), fall back to non-tracking behavior
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not create file tracker: %v", err)))
		}
		tracker = nil
	}
	return addWorkflowsWithTracking(workflows, tracker, opts)
}

// addWorkflows handles workflow addition using pre-fetched content
func addWorkflowsWithTracking(workflows []*ResolvedWorkflow, tracker *FileTracker, opts AddOptions) error {
	// Ensure .gitattributes is configured unless flag is set
	if !opts.NoGitattributes {
		addLog.Print("Configuring .gitattributes")
		if err := ensureGitAttributes(); err != nil {
			addLog.Printf("Failed to configure .gitattributes: %v", err)
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
			}
			// Don't fail the entire operation if gitattributes update fails
		} else if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
		}
	}

	if !opts.Quiet && len(workflows) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding %d workflow(s)...", len(workflows))))
	}

	// Add each workflow using pre-fetched content
	for i, resolved := range workflows {
		if !opts.Quiet && len(workflows) > 1 {
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Adding workflow %d/%d: %s", i+1, len(workflows), resolved.Spec.WorkflowName)))
		}

		if err := addWorkflowWithTracking(resolved, tracker, opts); err != nil {
			return fmt.Errorf("failed to add workflow '%s': %w", resolved.Spec.String(), err)
		}
	}

	if !opts.Quiet && len(workflows) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully added all %d workflows", len(workflows))))
	}

	// If --push is enabled, commit and push changes
	if opts.Push {
		addLog.Print("Push enabled - preparing to commit and push changes")
		fmt.Fprintln(os.Stderr, "")

		// Check if we're on the default branch
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking current branch..."))
		if err := checkOnDefaultBranch(opts.Verbose); err != nil {
			addLog.Printf("Default branch check failed: %v", err)
			return fmt.Errorf("cannot push: %w", err)
		}

		// Confirm with user (skip in CI)
		if err := confirmPushOperation(opts.Verbose); err != nil {
			addLog.Printf("Push operation not confirmed: %v", err)
			return fmt.Errorf("push operation cancelled: %w", err)
		}

		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Preparing to commit and push changes..."))

		// Create commit message
		var commitMessage string
		if len(workflows) == 1 {
			commitMessage = "chore: add workflow " + workflows[0].Spec.WorkflowName
		} else {
			commitMessage = fmt.Sprintf("chore: add %d workflows", len(workflows))
		}

		// Use the helper function to orchestrate the full workflow
		if err := commitAndPushChanges(commitMessage, opts.Verbose); err != nil {
			// Check if it's the "no changes" case
			hasChanges, checkErr := hasChangesToCommit()
			if checkErr == nil && !hasChanges {
				addLog.Print("No changes to commit")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes to commit"))
			} else {
				return err
			}
		} else {
			// Print success messages based on whether remote exists
			fmt.Fprintln(os.Stderr, "")
			if hasRemote() {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes pushed to remote"))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes committed locally (no remote configured)"))
			}
		}
	}

	// Stage tracked files to git if in a git repository
	if isGitRepo() && tracker != nil {
		if err := tracker.StageAllFiles(opts.Verbose); err != nil {
			return fmt.Errorf("failed to stage workflow files: %w", err)
		}
	}

	return nil
}

// addWorkflowWithTracking adds a workflow using pre-fetched content with file tracking
func addWorkflowWithTracking(resolved *ResolvedWorkflow, tracker *FileTracker, opts AddOptions) error {
	workflowSpec := resolved.Spec
	sourceContent := resolved.Content
	sourceInfo := resolved.SourceInfo

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding workflow: "+workflowSpec.String()))
		if opts.Force {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Force flag enabled: will overwrite existing files"))
		}
	}

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Using pre-fetched workflow content (%d bytes)", len(sourceContent))))
	}

	// Security scan: reject workflows containing malicious or dangerous content
	if !opts.DisableSecurityScanner {
		if findings := workflow.ScanMarkdownSecurity(string(sourceContent)); len(findings) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Security scan failed for workflow"))
			fmt.Fprintln(os.Stderr, workflow.FormatSecurityFindings(findings, workflowSpec.WorkflowPath))
			return fmt.Errorf("workflow '%s' failed security scan: %d issue(s) detected", workflowSpec.WorkflowPath, len(findings))
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Security scan passed"))
		}
	} else if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Security scanning disabled"))
	}

	// Find git root to ensure consistent placement
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("add workflow requires being in a git repository: %w", err)
	}

	// Determine the target workflow directory
	var githubWorkflowsDir string
	if opts.WorkflowDir != "" {
		if filepath.IsAbs(opts.WorkflowDir) {
			return fmt.Errorf("workflow directory must be a relative path, got: %s", opts.WorkflowDir)
		}
		opts.WorkflowDir = filepath.Clean(opts.WorkflowDir)
		if !strings.HasPrefix(opts.WorkflowDir, ".github/workflows") {
			githubWorkflowsDir = filepath.Join(gitRoot, ".github/workflows", opts.WorkflowDir)
		} else {
			githubWorkflowsDir = filepath.Join(gitRoot, opts.WorkflowDir)
		}
	} else {
		githubWorkflowsDir = filepath.Join(gitRoot, ".github/workflows")
	}

	// Ensure the target directory exists
	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow directory %s: %w", githubWorkflowsDir, err)
	}

	// Determine the workflowName to use
	var workflowName string
	if opts.Name != "" {
		workflowName = opts.Name
	} else {
		workflowName = workflowSpec.WorkflowName
	}

	// Check if a workflow with this name already exists
	existingFile := filepath.Join(githubWorkflowsDir, workflowName+".md")
	if _, err := os.Stat(existingFile); err == nil && !opts.Force {
		if opts.FromWildcard {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow '%s' already exists in .github/workflows/. Skipping.", workflowName)))
			return nil
		}
		return fmt.Errorf("workflow '%s' already exists in .github/workflows/. Use a different name with -n flag, remove the existing workflow first, or use --force to overwrite", workflowName)
	}

	// For remote workflows, fetch and save include dependencies directly from the source
	if !isLocalWorkflowPath(workflowSpec.WorkflowPath) {
		if err := fetchAndSaveRemoteIncludes(string(sourceContent), workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker); err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch include dependencies: %v", err)))
			}
		}
		// Also fetch and save frontmatter 'imports:' dependencies so they are available
		// locally during compilation. Keeping these as relative paths (not workflowspecs)
		// ensures the compiler resolves them from disk rather than downloading from GitHub.
		if err := fetchAndSaveRemoteFrontmatterImports(string(sourceContent), workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker); err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch frontmatter import dependencies: %v", err)))
			}
		}
	} else if sourceInfo != nil && sourceInfo.IsLocal {
		// For local workflows, collect and copy include dependencies from local paths
		// The source directory is derived from the workflow's path
		sourceDir := filepath.Dir(workflowSpec.WorkflowPath)
		includeDeps, err := collectLocalIncludeDependencies(string(sourceContent), sourceDir, opts.Verbose)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to collect include dependencies: %v", err)))
		}
		if err := copyIncludeDependenciesFromPackageWithForce(includeDeps, githubWorkflowsDir, opts.Verbose, opts.Force, tracker); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy include dependencies: %v", err)))
		}
	}

	// Process the workflow
	destFile := filepath.Join(githubWorkflowsDir, workflowName+".md")

	fileExists := false
	if _, err := os.Stat(destFile); err == nil {
		fileExists = true
		if !opts.Force {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Destination file '%s' already exists, skipping.", destFile)))
			return nil
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Overwriting existing file: "+destFile))
	}

	content := string(sourceContent)

	// Add source field to frontmatter
	commitSHA := ""
	if sourceInfo != nil {
		commitSHA = sourceInfo.CommitSHA
	}
	sourceString := buildSourceStringWithCommitSHA(workflowSpec, commitSHA)
	if sourceString != "" {
		updatedContent, err := addSourceToWorkflow(content, sourceString)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to add source field: %v", err)))
			}
		} else {
			content = updatedContent
		}

		// Note: frontmatter 'imports:' are intentionally kept as relative paths here.
		// fetchAndSaveRemoteFrontmatterImports already downloaded those files locally, so
		// the compiler can resolve them from disk without any GitHub API calls.

		// Process @include directives and replace with workflowspec
		// For local workflows, use the workflow's directory as the base path
		includeSourceDir := ""
		if sourceInfo != nil && sourceInfo.IsLocal {
			includeSourceDir = filepath.Dir(workflowSpec.WorkflowPath)
		}
		processedContent, err := processIncludesWithWorkflowSpec(content, workflowSpec, commitSHA, includeSourceDir, opts.Verbose)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
			}
		} else {
			content = processedContent
		}
	}

	// Handle stop-after field modifications
	if opts.NoStopAfter {
		cleanedContent, err := RemoveFieldFromOnTrigger(content, "stop-after")
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove stop-after field: %v", err)))
			}
		} else {
			content = cleanedContent
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed stop-after field from workflow"))
			}
		}
	} else if opts.StopAfter != "" {
		updatedContent, err := SetFieldInOnTrigger(content, "stop-after", opts.StopAfter)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set stop-after field: %v", err)))
			}
		} else {
			content = updatedContent
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Set stop-after field to: "+opts.StopAfter))
			}
		}
	}

	// Handle engine override - add/update the engine field in frontmatter
	if opts.EngineOverride != "" {
		updatedContent, err := addEngineToWorkflow(content, opts.EngineOverride)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set engine field: %v", err)))
			}
		} else {
			content = updatedContent
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Set engine field to: "+opts.EngineOverride))
			}
		}
	}

	// Append text if provided
	if opts.AppendText != "" {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + opts.AppendText
	}

	// Track the file
	if tracker != nil {
		if fileExists {
			tracker.TrackModified(destFile)
		} else {
			tracker.TrackCreated(destFile)
		}
	}

	// Write the file
	if err := os.WriteFile(destFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write destination file '%s': %w", destFile, err)
	}

	// Show output
	if !opts.Quiet {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added workflow: "+destFile))

		if description := ExtractWorkflowDescription(content); description != "" {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(description))
			fmt.Fprintln(os.Stderr, "")
		}
	}

	// Compile the workflow
	if tracker != nil {
		if err := compileWorkflowWithTracking(destFile, opts.Verbose, opts.Quiet, opts.EngineOverride, tracker); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		}
	} else {
		if err := compileWorkflow(destFile, opts.Verbose, opts.Quiet, opts.EngineOverride); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		}
	}

	return nil
}
