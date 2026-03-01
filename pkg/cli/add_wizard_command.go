package cli

import (
	"errors"
	"os"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/tty"
	"github.com/spf13/cobra"
)

// NewAddWizardCommand creates the add-wizard command, which is always interactive.
func NewAddWizardCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-wizard <workflow>...",
		Short: "Interactively add an agentic workflow with guided setup",
		Long: `Interactively add one or more workflows with guided setup.

This command walks you through:
  - Selecting an AI engine (Copilot, Claude, or Codex)
  - Configuring API keys and secrets
  - Creating a pull request with the workflow
  - Optionally running the workflow immediately

Use 'add' for non-interactive workflow addition.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` add-wizard githubnext/agentics/daily-repo-status    # Guided setup
  ` + string(constants.CLIExtensionPrefix) + ` add-wizard githubnext/agentics/ci-doctor@v1.0.0     # Guided setup with version
  ` + string(constants.CLIExtensionPrefix) + ` add-wizard ./my-workflow.md                         # Guided setup for local workflow
  ` + string(constants.CLIExtensionPrefix) + ` add-wizard githubnext/agentics/ci-doctor --engine copilot  # Pre-select engine

Workflow specifications:
  - Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)
  - Four+ parts: "owner/repo/workflows/workflow-name.md[@version]" (requires explicit .md extension)
  - GitHub URL: "https://github.com/owner/repo/blob/branch/path/to/workflow.md"
  - Local file: "./path/to/workflow.md"
  - Version can be tag, branch, or SHA (for remote workflows)

Note: Requires an interactive terminal. Use 'add' for CI/automation environments.
Note: To create a new workflow from scratch, use the 'new' command instead.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing workflow specification\n\nRun 'gh aw add-wizard --help' for usage information")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			workflows := args
			engineOverride, _ := cmd.Flags().GetString("engine")
			verbose, _ := cmd.Flags().GetBool("verbose")
			noGitattributes, _ := cmd.Flags().GetBool("no-gitattributes")
			workflowDir, _ := cmd.Flags().GetString("dir")
			noStopAfter, _ := cmd.Flags().GetBool("no-stop-after")
			stopAfter, _ := cmd.Flags().GetString("stop-after")

			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			// add-wizard requires an interactive terminal
			if !tty.IsStdoutTerminal() || os.Getenv("CI") != "" {
				return errors.New("add-wizard requires an interactive terminal; use 'add' for non-interactive environments")
			}

			return RunAddInteractive(cmd.Context(), workflows, verbose, engineOverride, noGitattributes, workflowDir, noStopAfter, stopAfter)
		},
	}

	// Add AI engine flag
	addEngineFlag(cmd)

	// Add no-gitattributes flag
	cmd.Flags().Bool("no-gitattributes", false, "Skip updating .gitattributes file")

	// Add workflow directory flag
	cmd.Flags().StringP("dir", "d", "", "Subdirectory under .github/workflows/ (e.g., 'shared' creates .github/workflows/shared/)")

	// Add no-stop-after flag
	cmd.Flags().Bool("no-stop-after", false, "Remove any stop-after field from the workflow")

	// Add stop-after flag
	cmd.Flags().String("stop-after", "", "Override stop-after value in the workflow (e.g., '+48h', '2025-12-31 23:59:59')")

	// Register completions
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}
