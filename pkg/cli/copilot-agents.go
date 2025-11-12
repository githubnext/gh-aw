package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// ensureCopilotInstructions ensures that .github/instructions/github-agentic-workflows.md contains the copilot instructions
func ensureCopilotInstructions(verbose bool, skipInstructions bool) error {
	if skipInstructions {
		return nil // Skip writing instructions if flag is set
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	copilotDir := filepath.Join(gitRoot, ".github", "instructions")
	copilotInstructionsPath := filepath.Join(copilotDir, "github-agentic-workflows.instructions.md")

	// Ensure the .github/instructions directory exists
	if err := os.MkdirAll(copilotDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/instructions directory: %w", err)
	}

	// Check if the instructions file already exists and matches the template
	existingContent := ""
	if content, err := os.ReadFile(copilotInstructionsPath); err == nil {
		existingContent = string(content)
	}

	// Check if content matches our expected template
	expectedContent := strings.TrimSpace(copilotInstructionsTemplate)
	if strings.TrimSpace(existingContent) == expectedContent {
		if verbose {
			fmt.Printf("Copilot instructions are up-to-date: %s\n", copilotInstructionsPath)
		}
		return nil
	}

	// Write the copilot instructions file
	if err := os.WriteFile(copilotInstructionsPath, []byte(copilotInstructionsTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write copilot instructions: %w", err)
	}

	if verbose {
		if existingContent == "" {
			fmt.Printf("Created copilot instructions: %s\n", copilotInstructionsPath)
		} else {
			fmt.Printf("Updated copilot instructions: %s\n", copilotInstructionsPath)
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
