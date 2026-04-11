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
	Verbose     bool
	WorkflowDir string
	NoFix       bool
	NoCompile   bool
	CreatePR    bool
	NoActions   bool
	Audit       bool
	JSON        bool
}

// NewUpgradeCommand creates the upgrade command
func NewUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade repository with latest agent files and apply codemods to all workflows",
		Long: `Upgrade the repository for the latest version of agentic workflows.

This command:
  1. Updates the dispatcher agent file to the latest template (like 'init' command)
  2. Applies automatic codemods to fix deprecated fields in all workflows (like 'fix --write')
  3. Updates GitHub Actions versions in .github/aw/actions-lock.json (unless --no-actions is set)
  4. Compiles all workflows to generate lock files (like 'compile' command)

DEPENDENCY HEALTH AUDIT:
Use --audit to check dependency health without performing upgrades. This includes:
- Outdated Go dependencies with available updates
- Security advisories from GitHub Security Advisory API
- Dependency maturity analysis (v0.x vs stable versions)
- Comprehensive dependency health report

The --audit flag skips the normal upgrade process.

The upgrade process ensures:
- Dispatcher agent is current (.github/agents/agentic-workflows.agent.md)
- All workflows use the latest syntax and configuration options
- Deprecated fields are automatically migrated across all workflows
- GitHub Actions are pinned to the latest versions
- All workflows are compiled and lock files are up-to-date

This command always upgrades all Markdown files in .github/workflows.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` upgrade                    # Upgrade all workflows
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-fix          # Update agent files only (skip codemods, actions, and compilation)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-actions      # Skip updating GitHub Actions versions
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-compile      # Skip recompiling workflows (do not modify lock files)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --create-pull-request  # Upgrade and open a pull request
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --dir custom/workflows  # Upgrade workflows in custom directory
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --audit           # Check dependency health without upgrading
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --audit --json    # Output audit results in JSON format`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			dir, _ := cmd.Flags().GetString("dir")
			noFix, _ := cmd.Flags().GetBool("no-fix")
			createPRFlag, _ := cmd.Flags().GetBool("create-pull-request")
			prFlagAlias, _ := cmd.Flags().GetBool("pr")
			createPR := createPRFlag || prFlagAlias
			noActions, _ := cmd.Flags().GetBool("no-actions")
			noCompile, _ := cmd.Flags().GetBool("no-compile")
			auditFlag, _ := cmd.Flags().GetBool("audit")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			skipExtensionUpgrade, _ := cmd.Flags().GetBool("skip-extension-upgrade")

			// Handle audit mode
			if auditFlag {
				return runDependencyAudit(verbose, jsonOutput)
			}

			if createPR {
				if err := PreflightCheckForCreatePR(verbose); err != nil {
					return err
				}
			}

			if err := runUpgradeCommand(cmd.Context(), verbose, dir, noFix, noCompile, noActions, skipExtensionUpgrade); err != nil {
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
		},
	}

	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")
	cmd.Flags().Bool("no-fix", false, "Skip codemods, action version updates, and workflow compilation (only update agent files)")
	cmd.Flags().Bool("no-actions", false, "Skip updating GitHub Actions versions")
	cmd.Flags().Bool("no-compile", false, "Skip recompiling workflows (do not modify lock files)")
	cmd.Flags().Bool("create-pull-request", false, "Create a pull request with the upgrade changes")
	cmd.Flags().Bool("pr", false, "Alias for --create-pull-request")
	_ = cmd.Flags().MarkHidden("pr") // Hide the short alias from help output
	cmd.Flags().Bool("audit", false, "Check dependency health without performing upgrades")
	cmd.Flags().Bool("skip-extension-upgrade", false, "Skip automatic extension upgrade (used internally to prevent recursion after upgrade)")
	_ = cmd.Flags().MarkHidden("skip-extension-upgrade")
	addJSONFlag(cmd)

	// Register completions
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// runDependencyAudit performs a dependency health audit
func runDependencyAudit(verbose bool, jsonOutput bool) error {
	upgradeLog.Print("Running dependency health audit")

	// Generate comprehensive report
	report, err := GenerateDependencyReport(verbose)
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

// runUpgradeCommand executes the upgrade process
func runUpgradeCommand(ctx context.Context, verbose bool, workflowDir string, noFix bool, noCompile bool, noActions bool, skipExtensionUpgrade bool) error {
	upgradeLog.Printf("Running upgrade command: verbose=%v, workflowDir=%s, noFix=%v, noCompile=%v, noActions=%v, skipExtensionUpgrade=%v",
		verbose, workflowDir, noFix, noCompile, noActions, skipExtensionUpgrade)

	// Step 0b: Ensure gh-aw extension is on the latest version.
	// If the extension was just upgraded, re-launch the freshly-installed binary
	// with the same flags so that all subsequent steps (e.g. lock-file compilation)
	// use the correct new version string.  The hidden --skip-extension-upgrade flag
	// prevents the re-launched process from entering this branch again.
	if !skipExtensionUpgrade {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking gh-aw extension version..."))
		upgraded, installPath, err := upgradeExtensionIfOutdated(verbose)
		if err != nil {
			upgradeLog.Printf("Extension upgrade failed: %v", err)
			return err
		}
		if upgraded {
			upgradeLog.Print("Extension was upgraded; re-launching with new binary")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Continuing upgrade with newly installed version..."))
			// Pass installPath so relaunchWithSameArgs uses the pre-rename path;
			// on Linux os.Executable() returns a "(deleted)" suffix after the rename.
			if err := relaunchWithSameArgs("--skip-extension-upgrade", installPath); err != nil {
				return err
			}
			// The child process completed all upgrade steps (including any PR creation).
			// Exit the parent so we do not repeat those steps.
			os.Exit(0)
		}
	}

	// Step 1: Update dispatcher agent file (like init command)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating agent file..."))
	upgradeLog.Print("Updating agent file")

	if err := updateAgentFiles(verbose); err != nil {
		upgradeLog.Printf("Failed to update agent file: %v", err)
		return fmt.Errorf("failed to update agent file: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated agent file"))
	}

	// Step 2: Apply codemods to all workflows (unless --no-fix is specified)
	if !noFix {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying codemods to all workflows..."))
		upgradeLog.Print("Applying codemods to all workflows")

		fixConfig := FixConfig{
			WorkflowIDs: nil, // nil means all workflows
			Write:       true,
			Verbose:     verbose,
			WorkflowDir: workflowDir,
		}

		if err := RunFix(fixConfig); err != nil {
			upgradeLog.Printf("Failed to apply codemods: %v", err)
			// Don't fail the upgrade if fix fails - this is non-critical
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to apply codemods: %v", err)))
		}
	} else {
		upgradeLog.Print("Skipping codemods (--no-fix specified)")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping codemods (--no-fix specified)"))
		}
	}

	// Step 3: Update GitHub Actions versions (unless --no-fix or --no-actions is specified)
	if !noFix && !noActions {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating GitHub Actions versions..."))
		upgradeLog.Print("Updating GitHub Actions versions")

		if err := UpdateActions(false, verbose, false); err != nil {
			upgradeLog.Printf("Failed to update actions: %v", err)
			// Don't fail the upgrade if action updates fail - this is non-critical
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update actions: %v", err)))
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated GitHub Actions versions"))
		}
	} else {
		if noFix {
			upgradeLog.Print("Skipping action updates (--no-fix specified)")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping action updates (--no-fix specified)"))
			}
		} else if noActions {
			upgradeLog.Print("Skipping action updates (--no-actions specified)")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping action updates (--no-actions specified)"))
			}
		}
	}

	// Step 3b: Update container image digest pins (unless --no-fix or --no-actions is specified)
	// Container pins are stored alongside action pins in .github/aw/actions-lock.json.
	// Running this before compilation means the next compile step will embed the
	// pinned @sha256: references in the generated lock files.
	if !noFix && !noActions {
		upgradeLog.Print("Updating container image digest pins")
		if err := UpdateContainerPins(ctx, workflowDir, verbose); err != nil {
			upgradeLog.Printf("Failed to update container pins: %v", err)
			// Non-critical — Docker may not be available in all environments.
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update container pins: %v", err)))
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated container image pins"))
		}
	}

	// Step 4: Compile all workflows (unless --no-fix or --no-compile is specified)
	if !noFix && !noCompile {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Compiling all workflows..."))
		upgradeLog.Print("Compiling all workflows")

		// Create and configure compiler
		compiler := createAndConfigureCompiler(CompileConfig{
			Verbose:     verbose,
			WorkflowDir: workflowDir,
		})

		// Determine workflow directory
		workflowsDir := workflowDir
		if workflowsDir == "" {
			workflowsDir = ".github/workflows"
		}

		// Compile all workflow files
		stats, compileErr := compileAllWorkflowFiles(compiler, workflowsDir, verbose)
		if compileErr != nil {
			upgradeLog.Printf("Failed to compile workflows: %v", compileErr)
			// Don't fail the upgrade if compilation fails - this is non-critical
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to compile workflows: %v", compileErr)))
		} else if stats != nil {
			// Print compilation summary
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Compiled %d workflow(s)", stats.Total-stats.Errors)))
			}
			if stats.Errors > 0 {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: %d workflow(s) failed to compile", stats.Errors)))
			}
		}
	} else {
		if noFix {
			upgradeLog.Print("Skipping compilation (--no-fix specified)")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping compilation (--no-fix specified)"))
			}
		} else if noCompile {
			upgradeLog.Print("Skipping compilation (--no-compile specified)")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping compilation (--no-compile specified)"))
			}
		}
	}

	// Print success message
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Upgrade complete"))

	return nil
}

// updateAgentFiles updates the dispatcher agent file to the latest template
func updateAgentFiles(verbose bool) error {
	// Update dispatcher agent
	if err := ensureAgenticWorkflowsDispatcher(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update dispatcher agent: %v", err)
		return fmt.Errorf("failed to update dispatcher agent: %w", err)
	}

	// Upgrade copilot-setup-steps.yml version
	actionMode := workflow.DetectActionMode(GetVersion())
	if err := upgradeCopilotSetupSteps(verbose, actionMode, GetVersion()); err != nil {
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
			// Preserve the child's exit code so the caller sees the real failure.
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
