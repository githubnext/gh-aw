package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
)

// InitRepository initializes the repository for agentic workflows
func InitRepository(verbose bool) error {
	// Ensure we're in a git repository
	if !isGitRepo() {
		return fmt.Errorf("not in a git repository")
	}

	// Configure .gitattributes
	if err := ensureGitAttributes(); err != nil {
		return fmt.Errorf("failed to configure .gitattributes: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
	}

	// Write copilot instructions
	if err := ensureCopilotInstructions(verbose, false); err != nil {
		return fmt.Errorf("failed to write copilot instructions: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created GitHub Copilot instructions"))
	}

	// Write agentic workflow prompt
	if err := ensureAgenticWorkflowPrompt(verbose, false); err != nil {
		return fmt.Errorf("failed to write agentic workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created /create-agentic-workflow command"))
	}

	// Write shared agentic workflow prompt
	if err := ensureSharedAgenticWorkflowPrompt(verbose, false); err != nil {
		return fmt.Errorf("failed to write shared agentic workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created /create-shared-agentic-workflow command"))
	}

	// Display success message with next steps
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("See .github/instructions for Copilot guidance or run "+constants.CLIExtensionPrefix+" add to get started."))
	fmt.Fprintln(os.Stderr, "")

	return nil
}
