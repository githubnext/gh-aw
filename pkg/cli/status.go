package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/spf13/cobra"
)

// NewStatusCommand creates the status command
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [pattern]",
		Short: "Show status of natural language action files and workflows",
		Run: func(cmd *cobra.Command, args []string) {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			if err := StatusWorkflows(pattern, verbose, jsonFlag); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().Bool("json", false, "Output status in JSON format")

	return cmd
}
