package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var upgradeLog = logger.New("cli:upgrade_command")

// UpgradeConfig contains configuration for the upgrade command
type UpgradeConfig struct {
	WorkflowIDs []string
	Verbose     bool
	WorkflowDir string
	NoFix       bool
}

// RunUpgrade runs the upgrade command with the given configuration
func RunUpgrade(config UpgradeConfig) error {
	return runUpgradeCommand(config.WorkflowIDs, config.Verbose, config.WorkflowDir, config.NoFix, false)
}

// NewUpgradeCommand creates the upgrade command
func NewUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [workflow]...",
		Short: "Upgrade repository with latest agent files and apply codemods to workflows",
		Long: `Upgrade the repository for the latest version of agentic workflows.

This command:
  1. Updates all agent and prompt files to the latest templates (like 'init' command)
  2. Applies automatic codemods to fix deprecated fields in workflows (like 'fix --write')

The upgrade process ensures:
- GitHub Copilot instructions are up-to-date (.github/aw/github-agentic-workflows.md)
- Dispatcher agent is current (.github/agents/agentic-workflows.agent.md)
- All workflow prompts are updated (create, update, debug, upgrade)
- Workflows use the latest syntax and configuration options
- Deprecated fields are automatically migrated

If no workflows are specified, all Markdown files in .github/workflows will be processed.

` + WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` upgrade                    # Upgrade all workflows
  ` + string(constants.CLIExtensionPrefix) + ` upgrade ci-doctor         # Upgrade specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --no-fix          # Update agent files only (skip codemods)
  ` + string(constants.CLIExtensionPrefix) + ` upgrade --dir custom/workflows  # Upgrade workflows in custom directory

After upgrading, compile workflows manually with:
  ` + string(constants.CLIExtensionPrefix) + ` compile`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			dir, _ := cmd.Flags().GetString("dir")
			noFix, _ := cmd.Flags().GetBool("no-fix")

			return runUpgradeCommand(args, verbose, dir, noFix, false)
		},
	}

	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")
	cmd.Flags().Bool("no-fix", false, "Skip applying codemods to workflows (only update agent files)")

	// Register completions
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// runUpgradeCommand executes the upgrade process
func runUpgradeCommand(workflowIDs []string, verbose bool, workflowDir string, noFix bool, noCompile bool) error {
	upgradeLog.Printf("Running upgrade command: workflowIDs=%v, verbose=%v, workflowDir=%s, noFix=%v, noCompile=%v",
		workflowIDs, verbose, workflowDir, noFix, noCompile)

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

	// Step 2: Apply codemods (unless --no-fix is specified)
	if !noFix {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying codemods to workflows..."))
		upgradeLog.Print("Applying codemods to workflows")

		fixConfig := FixConfig{
			WorkflowIDs: workflowIDs,
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

	// Step 3: Compile workflows (only if explicitly requested - compilation is optional)
	// Skipping compilation by default avoids issues when workflows might have other errors
	if !noCompile && !noFix {
		upgradeLog.Print("Skipping compilation (can be enabled with future --compile flag)")
		// Note: We skip compilation by default to avoid crashing on workflows that might have
		// validation errors. Users can compile manually after upgrade with: gh aw compile
	}

	// Print success message
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Upgrade complete"))

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
