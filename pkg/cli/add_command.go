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
	"github.com/github/gh-aw/pkg/fileutil"
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
	// AddCopilotRequestsPermission injects permissions.copilot-requests: write into
	// the workflow frontmatter, enabling GitHub Actions token auth for Copilot.
	// Set by the add-wizard when the user selects org-billing auth instead of a PAT.
	AddCopilotRequestsPermission bool
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
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` add githubnext/agentics/daily-repo-status        # Add workflow directly
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
`,
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
			disableSecurityScanner, _ := cmd.Flags().GetBool("no-security-scanner")
			disableSecurityScannerLegacy, _ := cmd.Flags().GetBool("disable-security-scanner")
			disableSecurityScanner = disableSecurityScanner || disableSecurityScannerLegacy

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

	// Add no-security-scanner flag to add command (--disable-security-scanner is kept as an undocumented alias)
	cmd.Flags().Bool("no-security-scanner", false, "Disable security scanning of workflow markdown content")
	cmd.Flags().Bool("disable-security-scanner", false, "Disable security scanning of workflow markdown content")
	_ = cmd.Flags().MarkHidden("disable-security-scanner")

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
		githubWorkflowsDir = filepath.Join(gitRoot, constants.GetWorkflowDir())
	}

	// Ensure the target directory exists
	if err := os.MkdirAll(githubWorkflowsDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create workflow directory %s: %w", githubWorkflowsDir, err)
	}

	// Determine the workflowName to use
	var workflowName string
	if opts.Name != "" {
		workflowName = opts.Name
	} else {
		workflowName = workflowSpec.WorkflowName
	}

	// Action workflow files (.yml) are copied as-is to .github/workflows/ without any
	// frontmatter processing, dependency fetching, or compilation.
	if resolved.IsActionWorkflow {
		return addActionWorkflowWithTracking(resolved, tracker, opts, githubWorkflowsDir, workflowName)
	}

	// Package skill files are copied as-is to the agentic engine skill directory.
	if resolved.IsPackageSkillFile {
		return addSkillFileWithTracking(resolved, tracker, opts, gitRoot)
	}

	// Package agent files are copied as-is to the agentic engine agents directory.
	if resolved.IsPackageAgentFile {
		return addAgentFileWithTracking(resolved, tracker, opts, gitRoot)
	}

	// Check if a workflow with this name already exists
	existingFile := filepath.Join(githubWorkflowsDir, workflowName+".md")
	if fileutil.FileExists(existingFile) && !opts.Force {
		if opts.FromWildcard {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow '%s' already exists in .github/workflows/. Skipping.", workflowName)))
			return nil
		}
		return fmt.Errorf("workflow '%s' already exists in .github/workflows/. Use a different name with -n flag, remove the existing workflow first, or use --force to overwrite", workflowName)
	}

	// For remote workflows, fetch and save all dependencies (includes, imports, dispatch workflows, resources)
	if workflowSpec.RawURL != "" {
		// Generic URL imports carry no GitHub repo context; dependency fetching is skipped.
	} else if !isLocalWorkflowPath(workflowSpec.WorkflowPath) {
		if err := fetchAllRemoteDependencies(ctx, string(sourceContent), workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker); err != nil {
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
	if fileutil.FileExists(destFile) {
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

	// Inject permissions.copilot-requests: write when the user chose org-billing auth.
	// Only inject for Copilot workflows — guard against the flag being inadvertently set
	// when multiple workflows with different engines are processed in the same batch.
	if opts.AddCopilotRequestsPermission && isCopilotWorkflowContent(content) {
		updatedContent, err := addCopilotRequestsPermissionToContent(content)
		if err != nil {
			// Always warn: user explicitly chose copilot-requests auth; a silent failure
			// means the deployed workflow will lack the required permission.
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to add copilot-requests permission: %v", err)))
		} else {
			content = updatedContent
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Added permissions.copilot-requests: write to workflow"))
			}
		}
	}

	// Add source field to frontmatter
	commitSHA := ""
	if sourceInfo != nil {
		commitSHA = sourceInfo.CommitSHA
	}
	// When the fetch used a fallback path (e.g. .github/workflows/my-workflow.md instead
	// of the short-form my-workflow.md), SourcePath holds the actual repo-root-relative
	// path. Propagate it to workflowSpec so all downstream processing (source field,
	// include/import resolution) uses the canonical path.
	if sourceInfo != nil && !sourceInfo.IsLocal && sourceInfo.SourcePath != "" && sourceInfo.SourcePath != workflowSpec.WorkflowPath {
		specCopy := *workflowSpec
		specCopy.WorkflowPath = sourceInfo.SourcePath
		workflowSpec = &specCopy
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
	if err := os.WriteFile(destFile, []byte(content), constants.FilePermSensitive); err != nil {
		return fmt.Errorf("failed to write destination file '%s': %w", destFile, err)
	}
	// Read back the just-written file to ensure downstream processing (including
	// frontmatter hash computation) uses the exact bytes on disk and avoids parity drift.
	writtenContent, err := os.ReadFile(destFile)
	if err != nil {
		return fmt.Errorf("failed to read back destination file '%s': %w", destFile, err)
	}

	// Show output
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

	// For remote workflows: now that the main workflow and all its imports are on disk,
	// parse the fully merged safe-outputs configuration to discover any dispatch workflows
	// that originate from imported shared workflows (not visible in the raw frontmatter).
	if !isLocalWorkflowPath(workflowSpec.WorkflowPath) {
		fetchAndSaveDispatchWorkflowsFromParsedFile(destFile, workflowSpec, githubWorkflowsDir, opts.Verbose, opts.Force, tracker)
	}

	// Compile any dispatch-workflow .md dependencies that were just fetched and lack a
	// .lock.yml. The dispatch-workflow validator requires every .md dispatch target to be
	// compiled before the main workflow can be validated.
	compileDispatchWorkflowDependencies(ctx, destFile, opts.Verbose, opts.Quiet, opts.EngineOverride, tracker)

	// Compile the workflow
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
	if fileutil.FileExists(destFile) {
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
	engineSkillDir := workflow.GetEngineSkillDir(opts.EngineOverride)
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
	if fileutil.FileExists(destFile) {
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
	engineAgentsDir := workflow.GetEngineSubAgentDir(opts.EngineOverride)
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
	if fileutil.FileExists(destFile) {
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

// isCopilotWorkflowContent returns true when the workflow frontmatter declares engine: copilot.
// It is used to guard AddCopilotRequestsPermission injection so that the flag is only applied
// to Copilot workflows even when multiple workflows of different engines are processed together.
func isCopilotWorkflowContent(content string) bool {
	lines, _, err := parseFrontmatterLines(content)
	if err != nil {
		return false
	}
	for _, line := range lines {
		if !isTopLevelKey(line) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if parseYAMLMapKey(trimmed) == "engine" {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "engine:"))
			return val == string(constants.CopilotEngine)
		}
	}
	return false
}

// addCopilotRequestsPermissionToContent injects `permissions.copilot-requests: write`
// into the workflow frontmatter, enabling GitHub Actions token auth for Copilot (org billing).
// It delegates to ensureCopilotRequestsWritePermission, which locates or creates the
// permissions block and appends the copilot-requests entry if not already present.
// The function is idempotent: calling it on content that already contains the permission
// returns the content unchanged.
// Returns an error if the permission could not be injected and is not already present
// (e.g., when `permissions:` is a non-mapping scalar like `read-all`).
func addCopilotRequestsPermissionToContent(content string) (string, error) {
	var injectionFailed bool
	newContent, _, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
		updated := ensureCopilotRequestsWritePermission(lines)
		// Detect whether ensureCopilotRequestsWritePermission actually made a change.
		// When lengths differ, a line was added — modified is true without needing element comparison.
		// When lengths are equal, compare element-by-element (safe since len(updated)==len(lines)).
		modified := len(updated) != len(lines)
		if !modified {
			for i := range lines {
				if lines[i] != updated[i] {
					modified = true
					break
				}
			}
		}
		if !modified {
			// Lines unchanged — either the permission is already present (idempotent) or
			// it could not be injected (e.g., `permissions:` is a scalar like `read-all`).
			if !copilotRequestsPermissionPresentInLines(updated) {
				injectionFailed = true
			}
		}
		return updated, modified
	})
	if injectionFailed {
		return content, errors.New("cannot inject permissions.copilot-requests: write: 'permissions' is a non-mapping scalar value; update it manually")
	}
	if err != nil {
		return content, err
	}
	return newContent, nil
}

// copilotRequestsPermissionPresentInLines returns true when the frontmatter lines contain
// a `copilot-requests:` key (ignoring comment lines). It is used to distinguish the idempotent
// case (permission already present) from the injection-failure case (scalar permissions field).
func copilotRequestsPermissionPresentInLines(lines []string) bool {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") && parseYAMLMapKey(trimmed) == "copilot-requests" {
			return true
		}
	}
	return false
}
