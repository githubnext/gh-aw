package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
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
			tempDir := testutil.TempDir(t, "test-*")

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

			// Call the function (no MCP or campaign)
			err = InitRepository(false, false, false, false, "", []string{}, false, false, nil)

			// Check error expectation
			if tt.wantError {
				if err == nil {
					t.Errorf("InitRepository(, false, nil) expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("InitRepository(, false, nil) returned unexpected error: %v", err)
			}

			// Verify .gitattributes was created
			gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
			if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
				t.Errorf("Expected .gitattributes file to exist")
			}

			// Verify copilot instructions were created
			copilotInstructionsPath := filepath.Join(tempDir, ".github", "aw", "github-agentic-workflows.md")
			if _, err := os.Stat(copilotInstructionsPath); os.IsNotExist(err) {
				t.Errorf("Expected copilot instructions file to exist")
			}

			// Verify logs .gitignore was created
			logsGitignorePath := filepath.Join(tempDir, ".github", "aw", "logs", ".gitignore")
			if _, err := os.Stat(logsGitignorePath); os.IsNotExist(err) {
				t.Errorf("Expected .github/aw/logs/.gitignore file to exist")
			}

			// Verify logs .gitignore content
			if content, err := os.ReadFile(logsGitignorePath); err == nil {
				contentStr := string(content)
				if !strings.Contains(contentStr, "# Ignore all downloaded workflow logs") {
					t.Errorf("Expected .gitignore to contain comment about ignoring logs")
				}
				if !strings.Contains(contentStr, "*") {
					t.Errorf("Expected .gitignore to contain wildcard pattern")
				}
				if !strings.Contains(contentStr, "!.gitignore") {
					t.Errorf("Expected .gitignore to keep itself")
				}
			} else {
				t.Errorf("Failed to read .github/aw/logs/.gitignore: %v", err)
			}

			// Verify agentic workflow agent was created
			agenticWorkflowAgentPath := filepath.Join(tempDir, ".github", "agents", "create-agentic-workflow.agent.md")
			if _, err := os.Stat(agenticWorkflowAgentPath); os.IsNotExist(err) {
				t.Errorf("Expected agentic workflow agent file to exist")
			}

			// Verify debug agentic workflow agent was created
			debugAgenticWorkflowAgentPath := filepath.Join(tempDir, ".github", "agents", "debug-agentic-workflow.agent.md")
			if _, err := os.Stat(debugAgenticWorkflowAgentPath); os.IsNotExist(err) {
				t.Errorf("Expected debug agentic workflow agent file to exist")
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
	tempDir := testutil.TempDir(t, "test-*")

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
	err = InitRepository(false, false, false, false, "", []string{}, false, false, nil)
	if err != nil {
		t.Fatalf("InitRepository(, false, nil) returned error on first call: %v", err)
	}

	// Call the function second time
	err = InitRepository(false, false, false, false, "", []string{}, false, false, nil)
	if err != nil {
		t.Fatalf("InitRepository(, false, nil) returned error on second call: %v", err)
	}

	// Verify files still exist and are correct
	gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Errorf("Expected .gitattributes file to exist after second call")
	}

	copilotInstructionsPath := filepath.Join(tempDir, ".github", "aw", "github-agentic-workflows.md")
	if _, err := os.Stat(copilotInstructionsPath); os.IsNotExist(err) {
		t.Errorf("Expected copilot instructions file to exist after second call")
	}

	agenticWorkflowAgentPath := filepath.Join(tempDir, ".github", "agents", "create-agentic-workflow.agent.md")
	if _, err := os.Stat(agenticWorkflowAgentPath); os.IsNotExist(err) {
		t.Errorf("Expected agentic workflow agent file to exist after second call")
	}

	debugAgenticWorkflowAgentPath := filepath.Join(tempDir, ".github", "agents", "debug-agentic-workflow.agent.md")
	if _, err := os.Stat(debugAgenticWorkflowAgentPath); os.IsNotExist(err) {
		t.Errorf("Expected debug agentic workflow agent file to exist after second call")
	}

	// Verify logs .gitignore still exists after second call
	logsGitignorePath := filepath.Join(tempDir, ".github", "aw", "logs", ".gitignore")
	if _, err := os.Stat(logsGitignorePath); os.IsNotExist(err) {
		t.Errorf("Expected .github/aw/logs/.gitignore file to exist after second call")
	}
}

func TestInitRepository_Verbose(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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
	err = InitRepository(true, false, false, false, "", []string{}, false, false, nil)
	if err != nil {
		t.Fatalf("InitRepository(, false, nil) returned error with verbose=true: %v", err)
	}

	// Verify files were created
	gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Errorf("Expected .gitattributes file to exist with verbose=true")
	}
}

func TestInitRepository_WithCampaignDesignerAgent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Call InitRepository with campaign flag enabled
	if err := InitRepository(false, false, true, false, "", []string{}, false, false, nil); err != nil {
		t.Fatalf("InitRepository(, false, nil) with campaign flag returned error: %v", err)
	}

	agentPath := filepath.Join(tempDir, ".github", "agents", "create-agentic-campaign.agent.md")
	content, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("Expected agentic campaign designer agent to be created at %s, got error: %v", agentPath, err)
	}

	// Ensure the written file matches the embedded template (ignoring leading/trailing whitespace)
	got := strings.TrimSpace(string(content))
	want := strings.TrimSpace(createAgenticCampaignAgentTemplate)
	if got != want {
		t.Errorf("create-agentic-campaign.agent.md content did not match embedded template")
	}
}
