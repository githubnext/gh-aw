package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var updateLog = logger.New("cli:update_command")

// NewUpdateCommand creates the update command
func NewUpdateCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [workflow]...",
		Short: "Update agentic workflows from their source repositories",
		Long: `Update one or more workflows from their source repositories.

The update command fetches the latest version of each workflow from its source
repository, merges upstream changes with any local modifications, and recompiles.

If no workflow names are specified, all workflows with a 'source' field are updated.

By default, the update performs a 3-way merge to preserve your local changes.
Use --no-merge to override local changes with the upstream version.

For workflow updates, it fetches the latest version based on the current ref:
- If the ref is a tag, it updates to the latest release (use --major for major version updates)
- If the ref is a branch, it fetches the latest commit from that branch
- If the ref is a commit SHA, it fetches the latest commit from the default branch

For extension updates, action updates, agent files, and codemods, use 'gh aw upgrade'.

` + WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` update                    # Update all workflows from source
  ` + string(constants.CLIExtensionPrefix) + ` update repo-assist        # Update a specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` update repo-assist.md     # Same (alternative format)
  ` + string(constants.CLIExtensionPrefix) + ` update --no-merge         # Override local changes with upstream
  ` + string(constants.CLIExtensionPrefix) + ` update repo-assist --major # Allow major version updates
  ` + string(constants.CLIExtensionPrefix) + ` update --force            # Force update even if no changes
  ` + string(constants.CLIExtensionPrefix) + ` update --disable-release-bump  # Update without force-bumping all action versions
  ` + string(constants.CLIExtensionPrefix) + ` update --dir custom/workflows  # Update workflows in custom directory
  ` + string(constants.CLIExtensionPrefix) + ` update --create-pull-request   # Update and open a pull request`,
		RunE: func(cmd *cobra.Command, args []string) error {
			majorFlag, _ := cmd.Flags().GetBool("major")
			forceFlag, _ := cmd.Flags().GetBool("force")
			engineOverride, _ := cmd.Flags().GetString("engine")
			verbose, _ := cmd.Flags().GetBool("verbose")
			workflowDir, _ := cmd.Flags().GetString("dir")
			noStopAfter, _ := cmd.Flags().GetBool("no-stop-after")
			stopAfter, _ := cmd.Flags().GetString("stop-after")
			noMergeFlag, _ := cmd.Flags().GetBool("no-merge")
			disableReleaseBump, _ := cmd.Flags().GetBool("disable-release-bump")
			createPRFlag, _ := cmd.Flags().GetBool("create-pull-request")
			prFlagAlias, _ := cmd.Flags().GetBool("pr")
			createPR := createPRFlag || prFlagAlias

			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			if createPR {
				if err := PreflightCheckForCreatePR(verbose); err != nil {
					return err
				}
			}

			if err := RunUpdateWorkflows(args, majorFlag, forceFlag, verbose, engineOverride, workflowDir, noStopAfter, stopAfter, noMergeFlag, disableReleaseBump); err != nil {
				return err
			}

			if createPR {
				prBody := "This PR updates agentic workflows from their source repositories."
				_, err := CreatePRWithChanges("update-workflows", "chore: update workflows",
					"Update workflows from source", prBody, verbose)
				return err
			}
			return nil
		},
	}

	cmd.Flags().Bool("major", false, "Allow major version updates when updating tagged releases")
	cmd.Flags().BoolP("force", "f", false, "Force update even if no changes are detected")
	addEngineFlag(cmd)
	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")
	cmd.Flags().Bool("no-stop-after", false, "Remove any stop-after field from the workflow")
	cmd.Flags().String("stop-after", "", "Override stop-after value in the workflow (e.g., '+48h', '2025-12-31 23:59:59')")
	cmd.Flags().Bool("no-merge", false, "Override local changes with upstream version instead of merging")
	cmd.Flags().Bool("disable-release-bump", false, "Disable automatic major version bumps for all actions (only core actions/* are force-updated)")
	cmd.Flags().Bool("create-pull-request", false, "Create a pull request with the update changes")
	cmd.Flags().Bool("pr", false, "Alias for --create-pull-request")
	_ = cmd.Flags().MarkHidden("pr") // Hide the short alias from help output

	// Register completions for update command
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// RunUpdateWorkflows updates workflows from their source repositories.
// Each workflow is compiled immediately after update.
func RunUpdateWorkflows(workflowNames []string, allowMajor, force, verbose bool, engineOverride string, workflowsDir string, noStopAfter bool, stopAfter string, noMerge bool, disableReleaseBump bool) error {
	updateLog.Printf("Starting update process: workflows=%v, allowMajor=%v, force=%v, noMerge=%v, disableReleaseBump=%v", workflowNames, allowMajor, force, noMerge, disableReleaseBump)

	var firstErr error

	if err := UpdateWorkflows(workflowNames, allowMajor, force, verbose, engineOverride, workflowsDir, noStopAfter, stopAfter, noMerge); err != nil {
		firstErr = fmt.Errorf("workflow update failed: %w", err)
	}

	// Update GitHub Actions versions in actions-lock.json.
	// By default all actions are updated to the latest major version.
	// Pass --disable-release-bump to revert to only forcing updates for core (actions/*) actions.
	if err := UpdateActions(allowMajor, verbose, disableReleaseBump); err != nil {
		// Non-fatal: warn but don't fail the update
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update actions-lock.json: %v", err)))
	}

	// Update action references in user-provided steps within workflow .md files.
	// By default all org/repo@version references are updated to the latest major version.
	if err := UpdateActionsInWorkflowFiles(workflowsDir, engineOverride, verbose, disableReleaseBump); err != nil {
		// Non-fatal: warn but don't fail the update
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update action references in workflow files: %v", err)))
	}

	return firstErr
}
