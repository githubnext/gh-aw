package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var secretCollectionLog = logger.New("cli:secret_collection")

// SecretCollectionConfig contains configuration for secret collection
type SecretCollectionConfig struct {
	// RepoSlug is the repository slug to check for existing secrets
	RepoSlug string
	// Engine is the engine type to collect secrets for (e.g., "copilot", "claude", "codex")
	Engine string
	// Verbose enables verbose output
	Verbose bool
	// ExistingSecrets is a map of secret names that already exist in the repository
	ExistingSecrets map[string]bool
}

// EnsureEngineSecret ensures that the required secret for the specified engine is available.
// If the secret exists in the repository or environment, it returns nil.
// If the secret is missing, it prompts the user interactively to provide it.
// For Copilot, it validates the token is a fine-grained PAT.
//
// Returns an error if the secret cannot be collected or is invalid.
func EnsureEngineSecret(config SecretCollectionConfig) error {
	secretCollectionLog.Printf("Ensuring engine secret for %s engine in repo %s", config.Engine, config.RepoSlug)

	// Copilot has special PAT validation requirements
	if config.Engine == "copilot" || config.Engine == "" || config.Engine == "copilot-sdk" {
		return ensureCopilotSecret(config)
	}

	// For any other engine, check if we have an EngineOption for it
	if opt := constants.GetEngineOption(config.Engine); opt != nil {
		return ensureEngineSecretGeneric(config, config.Engine)
	}

	// Unknown engine, default to copilot
	return ensureCopilotSecret(config)
}

// CheckExistingSecrets checks which secrets exist in the repository
func CheckExistingSecrets(repoSlug string) (map[string]bool, error) {
	secretCollectionLog.Printf("Checking existing secrets for repo: %s", repoSlug)

	existingSecrets := make(map[string]bool)

	// List secrets from repository
	output, err := workflow.RunGHCombined("Checking secrets...", "secret", "list", "--repo", repoSlug)
	if err != nil {
		secretCollectionLog.Printf("Could not list secrets for %s: %v", repoSlug, err)
		return existingSecrets, err
	}

	// Check for all known engine secrets (primary, alternative, and system-level)
	secretNames := constants.GetAllEngineSecretNames()

	outputStr := string(output)
	for _, name := range secretNames {
		if stringContainsSecret(outputStr, name) {
			existingSecrets[name] = true
		}
	}

	return existingSecrets, nil
}

// stringContainsSecret checks if the output contains a secret name
func stringContainsSecret(output, secretName string) bool {
	// The secret list output format is typically "SECRET_NAME\tUpdated ...\n"
	// We need to check for the exact secret name at the start of a line
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) >= len(secretName) {
			// Check if line starts with secret name followed by tab or space
			if line[:len(secretName)] == secretName && (len(line) == len(secretName) || line[len(secretName)] == '\t' || line[len(secretName)] == ' ') {
				return true
			}
		}
	}
	return false
}

// ensureCopilotSecret ensures the COPILOT_GITHUB_TOKEN secret is available
func ensureCopilotSecret(config SecretCollectionConfig) error {
	secretName := "COPILOT_GITHUB_TOKEN"
	secretCollectionLog.Printf("Ensuring Copilot secret: %s", secretName)

	// Check if secret already exists in the repository
	if config.ExistingSecrets[secretName] {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Using existing COPILOT_GITHUB_TOKEN secret in repository"))
		return nil
	}

	// Check if COPILOT_GITHUB_TOKEN is already in environment
	existingToken := os.Getenv(secretName)
	if existingToken != "" {
		// Validate the existing token is a fine-grained PAT
		if err := stringutil.ValidateCopilotPAT(existingToken); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("COPILOT_GITHUB_TOKEN in environment is not a fine-grained PAT: %s", stringutil.GetPATTypeDescription(existingToken))))
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			// Continue to prompt for a new token
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found valid fine-grained COPILOT_GITHUB_TOKEN in environment"))
			// Upload to repository if we have a repo slug
			if config.RepoSlug != "" {
				return uploadSecretIfNeeded(secretName, existingToken, config.RepoSlug, config.Verbose)
			}
			return nil
		}
	}

	// Prompt user for the token
	token, err := promptForCopilotPAT()
	if err != nil {
		return err
	}

	// Store in environment for later use
	_ = os.Setenv(secretName, token)

	// Upload to repository if we have a repo slug
	if config.RepoSlug != "" {
		return uploadSecretIfNeeded(secretName, token, config.RepoSlug, config.Verbose)
	}

	return nil
}

// promptForCopilotPAT prompts the user for a Copilot PAT
func promptForCopilotPAT() (string, error) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "GitHub Copilot requires a fine-grained Personal Access Token (PAT) with Copilot permissions.")
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Classic PATs (ghp_...) are not supported. You must use a fine-grained PAT (github_pat_...)."))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Please create a token at:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage("  https://github.com/settings/personal-access-tokens/new"))
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
					// Validate it's a fine-grained PAT
					return stringutil.ValidateCopilotPAT(s)
				}),
		),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("failed to get Copilot token: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Valid fine-grained Copilot token received"))
	return token, nil
}

// ensureEngineSecretGeneric ensures the required secret for a given engine is available.
// It checks existing repository secrets, environment variables, and prompts the user if needed.
// Uses the EngineOption data to determine primary and alternative secret names.
func ensureEngineSecretGeneric(config SecretCollectionConfig, engineName string) error {
	opt := constants.GetEngineOption(engineName)
	if opt == nil {
		return fmt.Errorf("unknown engine: %s", engineName)
	}

	secretCollectionLog.Printf("Ensuring %s secret", opt.Label)

	// Collect all possible secret names (primary + alternatives)
	allSecretNames := []string{opt.SecretName}
	allSecretNames = append(allSecretNames, opt.AlternativeSecrets...)

	// Check existing secrets in repository
	for _, secretName := range allSecretNames {
		if config.ExistingSecrets[secretName] {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Using existing %s secret in repository", secretName)))
			return nil
		}
	}

	// Check environment variables
	for _, secretName := range allSecretNames {
		envValue := os.Getenv(secretName)
		if envValue != "" {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %s in environment", secretName)))
			if config.RepoSlug != "" {
				return uploadSecretIfNeeded(secretName, envValue, config.RepoSlug, config.Verbose)
			}
			return nil
		}
	}

	// Prompt for API key
	apiKey, err := promptForGenericAPIKey(opt)
	if err != nil {
		return err
	}

	// Store in environment using primary secret name
	_ = os.Setenv(opt.SecretName, apiKey)

	// Upload to repository
	if config.RepoSlug != "" {
		return uploadSecretIfNeeded(opt.SecretName, apiKey, config.RepoSlug, config.Verbose)
	}

	return nil
}

// promptForGenericAPIKey prompts the user for an API key for a specific engine
func promptForGenericAPIKey(opt *constants.EngineOption) (string, error) {
	secretCollectionLog.Printf("Prompting for API key: %s", opt.Label)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "%s requires an API key.\n", opt.Label)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Get your API key from:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s", opt.KeyURL)))
	fmt.Fprintln(os.Stderr, "")

	var apiKey string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Paste your %s API key:", opt.Label)).
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
		return "", fmt.Errorf("failed to get %s API key: %w", opt.Label, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("%s API key received", opt.Label)))
	return apiKey, nil
}

// uploadSecretIfNeeded uploads a secret to the repository if it doesn't already exist
func uploadSecretIfNeeded(secretName, secretValue, repoSlug string, verbose bool) error {
	secretCollectionLog.Printf("Uploading secret %s to %s", secretName, repoSlug)

	// Check if secret already exists
	output, err := workflow.RunGHCombined("Checking secrets...", "secret", "list", "--repo", repoSlug)
	if err == nil && stringContainsSecret(string(output), secretName) {
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
