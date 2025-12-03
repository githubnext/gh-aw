package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureFileMatchesTemplate ensures a file in a subdirectory matches the expected template content
func ensureFileMatchesTemplate(subdir, fileName, templateContent, fileType string, verbose bool, skipInstructions bool) error {
	if skipInstructions {
		return nil
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	targetDir := filepath.Join(gitRoot, subdir)
	targetPath := filepath.Join(targetDir, fileName)

	// Ensure the target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", subdir, err)
	}

	// Check if the file already exists and matches the template
	existingContent := ""
	if content, err := os.ReadFile(targetPath); err == nil {
		existingContent = string(content)
	}

	// Check if content matches our expected template
	expectedContent := strings.TrimSpace(templateContent)
	if strings.TrimSpace(existingContent) == expectedContent {
		if verbose {
			fmt.Printf("%s is up-to-date: %s\n", fileType, targetPath)
		}
		return nil
	}

	// Write the file
	if err := os.WriteFile(targetPath, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", fileType, err)
	}

	if verbose {
		if existingContent == "" {
			fmt.Printf("Created %s: %s\n", fileType, targetPath)
		} else {
			fmt.Printf("Updated %s: %s\n", fileType, targetPath)
		}
	}

	return nil
}

// ensurePromptFromTemplate ensures that a prompt file exists and matches the embedded template
func ensurePromptFromTemplate(promptFileName, templateContent string, verbose bool, skipInstructions bool) error {
	return ensureFileMatchesTemplate(
		filepath.Join(".github", "prompts"),
		promptFileName,
		templateContent,
		"prompt",
		verbose,
		skipInstructions,
	)
}

// cleanupOldAgentFile removes an old agent file if it exists
func cleanupOldAgentFile(agentFileName string, verbose bool) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	oldPath := filepath.Join(gitRoot, ".github", "agents", agentFileName)

	// Check if the old file exists and remove it
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("failed to remove old agent file: %w", err)
		}
		if verbose {
			fmt.Printf("Removed old agent file: %s\n", oldPath)
		}
	}

	return nil
}

// ensureCopilotInstructions ensures that .github/aw/github-agentic-workflows.md contains the copilot instructions
func ensureCopilotInstructions(verbose bool, skipInstructions bool) error {
	// First, clean up the old file location if it exists
	if err := cleanupOldCopilotInstructions(verbose); err != nil {
		return err
	}

	return ensureFileMatchesTemplate(
		filepath.Join(".github", "aw"),
		"github-agentic-workflows.md",
		copilotInstructionsTemplate,
		"copilot instructions",
		verbose,
		skipInstructions,
	)
}

// cleanupOldCopilotInstructions removes the old instructions file from .github/instructions/
func cleanupOldCopilotInstructions(verbose bool) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	oldPath := filepath.Join(gitRoot, ".github", "instructions", "github-agentic-workflows.instructions.md")

	// Check if the old file exists and remove it
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("failed to remove old instructions file: %w", err)
		}
		if verbose {
			fmt.Printf("Removed old instructions file: %s\n", oldPath)
		}
	}

	return nil
}

// ensureAgenticWorkflowPrompt ensures that .github/prompts/create-agentic-workflow.prompt.md contains the workflow creation prompt
func ensureAgenticWorkflowPrompt(verbose bool, skipInstructions bool) error {
	// First, clean up the old agent file if it exists
	if err := cleanupOldAgentFile("create-agentic-workflow.md", verbose); err != nil {
		return err
	}

	return ensurePromptFromTemplate("create-agentic-workflow.prompt.md", agenticWorkflowPromptTemplate, verbose, skipInstructions)
}

// ensureSharedAgenticWorkflowPrompt ensures that .github/prompts/create-shared-agentic-workflow.prompt.md contains the shared workflow creation prompt
func ensureSharedAgenticWorkflowPrompt(verbose bool, skipInstructions bool) error {
	// First, clean up the old agent file if it exists
	if err := cleanupOldAgentFile("create-shared-agentic-workflow.md", verbose); err != nil {
		return err
	}

	return ensurePromptFromTemplate("create-shared-agentic-workflow.prompt.md", sharedAgenticWorkflowPromptTemplate, verbose, skipInstructions)
}

// ensureSetupAgenticWorkflowsPrompt ensures that .github/prompts/setup-agentic-workflows.prompt.md contains the setup guide prompt
func ensureSetupAgenticWorkflowsPrompt(verbose bool, skipInstructions bool) error {
	// First, clean up the old agent file if it exists
	if err := cleanupOldAgentFile("setup-agentic-workflows.md", verbose); err != nil {
		return err
	}

	return ensurePromptFromTemplate("setup-agentic-workflows.prompt.md", setupAgenticWorkflowsPromptTemplate, verbose, skipInstructions)
}

// ensureDebugAgenticWorkflowPrompt ensures that .github/prompts/debug-agentic-workflow.prompt.md contains the debug workflow prompt
func ensureDebugAgenticWorkflowPrompt(verbose bool, skipInstructions bool) error {
	// First, clean up the old agent file if it exists
	if err := cleanupOldAgentFile("debug-agentic-workflow.md", verbose); err != nil {
		return err
	}

	return ensurePromptFromTemplate("debug-agentic-workflow.prompt.md", debugAgenticWorkflowPromptTemplate, verbose, skipInstructions)
}
