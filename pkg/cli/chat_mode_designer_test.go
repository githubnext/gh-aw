package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureChatModeDesigner(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedContent string
	}{
		{
			name:            "creates new chat mode designer template file",
			existingContent: "",
			expectedContent: strings.TrimSpace(chatModeDesignerTemplate),
		},
		{
			name:            "does not modify existing correct file",
			existingContent: chatModeDesignerTemplate,
			expectedContent: strings.TrimSpace(chatModeDesignerTemplate),
		},
		{
			name:            "updates modified file",
			existingContent: "# Modified GitHub Agentic Workflows Designer\n\nThis is a modified version.",
			expectedContent: strings.TrimSpace(chatModeDesignerTemplate),
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

			instructionsDir := filepath.Join(tempDir, ".github", "instructions")
			chatModeDesignerPath := filepath.Join(instructionsDir, "github-agentic-workflows-designer.chatmode.md")

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(instructionsDir, 0755); err != nil {
					t.Fatalf("Failed to create instructions directory: %v", err)
				}
				if err := os.WriteFile(chatModeDesignerPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial chat mode designer template: %v", err)
				}
			}

			// Call the function with writeInstructions=true to test the functionality
			err = ensureChatModeDesigner(false, true)
			if err != nil {
				t.Fatalf("ensureChatModeDesigner() returned error: %v", err)
			}

			// Check that file exists
			if _, err := os.Stat(chatModeDesignerPath); os.IsNotExist(err) {
				t.Fatalf("Expected chat mode designer template file to exist")
			}

			// Check content
			content, err := os.ReadFile(chatModeDesignerPath)
			if err != nil {
				t.Fatalf("Failed to read chat mode designer template: %v", err)
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

func TestEnsureChatModeDesigner_WithWriteInstructionsFalse(t *testing.T) {
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

	// Call the function with writeInstructions=false
	err = ensureChatModeDesigner(false, false)
	if err != nil {
		t.Fatalf("ensureChatModeDesigner() returned error: %v", err)
	}

	// Check that file was NOT created
	instructionsDir := filepath.Join(tempDir, ".github", "instructions")
	chatModeDesignerPath := filepath.Join(instructionsDir, "github-agentic-workflows-designer.chatmode.md")
	if _, err := os.Stat(chatModeDesignerPath); !os.IsNotExist(err) {
		t.Fatalf("Expected chat mode designer template file to NOT exist when writeInstructions=false")
	}
}
