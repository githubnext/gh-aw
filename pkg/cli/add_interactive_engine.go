package cli

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/styles"
	"github.com/github/gh-aw/pkg/workflow"
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
		return c.configureEngineAPISecret(c.EngineOverride)
	}

	// Inform user if workflow specifies an engine
	if workflowSpecifiedEngine != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflow specifies engine: "+workflowSpecifiedEngine))
	}

	// Build engine options with notes about existing secrets and workflow specification.
	// The list of engines is derived from the catalog to ensure all registered engines appear.
	catalog := workflow.NewEngineCatalog(workflow.NewEngineRegistry())
	engineOptions := sliceutil.Map(catalog.All(), func(def *workflow.EngineDefinition) huh.Option[string] {
		opt := constants.GetEngineOption(def.ID)
		label := fmt.Sprintf("%s - %s", def.DisplayName, def.Description)
		// Add markers for secret availability and workflow specification.
		// opt may be nil for catalog engines not yet represented in EngineOptions;
		// in that case we conservatively show '[no secret]'.
		if opt != nil && c.existingSecrets[opt.SecretName] {
			label += " [secret exists]"
		} else {
			label += " [no secret]"
		}
		if def.ID == workflowSpecifiedEngine {
			label += " [specified in workflow]"
		}
		return huh.NewOption(label, def.ID)
	})

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
	).WithTheme(styles.HuhTheme()).WithAccessible(console.IsAccessibleMode())

	if err := form.RunWithContext(c.Ctx); err != nil {
		return fmt.Errorf("failed to select coding agent: %w", err)
	}

	c.EngineOverride = selectedEngine
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Selected engine: "+selectedEngine))

	return c.configureEngineAPISecret(selectedEngine)
}

// configureEngineAPISecret collects the API key for the selected engine using the unified engine secrets functions
func (c *AddInteractiveConfig) configureEngineAPISecret(engine string) error {
	addInteractiveLog.Printf("Collecting API key for engine: %s", engine)

	// If --skip-secret flag is set, skip secrets configuration entirely.
	if c.SkipSecret {
		opt := constants.GetEngineOption(engine)
		if opt != nil {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping %s secret setup (--skip-secret flag set).", opt.SecretName)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping secret setup (--skip-secret flag set)."))
		}
		return nil
	}

	// If user doesn't have write access, skip secrets configuration.
	// Users without write access cannot configure repository secrets.
	if !c.hasWriteAccess {
		opt := constants.GetEngineOption(engine)
		if opt != nil {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping %s secret setup — write access is required to configure repository secrets.", opt.SecretName)))
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Once you have write access or an admin configures the repository, set the secret with:")
			fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  gh aw secrets set %s --repo %s", opt.SecretName, c.RepoOverride)))
		}
		return nil
	}

	// Use the unified checkAndEnsureEngineSecrets function
	config := EngineSecretConfig{
		Ctx:                  c.Ctx,
		RepoSlug:             c.RepoOverride,
		Engine:               engine,
		Verbose:              c.Verbose,
		ExistingSecrets:      c.existingSecrets,
		IncludeSystemSecrets: false, // Don't include system secrets in add-wizard
		IncludeOptional:      false,
	}

	if err := checkAndEnsureEngineSecretsForEngine(config); err != nil {
		return err
	}

	// Update existingSecrets to reflect that the secret was uploaded
	// This prevents duplicate secret uploads in createWorkflowPRAndConfigureSecret later
	opt := constants.GetEngineOption(engine)
	if opt != nil {
		c.existingSecrets[opt.SecretName] = true
		addInteractiveLog.Printf("Updated existingSecrets to include %s after upload", opt.SecretName)
	}

	return nil
}
