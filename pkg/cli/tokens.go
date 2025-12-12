package cli

import (
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var tokensCommandLog = logger.New("cli:tokens")

// NewTokensCommand creates the main tokens command with subcommands
func NewTokensCommand() *cobra.Command {
	tokensCommandLog.Print("Creating tokens command with subcommands")
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Inspect and bootstrap GitHub tokens for gh-aw",
		Long: `Token utilities for GitHub Agentic Workflows.

Use this command to check which recommended secrets are configured
for the current repository and to see how to create them with
minimum required permissions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewTokensBootstrapSubcommand())

	return cmd
}
