package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var upgradeLog = logger.New("cli:upgrade_command")

// UpgradeConfig contains configuration for the upgrade command
type UpgradeConfig struct {
	Verbose     bool
	WorkflowDir string
	NoFix       bool
	Push        bool
}

// RunUpgrade runs the upgrade command with the given configuration
func RunUpgrade(config UpgradeConfig) error {
	return runUpgradeCommand(config.Verbose, config.WorkflowDir, config.NoFix, false, config.Push)
}

// NewUpgradeCommand creates the upgrade command
func NewUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade repository with latest agent files and apply codemods to all workflows",
		Long: `Upgrade the repository for the latest version of agentic workflows.

This command:
  1. Updates all agent and prompt files to the latest templates (like 'init' command)
  2. Applies automatic codemods to fix deprecated fields in all workflows (like 'fix --write')
  3. Compiles all workflows to generate lock files (like 'compile' command)

The upgrade process ensures:
- GitHub Copilot instructions are up-to-date (.github/aw/github-agentic-workflows.md)
- Dispatcher agent is current (.github/agents/agentic-workflows.agent.md)
- All workflow prompts are updated (create, update, debug, upgrade)
- All workflows use the latest syntax and configuration options
- Deprecated fields are automatically migrated across all workflows
- All workflows are compiled and lock files are up-to-date

This command always upgrades all Markdown files in .github/workflows.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` upgrade                    # Upgrade all workflows
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-fix          # Update agent files only (skip codemods and compilation)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --push            # Upgrade and automatically commit/push changes
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --dir custom/workflows  # Upgrade workflows in custom directory`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			dir, _ := cmd.Flags().GetString("dir")
			noFix, _ := cmd.Flags().GetBool("no-fix")
			push, _ := cmd.Flags().GetBool("push")

			return runUpgradeCommand(verbose, dir, noFix, false, push)
		},
	}

	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")
	cmd.Flags().Bool("no-fix", false, "Skip applying codemods and compiling workflows (only update agent files)")
	cmd.Flags().Bool("push", false, "Automatically commit and push changes after successful upgrade")

	// Register completions
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// runUpgradeCommand executes the upgrade process
func runUpgradeCommand(verbose bool, workflowDir string, noFix bool, noCompile bool, push bool) error {
	upgradeLog.Printf("Running upgrade command: verbose=%v, workflowDir=%s, noFix=%v, noCompile=%v, push=%v",
		verbose, workflowDir, noFix, noCompile, push)

	// Step 0a: If --push is enabled, ensure git status is clean before starting
	if push {
		upgradeLog.Print("Checking for clean working directory (--push enabled)")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking git status..."))
		if err := checkCleanWorkingDirectory(verbose); err != nil {
			upgradeLog.Printf("Git status check failed: %v", err)
			return fmt.Errorf("--push requires a clean working directory: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Working directory is clean"))
		}
	}

	// Step 0b: Ensure gh-aw extension is on the latest version
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking gh-aw extension version..."))
	if err := ensureLatestExtensionVersion(verbose); err != nil {
		upgradeLog.Printf("Extension version check failed: %v", err)
		return err
	}

	// Step 1: Update all agent and prompt files (like init command)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating agent and prompt files..."))
	upgradeLog.Print("Updating agent and prompt files")

	if err := updateAgentFiles(verbose); err != nil {
		upgradeLog.Printf("Failed to update agent files: %v", err)
		return fmt.Errorf("failed to update agent files: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated agent and prompt files"))
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

	// Step 3: Compile all workflows (unless --no-fix is specified)
	if !noFix {
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
		upgradeLog.Print("Skipping compilation (--no-fix specified)")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping compilation (--no-fix specified)"))
		}
	}

	// Print success message
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Upgrade complete"))

	// Step 4: If --push is enabled, commit and push changes
	if push {
		upgradeLog.Print("Push enabled - preparing to commit and push changes")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Preparing to commit and push changes..."))

		// Check if there are any changes to commit
		upgradeLog.Print("Checking for modified files")
		cmd := exec.Command("git", "status", "--porcelain")
		output, err := cmd.Output()
		if err != nil {
			upgradeLog.Printf("Failed to check git status: %v", err)
			return fmt.Errorf("failed to check git status: %w", err)
		}

		if len(strings.TrimSpace(string(output))) == 0 {
			upgradeLog.Print("No changes to commit")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes to commit"))
			return nil
		}

		// Pull latest changes from remote before committing (if remote exists)
		upgradeLog.Print("Checking for remote repository")
		checkRemoteCmd := exec.Command("git", "remote", "get-url", "origin")
		if err := checkRemoteCmd.Run(); err == nil {
			// Remote exists, pull changes
			upgradeLog.Print("Pulling latest changes from remote")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Pulling latest changes from remote..."))
			}
			pullCmd := exec.Command("git", "pull", "--rebase")
			if output, err := pullCmd.CombinedOutput(); err != nil {
				upgradeLog.Printf("Failed to pull changes: %v, output: %s", err, string(output))
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to pull changes: %v", err)))
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("You may need to manually resolve conflicts and push"))
				return fmt.Errorf("failed to pull changes: %w", err)
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Pulled latest changes"))
			}
		} else {
			upgradeLog.Print("No remote repository configured, skipping pull")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No remote repository configured, skipping pull"))
			}
		}

		// Stage all modified files
		upgradeLog.Print("Staging all changes")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Staging changes..."))
		}
		addCmd := exec.Command("git", "add", "-A")
		if output, err := addCmd.CombinedOutput(); err != nil {
			upgradeLog.Printf("Failed to stage changes: %v, output: %s", err, string(output))
			return fmt.Errorf("failed to stage changes: %w", err)
		}

		// Commit the changes
		upgradeLog.Print("Committing changes")
		commitMessage := "chore: upgrade agentic workflows"
		if err := commitChanges(commitMessage, verbose); err != nil {
			upgradeLog.Printf("Failed to commit changes: %v", err)
			return fmt.Errorf("failed to commit changes: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes committed"))
		}

		// Push the changes (only if remote exists)
		upgradeLog.Print("Checking if remote repository exists for push")
		checkRemoteCmd = exec.Command("git", "remote", "get-url", "origin")
		if err := checkRemoteCmd.Run(); err == nil {
			// Remote exists, push changes
			upgradeLog.Print("Pushing changes to remote")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Pushing to remote..."))
			}
			pushCmd := exec.Command("git", "push")
			if output, err := pushCmd.CombinedOutput(); err != nil {
				upgradeLog.Printf("Failed to push changes: %v, output: %s", err, string(output))
				return fmt.Errorf("failed to push changes: %w\nOutput: %s", err, string(output))
			}

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes pushed to remote"))
		} else {
			upgradeLog.Print("No remote repository configured, skipping push")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No remote repository configured, changes committed locally"))
			}
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes committed locally (no remote configured)"))
		}
	}

	return nil
}

// updateAgentFiles updates all agent and prompt files to the latest templates
func updateAgentFiles(verbose bool) error {
	// Update copilot instructions
	if err := ensureCopilotInstructions(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update copilot instructions: %v", err)
		return fmt.Errorf("failed to update copilot instructions: %w", err)
	}

	// Update dispatcher agent
	if err := ensureAgenticWorkflowsDispatcher(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update dispatcher agent: %v", err)
		return fmt.Errorf("failed to update dispatcher agent: %w", err)
	}

	// Update create workflow prompt
	if err := ensureCreateWorkflowPrompt(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update create workflow prompt: %v", err)
		return fmt.Errorf("failed to update create workflow prompt: %w", err)
	}

	// Update update workflow prompt
	if err := ensureUpdateWorkflowPrompt(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update update workflow prompt: %v", err)
		return fmt.Errorf("failed to update update workflow prompt: %w", err)
	}

	// Update create shared agentic workflow prompt
	if err := ensureCreateSharedAgenticWorkflowPrompt(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update create shared workflow prompt: %v", err)
		return fmt.Errorf("failed to update create shared workflow prompt: %w", err)
	}

	// Update debug workflow prompt
	if err := ensureDebugWorkflowPrompt(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update debug workflow prompt: %v", err)
		return fmt.Errorf("failed to update debug workflow prompt: %w", err)
	}

	// Update upgrade agentic workflows prompt
	if err := ensureUpgradeAgenticWorkflowsPrompt(verbose, false); err != nil {
		upgradeLog.Printf("Failed to update upgrade workflows prompt: %v", err)
		return fmt.Errorf("failed to update upgrade workflows prompt: %w", err)
	}

	return nil
}
