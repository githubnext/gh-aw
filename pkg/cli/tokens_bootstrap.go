package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var tokensBootstrapLog = logger.New("cli:tokens_bootstrap")

// newSecretsBootstrapSubcommand creates the `secrets bootstrap` subcommand
func newSecretsBootstrapSubcommand() *cobra.Command {
	var engineFlag string
	var ownerFlag string
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Check and suggest setup for gh aw GitHub token secrets",
		Long: `Check which recommended GitHub token secrets (like GH_AW_GITHUB_TOKEN)
are configured for the current repository, and print least-privilege setup
instructions for any that are missing.

This command is read-only: it does not create tokens or secrets for you.
Instead, it inspects repository secrets (using the GitHub CLI where
available) and prints the exact secrets to add and suggested scopes.

For full details, including precedence rules, see the GitHub Tokens
reference in the documentation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTokensBootstrap(engineFlag, ownerFlag, repoFlag)
		},
	}

	cmd.Flags().StringVarP(&engineFlag, "engine", "e", "", "Check tokens for specific engine (copilot, claude, codex)")
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Repository owner (defaults to current repository)")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository name (defaults to current repository)")

	return cmd
}

func runTokensBootstrap(engine, owner, repo string) error {
	tokensBootstrapLog.Printf("Running tokens bootstrap: engine=%s, owner=%s, repo=%s", engine, owner, repo)
	var repoSlug string
	var err error

	// Determine target repository
	if owner != "" && repo != "" {
		repoSlug = fmt.Sprintf("%s/%s", owner, repo)
	} else if owner != "" || repo != "" {
		return fmt.Errorf("both --owner and --repo must be specified together")
	} else {
		repoSlug, err = GetCurrentRepoSlug()
		if err != nil {
			return fmt.Errorf("failed to detect current repository: %w", err)
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking recommended gh-aw token secrets in %s...", repoSlug)))

	// Use the unified GetRequiredSecretsForEngine function
	var requirements []SecretRequirement
	if engine != "" {
		requirements = GetRequiredSecretsForEngine(engine, true, true)
		tokensBootstrapLog.Printf("Checking tokens for specific engine: %s (%d tokens)", engine, len(requirements))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking tokens for engine: %s", engine)))
	} else {
		// When no engine specified, get all engine secrets plus system secrets
		requirements = GetRequiredSecretsForEngine("", true, true)
		// Add all engine-specific secrets as optional, deduplicating by secret name
		// (e.g., CopilotEngine and CopilotSDKEngine both use COPILOT_GITHUB_TOKEN)
		seenSecrets := make(map[string]bool)
		for _, req := range requirements {
			seenSecrets[req.Name] = true
		}
		for _, opt := range constants.EngineOptions {
			if seenSecrets[opt.SecretName] {
				continue // Skip duplicate secret names
			}
			seenSecrets[opt.SecretName] = true
			requirements = append(requirements, SecretRequirement{
				Name:               opt.SecretName,
				WhenNeeded:         opt.WhenNeeded,
				Description:        getEngineSecretDescription(&opt),
				Optional:           true, // All engines are optional when no specific engine is selected
				AlternativeEnvVars: opt.AlternativeSecrets,
				KeyURL:             opt.KeyURL,
				IsEngineSecret:     true,
				EngineName:         opt.Value,
			})
		}
		tokensBootstrapLog.Printf("Checking all recommended tokens: count=%d", len(requirements))
	}

	// Check existing secrets in repository
	existingSecrets, err := CheckExistingSecretsInRepo(repoSlug)
	if err != nil {
		return fmt.Errorf("unable to inspect repository secrets: %w", err)
	}

	// Check which secrets are missing
	var missing []SecretRequirement
	for _, req := range requirements {
		exists := existingSecrets[req.Name]
		if !exists {
			// Check alternatives
			for _, alt := range req.AlternativeEnvVars {
				if existingSecrets[alt] {
					exists = true
					break
				}
			}
		}
		if !exists {
			missing = append(missing, req)
		}
	}

	if len(missing) == 0 {
		tokensBootstrapLog.Print("All required tokens present")
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All recommended gh-aw token secrets are present in this repository."))
		return nil
	}

	tokensBootstrapLog.Printf("Found missing tokens: count=%d", len(missing))

	// Display missing secrets using the unified helper
	DisplayMissingSecrets(missing, repoSlug, existingSecrets)

	return nil
}
