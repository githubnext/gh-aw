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
	"github.com/github/gh-aw/pkg/parser"
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
		Short: "Add agentic workflows from repositories or local files to .github/workflows",
		Long: `Add one or more agentic workflows from repositories to .github/workflows.

This command adds workflows directly without interactive prompts. Use 'add-wizard'
for a guided setup that configures secrets, creates a pull request, and more.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/daily-repo-status        # Add workflow directly
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/repo-assist              # Add package from repository root aw.yml
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/packages/repo-assist     # Add package from nested aw.yml
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor@v1.0.0         # Add with version
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/workflows/ci-doctor.md@main
  ` + string(constants.CLIExtensionPrefix) + ` add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md
  ` + string(constants.CLIExtensionPrefix) + ` add https://example.com/my-workflow.md           # Add workflow from any HTTPS URL
  ` + string(constants.CLIExtensionPrefix) + ` add https://example.com/workflow.json            # Import JSON workflow definition
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --create-pull-request --force
  ` + string(constants.CLIExtensionPrefix) + ` add ./my-workflow.md                             # Add local workflow
  ` + string(constants.CLIExtensionPrefix) + ` add ./*.md                                       # Add all local workflows
  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/ci-doctor --dir .github/workflows/shared   # Add to .github/workflows/shared/

Workflow specifications:
  - Two parts: "owner/repo[@version]" (loads repository-root aw.yml package)
  - Three+ parts without .md: "owner/repo/folder[@version]" (loads nested aw.yml package when present)
  - Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)
  - Four+ parts: "owner/repo/workflows/workflow-name.md[@version]" (requires explicit .md extension)
  - GitHub URL: "https://github.com/owner/repo/blob/branch/path/to/workflow.md"
  - Arbitrary URL: "https://example.com/workflow.md" (fetches and dispatches on Content-Type)
    - text/markdown → treated as a gh-aw workflow markdown file
    - application/json → converted from a JSON workflow definition
  - Local file: "./path/to/workflow.md" (adds a workflow from local filesystem)
  - Local wildcard: "./*.md" or "./dir/*.md" (adds all .md files matching pattern)
  - Version can be tag, branch, or SHA (for remote workflows)

The -n flag allows you to specify a custom name for the workflow file (not allowed when adding multiple workflows at once).
The --dir flag allows you to specify the workflow directory (default: .github/workflows).
The --create-pull-request flag creates a pull request with the workflow changes.
The --force flag overwrites existing workflow files.

Note: In GitHub Enterprise repos, shorthand source specs resolve on your enterprise host by default.
      For github/*, githubnext/*, and microsoft/* sources, shorthand resolves on github.com.
      Use full https://github.com/... source URLs for other public github.com workflows.
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

			if nameFlag != "" && len(workflows) > 1 {
				return errors.New("--name flag cannot be used when adding multiple workflows at once")
			}

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
			_, err := AddWorkflows(cmd.Context(), workflows, opts)
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
func AddWorkflows(ctx context.Context, workflows []string, opts AddOptions) (*AddWorkflowsResult, error) {
	// Resolve workflows first - fetches content directly from GitHub
	resolved, err := ResolveWorkflows(ctx, workflows, opts.Verbose)
	if err != nil {
		return nil, err
	}

	return AddResolvedWorkflows(ctx, workflows, resolved, opts)
}

// AddResolvedWorkflows adds workflows using pre-resolved workflow data.
// This allows callers to resolve workflows early (e.g., to show descriptions) and then add them later.
// The opts.Quiet parameter suppresses detailed output (useful for interactive mode where output is already shown).
func AddResolvedWorkflows(ctx context.Context, workflowStrings []string, resolved *ResolvedWorkflows, opts AddOptions) (*AddWorkflowsResult, error) {
	addLog.Printf("Adding workflows: count=%d, engineOverride=%s, createPR=%v, noGitattributes=%v, opts.WorkflowDir=%s, noStopAfter=%v, stopAfter=%s", len(workflowStrings), opts.EngineOverride, opts.CreatePR, opts.NoGitattributes, opts.WorkflowDir, opts.NoStopAfter, opts.StopAfter)

	result := &AddWorkflowsResult{}

	for _, warning := range resolved.Warnings {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warning))
	}

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
		prNumber, prURL, err := addWorkflowsWithPR(ctx, resolved.Workflows, opts)
		if err != nil {
			return nil, err
		}
		result.PRNumber = prNumber
		result.PRURL = prURL
		return result, nil
	}

	// Handle normal workflow addition - pass resolved workflows with content
	addLog.Print("Adding workflows normally without PR")
	return result, addWorkflows(ctx, resolved.Workflows, opts)
}

// addWorkflows handles workflow addition using pre-fetched content
func addWorkflows(ctx context.Context, workflows []*ResolvedWorkflow, opts AddOptions) error {
	addLog.Printf("Adding %d workflow(s) to repository", len(workflows))
	// Create file tracker for all operations
	tracker := NewFileTracker()
	return addWorkflowsWithTracking(ctx, workflows, tracker, opts)
}

// addWorkflows handles workflow addition using pre-fetched content
func addWorkflowsWithTracking(ctx context.Context, workflows []*ResolvedWorkflow, tracker *FileTracker, opts AddOptions) error {
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

		if err := addWorkflowWithTracking(ctx, resolved, tracker, opts); err != nil {
			return fmt.Errorf("failed to add workflow '%s': %w", resolved.Spec.String(), err)
		}
	}

	if !opts.Quiet && len(workflows) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully added all %d workflows", len(workflows))))
	}

	return nil
}

// addWorkflowWithTracking adds a workflow using pre-fetched content with file tracking
func addWorkflowWithTracking(ctx context.Context, resolved *ResolvedWorkflow, tracker *FileTracker, opts AddOptions) error {
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

	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("add workflow requires being in a git repository: %w", err)
	}

	githubWorkflowsDir, err := resolveWorkflowTargetDir(gitRoot, &opts)
	if err != nil {
		return err
	}

	workflowName := workflowSpec.WorkflowName
	if opts.Name != "" {
		workflowName = opts.Name
	}

	if resolved.IsActionWorkflow {
		return addActionWorkflowWithTracking(resolved, tracker, opts, githubWorkflowsDir, workflowName)
	}
	if resolved.IsPackageSkillFile {
		return addSkillFileWithTracking(resolved, tracker, opts, gitRoot)
	}
	if resolved.IsPackageAgentFile {
		return addAgentFileWithTracking(resolved, tracker, opts, gitRoot)
	}

	existingFile := filepath.Join(githubWorkflowsDir, workflowName+".md")
	if _, err := os.Stat(existingFile); err == nil && !opts.Force {
		if opts.FromWildcard {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow '%s' already exists in .github/workflows/. Skipping.", workflowName)))
			return nil
		}
		return fmt.Errorf("workflow '%s' already exists in .github/workflows/. Use a different name with -n flag, remove the existing workflow first, or use --force to overwrite", workflowName)
	}

	if err := fetchWorkflowDependencies(ctx, string(sourceContent), workflowSpec, sourceInfo, githubWorkflowsDir, opts, tracker); err != nil {
		return err
	}

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

	content := addWorkflowSourceAndIncludes(string(sourceContent), workflowSpec, sourceInfo, githubWorkflowsDir, opts)
	content = applyWorkflowStopAfterAndAppend(content, opts)

	return writeAndCompileWorkflowFile(ctx, content, destFile, workflowSpec, githubWorkflowsDir, opts, tracker, fileExists)
}

// resolveWorkflowTargetDir resolves and creates the workflow target directory.
func resolveWorkflowTargetDir(gitRoot string, opts *AddOptions) (string, error) {
	var dir string
	if opts.WorkflowDir != "" {
		if filepath.IsAbs(opts.WorkflowDir) {
			return "", fmt.Errorf("workflow directory must be a relative path, got: %s", opts.WorkflowDir)
		}
		opts.WorkflowDir = filepath.Clean(opts.WorkflowDir)
		dir = filepath.Join(gitRoot, opts.WorkflowDir)
	} else {
		dir = filepath.Join(gitRoot, constants.GetWorkflowDir())
	}
	if err := os.MkdirAll(dir, constants.DirPermPublic); err != nil {
		return "", fmt.Errorf("failed to create workflow directory %s: %w", dir, err)
	}
	return dir, nil
}

// fetchWorkflowDependencies fetches remote or copies local include dependencies.
func fetchWorkflowDependencies(ctx context.Context, content string, workflowSpec *WorkflowSpec, sourceInfo *FetchedWorkflow, githubWorkflowsDir string, opts AddOptions, tracker *FileTracker) error {
	if workflowSpec.RawURL != "" {
		return nil
	}
	if !isLocalWorkflowPath(workflowSpec.WorkflowPath) {
		return fetchAllRemoteDependencies(ctx, content, workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker)
	}
	if sourceInfo != nil && sourceInfo.IsLocal {
		sourceDir := filepath.Dir(workflowSpec.WorkflowPath)
		includeDeps, err := collectLocalIncludeDependencies(content, sourceDir, opts.Verbose)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to collect include dependencies: %v", err)))
		}
		if err := copyIncludeDependenciesFromPackageWithForce(includeDeps, githubWorkflowsDir, opts.Verbose, opts.Force, tracker); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy include dependencies: %v", err)))
		}
	}
	return nil
}

// addWorkflowSourceAndIncludes applies engine override, source field, and include processing.
func addWorkflowSourceAndIncludes(content string, workflowSpec *WorkflowSpec, sourceInfo *FetchedWorkflow, githubWorkflowsDir string, opts AddOptions) string {
	if opts.EngineOverride != "" && opts.EngineOverride != string(constants.DefaultEngine) {
		if updated, err := addEngineToWorkflow(content, opts.EngineOverride); err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set engine field: %v", err)))
			}
		} else {
			content = updated
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Set engine field to: "+opts.EngineOverride))
			}
		}
	}

	commitSHA := ""
	if sourceInfo != nil {
		commitSHA = sourceInfo.CommitSHA
	}
	// When the fetch used a fallback path, SourcePath holds the canonical path.
	if sourceInfo != nil && !sourceInfo.IsLocal && sourceInfo.SourcePath != "" && sourceInfo.SourcePath != workflowSpec.WorkflowPath {
		specCopy := *workflowSpec
		specCopy.WorkflowPath = sourceInfo.SourcePath
		workflowSpec = &specCopy
	}
	sourceString := buildSourceStringWithCommitSHA(workflowSpec, commitSHA)
	if sourceString == "" {
		return content
	}
	if updated, err := addSourceToWorkflow(content, sourceString); err != nil {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to add source field: %v", err)))
		}
	} else {
		content = updated
	}

	includeSourceDir := ""
	if sourceInfo != nil && sourceInfo.IsLocal {
		includeSourceDir = filepath.Dir(workflowSpec.WorkflowPath)
	}
	if processed, err := processIncludesWithWorkflowSpec(content, workflowSpec, commitSHA, includeSourceDir, githubWorkflowsDir, opts.Verbose); err != nil {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
		}
	} else {
		content = processed
	}
	return content
}

// applyWorkflowStopAfterAndAppend applies stop-after modifications and appends extra text.
func applyWorkflowStopAfterAndAppend(content string, opts AddOptions) string {
	if opts.NoStopAfter {
		if cleaned, err := RemoveFieldFromOnTrigger(content, "stop-after"); err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove stop-after field: %v", err)))
			}
		} else {
			content = cleaned
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed stop-after field from workflow"))
			}
		}
	} else if opts.StopAfter != "" {
		if updated, err := SetFieldInOnTrigger(content, "stop-after", opts.StopAfter); err != nil {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to set stop-after field: %v", err)))
			}
		} else {
			content = updated
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Set stop-after field to: "+opts.StopAfter))
			}
		}
	}
	if opts.AppendText != "" {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + opts.AppendText
	}
	return content
}

// writeAndCompileWorkflowFile writes the workflow file, shows output, and compiles it.
func writeAndCompileWorkflowFile(ctx context.Context, content, destFile string, workflowSpec *WorkflowSpec, githubWorkflowsDir string, opts AddOptions, tracker *FileTracker, fileExists bool) error {
	if tracker != nil {
		if fileExists {
			tracker.TrackModified(destFile)
		} else {
			tracker.TrackCreated(destFile)
		}
	}

	if err := os.WriteFile(destFile, []byte(content), constants.FilePermSensitive); err != nil {
		return fmt.Errorf("failed to write destination file '%s': %w", destFile, err)
	}
	writtenContent, err := os.ReadFile(destFile)
	if err != nil {
		return fmt.Errorf("failed to read back destination file '%s': %w", destFile, err)
	}

	if !opts.Quiet {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added workflow: "+filepath.Base(destFile)))
		if opts.Verbose {
			if description := ExtractWorkflowDescription(string(writtenContent)); description != "" {
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(description))
				fmt.Fprintln(os.Stderr, "")
			}
		}
	}

	if !isLocalWorkflowPath(workflowSpec.WorkflowPath) {
		fetchAndSaveDispatchWorkflowsFromParsedFile(destFile, workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker)
	}
	compileDispatchWorkflowDependencies(ctx, destFile, opts.Verbose, opts.Quiet, opts.EngineOverride, tracker)

	if tracker != nil {
		if err := compileWorkflowWithTracking(ctx, destFile, opts.Verbose, opts.Quiet, opts.EngineOverride, tracker); err != nil {
			printCompilationError(err, opts.Quiet)
		}
	} else {
		if err := compileWorkflow(ctx, destFile, opts.Verbose, opts.Quiet, opts.EngineOverride); err != nil {
			printCompilationError(err, opts.Quiet)
		}
	}
	return nil
}

// addActionWorkflowWithTracking installs a raw GitHub Actions YAML workflow file (.yml)
// directly to the target directory without any frontmatter processing or compilation.
func addActionWorkflowWithTracking(resolved *ResolvedWorkflow, tracker *FileTracker, opts AddOptions, githubWorkflowsDir, workflowName string) error {
	destFile := filepath.Join(githubWorkflowsDir, workflowName+".yml")

	addLog.Printf("Adding action workflow: dest=%s, content_size=%d bytes", destFile, len(resolved.Content))

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding action workflow: "+destFile))
	}

	fileExists := false
	if _, err := os.Stat(destFile); err == nil {
		fileExists = true
		if !opts.Force {
			if opts.FromWildcard {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Action workflow '%s' already exists. Skipping.", workflowName+".yml")))
				return nil
			}
			return fmt.Errorf("action workflow '%s' already exists in %s. Use --force to overwrite", workflowName+".yml", githubWorkflowsDir)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Overwriting existing file: "+destFile))
	}

	if tracker != nil {
		if fileExists {
			tracker.TrackModified(destFile)
		} else {
			tracker.TrackCreated(destFile)
		}
	}

	if err := os.WriteFile(destFile, resolved.Content, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write action workflow file '%s': %w", destFile, err)
	}

	if !opts.Quiet {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added action workflow: "+filepath.Base(destFile)))
	}

	return nil
}

// addSkillFileWithTracking installs a single skill file from a package to the agentic engine
// skill directory. The file's path relative to the skill directory is preserved so that
// nested files (e.g. scripts/ subdirectories) are written with their full structure intact.
func addSkillFileWithTracking(resolved *ResolvedWorkflow, tracker *FileTracker, opts AddOptions, gitRoot string) error {
	engineSkillDir := parser.GetEngineSkillDir(opts.EngineOverride)
	skillDir := filepath.Join(gitRoot, engineSkillDir, resolved.SkillName)

	// Determine the relative path under the skill directory so nested files preserve
	// structure (e.g. "scripts/query.sh"). Match a skill-name path component that is
	// immediately under skills/ or .github/skills/ to avoid accidental first matches.
	parts := strings.Split(filepath.ToSlash(resolved.Spec.WorkflowPath), "/")
	var relParts []string
	for i, part := range parts {
		if i >= len(parts)-1 {
			break
		}
		if part != resolved.SkillName {
			continue
		}
		if i > 0 && parts[i-1] == "skills" {
			relParts = parts[i+1:]
			break
		}
		if i > 1 && parts[i-1] == "skills" && parts[i-2] == ".github" {
			relParts = parts[i+1:]
			break
		}
	}
	if len(relParts) == 0 {
		return fmt.Errorf("failed to determine relative path for skill %q from source path %q", resolved.SkillName, resolved.Spec.WorkflowPath)
	}
	relPath := filepath.Clean(filepath.Join(relParts...))
	if relPath == "." || relPath == "" || relPath == string(os.PathSeparator) {
		return fmt.Errorf("invalid relative skill path %q from source path %q", relPath, resolved.Spec.WorkflowPath)
	}

	destFile := filepath.Join(skillDir, relPath)
	relToSkillDir, err := filepath.Rel(skillDir, destFile)
	if err != nil {
		return fmt.Errorf("failed to validate destination path %q for skill %q: %w", destFile, resolved.SkillName, err)
	}
	if relToSkillDir == ".." || strings.HasPrefix(relToSkillDir, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("skill file path %q escapes destination skill directory %q", relPath, skillDir)
	}

	// Ensure the destination directory exists (handles nested subdirectories).
	if err := os.MkdirAll(filepath.Dir(destFile), constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create skill directory %s: %w", filepath.Dir(destFile), err)
	}

	addLog.Printf("Adding skill file: dest=%s, skill=%s, content_size=%d bytes", destFile, resolved.SkillName, len(resolved.Content))

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding skill file to %s: %s", engineSkillDir+"/"+resolved.SkillName, relPath)))
	}

	fileExists := false
	if _, err := os.Stat(destFile); err == nil {
		fileExists = true
		if !opts.Force {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skill file '%s' already exists. Skipping.", destFile)))
			}
			return nil
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Overwriting existing skill file: "+destFile))
	}

	if tracker != nil {
		if fileExists {
			tracker.TrackModified(destFile)
		} else {
			tracker.TrackCreated(destFile)
		}
	}

	if err := os.WriteFile(destFile, resolved.Content, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write skill file '%s': %w", destFile, err)
	}

	if !opts.Quiet {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Added skill file: %s/%s/%s", engineSkillDir, resolved.SkillName, relPath)))
	}

	return nil
}

// addAgentFileWithTracking installs a single agent file from a package to the agentic engine
// agents directory.
func addAgentFileWithTracking(resolved *ResolvedWorkflow, tracker *FileTracker, opts AddOptions, gitRoot string) error {
	engineAgentsDir := parser.GetEngineSubAgentDir(opts.EngineOverride)
	agentsDir := filepath.Join(gitRoot, engineAgentsDir)
	if err := os.MkdirAll(agentsDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create agents directory %s: %w", agentsDir, err)
	}

	fileName := filepath.Base(resolved.Spec.WorkflowPath)
	destFile := filepath.Join(agentsDir, fileName)

	addLog.Printf("Adding agent file: dest=%s, content_size=%d bytes", destFile, len(resolved.Content))

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding agent file to %s: %s", engineAgentsDir, fileName)))
	}

	fileExists := false
	if _, err := os.Stat(destFile); err == nil {
		fileExists = true
		if !opts.Force {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Agent file '%s' already exists. Skipping.", destFile)))
			}
			return nil
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Overwriting existing agent file: "+destFile))
	}

	if tracker != nil {
		if fileExists {
			tracker.TrackModified(destFile)
		} else {
			tracker.TrackCreated(destFile)
		}
	}

	if err := os.WriteFile(destFile, resolved.Content, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write agent file '%s': %w", destFile, err)
	}

	if !opts.Quiet {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Added agent file: %s/%s", engineAgentsDir, fileName)))
	}

	return nil
}

// printCompilationError formats and writes a compilation error to stderr.
// Redirect-only workflow errors are treated as informational messages rather than errors,
// since they occur when a redirect placeholder was downloaded without resolving to the full
// workflow content. In that case the user is directed to run `gh aw update`.
// All other errors are written using FormatErrorChain for standard error formatting.
func printCompilationError(err error, quiet bool) {
	var redirectErr *workflow.RedirectOnlyWorkflowError
	if errors.As(err, &redirectErr) {
		if !quiet {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(redirectErr.Error()))
		}
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatErrorChain(err))
}
