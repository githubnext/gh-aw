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
func InitRepository(verbose bool, mcp bool) error {
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

	// Write agentic workflow agent
	initLog.Print("Writing agentic workflow agent")
	if err := ensureAgenticWorkflowAgent(verbose, false); err != nil {
		initLog.Printf("Failed to write agentic workflow agent: %v", err)
		return fmt.Errorf("failed to write agentic workflow agent: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created agent for workflow creation"))
	}

	// Write shared agentic workflow agent
	initLog.Print("Writing shared agentic workflow agent")
	if err := ensureSharedAgenticWorkflowAgent(verbose, false); err != nil {
		initLog.Printf("Failed to write shared agentic workflow agent: %v", err)
		return fmt.Errorf("failed to write shared agentic workflow agent: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created /create-shared-agentic-workflow agent"))
	}

	// Delete existing setup agentic workflows agent if it exists
	initLog.Print("Cleaning up setup agentic workflows agent")
	if err := deleteSetupAgenticWorkflowsAgent(verbose); err != nil {
		initLog.Printf("Failed to delete setup agentic workflows agent: %v", err)
		return fmt.Errorf("failed to delete setup agentic workflows agent: %w", err)
	}

	// Write debug agentic workflow agent
	initLog.Print("Writing debug agentic workflow agent")
	if err := ensureDebugAgenticWorkflowAgent(verbose, false); err != nil {
		initLog.Printf("Failed to write debug agentic workflow agent: %v", err)
		return fmt.Errorf("failed to write debug agentic workflow agent: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created debug agentic workflow agent"))
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

	initLog.Print("Repository initialization completed successfully")

	// Display success message with next steps
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	if mcp {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Copilot Agent MCP integration configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, launch Copilot CLI: npx @github/copilot"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then type /agent and select create-agentic-workflow"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+constants.CLIExtensionPrefix+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}
