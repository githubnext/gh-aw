package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var initLog = logger.New("cli:init")

// InitRepository initializes the repository for agentic workflows
func InitRepository(verbose bool, mcp bool, campaign bool, tokens bool, engine string, codespaceRepos []string, codespaceEnabled bool, completions bool, rootCmd CommandProvider) error {
	initLog.Print("Starting repository initialization for agentic workflows")

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

	// Write create agentic workflow prompt (legacy)
	initLog.Print("Writing create agentic workflow prompt (legacy)")
	if err := ensureCreateAgenticWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write create workflow prompt: %v", err)
		return fmt.Errorf("failed to write create workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created workflow creation prompt (legacy)"))
	}

	// Write create workflow prompt (new)
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

	// Write debug agentic workflow prompt
	initLog.Print("Writing debug agentic workflow prompt")
	if err := ensureDebugAgenticWorkflowPrompt(verbose, false); err != nil {
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
		initLog.Print("Writing campaign dispatcher agent")
		if err := ensureAgenticCampaignsDispatcher(verbose, false); err != nil {
			initLog.Printf("Failed to write campaign dispatcher agent: %v", err)
			return fmt.Errorf("failed to write campaign dispatcher agent: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created campaign dispatcher agent"))
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

	initLog.Print("Repository initialization completed successfully")

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
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then type /agent and select create-agentic-workflow"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+string(constants.CLIExtensionPrefix)+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}
