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

// ensureAgentFromTemplate ensures that an agent file exists and matches the embedded template
func ensureAgentFromTemplate(agentFileName, templateContent string, verbose bool, skipInstructions bool) error {
	return ensureFileMatchesTemplate(
		filepath.Join(".github", "agents"),
		agentFileName,
		templateContent,
		"agent",
		verbose,
		skipInstructions,
	)
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

// ensureAgenticWorkflowAgent ensures that .github/agents/create-agentic-workflow.md contains the agentic workflow creation agent
func ensureAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("create-agentic-workflow.md", agenticWorkflowAgentTemplate, verbose, skipInstructions)
}

// ensureSharedAgenticWorkflowAgent ensures that .github/agents/create-shared-agentic-workflow.md contains the shared workflow creation agent
func ensureSharedAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("create-shared-agentic-workflow.md", sharedAgenticWorkflowAgentTemplate, verbose, skipInstructions)
}

// ensureSetupAgenticWorkflowsAgent ensures that .github/agents/setup-agentic-workflows.md contains the setup guide agent
func ensureSetupAgenticWorkflowsAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("setup-agentic-workflows.md", setupAgenticWorkflowsAgentTemplate, verbose, skipInstructions)
}

// ensureDebugAgenticWorkflowAgent ensures that .github/agents/debug-agentic-workflow.md contains the debug workflow agent
func ensureDebugAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("debug-agentic-workflow.md", debugAgenticWorkflowAgentTemplate, verbose, skipInstructions)
}
