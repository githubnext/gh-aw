package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var copilotAgentsLog = logger.New("cli:copilot_agents")

// ensureFileMatchesTemplate ensures a file in a subdirectory matches the expected template content
func ensureFileMatchesTemplate(subdir, fileName, templateContent, fileType string, verbose bool, skipInstructions bool) error {
	copilotAgentsLog.Printf("Ensuring file matches template: subdir=%s, file=%s, type=%s", subdir, fileName, fileType)

	if skipInstructions {
		copilotAgentsLog.Print("Skipping template update: instructions disabled")
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
		copilotAgentsLog.Printf("File is up-to-date: %s", targetPath)
		if verbose {
			fmt.Printf("%s is up-to-date: %s\n", fileType, targetPath)
		}
		return nil
	}

	// Write the file with restrictive permissions (0600) to follow security best practices
	// Agent files and instructions may contain sensitive configuration
	if err := os.WriteFile(targetPath, []byte(templateContent), 0600); err != nil {
		copilotAgentsLog.Printf("Failed to write file: %s, error: %v", targetPath, err)
		return fmt.Errorf("failed to write %s: %w", fileType, err)
	}

	if existingContent == "" {
		copilotAgentsLog.Printf("Created %s: %s", fileType, targetPath)
	} else {
		copilotAgentsLog.Printf("Updated %s: %s", fileType, targetPath)
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

// cleanupOldPromptFile removes an old prompt file from .github/prompts/ if it exists
func cleanupOldPromptFile(promptFileName string, verbose bool) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	oldPath := filepath.Join(gitRoot, ".github", "prompts", promptFileName)

	// Check if the old file exists and remove it
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("failed to remove old prompt file: %w", err)
		}
		if verbose {
			fmt.Printf("Removed old prompt file: %s\n", oldPath)
		}
	}

	return nil
}

// cleanupOldAgentFile removes an old agent file from .github/agents/ if it exists
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

// ensureAgenticWorkflowAgent ensures that .github/agents/create-agentic-workflow.agent.md contains the workflow creation agent
func ensureAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	// First, clean up the old prompt file if it exists
	if err := cleanupOldPromptFile("create-agentic-workflow.prompt.md", verbose); err != nil {
		return err
	}

	return ensureAgentFromTemplate("create-agentic-workflow.agent.md", agenticWorkflowAgentTemplate, verbose, skipInstructions)
}

// ensureDebugAgenticWorkflowAgent ensures that .github/agents/debug-agentic-workflow.agent.md contains the debug workflow agent
func ensureDebugAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	// First, clean up the old prompt file if it exists
	if err := cleanupOldPromptFile("debug-agentic-workflow.prompt.md", verbose); err != nil {
		return err
	}

	return ensureAgentFromTemplate("debug-agentic-workflow.agent.md", debugAgenticWorkflowAgentTemplate, verbose, skipInstructions)
}

// ensureUpgradeAgenticWorkflowAgent ensures that .github/agents/upgrade-agentic-workflows.md contains the upgrade workflow agent
func ensureUpgradeAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("upgrade-agentic-workflows.md", upgradeAgenticWorkflowAgentTemplate, verbose, skipInstructions)
}

// deleteSetupAgenticWorkflowsAgent deletes the setup-agentic-workflows.agent.md file if it exists
func deleteSetupAgenticWorkflowsAgent(verbose bool) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	agentPath := filepath.Join(gitRoot, ".github", "agents", "setup-agentic-workflows.agent.md")

	// Check if the file exists and remove it
	if _, err := os.Stat(agentPath); err == nil {
		if err := os.Remove(agentPath); err != nil {
			return fmt.Errorf("failed to remove setup-agentic-workflows agent: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Removed setup-agentic-workflows agent: %s\n", agentPath)
		}
	}

	// Also clean up the old prompt file if it exists
	return cleanupOldPromptFile("setup-agentic-workflows.prompt.md", verbose)
}
