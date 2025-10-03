package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewAddCommand creates the add command
func NewAddCommand(verbose bool, validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <workflow>...",
		Short: "Add one or more workflows from the components to .github/workflows",
		Long: `Add one or more workflows from repositories to .github/workflows.

Examples:
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/ci-doctor
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/ci-doctor@v1.0.0
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/workflows/ci-doctor.md@main
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/ci-doctor --pr --force

Workflow specifications:
  - Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)
  - Four+ parts: "owner/repo/workflows/workflow-name.md[@version]" (requires explicit .md extension)
  - Version can be tag, branch, or SHA

The -n flag allows you to specify a custom name for the workflow file (only applies to the first workflow when adding multiple).
The --pr flag automatically creates a pull request with the workflow changes.
The --force flag overwrites existing workflow files.`,
		Args: func(cmd *cobra.Command, args []string) error {
			// If no arguments provided and not in CI, automatically use interactive mode
			if len(args) == 0 && !IsRunningInCI() {
				// Auto-enable interactive mode
				var workflowName = "my-workflow" // Default name
				if err := CreateWorkflowInteractively(workflowName, verbose, false); err != nil {
					return fmt.Errorf("failed to create workflow interactively: %w", err)
				}
				// Exit successfully after interactive creation
				os.Exit(0)
			}

			// Normal mode requires at least one workflow
			if len(args) < 1 {
				return fmt.Errorf("requires at least 1 arg(s), received %d", len(args))
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			workflows := args
			numberFlag, _ := cmd.Flags().GetInt("number")
			engineOverride, _ := cmd.Flags().GetString("engine")
			repoFlag, _ := cmd.Flags().GetString("repo")
			nameFlag, _ := cmd.Flags().GetString("name")
			prFlag, _ := cmd.Flags().GetBool("pr")
			forceFlag, _ := cmd.Flags().GetBool("force")

			if err := validateEngine(engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}

			// Handle normal mode
			if prFlag {
				if err := AddWorkflows(workflows, numberFlag, verbose, engineOverride, repoFlag, nameFlag, forceFlag, true); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
					os.Exit(1)
				}
			} else {
				if err := AddWorkflows(workflows, numberFlag, verbose, engineOverride, repoFlag, nameFlag, forceFlag, false); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
					os.Exit(1)
				}
			}
		},
	}

	// Add number flag to add command
	cmd.Flags().IntP("number", "c", 1, "Create multiple numbered copies")

	// Add name flag to add command
	cmd.Flags().StringP("name", "n", "", "Specify name for the added workflow (without .md extension)")

	// Add AI flag to add command
	cmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex, copilot, custom)")

	// Add repository flag to add command
	cmd.Flags().StringP("repo", "r", "", "Install and use workflows from specified repository (org/repo)")

	// Add PR flag to add command
	cmd.Flags().Bool("pr", false, "Create a pull request with the workflow changes")

	// Add force flag to add command
	cmd.Flags().Bool("force", false, "Overwrite existing workflow files")

	return cmd
}
