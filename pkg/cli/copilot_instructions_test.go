package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCopilotInstructions(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedContent string
	}{
		{
			name:            "creates new copilot instructions file",
			existingContent: "",
			expectedContent: strings.TrimSpace(copilotInstructionsTemplate),
		},
		{
			name:            "does not modify existing correct file",
			existingContent: copilotInstructionsTemplate,
			expectedContent: strings.TrimSpace(copilotInstructionsTemplate),
		},
		{
			name:            "updates modified file",
			existingContent: "# Modified GitHub Agentic Workflows - Copilot Instructions\n\nThis is a modified version.",
			expectedContent: strings.TrimSpace(copilotInstructionsTemplate),
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

			copilotDir := filepath.Join(tempDir, ".github", "instructions")
			copilotInstructionsPath := filepath.Join(copilotDir, "github-agentic-workflows.instructions.md")

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(copilotDir, 0755); err != nil {
					t.Fatalf("Failed to create copilot directory: %v", err)
				}
				if err := os.WriteFile(copilotInstructionsPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial copilot instructions: %v", err)
				}
			}

			// Call the function with writeInstructions=true to test the functionality
			err = ensureCopilotInstructions(false, true)
			if err != nil {
				t.Fatalf("ensureCopilotInstructions() returned error: %v", err)
			}

			// Check that file exists
			if _, err := os.Stat(copilotInstructionsPath); os.IsNotExist(err) {
				t.Fatalf("Expected copilot instructions file to exist")
			}

			// Check content
			content, err := os.ReadFile(copilotInstructionsPath)
			if err != nil {
				t.Fatalf("Failed to read copilot instructions: %v", err)
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

func TestEnsureCopilotInstructions_WithWriteInstructionsFalse(t *testing.T) {
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

	copilotDir := filepath.Join(tempDir, ".github", "instructions")
	copilotInstructionsPath := filepath.Join(copilotDir, "github-agentic-workflows.instructions.md")

	// Call the function with writeInstructions=false
	err = ensureCopilotInstructions(false, false)
	if err != nil {
		t.Fatalf("ensureCopilotInstructions() returned error: %v", err)
	}

	// Check that file does not exist
	if _, err := os.Stat(copilotInstructionsPath); !os.IsNotExist(err) {
		t.Fatalf("Expected copilot instructions file to not exist when writeInstructions=false")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
