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
func InitRepository(verbose bool) error {
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

	// Write copilot instructions
	initLog.Print("Writing GitHub Copilot instructions")
	if err := ensureCopilotInstructions(verbose, false); err != nil {
		initLog.Printf("Failed to write copilot instructions: %v", err)
		return fmt.Errorf("failed to write copilot instructions: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created GitHub Copilot instructions"))
	}

	// Write agentic workflow prompt
	initLog.Print("Writing agentic workflow prompt")
	if err := ensureAgenticWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write agentic workflow prompt: %v", err)
		return fmt.Errorf("failed to write agentic workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created /create-agentic-workflow command"))
	}

	// Write shared agentic workflow prompt
	initLog.Print("Writing shared agentic workflow prompt")
	if err := ensureSharedAgenticWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write shared agentic workflow prompt: %v", err)
		return fmt.Errorf("failed to write shared agentic workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created /create-shared-agentic-workflow command"))
	}

	// Write getting started prompt
	initLog.Print("Writing getting started prompt")
	if err := ensureGettingStartedPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write getting started prompt: %v", err)
		return fmt.Errorf("failed to write getting started prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created getting started guide"))
	}

	initLog.Print("Repository initialization completed successfully")

	// Display success message with next steps
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Start a chat and copy the following prompt to create a new workflow:"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "     activate @.github/prompts/create-agentic-workflow.prompt.md")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+constants.CLIExtensionPrefix+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}
