package cli

import (
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var secretsCommandLog = logger.New("cli:secrets_command")

// NewSecretsCommand creates the main secrets command with subcommands
func NewSecretsCommand() *cobra.Command {
	secretsCommandLog.Print("Creating secrets command with subcommands")
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage repository secrets and GitHub tokens",
		Long: `Manage GitHub Actions secrets and tokens for GitHub Agentic Workflows.

Use this command to set secrets for workflows and check which recommended
token secrets are configured for your repository.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newSecretsSetSubcommand())
	cmd.AddCommand(newSecretsBootstrapSubcommand())

	return cmd
}
