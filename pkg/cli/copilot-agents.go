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

// deleteOldAgentFile deletes the corresponding old agent file from .github/agents/
func deleteOldAgentFile(promptFileName string, verbose bool) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	// Convert prompt filename to agent filename (remove .prompt suffix)
	agentFileName := strings.Replace(promptFileName, ".prompt.md", ".md", 1)
	oldAgentPath := filepath.Join(gitRoot, ".github", "agents", agentFileName)

	// Check if the old agent file exists and remove it
	if _, err := os.Stat(oldAgentPath); err == nil {
		if err := os.Remove(oldAgentPath); err != nil {
			return fmt.Errorf("failed to remove old agent file: %w", err)
		}
		if verbose {
			fmt.Printf("Removed old agent file: %s\n", oldAgentPath)
		}
	}

	return nil
}

// ensureCopilotInstructions ensures that .github/instructions/github-agentic-workflows.md contains the copilot instructions
func ensureCopilotInstructions(verbose bool, skipInstructions bool) error {
	return ensureFileMatchesTemplate(
		filepath.Join(".github", "instructions"),
		"github-agentic-workflows.instructions.md",
		copilotInstructionsTemplate,
		"copilot instructions",
		verbose,
		skipInstructions,
	)
}

// ensureAgenticWorkflowPrompt removes the old agentic workflow prompt file if it exists
func ensureAgenticWorkflowPrompt(verbose bool, skipInstructions bool) error {
	// This function now removes the old prompt file since we've migrated to agent format
	if skipInstructions {
		return nil
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	promptsDir := filepath.Join(gitRoot, ".github", "prompts")
	oldPromptPath := filepath.Join(promptsDir, "create-agentic-workflow.prompt.md")

	// Check if the old prompt file exists and remove it
	if _, err := os.Stat(oldPromptPath); err == nil {
		if err := os.Remove(oldPromptPath); err != nil {
			return fmt.Errorf("failed to remove old prompt file: %w", err)
		}
		if verbose {
			fmt.Printf("Removed old prompt file: %s\n", oldPromptPath)
		}
	}

	return nil
}

// ensureAgenticWorkflowAgent ensures that .github/prompts/create-agentic-workflow.prompt.md contains the agentic workflow creation prompt
func ensureAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	if err := ensurePromptFromTemplate("create-agentic-workflow.prompt.md", agenticWorkflowAgentTemplate, verbose, skipInstructions); err != nil {
		return err
	}
	return deleteOldAgentFile("create-agentic-workflow.prompt.md", verbose)
}

// ensureSharedAgenticWorkflowAgent ensures that .github/prompts/create-shared-agentic-workflow.prompt.md contains the shared workflow creation prompt
func ensureSharedAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	if err := ensurePromptFromTemplate("create-shared-agentic-workflow.prompt.md", sharedAgenticWorkflowAgentTemplate, verbose, skipInstructions); err != nil {
		return err
	}
	return deleteOldAgentFile("create-shared-agentic-workflow.prompt.md", verbose)
}

// ensureSetupAgenticWorkflowsAgent ensures that .github/prompts/setup-agentic-workflows.prompt.md contains the setup guide prompt
func ensureSetupAgenticWorkflowsAgent(verbose bool, skipInstructions bool) error {
	if err := ensurePromptFromTemplate("setup-agentic-workflows.prompt.md", setupAgenticWorkflowsAgentTemplate, verbose, skipInstructions); err != nil {
		return err
	}
	return deleteOldAgentFile("setup-agentic-workflows.prompt.md", verbose)
}

// ensureDebugAgenticWorkflowAgent ensures that .github/prompts/debug-agentic-workflow.prompt.md contains the debug workflow prompt
func ensureDebugAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	if err := ensurePromptFromTemplate("debug-agentic-workflow.prompt.md", debugAgenticWorkflowAgentTemplate, verbose, skipInstructions); err != nil {
		return err
	}
	return deleteOldAgentFile("debug-agentic-workflow.prompt.md", verbose)
}
