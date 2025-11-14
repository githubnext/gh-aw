package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewNewCommand creates the new workflow command
func NewNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [workflow-base-name]",
		Short: "Create a new workflow Markdown file with example configuration",
		Long: `Create a new workflow Markdown file with commented examples and explanations of all available options.

When called without a workflow name (or with --interactive flag), launches an interactive wizard
to guide you through creating a workflow with custom settings.

When called with a workflow name, creates a template file with comprehensive examples of:
- All trigger types (on: events)
- Permissions configuration
- AI processor settings
- Tools configuration (github, claude, mcps)
- All frontmatter options with explanations

Examples:
  ` + constants.CLIExtensionPrefix + ` new                      # Interactive mode
  ` + constants.CLIExtensionPrefix + ` new --interactive        # Interactive mode (explicit)
  ` + constants.CLIExtensionPrefix + ` new my-workflow          # Create template file
  ` + constants.CLIExtensionPrefix + ` new issue-handler --force`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			forceFlag, _ := cmd.Flags().GetBool("force")
			verbose, _ := cmd.Flags().GetBool("verbose")
			interactiveFlag, _ := cmd.Flags().GetBool("interactive")

			// If no arguments provided or interactive flag is set, use interactive mode
			if len(args) == 0 || interactiveFlag {
				// Check if running in CI environment
				if IsRunningInCI() {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Interactive mode cannot be used in CI environments. Please provide a workflow name."))
					os.Exit(1)
				}

				// Use default workflow name for interactive mode
				workflowName := "my-workflow"
				if len(args) > 0 {
					workflowName = args[0]
				}

				if err := CreateWorkflowInteractively(workflowName, verbose, forceFlag); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
					os.Exit(1)
				}
				return
			}

			// Template mode with workflow name
			workflowName := args[0]
			if err := NewWorkflow(workflowName, verbose, forceFlag); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().Bool("force", false, "Overwrite existing workflow files")
	cmd.Flags().BoolP("interactive", "i", false, "Launch interactive workflow creation wizard")

	return cmd
}
