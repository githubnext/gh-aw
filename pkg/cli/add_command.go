package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
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
		Use:   "add <workflow>...",
		Short: "Add agentic workflows from repositories to .github/workflows",
		Long: `Add one or more agentic workflows from repositories to .github/workflows.

This command adds workflows directly without interactive prompts. Use 'add-wizard'
for a guided setup that configures secrets, creates a pull request, and more.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/daily-repo-status        # Add workflow directly
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor@v1.0.0         # Add with version
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/workflows/ci-doctor.md@main
  ` + string(constants.CLIExtensionPrefix) + ` add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --create-pull-request --force
  ` + string(constants.CLIExtensionPrefix) + ` add ./my-workflow.md                             # Add local workflow
  ` + string(constants.CLIExtensionPrefix) + ` add ./*.md                                       # Add all local workflows
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --dir .github/workflows/shared   # Add to .github/workflows/shared/

Workflow specifications:
  - Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)
  - Four+ parts: "owner/repo/workflows/workflow-name.md[@version]" (requires explicit .md extension)
  - GitHub URL: "https://github.com/owner/repo/blob/branch/path/to/workflow.md"
  - Local file: "./path/to/workflow.md" (adds a workflow from local filesystem)
  - Local wildcard: "./*.md" or "./dir/*.md" (adds all .md files matching pattern)
  - Version can be tag, branch, or SHA (for remote workflows)

The -n flag allows you to specify a custom name for the workflow file (only applies to the first workflow when adding multiple).
The --dir flag allows you to specify the workflow directory (default: .github/workflows).
The --create-pull-request flag creates a pull request with the workflow changes.
The --force flag overwrites existing workflow files.

Note: To create a new workflow from scratch, use the 'new' command instead.
Note: For guided interactive setup, use the 'add-wizard' command instead.`,
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
			forceFlag, _ := cmd.Flags().GetBool("force")
			appendText, _ := cmd.Flags().GetString("append")
			verbose, _ := cmd.Flags().GetBool("verbose")
			noGitattributes, _ := cmd.Flags().GetBool("no-gitattributes")
			workflowDir, _ := cmd.Flags().GetString("dir")
			noStopAfter, _ := cmd.Flags().GetBool("no-stop-after")
			stopAfter, _ := cmd.Flags().GetString("stop-after")
			disableSecurityScanner, _ := cmd.Flags().GetBool("disable-security-scanner")
			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			opts := AddOptions{
				Verbose:                verbose,
				EngineOverride:         engineOverride,
				Name:                   nameFlag,
				Force:                  forceFlag,
				AppendText:             appendText,
				CreatePR:               prFlag,
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

	// Add repository flag to add command.
	// Note: the repo is specified directly in the workflow path argument (e.g., "owner/repo/workflow-name"),
	// so this flag is not read by the command. It is kept hidden to avoid breaking existing scripts
	// that may pass --repo but should not be advertised in help text.
	cmd.Flags().StringP("repo", "r", "", "Source repository containing workflows (owner/repo format)")
	_ = cmd.Flags().MarkHidden("repo") // Hidden: repo is already embedded in the workflow path spec

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
	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")

	// Add no-stop-after flag to add command
	cmd.Flags().Bool("no-stop-after", false, "Remove any stop-after field from the workflow")

	// Add stop-after flag to add command
	cmd.Flags().String("stop-after", "", "Override stop-after value in the workflow (e.g., '+48h', '2025-12-31 23:59:59')")

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
	addLog.Printf("Adding %d workflow(s) to repository", len(workflows))
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
	addLog.Printf("Adding %d workflow(s) with tracking: force=%v, disableSecurityScanner=%v", len(workflows), opts.Force, opts.DisableSecurityScanner)
	// Ensure .gitattributes is configured unless flag is set
	if !opts.NoGitattributes {
		addLog.Print("Configuring .gitattributes")
		if updated, err := ensureGitAttributes(); err != nil {
			addLog.Printf("Failed to configure .gitattributes: %v", err)
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
			}
			// Don't fail the entire operation if gitattributes update fails
		} else if updated && opts.Verbose {
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

	return nil
}

// addWorkflowWithTracking adds a workflow using pre-fetched content with file tracking
func addWorkflowWithTracking(resolved *ResolvedWorkflow, tracker *FileTracker, opts AddOptions) error {
	workflowSpec := resolved.Spec
	sourceContent := resolved.Content
	sourceInfo := resolved.SourceInfo

	addLog.Printf("Adding workflow: name=%s, content_size=%d bytes", workflowSpec.WorkflowName, len(sourceContent))

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding workflow: "+workflowSpec.String()))
		if opts.Force {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Force flag enabled: will overwrite existing files"))
		}
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
	gitRoot, err := gitutil.FindGitRoot()
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
		githubWorkflowsDir = filepath.Join(gitRoot, opts.WorkflowDir)
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

	// For remote workflows, fetch and save all dependencies (includes, imports, dispatch workflows, resources)
	if !isLocalWorkflowPath(workflowSpec.WorkflowPath) {
		if err := fetchAllRemoteDependencies(context.Background(), string(sourceContent), workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker); err != nil {
			return err
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

	// Handle engine override - add/update the engine field in frontmatter before source so
	// the engine declaration appears above the source field in the final file.
	// The default engine is omitted to avoid unnecessary noise and prevent conflicts during
	// later workflow updates.
	if opts.EngineOverride != "" && opts.EngineOverride != string(constants.DefaultEngine) {
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

		// Process @include directives and replace with workflowspec.
		// For local workflows, use the workflow's directory as the package source path.
		// Pass githubWorkflowsDir as localWorkflowDir so that any body-level import
		// whose target already exists locally is preserved as a local reference rather
		// than being rewritten to a cross-repo workflowspec.
		includeSourceDir := ""
		if sourceInfo != nil && sourceInfo.IsLocal {
			includeSourceDir = filepath.Dir(workflowSpec.WorkflowPath)
		}
		processedContent, err := processIncludesWithWorkflowSpec(content, workflowSpec, commitSHA, includeSourceDir, githubWorkflowsDir, opts.Verbose)
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

	// For remote workflows: now that the main workflow and all its imports are on disk,
	// parse the fully merged safe-outputs configuration to discover any dispatch workflows
	// that originate from imported shared workflows (not visible in the raw frontmatter).
	if !isLocalWorkflowPath(workflowSpec.WorkflowPath) {
		fetchAndSaveDispatchWorkflowsFromParsedFile(destFile, workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker)
	}

	// Compile any dispatch-workflow .md dependencies that were just fetched and lack a
	// .lock.yml. The dispatch-workflow validator requires every .md dispatch target to be
	// compiled before the main workflow can be validated.
	compileDispatchWorkflowDependencies(destFile, opts.Verbose, opts.Quiet, opts.EngineOverride, tracker)

	// Compile the workflow
	if tracker != nil {
		if err := compileWorkflowWithTracking(destFile, opts.Verbose, opts.Quiet, opts.EngineOverride, tracker); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorChain(err))
		}
	} else {
		if err := compileWorkflow(destFile, opts.Verbose, opts.Quiet, opts.EngineOverride); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorChain(err))
		}
	}

	return nil
}
