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
	"github.com/github/gh-aw/pkg/envutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var addInteractiveLog = logger.New("cli:add_interactive")

// AddInteractiveConfig holds configuration for interactive add mode
type AddInteractiveConfig struct {
	Ctx                    context.Context // Context for cancellation (Ctrl-C handling)
	WorkflowSpecs          []string
	Verbose                bool
	EngineOverride         string
	NoGitattributes        bool
	WorkflowDir            string
	NoStopAfter            bool
	StopAfter              string
	SkipWorkflowRun        bool
	SkipSecret             bool   // Skip the API secret prompt (useful when secret is set at org level)
	RepoOverride           string // owner/repo format, if user provides it
	AppendText             string // Extra content to append to the workflow on installation
	DisableSecurityScanner bool   // Disable security scanning of workflow markdown content

	// UseCopilotRequests indicates the user chose org-billing (copilot-requests) auth
	// instead of a PAT when setting up the Copilot engine during the wizard.
	// When true, COPILOT_GITHUB_TOKEN secret setup is skipped and
	// permissions.copilot-requests: write is injected into the workflow.
	UseCopilotRequests bool

	// copilotCLIBillingStatus is the detected org Copilot CLI billing status.
	// "enabled" — confirmed available; "disabled" — confirmed unavailable; "" — inconclusive.
	// Populated by selectCopilotAuthMethod() via probeCopilotBillingForOrg().
	copilotCLIBillingStatus string

	// isPublicRepo tracks whether the target repository is public
	// This is populated by checkGitRepository() when determining the repo
	isPublicRepo bool

	// hasWriteAccess tracks whether the user has write access to the target repository.
	// When false, secrets configuration is skipped since users cannot configure repository secrets.
	hasWriteAccess bool

	// existingSecrets tracks which secrets already exist in the repository
	// This is populated by checkExistingSecrets() before engine selection
	existingSecrets map[string]struct{}

	// addResult holds the result from AddWorkflows, including HasWorkflowDispatch
	addResult *AddWorkflowsResult

	// resolvedWorkflows holds the pre-resolved workflow data including descriptions
	// This is populated early in the flow by resolveWorkflows()
	resolvedWorkflows *ResolvedWorkflows
}

// RunAddInteractive runs the interactive add workflow
// This walks the user through adding an agentic workflow to their repository.
// ctx is applied to config.Ctx; callers should not rely on config.Ctx after this call
// as it will be overwritten by the provided ctx.
func RunAddInteractive(ctx context.Context, config *AddInteractiveConfig) error {
	addInteractiveLog.Print("Starting interactive add workflow")

	// Assert this function is not running in automated unit tests or CI.
	// GO_TEST_MODE intentionally uses GetBoolFromEnv so common boolean spellings
	// are treated consistently across test and automation environments, while
	// IsRunningInCI centralizes the broader CI environment detection logic.
	if envutil.GetBoolFromEnv("GO_TEST_MODE", false, addInteractiveLog) || IsRunningInCI() {
		return errors.New("interactive add cannot be used in automated tests or CI environments")
	}

	// Set context on the config
	config.Ctx = ctx

	config.configureDefaultGHHostFromRemote()

	if err := config.runInitialAddInteractiveChecks(); err != nil {
		return err
	}

	remainingBootstrapProfile := config.getRemainingBootstrapProfile()

	filesToAdd, initFiles, secretName, secretValue, createPR, err := config.prepareAndConfirmAddInteractive()
	if err != nil {
		return err
	}

	if err := config.createWorkflowChangesAndConfigureSecret(ctx, filesToAdd, initFiles, secretName, secretValue, createPR); err != nil {
		return err
	}
	if !createPR {
		config.showFinalInstructions()
		return nil
	}

	if err := config.applyBootstrapConfigIfNeeded(ctx, remainingBootstrapProfile); err != nil {
		return err
	}

	// Step 10: Check status and offer to run
	if err := config.checkStatusAndOfferRun(ctx); err != nil {
		return err
	}

	return nil
}

func (c *AddInteractiveConfig) configureDefaultGHHostFromRemote() {
	if os.Getenv("GH_HOST") != "" { //nolint:osgetenvlibrary
		return
	}
	detectedHost := getHostFromOriginRemote()
	if detectedHost == "github.com" {
		return
	}
	addInteractiveLog.Printf("Auto-detected GHES host from git remote: %s", detectedHost)
	workflow.SetDefaultGHHost(detectedHost)
	if c.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Auto-detected GitHub Enterprise host: "+detectedHost))
	}
}

func (c *AddInteractiveConfig) getRemainingBootstrapProfile() *resolvedBootstrapProfile {
	if c.resolvedWorkflows == nil {
		return nil
	}
	// All config steps run post-install in the exact order they are declared in the
	// manifest. We no longer split them into a pre-install and post-install phase so
	// that the declared ordering is preserved.
	return c.resolvedWorkflows.BootstrapProfile
}

func (c *AddInteractiveConfig) applyBootstrapConfigIfNeeded(ctx context.Context, profile *resolvedBootstrapProfile) error {
	if profile == nil {
		return nil
	}
	if c.hasWriteAccess {
		return executeBootstrapConfigForAdd(ctx, c.RepoOverride, c.WorkflowSpecs, profile, c.UseCopilotRequests, c.Verbose)
	}
	printBootstrapConfigTODO(os.Stderr, profile)
	return nil
}

func (c *AddInteractiveConfig) runInitialAddInteractiveChecks() error {
	console.ShowWelcomeBanner("This tool will walk you through adding an automated workflow to your repository.")
	if err := c.resolveWorkflows(); err != nil {
		return err
	}
	c.showWorkflowDescriptions()
	if err := c.checkGHAuthStatus(); err != nil {
		return err
	}
	if err := c.checkGitRepository(); err != nil {
		return err
	}
	if err := c.checkActionsEnabled(); err != nil {
		return err
	}
	return c.checkUserPermissions()
}

func (c *AddInteractiveConfig) prepareAndConfirmAddInteractive() (workflowFiles, initFiles []string, secretName, secretValue string, createPR bool, err error) {
	if err := c.selectAIEngineAndKey(); err != nil {
		return nil, nil, "", "", false, err
	}

	initFiles, err = ensureAddRepositoryInitializedWithDetails(c.EngineOverride, c.Verbose, c.NoGitattributes)
	if err != nil {
		return nil, nil, "", "", false, err
	}

	workflowFiles, _, err = c.determineFilesToAdd()
	if err != nil {
		return nil, nil, "", "", false, err
	}

	if err := c.selectScheduleFrequency(); err != nil {
		return nil, nil, "", "", false, err
	}

	if c.hasWriteAccess && !c.SkipSecret && !c.UseCopilotRequests {
		secretName, secretValue, err = c.resolveEngineApiKeyCredential()
		if err != nil {
			return nil, nil, "", "", false, err
		}
	}

	createPR, err = c.confirmChanges(workflowFiles, initFiles, secretName, secretValue)
	if err != nil {
		return nil, nil, "", "", false, err
	}

	return workflowFiles, initFiles, secretName, secretValue, createPR, nil
}

// resolveWorkflows resolves workflow specifications by installing repositories,
// expanding wildcards, and fetching workflow content (including descriptions).
// This is called early to show workflow information before the user commits to adding them.
func (c *AddInteractiveConfig) resolveWorkflows() error {
	addInteractiveLog.Print("Resolving workflows early for description display")

	resolved, err := ResolveWorkflows(c.Ctx, c.WorkflowSpecs, c.Verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve workflows: %w", err)
	}

	c.resolvedWorkflows = resolved
	return nil
}

// showWorkflowDescriptions displays the descriptions of resolved workflows
func (c *AddInteractiveConfig) showWorkflowDescriptions() {
	if !c.Verbose {
		return
	}

	if c.resolvedWorkflows == nil || len(c.resolvedWorkflows.Workflows) == 0 {
		return
	}

	// Show descriptions for all workflows that have one
	for _, rw := range c.resolvedWorkflows.Workflows {
		if rw.Description != "" {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(rw.Description))
			fmt.Fprintln(os.Stderr, "")
		}
	}
}

// determineFilesToAdd determines which files will be added
func (c *AddInteractiveConfig) determineFilesToAdd() (workflowFiles []string, initFiles []string, err error) {
	addInteractiveLog.Print("Determining files to add")

	// Prefer the pre-resolved workflows (populated by resolveWorkflows). Fall back
	// to parsing the raw WorkflowSpecs when no workflows were resolved.
	if c.resolvedWorkflows != nil && len(c.resolvedWorkflows.Workflows) > 0 {
		workflowSpecsForError := strings.Join(c.WorkflowSpecs, ", ")
		for i, rw := range c.resolvedWorkflows.Workflows {
			if rw == nil {
				return nil, nil, fmt.Errorf("resolved workflow at position %d from %q is nil", i+1, workflowSpecsForError)
			}
			if rw.Spec == nil {
				return nil, nil, fmt.Errorf("resolved workflow at position %d from %q is missing its specification", i+1, workflowSpecsForError)
			}
			workflowName := strings.TrimSpace(rw.Spec.WorkflowName)
			if workflowName == "" {
				return nil, nil, fmt.Errorf("resolved workflow at position %d from %q is missing its workflow name", i+1, workflowSpecsForError)
			}
			if rw.IsActionWorkflow {
				// Raw GitHub Actions YAML files are installed as-is; no .lock.yml is produced.
				workflowFiles = append(workflowFiles, workflowName+".yml")
			} else {
				workflowFiles = append(workflowFiles, workflowName+".md")
				workflowFiles = append(workflowFiles, workflowName+".lock.yml")
			}
		}
	} else {
		// Fallback: derive file names from unresolved spec strings. All are assumed to be
		// agentic workflow .md files since we have no resolution metadata here.
		workflowNames, nameErr := c.workflowNamesForInteractiveAdd()
		if nameErr != nil {
			return nil, nil, nameErr
		}
		for _, workflowName := range workflowNames {
			workflowFiles = append(workflowFiles, workflowName+".md")
			workflowFiles = append(workflowFiles, workflowName+".lock.yml")
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "The following workflow files will be added:")
	for _, f := range workflowFiles {
		fmt.Fprintf(os.Stderr, "  • .github/workflows/%s\n", f)
	}

	return workflowFiles, initFiles, nil
}

func (c *AddInteractiveConfig) workflowNamesForInteractiveAdd() ([]string, error) {
	workflowSpecsForError := strings.Join(c.WorkflowSpecs, ", ")
	if c.resolvedWorkflows != nil && len(c.resolvedWorkflows.Workflows) > 0 {
		workflowNames := make([]string, 0, len(c.resolvedWorkflows.Workflows))
		for i, resolvedWorkflow := range c.resolvedWorkflows.Workflows {
			if resolvedWorkflow == nil {
				return nil, fmt.Errorf("resolved manifest workflow at position %d from %q is nil", i+1, workflowSpecsForError)
			}
			if resolvedWorkflow.Spec == nil {
				return nil, fmt.Errorf("resolved manifest workflow at position %d from %q is missing its specification", i+1, workflowSpecsForError)
			}
			workflowName := strings.TrimSpace(resolvedWorkflow.Spec.WorkflowName)
			if workflowName == "" {
				return nil, fmt.Errorf("resolved manifest workflow at position %d from %q is missing its workflow name", i+1, workflowSpecsForError)
			}
			workflowNames = append(workflowNames, workflowName)
		}
		return workflowNames, nil
	}

	workflowNames := make([]string, 0, len(c.WorkflowSpecs))
	for _, spec := range c.WorkflowSpecs {
		parsed, parseErr := parseWorkflowSpec(spec)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid workflow specification '%s': %w", spec, parseErr)
		}
		workflowNames = append(workflowNames, parsed.WorkflowName)
	}
	return workflowNames, nil
}

func (c *AddInteractiveConfig) primaryWorkflowName() string {
	workflowNames, err := c.workflowNamesForInteractiveAdd()
	if err != nil || len(workflowNames) == 0 {
		return ""
	}
	return workflowNames[0]
}

// confirmChanges asks the user to confirm the changes
// secretValue is empty if the secret already exists in the repository
func (c *AddInteractiveConfig) confirmChanges(workflowFiles, initFiles []string, secretName string, secretValue string) (bool, error) {
	addInteractiveLog.Print("Confirming changes with user")

	fmt.Fprintln(os.Stderr, "")
	if len(initFiles) > 0 {
		fmt.Fprintln(os.Stderr, "The repository will also be initialized with:")
		for _, f := range initFiles {
			fmt.Fprintf(os.Stderr, "  • %s\n", f)
		}
		fmt.Fprintln(os.Stderr, "")
	}

	createPR := true // Default to yes
	form := console.NewConfirmForm(
		huh.NewConfirm().
			Title("Do you want to create a pull request with these changes?").
			Description("Choose No to write the workflow files locally without creating a pull request").
			Affirmative("Yes, create pull request").
			Negative("No, write files locally").
			Value(&createPR),
	)

	if err := form.RunWithContext(c.Ctx); err != nil {
		return false, fmt.Errorf("confirmation failed: %w", err)
	}

	return createPR, nil
}

// showFinalInstructions shows final instructions to the user
func (c *AddInteractiveConfig) showFinalInstructions() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("🎉 Addition complete!"))
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, "")

	// Show summary with workflow name(s)
	if c.resolvedWorkflows != nil && len(c.resolvedWorkflows.Workflows) > 0 {
		wf := c.resolvedWorkflows.Workflows[0]
		fmt.Fprintf(os.Stderr, "The workflow '%s' has been added to the repository and will now run automatically.\n", wf.Spec.WorkflowName)
		c.showWorkflowDescriptions()
	}

	fmt.Fprintln(os.Stderr, "Useful commands:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s status          # Check workflow status", string(constants.CLIExtensionPrefix))))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s run <workflow>  # Trigger a workflow", string(constants.CLIExtensionPrefix))))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s logs            # View workflow logs", string(constants.CLIExtensionPrefix))))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Learn more at: https://github.github.com/gh-aw/")
	fmt.Fprintln(os.Stderr, "")
}
