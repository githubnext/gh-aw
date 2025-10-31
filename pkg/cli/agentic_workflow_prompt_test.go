package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAgenticWorkflowAgent(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedContent string
	}{
		{
			name:            "creates new agentic workflow agent file",
			existingContent: "",
			expectedContent: strings.TrimSpace(agenticWorkflowAgentTemplate),
		},
		{
			name:            "does not modify existing correct file",
			existingContent: agenticWorkflowAgentTemplate,
			expectedContent: strings.TrimSpace(agenticWorkflowAgentTemplate),
		},
		{
			name:            "updates modified file",
			existingContent: "# Modified Agentic Workflow Agent\n\nThis is a modified version.",
			expectedContent: strings.TrimSpace(agenticWorkflowAgentTemplate),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()

			// Change to temp directory and initialize git repo for findGitRoot to work
			oldWd, _ := os.Getwd()
			defer func() {
				_ = os.Chdir(oldWd)
			}()
			err := os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Initialize git repo
			if err := exec.Command("git", "init").Run(); err != nil {
				t.Fatalf("Failed to init git repo: %v", err)
			}

			agentsDir := filepath.Join(tempDir, ".github", "agents")
			agenticWorkflowAgentPath := filepath.Join(agentsDir, "create-agentic-workflow.agent.md")

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatalf("Failed to create agents directory: %v", err)
				}
				if err := os.WriteFile(agenticWorkflowAgentPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial agentic workflow agent: %v", err)
				}
			}

			// Call the function with skipInstructions=false to test the functionality
			err = ensureAgenticWorkflowAgent(false, false)
			if err != nil {
				t.Fatalf("ensureAgenticWorkflowAgent() returned error: %v", err)
			}

			// Check that file exists
			if _, err := os.Stat(agenticWorkflowAgentPath); os.IsNotExist(err) {
				t.Fatalf("Expected agentic workflow agent file to exist")
			}

			// Check content
			content, err := os.ReadFile(agenticWorkflowAgentPath)
			if err != nil {
				t.Fatalf("Failed to read agentic workflow agent: %v", err)
			}

			contentStr := strings.TrimSpace(string(content))
			expectedStr := strings.TrimSpace(tt.expectedContent)

			if contentStr != expectedStr {
				t.Errorf("Expected content does not match.\nExpected first 100 chars: %q\nActual first 100 chars: %q",
					expectedStr[:min(100, len(expectedStr))],
					contentStr[:min(100, len(contentStr))])
			}
		})
	}
}

func TestEnsureAgenticWorkflowAgent_WithSkipInstructionsTrue(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory and initialize git repo for findGitRoot to work
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err := os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Call the function with skipInstructions=true
	err = ensureAgenticWorkflowAgent(false, true)
	if err != nil {
		t.Fatalf("ensureAgenticWorkflowAgent() returned error: %v", err)
	}

	// Check that file was NOT created
	agentsDir := filepath.Join(tempDir, ".github", "agents")
	agenticWorkflowAgentPath := filepath.Join(agentsDir, "create-agentic-workflow.agent.md")
	if _, err := os.Stat(agenticWorkflowAgentPath); !os.IsNotExist(err) {
		t.Fatalf("Expected agentic workflow agent file to NOT exist when skipInstructions=true")
	}
}

func TestEnsureAgenticWorkflowPrompt_RemovesOldFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory and initialize git repo for findGitRoot to work
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err := os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create the old prompt file
	promptsDir := filepath.Join(tempDir, ".github", "prompts")
	oldPromptPath := filepath.Join(promptsDir, "create-agentic-workflow.prompt.md")
	
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("Failed to create prompts directory: %v", err)
	}
	if err := os.WriteFile(oldPromptPath, []byte("old content"), 0644); err != nil {
		t.Fatalf("Failed to create old prompt file: %v", err)
	}

	// Verify old file exists
	if _, err := os.Stat(oldPromptPath); os.IsNotExist(err) {
		t.Fatalf("Old prompt file should exist before test")
	}

	// Call the function to remove old prompt
	err = ensureAgenticWorkflowPrompt(false, false)
	if err != nil {
		t.Fatalf("ensureAgenticWorkflowPrompt() returned error: %v", err)
	}

	// Check that old file was removed
	if _, err := os.Stat(oldPromptPath); !os.IsNotExist(err) {
		t.Fatalf("Expected old prompt file to be removed")
	}
}
