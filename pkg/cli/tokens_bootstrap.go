package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var tokensBootstrapLog = logger.New("cli:tokens_bootstrap")

// newSecretsBootstrapSubcommand creates the `secrets bootstrap` subcommand
func newSecretsBootstrapSubcommand() *cobra.Command {
	var engineFlag string
	var ownerFlag string
	var repoFlag string
	var nonInteractiveFlag bool

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Analyze workflows and set up required secrets",
		Long: `Analyzes all workflows in the repository to determine which secrets
are required, checks which ones are already configured, and interactively
prompts for any missing required secrets.

This command:
- Discovers all workflow files in .github/workflows/
- Analyzes required secrets for each workflow's engine
- Checks which secrets already exist in the repository
- Interactively prompts for missing required secrets (unless --non-interactive)

Only required secrets are prompted for. Optional secrets are not shown.

For full details, including precedence rules, see the GitHub Tokens
reference in the documentation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTokensBootstrap(engineFlag, ownerFlag, repoFlag, nonInteractiveFlag)
		},
	}

	cmd.Flags().StringVarP(&engineFlag, "engine", "e", "", "Check tokens for specific engine only (copilot, claude, codex)")
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Repository owner (defaults to current repository)")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository name (defaults to current repository)")
	cmd.Flags().BoolVar(&nonInteractiveFlag, "non-interactive", false, "Check secrets without prompting (display-only mode)")

	return cmd
}

func runTokensBootstrap(engine, owner, repo string, nonInteractive bool) error {
	tokensBootstrapLog.Printf("Running tokens bootstrap: engine=%s, owner=%s, repo=%s, nonInteractive=%v", engine, owner, repo, nonInteractive)
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

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Analyzing workflows in %s...", repoSlug)))

	// Discover workflows in the repository
	requirements, err := collectRequiredSecretsFromWorkflows(engine)
	if err != nil {
		return fmt.Errorf("failed to analyze workflows: %w", err)
	}

	tokensBootstrapLog.Printf("Collected %d required secrets from workflows", len(requirements))

	// Check existing secrets in repository
	existingSecrets, err := CheckExistingSecretsInRepo(repoSlug)
	if err != nil {
		// If we can't check existing secrets (e.g., no gh auth), continue with empty map
		tokensBootstrapLog.Printf("Could not check existing secrets: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Unable to check existing repository secrets. Will assume all secrets need to be configured."))
		existingSecrets = make(map[string]bool)
	}

	// Filter to only required secrets (not optional)
	// Check which secrets are missing
	var missing []SecretRequirement
	for _, req := range requirements {
		// Skip optional secrets - we only care about required ones
		if req.Optional {
			continue
		}

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

	// Always display summary table of all required secrets with their status
	displaySecretsSummaryTable(requirements, existingSecrets)

	if len(missing) == 0 {
		tokensBootstrapLog.Print("All required secrets present")
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All required secrets are configured."))
		return nil
	}

	tokensBootstrapLog.Printf("Found %d missing required secrets", len(missing))

	// In non-interactive mode, just display what's missing
	if nonInteractive {
		displayMissingSecrets(missing, repoSlug, existingSecrets)
		return nil
	}

	// Interactive mode: prompt for missing secrets
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d missing required secret(s). You will be prompted to provide them.", len(missing))))
	fmt.Fprintln(os.Stderr, "")

	config := EngineSecretConfig{
		RepoSlug:             repoSlug,
		ExistingSecrets:      existingSecrets,
		IncludeSystemSecrets: true,
		IncludeOptional:      false,
	}

	// Prompt for each missing secret
	for _, req := range missing {
		if err := promptForSecret(req, config); err != nil {
			return fmt.Errorf("failed to collect secret %s: %w", req.Name, err)
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All required secrets have been configured."))

	return nil
}

// collectRequiredSecretsFromWorkflows discovers all workflows and collects their required secrets
func collectRequiredSecretsFromWorkflows(engineFilter string) ([]SecretRequirement, error) {
	tokensBootstrapLog.Printf("Discovering workflows (engine filter: %s)", engineFilter)

	// If engine is explicitly specified, we can bootstrap without workflows
	if engineFilter != "" {
		tokensBootstrapLog.Printf("Engine explicitly specified, bootstrapping for %s regardless of workflows", engineFilter)
		return getRequiredSecretsForEngines([]string{engineFilter}), nil
	}

	// Get engines from discovered workflows
	engines, err := getRequiredEnginesForWorkflows()
	if err != nil {
		return nil, err
	}

	return getRequiredSecretsForEngines(engines), nil
}

// getRequiredEnginesForWorkflows discovers workflow files and returns unique engines used
func getRequiredEnginesForWorkflows() ([]string, error) {
	tokensBootstrapLog.Print("Discovering workflow files to extract engines")

	// Discover workflow files
	workflowFiles, err := getMarkdownWorkflowFiles("")
	if err != nil {
		return nil, fmt.Errorf("failed to discover workflows: %w", err)
	}

	if len(workflowFiles) == 0 {
		return nil, fmt.Errorf("no workflow files found in .github/workflows/")
	}

	tokensBootstrapLog.Printf("Found %d workflow files", len(workflowFiles))

	return extractEnginesFromWorkflows(workflowFiles), nil
}
