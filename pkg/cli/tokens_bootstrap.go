package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/spf13/cobra"
)

// tokenSpec describes a recommended token secret for gh-aw
type tokenSpec struct {
	Name        string
	When        string
	Description string
}

// recommendedTokenSpecs defines the core tokens we surface in tokens.md
var recommendedTokenSpecs = []tokenSpec{
	{
		Name:        "GH_AW_GITHUB_TOKEN",
		When:        "Cross-repo Project Ops / remote GitHub tools",
		Description: "Fine-grained or classic PAT with contents/issues/pull-requests read+write on the repos gh-aw will touch.",
	},
	{
		Name:        "COPILOT_GITHUB_TOKEN",
		When:        "Copilot workflows (CLI, engine, agent tasks, etc.)",
		Description: "PAT with Copilot Requests permission and repo access where Copilot workflows run.",
	},
	{
		Name:        "GH_AW_AGENT_TOKEN",
		When:        "Assigning agents/bots to issues or pull requests",
		Description: "PAT for agent assignment with issues and pull-requests write on the repos where agents act.",
	},
	{
		Name:        "GH_AW_GITHUB_MCP_SERVER_TOKEN",
		When:        "Isolating MCP server permissions (advanced, optional)",
		Description: "Optional read-mostly token for the GitHub MCP server when you want different scopes than GH_AW_GITHUB_TOKEN.",
	},
}

// NewTokensBootstrapSubcommand creates the `tokens bootstrap` subcommand
func NewTokensBootstrapSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Check and suggest setup for gh-aw GitHub token secrets",
		Long: `Check which recommended GitHub token secrets (like GH_AW_GITHUB_TOKEN)
are configured for the current repository, and print least-privilege setup
instructions for any that are missing.

This command is read-only: it does not create tokens or secrets for you.
Instead, it inspects repository secrets (using the GitHub CLI where
available) and prints the exact secrets to add and suggested scopes.

For full details, including precedence rules, see the GitHub Tokens
reference in the documentation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTokensBootstrap()
		},
	}

	return cmd
}

func runTokensBootstrap() error {
	repoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		return fmt.Errorf("failed to detect current repository: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking recommended gh-aw token secrets in %s...", repoSlug)))

	missing := make([]tokenSpec, 0, len(recommendedTokenSpecs))

	for _, spec := range recommendedTokenSpecs {
		exists, err := checkSecretExists(spec.Name)
		if err != nil {
			// If we hit a 403 or other error, surface a friendly message and abort
			return fmt.Errorf("unable to inspect repository secrets (gh secret list failed for %s): %w", spec.Name, err)
		}
		if !exists {
			missing = append(missing, spec)
		}
	}

	if len(missing) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All recommended gh-aw token secrets are present in this repository."))
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Some recommended gh-aw token secrets are missing:"))
	for _, spec := range missing {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret: %s", spec.Name)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("When needed: %s", spec.When)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Recommended scopes: %s", spec.Description)))
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("gh secret set %s -a actions", spec.Name)))
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("For detailed token behavior and precedence, see the GitHub Tokens reference in the documentation."))

	return nil
}
