package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
)

// selectAIEngineAndKey prompts the user to select an AI engine and provide API key
func (c *AddInteractiveConfig) selectAIEngineAndKey() error {
	addInteractiveLog.Print("Starting coding agent selection")

	// First, check which secrets already exist in the repository
	if err := c.checkExistingSecrets(); err != nil {
		return err
	}

	// Determine default engine based on existing secrets, workflow preference, then environment
	// Priority order: flag override > existing secrets > workflow frontmatter > environment > default
	defaultEngine := string(constants.CopilotEngine)
	workflowSpecifiedEngine := ""

	// Check if workflow specifies a preferred engine in frontmatter
	if c.resolvedWorkflows != nil && len(c.resolvedWorkflows.Workflows) > 0 {
		for _, wf := range c.resolvedWorkflows.Workflows {
			if wf.Engine != "" {
				workflowSpecifiedEngine = wf.Engine
				addInteractiveLog.Printf("Workflow specifies engine in frontmatter: %s", wf.Engine)
				break
			}
		}
	}

	// If engine is explicitly overridden via flag, use that
	if c.EngineOverride != "" {
		defaultEngine = c.EngineOverride
	} else {
		// Priority 1: Check existing repository secrets using EngineOptions
		// This takes precedence over workflow preference since users should use what's already available
		for _, opt := range constants.EngineOptions {
			if c.existingSecrets[opt.SecretName] {
				defaultEngine = opt.Value
				addInteractiveLog.Printf("Found existing secret %s, recommending engine: %s", opt.SecretName, opt.Value)
				break
			}
		}

		// Priority 2: If no existing secret found, use workflow frontmatter preference
		if defaultEngine == string(constants.CopilotEngine) && workflowSpecifiedEngine != "" {
			defaultEngine = workflowSpecifiedEngine
		}

		// Priority 3: Check environment variables if no existing secret or workflow preference found
		if defaultEngine == string(constants.CopilotEngine) && workflowSpecifiedEngine == "" {
			for _, opt := range constants.EngineOptions {
				envVar := opt.SecretName
				if opt.EnvVarName != "" {
					envVar = opt.EnvVarName
				}
				if os.Getenv(envVar) != "" {
					defaultEngine = opt.Value
					addInteractiveLog.Printf("Found env var %s, recommending engine: %s", envVar, opt.Value)
					break
				}
			}
		}
	}

	// If engine is already overridden, skip selection
	if c.EngineOverride != "" {
		fmt.Fprintf(os.Stderr, "Using coding agent: %s\n", c.EngineOverride)
		return c.collectAPIKey(c.EngineOverride)
	}

	// Inform user if workflow specifies an engine
	if workflowSpecifiedEngine != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow specifies engine: %s", workflowSpecifiedEngine)))
	}

	// Build engine options with notes about existing secrets and workflow specification
	var engineOptions []huh.Option[string]
	for _, opt := range constants.EngineOptions {
		label := fmt.Sprintf("%s - %s", opt.Label, opt.Description)
		// Add markers for secret availability and workflow specification
		if c.existingSecrets[opt.SecretName] {
			label += " [secret exists]"
		} else {
			label += " [no secret]"
		}
		if opt.Value == workflowSpecifiedEngine {
			label += " [specified in workflow]"
		}
		engineOptions = append(engineOptions, huh.NewOption(label, opt.Value))
	}

	var selectedEngine string

	// Set the default selection by moving it to front
	for i, opt := range engineOptions {
		if opt.Value == defaultEngine {
			if i > 0 {
				engineOptions[0], engineOptions[i] = engineOptions[i], engineOptions[0]
			}
			break
		}
	}

	fmt.Fprintln(os.Stderr, "")
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which coding agent would you like to use?").
				Description("This determines which coding agent processes your workflows").
				Options(engineOptions...).
				Value(&selectedEngine),
		),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to select coding agent: %w", err)
	}

	c.EngineOverride = selectedEngine
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Selected engine: %s", selectedEngine)))

	return c.collectAPIKey(selectedEngine)
}

// collectAPIKey collects the API key for the selected engine
func (c *AddInteractiveConfig) collectAPIKey(engine string) error {
	addInteractiveLog.Printf("Collecting API key for engine: %s", engine)

	// Copilot requires special handling with PAT creation instructions
	if engine == "copilot" {
		return c.collectCopilotPAT()
	}

	// All other engines use the generic API key collection
	opt := constants.GetEngineOption(engine)
	if opt == nil {
		return fmt.Errorf("unknown engine: %s", engine)
	}

	return c.collectGenericAPIKey(opt)
}

// collectCopilotPAT walks the user through creating a Copilot PAT
func (c *AddInteractiveConfig) collectCopilotPAT() error {
	addInteractiveLog.Print("Collecting Copilot PAT")

	// Check if secret already exists in the repository
	if c.existingSecrets["COPILOT_GITHUB_TOKEN"] {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Using existing COPILOT_GITHUB_TOKEN secret in repository"))
		return nil
	}

	// Check if COPILOT_GITHUB_TOKEN is already in environment
	existingToken := os.Getenv("COPILOT_GITHUB_TOKEN")
	if existingToken != "" {
		// Validate the existing token is a fine-grained PAT
		if err := stringutil.ValidateCopilotPAT(existingToken); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("COPILOT_GITHUB_TOKEN in environment is not a fine-grained PAT: %s", stringutil.GetPATTypeDescription(existingToken))))
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			// Continue to prompt for a new token
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found valid fine-grained COPILOT_GITHUB_TOKEN in environment"))
			return nil
		}
	}

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
		return fmt.Errorf("failed to get Copilot token: %w", err)
	}

	// Store in environment for later use
	_ = os.Setenv("COPILOT_GITHUB_TOKEN", token)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Valid fine-grained Copilot token received"))

	return nil
}

// collectGenericAPIKey collects an API key for engines that use a simple key-based authentication
func (c *AddInteractiveConfig) collectGenericAPIKey(opt *constants.EngineOption) error {
	addInteractiveLog.Printf("Collecting API key for %s", opt.Label)

	// Check if secret already exists in the repository
	if c.existingSecrets[opt.SecretName] {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Using existing %s secret in repository", opt.SecretName)))
		return nil
	}

	// Check if key is already in environment
	envVar := opt.SecretName
	if opt.EnvVarName != "" {
		envVar = opt.EnvVarName
	}
	existingKey := os.Getenv(envVar)
	if existingKey != "" {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %s in environment", envVar)))
		return nil
	}

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
		return fmt.Errorf("failed to get %s API key: %w", opt.Label, err)
	}

	// Store in environment for later use
	_ = os.Setenv(opt.SecretName, apiKey)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("%s API key received", opt.Label)))

	return nil
}
