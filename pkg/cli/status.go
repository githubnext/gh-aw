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
		Short: "Show status of agentic workflows",
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
