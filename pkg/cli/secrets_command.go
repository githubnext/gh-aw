package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var secretsCommandLog = logger.New("cli:secrets_command")

// NewSecretsCommand creates the main secrets command with subcommands
func NewSecretsCommand() *cobra.Command {
	secretsCommandLog.Print("Creating secrets command with subcommands")
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage repository secrets",
		Long: `Manage GitHub Actions secrets for GitHub Agentic Workflows.

This command provides tools for managing secrets required by agentic workflows, including
AI API keys (Anthropic, OpenAI, GitHub Copilot) and GitHub tokens for workflow execution.

Available subcommands:
  • set       - Create or update individual secrets
  • bootstrap - Validate and configure all required secrets for workflows

Examples:
  gh aw secrets set MY_SECRET --value "secret123"    # Set a secret directly
  gh aw secrets bootstrap                             # Check all required secrets`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newSecretsSetSubcommand())
	cmd.AddCommand(newSecretsBootstrapSubcommand())

	return cmd
}
