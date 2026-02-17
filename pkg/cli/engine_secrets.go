package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var engineSecretsLog = logger.New("cli:engine_secrets")

// SecretRequirement represents a unified secret requirement for agentic workflows.
// This type unifies the legacy tokenSpec and EngineOption secret information.
type SecretRequirement struct {
	Name               string   // The secret name (e.g., "COPILOT_GITHUB_TOKEN")
	WhenNeeded         string   // Human-readable description of when this secret is needed
	Description        string   // Detailed description of the secret's purpose and required permissions
	Optional           bool     // Whether this secret is optional
	AlternativeEnvVars []string // Alternative environment variable names to check
	KeyURL             string   // URL where users can obtain their API key
	IsEngineSecret     bool     // True if this is an engine-specific secret (vs system-level)
	EngineName         string   // The engine this secret is for (if IsEngineSecret is true)
}

// EngineSecretConfig contains configuration for engine secret collection operations
type EngineSecretConfig struct {
	// RepoSlug is the repository slug to check for existing secrets (optional)
	RepoSlug string
	// Engine is the engine type to collect secrets for (e.g., "copilot", "claude", "codex")
	Engine string
	// Verbose enables verbose output
	Verbose bool
	// ExistingSecrets is a map of secret names that already exist in the repository
	ExistingSecrets map[string]bool
	// IncludeSystemSecrets includes system-level secrets like GH_AW_GITHUB_TOKEN
	IncludeSystemSecrets bool
	// IncludeOptional includes optional secrets in the requirements list
	IncludeOptional bool
}

// GetRequiredSecretsForEngine returns all secrets needed for a specific engine.
// This combines engine-specific secrets with optional system-level secrets.
func GetRequiredSecretsForEngine(engine string, includeSystemSecrets bool, includeOptional bool) []SecretRequirement {
	engineSecretsLog.Printf("Getting required secrets for engine: %s (system=%v, optional=%v)", engine, includeSystemSecrets, includeOptional)

	var requirements []SecretRequirement

	// Add system-level secrets first if requested
	if includeSystemSecrets {
		for _, sys := range constants.SystemSecrets {
			if sys.Optional && !includeOptional {
				continue
			}
			requirements = append(requirements, SecretRequirement{
				Name:           sys.Name,
				WhenNeeded:     sys.WhenNeeded,
				Description:    sys.Description,
				Optional:       sys.Optional,
				IsEngineSecret: false,
			})
		}
	}

	// Add engine-specific secret
	opt := constants.GetEngineOption(engine)
	if opt != nil {
		requirements = append(requirements, SecretRequirement{
			Name:               opt.SecretName,
			WhenNeeded:         opt.WhenNeeded,
			Description:        getEngineSecretDescription(opt),
			Optional:           false,
			AlternativeEnvVars: opt.AlternativeSecrets,
			KeyURL:             opt.KeyURL,
			IsEngineSecret:     true,
			EngineName:         engine,
		})
	}

	engineSecretsLog.Printf("Returning %d secret requirements for engine %s", len(requirements), engine)
	return requirements
}

// getEngineSecretDescription returns a detailed description for an engine secret
func getEngineSecretDescription(opt *constants.EngineOption) string {
	switch opt.Value {
	case string(constants.CopilotEngine), string(constants.CopilotSDKEngine):
		return "Fine-grained PAT with Copilot Requests permission and repo access where Copilot workflows run."
	case string(constants.ClaudeEngine):
		return "API key from Anthropic Console for Claude API access."
	case string(constants.CodexEngine):
		return "API key from OpenAI for Codex/GPT API access."
	default:
		return fmt.Sprintf("API key for %s engine.", opt.Label)
	}
}

// CheckAndCollectEngineSecrets is the unified entry point for checking and collecting engine secrets.
// It checks existing secrets in the repository and environment, and prompts for missing ones.
func CheckAndCollectEngineSecrets(config EngineSecretConfig) error {
	engineSecretsLog.Printf("Checking and collecting secrets for engine: %s in repo: %s", config.Engine, config.RepoSlug)

	// Get required secrets for the engine
	requirements := GetRequiredSecretsForEngine(config.Engine, config.IncludeSystemSecrets, config.IncludeOptional)

	// Check each requirement
	for _, req := range requirements {
		if req.Optional {
			// For optional secrets, just check and report
			if err := checkOptionalSecret(req, config); err != nil && config.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Optional secret %s: %v", req.Name, err)))
			}
			continue
		}

		// For required secrets, ensure they're available
		if err := ensureSecretAvailable(req, config); err != nil {
			return fmt.Errorf("failed to ensure secret %s: %w", req.Name, err)
		}
	}

	return nil
}

// ensureSecretAvailable ensures that a required secret is available.
// It checks the repository, environment, and prompts the user if needed.
func ensureSecretAvailable(req SecretRequirement, config EngineSecretConfig) error {
	engineSecretsLog.Printf("Ensuring secret available: %s", req.Name)

	// Check if secret already exists in the repository
	if config.ExistingSecrets[req.Name] {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Using existing %s secret in repository", req.Name)))
		return nil
	}

	// Check alternative secret names in repository
	for _, alt := range req.AlternativeEnvVars {
		if config.ExistingSecrets[alt] {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Using existing %s secret in repository (alternative for %s)", alt, req.Name)))
			return nil
		}
	}

	// Check environment variable
	envValue := os.Getenv(req.Name)
	if envValue == "" {
		// Check alternative environment variables
		for _, alt := range req.AlternativeEnvVars {
			envValue = os.Getenv(alt)
			if envValue != "" {
				engineSecretsLog.Printf("Found secret in alternative env var: %s", alt)
				break
			}
		}
	}

	if envValue != "" {
		// Validate if it's a Copilot token
		if req.IsEngineSecret && (req.EngineName == string(constants.CopilotEngine) || req.EngineName == string(constants.CopilotSDKEngine)) {
			if err := stringutil.ValidateCopilotPAT(envValue); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%s in environment is not a valid fine-grained PAT: %s", req.Name, stringutil.GetPATTypeDescription(envValue))))
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				// Continue to prompt for a new token
			} else {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found valid %s in environment", req.Name)))
				// Upload to repository if we have a repo slug
				if config.RepoSlug != "" {
					return uploadSecretToRepo(req.Name, envValue, config.RepoSlug, config.Verbose)
				}
				return nil
			}
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %s in environment", req.Name)))
			// Upload to repository if we have a repo slug
			if config.RepoSlug != "" {
				return uploadSecretToRepo(req.Name, envValue, config.RepoSlug, config.Verbose)
			}
			return nil
		}
	}

	// Secret not found, prompt user for it
	return promptForSecret(req, config)
}

// promptForSecret prompts the user to provide a secret value
func promptForSecret(req SecretRequirement, config EngineSecretConfig) error {
	engineSecretsLog.Printf("Prompting for secret: %s", req.Name)

	// Copilot requires special handling with PAT creation instructions
	if req.IsEngineSecret && (req.EngineName == string(constants.CopilotEngine) || req.EngineName == string(constants.CopilotSDKEngine)) {
		return promptForCopilotPATUnified(req, config)
	}

	// System secrets (GH_AW_*) require PAT-specific prompting, not API key wording
	if !req.IsEngineSecret {
		return promptForSystemTokenUnified(req, config)
	}

	return promptForGenericAPIKeyUnified(req, config)
}

// promptForCopilotPATUnified prompts the user for a Copilot PAT with detailed instructions
func promptForCopilotPATUnified(req SecretRequirement, config EngineSecretConfig) error {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "GitHub Copilot requires a fine-grained Personal Access Token (PAT) with Copilot permissions.")
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Classic PATs (ghp_...) are not supported. You must use a fine-grained PAT (github_pat_...)."))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Please create a token at:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s", req.KeyURL)))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Configure the token with:")
	fmt.Fprintln(os.Stderr, "  • Token name: Agentic Workflows Copilot")
	fmt.Fprintln(os.Stderr, "  • Expiration: 90 days (recommended for testing)")
	fmt.Fprintln(os.Stderr, "  • Resource owner: Your personal account")
	fmt.Fprintln(os.Stderr, "  • Repository access: \"Public repositories\" (you must use this setting even for private repos)")
	fmt.Fprintln(os.Stderr, "  • Account permissions → Copilot Requests: Read-only")
	fmt.Fprintln(os.Stderr, "")

	var token string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("After creating, please paste your fine-grained Copilot PAT:").
				Description("Must start with 'github_pat_'. Classic PATs (ghp_...) are not supported.").
				EchoMode(huh.EchoModePassword).
				Value(&token).
				Validate(func(s string) error {
					if len(s) < 10 {
						return fmt.Errorf("token appears to be too short")
					}
					return stringutil.ValidateCopilotPAT(s)
				}),
		),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to get Copilot token: %w", err)
	}

	// Store in environment for later use
	_ = os.Setenv(req.Name, token)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Valid fine-grained Copilot token received"))

	// Upload to repository if we have a repo slug
	if config.RepoSlug != "" {
		return uploadSecretToRepo(req.Name, token, config.RepoSlug, config.Verbose)
	}

	return nil
}

// promptForSystemTokenUnified prompts the user for a system-level GitHub token (PAT)
// This uses PAT-specific wording instead of "API key" since system secrets are GitHub tokens
func promptForSystemTokenUnified(req SecretRequirement, config EngineSecretConfig) error {
	engineSecretsLog.Printf("Prompting for system token: %s", req.Name)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "%s requires a GitHub Personal Access Token (PAT).\n", req.Name)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("When needed: %s", req.WhenNeeded)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Recommended scopes: %s", req.Description)))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Create a token at:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage("  https://github.com/settings/personal-access-tokens/new"))
	fmt.Fprintln(os.Stderr, "")

	var token string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Paste your %s token:", req.Name)).
				Description("The token will be stored securely as a repository secret").
				EchoMode(huh.EchoModePassword).
				Value(&token).
				Validate(func(s string) error {
					if len(s) < 10 {
						return fmt.Errorf("token appears to be too short")
					}
					return nil
				}),
		),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to get %s token: %w", req.Name, err)
	}

	// Store in environment for later use
	_ = os.Setenv(req.Name, token)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("%s token received", req.Name)))

	// Upload to repository if we have a repo slug
	if config.RepoSlug != "" {
		return uploadSecretToRepo(req.Name, token, config.RepoSlug, config.Verbose)
	}

	return nil
}

// promptForGenericAPIKeyUnified prompts the user for a generic API key
func promptForGenericAPIKeyUnified(req SecretRequirement, config EngineSecretConfig) error {
	engineSecretsLog.Printf("Prompting for API key: %s", req.Name)

	// Get engine option for label
	opt := constants.GetEngineOption(req.EngineName)
	label := req.Name
	if opt != nil {
		label = opt.Label
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "%s requires an API key.\n", label)
	fmt.Fprintln(os.Stderr, "")
	if req.KeyURL != "" {
		fmt.Fprintln(os.Stderr, "Get your API key from:")
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s", req.KeyURL)))
		fmt.Fprintln(os.Stderr, "")
	}

	var apiKey string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Paste your %s API key:", label)).
				Description("The key will be stored securely as a repository secret").
				EchoMode(huh.EchoModePassword).
				Value(&apiKey).
				Validate(func(s string) error {
					if len(s) < 10 {
						return fmt.Errorf("API key appears to be too short")
					}
					return nil
				}),
		),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to get %s API key: %w", label, err)
	}

	// Store in environment for later use
	_ = os.Setenv(req.Name, apiKey)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("%s API key received", label)))

	// Upload to repository if we have a repo slug
	if config.RepoSlug != "" {
		return uploadSecretToRepo(req.Name, apiKey, config.RepoSlug, config.Verbose)
	}

	return nil
}

// checkOptionalSecret checks if an optional secret is available (without prompting)
func checkOptionalSecret(req SecretRequirement, config EngineSecretConfig) error {
	// Check repository
	if config.ExistingSecrets[req.Name] {
		if config.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Optional secret %s exists in repository", req.Name)))
		}
		return nil
	}

	// Check environment
	if os.Getenv(req.Name) != "" {
		if config.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Optional secret %s found in environment", req.Name)))
		}
		return nil
	}

	return fmt.Errorf("not configured")
}

// uploadSecretToRepo uploads a secret to the repository if it doesn't already exist
func uploadSecretToRepo(secretName, secretValue, repoSlug string, verbose bool) error {
	engineSecretsLog.Printf("Uploading secret %s to %s", secretName, repoSlug)

	// Check if secret already exists
	output, err := workflow.RunGHCombined("Checking secrets...", "secret", "list", "--repo", repoSlug)
	if err == nil && stringContainsSecretName(string(output), secretName) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret %s already exists, skipping upload", secretName)))
		}
		return nil
	}

	// Upload the secret
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Uploading %s secret to repository", secretName)))
	}

	output, err = workflow.RunGHCombined("Setting secret...", "secret", "set", secretName, "--repo", repoSlug, "--body", secretValue)
	if err != nil {
		return fmt.Errorf("failed to set %s secret: %w (output: %s)", secretName, err, string(output))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Uploaded %s secret to repository", secretName)))
	return nil
}

// stringContainsSecretName checks if the gh secret list output contains a secret name
func stringContainsSecretName(output, secretName string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) >= len(secretName) {
			if line[:len(secretName)] == secretName && (len(line) == len(secretName) || line[len(secretName)] == '\t' || line[len(secretName)] == ' ') {
				return true
			}
		}
	}
	return false
}

// CheckExistingSecretsInRepo checks which secrets exist in the repository
func CheckExistingSecretsInRepo(repoSlug string) (map[string]bool, error) {
	engineSecretsLog.Printf("Checking existing secrets for repo: %s", repoSlug)

	existingSecrets := make(map[string]bool)

	// List secrets from repository
	output, err := workflow.RunGHCombined("Checking secrets...", "secret", "list", "--repo", repoSlug)
	if err != nil {
		engineSecretsLog.Printf("Could not list secrets for %s: %v", repoSlug, err)
		return existingSecrets, err
	}

	// Check for all known engine secrets
	secretNames := constants.GetAllEngineSecretNames()

	outputStr := string(output)
	for _, name := range secretNames {
		if stringContainsSecretName(outputStr, name) {
			existingSecrets[name] = true
		}
	}

	return existingSecrets, nil
}

// DisplayMissingSecrets shows information about missing secrets with setup instructions
func DisplayMissingSecrets(requirements []SecretRequirement, repoSlug string, existingSecrets map[string]bool) {
	var requiredMissing, optionalMissing []SecretRequirement

	for _, req := range requirements {
		// Check if secret exists
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
			if req.Optional {
				optionalMissing = append(optionalMissing, req)
			} else {
				requiredMissing = append(requiredMissing, req)
			}
		}
	}

	// Extract owner and repo from slug for command examples
	parts := splitRepoSlug(repoSlug)
	cmdOwner := parts[0]
	cmdRepo := parts[1]

	if len(requiredMissing) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Required secrets are missing:"))
		for _, req := range requiredMissing {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret: %s", req.Name)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("When needed: %s", req.WhenNeeded)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Recommended scopes: %s", req.Description)))
			fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("gh aw secrets set %s --owner %s --repo %s", req.Name, cmdOwner, cmdRepo)))
		}
	}

	if len(optionalMissing) > 0 {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Optional secrets are missing:"))
		for _, req := range optionalMissing {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret: %s (optional)", req.Name)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("When needed: %s", req.WhenNeeded)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Recommended scopes: %s", req.Description)))
			fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("gh aw secrets set %s --owner %s --repo %s", req.Name, cmdOwner, cmdRepo)))
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("For detailed token behavior and precedence, see the GitHub Tokens reference in the documentation."))
}

// splitRepoSlug splits "owner/repo" into [owner, repo]
// Uses repoutil.SplitRepoSlug internally but provides backward-compatible array return
func splitRepoSlug(slug string) [2]string {
	owner, repo, err := repoutil.SplitRepoSlug(slug)
	if err != nil {
		// Fallback behavior for invalid format
		return [2]string{slug, ""}
	}
	return [2]string{owner, repo}
}
