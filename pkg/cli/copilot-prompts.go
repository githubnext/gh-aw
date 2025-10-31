package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensurePromptFromTemplate ensures that a prompt file exists and matches the embedded template
func ensurePromptFromTemplate(promptFileName, templateContent string, verbose bool, skipInstructions bool) error {
	if skipInstructions {
		return nil // Skip writing prompt if flag is set
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	promptsDir := filepath.Join(gitRoot, ".github", "prompts")
	promptPath := filepath.Join(promptsDir, promptFileName)

	// Ensure the .github/prompts directory exists
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/prompts directory: %w", err)
	}

	// Check if the prompt file already exists and matches the template
	existingContent := ""
	if content, err := os.ReadFile(promptPath); err == nil {
		existingContent = string(content)
	}

	// Check if content matches our expected template
	expectedContent := strings.TrimSpace(templateContent)
	if strings.TrimSpace(existingContent) == expectedContent {
		if verbose {
			fmt.Printf("Prompt is up-to-date: %s\n", promptPath)
		}
		return nil
	}

	// Write the prompt file
	if err := os.WriteFile(promptPath, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	if verbose {
		if existingContent == "" {
			fmt.Printf("Created prompt: %s\n", promptPath)
		} else {
			fmt.Printf("Updated prompt: %s\n", promptPath)
		}
	}

	return nil
}

// ensureAgentFromTemplate ensures that an agent file exists and matches the embedded template
func ensureAgentFromTemplate(agentFileName, templateContent string, verbose bool, skipInstructions bool) error {
	if skipInstructions {
		return nil // Skip writing agent if flag is set
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	agentsDir := filepath.Join(gitRoot, ".github", "agents")
	agentPath := filepath.Join(agentsDir, agentFileName)

	// Ensure the .github/agents directory exists
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/agents directory: %w", err)
	}

	// Check if the agent file already exists and matches the template
	existingContent := ""
	if content, err := os.ReadFile(agentPath); err == nil {
		existingContent = string(content)
	}

	// Check if content matches our expected template
	expectedContent := strings.TrimSpace(templateContent)
	if strings.TrimSpace(existingContent) == expectedContent {
		if verbose {
			fmt.Printf("Agent is up-to-date: %s\n", agentPath)
		}
		return nil
	}

	// Write the agent file
	if err := os.WriteFile(agentPath, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("failed to write agent file: %w", err)
	}

	if verbose {
		if existingContent == "" {
			fmt.Printf("Created agent: %s\n", agentPath)
		} else {
			fmt.Printf("Updated agent: %s\n", agentPath)
		}
	}

	return nil
}
