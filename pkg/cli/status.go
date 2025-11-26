package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewStatusCommand creates the status command
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [pattern]",
		Short: "Show status of agentic workflows",
		Long: `Show status of all agentic workflows in the repository.

Displays a table with workflow name, AI engine, compilation status, enabled/disabled state,
and time remaining until expiration (if stop-after is configured).

The optional pattern argument filters workflows by name (case-insensitive substring match).

Examples:
  ` + constants.CLIExtensionPrefix + ` status                    # Show all workflow status
  ` + constants.CLIExtensionPrefix + ` status ci-                 # Show workflows with 'ci-' in name
  ` + constants.CLIExtensionPrefix + ` status --json              # Output in JSON format
  ` + constants.CLIExtensionPrefix + ` status --ref main          # Show latest run status for main branch`,
		Run: func(cmd *cobra.Command, args []string) {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ref, _ := cmd.Flags().GetString("ref")
			if err := StatusWorkflows(pattern, verbose, jsonFlag, ref); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().Bool("json", false, "Output results in JSON format")
	cmd.Flags().String("ref", "", "Filter runs by branch or tag name (e.g., main, v1.0.0)")

	return cmd
}
