package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitRepository(t *testing.T) {
	tests := []struct {
		name      string
		setupRepo bool
		wantError bool
	}{
		{
			name:      "successfully initializes repository",
			setupRepo: true,
			wantError: false,
		},
		{
			name:      "fails when not in git repository",
			setupRepo: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()

			// Change to temp directory
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				_ = os.Chdir(oldWd)
			}()
			err = os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Initialize git repo if needed
			if tt.setupRepo {
				if err := exec.Command("git", "init").Run(); err != nil {
					t.Fatalf("Failed to init git repo: %v", err)
				}
			}

			// Call the function
			err = InitRepository(false, false)

			// Check error expectation
			if tt.wantError {
				if err == nil {
					t.Errorf("InitRepository() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("InitRepository() returned unexpected error: %v", err)
			}

			// Verify .gitattributes was created
			gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
			if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
				t.Errorf("Expected .gitattributes file to exist")
			}

			// Verify copilot instructions were created
			copilotInstructionsPath := filepath.Join(tempDir, ".github", "instructions", "github-agentic-workflows.instructions.md")
			if _, err := os.Stat(copilotInstructionsPath); os.IsNotExist(err) {
				t.Errorf("Expected copilot instructions file to exist")
			}

			// Verify agentic workflow agent was created
			agenticWorkflowAgentPath := filepath.Join(tempDir, ".github", "agents", "create-agentic-workflow.md")
			if _, err := os.Stat(agenticWorkflowAgentPath); os.IsNotExist(err) {
				t.Errorf("Expected agentic workflow agent file to exist")
			}

			// Verify .gitattributes contains the correct entry
			content, err := os.ReadFile(gitAttributesPath)
			if err != nil {
				t.Fatalf("Failed to read .gitattributes: %v", err)
			}
			if !strings.Contains(string(content), ".github/workflows/*.lock.yml linguist-generated=true merge=ours") {
				t.Errorf("Expected .gitattributes to contain '.github/workflows/*.lock.yml linguist-generated=true merge=ours'")
			}
		})
	}
}

func TestInitRepository_Idempotent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Call the function first time
	err = InitRepository(false, false)
	if err != nil {
		t.Fatalf("InitRepository() returned error on first call: %v", err)
	}

	// Call the function second time
	err = InitRepository(false, false)
	if err != nil {
		t.Fatalf("InitRepository() returned error on second call: %v", err)
	}

	// Verify files still exist and are correct
	gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Errorf("Expected .gitattributes file to exist after second call")
	}

	copilotInstructionsPath := filepath.Join(tempDir, ".github", "instructions", "github-agentic-workflows.instructions.md")
	if _, err := os.Stat(copilotInstructionsPath); os.IsNotExist(err) {
		t.Errorf("Expected copilot instructions file to exist after second call")
	}

	agenticWorkflowAgentPath := filepath.Join(tempDir, ".github", "agents", "create-agentic-workflow.md")
	if _, err := os.Stat(agenticWorkflowAgentPath); os.IsNotExist(err) {
		t.Errorf("Expected agentic workflow agent file to exist after second call")
	}
}

func TestInitRepository_Verbose(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Call the function with verbose=true (should not error)
	err = InitRepository(true, false)
	if err != nil {
		t.Fatalf("InitRepository() returned error with verbose=true: %v", err)
	}

	// Verify files were created
	gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Errorf("Expected .gitattributes file to exist with verbose=true")
	}
}
