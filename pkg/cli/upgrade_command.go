package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var upgradeLog = logger.New("cli:upgrade_command")

// UpgradeConfig contains configuration for the upgrade command
type UpgradeConfig struct {
	Verbose            bool
	WorkflowDir        string
	NoFix              bool
	NoCompile          bool
	CreatePR           bool
	NoActions          bool
	Audit              bool
	JSON               bool
	DisabledCodemodIDs []string
}

// NewUpgradeCommand creates the upgrade command
func NewUpgradeCommand(validateEngine func(string) error) *cobra.Command {
	cmd := newUpgradeCommandBase(validateEngine)
	newUpgradeCommandFlags(cmd)
	newUpgradeCommandCompletions(cmd)
	return cmd
}

func newUpgradeCommandBase(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade local agent files and workflows (codemods, action updates, and compilation)",
		Long: `Upgrade the repository to the latest version of agentic workflows.

This command:
  1. Updates the dispatcher agent file to the latest template (like 'init' command)
  2. Applies automatic codemods to fix deprecated fields in all workflows (like 'fix --write')
  3. Updates GitHub Actions versions in .github/aw/actions-lock.json (unless --no-actions is set)
  4. Compiles all workflows to generate lock files (like 'compile' command)

Flag behavior:
- Upgrade runs codemods, action version updates, and workflow compilation by default; use --no-fix to skip all three steps
- --no-actions and --no-compile are only applied when --no-fix is not set

DEPENDENCY HEALTH AUDIT:
Use --audit to check dependency health without performing upgrades. This includes:
- Outdated Go dependencies with available updates
- Security advisories from GitHub Security Advisory API
- Dependency maturity analysis (v0.x vs stable versions)
- Comprehensive dependency health report

The --audit flag skips the normal upgrade process.

This command always upgrades all Markdown files in .github/workflows.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` upgrade                              # Upgrade all workflows
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-fix                    # Update agent files only (skip codemods, actions, and compilation)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-actions                # Skip updating GitHub Actions versions
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-compile                # Skip recompiling workflows (do not modify lock files)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --create-pull-request       # Upgrade and open a pull request
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --dir custom/workflows      # Upgrade workflows in custom directory
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --engine claude             # Override AI engine for compilation
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --repo owner/repo           # Upgrade workflows in another repository
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --org my-org                # Preview upgrade pull requests across an organization
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --org my-org --repos '*-service'  # Limit org mode to matching repositories
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --org my-org --create-pull-request  # Open upgrade pull requests in org repositories
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --org my-org --create-pull-request --yes  # Auto-accept per-repo confirmations for PR creation (required in CI)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --org my-org --create-issue  # Open issues in org repos with agentic workflows
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --org my-org --create-issue --yes  # Auto-accept per-repo confirmations (required in CI)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --audit                     # Check dependency health without upgrading
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --audit --json              # Output audit results in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --pre-releases              # Include pre-release versions when upgrading the extension (stable releases are the default)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgradeCommandFromCobra(cmd, validateEngine)
		},
	}
	return cmd
}

func newUpgradeCommandFlags(cmd *cobra.Command) {
	addEngineFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: $GH_AW_WORKFLOWS_DIR or .github/workflows)")
	cmd.Flags().Bool("no-fix", false, "Skip codemods, action version updates, and workflow compilation (only update agent files)")
	cmd.Flags().Bool("no-actions", false, "Skip updating GitHub Actions versions (ignored when --no-fix is set)")
	cmd.Flags().Bool("no-compile", false, "Skip recompiling workflows (do not modify lock files; ignored when --no-fix is set)")
	cmd.Flags().StringSlice("disable-codemod", nil, "Disable specific codemod IDs during the fix step (repeatable)")
	cmd.Flags().Bool("create-pull-request", false, "Create a pull request with the upgrade changes")
	cmd.Flags().Bool("pr", false, "Alias for --create-pull-request")
	_ = cmd.Flags().MarkHidden("pr") // Hide the short alias from help output
	cmd.Flags().Bool("create-issue", false, "Open a GitHub issue in each org repository with agentic workflows (requires --org)")
	cmd.Flags().BoolP("yes", "y", false, "Auto-accept org-mode upgrade confirmations (required in CI)")
	cmd.Flags().Bool("audit", false, "Check dependency health without performing upgrades")
	cmd.Flags().Bool("pre-releases", false, "Include pre-release versions when checking for extension upgrades; pre-releases are installed by exact tag")
	cmd.Flags().Bool("approve", false, "Approve all safe update changes. When strict mode is active (the default), the compiler emits warnings for new restricted secrets or unapproved action additions/removals not present in the existing gh-aw-manifest. Use this flag to approve and skip safe update enforcement")
	cmd.Flags().Bool("skip-extension-upgrade", false, "Skip automatic extension upgrade (used internally to prevent recursion after upgrade)")
	_ = cmd.Flags().MarkHidden("skip-extension-upgrade")
	cmd.Flags().String("org", "", "Preview or create upgrade pull requests across an organization")
	cmd.Flags().StringSlice("repos", nil, "Limit --org mode to repositories matching one or more glob patterns")
	addJSONFlag(cmd)
}

func newUpgradeCommandCompletions(cmd *cobra.Command) {
	// Register completions
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")
}

type runUpgradeCommandFlags struct {
	verbose              bool
	dir                  string
	noFix                bool
	createPR             bool
	createIssue          bool
	yes                  bool
	noActions            bool
	noCompile            bool
	auditFlag            bool
	jsonOutput           bool
	disabledCodemods     []string
	skipExtensionUpgrade bool
	approveUpgrade       bool
	preReleases          bool
	engineOverride       string
	targetRepo           string
	targetOrg            string
	repoGlobs            []string
}

func runUpgradeCommandFromCobra(cmd *cobra.Command, validateEngine func(string) error) error {
	flags := runUpgradeCommandReadFlags(cmd)
	if err := runUpgradeCommandValidate(flags, validateEngine); err != nil {
		return err
	}
	if flags.auditFlag {
		return runDependencyAudit(cmd.Context(), flags.verbose, flags.jsonOutput)
	}
	opts := runUpgradeCommandOptions(cmd.Context(), flags)
	if flags.targetRepo != "" {
		return runUpgradeForTargetRepoFn(cmd.Context(), flags.targetRepo, opts, flags.createPR, flags.verbose)
	}
	if flags.targetOrg != "" {
		return runUpgradeForOrg(cmd.Context(), flags.targetOrg, flags.repoGlobs, opts, flags.createPR, flags.createIssue, flags.verbose)
	}
	return runUpgradeCommandLocal(opts, flags.createPR, flags.verbose)
}

func runUpgradeCommandReadFlags(cmd *cobra.Command) runUpgradeCommandFlags {
	flags := runUpgradeCommandFlags{}
	flags.verbose, _ = cmd.Flags().GetBool("verbose")
	flags.dir, _ = cmd.Flags().GetString("dir")
	flags.noFix, _ = cmd.Flags().GetBool("no-fix")
	createPRFlag, _ := cmd.Flags().GetBool("create-pull-request")
	prFlagAlias, _ := cmd.Flags().GetBool("pr")
	flags.createPR = createPRFlag || prFlagAlias
	flags.createIssue, _ = cmd.Flags().GetBool("create-issue")
	flags.yes, _ = cmd.Flags().GetBool("yes")
	flags.noActions, _ = cmd.Flags().GetBool("no-actions")
	flags.noCompile, _ = cmd.Flags().GetBool("no-compile")
	flags.auditFlag, _ = cmd.Flags().GetBool("audit")
	flags.jsonOutput, _ = cmd.Flags().GetBool("json")
	flags.disabledCodemods, _ = cmd.Flags().GetStringSlice("disable-codemod")
	flags.skipExtensionUpgrade, _ = cmd.Flags().GetBool("skip-extension-upgrade")
	flags.approveUpgrade, _ = cmd.Flags().GetBool("approve")
	flags.preReleases, _ = cmd.Flags().GetBool("pre-releases")
	flags.engineOverride, _ = cmd.Flags().GetString("engine")
	flags.targetRepo, _ = cmd.Flags().GetString("repo")
	flags.targetOrg, _ = cmd.Flags().GetString("org")
	flags.repoGlobs, _ = cmd.Flags().GetStringSlice("repos")
	return flags
}

func runUpgradeCommandValidate(flags runUpgradeCommandFlags, validateEngine func(string) error) error {
	if err := validateEngine(flags.engineOverride); err != nil {
		return err
	}
	if flags.targetRepo != "" && flags.targetOrg != "" {
		return errors.New("cannot specify both --repo and --org flags; use --repo for a single repository or --org for organization-wide upgrades")
	}
	if len(flags.repoGlobs) > 0 && flags.targetOrg == "" {
		return errors.New("--repos requires --org to be specified")
	}
	if flags.createIssue && flags.targetOrg == "" {
		return errors.New("--create-issue requires --org to be specified")
	}
	if flags.createPR && flags.createIssue {
		return errors.New("cannot specify both --create-pull-request and --create-issue")
	}
	return nil
}

func runUpgradeCommandOptions(ctx context.Context, flags runUpgradeCommandFlags) upgradeOptions {
	return upgradeOptions{
		ctx:                  ctx,
		verbose:              flags.verbose,
		workflowDir:          flags.dir,
		noFix:                flags.noFix,
		noCompile:            flags.noCompile,
		noActions:            flags.noActions,
		disabledCodemodIDs:   flags.disabledCodemods,
		skipExtensionUpgrade: flags.skipExtensionUpgrade,
		approve:              flags.approveUpgrade,
		preReleases:          flags.preReleases,
		yes:                  flags.yes,
		engineOverride:       flags.engineOverride,
	}
}

func runUpgradeCommandLocal(opts upgradeOptions, createPR bool, verbose bool) error {
	if createPR {
		if err := PreflightCheckForCreatePR(verbose); err != nil {
			return err
		}
	}
	if err := runUpgradeCommand(opts); err != nil {
		return err
	}
	if createPR {
		prBody := "This PR upgrades agentic workflows by applying the latest codemods, " +
			"updating GitHub Actions versions, and recompiling all workflows."
		_, err := CreatePRWithChanges("upgrade-agentic-workflows", "chore: upgrade agentic workflows",
			"Upgrade agentic workflows", prBody, verbose)
		return err
	}
	return nil
}

// runDependencyAudit performs a dependency health audit
func runDependencyAudit(ctx context.Context, verbose bool, jsonOutput bool) error {
	upgradeLog.Print("Running dependency health audit")

	// Generate comprehensive report
	report, err := GenerateDependencyReport(ctx, verbose)
	if err != nil {
		return fmt.Errorf("failed to generate dependency report: %w", err)
	}

	// Display the report
	if jsonOutput {
		return DisplayDependencyReportJSON(report)
	}
	DisplayDependencyReport(report)

	return nil
}

// upgradeOptions holds parameters for runUpgradeCommand.
type upgradeOptions struct {
	ctx                  context.Context
	verbose              bool
	workflowDir          string
	noFix                bool
	noCompile            bool
	noActions            bool
	disabledCodemodIDs   []string
	skipExtensionUpgrade bool
	approve              bool
	preReleases          bool
	yes                  bool
	engineOverride       string
}

// runUpgradeCommand executes the upgrade process
func runUpgradeCommand(opts upgradeOptions) error {
	upgradeLog.Printf("Running upgrade command: verbose=%v, workflowDir=%s, noFix=%v, noCompile=%v, noActions=%v, disabledCodemodIDs=%v, skipExtensionUpgrade=%v",
		opts.verbose, opts.workflowDir, opts.noFix, opts.noCompile, opts.noActions, opts.disabledCodemodIDs, opts.skipExtensionUpgrade)

	if err := runUpgradeCommandMaybeRelaunch(opts); err != nil {
		return err
	}
	if err := runUpgradeCommandUpdateArtifacts(opts); err != nil {
		return err
	}
	runUpgradeCommandCodemods(opts)
	runUpgradeCommandActions(opts)
	runUpgradeCommandCompile(opts)
	runUpgradeCommandContainerPins(opts)

	// Print success message
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Upgrade complete"))

	return nil
}

func runUpgradeCommandMaybeRelaunch(opts upgradeOptions) error {
	if opts.skipExtensionUpgrade {
		return nil
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking gh-aw extension version..."))
	upgraded, installPath, err := upgradeExtensionIfOutdated(opts.verbose, opts.preReleases)
	if err != nil {
		upgradeLog.Printf("Extension upgrade failed: %v", err)
		return err
	}
	if !upgraded {
		return nil
	}
	upgradeLog.Print("Extension was upgraded; re-launching with new binary")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Continuing upgrade with newly installed version..."))
	if err := relaunchWithSameArgs("--skip-extension-upgrade", installPath); err != nil {
		return err
	}
	return &ExitCodeError{Code: 0}
}

func runUpgradeCommandUpdateArtifacts(opts upgradeOptions) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating dispatcher skill..."))
	upgradeLog.Print("Updating dispatcher skill")
	if err := updateCopilotArtifacts(opts.ctx, opts.verbose); err != nil {
		upgradeLog.Printf("Failed to update dispatcher skill: %v", err)
		return fmt.Errorf("failed to update dispatcher skill: %w", err)
	}
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated dispatcher skill"))
	}
	return nil
}

func runUpgradeCommandCodemods(opts upgradeOptions) {
	if opts.noFix {
		upgradeLog.Print("Skipping codemods (--no-fix specified)")
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping codemods (--no-fix specified)"))
		}
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying codemods to all workflows..."))
	upgradeLog.Print("Applying codemods to all workflows")
	fixConfig := FixConfig{WorkflowIDs: nil, Write: true, Verbose: opts.verbose, WorkflowDir: opts.workflowDir, DisabledCodemodIDs: opts.disabledCodemodIDs}
	if err := RunFix(fixConfig); err != nil {
		upgradeLog.Printf("Failed to apply codemods: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to apply codemods: %v", err)))
	}
}

func runUpgradeCommandActions(opts upgradeOptions) {
	if opts.noFix || opts.noActions {
		runUpgradeCommandActionsSkipped(opts)
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating GitHub Actions versions..."))
	upgradeLog.Print("Updating GitHub Actions versions")
	if err := UpdateActions(opts.ctx, false, opts.verbose, false, 0); err != nil {
		upgradeLog.Printf("Failed to update actions: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update actions: %v", err)))
		return
	}
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated GitHub Actions versions"))
	}
	upgradeLog.Print("Updating action references in workflow .md files")
	if err := UpdateActionsInWorkflowFiles(opts.ctx, opts.workflowDir, opts.engineOverride, opts.verbose, false, true, 0, false); err != nil {
		msg := fmt.Sprintf("Failed to update action references in workflow files: %v", err)
		upgradeLog.Print(msg)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Warning: "+msg))
	}
}

func runUpgradeCommandActionsSkipped(opts upgradeOptions) {
	if opts.noFix {
		upgradeLog.Print("Skipping action updates (--no-fix specified)")
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping action updates (--no-fix specified)"))
		}
	} else if opts.noActions {
		upgradeLog.Print("Skipping action updates (--no-actions specified)")
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping action updates (--no-actions specified)"))
		}
	}
}

func runUpgradeCommandCompile(opts upgradeOptions) {
	if opts.noFix || opts.noCompile {
		runUpgradeCommandCompileSkipped(opts)
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Compiling all workflows..."))
	upgradeLog.Print("Compiling all workflows")
	compiler := createAndConfigureCompiler(CompileConfig{Verbose: opts.verbose, WorkflowDir: opts.workflowDir, Approve: opts.approve, EngineOverride: opts.engineOverride})
	workflowsDir := opts.workflowDir
	if workflowsDir == "" {
		workflowsDir = constants.GetWorkflowDir()
	}
	stats, compileErr := compileAllWorkflowFiles(opts.ctx, compiler, workflowsDir, opts.verbose)
	if compileErr != nil {
		upgradeLog.Printf("Failed to compile workflows: %v", compileErr)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to compile workflows: %v", compileErr)))
	} else if stats != nil {
		runUpgradeCommandCompileSummary(stats, opts.verbose)
	}
}

func runUpgradeCommandCompileSkipped(opts upgradeOptions) {
	if opts.noFix {
		upgradeLog.Print("Skipping compilation (--no-fix specified)")
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping compilation (--no-fix specified)"))
		}
	} else if opts.noCompile {
		upgradeLog.Print("Skipping compilation (--no-compile specified)")
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping compilation (--no-compile specified)"))
		}
	}
}

func runUpgradeCommandCompileSummary(stats *CompilationStats, verbose bool) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Compiled %d workflow(s)", stats.Total-stats.Errors)))
	}
	if stats.Errors > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: %d workflow(s) failed to compile", stats.Errors)))
	}
}

func runUpgradeCommandContainerPins(opts upgradeOptions) {
	if opts.noFix || opts.noActions {
		return
	}
	upgradeLog.Print("Updating container image digest pins")
	newPins, err := UpdateContainerPins(opts.ctx, opts.workflowDir, opts.verbose)
	if err != nil {
		upgradeLog.Printf("Failed to update container pins: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update container pins: %v", err)))
	} else if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated container image pins"))
	}
	if newPins && !opts.noCompile {
		upgradeLog.Print("Recompiling workflows to embed new container digest pins")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Recompiling workflows to embed container digest pins..."))
		if recompileErr := recompileAllWorkflows(opts.ctx, opts.workflowDir, opts.engineOverride, opts.verbose, opts.approve); recompileErr != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to recompile after container pin update: %v", recompileErr)))
		}
	}
}

// updateCopilotArtifacts updates the dispatcher skill and related Copilot setup artifacts.
func updateCopilotArtifacts(ctx context.Context, verbose bool) error {
	// Update dispatcher skill
	if err := ensureAgenticWorkflowsDispatcher(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update dispatcher skill: %v", err)
		return fmt.Errorf("failed to update dispatcher skill: %w", err)
	}
	if err := ensureAgenticWorkflowsAgent(verbose); err != nil {
		upgradeLog.Printf("Failed to update Agentic Workflows custom agent: %v", err)
		return fmt.Errorf("failed to update Agentic Workflows custom agent: %w", err)
	}
	if err := deleteLegacyAgentFiles(verbose); err != nil {
		upgradeLog.Printf("Failed to delete legacy agent files: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to delete legacy agent files: %v", err)))
	}

	// Upgrade copilot-setup-steps.yml version
	actionMode := workflow.DetectActionMode(GetVersion())
	if err := upgradeCopilotSetupSteps(ctx, verbose, actionMode, GetVersion()); err != nil {
		upgradeLog.Printf("Failed to upgrade copilot-setup-steps.yml: %v", err)
		// Don't fail the upgrade if copilot-setup-steps upgrade fails - this is non-critical
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to upgrade copilot-setup-steps.yml: %v", err)))
	}

	return nil
}

// relaunchWithSameArgs re-executes the current binary with the original command-line
// arguments plus the provided extraFlag. stdin/stdout/stderr are forwarded to the child
// process. The function blocks until the child exits and returns its error.
// It is used after a successful extension upgrade so that the freshly-installed binary
// (which carries the new version string) handles all subsequent work.
//
// exeOverride, when non-empty, is used directly as the executable path instead of
// calling os.Executable(). On Linux the caller should pass the pre-rename install
// path because os.Executable() returns a "(deleted)"-suffixed path after the binary
// has been renamed out of the way during the upgrade.
func relaunchWithSameArgs(extraFlag string, exeOverride string) error {
	var exe string
	if exeOverride != "" {
		exe = exeOverride
	} else {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}

		// Resolve symlinks to ensure we exec the real binary, not a wrapper.
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		} else {
			upgradeLog.Printf("Failed to resolve symlink for executable %s (using as-is): %v", exe, err)
		}
	}

	// Explicitly copy os.Args[1:] so appending the extra flag does not modify
	// the original slice backing array.
	newArgs := append(append([]string(nil), os.Args[1:]...), extraFlag)
	upgradeLog.Printf("Re-launching with new binary: %s %v", exe, newArgs)

	cmd := exec.Command(exe, newArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Preserve the child's exit code so the entry-point can propagate it.
			return &ExitCodeError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}
