package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var initLog = logger.New("cli:init")

// InitRepositoryInteractive runs an interactive setup for the repository
func InitRepositoryInteractive(verbose bool, rootCmd CommandProvider) error {
	initLog.Print("Starting interactive repository initialization")

	// Assert this function is not running in automated unit tests
	if os.Getenv("GO_TEST_MODE") == "true" || os.Getenv("CI") != "" {
		return fmt.Errorf("interactive init cannot be used in automated tests or CI environments")
	}

	// Ensure we're in a git repository
	if !isGitRepo() {
		initLog.Print("Not in a git repository, initialization failed")
		return fmt.Errorf("not in a git repository")
	}
	initLog.Print("Verified git repository")

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Welcome to GitHub Agentic Workflows setup!"))
	fmt.Fprintln(os.Stderr, "")

	// Prompt for engine selection
	var selectedEngine string
	engineOptions := []struct {
		value       string
		label       string
		description string
	}{
		{string(constants.CopilotEngine), "GitHub Copilot", "GitHub Copilot CLI with agent support"},
		{string(constants.ClaudeEngine), "Claude", "Anthropic Claude Code coding agent"},
		{string(constants.CodexEngine), "Codex", "OpenAI Codex/GPT engine"},
	}

	// Use interactive prompt to select engine
	form := createEngineSelectionForm(&selectedEngine, engineOptions)
	if err := form.Run(); err != nil {
		return fmt.Errorf("engine selection failed: %w", err)
	}

	initLog.Printf("User selected engine: %s", selectedEngine)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Configuring repository for %s engine...", selectedEngine)))
	fmt.Fprintln(os.Stderr, "")

	// Initialize repository with basic settings
	if err := initializeBasicRepository(verbose); err != nil {
		return err
	}

	// Configure engine-specific settings
	copilotMcp := false
	if selectedEngine == string(constants.CopilotEngine) {
		copilotMcp = true
		initLog.Print("Copilot engine selected, enabling MCP configuration")
	}

	// Configure MCP if copilot is selected
	if copilotMcp {
		initLog.Print("Configuring GitHub Copilot Agent MCP integration")

		// Create copilot-setup-steps.yml
		if err := ensureCopilotSetupSteps(verbose); err != nil {
			initLog.Printf("Failed to create copilot-setup-steps.yml: %v", err)
			return fmt.Errorf("failed to create copilot-setup-steps.yml: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .github/workflows/copilot-setup-steps.yml"))
		}

		// Create .vscode/mcp.json
		if err := ensureMCPConfig(verbose); err != nil {
			initLog.Printf("Failed to create MCP config: %v", err)
			return fmt.Errorf("failed to create MCP config: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .vscode/mcp.json"))
		}
	}

	// Configure VSCode settings
	initLog.Print("Configuring VSCode settings")
	if err := ensureVSCodeSettings(verbose); err != nil {
		initLog.Printf("Failed to update VSCode settings: %v", err)
		return fmt.Errorf("failed to update VSCode settings: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .vscode/settings.json"))
	}

	// Check and setup secrets for the selected engine
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking required secrets for the selected engine..."))
	fmt.Fprintln(os.Stderr, "")

	if err := setupEngineSecrets(selectedEngine, verbose); err != nil {
		// Secret setup is non-fatal, just warn the user
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Secret setup encountered an issue: %v", err)))
	}

	// Display success message
	initLog.Print("Interactive repository initialization completed successfully")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	if copilotMcp {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Copilot Agent MCP integration configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, launch Copilot CLI: npx @github/copilot"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then type /agent and select agentic-workflows"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+string(constants.CLIExtensionPrefix)+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// createEngineSelectionForm creates an interactive form for engine selection
func createEngineSelectionForm(selectedEngine *string, engineOptions []struct {
	value       string
	label       string
	description string
}) *huh.Form {
	// Build options for huh.Select
	var options []huh.Option[string]
	for _, opt := range engineOptions {
		options = append(options, huh.NewOption(fmt.Sprintf("%s - %s", opt.label, opt.description), opt.value))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which AI engine would you like to use?").
				Description("Select the AI engine that will power your agentic workflows").
				Options(options...).
				Value(selectedEngine),
		),
	).WithAccessible(console.IsAccessibleMode())
}

// initializeBasicRepository sets up the basic repository structure
func initializeBasicRepository(verbose bool) error {
	// Configure .gitattributes
	initLog.Print("Configuring .gitattributes")
	if err := ensureGitAttributes(); err != nil {
		initLog.Printf("Failed to configure .gitattributes: %v", err)
		return fmt.Errorf("failed to configure .gitattributes: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
	}

	// Ensure .github/aw/logs/.gitignore exists
	initLog.Print("Ensuring .github/aw/logs/.gitignore exists")
	if err := ensureLogsGitignore(); err != nil {
		initLog.Printf("Failed to ensure logs .gitignore: %v", err)
		return fmt.Errorf("failed to ensure logs .gitignore: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .github/aw/logs/.gitignore"))
	}

	// Write copilot instructions
	initLog.Print("Writing GitHub Copilot instructions")
	if err := ensureCopilotInstructions(verbose, false); err != nil {
		initLog.Printf("Failed to write copilot instructions: %v", err)
		return fmt.Errorf("failed to write copilot instructions: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created GitHub Copilot instructions"))
	}

	// Write dispatcher agent
	initLog.Print("Writing agentic workflows dispatcher agent")
	if err := ensureAgenticWorkflowsDispatcher(verbose, false); err != nil {
		initLog.Printf("Failed to write dispatcher agent: %v", err)
		return fmt.Errorf("failed to write dispatcher agent: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created dispatcher agent"))
	}

	// Write create workflow prompt
	initLog.Print("Writing create workflow prompt")
	if err := ensureCreateWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write create workflow prompt: %v", err)
		return fmt.Errorf("failed to write create workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created create workflow prompt"))
	}

	// Write update workflow prompt
	initLog.Print("Writing update workflow prompt")
	if err := ensureUpdateWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write update workflow prompt: %v", err)
		return fmt.Errorf("failed to write update workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created update workflow prompt"))
	}

	// Write create shared agentic workflow prompt
	initLog.Print("Writing create shared agentic workflow prompt")
	if err := ensureCreateSharedAgenticWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write create shared workflow prompt: %v", err)
		return fmt.Errorf("failed to write create shared workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created shared workflow creation prompt"))
	}

	// Delete existing setup agentic workflows agent if it exists
	initLog.Print("Cleaning up setup agentic workflows agent")
	if err := deleteSetupAgenticWorkflowsAgent(verbose); err != nil {
		initLog.Printf("Failed to delete setup agentic workflows agent: %v", err)
		return fmt.Errorf("failed to delete setup agentic workflows agent: %w", err)
	}

	// Write debug workflow prompt
	initLog.Print("Writing debug workflow prompt")
	if err := ensureDebugWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write debug workflow prompt: %v", err)
		return fmt.Errorf("failed to write debug workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created debug workflow prompt"))
	}

	// Write upgrade agentic workflows prompt
	initLog.Print("Writing upgrade agentic workflows prompt")
	if err := ensureUpgradeAgenticWorkflowsPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write upgrade workflows prompt: %v", err)
		return fmt.Errorf("failed to write upgrade workflows prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created upgrade workflows prompt"))
	}

	return nil
}

// setupEngineSecrets checks for engine-specific secrets and attempts to configure them
func setupEngineSecrets(engine string, verbose bool) error {
	initLog.Printf("Setting up secrets for engine: %s", engine)

	// Get current repository
	repoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		initLog.Printf("Failed to get current repository: %v", err)
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Get required secrets for the engine
	tokensToCheck := getRecommendedTokensForEngine(engine)

	// Check environment for secrets
	var availableSecrets []string
	var missingSecrets []tokenSpec

	for _, spec := range tokensToCheck {
		// Check if secret is available in environment
		secretValue := os.Getenv(spec.Name)

		// Try alternative environment variable names
		if secretValue == "" {
			switch spec.Name {
			case "ANTHROPIC_API_KEY":
				secretValue = os.Getenv("ANTHROPIC_KEY")
			case "OPENAI_API_KEY":
				secretValue = os.Getenv("OPENAI_KEY")
			case "COPILOT_GITHUB_TOKEN":
				// Use the proper GitHub token helper
				secretValue, _ = parser.GetGitHubToken()
			}
		}

		if secretValue != "" {
			availableSecrets = append(availableSecrets, spec.Name)
		} else {
			missingSecrets = append(missingSecrets, spec)
		}
	}

	// Display found secrets
	if len(availableSecrets) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found the following secrets in your environment:"))
		for _, secretName := range availableSecrets {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ %s", secretName)))
		}
		fmt.Fprintln(os.Stderr, "")

		// Ask for confirmation before configuring secrets
		var confirmSetSecrets bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Would you like to configure these secrets as repository Actions secrets?").
					Description("This will use the gh CLI to set the secrets in your repository").
					Affirmative("Yes, configure secrets").
					Negative("No, skip").
					Value(&confirmSetSecrets),
			),
		).WithAccessible(console.IsAccessibleMode())

		if err := confirmForm.Run(); err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}

		if !confirmSetSecrets {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipped configuring secrets"))
			fmt.Fprintln(os.Stderr, "")
		} else {
			// Attempt to configure them as repository secrets
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Configuring secrets for repository Actions..."))
			fmt.Fprintln(os.Stderr, "")

			successCount := 0
			for _, secretName := range availableSecrets {
				if err := attemptSetSecret(secretName, repoSlug, verbose); err != nil {
					// Handle different types of errors gracefully
					errMsg := err.Error()
					if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "Forbidden") ||
						strings.Contains(errMsg, "permissions") || strings.Contains(errMsg, "Resource not accessible") {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("  ✗ Insufficient permissions to set %s", secretName)))
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage("    You may need to grant additional permissions to your GitHub token"))
					} else {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("  ✗ Failed to set %s: %v", secretName, err)))
					}
				} else {
					successCount++
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("  ✓ Configured %s", secretName)))
				}
			}

			if successCount > 0 {
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully configured %d secret(s) for repository Actions", successCount)))
			} else {
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No secrets were configured. You may need to set them manually."))
			}
		}
	}

	// Display missing secrets
	if len(missingSecrets) > 0 {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("The following required secrets are not available in your environment:"))

		parts := splitRepoSlug(repoSlug)
		cmdOwner := parts[0]
		cmdRepo := parts[1]

		for _, spec := range missingSecrets {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✗ %s", spec.Name)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("    When needed: %s", spec.When)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("    Description: %s", spec.Description)))
			fmt.Fprintln(os.Stderr, console.FormatCommandMessage(fmt.Sprintf("    gh aw secrets set %s --owner %s --repo %s", spec.Name, cmdOwner, cmdRepo)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	return nil
}

// attemptSetSecret attempts to set a secret for the repository
func attemptSetSecret(secretName, repoSlug string, verbose bool) error {
	initLog.Printf("Attempting to set secret: %s for repo: %s", secretName, repoSlug)

	// Check if secret already exists
	exists, err := checkSecretExistsInRepo(secretName, repoSlug)
	if err != nil {
		// If we get a permission error, return it immediately
		if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
			return fmt.Errorf("insufficient permissions to access repository secrets: %w", err)
		}
		// For other errors, log but try to set anyway
		if verbose {
			initLog.Printf("Could not check if secret exists: %v", err)
		}
	} else if exists {
		// Secret already exists, skip
		if verbose {
			initLog.Printf("Secret %s already exists, skipping", secretName)
		}
		return nil
	}

	// Get secret value from environment
	secretValue := os.Getenv(secretName)
	if secretValue == "" {
		// Try alternative names
		switch secretName {
		case "ANTHROPIC_API_KEY":
			secretValue = os.Getenv("ANTHROPIC_KEY")
		case "OPENAI_API_KEY":
			secretValue = os.Getenv("OPENAI_KEY")
		case "COPILOT_GITHUB_TOKEN":
			secretValue, err = parser.GetGitHubToken()
			if err != nil {
				return fmt.Errorf("failed to get GitHub token: %w", err)
			}
		}
	}

	if secretValue == "" {
		return fmt.Errorf("secret value not found in environment")
	}

	// Set the secret using gh CLI
	cmd := workflow.ExecGH("secret", "set", secretName, "--repo", repoSlug, "--body", secretValue)
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		// Check for permission-related errors
		if strings.Contains(outputStr, "403") || strings.Contains(outputStr, "Forbidden") ||
			strings.Contains(outputStr, "Resource not accessible") || strings.Contains(err.Error(), "403") {
			return fmt.Errorf("insufficient permissions to set secrets in repository: %w", err)
		}
		return fmt.Errorf("failed to set secret: %w (output: %s)", err, outputStr)
	}

	initLog.Printf("Successfully set secret: %s", secretName)
	return nil
}

// InitRepository initializes the repository for agentic workflows
func InitRepository(verbose bool, mcp bool, campaign bool, tokens bool, engine string, codespaceRepos []string, codespaceEnabled bool, completions bool, push bool, rootCmd CommandProvider) error {
	initLog.Print("Starting repository initialization for agentic workflows")

	// If --push is enabled, ensure git status is clean before starting
	if push {
		initLog.Print("Checking for clean working directory (--push enabled)")
		if err := checkCleanWorkingDirectory(verbose); err != nil {
			initLog.Printf("Git status check failed: %v", err)
			return fmt.Errorf("--push requires a clean working directory: %w", err)
		}
	}

	// Ensure we're in a git repository
	if !isGitRepo() {
		initLog.Print("Not in a git repository, initialization failed")
		return fmt.Errorf("not in a git repository")
	}
	initLog.Print("Verified git repository")

	// Configure .gitattributes
	initLog.Print("Configuring .gitattributes")
	if err := ensureGitAttributes(); err != nil {
		initLog.Printf("Failed to configure .gitattributes: %v", err)
		return fmt.Errorf("failed to configure .gitattributes: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
	}

	// Ensure .github/aw/logs/.gitignore exists
	initLog.Print("Ensuring .github/aw/logs/.gitignore exists")
	if err := ensureLogsGitignore(); err != nil {
		initLog.Printf("Failed to ensure logs .gitignore: %v", err)
		return fmt.Errorf("failed to ensure logs .gitignore: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .github/aw/logs/.gitignore"))
	}

	// Write copilot instructions
	initLog.Print("Writing GitHub Copilot instructions")
	if err := ensureCopilotInstructions(verbose, false); err != nil {
		initLog.Printf("Failed to write copilot instructions: %v", err)
		return fmt.Errorf("failed to write copilot instructions: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created GitHub Copilot instructions"))
	}

	// Write dispatcher agent
	initLog.Print("Writing agentic workflows dispatcher agent")
	if err := ensureAgenticWorkflowsDispatcher(verbose, false); err != nil {
		initLog.Printf("Failed to write dispatcher agent: %v", err)
		return fmt.Errorf("failed to write dispatcher agent: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created dispatcher agent"))
	}

	// Write create workflow prompt
	initLog.Print("Writing create workflow prompt")
	if err := ensureCreateWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write create workflow prompt: %v", err)
		return fmt.Errorf("failed to write create workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created create workflow prompt"))
	}

	// Write update workflow prompt (new)
	initLog.Print("Writing update workflow prompt")
	if err := ensureUpdateWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write update workflow prompt: %v", err)
		return fmt.Errorf("failed to write update workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created update workflow prompt"))
	}

	// Write create shared agentic workflow prompt
	initLog.Print("Writing create shared agentic workflow prompt")
	if err := ensureCreateSharedAgenticWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write create shared workflow prompt: %v", err)
		return fmt.Errorf("failed to write create shared workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created shared workflow creation prompt"))
	}

	// Delete existing setup agentic workflows agent if it exists
	initLog.Print("Cleaning up setup agentic workflows agent")
	if err := deleteSetupAgenticWorkflowsAgent(verbose); err != nil {
		initLog.Printf("Failed to delete setup agentic workflows agent: %v", err)
		return fmt.Errorf("failed to delete setup agentic workflows agent: %w", err)
	}

	// Write debug workflow prompt
	initLog.Print("Writing debug workflow prompt")
	if err := ensureDebugWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write debug workflow prompt: %v", err)
		return fmt.Errorf("failed to write debug workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created debug workflow prompt"))
	}

	// Write upgrade agentic workflows prompt
	initLog.Print("Writing upgrade agentic workflows prompt")
	if err := ensureUpgradeAgenticWorkflowsPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write upgrade workflows prompt: %v", err)
		return fmt.Errorf("failed to write upgrade workflows prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created upgrade workflows prompt"))
	}

	// Write campaign dispatcher agent if requested
	if campaign {
		// Write campaign instruction files
		initLog.Print("Writing campaign instruction files")
		campaignEnsureFuncs := []struct {
			fn   func(bool, bool) error
			name string
		}{
			{ensureCampaignGeneratorInstructions, "campaign generator instructions"},
			{ensureCampaignOrchestratorInstructions, "campaign orchestrator instructions"},
			{ensureCampaignProjectUpdateInstructions, "campaign project update instructions"},
			{ensureCampaignWorkflowExecution, "campaign workflow execution"},
			{ensureCampaignClosingInstructions, "campaign closing instructions"},
		}

		for _, item := range campaignEnsureFuncs {
			if err := item.fn(verbose, false); err != nil {
				initLog.Printf("Failed to write %s: %v", item.name, err)
				return fmt.Errorf("failed to write %s: %w", item.name, err)
			}
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created campaign instruction files"))
		}

		// Add campaign-generator workflow from gh-aw repository
		initLog.Print("Adding campaign-generator workflow")
		if err := addCampaignGeneratorWorkflow(verbose); err != nil {
			initLog.Printf("Failed to add campaign-generator workflow: %v", err)
			return fmt.Errorf("failed to add campaign-generator workflow: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added campaign-generator workflow"))
		}

		// Create the 'create-agentic-campaign' label
		initLog.Print("Creating 'create-agentic-campaign' label")
		if err := createCampaignLabel(verbose); err != nil {
			// Label creation is non-fatal, just log the error
			initLog.Printf("Label creation encountered an issue: %v", err)
		}
	}

	// Configure MCP if requested
	if mcp {
		initLog.Print("Configuring GitHub Copilot Agent MCP integration")

		// Create copilot-setup-steps.yml
		if err := ensureCopilotSetupSteps(verbose); err != nil {
			initLog.Printf("Failed to create copilot-setup-steps.yml: %v", err)
			return fmt.Errorf("failed to create copilot-setup-steps.yml: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .github/workflows/copilot-setup-steps.yml"))
		}

		// Create .vscode/mcp.json
		if err := ensureMCPConfig(verbose); err != nil {
			initLog.Printf("Failed to create MCP config: %v", err)
			return fmt.Errorf("failed to create MCP config: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .vscode/mcp.json"))
		}
	}

	// Configure Codespaces if requested
	if codespaceEnabled {
		initLog.Printf("Configuring GitHub Codespaces devcontainer with additional repos: %v", codespaceRepos)

		// Create or update .devcontainer/devcontainer.json
		if err := ensureDevcontainerConfig(verbose, codespaceRepos); err != nil {
			initLog.Printf("Failed to configure devcontainer: %v", err)
			return fmt.Errorf("failed to configure devcontainer: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .devcontainer/devcontainer.json"))
		}
	}

	// Configure VSCode settings
	initLog.Print("Configuring VSCode settings")

	// Update .vscode/settings.json
	if err := ensureVSCodeSettings(verbose); err != nil {
		initLog.Printf("Failed to update VSCode settings: %v", err)
		return fmt.Errorf("failed to update VSCode settings: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .vscode/settings.json"))
	}

	// Validate tokens if requested
	if tokens {
		initLog.Print("Validating repository secrets for agentic workflows")
		fmt.Fprintln(os.Stderr, "")

		// Run token bootstrap validation
		if err := runTokensBootstrap(engine, "", ""); err != nil {
			initLog.Printf("Token validation failed: %v", err)
			// Don't fail init if token validation has issues
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Token validation encountered an issue: %v", err)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Install shell completions if requested
	if completions {
		initLog.Print("Installing shell completions")
		fmt.Fprintln(os.Stderr, "")

		if err := InstallShellCompletion(verbose, rootCmd); err != nil {
			initLog.Printf("Shell completion installation failed: %v", err)
			// Don't fail init if completion installation has issues
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Shell completion installation encountered an issue: %v", err)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Generate/update maintenance workflow if any workflows use expires field
	initLog.Print("Checking for workflows with expires field to generate maintenance workflow")
	if err := ensureMaintenanceWorkflow(verbose); err != nil {
		initLog.Printf("Failed to generate maintenance workflow: %v", err)
		// Don't fail init if maintenance workflow generation has issues
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate maintenance workflow: %v", err)))
	}

	initLog.Print("Repository initialization completed successfully")

	// If --push is enabled, commit and push changes
	if push {
		initLog.Print("Push enabled - preparing to commit and push changes")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Preparing to commit and push changes..."))

		// Use the helper function to orchestrate the full workflow
		commitMessage := "chore: initialize agentic workflows"
		if err := commitAndPushChanges(commitMessage, verbose); err != nil {
			// Check if it's the "no changes" case
			hasChanges, checkErr := hasChangesToCommit()
			if checkErr == nil && !hasChanges {
				initLog.Print("No changes to commit")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes to commit"))
			} else {
				return err
			}
		} else {
			// Print success messages based on whether remote exists
			fmt.Fprintln(os.Stderr, "")
			if hasRemote() {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes pushed to remote"))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes committed locally (no remote configured)"))
			}
		}
	}

	// Display success message with next steps
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	if mcp {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Copilot Agent MCP integration configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	if len(codespaceRepos) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Codespaces devcontainer configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	if tokens {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To configure missing secrets, use: gh aw secret set <secret-name> --owner <owner> --repo <repo>"))
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, launch Copilot CLI: npx @github/copilot"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then type /agent and select agentic-workflows"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+string(constants.CLIExtensionPrefix)+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// addCampaignGeneratorWorkflow generates and compiles the agentic-campaign-generator workflow
func addCampaignGeneratorWorkflow(verbose bool) error {
	initLog.Print("Generating agentic-campaign-generator workflow")

	// Get the git root directory
	gitRoot, err := findGitRoot()
	if err != nil {
		initLog.Printf("Failed to find git root: %v", err)
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Keep the campaign generator source next to its lock file for consistency.
	workflowsDir := filepath.Join(gitRoot, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		initLog.Printf("Failed to create workflows directory: %v", err)
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	// Build the agentic-campaign-generator workflow
	data := campaign.BuildCampaignGenerator()
	workflowPath := filepath.Join(workflowsDir, "agentic-campaign-generator.md")

	// Render the workflow to markdown
	content := renderCampaignGeneratorMarkdown(data)

	// Write markdown file with restrictive permissions
	if err := os.WriteFile(workflowPath, []byte(content), 0600); err != nil {
		initLog.Printf("Failed to write agentic-campaign-generator.md: %v", err)
		return fmt.Errorf("failed to write agentic-campaign-generator.md: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Created agentic-campaign-generator workflow: %s\n", workflowPath)
	}

	// Compile to lock file using the standard compiler.
	compiler := workflow.NewCompiler(verbose, "", GetVersion())
	if err := CompileWorkflowWithValidation(compiler, workflowPath, verbose, false, false, false, false, false); err != nil {
		initLog.Printf("Failed to compile agentic-campaign-generator: %v", err)
		return fmt.Errorf("failed to compile agentic-campaign-generator: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Compiled agentic-campaign-generator workflow\n")
	}

	initLog.Print("Agentic-campaign-generator workflow generated successfully")
	return nil
}

// createCampaignLabel creates the 'create-agentic-campaign' label in the repository
func createCampaignLabel(verbose bool) error {
	initLog.Print("Creating 'create-agentic-campaign' label")

	// Get the current repository
	repo, err := getCurrentRepositoryForInit()
	if err != nil {
		initLog.Printf("Could not determine repository: %v", err)
		// Don't fail if we can't determine the repository
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not determine repository for label creation: %v", err)))
		}
		return nil
	}

	initLog.Printf("Creating label for repository: %s", repo)

	// Split repo into owner and name
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		initLog.Printf("Invalid repository format: %s", repo)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Invalid repository format: %s", repo)))
		}
		return nil
	}

	// Create the label using gh api
	// See https://docs.github.com/en/rest/issues/labels?apiVersion=2022-11-28#create-a-label
	cmd := workflow.ExecGH("api",
		fmt.Sprintf("repos/%s/labels", repo),
		"-X", "POST",
		"-f", "name=create-agentic-campaign",
		"-f", "color=0E8A16",
		"-f", "description=Create a new agentic campaign")

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Check if the error is because the label already exists
		if strings.Contains(outputStr, "already_exists") {
			initLog.Print("Label 'create-agentic-campaign' already exists")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Label 'create-agentic-campaign' already exists"))
			}
			return nil
		}

		// For other errors, log but don't fail the init
		initLog.Printf("Failed to create label (non-fatal): %v (output: %s)", err, outputStr)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create label 'create-agentic-campaign': %v", err)))
		}
		return nil
	}

	initLog.Print("Successfully created label 'create-agentic-campaign'")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created label 'create-agentic-campaign'"))
	}

	return nil
}

// getCurrentRepositoryForInit gets the current repository for init command
func getCurrentRepositoryForInit() (string, error) {
	initLog.Print("Getting current repository for init")

	// Use the same approach as repository_features_validation.go
	// Try to get the repository using gh CLI
	cmd := workflow.ExecGH("repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current repository: %w", err)
	}

	repo := strings.TrimSpace(string(output))
	if repo == "" {
		return "", fmt.Errorf("repository name is empty")
	}

	initLog.Printf("Current repository: %s", repo)
	return repo, nil
}

// renderCampaignGeneratorMarkdown converts WorkflowData to markdown format for campaign-generator
func renderCampaignGeneratorMarkdown(data *workflow.WorkflowData) string {
	var b strings.Builder

	b.WriteString("---\n")
	if strings.TrimSpace(data.Name) != "" {
		fmt.Fprintf(&b, "name: %q\n", data.Name)
	}
	if strings.TrimSpace(data.Description) != "" {
		fmt.Fprintf(&b, "description: %q\n", data.Description)
	}
	if strings.TrimSpace(data.On) != "" {
		b.WriteString(strings.TrimSuffix(data.On, "\n"))
		b.WriteString("\n")
	}
	if strings.TrimSpace(data.Permissions) != "" {
		b.WriteString(strings.TrimSuffix(data.Permissions, "\n"))
		b.WriteString("\n")
	}

	// Engine configuration
	engineID := "copilot"
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	}
	fmt.Fprintf(&b, "engine: %s\n", engineID)

	// Tools
	if len(data.Tools) > 0 {
		b.WriteString("tools:\n")
		if gh, ok := data.Tools["github"].(map[string]any); ok {
			b.WriteString("  github:\n")
			if toolsets, ok := gh["toolsets"].([]any); ok {
				b.WriteString("    toolsets: [")
				for i, ts := range toolsets {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%v", ts)
				}
				b.WriteString("]\n")
			}
		}
	}

	// Safe outputs
	if data.SafeOutputs != nil {
		b.WriteString("safe-outputs:\n")

		if data.SafeOutputs.AddComments != nil {
			b.WriteString("  add-comment:\n")
			b.WriteString("    max: 10\n")
		}

		if data.SafeOutputs.UpdateIssues != nil {
			b.WriteString("  update-issue:\n")
		}

		if data.SafeOutputs.AssignToAgent != nil {
			b.WriteString("  assign-to-agent:\n")
		}

		if data.SafeOutputs.CreateProjects != nil {
			b.WriteString("  create-project:\n")
			b.WriteString("    max: 1\n")
			if data.SafeOutputs.CreateProjects.GitHubToken != "" {
				fmt.Fprintf(&b, "    github-token: \"%s\"\n", data.SafeOutputs.CreateProjects.GitHubToken)
			}
			if data.SafeOutputs.CreateProjects.TargetOwner != "" {
				fmt.Fprintf(&b, "    target-owner: \"%s\"\n", data.SafeOutputs.CreateProjects.TargetOwner)
			}
			if len(data.SafeOutputs.CreateProjects.Views) > 0 {
				b.WriteString("    views:\n")
				for _, view := range data.SafeOutputs.CreateProjects.Views {
					fmt.Fprintf(&b, "      - name: \"%s\"\n", view.Name)
					fmt.Fprintf(&b, "        layout: \"%s\"\n", view.Layout)
					fmt.Fprintf(&b, "        filter: \"%s\"\n", view.Filter)
				}
			}
		}

		if data.SafeOutputs.UpdateProjects != nil {
			b.WriteString("  update-project:\n")
			b.WriteString("    max: 10\n")
			if data.SafeOutputs.UpdateProjects.GitHubToken != "" {
				fmt.Fprintf(&b, "    github-token: \"%s\"\n", data.SafeOutputs.UpdateProjects.GitHubToken)
			}
			if len(data.SafeOutputs.UpdateProjects.Views) > 0 {
				b.WriteString("    views:\n")
				for _, view := range data.SafeOutputs.UpdateProjects.Views {
					fmt.Fprintf(&b, "      - name: \"%s\"\n", view.Name)
					fmt.Fprintf(&b, "        layout: \"%s\"\n", view.Layout)
					if strings.TrimSpace(view.Filter) != "" {
						fmt.Fprintf(&b, "        filter: \"%s\"\n", view.Filter)
					}
					if strings.TrimSpace(view.Description) != "" {
						fmt.Fprintf(&b, "        description: \"%s\"\n", view.Description)
					}
					if len(view.VisibleFields) > 0 {
						b.WriteString("        visible-fields:\n")
						for _, fieldIndex := range view.VisibleFields {
							fmt.Fprintf(&b, "          - %d\n", fieldIndex)
						}
					}
				}
			}
			if len(data.SafeOutputs.UpdateProjects.FieldDefinitions) > 0 {
				b.WriteString("    field-definitions:\n")
				for _, field := range data.SafeOutputs.UpdateProjects.FieldDefinitions {
					fmt.Fprintf(&b, "      - name: \"%s\"\n", field.Name)
					fmt.Fprintf(&b, "        data-type: \"%s\"\n", field.DataType)
					if len(field.Options) > 0 {
						b.WriteString("        options:\n")
						for _, opt := range field.Options {
							fmt.Fprintf(&b, "          - \"%s\"\n", opt)
						}
					}
				}
			}
		}

		if data.SafeOutputs.Messages != nil {
			b.WriteString("  messages:\n")
			if data.SafeOutputs.Messages.Footer != "" {
				fmt.Fprintf(&b, "    footer: \"%s\"\n", data.SafeOutputs.Messages.Footer)
			}
			if data.SafeOutputs.Messages.RunStarted != "" {
				fmt.Fprintf(&b, "    run-started: \"%s\"\n", data.SafeOutputs.Messages.RunStarted)
			}
			if data.SafeOutputs.Messages.RunSuccess != "" {
				fmt.Fprintf(&b, "    run-success: \"%s\"\n", data.SafeOutputs.Messages.RunSuccess)
			}
			if data.SafeOutputs.Messages.RunFailure != "" {
				fmt.Fprintf(&b, "    run-failure: \"%s\"\n", data.SafeOutputs.Messages.RunFailure)
			}
		}
	}

	if strings.TrimSpace(data.TimeoutMinutes) != "" {
		fmt.Fprintf(&b, "timeout-minutes: %s\n", data.TimeoutMinutes)
	}

	b.WriteString("---\n\n")

	// Write the prompt/body
	if strings.TrimSpace(data.MarkdownContent) != "" {
		b.WriteString(data.MarkdownContent)
	}

	return b.String()
}

// ensureMaintenanceWorkflow checks existing workflows for expires field and generates/updates
// the maintenance workflow file if any workflows use it
func ensureMaintenanceWorkflow(verbose bool) error {
	initLog.Print("Checking for workflows with expires field")

	// Find git root
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Determine the workflows directory
	workflowsDir := filepath.Join(gitRoot, ".github", "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		// No workflows directory yet, skip maintenance workflow generation
		initLog.Print("No workflows directory found, skipping maintenance workflow generation")
		return nil
	}

	// Find all workflow markdown files
	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to find workflow files: %w", err)
	}

	// Filter out README.md files
	files = filterWorkflowFiles(files)

	// Create a compiler to parse workflows
	compiler := workflow.NewCompiler(false, "", GetVersion())

	// Detect and set the action mode (dev/release) based on binary version and GitHub context
	// This ensures the maintenance workflow uses the correct action references
	mode := workflow.DetectActionMode(GetVersion())
	compiler.SetActionMode(mode)
	initLog.Printf("Action mode detected for maintenance workflow: %s", mode)

	// Parse all workflows to collect WorkflowData
	var workflowDataList []*workflow.WorkflowData
	for _, file := range files {
		// Skip campaign specs and generated files
		if strings.HasSuffix(file, ".campaign.md") || strings.HasSuffix(file, ".campaign.g.md") {
			continue
		}

		initLog.Printf("Parsing workflow: %s", file)
		workflowData, err := compiler.ParseWorkflowFile(file)
		if err != nil {
			// Ignore parse errors - workflows might be incomplete during init
			initLog.Printf("Skipping workflow %s due to parse error: %v", file, err)
			continue
		}

		workflowDataList = append(workflowDataList, workflowData)
	}

	// Always call GenerateMaintenanceWorkflow even with empty list
	// This allows it to delete existing maintenance workflow if no workflows have expires
	initLog.Printf("Generating maintenance workflow for %d workflows", len(workflowDataList))
	if err := workflow.GenerateMaintenanceWorkflow(workflowDataList, workflowsDir, GetVersion(), compiler.GetActionMode(), verbose); err != nil {
		return fmt.Errorf("failed to generate maintenance workflow: %w", err)
	}

	if verbose && len(workflowDataList) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Generated/updated maintenance workflow"))
	}

	return nil
}
