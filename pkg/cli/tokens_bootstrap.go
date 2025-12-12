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
	Optional    bool
}

// getRecommendedTokensForEngine returns token specs based on the workflow engine
func getRecommendedTokensForEngine(engine string) []tokenSpec {
	// Base tokens needed for most workflows
	tokens := []tokenSpec{
		{
			Name:        "GH_AW_GITHUB_TOKEN",
			When:        "Cross-repo Project Ops / remote GitHub tools",
			Description: "Fine-grained or classic PAT with contents/issues/pull-requests read+write on the repos gh-aw will touch.",
			Optional:    false,
		},
	}

	// Engine-specific tokens
	switch engine {
	case "copilot":
		tokens = append(tokens, tokenSpec{
			Name:        "COPILOT_GITHUB_TOKEN",
			When:        "Copilot workflows (CLI, engine, agent tasks, etc.)",
			Description: "PAT with Copilot Requests permission and repo access where Copilot workflows run.",
			Optional:    false,
		})
	case "claude":
		tokens = append(tokens, tokenSpec{
			Name:        "ANTHROPIC_API_KEY",
			When:        "Claude engine workflows",
			Description: "API key from Anthropic Console for Claude API access.",
			Optional:    false,
		})
	case "codex":
		tokens = append(tokens, tokenSpec{
			Name:        "OPENAI_API_KEY",
			When:        "Codex/OpenAI engine workflows",
			Description: "API key from OpenAI for Codex/GPT API access.",
			Optional:    false,
		})
	}

	// Optional tokens for advanced use cases
	tokens = append(tokens,
		tokenSpec{
			Name:        "GH_AW_AGENT_TOKEN",
			When:        "Assigning agents/bots to issues or pull requests",
			Description: "PAT for agent assignment with issues and pull-requests write on the repos where agents act.",
			Optional:    true,
		},
		tokenSpec{
			Name:        "GH_AW_GITHUB_MCP_SERVER_TOKEN",
			When:        "Isolating MCP server permissions (advanced, optional)",
			Description: "Optional read-mostly token for the GitHub MCP server when you want different scopes than GH_AW_GITHUB_TOKEN.",
			Optional:    true,
		},
	)

	return tokens
}

// recommendedTokenSpecs defines the core tokens we surface in tokens.md
// This is kept for backward compatibility and default listing
var recommendedTokenSpecs = []tokenSpec{
	{
		Name:        "GH_AW_GITHUB_TOKEN",
		When:        "Cross-repo Project Ops / remote GitHub tools",
		Description: "Fine-grained or classic PAT with contents/issues/pull-requests read+write on the repos gh-aw will touch.",
		Optional:    false,
	},
	{
		Name:        "COPILOT_GITHUB_TOKEN",
		When:        "Copilot workflows (CLI, engine, agent tasks, etc.)",
		Description: "PAT with Copilot Requests permission and repo access where Copilot workflows run.",
		Optional:    true,
	},
	{
		Name:        "ANTHROPIC_API_KEY",
		When:        "Claude engine workflows",
		Description: "API key from Anthropic Console for Claude API access.",
		Optional:    true,
	},
	{
		Name:        "OPENAI_API_KEY",
		When:        "Codex/OpenAI engine workflows",
		Description: "API key from OpenAI for Codex/GPT API access.",
		Optional:    true,
	},
	{
		Name:        "GH_AW_AGENT_TOKEN",
		When:        "Assigning agents/bots to issues or pull requests",
		Description: "PAT for agent assignment with issues and pull-requests write on the repos where agents act.",
		Optional:    true,
	},
	{
		Name:        "GH_AW_GITHUB_MCP_SERVER_TOKEN",
		When:        "Isolating MCP server permissions (advanced, optional)",
		Description: "Optional read-mostly token for the GitHub MCP server when you want different scopes than GH_AW_GITHUB_TOKEN.",
		Optional:    true,
	},
}

// NewTokensBootstrapSubcommand creates the `tokens bootstrap` subcommand
func NewTokensBootstrapSubcommand() *cobra.Command {
	var engineFlag string

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
			return runTokensBootstrap(engineFlag)
		},
	}

	cmd.Flags().StringVarP(&engineFlag, "engine", "e", "", "Check tokens for specific engine (copilot, claude, codex)")

	return cmd
}

func runTokensBootstrap(engine string) error {
	repoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		return fmt.Errorf("failed to detect current repository: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking recommended gh-aw token secrets in %s...", repoSlug)))

	// Get tokens based on engine or use all recommended tokens
	var tokensToCheck []tokenSpec
	if engine != "" {
		tokensToCheck = getRecommendedTokensForEngine(engine)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking tokens for engine: %s", engine)))
	} else {
		tokensToCheck = recommendedTokenSpecs
	}

	missing := make([]tokenSpec, 0, len(tokensToCheck))

	for _, spec := range tokensToCheck {
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

	// Separate required and optional missing secrets
	var requiredMissing, optionalMissing []tokenSpec
	for _, spec := range missing {
		if spec.Optional {
			optionalMissing = append(optionalMissing, spec)
		} else {
			requiredMissing = append(requiredMissing, spec)
		}
	}

	if len(requiredMissing) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Required gh-aw token secrets are missing:"))
		for _, spec := range requiredMissing {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret: %s", spec.Name)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("When needed: %s", spec.When)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Recommended scopes: %s", spec.Description)))
			fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("gh aw secret set %s --owner <owner> --repo <repo>", spec.Name)))
		}
	}

	if len(optionalMissing) > 0 {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Optional gh-aw token secrets are missing:"))
		for _, spec := range optionalMissing {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret: %s (optional)", spec.Name)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("When needed: %s", spec.When)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Recommended scopes: %s", spec.Description)))
			fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("gh aw secret set %s --owner <owner> --repo <repo>", spec.Name)))
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("For detailed token behavior and precedence, see the GitHub Tokens reference in the documentation."))

	return nil
}
