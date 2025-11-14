package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewDisableCommand creates the disable command
func NewDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable [workflow-id]...",
		Short: "Disable agentic workflows and cancel any in-progress runs",
		Long: `Disable one or more workflows by ID, or all workflows if no IDs are provided.

Examples:
  ` + constants.CLIExtensionPrefix + ` disable                    # Disable all workflows
  ` + constants.CLIExtensionPrefix + ` disable ci-doctor         # Disable specific workflow
  ` + constants.CLIExtensionPrefix + ` disable ci-doctor daily   # Disable multiple workflows`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := DisableWorkflowsByNames(args); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}
}
