package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
	defaultEngine := string(constants.DefaultEngine)
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
		if defaultEngine == string(constants.DefaultEngine) && workflowSpecifiedEngine != "" {
			defaultEngine = workflowSpecifiedEngine
		}

		// Priority 3: Check environment variables if no existing secret or workflow preference found
		if defaultEngine == string(constants.DefaultEngine) && workflowSpecifiedEngine == "" {
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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using coding agent: "+c.EngineOverride))
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
	).WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())

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
	// Note: for Copilot workflows, --skip-secret implies the PAT path; users who want
	// copilot-requests (org billing) should not pass --skip-secret.
	if c.SkipSecret {
		opt := constants.GetEngineOption(engine)
		if opt != nil {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping %s secret setup (--skip-secret flag set).", opt.SecretName)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping secret setup (--skip-secret flag set)."))
		}
		return nil
	}

	// For Copilot, ask the user whether to use copilot-requests (org billing) or a PAT.
	// Only prompt when an interactive context is available (wizard path); default to PAT otherwise.
	if engine == string(constants.CopilotEngine) && c.Ctx != nil {
		if err := c.selectCopilotAuthMethod(); err != nil {
			return err
		}
		if c.UseCopilotRequests {
			return nil
		}
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

// authMethodCopilotRequests is the wizard option value for Copilot org-billing authentication
// (permissions.copilot-requests: write). Extracted as a package-level constant so both the
// form definition and applyCopilotAuthMethodChoice reference the same sentinel.
const authMethodCopilotRequests = "copilot-requests"

// selectCopilotAuthMethod prompts the user to choose between copilot-requests (org billing)
// and a Personal Access Token for Copilot authentication.
// Sets c.UseCopilotRequests when org billing is chosen.
func (c *AddInteractiveConfig) selectCopilotAuthMethod() error {
	addInteractiveLog.Print("Prompting user for Copilot authentication method")

	const authMethodPAT = "pat"

	// Detect org Copilot CLI billing status before building the form.
	// c.RepoOverride is in "owner/repo" format; we need just the org login.
	// When no org login is available the result is inconclusive (same as a
	// non-200 response) so the user still sees the info note.
	copilotRequestsLabel := "Use copilot-requests (org's Copilot billing, no PAT)"

	var probe orgCopilotBillingProbeResult
	if orgLogin, _, found := strings.Cut(c.RepoOverride, "/"); found && orgLogin != "" {
		probe = probeCopilotBillingForOrg(c.Ctx, orgLogin)
	} else {
		probe = orgCopilotBillingProbeResult{
			InfoNote: copilotBillingInconclusiveNote,
		}
	}
	c.copilotCLIBillingStatus = probe.BillingStatus
	copilotRequestsLabel += probe.LabelSuffix
	if probe.InfoNote != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(probe.InfoNote))
	}

	fmt.Fprintln(os.Stderr, "")

	// Build select options.
	// When billing is confirmed enabled, copilot-requests is listed first (pre-selected).
	// When billing is disabled or inconclusive, PAT is listed first (default selection).
	// The copilot-requests option is always shown; when disabled a validation guard
	// prevents it from being submitted.
	patOpt := huh.NewOption("Use a Personal Access Token (PAT) as COPILOT_GITHUB_TOKEN", authMethodPAT)
	copilotRequestsOpt := huh.NewOption(copilotRequestsLabel, authMethodCopilotRequests)

	var options []huh.Option[string]
	switch probe.BillingStatus {
	case "enabled":
		// copilot-requests pre-selected
		options = []huh.Option[string]{copilotRequestsOpt.Selected(true), patOpt}
	default:
		// PAT is default (first) for disabled or inconclusive
		options = []huh.Option[string]{patOpt, copilotRequestsOpt}
	}

	var authMethod string
	selectField := huh.NewSelect[string]().
		Title("How would you like Copilot workflows to authenticate?").
		Description("copilot-requests uses the org's Copilot billing seat — no PAT required.\nPAT uses a fine-grained personal access token stored as COPILOT_GITHUB_TOKEN (requires repo write access to configure).").
		Options(options...).
		Value(&authMethod)

	if probe.Disabled {
		selectField = selectField.Validate(func(v string) error {
			if v == authMethodCopilotRequests {
				return errors.New("org Copilot CLI billing is disabled — please choose PAT")
			}
			return nil
		})
	}

	form := huh.NewForm(
		huh.NewGroup(selectField),
	).WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())

	if err := form.RunWithContext(c.Ctx); err != nil {
		return fmt.Errorf("failed to select Copilot authentication method: %w", err)
	}

	c.applyCopilotAuthMethodChoice(authMethod)
	return nil
}

// applyCopilotAuthMethodChoice records the user's Copilot auth method selection and prints
// the corresponding status message. It is pure (no I/O beyond stderr) and intentionally
// separated from the huh form so the assignment logic is unit-testable without mocking the TUI.
func (c *AddInteractiveConfig) applyCopilotAuthMethodChoice(authMethod string) {
	if authMethod == authMethodCopilotRequests {
		c.UseCopilotRequests = true
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Selected copilot-requests: permissions.copilot-requests: write will be added to your workflow"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No COPILOT_GITHUB_TOKEN secret is required — Copilot usage is billed to your org's Copilot seat."))
	} else {
		c.UseCopilotRequests = false
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("A fine-grained PAT with Copilot Requests permission will be required."))
	}
}
