package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var addInteractiveLog = logger.New("cli:add_interactive")

// AddInteractiveConfig holds configuration for interactive add mode
type AddInteractiveConfig struct {
	WorkflowSpecs   []string
	Verbose         bool
	EngineOverride  string
	NoGitattributes bool
	WorkflowDir     string
	NoStopAfter     bool
	StopAfter       string
	SkipWorkflowRun bool
	RepoOverride    string // owner/repo format, if user provides it

	// isPublicRepo tracks whether the target repository is public
	// This is populated by checkGitRepository() when determining the repo
	isPublicRepo bool

	// existingSecrets tracks which secrets already exist in the repository
	// This is populated by checkExistingSecrets() before engine selection
	existingSecrets map[string]bool

	// addResult holds the result from AddWorkflows, including HasWorkflowDispatch
	addResult *AddWorkflowsResult

	// resolvedWorkflows holds the pre-resolved workflow data including descriptions
	// This is populated early in the flow by resolveWorkflows()
	resolvedWorkflows *ResolvedWorkflows
}

// RunAddInteractive runs the interactive add workflow
// This walks the user through adding an agentic workflow to their repository
func RunAddInteractive(ctx context.Context, workflowSpecs []string, verbose bool, engineOverride string, noGitattributes bool, workflowDir string, noStopAfter bool, stopAfter string) error {
	addInteractiveLog.Print("Starting interactive add workflow")

	// Assert this function is not running in automated unit tests or CI
	if os.Getenv("GO_TEST_MODE") == "true" || os.Getenv("CI") != "" {
		return fmt.Errorf("interactive add cannot be used in automated tests or CI environments")
	}

	config := &AddInteractiveConfig{
		WorkflowSpecs:   workflowSpecs,
		Verbose:         verbose,
		EngineOverride:  engineOverride,
		NoGitattributes: noGitattributes,
		WorkflowDir:     workflowDir,
		NoStopAfter:     noStopAfter,
		StopAfter:       stopAfter,
	}

	// Clear the screen for a fresh interactive experience
	fmt.Fprint(os.Stderr, "\033[H\033[2J")

	// Step 1: Welcome message
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "🚀 Welcome to GitHub Agentic Workflows!")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "This tool will walk you through adding an automated workflow to your repository.")
	fmt.Fprintln(os.Stderr, "")

	// Step 1b: Resolve workflows early to get descriptions and validate specs
	if err := config.resolveWorkflows(); err != nil {
		return err
	}

	// Step 1c: Show workflow descriptions if available
	config.showWorkflowDescriptions()

	// Step 2: Check gh auth status
	if err := config.checkGHAuthStatus(); err != nil {
		return err
	}

	// Step 3: Check git repository and get org/repo
	if err := config.checkGitRepository(); err != nil {
		return err
	}

	// Step 4: Check GitHub Actions is enabled
	if err := config.checkActionsEnabled(); err != nil {
		return err
	}

	// Step 5: Check user permissions
	if err := config.checkUserPermissions(); err != nil {
		return err
	}

	// Step 6: Select coding agent and collect API key
	if err := config.selectAIEngineAndKey(); err != nil {
		return err
	}

	// Step 7: Determine files to add
	filesToAdd, initFiles, err := config.determineFilesToAdd()
	if err != nil {
		return err
	}

	// Step 8: Confirm with user
	secretName, secretValue, err := config.getSecretInfo()
	if err != nil {
		return err
	}

	if err := config.confirmChanges(filesToAdd, initFiles, secretName, secretValue); err != nil {
		return err
	}

	// Step 9: Apply changes (create PR, merge, add secret)
	if err := config.applyChanges(ctx, filesToAdd, initFiles, secretName, secretValue); err != nil {
		return err
	}

	// Step 10: Check status and offer to run
	if err := config.checkStatusAndOfferRun(ctx); err != nil {
		return err
	}

	return nil
}

// resolveWorkflows resolves workflow specifications by installing repositories,
// expanding wildcards, and fetching workflow content (including descriptions).
// This is called early to show workflow information before the user commits to adding them.
func (c *AddInteractiveConfig) resolveWorkflows() error {
	addInteractiveLog.Print("Resolving workflows early for description display")

	resolved, err := ResolveWorkflows(c.WorkflowSpecs, c.Verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve workflows: %w", err)
	}

	c.resolvedWorkflows = resolved
	return nil
}

// showWorkflowDescriptions displays the descriptions of resolved workflows
func (c *AddInteractiveConfig) showWorkflowDescriptions() {
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

// checkGHAuthStatus verifies the user is logged in to GitHub CLI
func (c *AddInteractiveConfig) checkGHAuthStatus() error {
	addInteractiveLog.Print("Checking GitHub CLI authentication status")

	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("You are not logged in to GitHub CLI."))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Please run the following command to authenticate:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage("  gh auth login"))
		fmt.Fprintln(os.Stderr, "")
		return fmt.Errorf("not authenticated with GitHub CLI")
	}

	if c.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("GitHub CLI authenticated"))
		addInteractiveLog.Printf("gh auth status output: %s", string(output))
	}

	return nil
}

// checkGitRepository verifies we're in a git repo and gets org/repo info
func (c *AddInteractiveConfig) checkGitRepository() error {
	addInteractiveLog.Print("Checking git repository status")

	// Check if we're in a git repository
	if !isGitRepo() {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Not in a git repository."))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Please navigate to a git repository or initialize one with:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage("  git init"))
		fmt.Fprintln(os.Stderr, "")
		return fmt.Errorf("not in a git repository")
	}

	// Try to get the repository slug
	repoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		addInteractiveLog.Printf("Could not determine repository automatically: %v", err)

		// Ask the user for the repository
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not determine the repository automatically."))
		fmt.Fprintln(os.Stderr, "")

		var userRepo string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter the target repository (owner/repo):").
					Description("For example: myorg/myrepo").
					Value(&userRepo).
					Validate(func(s string) error {
						parts := strings.Split(s, "/")
						if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
							return fmt.Errorf("please enter in format 'owner/repo'")
						}
						return nil
					}),
			),
		).WithAccessible(console.IsAccessibleMode())

		if err := form.Run(); err != nil {
			return fmt.Errorf("failed to get repository info: %w", err)
		}

		c.RepoOverride = userRepo
		repoSlug = userRepo
	} else {
		c.RepoOverride = repoSlug
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Target repository: %s", repoSlug)))
	addInteractiveLog.Printf("Target repository: %s", repoSlug)

	// Check if repository is public or private
	c.isPublicRepo = c.checkRepoVisibility()

	return nil
}

// checkRepoVisibility checks if the repository is public or private
func (c *AddInteractiveConfig) checkRepoVisibility() bool {
	addInteractiveLog.Print("Checking repository visibility")

	// Use gh api to check repository visibility
	args := []string{"api", fmt.Sprintf("/repos/%s", c.RepoOverride), "--jq", ".visibility"}
	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()
	if err != nil {
		addInteractiveLog.Printf("Could not check repository visibility: %v", err)
		// Default to public if we can't determine
		return true
	}

	visibility := strings.TrimSpace(string(output))
	isPublic := visibility == "public"
	addInteractiveLog.Printf("Repository visibility: %s (isPublic=%v)", visibility, isPublic)
	return isPublic
}

// checkActionsEnabled verifies that GitHub Actions is enabled for the repository
func (c *AddInteractiveConfig) checkActionsEnabled() error {
	addInteractiveLog.Print("Checking if GitHub Actions is enabled")

	// Use gh api to check Actions permissions
	args := []string{"api", fmt.Sprintf("/repos/%s/actions/permissions", c.RepoOverride), "--jq", ".enabled"}
	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()
	if err != nil {
		addInteractiveLog.Printf("Failed to check Actions status: %v", err)
		// If we can't check, warn but continue - actual operations will fail if Actions is disabled
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not verify GitHub Actions status. Proceeding anyway..."))
		return nil
	}

	enabled := strings.TrimSpace(string(output))
	if enabled != "true" {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("GitHub Actions is disabled for this repository."))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To enable GitHub Actions:")
		fmt.Fprintln(os.Stderr, "  1. Go to your repository on GitHub")
		fmt.Fprintln(os.Stderr, "  2. Navigate to Settings → Actions → General")
		fmt.Fprintln(os.Stderr, "  3. Under 'Actions permissions', select 'Allow all actions and reusable workflows'")
		fmt.Fprintln(os.Stderr, "  4. Click 'Save'")
		fmt.Fprintln(os.Stderr, "")
		return fmt.Errorf("GitHub Actions is not enabled for this repository")
	}

	if c.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("GitHub Actions is enabled"))
	}

	return nil
}

// checkUserPermissions verifies the user has write/admin access
func (c *AddInteractiveConfig) checkUserPermissions() error {
	addInteractiveLog.Print("Checking user permissions")

	parts := strings.Split(c.RepoOverride, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s", c.RepoOverride)
	}
	owner, repo := parts[0], parts[1]

	hasAccess, err := checkRepositoryAccess(owner, repo)
	if err != nil {
		addInteractiveLog.Printf("Failed to check repository access: %v", err)
		// If we can't check, warn but continue - actual operations will fail if no access
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not verify repository permissions. Proceeding anyway..."))
		return nil
	}

	if !hasAccess {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("You do not have write access to %s/%s.", owner, repo)))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "You need to be a maintainer, admin, or have write permissions on this repository.")
		fmt.Fprintln(os.Stderr, "Please contact the repository owner or request access.")
		fmt.Fprintln(os.Stderr, "")
		return fmt.Errorf("insufficient repository permissions")
	}

	if c.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository permissions verified"))
	}

	return nil
}

// checkExistingSecrets fetches which secrets already exist in the repository
func (c *AddInteractiveConfig) checkExistingSecrets() error {
	addInteractiveLog.Print("Checking existing repository secrets")

	c.existingSecrets = make(map[string]bool)

	// Use gh api to list repository secrets
	args := []string{"api", fmt.Sprintf("/repos/%s/actions/secrets", c.RepoOverride), "--jq", ".secrets[].name"}
	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()
	if err != nil {
		addInteractiveLog.Printf("Could not fetch existing secrets: %v", err)
		// Continue without error - we'll just assume no secrets exist
		return nil
	}

	// Parse the output - each secret name is on its own line
	secretNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, name := range secretNames {
		name = strings.TrimSpace(name)
		if name != "" {
			c.existingSecrets[name] = true
			addInteractiveLog.Printf("Found existing secret: %s", name)
		}
	}

	if c.Verbose && len(c.existingSecrets) > 0 {
		fmt.Fprintf(os.Stderr, "Found %d existing repository secret(s)\n", len(c.existingSecrets))
	}

	return nil
}

// selectAIEngineAndKey prompts the user to select an AI engine and provide API key
func (c *AddInteractiveConfig) selectAIEngineAndKey() error {
	addInteractiveLog.Print("Starting coding agent selection")

	// First, check which secrets already exist in the repository
	if err := c.checkExistingSecrets(); err != nil {
		return err
	}

	// Determine default engine based on workflow preference, existing secrets, then environment
	defaultEngine := string(constants.CopilotEngine)
	existingSecretNote := ""

	// If engine is explicitly overridden via flag, use that
	if c.EngineOverride != "" {
		defaultEngine = c.EngineOverride
	} else {
		// Priority 0: Check if workflow specifies a preferred engine in frontmatter
		if c.resolvedWorkflows != nil && len(c.resolvedWorkflows.Workflows) > 0 {
			for _, wf := range c.resolvedWorkflows.Workflows {
				if wf.Engine != "" {
					defaultEngine = wf.Engine
					addInteractiveLog.Printf("Using engine from workflow frontmatter: %s", wf.Engine)
					break
				}
			}
		}
	}

	// Only check secrets/environment if we haven't already set a preference
	workflowHasPreference := c.resolvedWorkflows != nil && len(c.resolvedWorkflows.Workflows) > 0 && c.resolvedWorkflows.Workflows[0].Engine != ""
	if c.EngineOverride == "" && !workflowHasPreference {
		// Priority 1: Check existing repository secrets using EngineOptions
		for _, opt := range constants.EngineOptions {
			if c.existingSecrets[opt.SecretName] {
				defaultEngine = opt.Value
				existingSecretNote = fmt.Sprintf(" (existing %s secret will be used)", opt.SecretName)
				break
			}
		}

		// Priority 2: Check environment variables if no existing secret found
		if existingSecretNote == "" {
			for _, opt := range constants.EngineOptions {
				envVar := opt.SecretName
				if opt.EnvVarName != "" {
					envVar = opt.EnvVarName
				}
				if os.Getenv(envVar) != "" {
					defaultEngine = opt.Value
					break
				}
			}
			// Priority 3: Check if user likely has Copilot (default)
			if token, err := parser.GetGitHubToken(); err == nil && token != "" {
				defaultEngine = string(constants.CopilotEngine)
			}
		}
	}

	// If engine is already overridden, skip selection
	if c.EngineOverride != "" {
		fmt.Fprintf(os.Stderr, "Using coding agent: %s\n", c.EngineOverride)
		return c.collectAPIKey(c.EngineOverride)
	}

	// Build engine options with notes about existing secrets
	var engineOptions []huh.Option[string]
	for _, opt := range constants.EngineOptions {
		label := fmt.Sprintf("%s - %s", opt.Label, opt.Description)
		if c.existingSecrets[opt.SecretName] {
			label += " [secret exists]"
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
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found COPILOT_GITHUB_TOKEN in environment"))
		return nil
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "GitHub Copilot requires a Personal Access Token (PAT) with Copilot permissions.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Please create a token at:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage("  https://github.com/settings/personal-access-tokens/new"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Configure the token with:")
	fmt.Fprintln(os.Stderr, "  • Token name: Agentic Workflows Copilot")
	fmt.Fprintln(os.Stderr, "  • Expiration: 90 days (recommended for testing)")
	fmt.Fprintln(os.Stderr, "  • Resource owner: Your personal account")
	if c.isPublicRepo {
		fmt.Fprintln(os.Stderr, "  • Repository access: \"Public repositories\"")
	} else {
		fmt.Fprintf(os.Stderr, "  • Repository access: \"Only select repositories\" → select %s\n", c.RepoOverride)
	}
	fmt.Fprintln(os.Stderr, "  • Account permissions → Copilot Requests: Read")
	fmt.Fprintln(os.Stderr, "")

	var token string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("After creating, please paste your Copilot PAT:").
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
		return fmt.Errorf("failed to get Copilot token: %w", err)
	}

	// Store in environment for later use
	os.Setenv("COPILOT_GITHUB_TOKEN", token)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Copilot token received"))

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
	os.Setenv(opt.SecretName, apiKey)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("%s API key received", opt.Label)))

	return nil
}

// determineFilesToAdd determines which files will be added
func (c *AddInteractiveConfig) determineFilesToAdd() (workflowFiles []string, initFiles []string, err error) {
	addInteractiveLog.Print("Determining files to add")

	// Parse the workflow specs to get the files that will be added
	// This reuses logic from addWorkflowsNormal to determine what files get created
	for _, spec := range c.WorkflowSpecs {
		parsed, parseErr := parseWorkflowSpec(spec)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("invalid workflow specification '%s': %w", spec, parseErr)
		}
		workflowFiles = append(workflowFiles, parsed.WorkflowName+".md")
		workflowFiles = append(workflowFiles, parsed.WorkflowName+".lock.yml")
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "The following workflow files will be added:")
	for _, f := range workflowFiles {
		fmt.Fprintf(os.Stderr, "  • .github/workflows/%s\n", f)
	}

	return workflowFiles, initFiles, nil
}

// getSecretInfo returns the secret name and value based on the selected engine
// Returns empty value if the secret already exists in the repository
func (c *AddInteractiveConfig) getSecretInfo() (name string, value string, err error) {
	addInteractiveLog.Printf("Getting secret info for engine: %s", c.EngineOverride)

	opt := constants.GetEngineOption(c.EngineOverride)
	if opt == nil {
		return "", "", fmt.Errorf("unknown engine: %s", c.EngineOverride)
	}

	name = opt.SecretName

	// If secret already exists in repo, we don't need a value
	if c.existingSecrets[name] {
		addInteractiveLog.Printf("Secret %s already exists in repository", name)
		return name, "", nil
	}

	// Get value from environment variable (use EnvVarName if specified, otherwise SecretName)
	envVar := opt.SecretName
	if opt.EnvVarName != "" {
		envVar = opt.EnvVarName
	}
	value = os.Getenv(envVar)

	if value == "" {
		return "", "", fmt.Errorf("API key not found for engine %s", c.EngineOverride)
	}

	return name, value, nil
}

// confirmChanges asks the user to confirm the changes
// secretValue is empty if the secret already exists in the repository
func (c *AddInteractiveConfig) confirmChanges(workflowFiles, initFiles []string, secretName string, secretValue string) error {
	addInteractiveLog.Print("Confirming changes with user")

	fmt.Fprintln(os.Stderr, "")

	confirmed := true // Default to yes
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Do you want to proceed with these changes?").
				Description("A pull request will be created and merged automatically").
				Affirmative("Yes, create and merge").
				Negative("No, cancel").
				Value(&confirmed),
		),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if !confirmed {
		fmt.Fprintln(os.Stderr, "Operation cancelled.")
		return fmt.Errorf("user cancelled the operation")
	}

	return nil
}

// applyChanges creates the PR, merges it, and adds the secret
func (c *AddInteractiveConfig) applyChanges(ctx context.Context, workflowFiles, initFiles []string, secretName, secretValue string) error {
	addInteractiveLog.Print("Applying changes")

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Creating pull request..."))

	// Add the workflow using existing implementation with --create-pull-request
	// Pass the resolved workflows to avoid re-fetching them
	// Pass quiet=true to suppress detailed output (already shown earlier in interactive mode)
	// This returns the result including PR number and HasWorkflowDispatch
	result, err := AddResolvedWorkflows(c.WorkflowSpecs, c.resolvedWorkflows, 1, c.Verbose, true, c.EngineOverride, "", false, "", true, false, c.NoGitattributes, c.WorkflowDir, c.NoStopAfter, c.StopAfter)
	if err != nil {
		return fmt.Errorf("failed to add workflow: %w", err)
	}
	c.addResult = result

	// Step 8b: Auto-merge the PR
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Merging pull request..."))

	if result.PRNumber == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not determine PR number"))
		fmt.Fprintln(os.Stderr, "Please merge the PR manually from the GitHub web interface.")
	} else {
		if err := c.mergePullRequest(result.PRNumber); err != nil {
			// Check if already merged
			if strings.Contains(err.Error(), "already merged") || strings.Contains(err.Error(), "MERGED") {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Merged pull request %s", result.PRURL)))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to merge PR: %v", err)))
				fmt.Fprintln(os.Stderr, "Please merge the PR manually from the GitHub web interface.")
			}
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Merged pull request %s", result.PRURL)))
		}
	}

	// Step 8c: Add the secret (skip if already exists in repository)
	if secretValue == "" {
		// Secret already exists in repo, nothing to do
		if c.Verbose {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Secret '%s' already configured", secretName)))
		}
	} else {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Adding secret '%s' to repository...", secretName)))

		if err := c.addRepositorySecret(secretName, secretValue); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to add secret: %v", err)))
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Please add the secret manually:")
			fmt.Fprintln(os.Stderr, "  1. Go to your repository Settings → Secrets and variables → Actions")
			fmt.Fprintf(os.Stderr, "  2. Click 'New repository secret' and add '%s'\n", secretName)
			return fmt.Errorf("failed to add secret: %w", err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Secret '%s' added", secretName)))
	}

	return nil
}

// mergePullRequest merges the specified PR
func (c *AddInteractiveConfig) mergePullRequest(prNumber int) error {
	cmd := workflow.ExecGH("pr", "merge", fmt.Sprintf("%d", prNumber), "--repo", c.RepoOverride, "--merge")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("merge failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// addRepositorySecret adds a secret to the repository
func (c *AddInteractiveConfig) addRepositorySecret(name, value string) error {
	cmd := workflow.ExecGH("secret", "set", name, "--repo", c.RepoOverride, "--body", value)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set secret: %w (output: %s)", err, string(output))
	}
	return nil
}

// checkStatusAndOfferRun checks if the workflow appears in status and offers to run it
func (c *AddInteractiveConfig) checkStatusAndOfferRun(ctx context.Context) error {
	addInteractiveLog.Print("Checking workflow status and offering to run")

	// Wait a moment for GitHub to process the merge
	fmt.Fprintln(os.Stderr, "")

	// Use spinner only in non-verbose mode (spinner can't be restarted after stop)
	var spinner *console.SpinnerWrapper
	if !c.Verbose {
		spinner = console.NewSpinner("Waiting for workflow to be available...")
		spinner.Start()
	}

	// Try a few times to see the workflow in status
	var workflowFound bool
	for i := 0; i < 5; i++ {
		// Wait 2 seconds before each check (including the first)
		select {
		case <-ctx.Done():
			if spinner != nil {
				spinner.Stop()
			}
			return ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue with check
		}

		// Use the workflow name from the first spec
		if len(c.WorkflowSpecs) > 0 {
			parsed, _ := parseWorkflowSpec(c.WorkflowSpecs[0])
			if parsed != nil {
				if c.Verbose {
					fmt.Fprintf(os.Stderr, "Checking workflow status (attempt %d/5) for: %s\n", i+1, parsed.WorkflowName)
				}
				// Check if workflow is in status
				statuses, err := getWorkflowStatuses(parsed.WorkflowName, c.RepoOverride, c.Verbose)
				if err != nil {
					if c.Verbose {
						fmt.Fprintf(os.Stderr, "Status check error: %v\n", err)
					}
				} else if len(statuses) > 0 {
					if c.Verbose {
						fmt.Fprintf(os.Stderr, "Found %d workflow(s) matching pattern\n", len(statuses))
					}
					workflowFound = true
					break
				} else if c.Verbose {
					fmt.Fprintln(os.Stderr, "No workflows found matching pattern yet")
				}
			}
		}
	}

	if spinner != nil {
		spinner.Stop()
	}

	if !workflowFound {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not verify workflow status."))
		fmt.Fprintf(os.Stderr, "You can check status with: %s status\n", string(constants.CLIExtensionPrefix))
		c.showFinalInstructions()
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow is ready"))

	// Only offer to run if workflow has workflow_dispatch trigger
	if c.addResult == nil || !c.addResult.HasWorkflowDispatch {
		addInteractiveLog.Print("Workflow does not have workflow_dispatch trigger, skipping run offer")
		c.showFinalInstructions()
		return nil
	}

	// Ask if user wants to run the workflow
	fmt.Fprintln(os.Stderr, "")
	runNow := true // Default to yes
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Would you like to run the workflow once now?").
				Description("This will trigger the workflow immediately").
				Affirmative("Yes, run once now").
				Negative("No, I'll run later").
				Value(&runNow),
		),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return nil // Not critical, just skip
	}

	if !runNow {
		c.showFinalInstructions()
		return nil
	}

	// Run the workflow
	if len(c.WorkflowSpecs) > 0 {
		parsed, _ := parseWorkflowSpec(c.WorkflowSpecs[0])
		if parsed != nil {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Triggering workflow..."))

			if err := RunWorkflowOnGitHub(ctx, parsed.WorkflowName, false, c.EngineOverride, c.RepoOverride, "", false, false, false, true, nil, c.Verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to run workflow: %v", err)))
				c.showFinalInstructions()
				return nil
			}

			// Get the run URL for step 10
			runInfo, err := getLatestWorkflowRunWithRetry(parsed.WorkflowName+".lock.yml", c.RepoOverride, c.Verbose)
			if err == nil && runInfo.URL != "" {
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow triggered successfully!"))
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintf(os.Stderr, "🔗 View workflow run: %s\n", runInfo.URL)
			}
		}
	}

	c.showFinalInstructions()
	return nil
}

// getWorkflowStatuses is a helper to get workflow statuses for a pattern
// The pattern is matched against the workflow filename (basename without extension)
func getWorkflowStatuses(pattern, repoOverride string, verbose bool) ([]WorkflowStatus, error) {
	// This would normally call StatusWorkflows but we need just a simple check
	// For now, we'll use the gh CLI directly
	// Request 'path' field so we can match by filename, not by workflow name
	args := []string{"workflow", "list", "--json", "name,state,path"}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gh %s\n", strings.Join(args, " "))
	}

	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "gh workflow list failed: %v\n", err)
		}
		return nil, err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "gh workflow list output: %s\n", string(output))
		fmt.Fprintf(os.Stderr, "Looking for workflow with filename containing: %s\n", pattern)
	}

	// Check if any workflow path contains the pattern
	// The pattern is the workflow name (e.g., "daily-repo-status")
	// The path is like ".github/workflows/daily-repo-status.lock.yml"
	// We check if the path contains the pattern
	if strings.Contains(string(output), pattern+".lock.yml") || strings.Contains(string(output), pattern+".md") {
		if verbose {
			fmt.Fprintf(os.Stderr, "Workflow with filename '%s' found in workflow list\n", pattern)
		}
		return []WorkflowStatus{{Workflow: pattern}}, nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Workflow with filename '%s' NOT found in workflow list\n", pattern)
	}
	return nil, nil
}

// showFinalInstructions shows final instructions to the user
func (c *AddInteractiveConfig) showFinalInstructions() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("🎉 Addition complete!"))
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Useful commands:")
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s status          # Check workflow status", string(constants.CLIExtensionPrefix))))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s run <workflow>  # Trigger a workflow", string(constants.CLIExtensionPrefix))))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("  %s logs            # View workflow logs", string(constants.CLIExtensionPrefix))))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Learn more at: https://githubnext.github.io/gh-aw/")
	fmt.Fprintln(os.Stderr, "")
}
