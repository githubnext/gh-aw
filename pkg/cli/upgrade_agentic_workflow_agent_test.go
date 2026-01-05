package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestEnsureUpgradeAgenticWorkflowAgent(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedContent string
	}{
		{
			name:            "creates new upgrade agentic workflow agent file",
			existingContent: "",
			expectedContent: strings.TrimSpace(upgradeAgenticWorkflowAgentTemplate),
		},
		{
			name:            "does not modify existing correct file",
			existingContent: upgradeAgenticWorkflowAgentTemplate,
			expectedContent: strings.TrimSpace(upgradeAgenticWorkflowAgentTemplate),
		},
		{
			name:            "updates modified file",
			existingContent: "# Modified Upgrade Agentic Workflow Agent\n\nThis is a modified version.",
			expectedContent: strings.TrimSpace(upgradeAgenticWorkflowAgentTemplate),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := testutil.TempDir(t, "test-*")

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
			upgradeAgenticWorkflowAgentPath := filepath.Join(agentsDir, "upgrade-agentic-workflow.agent.md")

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatalf("Failed to create agents directory: %v", err)
				}
				if err := os.WriteFile(upgradeAgenticWorkflowAgentPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial upgrade agentic workflow agent: %v", err)
				}
			}

			// Call the function with skipInstructions=false to test the functionality
			err = ensureUpgradeAgenticWorkflowAgent(false, false)
			if err != nil {
				t.Fatalf("ensureUpgradeAgenticWorkflowAgent() returned error: %v", err)
			}

			// Check that file exists
			if _, err := os.Stat(upgradeAgenticWorkflowAgentPath); os.IsNotExist(err) {
				t.Fatalf("Expected upgrade agentic workflow agent file to exist")
			}

			// Check content
			content, err := os.ReadFile(upgradeAgenticWorkflowAgentPath)
			if err != nil {
				t.Fatalf("Failed to read upgrade agentic workflow agent: %v", err)
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

func TestEnsureUpgradeAgenticWorkflowAgent_WithSkipInstructionsTrue(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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
	err = ensureUpgradeAgenticWorkflowAgent(false, true)
	if err != nil {
		t.Fatalf("ensureUpgradeAgenticWorkflowAgent() returned error: %v", err)
	}

	// Check that file was NOT created
	agentsDir := filepath.Join(tempDir, ".github", "agents")
	upgradeAgenticWorkflowAgentPath := filepath.Join(agentsDir, "upgrade-agentic-workflow.agent.md")
	if _, err := os.Stat(upgradeAgenticWorkflowAgentPath); !os.IsNotExist(err) {
		t.Fatalf("Expected upgrade agentic workflow agent file to NOT exist when skipInstructions=true")
	}
}
