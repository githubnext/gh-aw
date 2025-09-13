package main

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [workflow-name]",
	Short: "Update workflows from installed packages",
	Long: `Update workflows from installed packages to get the latest changes.

If no workflow name is specified, all installed workflows will be updated.
If a workflow name is provided, only that specific workflow will be updated.

Examples:
  ` + constants.CLIExtensionPrefix + ` update                    # Update all workflows
  ` + constants.CLIExtensionPrefix + ` update weekly-research    # Update specific workflow
  ` + constants.CLIExtensionPrefix + ` update --staged           # Show what would be updated
  ` + constants.CLIExtensionPrefix + ` update --workflow-dir custom/workflows  # Use custom workflow directory`,
	Run: func(cmd *cobra.Command, args []string) {
		var workflowName string
		if len(args) > 0 {
			workflowName = args[0]
		}
		staged, _ := cmd.Flags().GetBool("staged")
		workflowDir, _ := cmd.Flags().GetString("workflow-dir")
		if err := cli.UpdateWorkflows(workflowName, staged, verbose, workflowDir); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}
