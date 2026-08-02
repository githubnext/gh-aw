package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newLegacyGHGuardSubcommand rejects legacy-style nested "gh" invocations such as
// "gh aw <command> gh --help" so they do not silently render parent help.
func newLegacyGHGuardSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:                "gh",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("unknown command %q for %q", cmd.Name(), cmd.Parent().CommandPath())
		},
	}
}
