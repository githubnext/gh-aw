package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewEnableCommand creates the enable command
func NewEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable [workflow-id]...",
		Short: "Enable agentic workflows",
		Long: `Enable one or more workflows by ID, or all workflows if no IDs are provided.

Examples:
  ` + constants.CLIExtensionPrefix + ` enable                    # Enable all workflows
  ` + constants.CLIExtensionPrefix + ` enable ci-doctor         # Enable specific workflow
  ` + constants.CLIExtensionPrefix + ` enable ci-doctor daily   # Enable multiple workflows`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := EnableWorkflowsByNames(args); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}
}
