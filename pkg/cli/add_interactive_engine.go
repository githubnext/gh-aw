package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
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

// collectAPIKey collects the API key for the selected engine using the unified engine secrets functions
func (c *AddInteractiveConfig) collectAPIKey(engine string) error {
	addInteractiveLog.Printf("Collecting API key for engine: %s", engine)

	// Use the unified CheckAndCollectEngineSecrets function
	config := EngineSecretConfig{
		RepoSlug:             c.RepoOverride,
		Engine:               engine,
		Verbose:              c.Verbose,
		ExistingSecrets:      c.existingSecrets,
		IncludeSystemSecrets: false, // Don't include system secrets in add-wizard
		IncludeOptional:      false,
	}

	return CheckAndCollectEngineSecrets(config)
}
