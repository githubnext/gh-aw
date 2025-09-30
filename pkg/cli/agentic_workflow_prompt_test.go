package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAgenticWorkflowPrompt(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedContent string
	}{
		{
			name:            "creates new agentic workflow prompt file",
			existingContent: "",
			expectedContent: strings.TrimSpace(agenticWorkflowPromptTemplate),
		},
		{
			name:            "does not modify existing correct file",
			existingContent: agenticWorkflowPromptTemplate,
			expectedContent: strings.TrimSpace(agenticWorkflowPromptTemplate),
		},
		{
			name:            "updates modified file",
			existingContent: "# Modified Agentic Workflow Prompt\n\nThis is a modified version.",
			expectedContent: strings.TrimSpace(agenticWorkflowPromptTemplate),
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

			promptsDir := filepath.Join(tempDir, ".github", "prompts")
			agenticWorkflowPromptPath := filepath.Join(promptsDir, "create-agentic-workflow.prompt.md")

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(promptsDir, 0755); err != nil {
					t.Fatalf("Failed to create prompts directory: %v", err)
				}
				if err := os.WriteFile(agenticWorkflowPromptPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial agentic workflow prompt: %v", err)
				}
			}

			// Call the function with skipInstructions=false to test the functionality
			err = ensureAgenticWorkflowPrompt(false, false)
			if err != nil {
				t.Fatalf("ensureAgenticWorkflowPrompt() returned error: %v", err)
			}

			// Check that file exists
			if _, err := os.Stat(agenticWorkflowPromptPath); os.IsNotExist(err) {
				t.Fatalf("Expected agentic workflow prompt file to exist")
			}

			// Check content
			content, err := os.ReadFile(agenticWorkflowPromptPath)
			if err != nil {
				t.Fatalf("Failed to read agentic workflow prompt: %v", err)
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

func TestEnsureAgenticWorkflowPrompt_WithSkipInstructionsTrue(t *testing.T) {
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
	err = ensureAgenticWorkflowPrompt(false, true)
	if err != nil {
		t.Fatalf("ensureAgenticWorkflowPrompt() returned error: %v", err)
	}

	// Check that file was NOT created
	promptsDir := filepath.Join(tempDir, ".github", "prompts")
	agenticWorkflowPromptPath := filepath.Join(promptsDir, "create-agentic-workflow.prompt.md")
	if _, err := os.Stat(agenticWorkflowPromptPath); !os.IsNotExist(err) {
		t.Fatalf("Expected agentic workflow prompt file to NOT exist when skipInstructions=true")
	}
}
