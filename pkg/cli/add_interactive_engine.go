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
	if err := c.checkExistingSecrets(); err != nil {
		return err
	}

	workflowSpecifiedEngine := ""
	if c.resolvedWorkflows != nil && len(c.resolvedWorkflows.Workflows) > 0 {
		for _, wf := range c.resolvedWorkflows.Workflows {
			if wf.Engine != "" {
				workflowSpecifiedEngine = wf.Engine
				addInteractiveLog.Printf("Workflow specifies engine in frontmatter: %s", wf.Engine)
				break
			}
		}
	}

	defaultEngine := c.determineDefaultEngine(workflowSpecifiedEngine)
	if c.EngineOverride != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using coding agent: "+c.EngineOverride))
		return c.configureEngineAPISecret(c.EngineOverride)
	}
	if workflowSpecifiedEngine != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflow specifies engine: "+workflowSpecifiedEngine))
	}

	engineOptions := c.buildEngineOptions(workflowSpecifiedEngine)
	var selectedEngine string
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
	).WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())
	if err := form.RunWithContext(c.Ctx); err != nil {
		return fmt.Errorf("failed to select coding agent: %w", err)
	}

	c.EngineOverride = selectedEngine
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Selected engine: "+selectedEngine))
	return c.configureEngineAPISecret(selectedEngine)
}

func (c *AddInteractiveConfig) determineDefaultEngine(workflowSpecifiedEngine string) string {
	defaultEngine := string(constants.DefaultEngine)
	if c.EngineOverride != "" {
		return c.EngineOverride
	}

	for _, opt := range constants.EngineOptions {
		if c.existingSecrets[opt.SecretName] {
			defaultEngine = opt.Value
			addInteractiveLog.Printf("Found existing secret %s, recommending engine: %s", opt.SecretName, opt.Value)
			break
		}
	}
	if defaultEngine == string(constants.DefaultEngine) && workflowSpecifiedEngine != "" {
		defaultEngine = workflowSpecifiedEngine
	}
	if defaultEngine != string(constants.DefaultEngine) || workflowSpecifiedEngine != "" {
		return defaultEngine
	}

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
	return defaultEngine
}

func (c *AddInteractiveConfig) buildEngineOptions(workflowSpecifiedEngine string) []huh.Option[string] {
	catalog := workflow.NewEngineCatalog(workflow.NewEngineRegistry())
	return sliceutil.Map(catalog.All(), func(def *workflow.EngineDefinition) huh.Option[string] {
		opt := constants.GetEngineOption(def.ID)
		label := fmt.Sprintf("%s - %s", def.DisplayName, def.Description)
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
