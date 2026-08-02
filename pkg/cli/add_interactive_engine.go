package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// selectAIEngineAndKey prompts the user to select an AI engine and provide API key
func (c *AddInteractiveConfig) selectAIEngineAndKey() error {
	addInteractiveLog.Print("Starting coding agent selection")
	if err := c.checkExistingSecrets(); err != nil {
		return err
	}
	workflowSpecifiedEngine := c.workflowSpecifiedEngine()
	defaultEngine := c.defaultInteractiveEngine(workflowSpecifiedEngine)
	if c.EngineOverride != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using coding agent: "+c.EngineOverride))
		return c.configureEngineAPISecret(c.EngineOverride)
	}
	c.printWorkflowSpecifiedEngine(workflowSpecifiedEngine)
	selectedEngine, err := c.promptForInteractiveEngine(defaultEngine, workflowSpecifiedEngine)
	if err != nil {
		return err
	}
	c.EngineOverride = selectedEngine
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Selected engine: "+selectedEngine))
	return c.configureEngineAPISecret(selectedEngine)
}

func (c *AddInteractiveConfig) workflowSpecifiedEngine() string {
	if c.resolvedWorkflows == nil || len(c.resolvedWorkflows.Workflows) == 0 {
		return ""
	}
	for _, wf := range c.resolvedWorkflows.Workflows {
		if wf.Engine != "" {
			addInteractiveLog.Printf("Workflow specifies engine in frontmatter: %s", wf.Engine)
			return wf.Engine
		}
	}
	return ""
}

func (c *AddInteractiveConfig) defaultInteractiveEngine(workflowSpecifiedEngine string) string {
	if c.EngineOverride != "" {
		return c.EngineOverride
	}
	if engine := c.defaultEngineFromSecrets(); engine != "" {
		return engine
	}
	if workflowSpecifiedEngine != "" {
		return workflowSpecifiedEngine
	}
	if engine := defaultEngineFromEnvironment(); engine != "" {
		return engine
	}
	return string(constants.DefaultEngine)
}

func (c *AddInteractiveConfig) defaultEngineFromSecrets() string {
	for _, opt := range constants.EngineOptions {
		if setutil.Contains(c.existingSecrets, opt.SecretName) {
			addInteractiveLog.Printf("Found existing secret %s, recommending engine: %s", opt.SecretName, opt.Value)
			return opt.Value
		}
	}
	return ""
}

func defaultEngineFromEnvironment() string {
	for _, opt := range constants.EngineOptions {
		envVar := opt.SecretName
		if opt.EnvVarName != "" {
			envVar = opt.EnvVarName
		}
		if lookupEnv(envVar) != "" {
			addInteractiveLog.Printf("Found env var %s, recommending engine: %s", envVar, opt.Value)
			return opt.Value
		}
	}
	return ""
}

func (c *AddInteractiveConfig) printWorkflowSpecifiedEngine(workflowSpecifiedEngine string) {
	if workflowSpecifiedEngine != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflow specifies engine: "+workflowSpecifiedEngine))
	}
}

func (c *AddInteractiveConfig) promptForInteractiveEngine(defaultEngine, workflowSpecifiedEngine string) (string, error) {
	engineOptions := reorderEngineOptions(buildInteractiveEngineOptions(c.existingSecrets, workflowSpecifiedEngine), defaultEngine)
	var selectedEngine string
	fmt.Fprintln(os.Stderr, "")
	form := console.NewSelectForm(huh.NewSelect[string]().Title("Which coding agent would you like to use?").Description("This determines which coding agent processes your workflows").Options(engineOptions...).Value(&selectedEngine))
	if err := form.RunWithContext(c.Ctx); err != nil {
		return "", fmt.Errorf("failed to select coding agent: %w", err)
	}
	return selectedEngine, nil
}

func buildInteractiveEngineOptions(existingSecrets map[string]struct{}, workflowSpecifiedEngine string) []huh.Option[string] {
	catalog := workflow.NewEngineCatalog(workflow.NewEngineRegistry())
	return sliceutil.Map(catalog.All(), func(def *workflow.EngineDefinition) huh.Option[string] {
		label := fmt.Sprintf("%s - %s", def.DisplayName, def.Description)
		label += interactiveEngineSecretMarker(existingSecrets, def.ID)
		if def.ID == workflowSpecifiedEngine {
			label += " [specified in workflow]"
		}
		return huh.NewOption(label, def.ID)
	})
}

func interactiveEngineSecretMarker(existingSecrets map[string]struct{}, engineID string) string {
	opt := constants.GetEngineOption(engineID)
	if opt != nil && setutil.Contains(existingSecrets, opt.SecretName) {
		return " [secret exists]"
	}
	return " [no secret]"
}

func reorderEngineOptions(engineOptions []huh.Option[string], defaultEngine string) []huh.Option[string] {
	for i, opt := range engineOptions {
		if opt.Value == defaultEngine && i > 0 {
			engineOptions[0], engineOptions[i] = engineOptions[i], engineOptions[0]
			break
		}
	}
	return engineOptions
}

// configureEngineAPISecret collects the API key for the selected engine using the unified engine secrets functions
// configureEngineAPISecret collects the API key for the selected engine using the unified engine secrets functions
func (c *AddInteractiveConfig) configureEngineAPISecret(engine string) error {
	addInteractiveLog.Printf("Collecting API key for engine: %s", engine)
	if c.SkipSecret {
		printSkippedEngineSecretSetup(engine)
		return nil
	}
	if err := c.maybeConfigureCopilotAuth(engine); err != nil || c.UseCopilotRequests {
		return err
	}
	if !c.hasWriteAccess {
		printEngineSecretWriteAccessMessage(engine, c.RepoOverride)
		return nil
	}
	return c.ensureEngineSecretConfigured(engine)
}

func printSkippedEngineSecretSetup(engine string) {
	if opt := constants.GetEngineOption(engine); opt != nil {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping %s secret setup (--no-secret flag set).", opt.SecretName)))
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping secret setup (--no-secret flag set)."))
}

func (c *AddInteractiveConfig) maybeConfigureCopilotAuth(engine string) error {
	if engine != string(constants.CopilotEngine) || c.Ctx == nil {
		return nil
	}
	if err := c.selectCopilotAuthMethod(); err != nil {
		return err
	}
	return nil
}

func printEngineSecretWriteAccessMessage(engine, repoOverride string) {
	opt := constants.GetEngineOption(engine)
	if opt == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping %s secret setup — write access is required to configure repository secrets.", opt.SecretName)))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Once you have write access or an admin configures the repository, set the secret with:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  gh aw secrets set %s --repo %s", opt.SecretName, repoOverride)))
}

func (c *AddInteractiveConfig) ensureEngineSecretConfigured(engine string) error {
	config := EngineSecretConfig{Ctx: c.Ctx, RepoSlug: c.RepoOverride, Engine: engine, Verbose: c.Verbose, ExistingSecrets: c.existingSecrets, IncludeSystemSecrets: false, IncludeOptional: false}
	if err := checkAndEnsureEngineSecretsForEngine(config); err != nil {
		return err
	}
	if opt := constants.GetEngineOption(engine); opt != nil {
		c.existingSecrets[opt.SecretName] = struct{}{}
		addInteractiveLog.Printf("Updated existingSecrets to include %s after upload", opt.SecretName)
	}
	return nil
}

// authMethodCopilotRequests is the wizard option value for Copilot org-billing authentication
// authMethodCopilotRequests is the wizard option value for Copilot org-billing authentication
// (permissions.copilot-requests: write). Extracted as a package-level constant so both the
// form definition and applyCopilotAuthMethodChoice reference the same sentinel.
const authMethodCopilotRequests = "copilot-requests"

// selectCopilotAuthMethod prompts the user to choose between copilot-requests (org billing)
// and a Personal Access Token for Copilot authentication.
// Sets c.UseCopilotRequests when org billing is chosen.
func (c *AddInteractiveConfig) selectCopilotAuthMethod() error {
	addInteractiveLog.Print("Prompting user for Copilot authentication method")
	probe, copilotRequestsLabel := c.probeCopilotAuthMethod()
	options := buildCopilotAuthOptions(probe, copilotRequestsLabel)
	if probe.InfoNote != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(probe.InfoNote))
	}
	fmt.Fprintln(os.Stderr, "")
	authMethod, err := runCopilotAuthMethodForm(c.Ctx, probe, options)
	if err != nil {
		return err
	}
	c.applyCopilotAuthMethodChoice(authMethod)
	return nil
}

func (c *AddInteractiveConfig) probeCopilotAuthMethod() (orgCopilotBillingProbeResult, string) {
	copilotRequestsLabel := "Use copilot-requests (org's Copilot billing, no PAT)"
	probe := orgCopilotBillingProbeResult{InfoNote: copilotBillingInconclusiveNote}
	if orgLogin, _, found := strings.Cut(c.RepoOverride, "/"); found && orgLogin != "" {
		probe = probeCopilotBillingForOrg(c.Ctx, orgLogin)
	}
	c.copilotCLIBillingStatus = probe.BillingStatus
	return probe, copilotRequestsLabel + probe.LabelSuffix
}

func buildCopilotAuthOptions(probe orgCopilotBillingProbeResult, copilotRequestsLabel string) []huh.Option[string] {
	const authMethodPAT = "pat"
	patOpt := huh.NewOption("Use a Personal Access Token (PAT) as COPILOT_GITHUB_TOKEN", authMethodPAT)
	copilotRequestsOpt := huh.NewOption(copilotRequestsLabel, authMethodCopilotRequests)
	if probe.BillingStatus == "enabled" {
		return []huh.Option[string]{copilotRequestsOpt.Selected(true), patOpt}
	}
	return []huh.Option[string]{patOpt.Selected(true), copilotRequestsOpt}
}

func runCopilotAuthMethodForm(ctx context.Context, probe orgCopilotBillingProbeResult, options []huh.Option[string]) (string, error) {
	var authMethod string
	description := "copilot-requests uses the org's Copilot billing seat — no PAT required.\n" +
		"PAT uses a fine-grained personal access token stored as COPILOT_GITHUB_TOKEN (requires repo write access to configure)."
	selectField := huh.NewSelect[string]().Title("How would you like Copilot workflows to authenticate?").Description(description).Options(options...).Value(&authMethod)
	if probe.Disabled {
		selectField = selectField.Validate(func(v string) error {
			if v == authMethodCopilotRequests {
				return errors.New("org Copilot CLI billing is disabled — please choose PAT")
			}
			return nil
		})
	}
	if err := console.NewSelectForm(selectField).RunWithContext(ctx); err != nil {
		return "", fmt.Errorf("failed to select Copilot authentication method: %w", err)
	}
	return authMethod, nil
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
