package cli

import (
	"context"
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
		Long: `Update one or more agentic workflows from their source repositories.

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
  ` + string(constants.CLIExtensionPrefix) + ` update --no-compile           # Update without regenerating lock files
  ` + string(constants.CLIExtensionPrefix) + ` update --no-redirect          # Refuse workflows that use redirect frontmatter
  ` + string(constants.CLIExtensionPrefix) + ` update --dir custom/workflows  # Update workflows in custom directory
  ` + string(constants.CLIExtensionPrefix) + ` update --create-pull-request   # Update and open a pull request
  ` + string(constants.CLIExtensionPrefix) + ` update --cool-down 0           # Disable cooldown and apply all pending releases immediately
  ` + string(constants.CLIExtensionPrefix) + ` update --cool-down 3d          # Apply a custom 3-day cooldown period`,
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
			noCompile, _ := cmd.Flags().GetBool("no-compile")
			noRedirect, _ := cmd.Flags().GetBool("no-redirect")
			createPRFlag, _ := cmd.Flags().GetBool("create-pull-request")
			prFlagAlias, _ := cmd.Flags().GetBool("pr")
			createPR := createPRFlag || prFlagAlias
			coolDownStr, _ := cmd.Flags().GetString("cool-down")

			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			coolDown, err := parseCoolDownFlag(coolDownStr)
			if err != nil {
				return fmt.Errorf("invalid --cool-down value: %w", err)
			}

			if createPR {
				if err := PreflightCheckForCreatePR(verbose); err != nil {
					return err
				}
			}

			opts := UpdateWorkflowsOptions{
				WorkflowNames:      args,
				AllowMajor:         majorFlag,
				Force:              forceFlag,
				Verbose:            verbose,
				EngineOverride:     engineOverride,
				WorkflowsDir:       workflowDir,
				NoStopAfter:        noStopAfter,
				StopAfter:          stopAfter,
				NoMerge:            noMergeFlag,
				DisableReleaseBump: disableReleaseBump,
				NoCompile:          noCompile,
				NoRedirect:         noRedirect,
				CoolDown:           coolDown,
			}

			if err := RunUpdateWorkflows(cmd.Context(), opts); err != nil {
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
	cmd.Flags().Bool("no-compile", false, "Skip recompiling workflows (do not modify lock files)")
	cmd.Flags().Bool("no-redirect", false, "Refuse updates when redirect frontmatter is present")
	cmd.Flags().Bool("create-pull-request", false, "Create a pull request with the update changes")
	cmd.Flags().Bool("pr", false, "Alias for --create-pull-request")
	cmd.Flags().String("cool-down", "7d", "Cooldown period before applying a new release (e.g. 7d, 24h, 0 to disable). Does not apply to actions/* or github/* repositories")
	_ = cmd.Flags().MarkHidden("pr") // Hide the short alias from help output

	// Register completions for update command
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// RunUpdateWorkflows updates workflows from their source repositories.
// Each workflow is compiled immediately after update.
func RunUpdateWorkflows(ctx context.Context, opts UpdateWorkflowsOptions) error {
	updateLog.Printf("Starting update process: workflows=%v, allowMajor=%v, force=%v, noMerge=%v, disableReleaseBump=%v, noCompile=%v, noRedirect=%v, coolDown=%v", opts.WorkflowNames, opts.AllowMajor, opts.Force, opts.NoMerge, opts.DisableReleaseBump, opts.NoCompile, opts.NoRedirect, opts.CoolDown)

	var firstErr error

	if err := UpdateWorkflows(ctx, opts); err != nil {
		firstErr = fmt.Errorf("workflow update failed: %w", err)
	}

	// Update GitHub Actions versions in actions-lock.json.
	// By default all actions are updated to the latest major version.
	// Pass --disable-release-bump to revert to only forcing updates for core (actions/*) actions.
	updateLog.Printf("Updating GitHub Actions versions in actions-lock.json: allowMajor=%v, disableReleaseBump=%v", opts.AllowMajor, opts.DisableReleaseBump)
	if err := UpdateActions(ctx, opts.AllowMajor, opts.Verbose, opts.DisableReleaseBump, opts.CoolDown); err != nil {
		// Non-fatal: warn but don't fail the update
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update actions-lock.json: %v", err)))
	}

	// Resolve and store SHA-256 digest pins for container images referenced in lock files.
	updateLog.Print("Updating container image digest pins")
	if err := UpdateContainerPins(ctx, opts.WorkflowsDir, opts.Verbose); err != nil {
		// Non-fatal: Docker may not be available in all environments.
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update container pins: %v", err)))
	}

	// Update action references in user-provided steps within workflow .md files.
	// By default all org/repo@version references are updated to the latest major version.
	updateLog.Print("Updating action references in workflow .md files")
	if err := UpdateActionsInWorkflowFiles(ctx, opts.WorkflowsDir, opts.EngineOverride, opts.Verbose, opts.DisableReleaseBump, opts.NoCompile, opts.CoolDown); err != nil {
		// Non-fatal: warn but don't fail the update
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update action references in workflow files: %v", err)))
	}

	updateLog.Printf("Update process complete: had_error=%v", firstErr != nil)
	return firstErr
}
